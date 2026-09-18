package gtd

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

const dateLayout = "2006-01-02"

var (
	validBuckets  = map[string]bool{"inbox": true, "next": true, "waiting": true, "calendar": true, "someday": true, "reference": true}
	validStatuses = map[string]bool{"active": true, "someday": true, "done": true, "dropped": true}
	validEnergy   = map[string]bool{"": true, "low": true, "medium": true, "high": true}
)

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/contexts", h.contexts)
	r.Get("/review", h.review)
	r.Route("/projects", func(r chi.Router) {
		r.Get("/", h.listProjects)
		r.Post("/", h.createProject)
		r.Put("/{id}", h.updateProject)
		r.Put("/{id}/priority", h.setProjectPriority)
		r.Delete("/{id}", h.deleteProject)
	})
	r.Route("/items", func(r chi.Router) {
		r.Get("/", h.listItems)
		r.Post("/", h.createItem)
		r.Put("/{id}", h.updateItem)
		r.Put("/{id}/priority", h.setItemPriority)
		r.Delete("/{id}", h.deleteItem)
		r.Post("/{id}/done", h.markDone)
		r.Delete("/{id}/done", h.markUndone)
	})
	r.Route("/graph", func(r chi.Router) {
		r.Get("/", h.getGraph)
		r.Get("/boards", h.getBoards)
		r.Get("/feed", h.getGraphFeed)
		r.Get("/settings", h.getGraphSettings)
		r.Put("/settings", h.putGraphSettings)
		r.Post("/image-upload-url", h.graphImageUploadURL)
		r.Post("/nodes", h.createGraphNode)
		r.Put("/nodes/{id}", h.updateGraphNode)
		r.Delete("/nodes/{id}", h.deleteGraphNode)
		r.Post("/nodes/{id}/images", h.addGraphNodeImage)
		r.Delete("/nodes/{id}/images", h.removeGraphNodeImage)
		r.Post("/edges", h.createGraphEdge)
		r.Delete("/edges/{id}", h.deleteGraphEdge)
	})
	return r
}

// ---------- DTOs ----------

type projectDTO struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Outcome     string     `json:"outcome"`
	Notes       string     `json:"notes"`
	Status      string     `json:"status"`
	Priority    int        `json:"priority"`
	OpenActions int        `json:"openActions"`
	NextActions int        `json:"nextActions"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt"`
}

func toProjectDTO(p Project) projectDTO {
	return projectDTO{
		ID: p.ID.String(), Title: p.Title, Outcome: p.Outcome, Notes: p.Notes, Status: p.Status, Priority: p.Priority,
		OpenActions: p.OpenActions, NextActions: p.NextActions, CreatedAt: p.CreatedAt, CompletedAt: p.CompletedAt,
	}
}

type itemDTO struct {
	ID           string     `json:"id"`
	ProjectID    *string    `json:"projectId"`
	ProjectTitle string     `json:"projectTitle"`
	Title        string     `json:"title"`
	Notes        string     `json:"notes"`
	Bucket       string     `json:"bucket"`
	Context      string     `json:"context"`
	WaitingFor   string     `json:"waitingFor"`
	ScheduledAt  *time.Time `json:"scheduledAt"`
	EndAt        *time.Time `json:"endAt"`
	AllDay       bool       `json:"allDay"`
	DueOn        *string    `json:"dueOn"`
	Energy       string     `json:"energy"`
	TimeMinutes  *int       `json:"timeMinutes"`
	Priority     int        `json:"priority"`
	Done         bool       `json:"done"`
	CompletedAt  *time.Time `json:"completedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}

func toItemDTO(it Item) itemDTO {
	var pid *string
	if it.ProjectID != nil {
		s := it.ProjectID.String()
		pid = &s
	}
	return itemDTO{
		ID: it.ID.String(), ProjectID: pid, ProjectTitle: it.ProjectTitle, Title: it.Title, Notes: it.Notes,
		Bucket: it.Bucket, Context: it.Context, WaitingFor: it.WaitingFor, ScheduledAt: it.ScheduledAt,
		EndAt: it.EndAt, AllDay: it.AllDay, DueOn: it.DueOn, Energy: it.Energy, TimeMinutes: it.TimeMinutes,
		Priority: it.Priority, Done: it.Done, CompletedAt: it.CompletedAt, CreatedAt: it.CreatedAt,
	}
}

func itemDTOs(items []Item) []itemDTO {
	out := make([]itemDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toItemDTO(it))
	}
	return out
}

// ---------- projects ----------

type projectRequest struct {
	Title    string `json:"title"`
	Outcome  string `json:"outcome"`
	Notes    string `json:"notes"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

func (req *projectRequest) toInput() (ProjectInput, *httpx.APIError) {
	v := validate.New()
	v.Required("title", req.Title)
	v.MaxLen("title", req.Title, 200)
	v.MaxLen("outcome", req.Outcome, 500)
	v.MaxLen("notes", req.Notes, 5000)
	status := req.Status
	if status == "" {
		status = "active"
	}
	v.Check(validStatuses[status], "status", "invalid status")
	priority := req.Priority
	if priority == 0 {
		priority = 3 // default: medium
	}
	v.Check(priority >= 1 && priority <= 5, "priority", "must be 1..5")
	if !v.Valid() {
		return ProjectInput{}, httpx.ValidationError(v.Errors)
	}
	return ProjectInput{Title: req.Title, Outcome: req.Outcome, Notes: req.Notes, Status: status, Priority: priority}, nil
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	ps, err := h.svc.ListProjects(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]projectDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, toProjectDTO(p))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var req projectRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	p, err := h.svc.CreateProject(r.Context(), auth.UserID(r.Context()), in)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toProjectDTO(*p))
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req projectRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	p, err := h.svc.UpdateProject(r.Context(), id, auth.UserID(r.Context()), in)
	if err != nil {
		renderErr(w, r, err, "project not found")
		return
	}
	httpx.JSON(w, http.StatusOK, toProjectDTO(*p))
}

