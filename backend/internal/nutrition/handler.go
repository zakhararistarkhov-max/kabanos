package nutrition

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/auth"
	"github.com/kabanos/backend/internal/httpx"
	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/validate"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/goal", h.getGoal)
	r.Put("/goal", h.setGoal)
	r.Get("/day", h.day)
	r.Get("/history", h.history)

	r.Get("/activity-types", h.activityTypes)
	r.Post("/activities", h.addActivity)
	r.Delete("/activities/{id}", h.deleteActivity)

	r.Post("/diet", h.addManual)
	r.Delete("/diet/{id}", h.deleteDiet)

	r.Route("/dishes", func(r chi.Router) {
		r.Get("/", h.listDishes)
		r.Post("/", h.createDish)
		r.Post("/image-upload-url", h.imageUploadURL)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.getDish)
			r.Put("/", h.updateDish)
			r.Delete("/", h.deleteDish)
			r.Put("/rating", h.rateDish)
			r.Delete("/rating", h.unrateDish)
			r.Put("/favorite", h.favorite)
			r.Delete("/favorite", h.unfavorite)
			r.Get("/comments", h.listComments)
			r.Post("/comments", h.addComment)
			r.Delete("/comments/{commentId}", h.deleteComment)
			r.Post("/diet", h.addDishToDiet)
		})
	})
	return r
}

// ---------- DTOs ----------

type macrosDTO struct {
	Kcal    float64 `json:"kcal"`
	Protein float64 `json:"protein"`
	Fat     float64 `json:"fat"`
	Carbs   float64 `json:"carbs"`
}

type ingredientDTO struct {
	DishID       string    `json:"dishId"`
	Name         string    `json:"name"`
	Grams        float64   `json:"grams"`
	Per100       macrosDTO `json:"per100g"`
	Contribution macrosDTO `json:"contribution"`
}

type dishDTO struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Recipe       string          `json:"recipe"`
	ImageURL     *string         `json:"imageUrl"`
	Per100       macrosDTO       `json:"per100g"`
	ServingGrams *float64        `json:"servingGrams"`
	RatingAvg    float64         `json:"ratingAvg"`
	RatingCount  int             `json:"ratingCount"`
	MyRating     *int            `json:"myRating"`
	IsFavorite   bool            `json:"isFavorite"`
	IsMine       bool            `json:"isMine"`
	AuthorName   string          `json:"authorName"`
	CreatedAt    time.Time       `json:"createdAt"`
	Ingredients  []ingredientDTO `json:"ingredients"`
}

func (h *Handler) toDishDTO(r *http.Request, d *Dish, viewer uuid.UUID) dishDTO {
	ings := make([]ingredientDTO, 0, len(d.Ingredients))
	for _, ing := range d.Ingredients {
		c := ing.Contribution()
		ings = append(ings, ingredientDTO{
			DishID: ing.DishID.String(), Name: ing.Name, Grams: ing.Grams,
			Per100:       macrosDTO{ing.Per100.Kcal, ing.Per100.Protein, ing.Per100.Fat, ing.Per100.Carbs},
			Contribution: macrosDTO{round1(c.Kcal), round1(c.Protein), round1(c.Fat), round1(c.Carbs)},
		})
	}
	return dishDTO{
		ID:           d.ID.String(),
		Name:         d.Name,
		Description:  d.Description,
		Recipe:       d.Recipe,
		ImageURL:     h.svc.ImageURL(r.Context(), d.ImageKey),
		Per100:       macrosDTO{d.KcalPer100, d.ProteinPer100, d.FatPer100, d.CarbsPer100},
		ServingGrams: d.ServingGrams,
		RatingAvg:    round1(d.AvgRating()),
		RatingCount:  d.RatingCount,
		MyRating:     d.MyRating,
		IsFavorite:   d.IsFavorite,
		IsMine:       d.CreatedBy == viewer,
		AuthorName:   d.AuthorName,
		CreatedAt:    d.CreatedAt,
		Ingredients:  ings,
	}
}

