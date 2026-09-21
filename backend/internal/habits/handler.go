package habits

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/auth"
	"github.com/kabanos/backend/internal/httpx"
	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/reminders"
	"github.com/kabanos/backend/internal/timex"
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
	r.Get("/{id}/checkins", h.listCheckins)
	r.Put("/{id}/checkins", h.setCheckin)
	r.Delete("/{id}/checkins", h.clearCheckin)
	r.Get("/{id}/logs", h.listLogs)
	r.Post("/{id}/logs", h.addLog)
	r.Delete("/{id}/logs/{logId}", h.deleteLog)
	r.Get("/{id}/reminders", h.listReminders)
	r.Post("/{id}/reminders", h.addReminder)
	r.Put("/{id}/reminders/{rid}", h.toggleReminder)
	r.Delete("/{id}/reminders/{rid}", h.deleteReminder)
	return r
}

// ---------- DTOs ----------

type habitDTO struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Kind          string     `json:"kind"`
	Description   string     `json:"description"`
	Archived      bool       `json:"archived"`
	ReminderCount int        `json:"reminderCount"`
	LogCount      int        `json:"logCount"`
	LastStatus    string     `json:"lastStatus"`
	LastLogAt     *time.Time `json:"lastLogAt"`
	Color         string     `json:"color"`
	Fails         int        `json:"fails"`
	Successes     int        `json:"successes"`
	Streak        int        `json:"streak"`
	Recent        []Checkin  `json:"recent"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func toHabitDTO(h Habit) habitDTO {
	recent := h.Recent
	if recent == nil {
		recent = []Checkin{}
	}
	color := h.Color
	if color == "" {
		color = "green"
	}
	return habitDTO{
		ID: h.ID.String(), Name: h.Name, Kind: h.Kind, Description: h.Description, Archived: h.Archived,
		ReminderCount: h.ReminderCount, LogCount: h.LogCount, LastStatus: h.LastStatus, LastLogAt: h.LastLogAt,
		Color: color, Fails: h.Fails, Successes: h.Successes, Streak: h.Streak, Recent: recent,
		CreatedAt: h.CreatedAt,
	}
}

type logDTO struct {
	ID        string    `json:"id"`
	Note      string    `json:"note"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type habitReminderDTO struct {
	ID       string   `json:"id"`
	Text     string   `json:"text"`
	Times    []string `json:"times"`
	Days     []int    `json:"days"`
	Timezone string   `json:"timezone"`
	Enabled  bool     `json:"enabled"`
}

func toReminderDTO(r reminders.Reminder) habitReminderDTO {
	times := r.Times
	if times == nil {
		times = []string{}
	}
	days := r.Days
	if days == nil {
		days = []int{}
	}
	return habitReminderDTO{ID: r.ID.String(), Text: r.Body, Times: times, Days: days, Timezone: r.Timezone, Enabled: r.Enabled}
}

// ---------- habits ----------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	today := timex.Today(timex.Location(r.URL.Query().Get("tz")))
	items, err := h.svc.List(r.Context(), auth.UserID(r.Context()), today)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]habitDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toHabitDTO(it))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

type habitReq struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Archived    bool   `json:"archived"`
}

func (req *habitReq) toInput(requireKind bool) (Input, *httpx.APIError) {
	v := validate.New()
	v.Required("name", req.Name)
	v.MaxLen("name", req.Name, 140)
	v.MaxLen("description", req.Description, 2000)
	if requireKind {
		v.Check(req.Kind == "good" || req.Kind == "bad", "kind", "must be good or bad")
	}
	if !v.Valid() {
		return Input{}, httpx.ValidationError(v.Errors)
	}
	return Input{Name: req.Name, Kind: req.Kind, Description: req.Description, Archived: req.Archived}, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req habitReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput(true)
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	habit, err := h.svc.Create(r.Context(), auth.UserID(r.Context()), in)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toHabitDTO(*habit))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req habitReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput(false)
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	habit, err := h.svc.Update(r.Context(), id, auth.UserID(r.Context()), in)
	if err != nil {
		renderErr(w, r, err, "habit not found")
		return
	}
	httpx.JSON(w, http.StatusOK, toHabitDTO(*habit))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id, auth.UserID(r.Context())); err != nil {
		renderErr(w, r, err, "habit not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// ---------- daily check-ins ----------

func (h *Handler) listCheckins(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if _, err := time.Parse(dayLayout, from); err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("from (YYYY-MM-DD) required"))
		return
	}
	if _, err := time.Parse(dayLayout, to); err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("to (YYYY-MM-DD) required"))
		return
	}
	marks, err := h.svc.Checkins(r.Context(), auth.UserID(r.Context()), id, from, to)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	// Return only the marked days as {day: success}; the client fills the grid.
	httpx.JSON(w, http.StatusOK, map[string]any{"days": marks})
}

type checkinReq struct {
	Day     string `json:"day"`
	Success bool   `json:"success"`
}

