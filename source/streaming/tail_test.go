package streaming

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileTailSource(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.log")

	// Write initial content
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("line1\nline2\n")
	f.Close()

	src := NewFileTailSource(tmpFile, WithTailDelay(10*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := src.Open(ctx); err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	// Read first two lines
	r1, err := src.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if r1 == nil {
		t.Error("expected record with data")
	}
	line, _ := r1.Get("line")
	if line == nil {
		t.Error("expected line field in record")
	}
}

func TestFileTailSource_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.log")

	os.Create(tmpFile)

	src := NewFileTailSource(tmpFile)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	if err := src.Open(ctx); err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	_, err := src.Read(ctx)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}
