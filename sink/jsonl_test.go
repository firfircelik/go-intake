package sink

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONLSinkBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jsonl")
	s := JSONL(path)
	if err := s.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, r := range []map[string]any{
		{"a": 1, "b": "x"},
		{"a": 2, "b": "y"},
	} {
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
	sc := bufio.NewScanner(f)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatal(err)
	}
	if got["a"].(float64) != 1 || got["b"].(string) != "x" {
		t.Errorf("line 0 = %v", got)
	}
}

func TestJSONLSinkMissingDir(t *testing.T) {
	s := JSONL("/nonexistent/dir/out.jsonl")
	if err := s.Open(context.Background()); err == nil {
		t.Fatal("Open to missing dir should fail")
	}
}

func TestJSONLSinkWriteBeforeOpen(t *testing.T) {
	dir := t.TempDir()
	s := JSONL(filepath.Join(dir, "out.jsonl"))
	if err := s.Write(context.Background(), toRecord(map[string]any{"a": "1"})); err == nil {
		t.Fatal("Write before Open should fail")
	}
}

func TestJSONLSinkTruncatesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jsonl")
	if err := os.WriteFile(path, []byte("garbage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := JSONL(path)
	if err := s.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(context.Background(), toRecord(map[string]any{"a": 1})); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if contains(string(data), "garbage") {
		t.Errorf("output should not contain old content: %q", string(data))
	}
}

func TestJSONLSinkContextCancelled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jsonl")
	s := JSONL(path)
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
