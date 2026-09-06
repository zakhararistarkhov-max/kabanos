package weight

import (
	"errors"
	"net/http"
	"strconv"

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
	r.Put("/goal", h.setGoal)
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

type setGoalRequest struct {
	TargetKg float64 `json:"targetKg"`
}

func (h *Handler) setGoal(w http.ResponseWriter, r *http.Request) {
	var req setGoalRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.TargetKg > 20 && req.TargetKg < 500, "targetKg", "must be between 20 and 500 kg")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	g, err := h.svc.SetGoal(r.Context(), auth.UserID(r.Context()), req.TargetKg)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"targetKg": g.TargetKg})
}

type addEntryRequest struct {
	WeightKg   float64 `json:"weightKg"`
	Note       string  `json:"note"`
	MeasuredOn string  `json:"measuredOn"` // optional YYYY-MM-DD
	TZ         string  `json:"tz"`
}

func (h *Handler) addEntry(w http.ResponseWriter, r *http.Request) {
	var req addEntryRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.WeightKg > 20 && req.WeightKg < 500, "weightKg", "must be between 20 and 500 kg")
	v.MaxLen("note", req.Note, 280)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	e, err := h.svc.AddEntry(r.Context(), auth.UserID(r.Context()), req.WeightKg, req.Note, req.MeasuredOn, req.TZ)
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid measuredOn date"))
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
