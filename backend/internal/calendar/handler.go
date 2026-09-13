package calendar

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kabanos/backend/internal/auth"
	"github.com/kabanos/backend/internal/httpx"
	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/validate"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/status", h.status)
	r.Post("/connect", h.connect)
	r.Post("/disconnect", h.disconnect)
	r.Get("/calendars", h.calendars)
	r.Put("/select", h.selectCalendar)
	r.Post("/sync", h.sync)
	return r
}

type statusDTO struct {
	Connected    bool       `json:"connected"`
	Provider     string     `json:"provider"`
	Login        string     `json:"login"`
	CalendarName string     `json:"calendarName"`
	CalendarURL  string     `json:"calendarUrl"`
	Enabled      bool       `json:"enabled"`
	LastSyncAt   *time.Time `json:"lastSyncAt"`
	LastError    string     `json:"lastError"`
}

func toStatus(c *Connection) statusDTO {
	return statusDTO{
		Connected: true, Provider: c.Provider, Login: c.Login, CalendarName: c.CalendarName,
		CalendarURL: c.CalendarURL, Enabled: c.Enabled, LastSyncAt: c.LastSyncAt, LastError: c.LastError,
	}
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	conn, err := h.svc.Status(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.JSON(w, http.StatusOK, statusDTO{Connected: false})
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toStatus(conn))
}

type connectRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) connect(w http.ResponseWriter, r *http.Request) {
	var req connectRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Required("login", req.Login)
	v.MaxLen("login", req.Login, 200)
	v.Required("password", req.Password)
	v.MaxLen("password", req.Password, 200)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	conn, err := h.svc.Connect(r.Context(), auth.UserID(r.Context()), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCreds) {
			httpx.Error(w, r, httpx.ErrBadRequest("Не удалось войти: проверьте логин и пароль приложения Яндекса"))
			return
		}
		httpx.Error(w, r, httpx.ErrBadRequest(err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, toStatus(conn))
}

func (h *Handler) disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Disconnect(r.Context(), auth.UserID(r.Context())); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.JSON(w, http.StatusNoContent, nil)
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) calendars(w http.ResponseWriter, r *http.Request) {
	cals, err := h.svc.ListCalendars(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.Error(w, r, httpx.ErrBadRequest("сначала подключите календарь"))
			return
		}
		if errors.Is(err, ErrInvalidCreds) {
			httpx.Error(w, r, httpx.ErrBadRequest("Неверный логин или пароль приложения"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": cals})
}

type selectRequest struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

func (h *Handler) selectCalendar(w http.ResponseWriter, r *http.Request) {
	var req selectRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Required("url", req.URL)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	if err := h.svc.SelectCalendar(r.Context(), auth.UserID(r.Context()), req.URL, req.Name); err != nil {
		httpx.Error(w, r, err)
		return
	}
	conn, err := h.svc.Status(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toStatus(conn))
}

func (h *Handler) sync(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Sync(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.Error(w, r, httpx.ErrBadRequest("сначала подключите календарь"))
			return
		}
		if errors.Is(err, ErrInvalidCreds) {
			httpx.Error(w, r, httpx.ErrBadRequest("Неверный логин или пароль приложения — переподключите календарь"))
			return
		}
		httpx.Error(w, r, httpx.ErrBadRequest(err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
