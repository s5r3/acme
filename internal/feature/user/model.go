package user

import (
	"github.com/google/uuid"

	"example.com/acme/internal/platform/database"
)

// User is the users table. It carries no json tags: the wire format is owned
// by dto.go.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email        string
	Name         string
	PasswordHash string
	Role         string
	database.Timestamps
}

func (User) TableName() string { return "users" }

// Roles returns the user's roles as stored in tokens and sessions.
func (u *User) Roles() []string { return []string{u.Role} }