type priorityRequest struct {
	Priority int `json:"priority"`
}

func (h *Handler) setProjectPriority(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req priorityRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.Priority >= 1 && req.Priority <= 5, "priority", "must be 1..5")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	if err := h.svc.SetProjectPriority(r.Context(), id, auth.UserID(r.Context()), req.Priority); err != nil {
		renderErr(w, r, err, "project not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) setItemPriority(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req priorityRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.Priority >= 0 && req.Priority <= 5, "priority", "must be 0..5")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	it, err := h.svc.SetItemPriority(r.Context(), id, auth.UserID(r.Context()), req.Priority)
	if err != nil {
		renderErr(w, r, err, "item not found")
		return
	}
	httpx.JSON(w, http.StatusOK, toItemDTO(*it))
}

func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteProject(r.Context(), id, auth.UserID(r.Context())); err != nil {
		renderErr(w, r, err, "project not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// ---------- items ----------

type itemRequest struct {
	ProjectID   *string `json:"projectId"`
	Title       string  `json:"title"`
	Notes       string  `json:"notes"`
	Bucket      string  `json:"bucket"`
	Context     string  `json:"context"`
	WaitingFor  string  `json:"waitingFor"`
	ScheduledAt *string `json:"scheduledAt"`
	EndAt       *string `json:"endAt"`
	AllDay      bool    `json:"allDay"`
	DueOn       *string `json:"dueOn"`
	Energy      string  `json:"energy"`
	TimeMinutes *int    `json:"timeMinutes"`
	Priority    int     `json:"priority"`
}

func (req *itemRequest) toInput() (ItemInput, *httpx.APIError) {
	v := validate.New()
	v.Required("title", req.Title)
	v.MaxLen("title", req.Title, 500)
	v.MaxLen("notes", req.Notes, 10000)
	v.MaxLen("context", req.Context, 60)
	v.MaxLen("waitingFor", req.WaitingFor, 200)

	bucket := req.Bucket
	if bucket == "" {
		bucket = "inbox"
	}
	v.Check(validBuckets[bucket], "bucket", "invalid bucket")
	v.Check(validEnergy[req.Energy], "energy", "invalid energy")
	v.Check(req.Priority >= 0 && req.Priority <= 5, "priority", "must be 0..5")
	if req.TimeMinutes != nil {
		v.Check(*req.TimeMinutes > 0 && *req.TimeMinutes <= 100000, "timeMinutes", "must be > 0")
	}

	var projectID *uuid.UUID
	if req.ProjectID != nil && *req.ProjectID != "" {
		pid, err := uuid.Parse(*req.ProjectID)
		if err != nil {
			v.Check(false, "projectId", "invalid id")
		} else {
			projectID = &pid
		}
	}

	scheduledAt := parseTimePtr(v, "scheduledAt", req.ScheduledAt)
	endAt := parseTimePtr(v, "endAt", req.EndAt)
	if bucket == "calendar" {
		v.Check(scheduledAt != nil, "scheduledAt", "calendar item needs a time")
	}

	var dueOn *string
	if req.DueOn != nil && *req.DueOn != "" {
		if _, err := time.Parse(dateLayout, *req.DueOn); err != nil {
			v.Check(false, "dueOn", "must be a date (YYYY-MM-DD)")
		} else {
			d := *req.DueOn
			dueOn = &d
		}
	}

	if !v.Valid() {
		return ItemInput{}, httpx.ValidationError(v.Errors)
	}
	return ItemInput{
		ProjectID: projectID, Title: req.Title, Notes: req.Notes, Bucket: bucket, Context: req.Context,
		WaitingFor: req.WaitingFor, ScheduledAt: scheduledAt, EndAt: endAt, AllDay: req.AllDay,
		DueOn: dueOn, Energy: req.Energy, TimeMinutes: req.TimeMinutes, Priority: req.Priority,
	}, nil
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

func (h *Handler) listItems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ItemFilter{Bucket: q.Get("bucket"), Context: q.Get("context"), Energy: q.Get("energy")}
	if pid := q.Get("projectId"); pid != "" {
		id, err := uuid.Parse(pid)
		if err != nil {
			httpx.Error(w, r, httpx.ErrBadRequest("invalid projectId"))
			return
		}
		f.ProjectID = &id
	}
	switch q.Get("done") {
	case "true":
		t := true
		f.Done = &t
	case "false":
		fl := false
		f.Done = &fl
	}
	if mt := q.Get("maxTime"); mt != "" {
		n, err := strconv.Atoi(mt)
		if err == nil && n > 0 {
			f.MaxTime = &n
		}
	}
	items, err := h.svc.ListItems(r.Context(), auth.UserID(r.Context()), f)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": itemDTOs(items)})
}

func (h *Handler) createItem(w http.ResponseWriter, r *http.Request) {
	var req itemRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	it, err := h.svc.CreateItem(r.Context(), auth.UserID(r.Context()), in)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toItemDTO(*it))
}

func (h *Handler) updateItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req itemRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	in, apiErr := req.toInput()
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	it, err := h.svc.UpdateItem(r.Context(), id, auth.UserID(r.Context()), in)
	if err != nil {
		renderErr(w, r, err, "item not found")
		return
	}
	httpx.JSON(w, http.StatusOK, toItemDTO(*it))
}

func (h *Handler) deleteItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteItem(r.Context(), id, auth.UserID(r.Context())); err != nil {
		renderErr(w, r, err, "item not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) markDone(w http.ResponseWriter, r *http.Request)   { h.setDone(w, r, true) }
func (h *Handler) markUndone(w http.ResponseWriter, r *http.Request) { h.setDone(w, r, false) }

func (h *Handler) setDone(w http.ResponseWriter, r *http.Request, done bool) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	it, err := h.svc.SetDone(r.Context(), id, auth.UserID(r.Context()), done)
	if err != nil {
		renderErr(w, r, err, "item not found")
		return
	}
	httpx.JSON(w, http.StatusOK, toItemDTO(*it))
}

// ---------- contexts & review ----------

func (h *Handler) contexts(w http.ResponseWriter, r *http.Request) {
	cs, err := h.svc.Contexts(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": cs})
}

type contextCountDTO struct {
	Context string `json:"context"`
	Count   int    `json:"count"`
}

func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	rev, err := h.svc.Review(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	byCtx := make([]contextCountDTO, 0, len(rev.ByContext))
	for _, c := range rev.ByContext {
		byCtx = append(byCtx, contextCountDTO{Context: c.Context, Count: c.Count})
	}
	stalled := make([]projectDTO, 0, len(rev.StalledProjects))
	for _, p := range rev.StalledProjects {
		stalled = append(stalled, toProjectDTO(p))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"inboxCount":        rev.InboxCount,
		"nextCount":         rev.NextCount,
		"waitingCount":      rev.WaitingCount,
		"somedayCount":      rev.SomedayCount,
		"calendarUpcoming":  rev.CalendarUpcoming,
		"byContext":         byCtx,
		"stalledProjects":   stalled,
		"overdueCalendar":   itemDTOs(rev.OverdueCalendar),
		"completedThisWeek": rev.CompletedThisWeek,
	})
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
