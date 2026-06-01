package discover

import (
	"testing"
)

func TestClassifyValueBools(t *testing.T) {
	cases := []struct {
		in   any
		want FieldType
	}{
		{true, TypeBool},
		{false, TypeBool},
		{"true", TypeBool},
		{"FALSE", TypeBool},
		{"Yes", TypeBool},
		{"n", TypeBool},
		{"T", TypeBool},
		{"F", TypeBool},
	}
	for _, c := range cases {
		if got := classifyValue(c.in); got != c.want {
			t.Errorf("classifyValue(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClassifyValueInts(t *testing.T) {
	cases := []struct {
		in   any
		want FieldType
	}{
		{int(1), TypeInt},
		{int64(-42), TypeInt},
		{uint8(7), TypeInt},
		{"0", TypeInt},
		{"-12345", TypeInt},
		{"  42  ", TypeInt},
	}
	for _, c := range cases {
		if got := classifyValue(c.in); got != c.want {
			t.Errorf("classifyValue(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClassifyValueFloats(t *testing.T) {
	cases := []struct {
		in   any
		want FieldType
	}{
		{float64(1.5), TypeFloat},
		{float32(-2.75), TypeFloat},
		{"3.14", TypeFloat},
		{"-0.5", TypeFloat},
		{"1e10", TypeFloat},
	}
	for _, c := range cases {
		if got := classifyValue(c.in); got != c.want {
			t.Errorf("classifyValue(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClassifyValueDates(t *testing.T) {
	cases := []struct {
		in   any
		want FieldType
	}{
		{"2024-01-15", TypeDate},
		{"15/02/2024", TypeDate}, // 02/01/2006 layout reads the second part as month
		{"02/01/2024", TypeDate}, // could be Jan 2 or Feb 1; either way it is a date
		{"2024/01/15", TypeDate},
	}
	for _, c := range cases {
		if got := classifyValue(c.in); got != c.want {
			t.Errorf("classifyValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClassifyValueDatetimes(t *testing.T) {
	cases := []struct {
		in   any
		want FieldType
	}{
		{"2024-01-15T10:30:00Z", TypeDatetime},
		{"2024-01-15 10:30:00", TypeDatetime},
		{"2024-01-15T10:30:00", TypeDatetime},
	}
	for _, c := range cases {
		if got := classifyValue(c.in); got != c.want {
			t.Errorf("classifyValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClassifyValueStrings(t *testing.T) {
	cases := []struct {
		in   any
		want FieldType
	}{
		{"hello", TypeString},
		{"abc 123", TypeString},
		{"2024-13-45", TypeString}, // invalid date
		{"1.2.3", TypeString},
		{"maybe", TypeString}, // not a recognised bool
	}
	for _, c := range cases {
		if got := classifyValue(c.in); got != c.want {
			t.Errorf("classifyValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClassifyValueNilAndEmpty(t *testing.T) {
	if got := classifyValue(nil); got != "" {
		t.Errorf("classifyValue(nil) = %q, want empty", got)
	}
	if got := classifyValue(""); got != "" {
		t.Errorf("classifyValue(empty) = %q, want empty", got)
	}
	if got := classifyValue("   "); got != "" {
		t.Errorf("classifyValue(whitespace) = %q, want empty", got)
	}
}

func TestInferTypeAllInts(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe(int64(1))
	a.observe(int64(2))
	a.observe(int64(3))
	typ, conf, errs := inferType(a)
	if typ != TypeInt {
		t.Errorf("type = %q, want int", typ)
	}
	if conf != 1.0 {
		t.Errorf("confidence = %v, want 1.0", conf)
	}
	if errs != 0 {
		t.Errorf("parse errors = %d, want 0", errs)
	}
}

func TestInferTypeAllFloats(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe(1.0)
	a.observe(2.5)
	a.observe(3.0)
	typ, conf, errs := inferType(a)
	if typ != TypeFloat {
		t.Errorf("type = %q, want float", typ)
	}
	if conf != 1.0 {
		t.Errorf("confidence = %v, want 1.0", conf)
	}
	if errs != 0 {
		t.Errorf("parse errors = %d, want 0", errs)
	}
}

func TestInferTypeIntAndFloat(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe(int64(1))
	a.observe(2.5)
	a.observe(int64(3))
	typ, conf, _ := inferType(a)
	if typ != TypeFloat {
		t.Errorf("type = %q, want float", typ)
	}
	if conf != 1.0 {
		t.Errorf("confidence = %v, want 1.0", conf)
	}
}

func TestInferTypeNumericStrings(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe("1")
	a.observe("2")
	a.observe("3")
	typ, conf, _ := inferType(a)
	if typ != TypeInt {
		t.Errorf("type = %q, want int", typ)
	}
	if conf != 1.0 {
		t.Errorf("confidence = %v, want 1.0", conf)
	}
}

func TestInferTypeBools(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe(true)
	a.observe("false")
	a.observe("yes")
	typ, conf, _ := inferType(a)
	if typ != TypeBool {
		t.Errorf("type = %q, want bool", typ)
	}
	if conf != 1.0 {
		t.Errorf("confidence = %v, want 1.0", conf)
	}
}

func TestInferTypeDates(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe("2024-01-15")
	a.observe("2024-02-20")
	a.observe("2024-03-25")
	typ, conf, _ := inferType(a)
	if typ != TypeDate {
		t.Errorf("type = %q, want date", typ)
	}
	if conf != 1.0 {
		t.Errorf("confidence = %v, want 1.0", conf)
	}
}

func TestInferTypeMixedIntString(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe(int64(1))
	a.observe("two")
	a.observe(int64(3))
	typ, conf, errs := inferType(a)
	if typ != TypeString {
		t.Errorf("type = %q, want string", typ)
	}
	// Only "two" is a string; the other two values are int and do
	// not match the inferred type.
	if errs != 2 {
		t.Errorf("parse errors = %d, want 2", errs)
	}
	if conf <= 0 || conf >= 1 {
		t.Errorf("confidence = %v, want between 0 and 1", conf)
	}
}

func TestInferTypeInvalidDates(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe("2024-01-15")
	a.observe("2024-13-99")
	a.observe("2024-03-25")
	typ, conf, _ := inferType(a)
	if typ != TypeString {
		t.Errorf("type = %q, want string", typ)
	}
	if conf != 1.0/3.0 {
		t.Errorf("confidence = %v, want %v", conf, 1.0/3.0)
	}
}

func TestInferTypeAllNull(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe(nil)
	a.observe(nil)
	typ, conf, _ := inferType(a)
	if typ != TypeUnknown {
		t.Errorf("type = %q, want unknown", typ)
	}
	if conf != 0 {
		t.Errorf("confidence = %v, want 0", conf)
	}
}

func TestFieldAccNumericRange(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe(int64(1))
	a.observe(int64(5))
	a.observe(int64(3))
	prof := a.finalize()
	if prof.MinFloat != 1 {
		t.Errorf("MinFloat = %v, want 1", prof.MinFloat)
	}
	if prof.MaxFloat != 5 {
		t.Errorf("MaxFloat = %v, want 5", prof.MaxFloat)
	}
}

func TestFieldAccNumericStringRange(t *testing.T) {
	// A field of strings that all parse as floats must still
	// produce a meaningful MinFloat/MaxFloat.
	a := newFieldAcc("x", 3, 16)
	a.observe("1.5")
	a.observe("5.0")
	a.observe("-2.5")
	prof := a.finalize()
	if prof.Type != TypeFloat {
		t.Errorf("Type = %q, want float", prof.Type)
	}
	if prof.MinFloat != -2.5 {
		t.Errorf("MinFloat = %v, want -2.5", prof.MinFloat)
	}
	if prof.MaxFloat != 5.0 {
		t.Errorf("MaxFloat = %v, want 5.0", prof.MaxFloat)
	}
}

func TestFieldAccStringLength(t *testing.T) {
	a := newFieldAcc("x", 3, 16)
	a.observe("a")
	a.observe("abcde")
	a.observe("xyz")
	prof := a.finalize()
	if prof.MinLength != 1 {
		t.Errorf("MinLength = %d, want 1", prof.MinLength)
	}
	if prof.MaxLength != 5 {
		t.Errorf("MaxLength = %d, want 5", prof.MaxLength)
	}
}

func TestFieldAccExamplesCapped(t *testing.T) {
	a := newFieldAcc("x", 2, 16)
	a.observe("a")
	a.observe("b")
	a.observe("c")
	a.observe("d")
	prof := a.finalize()
	if len(prof.Examples) != 2 {
		t.Errorf("len(Examples) = %d, want 2 (capped at MaxExamples)", len(prof.Examples))
	}
}

func TestFieldAccExamplesDeduplicated(t *testing.T) {
	a := newFieldAcc("x", 5, 16)
	a.observe("a")
	a.observe("b")
	a.observe("a") // duplicate
	a.observe("c")
	prof := a.finalize()
	if len(prof.Examples) != 3 {
		t.Errorf("len(Examples) = %d, want 3 (deduplicated)", len(prof.Examples))
	}
}

func TestFieldAccDistinctCapped(t *testing.T) {
	a := newFieldAcc("x", 3, 2)
	a.observe("a")
	a.observe("b")
	a.observe("c")
	a.observe("d")
	prof := a.finalize()
	if prof.UniqueCount != 2 {
		t.Errorf("UniqueCount = %d, want 2 (capped at MaxUnique)", prof.UniqueCount)
	}
}
