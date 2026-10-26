package user

import "example.com/acme/internal/platform/httpx"

var (
	ErrUserNotFound       = httpx.New(httpx.KindNotFound, "user.not_found", "User not found")
	ErrEmailTaken         = httpx.New(httpx.KindConflict, "user.email_taken", "An account with this email already exists")
	ErrInvalidCredentials = httpx.New(httpx.KindUnauthenticated, "user.invalid_credentials", "Email or password is incorrect")
)
