package validate

import (
	"context"
	"fmt"

	"github.com/firfircelik/go-intake"
)

// Min returns a Validator that fails when the numeric value of
// field is strictly less than min. Non-numeric values fail with a
// type-mismatch error. Missing fields fail with a "required for
// range check" message; pair Min with Required when presence is
// the caller's responsibility.
func Min(field string, min float64) Validator {
	return numericCheck{field: field, min: &min, exclusive: false}
}

// Max returns a Validator that fails when the numeric value of
// field is strictly greater than max. See Min for missing-field
// behavior.
func Max(field string, max float64) Validator {
	return numericCheck{field: field, max: &max, exclusive: false}
}

// Between returns a Validator that fails when the numeric value of
// field is outside the inclusive range [min, max]. See Min for
// missing-field behavior.
func Between(field string, min, max float64) Validator {
	return numericCheck{field: field, min: &min, max: &max, exclusive: false}
}

// ExclusiveRange is a thin convenience wrapper. It returns a
// Validator that fails unless the numeric value of field is
// strictly inside (min, max). See Min for missing-field behavior.
// Callers that prefer the core primitives can express the same
// rule with Between plus a Min/Max exclusion, but this reads
// more clearly when both ends are exclusive.
func ExclusiveRange(field string, min, max float64) Validator {
	return numericCheck{field: field, min: &min, max: &max, exclusive: true}
}

type numericCheck struct {
	field     string
	min       *float64
	max       *float64
	exclusive bool
}

func (n numericCheck) Validate(_ context.Context, rec intake.Record) error {
	v, ok := rec[n.field]
	if !ok {
		return &ValidationError{Field: n.field, Rule: "between", Message: "field is required for range check"}
	}
	f, err := toFloat(v)
	if err != nil {
		return &ValidationError{Field: n.field, Rule: "between", Message: err.Error(), Value: v}
	}
	if n.min != nil {
		if n.exclusive {
			if f <= *n.min {
				return &ValidationError{Field: n.field, Rule: "min", Message: fmt.Sprintf("must be greater than %g", *n.min), Value: v}
			}
		} else if f < *n.min {
			return &ValidationError{Field: n.field, Rule: "min", Message: fmt.Sprintf("must be >= %g", *n.min), Value: v}
		}
	}
	if n.max != nil {
		if n.exclusive {
			if f >= *n.max {
				return &ValidationError{Field: n.field, Rule: "max", Message: fmt.Sprintf("must be less than %g", *n.max), Value: v}
			}
		} else if f > *n.max {
			return &ValidationError{Field: n.field, Rule: "max", Message: fmt.Sprintf("must be <= %g", *n.max), Value: v}
		}
	}
	return nil
}

func toFloat(v any) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case float32:
		return float64(t), nil
	case int:
		return float64(t), nil
	case int8:
		return float64(t), nil
	case int16:
		return float64(t), nil
	case int32:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case uint:
		return float64(t), nil
	case uint8:
		return float64(t), nil
	case uint16:
		return float64(t), nil
	case uint32:
		return float64(t), nil
	case uint64:
		return float64(t), nil
	case string:
		// Strings are not silently coerced. Apply a numeric
		// transform (e.g. transform.ParseFloat) first.
		return 0, fmt.Errorf("value is a string; apply a numeric transform first")
	}
	return 0, fmt.Errorf("value is not numeric: %T", v)
}
