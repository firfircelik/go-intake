package source

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONLSourceFromPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.jsonl")
	lines := []string{
		`{"a":1,"b":"x"}`,
		`{"a":2,"b":"y"}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := JSONL(path)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	for i, line := range lines {
		rec, err := src.Read(context.Background())
		if err != nil {
			t.Fatalf("Read %d: %v", i, err)
		}
		var want map[string]any
		if err := json.Unmarshal([]byte(line), &want); err != nil {
			t.Fatal(err)
		}
		for k, v := range want {
			if got := rec[k]; got != v {
				t.Errorf("rec %d field %q = %v, want %v", i, k, got, v)
			}
		}
	}
	_, err := src.Read(context.Background())
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestJSONLSourceFromReader(t *testing.T) {
	r := strings.NewReader("{\"a\":1}\n{\"a\":2}\n")
	src := NewJSONLSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	rec, err := src.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec["a"] != float64(1) {
		t.Errorf("a = %v, want 1", rec["a"])
	}
}

func TestJSONLSourceSkipsBlankLines(t *testing.T) {
	r := strings.NewReader("\n{\"a\":1}\n\n{\"a\":2}\n\n")
	src := NewJSONLSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	for i := 0; i < 2; i++ {
		if _, err := src.Read(context.Background()); err != nil {
			t.Fatalf("Read %d: %v", i, err)
		}
	}
	_, err := src.Read(context.Background())
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestJSONLSourceMalformed(t *testing.T) {
	r := strings.NewReader("not json\n")
	src := NewJSONLSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	_, err := src.Read(context.Background())
	if err == nil {
		t.Error("malformed JSON should fail")
	}
}

func TestJSONLSourceMissingFile(t *testing.T) {
	src := JSONL("/nonexistent/file.jsonl")
	if err := src.Open(context.Background()); err == nil {
		t.Fatal("Open on missing file should fail")
	}
}

func TestJSONLSourceReadBeforeOpen(t *testing.T) {
	src := JSONL("/dev/null")
	_, err := src.Read(context.Background())
	if err == nil {
		t.Fatal("Read before Open should fail")
	}
}

func TestJSONLSourceCloseIdempotent(t *testing.T) {
	r := strings.NewReader("")
	src := NewJSONLSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}
}
