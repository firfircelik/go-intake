package intake_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/firfircelik/go-intake"
	"github.com/firfircelik/go-intake/retry"
	"github.com/firfircelik/go-intake/sink"
	"github.com/firfircelik/go-intake/source"
	"github.com/firfircelik/go-intake/source/streaming"
	"github.com/firfircelik/go-intake/transform"
	"github.com/firfircelik/go-intake/validate"
)

func TestIntegration_FileTailSource(t *testing.T) {
	// Create temp files
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "input.log")

	// Write test data
	f, err := os.Create(logFile)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("line1\nline2\nline3\n")
	f.Close()

	// Test FileTailSource
	src := streaming.NewFileTailSource(logFile, streaming.WithTailDelay(10*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := src.Open(ctx); err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	count := 0
	for {
		rec, err := src.Read(ctx)
		if err != nil {
			break
		}
		// Verify record has line field
		if _, ok := rec.Get("line"); !ok {
			t.Error("expected line field in record")
		}
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 records, got %d", count)
	}
}

func TestIntegration_EnrichWithPipeline(t *testing.T) {
	// Create CSV source
	csvContent := "country_code\nUS\nUK\nFR\n"

	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "input.csv")
	outputFile := filepath.Join(tmpDir, "output.jsonl")

	os.WriteFile(csvFile, []byte(csvContent), 0644)

	// Test full pipeline
	p := intake.New().
		From(source.CSV(csvFile)).
		Transform(transform.TrimStrings()).
		Validate(validate.Required("country_code")).
		To(sink.JSONL(outputFile))

	if err := p.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	stats := p.Stats()
	if stats.Read != 3 {
		t.Errorf("expected 3 read, got %d", stats.Read)
	}
	if stats.Written != 3 {
		t.Errorf("expected 3 written, got %d", stats.Written)
	}
}

func TestIntegration_RetryPolicy(t *testing.T) {
	attempt := 0
	policy := retry.NewPolicy(3, retry.WithBackoff(10*time.Millisecond))

	err := policy.Do(context.Background(), func() error {
		attempt++
		if attempt < 2 {
			return os.ErrNotExist
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
	if attempt != 2 {
		t.Errorf("expected 2 attempts, got %d", attempt)
	}
}
