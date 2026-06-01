package transform

import (
	"context"
	"github.com/firfircelik/go-intake"
	"testing"
)

func TestTrimStringsAllFields(t *testing.T) {
	tr := TrimStrings()
	in := intake.Record{"a": "  x  ", "b": "y\n", "c": 1}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "x" {
		t.Errorf("a = %v", out["a"])
	}
	if out["b"] != "y" {
		t.Errorf("b = %v", out["b"])
	}
	if out["c"] != 1 {
		t.Errorf("c = %v", out["c"])
	}
}

func TestTrimStringsSpecificFields(t *testing.T) {
	tr := TrimStrings("a")
	in := intake.Record{"a": "  x  ", "b": "  y  "}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "x" {
		t.Errorf("a = %v", out["a"])
	}
	if out["b"] != "  y  " {
		t.Errorf("b = %v, should not be trimmed", out["b"])
	}
}

func TestLowerStrings(t *testing.T) {
	tr := LowerStrings()
	in := intake.Record{"a": "ABC", "b": "xYz"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "abc" || out["b"] != "xyz" {
		t.Errorf("got %v", out)
	}
}

func TestUpperStrings(t *testing.T) {
	tr := UpperStrings("a")
	in := intake.Record{"a": "abc", "b": "xyz"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out["a"] != "ABC" {
		t.Errorf("a = %v", out["a"])
	}
	if out["b"] != "xyz" {
		t.Errorf("b should be unchanged: %v", out["b"])
	}
}

func TestTrimStringsNilRecord(t *testing.T) {
	tr := TrimStrings()
	out, err := tr.Apply(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Errorf("nil record should produce nil")
	}
}

func TestTrimStringsNoMutation(t *testing.T) {
	tr := TrimStrings()
	in := intake.Record{"a": "  x  ", "b": "y\n"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["a"] != "  x  " {
		t.Errorf("input was mutated: a = %v", in["a"])
	}
	if in["b"] != "y\n" {
		t.Errorf("input was mutated: b = %v", in["b"])
	}
	out["a"] = "mutated"
	if in["a"] != "  x  " {
		t.Errorf("mutating output affected input: a = %v", in["a"])
	}
}

func TestLowerStringsNoMutation(t *testing.T) {
	tr := LowerStrings()
	in := intake.Record{"a": "ABC"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["a"] != "ABC" {
		t.Errorf("input was mutated: a = %v", in["a"])
	}
	out["a"] = "mutated"
	if in["a"] != "ABC" {
		t.Errorf("mutating output affected input: a = %v", in["a"])
	}
}

func TestUpperStringsNoMutation(t *testing.T) {
	tr := UpperStrings()
	in := intake.Record{"a": "abc"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["a"] != "abc" {
		t.Errorf("input was mutated: a = %v", in["a"])
	}
	out["a"] = "mutated"
	if in["a"] != "abc" {
		t.Errorf("mutating output affected input: a = %v", in["a"])
	}
}

func TestTrimStringsSpecificFieldsNoMutation(t *testing.T) {
	tr := TrimStrings("a")
	in := intake.Record{"a": "  x  ", "b": "  y  "}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if in["a"] != "  x  " {
		t.Errorf("input was mutated: a = %v", in["a"])
	}
	if in["b"] != "  y  " {
		t.Errorf("input was mutated: b = %v", in["b"])
	}
	out["a"] = "x"
	if in["a"] != "  x  " {
		t.Errorf("mutating output affected input: a = %v", in["a"])
	}
}
