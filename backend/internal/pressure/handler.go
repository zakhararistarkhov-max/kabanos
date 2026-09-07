package pressure

import (
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
	r.Get("/summary", h.summary)
	r.Post("/entries", h.addEntry)
	r.Delete("/entries/{id}", h.deleteEntry)
	return r
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	sum, err := h.svc.Summary(r.Context(), auth.UserID(r.Context()), limit)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}

type addEntryRequest struct {
	Systolic   int        `json:"systolic"`
	Diastolic  int        `json:"diastolic"`
	Pulse      *int       `json:"pulse"`
	Note       string     `json:"note"`
	MeasuredAt *time.Time `json:"measuredAt"` // optional RFC3339; defaults to now
}

func (h *Handler) addEntry(w http.ResponseWriter, r *http.Request) {
	var req addEntryRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.Systolic >= 50 && req.Systolic <= 300, "systolic", "must be between 50 and 300")
	v.Check(req.Diastolic >= 30 && req.Diastolic <= 200, "diastolic", "must be between 30 and 200")
	v.Check(req.Diastolic < req.Systolic, "diastolic", "must be lower than systolic")
	if req.Pulse != nil {
		v.Check(*req.Pulse >= 20 && *req.Pulse <= 300, "pulse", "must be between 20 and 300")
	}
	v.MaxLen("note", req.Note, 280)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	e, err := h.svc.AddEntry(r.Context(), auth.UserID(r.Context()), req.Systolic, req.Diastolic, req.Pulse, req.Note, req.MeasuredAt)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, e)
}

func (h *Handler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return
	}
	if err := h.svc.DeleteEntry(r.Context(), auth.UserID(r.Context()), id); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.Error(w, r, httpx.ErrNotFound("entry not found"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
