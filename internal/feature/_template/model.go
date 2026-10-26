package __name__

import (
	"github.com/google/uuid"

	"example.com/acme/internal/platform/database"
)

// __Name__ is the __name__s table. Add a migration: make migrate-new name=create___name__s
type __Name__ struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name string
	database.Timestamps
}

func (__Name__) TableName() string { return "__name__s" }
