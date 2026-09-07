package training

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
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
	r.Get("/meta", h.meta)

	r.Route("/exercises", func(r chi.Router) {
		r.Get("/", h.listExercises)
		r.Post("/", h.createExercise)
		r.Post("/image-upload-url", h.exerciseImageUploadURL)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.getExercise)
			r.Put("/", h.updateExercise)
			r.Delete("/", h.deleteExercise)
			r.Put("/rating", h.rateExercise)
			r.Delete("/rating", h.unrateExercise)
			r.Put("/publish", h.publishExercise)
			r.Delete("/publish", h.unpublishExercise)
			r.Put("/favorite", h.favoriteExercise)
			r.Delete("/favorite", h.unfavoriteExercise)
			r.Get("/comments", h.listExerciseComments)
			r.Post("/comments", h.addExerciseComment)
			r.Delete("/comments/{commentId}", h.deleteExerciseComment)
		})
	})

	r.Route("/workouts", func(r chi.Router) {
		r.Get("/", h.listWorkouts)
		r.Post("/", h.createWorkout)
		r.Post("/image-upload-url", h.workoutImageUploadURL)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.getWorkout)
			r.Put("/", h.updateWorkout)
			r.Delete("/", h.deleteWorkout)
			r.Put("/rating", h.rateWorkout)
			r.Delete("/rating", h.unrateWorkout)
			r.Put("/publish", h.publishWorkout)
			r.Delete("/publish", h.unpublishWorkout)
			r.Put("/favorite", h.favoriteWorkout)
			r.Delete("/favorite", h.unfavoriteWorkout)
			r.Get("/comments", h.listWorkoutComments)
			r.Post("/comments", h.addWorkoutComment)
			r.Delete("/comments/{commentId}", h.deleteWorkoutComment)
		})
	})
	return r
}

// ---------- meta ----------

func (h *Handler) meta(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"categories":   Categories,
		"difficulties": Difficulties,
		"jointImpacts": JointImpacts,
		"equipment":    Equipment,
		"muscles":      Muscles,
	})
}

// ---------- DTOs ----------

type exerciseDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Difficulty  string    `json:"difficulty"`
	JointImpact string    `json:"jointImpact"`
	Equipment   []string  `json:"equipment"`
	Muscles     []string  `json:"muscles"`
	ImageURL    *string   `json:"imageUrl"`
	VideoURL    string    `json:"videoUrl"`
	RatingAvg   float64   `json:"ratingAvg"`
	RatingCount int       `json:"ratingCount"`
	MyRating    *int      `json:"myRating"`
	IsFavorite  bool      `json:"isFavorite"`
	IsMine      bool      `json:"isMine"`
	IsPublic    bool      `json:"isPublic"`
	AuthorName  string    `json:"authorName"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (h *Handler) toExerciseDTO(r *http.Request, e *Exercise, viewer uuid.UUID) exerciseDTO {
	return exerciseDTO{
		ID: e.ID.String(), Name: e.Name, Description: e.Description,
		Category: e.Category, Difficulty: e.Difficulty, JointImpact: e.JointImpact,
		Equipment: nonNil(e.Equipment), Muscles: nonNil(e.Muscles),
		ImageURL: h.svc.ImageURL(r.Context(), e.ImageKey), VideoURL: e.VideoURL,
		RatingAvg: round1(e.AvgRating()), RatingCount: e.RatingCount, MyRating: e.MyRating,
		IsFavorite: e.IsFavorite, IsMine: e.CreatedBy == viewer, IsPublic: e.IsPublic,
		AuthorName: e.AuthorName, CreatedAt: e.CreatedAt,
	}
}

type exerciseRefDTO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Difficulty  string   `json:"difficulty"`
	JointImpact string   `json:"jointImpact"`
	Equipment   []string `json:"equipment"`
	ImageURL    *string  `json:"imageUrl"`
	RatingAvg   float64  `json:"ratingAvg"`
	RatingCount int      `json:"ratingCount"`
}

type workoutItemDTO struct {
	ID          string         `json:"id"`
	ExerciseID  string         `json:"exerciseId"`
	Position    int            `json:"position"`
	Sets        *int           `json:"sets"`
	Reps        *int           `json:"reps"`
	DurationSec *int           `json:"durationSec"`
	RestSec     *int           `json:"restSec"`
	WeightKg    *float64       `json:"weightKg"`
	Note        string         `json:"note"`
	Exercise    exerciseRefDTO `json:"exercise"`
}

type workoutDTO struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	Difficulty    string           `json:"difficulty"`
	ImageURL      *string          `json:"imageUrl"`
	ExerciseCount int              `json:"exerciseCount"`
	RatingAvg     float64          `json:"ratingAvg"`
	RatingCount   int              `json:"ratingCount"`
	MyRating      *int             `json:"myRating"`
	IsFavorite    bool             `json:"isFavorite"`
	IsMine        bool             `json:"isMine"`
	IsPublic      bool             `json:"isPublic"`
	AuthorName    string           `json:"authorName"`
	CreatedAt     time.Time        `json:"createdAt"`
	Items         []workoutItemDTO `json:"items,omitempty"`
}

func (h *Handler) toWorkoutCardDTO(r *http.Request, w *Workout, viewer uuid.UUID, count int) workoutDTO {
	return workoutDTO{
		ID: w.ID.String(), Name: w.Name, Description: w.Description, Difficulty: w.Difficulty,
		ImageURL: h.svc.ImageURL(r.Context(), w.ImageKey), ExerciseCount: count,
		RatingAvg: round1(w.AvgRating()), RatingCount: w.RatingCount, MyRating: w.MyRating,
		IsFavorite: w.IsFavorite, IsMine: w.CreatedBy == viewer, IsPublic: w.IsPublic,
		AuthorName: w.AuthorName, CreatedAt: w.CreatedAt,
	}
}

func (h *Handler) toWorkoutDetailDTO(r *http.Request, w *Workout, viewer uuid.UUID) workoutDTO {
	dto := h.toWorkoutCardDTO(r, w, viewer, len(w.Items))
	dto.Items = make([]workoutItemDTO, 0, len(w.Items))
	for _, it := range w.Items {
		dto.Items = append(dto.Items, workoutItemDTO{
			ID: it.ID.String(), ExerciseID: it.ExerciseID.String(), Position: it.Position,
			Sets: it.Sets, Reps: it.Reps, DurationSec: it.DurationSec, RestSec: it.RestSec,
			WeightKg: it.WeightKg, Note: it.Note,
			Exercise: exerciseRefDTO{
				ID: it.Exercise.ID.String(), Name: it.Exercise.Name, Category: it.Exercise.Category,
				Difficulty: it.Exercise.Difficulty, JointImpact: it.Exercise.JointImpact,
				Equipment: nonNil(it.Exercise.Equipment), ImageURL: h.svc.ImageURL(r.Context(), it.Exercise.ImageKey),
				RatingAvg:   round1(refAvg(it.Exercise)),
				RatingCount: it.Exercise.RatingCount,
			},
		})
	}
	return dto
}

type commentDTO struct {
	ID         string    `json:"id"`
	AuthorName string    `json:"authorName"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
	IsMine     bool      `json:"isMine"`
}

// ---------- exercises ----------

func (h *Handler) listExercises(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	viewer := auth.UserID(r.Context())
	items, total, err := h.svc.ListExercises(r.Context(), ListParams{
		ViewerID: viewer, Scope: scopeOf(q.Get("scope")), Query: q.Get("q"), Sort: q.Get("sort"),
		Category: q.Get("category"), Difficulty: q.Get("difficulty"), Equipment: q.Get("equipment"),
		Limit: limitOf(q.Get("limit")), Offset: offsetOf(q.Get("page"), q.Get("limit")),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]exerciseDTO, 0, len(items))
	for i := range items {
		out = append(out, h.toExerciseDTO(r, &items[i], viewer))
	}
	httpx.JSON(w, http.StatusOK, listEnvelope(out, total, q))
}

