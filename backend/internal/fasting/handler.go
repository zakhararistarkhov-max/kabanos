package fasting

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
	r.Get("/", h.state)
	r.Put("/settings", h.setSettings)
	r.Post("/start", h.start)
	r.Post("/stop", h.stop)
	r.Put("/schedule", h.setSchedule)
	r.Put("/active", h.updateActive)
	r.Get("/history", h.history)
	r.Delete("/sessions/{id}", h.deleteSession)
	return r
}

// ---------- DTOs ----------

type statsDTO struct {
	TotalFasts   int     `json:"totalFasts"`
	LongestHours float64 `json:"longestHours"`
	AvgHours     float64 `json:"avgHours"`
}

type scheduleDTO struct {
	Enabled      bool   `json:"enabled"`
	EatStartHour int    `json:"eatStartHour"`
	EatStartMin  int    `json:"eatStartMinute"`
	Timezone     string `json:"timezone"`
	AutoStart    bool   `json:"autoStart"`
	NotifyStart  bool   `json:"notifyStart"`
	NotifyHourly bool   `json:"notifyHourly"`
}

type stateDTO struct {
	Phase        string      `json:"phase"`
	FastingHours float64     `json:"fastingHours"`
	EatingHours  float64     `json:"eatingHours"`
	ActiveID     *string     `json:"activeId"`
	PhaseStartAt *time.Time  `json:"phaseStartAt"`
	PhaseEndAt   *time.Time  `json:"phaseEndAt"`
	GoalHours    float64     `json:"goalHours"`
	Overrun      bool        `json:"overrun"`
	ServerNow    time.Time   `json:"serverNow"`
	Stats        statsDTO    `json:"stats"`
	Schedule     scheduleDTO `json:"schedule"`
}

func toStateDTO(s *State) stateDTO {
	d := stateDTO{
		Phase: s.Phase, FastingHours: s.FastingHours, EatingHours: s.EatingHours,
		PhaseStartAt: s.PhaseStartAt, PhaseEndAt: s.PhaseEndAt, GoalHours: round1(s.GoalHours),
		Overrun: s.Overrun, ServerNow: s.ServerNow,
		Stats:    statsDTO{TotalFasts: s.Stats.TotalFasts, LongestHours: round1(s.Stats.LongestHours), AvgHours: round1(s.Stats.AvgHours)},
		Schedule: toScheduleDTO(s.Schedule),
	}
	if s.ActiveID != nil {
		id := s.ActiveID.String()
		d.ActiveID = &id
	}
	return d
}

func toScheduleDTO(s Schedule) scheduleDTO {
	return scheduleDTO{
		Enabled: s.Enabled, EatStartHour: s.EatStartHour, EatStartMin: s.EatStartMin, Timezone: s.Timezone,
		AutoStart: s.AutoStart, NotifyStart: s.NotifyStart, NotifyHourly: s.NotifyHourly,
	}
}

type sessionDTO struct {
	ID        string     `json:"id"`
	StartedAt time.Time  `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt"`
	GoalHours float64    `json:"goalHours"`
}

func round1(f float64) float64 { return float64(int64(f*10+0.5)) / 10 }

// ---------- handlers ----------

func (h *Handler) writeState(w http.ResponseWriter, r *http.Request, s *State, err error) {
	if err != nil {
		if errors.Is(err, ErrNoActiveFast) {
			httpx.Error(w, r, httpx.ErrBadRequest("нет активного голодания"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toStateDTO(s))
}

func (h *Handler) state(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.State(r.Context(), auth.UserID(r.Context()), time.Now())
	h.writeState(w, r, s, err)
}

type settingsRequest struct {
	FastingHours float64 `json:"fastingHours"`
	EatingHours  float64 `json:"eatingHours"`
}

func (h *Handler) setSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.FastingHours > 0 && req.FastingHours <= 48, "fastingHours", "must be 0..48")
	v.Check(req.EatingHours > 0 && req.EatingHours <= 48, "eatingHours", "must be 0..48")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	s, err := h.svc.SetSettings(r.Context(), auth.UserID(r.Context()), Settings{FastingHours: req.FastingHours, EatingHours: req.EatingHours})
	h.writeState(w, r, s, err)
}

type scheduleRequest struct {
	Enabled      bool   `json:"enabled"`
	EatStartHour int    `json:"eatStartHour"`
	EatStartMin  int    `json:"eatStartMinute"`
	Timezone     string `json:"timezone"`
	AutoStart    bool   `json:"autoStart"`
	NotifyStart  bool   `json:"notifyStart"`
	NotifyHourly bool   `json:"notifyHourly"`
}

func (h *Handler) setSchedule(w http.ResponseWriter, r *http.Request) {
	var req scheduleRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.EatStartHour >= 0 && req.EatStartHour <= 23, "eatStartHour", "must be 0..23")
	v.Check(req.EatStartMin >= 0 && req.EatStartMin <= 59, "eatStartMinute", "must be 0..59")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	s, err := h.svc.SetSchedule(r.Context(), auth.UserID(r.Context()), Schedule{
		Enabled: req.Enabled, EatStartHour: req.EatStartHour, EatStartMin: req.EatStartMin, Timezone: req.Timezone,
		AutoStart: req.AutoStart, NotifyStart: req.NotifyStart, NotifyHourly: req.NotifyHourly,
	})
	h.writeState(w, r, s, err)
}

type timeRequest struct {
	StartedAt *string `json:"startedAt"`
	EndedAt   *string `json:"endedAt"`
}

func parseTimePtr(v *validate.Validator, field string, s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		v.Check(false, field, "must be an RFC3339 timestamp")
		return nil
	}
	return &t
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	var req timeRequest
	if r.ContentLength != 0 {
		if err := httpx.Decode(r, &req); err != nil {
			httpx.Error(w, r, err)
			return
		}
	}
	v := validate.New()
	started := parseTimePtr(v, "startedAt", req.StartedAt)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	s, err := h.svc.Start(r.Context(), auth.UserID(r.Context()), started)
	h.writeState(w, r, s, err)
}

func (h *Handler) stop(w http.ResponseWriter, r *http.Request) {
	var req timeRequest
	if r.ContentLength != 0 {
		if err := httpx.Decode(r, &req); err != nil {
			httpx.Error(w, r, err)
			return
		}
	}
	v := validate.New()
	ended := parseTimePtr(v, "endedAt", req.EndedAt)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	s, err := h.svc.Stop(r.Context(), auth.UserID(r.Context()), ended)
	h.writeState(w, r, s, err)
}

func (h *Handler) updateActive(w http.ResponseWriter, r *http.Request) {
	var req timeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	started := parseTimePtr(v, "startedAt", req.StartedAt)
	v.Check(started != nil, "startedAt", "required")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	s, err := h.svc.SetActiveStart(r.Context(), auth.UserID(r.Context()), *started)
	h.writeState(w, r, s, err)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	sessions, err := h.svc.History(r.Context(), auth.UserID(r.Context()), limit)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	items := make([]sessionDTO, 0, len(sessions))
	for _, s := range sessions {
		items = append(items, sessionDTO{ID: s.ID.String(), StartedAt: s.StartedAt, EndedAt: s.EndedAt, GoalHours: round1(s.GoalHours)})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return
	}
	if err := h.svc.Delete(r.Context(), id, auth.UserID(r.Context())); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.Error(w, r, httpx.ErrNotFound("session not found"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
