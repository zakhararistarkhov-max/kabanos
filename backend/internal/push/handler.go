package push

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kabanos/backend/internal/auth"
	"github.com/kabanos/backend/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/key", h.key)
	r.Post("/subscribe", h.subscribe)
	r.Post("/unsubscribe", h.unsubscribe)
	r.Post("/test", h.test)
	return r
}

func (h *Handler) key(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"enabled": h.svc.Enabled(), "key": h.svc.PublicKey()})
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *Handler) subscribe(w http.ResponseWriter, r *http.Request) {
	if !h.svc.Enabled() {
		httpx.Error(w, r, httpx.ErrBadRequest("push notifications are not configured"))
		return
	}
	var req subscribeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid subscription"))
		return
	}
	if err := h.svc.Subscribe(r.Context(), auth.UserID(r.Context()), req.Endpoint, req.Keys.P256dh, req.Keys.Auth); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"status": "subscribed"})
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func (h *Handler) unsubscribe(w http.ResponseWriter, r *http.Request) {
	var req unsubscribeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.Unsubscribe(r.Context(), req.Endpoint); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) test(w http.ResponseWriter, r *http.Request) {
	sent, err := h.svc.Send(r.Context(), auth.UserID(r.Context()), Notification{
		Title: "Kabanos",
		Body:  "Уведомления включены — так они будут выглядеть 🐗",
		URL:   "/dashboard",
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": sent})
}