type exerciseRequest struct {
	Name        string   `json:"name"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Difficulty   string   `json:"difficulty"`
	JointImpact  string   `json:"jointImpact"`
	Equipment    []string `json:"equipment"`
	Muscles      []string `json:"muscles"`
	ImageKey     *string  `json:"imageKey"`
	VideoURL     string   `json:"videoUrl"`
}

func (req *exerciseRequest) normalizeDefaults() {
	if req.Category == "" {
		req.Category = "strength"
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}
	if req.JointImpact == "" {
		req.JointImpact = "medium"
	}
}

func (req *exerciseRequest) validated() *httpx.APIError {
	req.normalizeDefaults()
	v := validate.New()
	v.Required("name", req.Name)
	v.MaxLen("name", req.Name, 140)
	v.MaxLen("description", req.Description, 4000)
	v.Check(validCategory(req.Category), "category", "invalid category")
	v.Check(validDifficulty(req.Difficulty), "difficulty", "invalid difficulty")
	v.Check(validJoint(req.JointImpact), "jointImpact", "invalid joint impact")
	v.Check(validVideoURL(req.VideoURL), "videoUrl", "must be a valid http(s) link")
	if !v.Valid() {
		return httpx.ValidationError(v.Errors)
	}
	return nil
}

func (h *Handler) createExercise(w http.ResponseWriter, r *http.Request) {
	var req exerciseRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if apiErr := req.validated(); apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	viewer := auth.UserID(r.Context())
	e, err := h.svc.CreateExercise(r.Context(), &Exercise{
		CreatedBy: viewer, Name: req.Name, Description: req.Description, Category: req.Category,
		Difficulty: req.Difficulty, JointImpact: req.JointImpact, Equipment: req.Equipment,
		Muscles: req.Muscles, ImageKey: req.ImageKey, VideoURL: strings.TrimSpace(req.VideoURL),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, h.toExerciseDTO(r, e, viewer))
}

func (h *Handler) getExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	viewer := auth.UserID(r.Context())
	e, err := h.svc.GetExercise(r.Context(), id, viewer)
	if err != nil {
		renderErr(w, r, err, "exercise not found")
		return
	}
	httpx.JSON(w, http.StatusOK, h.toExerciseDTO(r, e, viewer))
}

func (h *Handler) updateExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req exerciseRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if apiErr := req.validated(); apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	viewer := auth.UserID(r.Context())
	e, err := h.svc.UpdateExercise(r.Context(), &Exercise{
		ID: id, Name: req.Name, Description: req.Description, Category: req.Category,
		Difficulty: req.Difficulty, JointImpact: req.JointImpact, Equipment: req.Equipment,
		Muscles: req.Muscles, ImageKey: req.ImageKey, VideoURL: strings.TrimSpace(req.VideoURL),
	}, viewer)
	if err != nil {
		renderErr(w, r, err, "exercise not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusOK, h.toExerciseDTO(r, e, viewer))
}

func (h *Handler) deleteExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteExercise(r.Context(), id, auth.UserID(r.Context())); err != nil {
		renderErr(w, r, err, "exercise not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) publishExercise(w http.ResponseWriter, r *http.Request)   { h.setExercisePublic(w, r, true) }
func (h *Handler) unpublishExercise(w http.ResponseWriter, r *http.Request) { h.setExercisePublic(w, r, false) }

func (h *Handler) setExercisePublic(w http.ResponseWriter, r *http.Request, public bool) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.PublishExercise(r.Context(), id, auth.UserID(r.Context()), public); err != nil {
		renderErr(w, r, err, "exercise not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"isPublic": public})
}

func (h *Handler) rateExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	rating, ok := decodeRating(w, r)
	if !ok {
		return
	}
	if err := h.svc.RateExercise(r.Context(), id, auth.UserID(r.Context()), rating); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) unrateExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.UnrateExercise(r.Context(), id, auth.UserID(r.Context())); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) favoriteExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.FavoriteExercise(r.Context(), auth.UserID(r.Context()), id); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) unfavoriteExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.UnfavoriteExercise(r.Context(), auth.UserID(r.Context()), id); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) listExerciseComments(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	comments, err := h.svc.ListExerciseComments(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": commentsToDTO(comments, auth.UserID(r.Context()))})
}

func (h *Handler) addExerciseComment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	body, ok := decodeComment(w, r)
	if !ok {
		return
	}
	c, err := h.svc.AddExerciseComment(r.Context(), id, auth.UserID(r.Context()), body)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, commentDTO{ID: c.ID.String(), AuthorName: c.AuthorName, Body: c.Body, CreatedAt: c.CreatedAt, IsMine: true})
}

func (h *Handler) deleteExerciseComment(w http.ResponseWriter, r *http.Request) {
	cid, err := uuid.Parse(chi.URLParam(r, "commentId"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return
	}
	if err := h.svc.DeleteExerciseComment(r.Context(), cid, auth.UserID(r.Context())); err != nil {
		renderErr(w, r, err, "comment not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) exerciseImageUploadURL(w http.ResponseWriter, r *http.Request) {
	h.imageUploadURL(w, r, "exercises")
}

// ---------- workouts ----------

func (h *Handler) listWorkouts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	viewer := auth.UserID(r.Context())
	items, total, err := h.svc.ListWorkouts(r.Context(), ListParams{
		ViewerID: viewer, Scope: scopeOf(q.Get("scope")), Query: q.Get("q"), Sort: q.Get("sort"),
		Difficulty: q.Get("difficulty"), Limit: limitOf(q.Get("limit")), Offset: offsetOf(q.Get("page"), q.Get("limit")),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	ids := make([]uuid.UUID, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	counts, err := h.svc.WorkoutItemCounts(r.Context(), ids)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]workoutDTO, 0, len(items))
	for i := range items {
		out = append(out, h.toWorkoutCardDTO(r, &items[i], viewer, counts[items[i].ID]))
	}
	httpx.JSON(w, http.StatusOK, listEnvelope(out, total, q))
}

type workoutItemRequest struct {
	ExerciseID  string   `json:"exerciseId"`
	Sets        *int     `json:"sets"`
	Reps        *int     `json:"reps"`
	DurationSec *int     `json:"durationSec"`
	RestSec     *int     `json:"restSec"`
	WeightKg    *float64 `json:"weightKg"`
	Note        string   `json:"note"`
}

type workoutRequest struct {
	Name        string               `json:"name"`
	Description  string               `json:"description"`
	Difficulty   string               `json:"difficulty"`
	ImageKey     *string              `json:"imageKey"`
	Items        []workoutItemRequest `json:"items"`
}

// parse validates the workout request and returns the domain item list.
func (req *workoutRequest) parse() ([]ItemInput, *httpx.APIError) {
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}
	v := validate.New()
	v.Required("name", req.Name)
	v.MaxLen("name", req.Name, 140)
	v.MaxLen("description", req.Description, 4000)
	v.Check(validDifficulty(req.Difficulty), "difficulty", "invalid difficulty")
	v.Check(len(req.Items) > 0, "items", "add at least one exercise")
	v.Check(len(req.Items) <= 100, "items", "too many exercises")

	items := make([]ItemInput, 0, len(req.Items))
	for i, it := range req.Items {
		exID, err := uuid.Parse(it.ExerciseID)
		if err != nil {
			v.Check(false, "items", "invalid exercise reference at position "+strconv.Itoa(i+1))
			continue
		}
		v.Check(len(it.Note) <= 300, "items", "note too long")
		items = append(items, ItemInput{
			ExerciseID: exID, Sets: it.Sets, Reps: it.Reps, DurationSec: it.DurationSec,
			RestSec: it.RestSec, WeightKg: it.WeightKg, Note: strings.TrimSpace(it.Note),
		})
	}
	if !v.Valid() {
		return nil, httpx.ValidationError(v.Errors)
	}
	return items, nil
}

func (h *Handler) createWorkout(w http.ResponseWriter, r *http.Request) {
	var req workoutRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	items, apiErr := req.parse()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	viewer := auth.UserID(r.Context())
	wk, err := h.svc.CreateWorkout(r.Context(), &Workout{
		CreatedBy: viewer, Name: req.Name, Description: req.Description, Difficulty: req.Difficulty, ImageKey: req.ImageKey,
	}, items)
	if err != nil {
		h.renderWorkoutSaveErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, h.toWorkoutDetailDTO(r, wk, viewer))
}

func (h *Handler) getWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	viewer := auth.UserID(r.Context())
	wk, err := h.svc.GetWorkout(r.Context(), id, viewer)
	if err != nil {
		renderErr(w, r, err, "workout not found")
		return
	}
	httpx.JSON(w, http.StatusOK, h.toWorkoutDetailDTO(r, wk, viewer))
}

func (h *Handler) updateWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req workoutRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	items, apiErr := req.parse()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	viewer := auth.UserID(r.Context())
	wk, err := h.svc.UpdateWorkout(r.Context(), &Workout{
		ID: id, Name: req.Name, Description: req.Description, Difficulty: req.Difficulty, ImageKey: req.ImageKey,
	}, items, viewer)
	if err != nil {
		h.renderWorkoutSaveErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, h.toWorkoutDetailDTO(r, wk, viewer))
}

func (h *Handler) deleteWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteWorkout(r.Context(), id, auth.UserID(r.Context())); err != nil {
		renderErr(w, r, err, "workout not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) publishWorkout(w http.ResponseWriter, r *http.Request)   { h.setWorkoutPublic(w, r, true) }
func (h *Handler) unpublishWorkout(w http.ResponseWriter, r *http.Request) { h.setWorkoutPublic(w, r, false) }

func (h *Handler) setWorkoutPublic(w http.ResponseWriter, r *http.Request, public bool) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.PublishWorkout(r.Context(), id, auth.UserID(r.Context()), public); err != nil {
		renderErr(w, r, err, "workout not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"isPublic": public})
}

func (h *Handler) rateWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	rating, ok := decodeRating(w, r)
	if !ok {
		return
	}
	if err := h.svc.RateWorkout(r.Context(), id, auth.UserID(r.Context()), rating); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) unrateWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.UnrateWorkout(r.Context(), id, auth.UserID(r.Context())); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) favoriteWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.FavoriteWorkout(r.Context(), auth.UserID(r.Context()), id); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) unfavoriteWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.UnfavoriteWorkout(r.Context(), auth.UserID(r.Context()), id); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) listWorkoutComments(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	comments, err := h.svc.ListWorkoutComments(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": commentsToDTO(comments, auth.UserID(r.Context()))})
}

func (h *Handler) addWorkoutComment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	body, ok := decodeComment(w, r)
	if !ok {
		return
	}
	c, err := h.svc.AddWorkoutComment(r.Context(), id, auth.UserID(r.Context()), body)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, commentDTO{ID: c.ID.String(), AuthorName: c.AuthorName, Body: c.Body, CreatedAt: c.CreatedAt, IsMine: true})
}

func (h *Handler) deleteWorkoutComment(w http.ResponseWriter, r *http.Request) {
	cid, err := uuid.Parse(chi.URLParam(r, "commentId"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return
	}
	if err := h.svc.DeleteWorkoutComment(r.Context(), cid, auth.UserID(r.Context())); err != nil {
		renderErr(w, r, err, "comment not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) workoutImageUploadURL(w http.ResponseWriter, r *http.Request) {
	h.imageUploadURL(w, r, "workouts")
}

func (h *Handler) renderWorkoutSaveErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrUnknownExercise):
		httpx.Error(w, r, httpx.ErrBadRequest("одно из упражнений не найдено"))
	case errors.Is(err, ErrEmptyWorkout):
		httpx.Error(w, r, httpx.ErrBadRequest("добавьте хотя бы одно упражнение"))
	case errors.Is(err, postgres.ErrNotFound):
		httpx.Error(w, r, httpx.ErrNotFound("workout not found or not yours"))
	default:
		httpx.Error(w, r, err)
	}
}

// ---------- shared helpers ----------

type imageUploadRequest struct {
	ContentType string `json:"contentType"`
}

func (h *Handler) imageUploadURL(w http.ResponseWriter, r *http.Request, prefix string) {
	var req imageUploadRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	url, key, err := h.svc.PrepareImageUpload(r.Context(), prefix, req.ContentType)
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

func decodeRating(w http.ResponseWriter, r *http.Request) (int, bool) {
	var req struct {
		Rating int `json:"rating"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return 0, false
	}
	if req.Rating < 1 || req.Rating > 5 {
		httpx.Error(w, r, httpx.ErrBadRequest("rating must be between 1 and 5"))
		return 0, false
	}
	return req.Rating, true
}

