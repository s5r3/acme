package web

import (
	"errors"

	"example.com/acme/internal/platform/httpx"
)

// HandleFormError decides how a failed form submission is answered.
// Errors the user can fix (validation and business rules) re-render the form
// via render, with per-field messages or one form-level message. Anything
// else is returned to the error handler.
func HandleFormError(err error, render func(fields map[string]string, formMsg string) error) error {
	appErr, ok := httpx.AsError(err)
	if !ok || errors.Is(err, httpx.ErrMalformedRequest) {
		return err
	}
	switch appErr.Kind {
	case httpx.KindInvalid, httpx.KindConflict, httpx.KindUnprocessable, httpx.KindUnauthenticated:
		if len(appErr.Fields) > 0 {
			return render(appErr.Fields, "")
		}
		return render(nil, appErr.Message)
	default:
		return err
	}
}
