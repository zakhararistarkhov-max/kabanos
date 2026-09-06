// Package httpx contains the transport-layer building blocks shared by every
// domain handler: a consistent JSON envelope, typed errors, middleware and a
// graceful HTTP server. Domain packages depend on this, never the reverse.
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/kabanos/backend/internal/observability"
)

// APIError is the single error type crossing the transport boundary. Every
// handler returns one (directly or via the sentinel constructors) and the
// router renders it into a stable JSON shape the frontend can rely on.
type APIError struct {
	Status  int            `json:"-"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"` // field-level validation errors
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

func newErr(status int, code, msg string) *APIError {
	return &APIError{Status: status, Code: code, Message: msg}
}

// Common constructors keep call sites terse and consistent.
func ErrBadRequest(msg string) *APIError   { return newErr(http.StatusBadRequest, "bad_request", msg) }
func ErrUnauthorized(msg string) *APIError { return newErr(http.StatusUnauthorized, "unauthorized", msg) }
func ErrForbidden(msg string) *APIError    { return newErr(http.StatusForbidden, "forbidden", msg) }
func ErrNotFound(msg string) *APIError     { return newErr(http.StatusNotFound, "not_found", msg) }
func ErrConflict(msg string) *APIError     { return newErr(http.StatusConflict, "conflict", msg) }
func ErrTooManyRequests(msg string) *APIError {
	return newErr(http.StatusTooManyRequests, "rate_limited", msg)
}
func ErrInternal() *APIError {
	return newErr(http.StatusInternalServerError, "internal_error", "something went wrong")
}

// ValidationError builds a 422 carrying per-field messages.
func ValidationError(fields map[string]string) *APIError {
	return &APIError{
		Status:  http.StatusUnprocessableEntity,
		Code:    "validation_failed",
		Message: "request validation failed",
		Fields:  fields,
	}
}

// JSON writes a successful JSON response.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode response", slog.String("error", err.Error()))
	}
}

// Error renders any error into the JSON envelope. Non-APIError values are
// treated as internal errors and their details are never leaked to the client.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		apiErr = ErrInternal()
		observability.LoggerFrom(r.Context()).Error("unhandled error", slog.String("error", err.Error()))
	} else if apiErr.Status >= 500 {
		observability.LoggerFrom(r.Context()).Error("server error", slog.String("error", apiErr.Error()))
	}
	JSON(w, apiErr.Status, map[string]any{"error": apiErr})
}

// Decode reads and strictly parses a JSON request body into dst, rejecting
// unknown fields and oversized payloads.
func Decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)) // 1 MiB cap
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return ErrBadRequest("invalid JSON body: " + err.Error())
	}
	return nil
}
