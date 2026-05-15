package model

// ValidationError represents a user-facing input validation failure.
// Handlers should map this to HTTP 400.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }
