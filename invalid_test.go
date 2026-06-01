package intake

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestStageConstants(t *testing.T) {
	if StageTransform != "transform" {
		t.Errorf("StageTransform = %q, want transform", StageTransform)
	}
	if StageValidation != "validation" {
		t.Errorf("StageValidation = %q, want validation", StageValidation)
	}
}

func TestMultiErrorError(t *testing.T) {
	m := &MultiError{Errors: []error{
		errors.New("first"),
		errors.New("second"),
	}}
	got := m.Error()
	if got == "" {
		t.Fatal("Error() returned empty string")
	}
	if !strings.Contains(got, "2 errors") {
		t.Errorf("Error() = %q, want to mention count", got)
	}
	if !strings.Contains(got, "first") || !strings.Contains(got, "second") {
		t.Errorf("Error() = %q, want to mention each error", got)
	}
}

func TestMultiErrorEmpty(t *testing.T) {
	m := &MultiError{}
	if m.Error() != "no errors" {
		t.Errorf("Error() = %q, want 'no errors'", m.Error())
	}
}

func TestMultiErrorUnwrap(t *testing.T) {
	inner := []error{
		errors.New("a"),
		errors.New("b"),
	}
	m := &MultiError{Errors: inner}
	got := m.Unwrap()
	if len(got) != len(inner) {
		t.Fatalf("Unwrap() returned %d errors, want %d", len(got), len(inner))
	}
	for i := range inner {
		if got[i] != inner[i] {
			t.Errorf("Unwrap()[%d] = %v, want %v", i, got[i], inner[i])
		}
	}
}

func TestMultiErrorUnwrapNilSafe(t *testing.T) {
	var m *MultiError
	if got := m.Unwrap(); got != nil {
		t.Errorf("nil MultiError Unwrap() = %v, want nil", got)
	}
}

func TestMultiErrorUnwrapDefensiveCopy(t *testing.T) {
	// The Unwrap contract must return a fresh copy so callers
	// cannot mutate the pipeline's internal state.
	m := &MultiError{Errors: []error{errors.New("a")}}
	got := m.Unwrap()
	got[0] = errors.New("mutated")
	if m.Errors[0].Error() != "a" {
		t.Errorf("MultiError internal state was mutated through Unwrap result")
	}
}

func TestMultiErrorWithErrorsAs(t *testing.T) {
	m := &MultiError{Errors: []error{
		fmt.Errorf("wrapped: %w", errors.New("inner")),
	}}
	var target *MultiError
	if !errors.As(error(m), &target) {
		t.Fatal("errors.As should extract *MultiError")
	}
	if len(target.Errors) != 1 {
		t.Errorf("target.Errors has %d entries, want 1", len(target.Errors))
	}
}

func TestMultiErrorIsFindsCollectedError(t *testing.T) {
	sentinel := errors.New("sentinel")
	m := &MultiError{Errors: []error{errors.New("a"), sentinel, errors.New("b")}}
	if !errors.Is(m, sentinel) {
		t.Errorf("errors.Is(MultiError, sentinel) = false, want true")
	}
}

func TestMultiErrorIsFindsWrappedError(t *testing.T) {
	inner := errors.New("inner")
	m := &MultiError{Errors: []error{fmt.Errorf("wrap: %w", inner)}}
	if !errors.Is(m, inner) {
		t.Errorf("errors.Is should walk wrapped errors inside MultiError")
	}
}

func TestMultiErrorIsNotFound(t *testing.T) {
	m := &MultiError{Errors: []error{errors.New("a")}}
	if errors.Is(m, errors.New("b")) {
		t.Errorf("errors.Is reported match for a non-collected error")
	}
}

func TestMultiErrorIsNilSafe(t *testing.T) {
	var m *MultiError
	if m.Is(errors.New("x")) {
		t.Errorf("nil MultiError Is() = true, want false")
	}
	if errors.Is(m, errors.New("x")) {
		t.Errorf("errors.Is(nil MultiError, x) = true, want false")
	}
}

func TestMultiErrorIsSkipsNilEntries(t *testing.T) {
	a := errors.New("a")
	m := &MultiError{Errors: []error{nil, a}}
	if m.Is(nil) {
		t.Errorf("Is(nil) on populated MultiError returned true")
	}
	if !m.Is(a) {
		t.Errorf("Is should skip nil entries and still find the real one")
	}
}

func TestInvalidRecordFields(t *testing.T) {
	rec := Record{"a": 1}
	errs := []error{errors.New("e1"), errors.New("e2")}
	info := InvalidRecord{
		Record:    rec,
		Errors:    errs,
		Stage:     StageValidation,
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if info.Record["a"] != 1 {
		t.Errorf("Record lost data")
	}
	if len(info.Errors) != 2 {
		t.Errorf("Errors lost")
	}
	if info.Stage != StageValidation {
		t.Errorf("Stage = %q, want %q", info.Stage, StageValidation)
	}
	if info.Timestamp.IsZero() {
		t.Errorf("Timestamp was lost")
	}
}
