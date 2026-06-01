package transform

import (
	"context"
	"strings"

	"github.com/firfircelik/go-intake"
)

// TrimStrings returns a Transformer that trims leading and trailing
// whitespace from string values in the record. If specific fields
// are provided, only those fields are trimmed; otherwise every
// string value is trimmed. Non-string values are left untouched.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func TrimStrings(fields ...string) intake.Transformer {
	return &trimStrings{fields: fields, all: len(fields) == 0}
}

type trimStrings struct {
	fields []string
	all    bool
}

func (t *trimStrings) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	out := make(intake.Record, len(r))
	for k, v := range r {
		if t.all || contains(t.fields, k) {
			if s, ok := v.(string); ok {
				out[k] = strings.TrimSpace(s)
				continue
			}
		}
		out[k] = v
	}
	return out, nil
}

// LowerStrings returns a Transformer that lowercases string values
// in the record. If no fields are provided, all string values are
// lowercased. Non-string values are left untouched.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func LowerStrings(fields ...string) intake.Transformer {
	return &caseStrings{fields: fields, all: len(fields) == 0, lower: true}
}

// UpperStrings returns a Transformer that uppercases string values
// in the record. If no fields are provided, all string values are
// uppercased. Non-string values are left untouched.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func UpperStrings(fields ...string) intake.Transformer {
	return &caseStrings{fields: fields, all: len(fields) == 0, lower: false}
}

type caseStrings struct {
	fields []string
	all    bool
	lower  bool
}

func (c *caseStrings) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	out := make(intake.Record, len(r))
	for k, v := range r {
		if c.all || contains(c.fields, k) {
			if s, ok := v.(string); ok {
				if c.lower {
					out[k] = strings.ToLower(s)
				} else {
					out[k] = strings.ToUpper(s)
				}
				continue
			}
		}
		out[k] = v
	}
	return out, nil
}

func contains(slice []string, s string) bool {
	for _, x := range slice {
		if x == s {
			return true
		}
	}
	return false
}
