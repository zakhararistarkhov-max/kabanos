package meds

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

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Route("/{id}", func(r chi.Router) {
		r.Put("/", h.update)
		r.Delete("/", h.delete)
		r.Post("/intake", h.take)
		r.Delete("/intake", h.undo)
	})
	return r
}

const dateLayout = "2006-01-02"

// ---------- DTO ----------

type medDTO struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Unit           string    `json:"unit"`
	Dose           float64   `json:"dose"`
	TimesPerDay    int       `json:"timesPerDay"`
	StartDate      string    `json:"startDate"`
	DurationDays   *int      `json:"durationDays"`
	CourseDay      int       `json:"courseDay"`
	CourseTotal    *int      `json:"courseTotal"`
	Status         string    `json:"status"`
	TakenToday     int       `json:"takenToday"`
	RemainingToday int       `json:"remainingToday"`
	WeekTaken      int       `json:"weekTaken"`
	Notes          string    `json:"notes"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"createdAt"`
}

func toDTO(v View) medDTO {
	remaining := v.TimesPerDay - v.TakenToday
	if remaining < 0 {
		remaining = 0
	}
	return medDTO{
		ID: v.ID.String(), Name: v.Name, Unit: v.Unit, Dose: v.Dose, TimesPerDay: v.TimesPerDay,
		StartDate: v.StartDate.Format(dateLayout), DurationDays: v.DurationDays,
		CourseDay: v.CourseDay, CourseTotal: v.CourseTotal, Status: v.Status,
		TakenToday: v.TakenToday, RemainingToday: remaining, WeekTaken: v.WeekTaken,
		Notes: v.Notes, Active: v.Active, CreatedAt: v.CreatedAt,
	}
}

// ---------- handlers ----------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	views, date, err := h.svc.List(r.Context(), auth.UserID(r.Context()), q.Get("date"), q.Get("tz"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid date or timezone"))
		return
	}
	items := make([]medDTO, 0, len(views))
	for _, v := range views {
		items = append(items, toDTO(v))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "date": date})
}

type medRequest struct {
	Name         string  `json:"name"`
	Unit         string  `json:"unit"`
	Dose         float64 `json:"dose"`
	TimesPerDay  int     `json:"timesPerDay"`
	StartDate    string  `json:"startDate"`
	DurationDays *int    `json:"durationDays"`
	Notes        string  `json:"notes"`
	Active       *bool   `json:"active"`
}

func (req *medRequest) toInput() (Input, *httpx.APIError) {
	v := validate.New()
	v.Required("name", req.Name)
	v.MaxLen("name", req.Name, 140)
	v.MaxLen("unit", req.Unit, 32)
	v.MaxLen("notes", req.Notes, 2000)
	v.Check(req.Dose > 0 && req.Dose <= 1000, "dose", "must be between 0 and 1000")
	v.Check(req.TimesPerDay >= 1 && req.TimesPerDay <= 24, "timesPerDay", "must be between 1 and 24")
	if req.DurationDays != nil {
		v.Check(*req.DurationDays > 0 && *req.DurationDays <= 3650, "durationDays", "must be between 1 and 3650")
	}
	start, err := time.Parse(dateLayout, req.StartDate)
	if err != nil {
		v.Check(false, "startDate", "must be a date (YYYY-MM-DD)")
	}
	if !v.Valid() {
		return Input{}, httpx.ValidationError(v.Errors)
	}

	unit := req.Unit
	if unit == "" {
		unit = "таблетка"
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	return Input{
		Name: req.Name, Unit: unit, Dose: req.Dose, TimesPerDay: req.TimesPerDay,
		StartDate: start, DurationDays: req.DurationDays, Notes: req.Notes, Active: active,
	}, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req medRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	m, err := h.svc.Create(r.Context(), auth.UserID(r.Context()), in)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	// A freshly created course has no intakes yet; present it via the view shape.
	httpx.JSON(w, http.StatusCreated, toDTO(buildView(Progress{Medication: *m}, m.StartDate)))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req medRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	m, err := h.svc.Update(r.Context(), id, auth.UserID(r.Context()), in)
	if err != nil {
		renderErr(w, r, err, "medication not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusOK, toDTO(buildView(Progress{Medication: *m}, m.StartDate)))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id, auth.UserID(r.Context())); err != nil {
		renderErr(w, r, err, "medication not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) take(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.TakeIntake(r.Context(), id, auth.UserID(r.Context()), r.URL.Query().Get("tz")); err != nil {
		renderErr(w, r, err, "medication not found or not yours")
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (h *Handler) undo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.UndoIntake(r.Context(), id, auth.UserID(r.Context()), r.URL.Query().Get("tz")); err != nil {
		renderErr(w, r, err, "nothing to undo")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// ---------- helpers ----------

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
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
