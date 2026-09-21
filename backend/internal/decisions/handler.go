package decisions

import (
	"errors"
	"net/http"
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

const dateLayout = "2006-01-02"

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Post("/{id}/review", h.review)
	r.Delete("/{id}", h.delete)
	return r
}

// ---------- DTO ----------

type decisionDTO struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Context    string     `json:"context"`
	Decision   string     `json:"decision"`
	Expected   string     `json:"expected"`
	Confidence *int       `json:"confidence"`
	DecidedOn  string     `json:"decidedOn"`
	ReviewAt   *string    `json:"reviewAt"`
	Status     string     `json:"status"`
	Result     string     `json:"result"`
	Rating     *int       `json:"rating"`
	Lessons    string     `json:"lessons"`
	ReviewedAt *time.Time `json:"reviewedAt"`
	Due        bool       `json:"due"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func toDTO(d Decision) decisionDTO {
	return decisionDTO{
		ID: d.ID.String(), Title: d.Title, Context: d.Context, Decision: d.Decision, Expected: d.Expected,
		Confidence: d.Confidence, DecidedOn: d.DecidedOn, ReviewAt: d.ReviewAt, Status: d.Status,
		Result: d.Result, Rating: d.Rating, Lessons: d.Lessons, ReviewedAt: d.ReviewedAt, Due: d.Due,
		CreatedAt: d.CreatedAt,
	}
}

// ---------- handlers ----------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]decisionDTO, 0, len(items))
	for _, d := range items {
		out = append(out, toDTO(d))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

type decisionReq struct {
	Title      string  `json:"title"`
	Context    string  `json:"context"`
	Decision   string  `json:"decision"`
	Expected   string  `json:"expected"`
	Confidence *int    `json:"confidence"`
	DecidedOn  string  `json:"decidedOn"`
	ReviewAt   *string `json:"reviewAt"`
}

func (req *decisionReq) toInput() (Input, *httpx.APIError) {
	v := validate.New()
	v.Required("title", req.Title)
	v.MaxLen("title", req.Title, 200)
	v.MaxLen("context", req.Context, 5000)
	v.MaxLen("decision", req.Decision, 5000)
	v.MaxLen("expected", req.Expected, 5000)
	if req.Confidence != nil {
		v.Check(*req.Confidence >= 1 && *req.Confidence <= 5, "confidence", "must be 1..5")
	}
	if req.DecidedOn != "" {
		if _, err := time.Parse(dateLayout, req.DecidedOn); err != nil {
			v.Check(false, "decidedOn", "must be YYYY-MM-DD")
		}
	}
	reviewAt := cleanDate(req.ReviewAt)
	if reviewAt != nil {
		if _, err := time.Parse(dateLayout, *reviewAt); err != nil {
			v.Check(false, "reviewAt", "must be YYYY-MM-DD")
		}
	}
	if !v.Valid() {
		return Input{}, httpx.ValidationError(v.Errors)
	}
	return Input{
		Title: req.Title, Context: req.Context, Decision: req.Decision, Expected: req.Expected,
		Confidence: req.Confidence, DecidedOn: req.DecidedOn, ReviewAt: reviewAt,
	}, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req decisionReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	d, err := h.svc.Create(r.Context(), auth.UserID(r.Context()), in)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toDTO(*d))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req decisionReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	d, err := h.svc.Update(r.Context(), id, auth.UserID(r.Context()), in)
	if err != nil {
		renderErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toDTO(*d))
}

type reviewReq struct {
	Result  string `json:"result"`
	Rating  *int   `json:"rating"`
	Lessons string `json:"lessons"`
}

func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req reviewReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.MaxLen("result", req.Result, 5000)
	v.MaxLen("lessons", req.Lessons, 5000)
	if req.Rating != nil {
		v.Check(*req.Rating >= 1 && *req.Rating <= 5, "rating", "must be 1..5")
	}
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	d, err := h.svc.Review(r.Context(), id, auth.UserID(r.Context()), ReviewInput{Result: req.Result, Rating: req.Rating, Lessons: req.Lessons})
	if err != nil {
		renderErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toDTO(*d))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id, auth.UserID(r.Context())); err != nil {
		renderErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// ---------- helpers ----------

func cleanDate(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return uuid.Nil, false
	}
	return id, true
}

func renderErr(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, postgres.ErrNotFound) {
		httpx.Error(w, r, httpx.ErrNotFound("decision not found"))
		return
	}
	httpx.Error(w, r, err)
}