type commentDTO struct {
	ID         string    `json:"id"`
	AuthorName string    `json:"authorName"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
	IsMine     bool      `json:"isMine"`
}

// ---------- goals ----------

func (h *Handler) getGoal(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.GetGoalOrDefault(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, goalToMap(g))
}

type setGoalRequest struct {
	Kcal    int     `json:"kcal"`
	Protein float64 `json:"protein"`
	Fat     float64 `json:"fat"`
	Carbs   float64 `json:"carbs"`
}

func (h *Handler) setGoal(w http.ResponseWriter, r *http.Request) {
	var req setGoalRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.Kcal > 0 && req.Kcal <= 20000, "kcal", "must be between 1 and 20000")
	v.Check(req.Protein >= 0 && req.Fat >= 0 && req.Carbs >= 0, "macros", "must be non-negative")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	g, err := h.svc.SetGoal(r.Context(), auth.UserID(r.Context()),
		Goal{Kcal: req.Kcal, Protein: req.Protein, Fat: req.Fat, Carbs: req.Carbs})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, goalToMap(g))
}

func goalToMap(g Goal) map[string]any {
	return map[string]any{"kcal": g.Kcal, "protein": g.Protein, "fat": g.Fat, "carbs": g.Carbs}
}

// ---------- day & history ----------

func (h *Handler) day(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	summary, err := h.svc.DaySummary(r.Context(), auth.UserID(r.Context()), q.Get("date"), q.Get("tz"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid date or timezone"))
		return
	}
	httpx.JSON(w, http.StatusOK, summary)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if from == "" || to == "" {
		httpx.Error(w, r, httpx.ErrBadRequest("from and to (YYYY-MM-DD) are required"))
		return
	}
	series, goal, err := h.svc.History(r.Context(), auth.UserID(r.Context()), from, to, q.Get("tz"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid range or timezone"))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"goal": goalToMap(goal), "series": series})
}

// ---------- activities ----------

func (h *Handler) activityTypes(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"types": ActivityTypes})
}

type addActivityRequest struct {
	Type        string     `json:"type"`
	MET         *float64   `json:"met"`
	DurationMin *int       `json:"durationMin"`
	Kcal        *float64   `json:"kcal"`
	PerformedAt *time.Time `json:"performedAt"`
}

func (h *Handler) addActivity(w http.ResponseWriter, r *http.Request) {
	var req addActivityRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Type == "" {
		req.Type = "activity"
	}
	a, err := h.svc.AddActivity(r.Context(), auth.UserID(r.Context()), req.Type, req.MET, req.DurationMin, req.Kcal, req.PerformedAt)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoWeight):
			httpx.Error(w, r, httpx.ErrBadRequest("укажите вес в разделе «Вес» или введите ккал вручную"))
		case errors.Is(err, ErrInvalidActivity):
			httpx.Error(w, r, httpx.ErrBadRequest("укажите либо ккал, либо тип/MET и длительность"))
		default:
			httpx.Error(w, r, err)
		}
		return
	}
	httpx.JSON(w, http.StatusCreated, a)
}

func (h *Handler) deleteActivity(w http.ResponseWriter, r *http.Request) {
	h.deleteOwned(w, r, "id", h.svc.DeleteActivity)
}

// ---------- diet ----------

type addManualRequest struct {
	Name       string     `json:"name"`
	Kcal       float64    `json:"kcal"`
	Protein    float64    `json:"protein"`
	Fat        float64    `json:"fat"`
	Carbs      float64    `json:"carbs"`
	Meal       *string    `json:"meal"`
	ConsumedAt *time.Time `json:"consumedAt"`
}

func (h *Handler) addManual(w http.ResponseWriter, r *http.Request) {
	var req addManualRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Required("name", req.Name)
	v.MaxLen("name", req.Name, 140)
	v.Check(req.Kcal >= 0 && req.Protein >= 0 && req.Fat >= 0 && req.Carbs >= 0, "macros", "must be non-negative")
	if req.Meal != nil {
		v.Check(validMeal(*req.Meal), "meal", "invalid meal")
	}
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	e, err := h.svc.AddManualEntry(r.Context(), auth.UserID(r.Context()), req.Name,
		Macros{Kcal: req.Kcal, Protein: req.Protein, Fat: req.Fat, Carbs: req.Carbs}, req.Meal, req.ConsumedAt)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, e)
}

func (h *Handler) deleteDiet(w http.ResponseWriter, r *http.Request) {
	h.deleteOwned(w, r, "id", h.svc.DeleteDietEntry)
}

// ---------- dishes ----------

func (h *Handler) listDishes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	viewer := auth.UserID(r.Context())
	limit, _ := strconv.Atoi(q.Get("limit"))
	page, _ := strconv.Atoi(q.Get("page"))
	if limit <= 0 {
		limit = 24
	}
	if page < 1 {
		page = 1
	}
	scope := q.Get("scope")
	if scope == "" {
		scope = "all"
	}
	dishes, total, err := h.svc.ListDishes(r.Context(), ListParams{
		ViewerID: viewer, Scope: scope, Query: q.Get("q"), Sort: q.Get("sort"),
		Limit: limit, Offset: (page - 1) * limit,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	items := make([]dishDTO, 0, len(dishes))
	for i := range dishes {
		items = append(items, h.toDishDTO(r, &dishes[i], viewer))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

type ingredientReq struct {
	DishID string  `json:"dishId"`
	Grams  float64 `json:"grams"`
}

type dishRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Recipe        string          `json:"recipe"`
	ImageKey      *string         `json:"imageKey"`
	KcalPer100    float64         `json:"kcalPer100"`
	ProteinPer100 float64         `json:"proteinPer100"`
	FatPer100     float64         `json:"fatPer100"`
	CarbsPer100   float64         `json:"carbsPer100"`
	ServingGrams  *float64        `json:"servingGrams"`
	Ingredients   []ingredientReq `json:"ingredients"`
}

// validated checks the request and parses ingredient references. For a composed
// dish (ingredients present) the macro fields are computed server-side, so their
// range checks are skipped.
func (req *dishRequest) validated() ([]IngredientInput, *httpx.APIError) {
	v := validate.New()
	v.Required("name", req.Name)
	v.MaxLen("name", req.Name, 140)
	v.MaxLen("description", req.Description, 2000)
	v.MaxLen("recipe", req.Recipe, 8000)

	ingredients := make([]IngredientInput, 0, len(req.Ingredients))
	for i, ing := range req.Ingredients {
		id, err := uuid.Parse(ing.DishID)
		if err != nil {
			v.Check(false, "ingredients", "invalid ingredient reference at position "+strconv.Itoa(i+1))
			continue
		}
		v.Check(ing.Grams > 0 && ing.Grams <= 100000, "ingredients", "grams must be between 1 and 100000")
		ingredients = append(ingredients, IngredientInput{DishID: id, Grams: ing.Grams})
	}
	v.Check(len(ingredients) <= 50, "ingredients", "too many ingredients")

	if len(ingredients) == 0 {
		v.Check(req.KcalPer100 >= 0 && req.KcalPer100 <= 1000, "kcalPer100", "must be between 0 and 1000")
		v.Check(req.ProteinPer100 >= 0 && req.FatPer100 >= 0 && req.CarbsPer100 >= 0, "macros", "must be non-negative")
		if req.ServingGrams != nil {
			v.Check(*req.ServingGrams > 0 && *req.ServingGrams <= 5000, "servingGrams", "must be between 1 and 5000")
		}
	}
	if !v.Valid() {
		return nil, httpx.ValidationError(v.Errors)
	}
	return ingredients, nil
}

func (h *Handler) createDish(w http.ResponseWriter, r *http.Request) {
	var req dishRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	ingredients, apiErr := req.validated()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	viewer := auth.UserID(r.Context())
	d, err := h.svc.CreateDish(r.Context(), &Dish{
		CreatedBy: viewer, Name: req.Name, Description: req.Description, Recipe: req.Recipe,
		ImageKey: req.ImageKey, KcalPer100: req.KcalPer100, ProteinPer100: req.ProteinPer100,
		FatPer100: req.FatPer100, CarbsPer100: req.CarbsPer100, ServingGrams: req.ServingGrams,
	}, ingredients)
	if err != nil {
		h.renderDishSaveErr(w, r, err, "")
		return
	}
	httpx.JSON(w, http.StatusCreated, h.toDishDTO(r, d, viewer))
}

// renderDishSaveErr maps composed-dish errors to 400 and otherwise defers to the
// standard not-found/internal handling.
func (h *Handler) renderDishSaveErr(w http.ResponseWriter, r *http.Request, err error, notFoundMsg string) {
	if errors.Is(err, ErrIngredientNotFound) {
		httpx.Error(w, r, httpx.ErrBadRequest("одно из блюд-ингредиентов не найдено"))
		return
	}
	if notFoundMsg != "" {
		h.renderErr(w, r, err, notFoundMsg)
		return
	}
	httpx.Error(w, r, err)
}

func (h *Handler) getDish(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	viewer := auth.UserID(r.Context())
	d, err := h.svc.GetDish(r.Context(), id, viewer)
	if err != nil {
		h.renderErr(w, r, err, "dish not found")
		return
	}
	httpx.JSON(w, http.StatusOK, h.toDishDTO(r, d, viewer))
}

func (h *Handler) updateDish(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req dishRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	ingredients, apiErr := req.validated()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	viewer := auth.UserID(r.Context())
	d, err := h.svc.UpdateDish(r.Context(), &Dish{
		ID: id, Name: req.Name, Description: req.Description, Recipe: req.Recipe,
		ImageKey: req.ImageKey, KcalPer100: req.KcalPer100, ProteinPer100: req.ProteinPer100,
		FatPer100: req.FatPer100, CarbsPer100: req.CarbsPer100, ServingGrams: req.ServingGrams,
	}, ingredients, viewer)
	if err != nil {
		h.renderDishSaveErr(w, r, err, "dish not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusOK, h.toDishDTO(r, d, viewer))
}

func (h *Handler) deleteDish(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteDish(r.Context(), id, auth.UserID(r.Context())); err != nil {
		h.renderErr(w, r, err, "dish not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type ratingRequest struct {
	Rating int `json:"rating"`
}

func (h *Handler) rateDish(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req ratingRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		httpx.Error(w, r, httpx.ErrBadRequest("rating must be between 1 and 5"))
		return
	}
	if err := h.svc.RateDish(r.Context(), id, auth.UserID(r.Context()), req.Rating); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) unrateDish(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.UnrateDish(r.Context(), id, auth.UserID(r.Context())); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) favorite(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.Favorite(r.Context(), auth.UserID(r.Context()), id); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) unfavorite(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.Unfavorite(r.Context(), auth.UserID(r.Context()), id); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) listComments(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	viewer := auth.UserID(r.Context())
	comments, err := h.svc.ListComments(r.Context(), id, 0)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	items := make([]commentDTO, 0, len(comments))
	for _, c := range comments {
		items = append(items, commentDTO{
			ID: c.ID.String(), AuthorName: c.AuthorName, Body: c.Body,
			CreatedAt: c.CreatedAt, IsMine: c.UserID == viewer,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

type commentRequest struct {
	Body string `json:"body"`
}

func (h *Handler) addComment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req commentRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Required("body", req.Body)
	v.MaxLen("body", req.Body, 1000)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	viewer := auth.UserID(r.Context())
	c, err := h.svc.AddComment(r.Context(), id, viewer, req.Body)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, commentDTO{
		ID: c.ID.String(), AuthorName: c.AuthorName, Body: c.Body, CreatedAt: c.CreatedAt, IsMine: true,
	})
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	cid, err := uuid.Parse(chi.URLParam(r, "commentId"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return
	}
	if err := h.svc.DeleteComment(r.Context(), cid, auth.UserID(r.Context())); err != nil {
		h.renderErr(w, r, err, "comment not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type addToDietRequest struct {
	Grams      *float64   `json:"grams"`
	Servings   *float64   `json:"servings"`
	Meal       *string    `json:"meal"`
	ConsumedAt *time.Time `json:"consumedAt"`
}

func (h *Handler) addDishToDiet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req addToDietRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Meal != nil && !validMeal(*req.Meal) {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid meal"))
		return
	}
	e, err := h.svc.AddDishToDiet(r.Context(), auth.UserID(r.Context()), id, req.Grams, req.Servings, req.Meal, req.ConsumedAt)
	if err != nil {
		switch {
		case errors.Is(err, ErrServingUnknown):
			httpx.Error(w, r, httpx.ErrBadRequest("у блюда не задан размер порции — укажите граммы"))
		case errors.Is(err, postgres.ErrNotFound):
			httpx.Error(w, r, httpx.ErrNotFound("dish not found"))
		default:
			httpx.Error(w, r, err)
		}
		return
	}
	httpx.JSON(w, http.StatusCreated, e)
}

// ---------- images ----------

type imageUploadRequest struct {
	ContentType string `json:"contentType"`
}

func (h *Handler) imageUploadURL(w http.ResponseWriter, r *http.Request) {
	var req imageUploadRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	url, key, err := h.svc.PrepareImageUpload(r.Context(), req.ContentType)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedMedia):
			httpx.Error(w, r, httpx.ErrBadRequest("поддерживаются JPEG, PNG, WEBP"))
		case errors.Is(err, ErrStorageDisabled):
			httpx.Error(w, r, httpx.ErrBadRequest("загрузка изображений отключена"))
		default:
			httpx.Error(w, r, err)
		}
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"uploadUrl": url, "key": key})
}

// ---------- helpers ----------

func parseID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) renderErr(w http.ResponseWriter, r *http.Request, err error, notFoundMsg string) {
	if errors.Is(err, postgres.ErrNotFound) {
		httpx.Error(w, r, httpx.ErrNotFound(notFoundMsg))
		return
	}
	httpx.Error(w, r, err)
}

// deleteOwned handles the common "delete my resource by id" pattern.
func (h *Handler) deleteOwned(w http.ResponseWriter, r *http.Request, param string, del func(ctx context.Context, userID, id uuid.UUID) error) {
	id, ok := parseID(w, r, param)
	if !ok {
		return
	}
	if err := del(r.Context(), auth.UserID(r.Context()), id); err != nil {
		h.renderErr(w, r, err, "not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
