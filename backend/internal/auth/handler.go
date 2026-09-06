package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kabanos/backend/internal/httpx"
	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/user"
	"github.com/kabanos/backend/internal/validate"
)

// Handler exposes the auth HTTP API. It is a thin adapter: parse → validate →
// call service → render. All business rules live in Service.
type Handler struct {
	svc   *Service
	users *user.Repo
}

func NewHandler(svc *Service, users *user.Repo) *Handler {
	return &Handler{svc: svc, users: users}
}

// PublicRoutes are reachable without authentication.
func (h *Handler) PublicRoutes() http.Handler {
	r := chi.NewRouter()
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/refresh", h.refresh)
	r.Post("/logout", h.logout)
	r.Post("/verify-email", h.verifyEmail)
	r.Post("/forgot-password", h.forgotPassword)
	r.Post("/reset-password", h.resetPassword)
	return r
}

// --- DTOs ---

type userDTO struct {
	ID               string   `json:"id"`
	Email            string   `json:"email"`
	DisplayName      string   `json:"displayName"`
	HeightCm         *float64 `json:"heightCm"`
	Sex              *string  `json:"sex"`
	TelegramUsername *string  `json:"telegramUsername"`
	EmailVerified    bool     `json:"emailVerified"`
	CreatedAt        time.Time `json:"createdAt"`
}

func toUserDTO(u *user.User) userDTO {
	return userDTO{
		ID:               u.ID.String(),
		Email:            u.Email,
		DisplayName:      u.DisplayName,
		HeightCm:         u.HeightCm,
		Sex:              u.Sex,
		TelegramUsername: u.TelegramUsername,
		EmailVerified:    u.IsEmailVerified(),
		CreatedAt:        u.CreatedAt,
	}
}

type sessionDTO struct {
	User             userDTO   `json:"user"`
	AccessToken      string    `json:"accessToken"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshToken     string    `json:"refreshToken"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
}

func toSessionDTO(s *Session) sessionDTO {
	return sessionDTO{
		User:             toUserDTO(s.User),
		AccessToken:      s.Tokens.AccessToken,
		AccessExpiresAt:  s.Tokens.AccessExpiresAt,
		RefreshToken:     s.Tokens.RefreshToken,
		RefreshExpiresAt: s.Tokens.RefreshExpiresAt,
	}
}

// --- handlers ---

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Email("email", req.Email)
	v.MinLen("password", req.Password, 8)
	v.MaxLen("password", req.Password, 128)
	v.MaxLen("displayName", req.DisplayName, 80)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}

	session, err := h.svc.Register(r.Context(), validate.NormalizeEmail(req.Email), req.Password, req.DisplayName, r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			httpx.Error(w, r, httpx.ErrConflict("email already registered"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toSessionDTO(session))
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Email == "" || req.Password == "" {
		httpx.Error(w, r, httpx.ErrBadRequest("email and password are required"))
		return
	}
	session, err := h.svc.Login(r.Context(), validate.NormalizeEmail(req.Email), req.Password, r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Error(w, r, httpx.ErrUnauthorized("invalid email or password"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toSessionDTO(session))
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.RefreshToken == "" {
		httpx.Error(w, r, httpx.ErrBadRequest("refreshToken is required"))
		return
	}
	toks, err := h.svc.Refresh(r.Context(), req.RefreshToken, r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrTokenReuse) {
			httpx.Error(w, r, httpx.ErrUnauthorized("invalid or expired refresh token"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"accessToken":      toks.AccessToken,
		"accessExpiresAt":  toks.AccessExpiresAt,
		"refreshToken":     toks.RefreshToken,
		"refreshExpiresAt": toks.RefreshExpiresAt,
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.RefreshToken != "" {
		if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
			httpx.Error(w, r, err)
			return
		}
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Token == "" {
		httpx.Error(w, r, httpx.ErrBadRequest("token is required"))
		return
	}
	if err := h.svc.VerifyEmail(r.Context(), req.Token); err != nil {
		if errors.Is(err, ErrInvalidToken) {
			httpx.Error(w, r, httpx.ErrBadRequest("verification link is invalid or expired"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	// Always return 202 regardless of whether the email exists.
	if req.Email != "" {
		if err := h.svc.RequestPasswordReset(r.Context(), validate.NormalizeEmail(req.Email)); err != nil {
			httpx.Error(w, r, err)
			return
		}
	}
	httpx.JSON(w, http.StatusAccepted, map[string]string{"status": "if the email exists, a reset link was sent"})
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Required("token", req.Token)
	v.MinLen("password", req.Password, 8)
	v.MaxLen("password", req.Password, 128)
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		if errors.Is(err, ErrInvalidToken) {
			httpx.Error(w, r, httpx.ErrBadRequest("reset link is invalid or expired"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "password updated"})
}

// --- authenticated endpoints (mounted behind Authenticator) ---

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, err := h.users.GetByID(r.Context(), UserID(r.Context()))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			httpx.Error(w, r, httpx.ErrNotFound("user not found"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toUserDTO(u))
}

type updateProfileRequest struct {
	DisplayName      *string  `json:"displayName"`
	HeightCm         *float64 `json:"heightCm"`
	Sex              *string  `json:"sex"`
	TelegramUsername *string  `json:"telegramUsername"`
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var req updateProfileRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	if req.HeightCm != nil {
		v.Check(*req.HeightCm > 50 && *req.HeightCm < 260, "heightCm", "must be between 50 and 260")
	}
	if req.Sex != nil {
		v.Check(*req.Sex == "male" || *req.Sex == "female" || *req.Sex == "other", "sex", "must be male, female or other")
	}
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	u, err := h.users.UpdateProfile(r.Context(), UserID(r.Context()), user.ProfileUpdate{
		DisplayName:      req.DisplayName,
		HeightCm:         req.HeightCm,
		Sex:              req.Sex,
		TelegramUsername: req.TelegramUsername,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toUserDTO(u))
}

func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ResendVerification(r.Context(), UserID(r.Context())); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, map[string]string{"status": "verification email sent"})
}
