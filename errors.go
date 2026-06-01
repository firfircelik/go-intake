package intake

import "errors"

// ErrorKind classifies the source of an error inside a pipeline run.
type ErrorKind string

const (
	// ErrKindSource indicates a source-level failure (open, read, close).
	ErrKindSource ErrorKind = "source"
	// ErrKindTransform indicates a transformer rejected a record.
	ErrKindTransform ErrorKind = "transform"
	// ErrKindValidation indicates a validator rejected a record.
	ErrKindValidation ErrorKind = "validation"
	// ErrKindSink indicates a sink-level failure (open, write, close).
	ErrKindSink ErrorKind = "sink"
	// ErrKindQuarantine indicates a quarantine sink failure.
	ErrKindQuarantine ErrorKind = "quarantine"
	// ErrKindConfig indicates a pipeline misconfiguration.
	ErrKindConfig ErrorKind = "config"
)

// Error is the structured error type returned by intake components.
// It carries the originating kind, a stable machine-readable code,
// a human-readable message, and an optional underlying cause.
type Error struct {
	Kind    ErrorKind
	Code    string
	Message string
	Cause   error
}

// NewError constructs a new Error.
func NewError(kind ErrorKind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Wrap returns a new Error that wraps the given cause.
func Wrap(kind ErrorKind, code, message string, cause error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Cause: cause}
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return string(e.Kind) + ": " + e.Code + ": " + e.Message + ": " + e.Cause.Error()
	}
	return string(e.Kind) + ": " + e.Code + ": " + e.Message
}

// Unwrap returns the underlying cause so errors.Is and errors.As work.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is supports matching by ErrorKind and Code. It is used by
// errors.Is to compare two *Error values: if both have the same
// Kind and Code, they are considered equal.
func (e *Error) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	if t, ok := target.(*Error); ok {
		return e.Kind == t.Kind && e.Code == t.Code
	}
	return false
}

// IsKind reports whether err is an *Error (or wraps one) with the
// given kind.
func IsKind(err error, kind ErrorKind) bool {
	var ie *Error
	if errors.As(err, &ie) {
		return ie.Kind == kind
	}
	return false
}
