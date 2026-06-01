package sink

import (
	"context"
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCSVSinkBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	s := CSV(path)
	if err := s.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	recs := []map[string]any{
		{"a": "1", "b": "2"},
		{"a": "3", "b": "4"},
	}
	for _, r := range recs {
		if err := s.Write(context.Background(), toRecord(r)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	// Header: a, b (sorted alphabetically).
	if rows[0][0] != "a" || rows[0][1] != "b" {
		t.Errorf("header = %v", rows[0])
	}
	if rows[1][0] != "1" || rows[1][1] != "2" {
		t.Errorf("row 1 = %v", rows[1])
	}
}

func TestCSVSinkExplicitHeaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	s := CSV(path).WithHeaders("x", "y", "z")
	if err := s.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(context.Background(), toRecord(map[string]any{"y": "2", "x": "1"})); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(context.Background(), toRecord(map[string]any{"z": "3", "x": "4", "y": "5"})); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if rows[1][0] != "1" || rows[1][1] != "2" || rows[1][2] != "" {
		t.Errorf("row 1 = %v, want [1 2 ]", rows[1])
	}
	if rows[2][0] != "4" || rows[2][1] != "5" || rows[2][2] != "3" {
		t.Errorf("row 2 = %v, want [4 5 3]", rows[2])
	}
}

func TestCSVSinkCustomComma(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	s := CSV(path).Comma(';')
	if err := s.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(context.Background(), toRecord(map[string]any{"a": "1", "b": "2"})); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), "1;2") {
		t.Errorf("output should use ; as delimiter: %q", string(data))
	}
}

func TestCSVSinkMissingDir(t *testing.T) {
	s := CSV("/nonexistent/dir/out.csv")
	err := s.Open(context.Background())
	if err == nil {
		t.Fatal("Open to missing dir should fail")
	}
}

func TestCSVSinkWriteBeforeOpen(t *testing.T) {
	dir := t.TempDir()
	s := CSV(filepath.Join(dir, "out.csv"))
	err := s.Write(context.Background(), toRecord(map[string]any{"a": "1"}))
	if err == nil {
		t.Fatal("Write before Open should fail")
	}
}

func TestCSVSinkContextCancelled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	s := CSV(path)
	if err := s.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := s.Write(ctx, toRecord(map[string]any{"a": "1"}))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestCSVSinkTypeStringConversion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	s := CSV(path)
	if err := s.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rec := toRecord(map[string]any{
		"s": "x",
		"b": true,
		"f": 1.5,
		"i": 42,
		"n": nil,
	})
	if err := s.Write(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Expect header line + row line.
	lines := splitLines(string(data))
	if len(lines) < 2 {
		t.Fatalf("expected header + row, got: %q", string(data))
	}
	row := lines[1]
	for _, want := range []string{"x", "true", "1.5", "42"} {
		if !contains(row, want) {
			t.Errorf("row %q missing %q", row, want)
		}
	}
}

func toRecord(m map[string]any) map[string]any { return m }

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
