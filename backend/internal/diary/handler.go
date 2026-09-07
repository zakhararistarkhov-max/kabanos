package diary

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/auth"
	"github.com/kabanos/backend/internal/httpx"
	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/timex"
	"github.com/kabanos/backend/internal/validate"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/day", h.day)
	r.Get("/dates", h.dates)
	r.Post("/media-upload-url", h.mediaUploadURL)
	r.Post("/entries", h.create)
	r.Put("/entries/{id}", h.update)
	r.Delete("/entries/{id}", h.delete)
	return r
}

const dateLayout = "2006-01-02"

// ---------- DTOs ----------

type attachmentDTO struct {
	Key         string  `json:"key"`
	Kind        string  `json:"kind"`
	ContentType string  `json:"contentType"`
	URL         *string `json:"url"`
}

type entryDTO struct {
	ID          string          `json:"id"`
	Date        string          `json:"date"`
	Body        string          `json:"body"`
	Attachments []attachmentDTO `json:"attachments"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

func (h *Handler) toDTO(r *http.Request, e *Entry) entryDTO {
	atts := make([]attachmentDTO, 0, len(e.Attachments))
	for _, a := range e.Attachments {
		atts = append(atts, attachmentDTO{
			Key: a.Key, Kind: a.Kind, ContentType: a.ContentType,
			URL: h.svc.MediaURL(r.Context(), a.Key),
		})
	}
	return entryDTO{
		ID: e.ID.String(), Date: e.Date, Body: e.Body, Attachments: atts,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

// ---------- handlers ----------

func (h *Handler) day(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	date := q.Get("date")
	if date == "" {
		date = timex.Today(timex.Location(q.Get("tz")))
	}
	if _, err := time.Parse(dateLayout, date); err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid date"))
		return
	}
	entries, err := h.svc.Day(r.Context(), auth.UserID(r.Context()), date)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	items := make([]entryDTO, 0, len(entries))
	for i := range entries {
		items = append(items, h.toDTO(r, &entries[i]))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"date": date, "items": items})
}

func (h *Handler) dates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if _, err := time.Parse(dateLayout, from); err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("from (YYYY-MM-DD) required"))
		return
	}
	if _, err := time.Parse(dateLayout, to); err != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("to (YYYY-MM-DD) required"))
		return
	}
	out, err := h.svc.Dates(r.Context(), auth.UserID(r.Context()), from, to)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"dates": out})
}

type attachmentReq struct {
	Key         string `json:"key"`
	ContentType string `json:"contentType"`
}

type entryReq struct {
	Date        string          `json:"date"`
	Body        string          `json:"body"`
	Attachments []attachmentReq `json:"attachments"`
}

func (req *entryReq) attachments() []Attachment {
	out := make([]Attachment, 0, len(req.Attachments))
	for _, a := range req.Attachments {
		out = append(out, Attachment{Key: a.Key, ContentType: a.ContentType})
	}
	return out
}

func (req *entryReq) validate(needDate bool) *httpx.APIError {
	v := validate.New()
	if needDate {
		if _, err := time.Parse(dateLayout, req.Date); err != nil {
			v.Check(false, "date", "must be a date (YYYY-MM-DD)")
		}
	}
	v.MaxLen("body", req.Body, 20000)
	v.Check(len(req.Attachments) <= 30, "attachments", "too many attachments")
	v.Check(len(req.Attachments) > 0 || len(req.Body) > 0, "body", "empty entry")
	if !v.Valid() {
		return httpx.ValidationError(v.Errors)
	}
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req entryReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if apiErr := req.validate(true); apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	e, err := h.svc.Create(r.Context(), auth.UserID(r.Context()), req.Date, req.Body, req.attachments())
	if err != nil {
		h.renderSaveErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, h.toDTO(r, e))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req entryReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if apiErr := req.validate(false); apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	e, err := h.svc.Update(r.Context(), id, auth.UserID(r.Context()), req.Body, req.attachments())
	if err != nil {
		h.renderSaveErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, h.toDTO(r, e))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id, auth.UserID(r.Context())); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.Error(w, r, httpx.ErrNotFound("entry not found"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type mediaUploadReq struct {
	ContentType string `json:"contentType"`
}

func (h *Handler) mediaUploadURL(w http.ResponseWriter, r *http.Request) {
	var req mediaUploadReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	url, key, kind, err := h.svc.PrepareUpload(r.Context(), req.ContentType)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedMedia):
			httpx.Error(w, r, httpx.ErrBadRequest("поддерживаются фото (JPEG/PNG/WEBP/GIF) и видео (MP4/WEBM/MOV)"))
		case errors.Is(err, ErrStorageDisabled):
			httpx.Error(w, r, httpx.ErrBadRequest("загрузка медиа отключена"))
		default:
			httpx.Error(w, r, err)
		}
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"uploadUrl": url, "key": key, "kind": kind})
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

func (h *Handler) renderSaveErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedMedia):
		httpx.Error(w, r, httpx.ErrBadRequest("неподдерживаемый тип вложения"))
	case errors.Is(err, ErrInvalidAttachment):
		httpx.Error(w, r, httpx.ErrBadRequest("некорректное вложение"))
	case errors.Is(err, postgres.ErrNotFound):
		httpx.Error(w, r, httpx.ErrNotFound("entry not found"))
	default:
		httpx.Error(w, r, err)
	}
}