func decodeComment(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req struct {
		Body string `json:"body"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return "", false
	}
	v := validate.New()
	v.Required("body", req.Body)
	v.MaxLen("body", req.Body, 1000)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return "", false
	}
	return req.Body, true
}

func commentsToDTO(comments []Comment, viewer uuid.UUID) []commentDTO {
	out := make([]commentDTO, 0, len(comments))
	for _, c := range comments {
		out = append(out, commentDTO{ID: c.ID.String(), AuthorName: c.AuthorName, Body: c.Body, CreatedAt: c.CreatedAt, IsMine: c.UserID == viewer})
	}
	return out
}

func listEnvelope[T any](items []T, total int, q map[string][]string) map[string]any {
	page := offsetPage(get(q, "page"))
	limit := limitOf(get(q, "limit"))
	return map[string]any{"items": items, "total": total, "page": page, "limit": limit}
}

func get(q map[string][]string, k string) string {
	if v, ok := q[k]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

func scopeOf(s string) string {
	switch s {
	case "mine", "favorites":
		return s
	default:
		return "all"
	}
}

func limitOf(s string) int {
	n, _ := strconv.Atoi(s)
	if n <= 0 || n > 100 {
		return 24
	}
	return n
}

func offsetPage(s string) int {
	n, _ := strconv.Atoi(s)
	if n < 1 {
		return 1
	}
	return n
}

func offsetOf(page, limit string) int {
	return (offsetPage(page) - 1) * limitOf(limit)
}

func parseID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return uuid.Nil, false
	}
	return id, true
}

func renderErr(w http.ResponseWriter, r *http.Request, err error, notFoundMsg string) {
	if errors.Is(err, postgres.ErrNotFound) {
		httpx.Error(w, r, httpx.ErrNotFound(notFoundMsg))
		return
	}
	httpx.Error(w, r, err)
}

func validVideoURL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true // optional
	}
	if len(s) > 500 {
		return false
	}
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func refAvg(ref ExerciseRef) float64 {
	if ref.RatingCount == 0 {
		return 0
	}
	return float64(ref.RatingSum) / float64(ref.RatingCount)
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }
