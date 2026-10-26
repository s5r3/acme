// Package id generates primary keys. UUID v7 keys are time-ordered, which
// keeps B-tree indexes compact compared with random v4 keys.
package id

import "github.com/google/uuid"

// New returns a new UUID v7.
func New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
