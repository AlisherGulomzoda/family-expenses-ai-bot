// Package clock abstracts the current time so that "today" and "yesterday"
// can be fixed in tests and in the agent evaluation suite.
package clock

import "time"

// Clock tells the current time.
type Clock interface {
	Now() time.Time
}

// System is the real wall clock.
type System struct{}

// Now returns the current time.
func (System) Now() time.Time { return time.Now() }

// Fixed always returns the same instant.
type Fixed time.Time

// Now returns the fixed instant.
func (f Fixed) Now() time.Time { return time.Time(f) }

// Today returns the current calendar date in loc as midnight UTC.
// Dates are kept as midnight UTC everywhere so they compare and store
// (PostgreSQL DATE) without time zone surprises.
func Today(c Clock, loc *time.Location) time.Time {
	return DateOf(c.Now().In(loc))
}

// DateOf returns the calendar date of t (in t's location) as midnight UTC.
func DateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
