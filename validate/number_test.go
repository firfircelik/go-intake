package validate

import (
	"context"
	"errors"
	"testing"

	"github.com/firfircelik/go-intake"
)

func TestMin(t *testing.T) {
	v := Min("n", 0)
	cases := []struct {
		rec  intake.Record
		want bool
	}{
		{intake.Record{"n": 1.0}, true},
		{intake.Record{"n": 0.0}, true},
		{intake.Record{"n": -1.0}, false},
		{intake.Record{"n": int64(5)}, true},
		{intake.Record{"n": int(-1)}, false},
		{intake.Record{}, false},
		{intake.Record{"n": "x"}, false},
	}
	for i, c := range cases {
		err := v.Validate(context.Background(), c.rec)
		if (err == nil) != c.want {
			t.Errorf("case %d: err = %v, want pass=%v", i, err, c.want)
		}
	}
}

func TestMax(t *testing.T) {
	v := Max("n", 10)
	if err := v.Validate(context.Background(), intake.Record{"n": 5.0}); err != nil {
		t.Errorf("5 should pass: %v", err)
	}
	if err := v.Validate(context.Background(), intake.Record{"n": 10.0}); err != nil {
		t.Errorf("10 should pass: %v", err)
	}
	if err := v.Validate(context.Background(), intake.Record{"n": 11.0}); err == nil {
		t.Error("11 should fail")
	}
}

func TestBetween(t *testing.T) {
	v := Between("n", 0, 10)
	if err := v.Validate(context.Background(), intake.Record{"n": 5.0}); err != nil {
		t.Error("5 should pass")
	}
	if err := v.Validate(context.Background(), intake.Record{"n": -1.0}); err == nil {
		t.Error("-1 should fail")
	}
	if err := v.Validate(context.Background(), intake.Record{"n": 11.0}); err == nil {
		t.Error("11 should fail")
	}
	if err := v.Validate(context.Background(), intake.Record{"n": 0.0}); err != nil {
		t.Error("0 should pass (inclusive)")
	}
	if err := v.Validate(context.Background(), intake.Record{"n": 10.0}); err != nil {
		t.Error("10 should pass (inclusive)")
	}
}

func TestExclusiveRange(t *testing.T) {
	v := ExclusiveRange("n", 0, 10)
	if err := v.Validate(context.Background(), intake.Record{"n": 0.0}); err == nil {
		t.Error("0 should fail (exclusive)")
	}
	if err := v.Validate(context.Background(), intake.Record{"n": 10.0}); err == nil {
		t.Error("10 should fail (exclusive)")
	}
	if err := v.Validate(context.Background(), intake.Record{"n": 5.0}); err != nil {
		t.Error("5 should pass")
	}
}

func TestMinValue(t *testing.T) {
	v := Min("n", 0)
	err := v.Validate(context.Background(), intake.Record{"n": -1.0})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Value != -1.0 {
		t.Errorf("Value = %v, want -1.0", ve.Value)
	}
}

func TestMaxValue(t *testing.T) {
	v := Max("n", 10)
	err := v.Validate(context.Background(), intake.Record{"n": 11})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Value != 11 {
		t.Errorf("Value = %v, want 11", ve.Value)
	}
}

func TestBetweenValue(t *testing.T) {
	v := Between("n", 0, 10)
	err := v.Validate(context.Background(), intake.Record{"n": -5.0})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Value != -5.0 {
		t.Errorf("Value = %v, want -5.0", ve.Value)
	}
}

func TestBetweenTypeMismatchValue(t *testing.T) {
	v := Between("n", 0, 10)
	err := v.Validate(context.Background(), intake.Record{"n": "x"})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Value != "x" {
		t.Errorf("Value = %v, want x", ve.Value)
	}
}
