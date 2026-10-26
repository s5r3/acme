// Package layout holds the base HTML layout and shared UI components used by
// every feature's views, plus the per-request Page data they read.
package layout

import (
	"context"

	"example.com/acme/internal/platform/auth"
)

// Flash is a one-time message shown after a redirect.
type Flash struct {
	Kind    string // "success" | "error" | "info"
	Message string
}

// Page is request data every page needs: who is logged in, the CSRF token
// for forms and any flash message. The web middleware fills it in.
type Page struct {
	User      *auth.Principal
	CSRFToken string
	Flash     *Flash
	Path      string
}

type pageKey struct{}

// WithPage stores p in ctx.
func WithPage(ctx context.Context, p Page) context.Context {
	return context.WithValue(ctx, pageKey{}, p)
}

// PageFrom returns the Page for this request (zero value if none).
func PageFrom(ctx context.Context) Page {
	p, _ := ctx.Value(pageKey{}).(Page)
	return p
}
