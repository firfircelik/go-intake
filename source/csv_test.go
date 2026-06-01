package source

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCSVSourceFromPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	if err := os.WriteFile(path, []byte("name,age\nalice,30\nbob,25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := CSV(path)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	want := []map[string]string{
		{"name": "alice", "age": "30"},
		{"name": "bob", "age": "25"},
	}
	for i, expected := range want {
		rec, err := src.Read(context.Background())
		if err != nil {
			t.Fatalf("Read %d: %v", i, err)
		}
		for k, v := range expected {
			if got, _ := rec[k]; got != v {
				t.Errorf("record %d field %q = %v, want %q", i, k, got, v)
			}
		}
	}
	_, err := src.Read(context.Background())
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestCSVSourceFromReader(t *testing.T) {
	r := strings.NewReader("a,b,c\n1,2,3\n4,5,6\n")
	src := NewCSVSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	rec, err := src.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec["a"] != "1" || rec["b"] != "2" || rec["c"] != "3" {
		t.Errorf("unexpected record: %v", rec)
	}
}

func TestCSVSourceEmpty(t *testing.T) {
	r := strings.NewReader("")
	src := NewCSVSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	_, err := src.Read(context.Background())
	if err != io.EOF {
		t.Errorf("expected io.EOF on empty CSV, got %v", err)
	}
}

func TestCSVSourceEmptyRows(t *testing.T) {
	r := strings.NewReader("a,b\n1,2\n\n\n3,4\n")
	src := NewCSVSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	rec1, err := src.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec1["a"] != "1" {
		t.Errorf("first record: %v", rec1)
	}
	rec2, err := src.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec2["a"] != "3" {
		t.Errorf("second record (after empty): %v", rec2)
	}
	_, err = src.Read(context.Background())
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestCSVSourceMissingFile(t *testing.T) {
	src := CSV("/nonexistent/file.csv")
	err := src.Open(context.Background())
	if err == nil {
		t.Fatal("Open on missing file should fail")
	}
}

func TestCSVSourceCustomComma(t *testing.T) {
	r := strings.NewReader("a;b;c\n1;2;3\n")
	src := NewCSVSource(r).Comma(';')
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	rec, err := src.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec["a"] != "1" || rec["b"] != "2" || rec["c"] != "3" {
		t.Errorf("unexpected record: %v", rec)
	}
}

func TestCSVSourceReadBeforeOpen(t *testing.T) {
	src := CSV("/dev/null")
	_, err := src.Read(context.Background())
	if err == nil {
		t.Fatal("Read before Open should fail")
	}
}

func TestCSVSourceHeaders(t *testing.T) {
	r := strings.NewReader("a,b\n1,2\n")
	src := NewCSVSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	h := src.Headers()
	if len(h) != 2 || h[0] != "a" || h[1] != "b" {
		t.Errorf("Headers = %v", h)
	}
}

func TestCSVSourceCloseIdempotent(t *testing.T) {
	r := strings.NewReader("a,b\n1,2\n")
	src := NewCSVSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestCSVSourceWithIOCloser(t *testing.T) {
	r := &closeCounter{Reader: bytes.NewReader([]byte("a,b\n1,2\n"))}
	src := NewCSVSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Read(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}
	if r.closes != 1 {
		t.Errorf("reader closed %d times, want 1", r.closes)
	}
}

func TestCSVSourceContextCancelled(t *testing.T) {
	r := strings.NewReader("a,b\n1,2\n3,4\n")
	src := NewCSVSource(r)
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := src.Read(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

type closeCounter struct {
	io.Reader
	closes int
}

func (c *closeCounter) Close() error {
	c.closes++
	return nil
}
