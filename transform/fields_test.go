package transform

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/firfircelik/go-intake"
)

func TestParseFloat(t *testing.T) {
	tr := ParseFloat("price")
	in := intake.Record{"price": "1.50", "name": "x"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out["price"] != 1.5 {
		t.Errorf("price = %v, want 1.5", out["price"])
	}
	if out["name"] != "x" {
		t.Errorf("name = %v", out["name"])
	}
}

func TestParseFloatEmpty(t *testing.T) {
	tr := ParseFloat("price")
	in := intake.Record{"price": "  "}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["price"]; ok {
		t.Errorf("empty price should be cleared, got %v", out["price"])
	}
}

func TestParseFloatError(t *testing.T) {
	tr := ParseFloat("price")
	in := intake.Record{"price": "not a number"}
	_, err := tr.Apply(context.Background(), in)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse float") {
		t.Errorf("error should mention parse float: %v", err)
	}
	if !strings.Contains(err.Error(), "price") {
		t.Errorf("error should mention field name: %v", err)
	}
}

func TestParseFloatNonString(t *testing.T) {
	tr := ParseFloat("price")
	in := intake.Record{"price": 5.0}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out["price"] != 5.0 {
		t.Errorf("price = %v, want 5.0", out["price"])
	}
}

func TestParseFloatMissingField(t *testing.T) {
	tr := ParseFloat("price")
	in := intake.Record{"other": "x"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["price"]; ok {
		t.Error("missing field should not appear")
	}
}

func TestParseFloatNoMutation(t *testing.T) {
	tr := ParseFloat("price")
	in := intake.Record{"price": "1.50", "name": "x"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["price"] != "1.50" {
		t.Errorf("input was mutated: price = %v", in["price"])
	}
	out["price"] = "mutated"
	if in["price"] != "1.50" {
		t.Errorf("mutating output affected input: price = %v", in["price"])
	}
}

func TestParseInt(t *testing.T) {
	tr := ParseInt("n")
	in := intake.Record{"n": "42"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out["n"] != int64(42) {
		t.Errorf("n = %v, want int64(42)", out["n"])
	}
}

func TestParseIntError(t *testing.T) {
	tr := ParseInt("n")
	_, err := tr.Apply(context.Background(), intake.Record{"n": "abc"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse int") {
		t.Errorf("error should mention parse int: %v", err)
	}
	if !strings.Contains(err.Error(), "n") {
		t.Errorf("error should mention field name: %v", err)
	}
}

func TestParseIntNoMutation(t *testing.T) {
	tr := ParseInt("n")
	in := intake.Record{"n": "42"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["n"] != "42" {
		t.Errorf("input was mutated: n = %v", in["n"])
	}
	out["n"] = "mutated"
	if in["n"] != "42" {
		t.Errorf("mutating output affected input: n = %v", in["n"])
	}
}

func TestParseBool(t *testing.T) {
	tr := ParseBool("active")
	cases := map[string]bool{
		"true":  true,
		"True":  true,
		"yes":   true,
		"Y":     true,
		"1":     true,
		"false": false,
		"no":    false,
		"0":     false,
	}
	for in, want := range cases {
		rec := intake.Record{"active": in}
		out, err := tr.Apply(context.Background(), rec)
		if err != nil {
			t.Fatalf("ParseBool(%q): %v", in, err)
		}
		if out["active"] != want {
			t.Errorf("ParseBool(%q) = %v, want %v", in, out["active"], want)
		}
	}
}

func TestParseBoolError(t *testing.T) {
	tr := ParseBool("active")
	_, err := tr.Apply(context.Background(), intake.Record{"active": "maybe"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse bool") {
		t.Errorf("error should mention parse bool: %v", err)
	}
	if !strings.Contains(err.Error(), "active") {
		t.Errorf("error should mention field name: %v", err)
	}
}

func TestParseBoolEmpty(t *testing.T) {
	tr := ParseBool("active")
	out, err := tr.Apply(context.Background(), intake.Record{"active": " "})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["active"]; ok {
		t.Error("empty should be cleared")
	}
}

func TestParseBoolNoMutation(t *testing.T) {
	tr := ParseBool("active")
	in := intake.Record{"active": "true"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["active"] != "true" {
		t.Errorf("input was mutated: active = %v", in["active"])
	}
	out["active"] = "mutated"
	if in["active"] != "true" {
		t.Errorf("mutating output affected input: active = %v", in["active"])
	}
}

func TestParseDate(t *testing.T) {
	tr := ParseDate("when", "2006-01-02")
	in := intake.Record{"when": "2024-05-15"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := out["when"].(time.Time)
	if !ok {
		t.Fatalf("when = %T, want time.Time", out["when"])
	}
	if got.Year() != 2024 || got.Month() != time.May || got.Day() != 15 {
		t.Errorf("when = %v, want 2024-05-15", got)
	}
}

func TestParseDateMultipleLayouts(t *testing.T) {
	tr := ParseDate("when", "2006-01-02", "01/02/2006", time.RFC3339)
	for _, c := range []struct {
		in   string
		want time.Time
	}{
		{"2024-05-15", time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)},
		{"05/15/2024", time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)},
		{"2024-05-15T10:30:00Z", time.Date(2024, 5, 15, 10, 30, 0, 0, time.UTC)},
	} {
		out, err := tr.Apply(context.Background(), intake.Record{"when": c.in})
		if err != nil {
			t.Fatalf("ParseDate(%q): %v", c.in, err)
		}
		got, ok := out["when"].(time.Time)
		if !ok {
			t.Fatalf("when = %T, want time.Time", out["when"])
		}
		if !got.Equal(c.want) {
			t.Errorf("ParseDate(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseDateEmpty(t *testing.T) {
	tr := ParseDate("when", "2006-01-02")
	out, err := tr.Apply(context.Background(), intake.Record{"when": "  "})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["when"]; ok {
		t.Error("empty should be cleared")
	}
}

func TestParseDateMissingField(t *testing.T) {
	tr := ParseDate("when", "2006-01-02")
	out, err := tr.Apply(context.Background(), intake.Record{"other": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["when"]; ok {
		t.Error("missing field should not appear")
	}
}

func TestParseDateNonString(t *testing.T) {
	tr := ParseDate("when", "2006-01-02")
	in := intake.Record{"when": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out["when"] != in["when"] {
		t.Errorf("non-string value should be passed through, got %v", out["when"])
	}
}

func TestParseDateInvalid(t *testing.T) {
	tr := ParseDate("when", "2006-01-02")
	_, err := tr.Apply(context.Background(), intake.Record{"when": "not a date"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse date") {
		t.Errorf("error should mention parse date: %v", err)
	}
	if !strings.Contains(err.Error(), "when") {
		t.Errorf("error should mention field name: %v", err)
	}
}

func TestParseDateNoLayouts(t *testing.T) {
	tr := ParseDate("when")
	_, err := tr.Apply(context.Background(), intake.Record{"when": "2024-05-15"})
	if err == nil {
		t.Fatal("expected error for no layouts")
	}
	if !strings.Contains(err.Error(), "no layouts") {
		t.Errorf("error should mention no layouts: %v", err)
	}
}

func TestParseDateNoMutation(t *testing.T) {
	tr := ParseDate("when", "2006-01-02")
	in := intake.Record{"when": "2024-05-15"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["when"] != "2024-05-15" {
		t.Errorf("input was mutated: when = %v", in["when"])
	}
	out["when"] = "mutated"
	if in["when"] != "2024-05-15" {
		t.Errorf("mutating output affected input: when = %v", in["when"])
	}
}

func TestRename(t *testing.T) {
	tr := Rename("a", "b")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if out["b"] != "1" {
		t.Errorf("b = %v", out["b"])
	}
	if _, ok := out["a"]; ok {
		t.Error("a should be gone")
	}
}

func TestRenameSameName(t *testing.T) {
	tr := Rename("a", "a")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "1" {
		t.Errorf("a = %v", out["a"])
	}
}

func TestRenameOverwrite(t *testing.T) {
	tr := Rename("a", "b")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1", "b": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if out["b"] != "1" {
		t.Errorf("b = %v, want 1 (rename should overwrite)", out["b"])
	}
	if _, ok := out["a"]; ok {
		t.Error("a should be gone after rename")
	}
}

func TestRenameMissing(t *testing.T) {
	tr := Rename("a", "b")
	out, err := tr.Apply(context.Background(), intake.Record{"c": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["b"]; ok {
		t.Error("b should not appear when source is missing")
	}
	if out["c"] != "1" {
		t.Errorf("c = %v, should be preserved", out["c"])
	}
}

func TestRenameNoMutation(t *testing.T) {
	tr := Rename("a", "b")
	in := intake.Record{"a": "1", "c": "3"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["a"] != "1" {
		t.Errorf("input was mutated: a = %v", in["a"])
	}
	if _, ok := in["b"]; ok {
		t.Error("input should not have b added")
	}
	out["a"] = "mutated"
	if in["a"] != "1" {
		t.Errorf("mutating output affected input: a = %v", in["a"])
	}
}

func TestDrop(t *testing.T) {
	tr := Drop("a", "b")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1", "b": "2", "c": "3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out["c"] != "3" {
		t.Errorf("got %v", out)
	}
}

func TestDropMissing(t *testing.T) {
	tr := Drop("missing")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "1" {
		t.Errorf("a = %v, want 1", out["a"])
	}
}

func TestDropEmpty(t *testing.T) {
	tr := Drop()
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "1" {
		t.Errorf("a = %v, want 1", out["a"])
	}
}

func TestDropNoMutation(t *testing.T) {
	tr := Drop("a")
	in := intake.Record{"a": "1", "b": "2"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in["a"]; !ok {
		t.Error("input was mutated: a was removed")
	}
	out["b"] = "mutated"
	if in["b"] != "2" {
		t.Errorf("mutating output affected input: b = %v", in["b"])
	}
}

func TestKeep(t *testing.T) {
	tr := Keep("a", "b")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1", "b": "2", "c": "3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Errorf("got %v", out)
	}
	if out["a"] != "1" || out["b"] != "2" {
		t.Errorf("got %v", out)
	}
}

func TestKeepEmpty(t *testing.T) {
	tr := Keep()
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("got %v, want empty", out)
	}
}

func TestKeepNoMutation(t *testing.T) {
	tr := Keep("a")
	in := intake.Record{"a": "1", "b": "2"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["b"] != "2" {
		t.Errorf("input was mutated: b = %v", in["b"])
	}
	out["a"] = "mutated"
	if in["a"] != "1" {
		t.Errorf("mutating output affected input: a = %v", in["a"])
	}
}

func TestAddField(t *testing.T) {
	tr := AddField("a", "x")
	out, err := tr.Apply(context.Background(), intake.Record{"b": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "x" {
		t.Errorf("a = %v, want x", out["a"])
	}
	if out["b"] != "1" {
		t.Errorf("b = %v, want 1", out["b"])
	}
}

func TestAddFieldOverwrite(t *testing.T) {
	tr := AddField("a", "new")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "old"})
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "new" {
		t.Errorf("a = %v, want new (overwrite)", out["a"])
	}
}

func TestAddFieldNilValue(t *testing.T) {
	tr := AddField("a", nil)
	out, err := tr.Apply(context.Background(), intake.Record{"b": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["a"]; !ok {
		t.Error("a should be present with nil value")
	}
	if out["a"] != nil {
		t.Errorf("a = %v, want nil", out["a"])
	}
}

func TestAddFieldNoMutation(t *testing.T) {
	tr := AddField("a", "x")
	in := intake.Record{"a": "old"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["a"] != "old" {
		t.Errorf("input was mutated: a = %v", in["a"])
	}
	out["a"] = "mutated"
	if in["a"] != "old" {
		t.Errorf("mutating output affected input: a = %v", in["a"])
	}
}

func TestCopy(t *testing.T) {
	tr := Copy("a", "b")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "1" || out["b"] != "1" {
		t.Errorf("got %v", out)
	}
}

func TestCopyMissing(t *testing.T) {
	tr := Copy("missing", "b")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["b"]; ok {
		t.Error("b should not appear when source is missing")
	}
	if out["a"] != "1" {
		t.Errorf("a = %v, want 1", out["a"])
	}
}

func TestCopySameName(t *testing.T) {
	tr := Copy("a", "a")
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "1" {
		t.Errorf("a = %v, want 1", out["a"])
	}
}

func TestCopyNoMutation(t *testing.T) {
	tr := Copy("a", "b")
	in := intake.Record{"a": "1"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in["b"]; ok {
		t.Error("input should not have b added")
	}
	out["a"] = "mutated"
	if in["a"] != "1" {
		t.Errorf("mutating output affected input: a = %v", in["a"])
	}
}

func TestMapField(t *testing.T) {
	tr := MapField("n", func(v any) (any, error) {
		return strings.ToUpper(v.(string)), nil
	})
	out, err := tr.Apply(context.Background(), intake.Record{"n": "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if out["n"] != "ABC" {
		t.Errorf("n = %v, want ABC", out["n"])
	}
}

func TestMapFieldMissing(t *testing.T) {
	tr := MapField("missing", func(v any) (any, error) {
		return "x", nil
	})
	out, err := tr.Apply(context.Background(), intake.Record{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["missing"]; ok {
		t.Error("missing field should not be added")
	}
	if out["a"] != "1" {
		t.Errorf("a = %v, want 1", out["a"])
	}
}

func TestMapFieldError(t *testing.T) {
	tr := MapField("n", func(v any) (any, error) {
		return nil, errors.New("boom")
	})
	_, err := tr.Apply(context.Background(), intake.Record{"n": "1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "map field") {
		t.Errorf("error should mention map field: %v", err)
	}
	if !strings.Contains(err.Error(), "n") {
		t.Errorf("error should mention field name: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error should wrap inner error: %v", err)
	}
}

func TestMapFieldNilValue(t *testing.T) {
	called := false
	tr := MapField("n", func(v any) (any, error) {
		called = true
		if v != nil {
			t.Errorf("expected nil, got %v", v)
		}
		return "x", nil
	})
	out, err := tr.Apply(context.Background(), intake.Record{"n": nil})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("function was not called")
	}
	if out["n"] != "x" {
		t.Errorf("n = %v, want x", out["n"])
	}
}

func TestMapFieldNoMutation(t *testing.T) {
	tr := MapField("n", func(v any) (any, error) {
		return "transformed", nil
	})
	in := intake.Record{"n": "1"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["n"] != "1" {
		t.Errorf("input was mutated: n = %v", in["n"])
	}
	out["n"] = "mutated"
	if in["n"] != "1" {
		t.Errorf("mutating output affected input: n = %v", in["n"])
	}
}

func TestTransformErrorPropagates(t *testing.T) {
	tr := ParseFloat("price")
	_, err := tr.Apply(context.Background(), intake.Record{"price": "abc"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, err) {
		t.Fatal("error should be non-nil")
	}
}
