package validate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/firfircelik/go-intake"
)

func TestRequired(t *testing.T) {
	v := Required("name")
	cases := []struct {
		rec  intake.Record
		want bool
	}{
		{intake.Record{"name": "x"}, true},
		{intake.Record{"name": ""}, false},
		{intake.Record{"name": "  "}, false},
		{intake.Record{}, false},
		{intake.Record{"name": nil}, false},
	}
	for i, c := range cases {
		err := v.Validate(context.Background(), c.rec)
		if (err == nil) != c.want {
			t.Errorf("case %d: Validate err = %v, want pass=%v", i, err, c.want)
		}
	}
}

func TestRequiredValue(t *testing.T) {
	v := Required("name")
	err := v.Validate(context.Background(), intake.Record{"name": "  "})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Value != "  " {
		t.Errorf("Value = %v, want whitespace string", ve.Value)
	}
}

func TestPresent(t *testing.T) {
	v := Present("name")
	cases := []struct {
		rec  intake.Record
		want bool
	}{
		{intake.Record{"name": "x"}, true},
		{intake.Record{"name": ""}, true},
		{intake.Record{"name": nil}, false},
		{intake.Record{}, false},
	}
	for i, c := range cases {
		err := v.Validate(context.Background(), c.rec)
		if (err == nil) != c.want {
			t.Errorf("case %d: err = %v, want pass=%v", i, err, c.want)
		}
	}
}

func TestForbidden(t *testing.T) {
	v := Forbidden("secret")
	cases := []struct {
		rec  intake.Record
		want bool
	}{
		{intake.Record{}, true},
		{intake.Record{"secret": nil}, true},
		{intake.Record{"secret": ""}, true},
		{intake.Record{"secret": "leaked"}, false},
	}
	for i, c := range cases {
		err := v.Validate(context.Background(), c.rec)
		if (err == nil) != c.want {
			t.Errorf("case %d: err = %v, want pass=%v", i, err, c.want)
		}
	}
}

func TestNotFuture(t *testing.T) {
	v := NotFuture("when")
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	cases := []struct {
		rec  intake.Record
		want bool
	}{
		{intake.Record{"when": past}, true},
		{intake.Record{"when": future}, false},
		{intake.Record{}, false},
		{intake.Record{"when": "yesterday"}, false},
		{intake.Record{"when": nil}, false},
	}
	for i, c := range cases {
		err := v.Validate(context.Background(), c.rec)
		if (err == nil) != c.want {
			t.Errorf("case %d: err = %v, want pass=%v", i, err, c.want)
		}
	}
}

func TestNotFutureValue(t *testing.T) {
	v := NotFuture("when")
	future := time.Now().Add(time.Hour)
	err := v.Validate(context.Background(), intake.Record{"when": future})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Value != future {
		t.Errorf("Value = %v, want future time", ve.Value)
	}
}

func TestCustom(t *testing.T) {
	v := Custom("name_check", func(r intake.Record) error {
		if s, _ := r["a"].(string); s == "bad" {
			return errors.New("no")
		}
		return nil
	})
	if err := v.Validate(context.Background(), intake.Record{"a": "ok"}); err != nil {
		t.Errorf("ok should pass, got %v", err)
	}
	if err := v.Validate(context.Background(), intake.Record{"a": "bad"}); err == nil {
		t.Error("bad should fail")
	}
}

func TestCustomNameAsRule(t *testing.T) {
	v := Custom("my_rule", func(r intake.Record) error {
		return errors.New("nope")
	})
	err := v.Validate(context.Background(), intake.Record{})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Rule != "my_rule" {
		t.Errorf("Rule = %q, want my_rule", ve.Rule)
	}
	if ve.Message != "nope" {
		t.Errorf("Message = %q, want nope", ve.Message)
	}
}

func TestCustomForwardsValidationError(t *testing.T) {
	// If fn returns a *ValidationError, it is forwarded verbatim so
	// callers can supply Field, Message, and Value.
	v := Custom("wrapper", func(r intake.Record) error {
		return &ValidationError{Field: "a", Rule: "inner", Message: "inner msg", Value: r["a"]}
	})
	err := v.Validate(context.Background(), intake.Record{"a": "x"})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Field != "a" || ve.Rule != "inner" || ve.Message != "inner msg" {
		t.Errorf("forwarded = %+v, want field=a rule=inner msg=inner msg", ve)
	}
	if ve.Value != "x" {
		t.Errorf("Value = %v, want x", ve.Value)
	}
}

func TestValidationErrorMessage(t *testing.T) {
	v := Required("name")
	err := v.Validate(context.Background(), intake.Record{})
	if err == nil {
		t.Fatal("expected error")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error is not ValidationError: %T", err)
	}
	if ve.Field != "name" {
		t.Errorf("Field = %q", ve.Field)
	}
	if ve.Rule != "required" {
		t.Errorf("Rule = %q", ve.Rule)
	}
}
