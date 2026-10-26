// Package httpx holds the API front-end's HTTP plumbing: the application error
// type, the JSON error handler, binding/validation and response envelopes.
// The Error type is also used by the web front-end, which maps the same kinds
// to HTML pages.
package httpx

import (
	"errors"
	"fmt"
	"net/http"
)

// Kind classifies an application error; each kind maps to one HTTP status.
type Kind int

const (
	KindInvalid         Kind = iota + 1 // 400: malformed or invalid input
	KindUnauthenticated                 // 401
	KindForbidden                       // 403
	KindNotFound                        // 404
	KindConflict                        // 409
	KindUnprocessable                   // 422: a business rule was violated
)

// Status returns the HTTP status code for k.
func (k Kind) Status() int {
	switch k {
	case KindInvalid:
		return http.StatusBadRequest
	case KindUnauthenticated:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindUnprocessable:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// Error is an error that is safe to show to clients.
// Features declare instances in their errors.go (ErrOrderNotFound, ...).
type Error struct {
	Kind    Kind
	Code    string            // stable, machine-readable: "order.closed"
	Message string            // human-readable, safe for clients
	Fields  map[string]string // per-field validation messages, optional
	Err     error             // wrapped cause; logged, never sent
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// Is makes errors.Is(err, ErrOrderNotFound) match on Code, so a sentinel
// still matches after being copied with WithFields or Wrap.
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return t.Code == e.Code
	}
	return false
}

// StatusCode lets echo.StatusCode(err) understand application errors too.
func (e *Error) StatusCode() int { return e.Kind.Status() }

// Wrap returns a copy of e carrying cause.
func (e *Error) Wrap(cause error) *Error {
	cp := *e
	cp.Err = cause
	return &cp
}

// New returns a new application error.
func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Validation returns a 400 error carrying per-field messages.
func Validation(fields map[string]string) *Error {
	return &Error{Kind: KindInvalid, Code: "validation.failed", Message: "Validation failed", Fields: fields}
}

// Common errors used by platform code.
var (
	ErrMalformedRequest = New(KindInvalid, "request.malformed", "Malformed request")
	ErrUnauthenticated  = New(KindUnauthenticated, "auth.required", "Authentication required")
	ErrForbidden        = New(KindForbidden, "auth.forbidden", "You do not have permission to do this")
	ErrNotFound         = New(KindNotFound, "resource.not_found", "Resource not found")
)

// AsError extracts an *Error from err, if it carries one.
func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
