package water

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

// Routes returns the water sub-router; it is mounted behind the auth middleware.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/goal", h.getGoal)
	r.Put("/goal", h.setGoal)
	r.Get("/day", h.day)
	r.Get("/history", h.history)
	r.Post("/intake", h.addIntake)
	r.Delete("/intake/{id}", h.deleteIntake)
	return r
}

func (h *Handler) getGoal(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.GetGoalOrDefault(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"dailyMl": g.DailyML})
}

type setGoalRequest struct {
	DailyML int `json:"dailyMl"`
}

func (h *Handler) setGoal(w http.ResponseWriter, r *http.Request) {
	var req setGoalRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.DailyML >= 250 && req.DailyML <= 20000, "dailyMl", "must be between 250 and 20000 ml")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	g, err := h.svc.SetGoal(r.Context(), auth.UserID(r.Context()), req.DailyML)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"dailyMl": g.DailyML})
}

func (h *Handler) day(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	tz := r.URL.Query().Get("tz")
	summary, err := h.svc.DaySummary(r.Context(), auth.UserID(r.Context()), date, tz)
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid date or timezone"))
		return
	}
	httpx.JSON(w, http.StatusOK, summary)
}

type addIntakeRequest struct {
	AmountML   int        `json:"amountMl"`
	Source     string     `json:"source"`
	ConsumedAt *time.Time `json:"consumedAt"`
}

func (h *Handler) addIntake(w http.ResponseWriter, r *http.Request) {
	var req addIntakeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Source == "" {
		req.Source = "custom"
	}
	v := validate.New()
	v.Check(req.AmountML > 0 && req.AmountML <= 5000, "amountMl", "must be between 1 and 5000 ml")
	v.Check(ValidSource(req.Source), "source", "unknown source")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	in, err := h.svc.AddIntake(r.Context(), auth.UserID(r.Context()), req.AmountML, req.Source, req.ConsumedAt)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, in)
}

func (h *Handler) deleteIntake(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return
	}
	if err := h.svc.DeleteIntake(r.Context(), auth.UserID(r.Context()), id); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.Error(w, r, httpx.ErrNotFound("intake not found"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	tz := q.Get("tz")
	if from == "" || to == "" {
		httpx.Error(w, r, httpx.ErrBadRequest("from and to (YYYY-MM-DD) are required"))
		return
	}
	series, goal, err := h.svc.History(r.Context(), auth.UserID(r.Context()), from, to, tz)
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid range or timezone"))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"goalMl": goal, "series": series})
}