func (h *Handler) setCheckin(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req checkinReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if _, err := time.Parse(dayLayout, req.Day); err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("day must be YYYY-MM-DD"))
		return
	}
	if err := h.svc.SetCheckin(r.Context(), auth.UserID(r.Context()), id, req.Day, req.Success); err != nil {
		renderErr(w, r, err, "habit not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) clearCheckin(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	day := r.URL.Query().Get("day")
	if _, err := time.Parse(dayLayout, day); err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("day (YYYY-MM-DD) required"))
		return
	}
	if err := h.svc.ClearCheckin(r.Context(), auth.UserID(r.Context()), id, day); err != nil {
		renderErr(w, r, err, "habit not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// ---------- mini-diary ----------

func (h *Handler) listLogs(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	logs, err := h.svc.Logs(r.Context(), auth.UserID(r.Context()), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]logDTO, 0, len(logs))
	for _, l := range logs {
		out = append(out, logDTO{ID: l.ID.String(), Note: l.Note, Status: l.Status, CreatedAt: l.CreatedAt})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

type logReq struct {
	Note   string `json:"note"`
	Status string `json:"status"`
}

func (h *Handler) addLog(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req logReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.MaxLen("note", req.Note, 2000)
	v.Check(req.Status == "" || req.Status == "positive" || req.Status == "negative", "status", "invalid status")
	v.Check(req.Note != "" || req.Status != "", "note", "empty entry")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	l, err := h.svc.AddLog(r.Context(), auth.UserID(r.Context()), id, req.Note, req.Status)
	if err != nil {
		renderErr(w, r, err, "habit not found")
		return
	}
	httpx.JSON(w, http.StatusCreated, logDTO{ID: l.ID.String(), Note: l.Note, Status: l.Status, CreatedAt: l.CreatedAt})
}

func (h *Handler) deleteLog(w http.ResponseWriter, r *http.Request) {
	logID, ok := parseID(w, r, "logId")
	if !ok {
		return
	}
	if err := h.svc.DeleteLog(r.Context(), auth.UserID(r.Context()), logID); err != nil {
		renderErr(w, r, err, "entry not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// ---------- reminders ----------

func (h *Handler) listReminders(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	rems, err := h.svc.ListReminders(r.Context(), auth.UserID(r.Context()), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]habitReminderDTO, 0, len(rems))
	for _, rm := range rems {
		out = append(out, toReminderDTO(rm))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

type reminderReq struct {
	Text     string   `json:"text"`
	Times    []string `json:"times"`
	Days     []int    `json:"days"`
	Timezone string   `json:"timezone"`
}

func (h *Handler) addReminder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req reminderReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.MaxLen("text", req.Text, 300)
	v.Check(len(req.Times) >= 1 && len(req.Times) <= 12, "times", "add 1..12 times")
	for _, t := range req.Times {
		if _, err := time.Parse("15:04", t); err != nil {
			v.Check(false, "times", "each time must be HH:MM")
		}
	}
	for _, d := range req.Days {
		v.Check(d >= 0 && d <= 6, "days", "weekday must be 0..6")
	}
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	tz := req.Timezone
	if tz == "" {
		tz = "UTC"
	}
	rem, err := h.svc.AddReminder(r.Context(), auth.UserID(r.Context()), id, ReminderInput{
		Text: req.Text, Times: req.Times, Days: req.Days, Timezone: tz,
	})
	if err != nil {
		renderErr(w, r, err, "habit not found")
		return
	}
	httpx.JSON(w, http.StatusCreated, toReminderDTO(*rem))
}

type toggleReq struct {
	Enabled bool `json:"enabled"`
}

func (h *Handler) toggleReminder(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.guardHabit(w, r); !ok {
		return
	}
	rid, ok := parseID(w, r, "rid")
	if !ok {
		return
	}
	var req toggleReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.SetReminderEnabled(r.Context(), auth.UserID(r.Context()), rid, req.Enabled); err != nil {
		renderErr(w, r, err, "reminder not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) deleteReminder(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.guardHabit(w, r); !ok {
		return
	}
	rid, ok := parseID(w, r, "rid")
	if !ok {
		return
	}
	if err := h.svc.DeleteReminder(r.Context(), auth.UserID(r.Context()), rid); err != nil {
		renderErr(w, r, err, "reminder not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// ---------- helpers ----------

// guardHabit verifies the {id} habit belongs to the caller before touching its
// reminders by id.
func (h *Handler) guardHabit(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return uuid.Nil, false
	}
	if _, err := h.svc.Owns(r.Context(), auth.UserID(r.Context()), id); err != nil {
		renderErr(w, r, err, "habit not found")
		return uuid.Nil, false
	}
	return id, true
}

func parseID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid id"))
		return uuid.Nil, false
	}
	return id, true
}

func renderErr(w http.ResponseWriter, r *http.Request, err error, notFound string) {
	if errors.Is(err, postgres.ErrNotFound) {
		httpx.Error(w, r, httpx.ErrNotFound(notFound))
		return
	}
	httpx.Error(w, r, err)
}
