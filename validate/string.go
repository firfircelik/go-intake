package validate

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/firfircelik/go-intake"
)

// MinLen is a thin convenience wrapper. It returns a Validator that
// fails when the string value of field is shorter than n runes.
// Non-string values fail with a type-mismatch error. Missing
// fields fail with a "required for length check" message; pair
// MinLen with Required when presence is the caller's
// responsibility.
func MinLen(field string, n int) Validator {
	return lengthCheck{field: field, min: &n}
}

// MaxLen is a thin convenience wrapper. It returns a Validator
// that fails when the string value of field is longer than n
// runes. See MinLen for missing-field behavior.
func MaxLen(field string, n int) Validator {
	return lengthCheck{field: field, max: &n}
}

// Len is a thin convenience wrapper. It returns a Validator that
// fails when the string value of field is not exactly n runes
// long. See MinLen for missing-field behavior.
func Len(field string, n int) Validator {
	return lengthCheck{field: field, exact: &n}
}

type lengthCheck struct {
	field string
	min   *int
	max   *int
	exact *int
}

func (l lengthCheck) Validate(_ context.Context, rec intake.Record) error {
	v, ok := rec[l.field]
	if !ok {
		return &ValidationError{Field: l.field, Rule: "length", Message: "field is required for length check"}
	}
	s, ok := v.(string)
	if !ok {
		return &ValidationError{Field: l.field, Rule: "length", Message: fmt.Sprintf("value is not a string: %T", v), Value: v}
	}
	runes := len([]rune(s))
	if l.exact != nil && runes != *l.exact {
		return &ValidationError{Field: l.field, Rule: "length", Message: fmt.Sprintf("must be exactly %d runes, got %d", *l.exact, runes), Value: v}
	}
	if l.min != nil && runes < *l.min {
		return &ValidationError{Field: l.field, Rule: "min_length", Message: fmt.Sprintf("must be at least %d runes, got %d", *l.min, runes), Value: v}
	}
	if l.max != nil && runes > *l.max {
		return &ValidationError{Field: l.field, Rule: "max_length", Message: fmt.Sprintf("must be at most %d runes, got %d", *l.max, runes), Value: v}
	}
	return nil
}

// Regex returns a Validator that fails when the string value of
// field does not match the regular expression. The pattern is
// compiled once at construction; a compile error produces a
// validator that always fails with a "compile pattern" error
// rather than panicking.
func Regex(field, pattern string) Validator {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return invalidValidator{err: fmt.Errorf("compile pattern: %w", err)}
	}
	return regex{field: field, re: re}
}

type regex struct {
	field string
	re    *regexp.Regexp
}

func (m regex) Validate(_ context.Context, rec intake.Record) error {
	v, ok := rec[m.field]
	if !ok {
		return &ValidationError{Field: m.field, Rule: "regex", Message: "field is required"}
	}
	s, ok := v.(string)
	if !ok {
		return &ValidationError{Field: m.field, Rule: "regex", Message: fmt.Sprintf("value is not a string: %T", v), Value: v}
	}
	if !m.re.MatchString(s) {
		return &ValidationError{Field: m.field, Rule: "regex", Message: "value does not match pattern", Value: v}
	}
	return nil
}

type invalidValidator struct{ err error }

func (i invalidValidator) Validate(_ context.Context, _ intake.Record) error {
	return i.err
}

// Enum returns a Validator that fails when the string value of
// field is not present in allowed. The comparison is
// case-sensitive. The error message lists allowed values in
// lexicographic order so output is stable.
func Enum(field string, allowed ...string) Validator {
	set := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		set[a] = struct{}{}
	}
	return enum{field: field, set: set}
}

type enum struct {
	field string
	set   map[string]struct{}
}

func (o enum) Validate(_ context.Context, rec intake.Record) error {
	v, ok := rec[o.field]
	if !ok {
		return &ValidationError{Field: o.field, Rule: "enum", Message: "field is required"}
	}
	s, ok := v.(string)
	if !ok {
		return &ValidationError{Field: o.field, Rule: "enum", Message: fmt.Sprintf("value is not a string: %T", v), Value: v}
	}
	if _, ok := o.set[s]; !ok {
		keys := make([]string, 0, len(o.set))
		for k := range o.set {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return &ValidationError{Field: o.field, Rule: "enum", Message: fmt.Sprintf("must be one of %v", keys), Value: v}
	}
	return nil
}

// Email is a thin convenience wrapper around Regex. It returns a
// Validator that fails when the string value of field is not a
// syntactically valid email address. The pattern is intentionally
// permissive; for stricter checks, build a Regex with your own
// pattern.
func Email(field string) Validator {
	return Regex(field, emailPattern)
}

const emailPattern = `^[^\s@]+@[^\s@]+\.[^\s@]+$`

// URL returns a Validator that fails when the string value of
// field is not a syntactically valid absolute URL with scheme
// http or https. The check uses net/url.Parse and verifies that a
// host is present.
func URL(field string) Validator {
	return urlValidator{field: field}
}

type urlValidator struct{ field string }

func (u urlValidator) Validate(_ context.Context, rec intake.Record) error {
	v, ok := rec[u.field]
	if !ok {
		return &ValidationError{Field: u.field, Rule: "url", Message: "field is required"}
	}
	s, ok := v.(string)
	if !ok {
		return &ValidationError{Field: u.field, Rule: "url", Message: fmt.Sprintf("value is not a string: %T", v), Value: v}
	}
	s = strings.TrimSpace(s)
	parsed, err := url.Parse(s)
	if err != nil {
		return &ValidationError{Field: u.field, Rule: "url", Message: "not a valid URL", Value: v}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return &ValidationError{Field: u.field, Rule: "url", Message: "scheme must be http or https", Value: v}
	}
	if parsed.Host == "" {
		return &ValidationError{Field: u.field, Rule: "url", Message: "host is required", Value: v}
	}
	return nil
}
