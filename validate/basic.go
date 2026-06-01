package validate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/firfircelik/go-intake"
)

// Required returns a Validator that fails when field is missing,
// nil, an empty string, or a whitespace-only string.
func Required(field string) Validator {
	return required{field: field}
}

type required struct{ field string }

func (r required) Validate(_ context.Context, rec intake.Record) error {
	v, ok := rec[r.field]
	if !ok || v == nil {
		return &ValidationError{Field: r.field, Rule: "required", Message: "field is required"}
	}
	if s, ok := v.(string); ok && trim(s) == "" {
		return &ValidationError{Field: r.field, Rule: "required", Message: "field is required", Value: v}
	}
	return nil
}

// Present is a thin convenience wrapper. It returns a Validator that
// fails only when field is missing or explicitly nil. Zero values
// (empty string, 0, false) are accepted. For most "field is not
// empty" use cases, prefer Required.
func Present(field string) Validator {
	return present{field: field}
}

type present struct{ field string }

func (p present) Validate(_ context.Context, rec intake.Record) error {
	v, ok := rec[p.field]
	if !ok || v == nil {
		return &ValidationError{Field: p.field, Rule: "present", Message: "field must be present"}
	}
	return nil
}

// Forbidden is a thin convenience wrapper. It returns a Validator
// that fails when field has a non-nil, non-empty value. Missing or
// nil values pass; whitespace-only strings also pass. For most
// "field must be empty" use cases, this reads more clearly than the
// core primitives, but it has no equivalent in the core set.
func Forbidden(field string) Validator {
	return forbidden{field: field}
}

type forbidden struct{ field string }

func (f forbidden) Validate(_ context.Context, rec intake.Record) error {
	v, ok := rec[f.field]
	if !ok || v == nil {
		return nil
	}
	if s, ok := v.(string); ok && trim(s) == "" {
		return nil
	}
	return &ValidationError{Field: f.field, Rule: "forbidden", Message: "field must not be set", Value: v}
}

// NotFuture returns a Validator that fails when field is a time.Time
// later than the current wall clock at validation time. Missing
// fields and non-time values produce a type-mismatch error. Use
// transform.ParseDate first when the source carries string dates.
func NotFuture(field string) Validator {
	return notFuture{field: field}
}

type notFuture struct{ field string }

func (n notFuture) Validate(_ context.Context, rec intake.Record) error {
	v, ok := rec[n.field]
	if !ok {
		return &ValidationError{Field: n.field, Rule: "not_future", Message: "field is required"}
	}
	t, ok := v.(time.Time)
	if !ok {
		return &ValidationError{Field: n.field, Rule: "not_future", Message: fmt.Sprintf("value is not a time.Time: %T", v), Value: v}
	}
	if t.After(time.Now()) {
		return &ValidationError{Field: n.field, Rule: "not_future", Message: "value is in the future", Value: v}
	}
	return nil
}

// Custom returns a Validator that invokes fn for each record. The
// validator fails when fn returns a non-nil error. The returned
// error is wrapped as a ValidationError whose Rule is name; if fn
// already returns a *ValidationError it is forwarded as-is so the
// caller can supply a Field, Message, and Value.
//
// fn must not mutate the record. The pipeline never mutates the
// record either, so fn can safely read it more than once or keep
// a pointer to it for use in error messages.
func Custom(name string, fn func(rec intake.Record) error) Validator {
	return custom{name: name, fn: fn}
}

type custom struct {
	name string
	fn   func(rec intake.Record) error
}

func (c custom) Validate(_ context.Context, rec intake.Record) error {
	err := c.fn(rec)
	if err == nil {
		return nil
	}
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve
	}
	return &ValidationError{Rule: c.name, Message: err.Error()}
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
