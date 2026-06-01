package validate

import (
	"context"
	"errors"
	"testing"

	"github.com/firfircelik/go-intake"
)

func TestMinLen(t *testing.T) {
	v := MinLen("s", 3)
	cases := []struct {
		rec  intake.Record
		want bool
	}{
		{intake.Record{"s": "abc"}, true},
		{intake.Record{"s": "abcd"}, true},
		{intake.Record{"s": "ab"}, false},
		{intake.Record{"s": ""}, false},
		{intake.Record{}, false},
		{intake.Record{"s": 1}, false},
	}
	for i, c := range cases {
		err := v.Validate(context.Background(), c.rec)
		if (err == nil) != c.want {
			t.Errorf("case %d: err = %v, want pass=%v", i, err, c.want)
		}
	}
}

func TestMaxLen(t *testing.T) {
	v := MaxLen("s", 3)
	if err := v.Validate(context.Background(), intake.Record{"s": "abc"}); err != nil {
		t.Error("3 should pass")
	}
	if err := v.Validate(context.Background(), intake.Record{"s": "abcd"}); err == nil {
		t.Error("4 should fail")
	}
}

func TestLen(t *testing.T) {
	v := Len("s", 3)
	if err := v.Validate(context.Background(), intake.Record{"s": "abc"}); err != nil {
		t.Error("3 should pass")
	}
	if err := v.Validate(context.Background(), intake.Record{"s": "ab"}); err == nil {
		t.Error("2 should fail")
	}
}

func TestRuneCount(t *testing.T) {
	v := MinLen("s", 3)
	// 3 runes, 6 bytes: "αβγ"
	if err := v.Validate(context.Background(), intake.Record{"s": "αβγ"}); err != nil {
		t.Errorf("3 runes should pass: %v", err)
	}
}

func TestRegex(t *testing.T) {
	v := Regex("s", `^\d+$`)
	if err := v.Validate(context.Background(), intake.Record{"s": "123"}); err != nil {
		t.Error("digits should pass")
	}
	if err := v.Validate(context.Background(), intake.Record{"s": "abc"}); err == nil {
		t.Error("letters should fail")
	}
}

func TestRegexValue(t *testing.T) {
	v := Regex("s", `^\d+$`)
	err := v.Validate(context.Background(), intake.Record{"s": "abc"})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Value != "abc" {
		t.Errorf("Value = %v, want abc", ve.Value)
	}
}

func TestInvalidPattern(t *testing.T) {
	v := Regex("s", "[invalid")
	if err := v.Validate(context.Background(), intake.Record{"s": "x"}); err == nil {
		t.Error("invalid pattern should always fail")
	}
}

func TestEnum(t *testing.T) {
	v := Enum("s", "a", "b", "c")
	if err := v.Validate(context.Background(), intake.Record{"s": "a"}); err != nil {
		t.Error("a should pass")
	}
	if err := v.Validate(context.Background(), intake.Record{"s": "d"}); err == nil {
		t.Error("d should fail")
	}
}

func TestEnumValue(t *testing.T) {
	v := Enum("s", "a", "b", "c")
	err := v.Validate(context.Background(), intake.Record{"s": "z"})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Value != "z" {
		t.Errorf("Value = %v, want z", ve.Value)
	}
}

func TestEnumSortedMessage(t *testing.T) {
	v := Enum("s", "c", "a", "b")
	err := v.Validate(context.Background(), intake.Record{"s": "z"})
	if err == nil || !contains(err.Error(), "[a b c]") {
		t.Errorf("error should list allowed values sorted, got %v", err)
	}
}

func TestEmail(t *testing.T) {
	v := Email("e")
	if err := v.Validate(context.Background(), intake.Record{"e": "alice@example.com"}); err != nil {
		t.Errorf("valid email: %v", err)
	}
	if err := v.Validate(context.Background(), intake.Record{"e": "not-an-email"}); err == nil {
		t.Error("invalid email should fail")
	}
}

func TestURL(t *testing.T) {
	v := URL("u")
	if err := v.Validate(context.Background(), intake.Record{"u": "https://example.com"}); err != nil {
		t.Errorf("valid URL: %v", err)
	}
	if err := v.Validate(context.Background(), intake.Record{"u": "ftp://example.com"}); err == nil {
		t.Error("ftp should fail")
	}
	if err := v.Validate(context.Background(), intake.Record{"u": "not a url"}); err == nil {
		t.Error("garbage should fail")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
