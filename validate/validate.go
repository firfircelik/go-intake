// Package validate provides Validators for intake pipelines.
//
// Validators inspect a Record and report whether it is acceptable.
// They are wired into a pipeline via intake.Pipeline.Validate.
// Every validator in this package honours the read-only contract
// documented on intake.Validator: Validate does not mutate the
// record it receives.
//
// The package exports two flavours of validator:
//
//   - A small set of core, composable primitives (Required, Min,
//     Max, Between, Regex, Enum, NotFuture, Custom) that cover the
//     common cases without overlap.
//   - A handful of thin convenience wrappers (Present, Forbidden,
//     ExclusiveRange, MinLen, MaxLen, Len, Email, URL) that read
//     more clearly in domain code and never add behaviour that
//     cannot be expressed with the core set.
//
// When multiple validators reject the same record, the pipeline
// collects every error and sends a single InvalidRecord event to
// the Quarantine. Use errors.As to recover the per-validator
// ValidationError entries.
package validate

import "github.com/firfircelik/go-intake"

// ValidationError describes why a record failed validation. It is
// returned by validators created by this package so callers can use
// errors.As to extract a structured reason.
type ValidationError struct {
	// Field is the name of the field that triggered the failure. It
	// may be empty for record-level rules.
	Field string
	// Rule is the short name of the rule that failed (e.g. "required",
	// "min", "regex").
	Rule string
	// Message is a human-readable description of the failure.
	Message string
	// Value is the actual value that triggered the failure. It is
	// nil for record-level rules or when the field is missing.
	// Validators that check types (Min, Max, Between, Regex, Enum,
	// NotFuture, etc.) populate this so downstream consumers can
	// inspect what was rejected without re-reading the record.
	Value any
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field == "" {
		return e.Rule + ": " + e.Message
	}
	return e.Field + ": " + e.Rule + ": " + e.Message
}

// Validator inspects a Record and reports whether it is acceptable.
// See intake.Validator for the full contract.
type Validator = intake.Validator
