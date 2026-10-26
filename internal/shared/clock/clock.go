// Package clock lets services read the time through an interface, so tests
// can freeze it.
package clock

import "time"

// Clock returns the current time.
type Clock interface {
	Now() time.Time
}

// System is the real clock, always in UTC.
type System struct{}

func (System) Now() time.Time { return time.Now().UTC() }

// Fixed always returns T. Use it in tests.
type Fixed struct{ T time.Time }

func (f Fixed) Now() time.Time { return f.T }
