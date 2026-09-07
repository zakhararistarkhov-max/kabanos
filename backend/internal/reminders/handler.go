package reminders

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
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	return r
}

// ---------- DTO ----------

type reminderDTO struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Body            string     `json:"body"`
	URL             string     `json:"url"`
	Mode            string     `json:"mode"`
	IntervalMinutes *int       `json:"intervalMinutes"`
	WindowStart     string     `json:"windowStart"`
	WindowEnd       string     `json:"windowEnd"`
	Times           []string   `json:"times"`
	Days            []int      `json:"days"`
	Condition       string     `json:"condition"`
	Timezone        string     `json:"timezone"`
	Enabled         bool       `json:"enabled"`
	LastFiredAt     *time.Time `json:"lastFiredAt"`
}

func toDTO(r *Reminder) reminderDTO {
	return reminderDTO{
		ID: r.ID.String(), Title: r.Title, Body: r.Body, URL: r.URL, Mode: r.Mode,
		IntervalMinutes: r.IntervalMinutes, WindowStart: r.WindowStart, WindowEnd: r.WindowEnd,
		Times: nonNilStr(r.Times), Days: nonNilInt(r.Days), Condition: r.Condition,
		Timezone: r.Timezone, Enabled: r.Enabled, LastFiredAt: r.LastFiredAt,
	}
}

// ---------- handlers ----------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]reminderDTO, 0, len(items))
	for i := range items {
		out = append(out, toDTO(&items[i]))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

type reminderReq struct {
	Title           string   `json:"title"`
	Body            string   `json:"body"`
	URL             string   `json:"url"`
	Mode            string   `json:"mode"`
	IntervalMinutes *int     `json:"intervalMinutes"`
	WindowStart     string   `json:"windowStart"`
	WindowEnd       string   `json:"windowEnd"`
	Times           []string `json:"times"`
	Days            []int    `json:"days"`
	Condition       string   `json:"condition"`
	Timezone        string   `json:"timezone"`
	Enabled         *bool    `json:"enabled"`
}

func (req *reminderReq) toInput() (Input, *httpx.APIError) {
	v := validate.New()
	v.Required("title", req.Title)
	v.MaxLen("title", req.Title, 140)
	v.MaxLen("body", req.Body, 300)
	v.MaxLen("url", req.URL, 200)
	v.Check(req.Mode == "interval" || req.Mode == "times", "mode", "must be interval or times")

	if req.Mode == "interval" {
		v.Check(req.IntervalMinutes != nil && *req.IntervalMinutes >= 5 && *req.IntervalMinutes <= 1440,
			"intervalMinutes", "must be between 5 and 1440")
		v.Check(validHM(req.WindowStart) && validHM(req.WindowEnd), "window", "must be HH:MM")
	}
	if req.Mode == "times" {
		v.Check(len(req.Times) >= 1 && len(req.Times) <= 12, "times", "add 1..12 times")
		for _, t := range req.Times {
			v.Check(validHM(t), "times", "each time must be HH:MM")
		}
	}
	for _, d := range req.Days {
		v.Check(d >= 0 && d <= 6, "days", "weekday must be 0..6")
	}
	switch req.Condition {
	case "", "water_below_goal", "meds_due":
	default:
		v.Check(false, "condition", "unknown condition")
	}
	if !v.Valid() {
		return Input{}, httpx.ValidationError(v.Errors)
	}

	url := req.URL
	if url == "" {
		url = "/dashboard"
	}
	tz := req.Timezone
	if tz == "" {
		tz = "UTC"
	}
	ws, we := req.WindowStart, req.WindowEnd
	if ws == "" {
		ws = "08:00"
	}
	if we == "" {
		we = "22:00"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return Input{
		Title: req.Title, Body: req.Body, URL: url, Mode: req.Mode, IntervalMinutes: req.IntervalMinutes,
		WindowStart: ws, WindowEnd: we, Times: nonNilStr(req.Times), Days: nonNilInt(req.Days),
		Condition: req.Condition, Timezone: tz, Enabled: enabled,
	}, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req reminderReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	rem, err := h.svc.Create(r.Context(), auth.UserID(r.Context()), in)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toDTO(rem))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req reminderReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	rem, err := h.svc.Update(r.Context(), id, auth.UserID(r.Context()), in)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.Error(w, r, httpx.ErrNotFound("reminder not found"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toDTO(rem))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id, auth.UserID(r.Context())); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.Error(w, r, httpx.ErrNotFound("reminder not found"))
			return
		}
		httpx.Error(w, r, err)
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

func validHM(s string) bool {
	_, err := time.Parse("15:04", s)
	return err == nil
}

func nonNilStr(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
func nonNilInt(s []int) []int {
	if s == nil {
		return []int{}
	}
	return s
}
