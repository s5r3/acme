package user

import (
	"time"

	"github.com/google/uuid"
)

// API request/response structs. Only these carry json and validate tags.

type TokenRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type TokenResponse struct {
	AccessToken string    `json:"accessToken"`
	TokenType   string    `json:"tokenType"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Name     string `json:"name" validate:"required,max=100"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type ListUsersQuery struct {
	Page     int `query:"page"`
	PageSize int `query:"pageSize" validate:"omitempty,max=100"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}
