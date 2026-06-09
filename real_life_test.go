package intake_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/firfircelik/go-intake"
	"github.com/firfircelik/go-intake/enrich"
	"github.com/firfircelik/go-intake/quarantine"
	"github.com/firfircelik/go-intake/retry"
	"github.com/firfircelik/go-intake/sink"
	"github.com/firfircelik/go-intake/source"
	"github.com/firfircelik/go-intake/source/streaming"
	"github.com/firfircelik/go-intake/transform"
	"github.com/firfircelik/go-intake/validate"
)

// RealLife_CSVToJSONL mimics real ETL pipeline
func TestRealLife_CSVToJSONL(t *testing.T) {
	// Simulate realistic CSV with headers and data
	csvData := `id,name,email,age,country
1,John Doe,john@example.com,30,US
2,Jane Smith,jane@example.com,25,UK
3,Bob Wilson,bob@example.com,35,CA
,Invalid User,no-email,15,US
,Test User,test@test.com,-5,US
` // Note: Row 4 and 5 have empty id (invalid)
	// Create temp files
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "users.csv")
	jsonlFile := filepath.Join(tmpDir, "users.jsonl")
	quarFile := filepath.Join(tmpDir, "quarantine.jsonl")

	os.WriteFile(csvFile, []byte(csvData), 0644)

	// Real pipeline: CSV → clean → validate → JSONL + quarantine
	p := intake.New().
		From(source.CSV(csvFile)).
		Transform(
			transform.NormalizeHeaders(transform.SnakeCase),
			transform.TrimStrings(),
		).
		Validate(
			validate.Required("id"),
			validate.Required("email"),
		).
		OnInvalid(quarantine.JSONL(quarFile)).
		To(sink.JSONL(jsonlFile))

	if err := p.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	stats := p.Stats()
	t.Logf("Stats: Read=%d, Written=%d, Invalid=%d", stats.Read, stats.Written, stats.Invalid)

	// Verify output
	// Row 4 has empty id (invalid), row 5 has empty id (invalid)
	// Rows 1-3 are valid
	if stats.Written != 3 { // 3 valid records
		t.Errorf("expected 3 valid, got %d", stats.Written)
	}
	if stats.Invalid != 2 { // 2 invalid records
		t.Errorf("expected 2 invalid, got %d", stats.Invalid)
	}
}

// RealLife_LogProcessing mimics log monitoring scenario
func TestRealLife_LogProcessing(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "app.log")

	// Simulate log file with mixed entries
	logContent := `2024-01-01 INFO Starting application
2024-01-01 ERROR Connection failed
2024-01-01 WARN Retrying...
2024-01-01 ERROR Max retries exceeded
2024-01-01 INFO Application stopped
`
	os.WriteFile(logFile, []byte(logContent), 0644)

	// Create streaming source for logs with timeout
	src := streaming.NewFileTailSource(logFile, streaming.WithTailDelay(1*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src.Open(ctx)

	var errors int
	for i := 0; i < 10; i++ { // Limit iterations
		rec, err := src.Read(ctx)
		if err != nil {
			break
		}
		line, _ := rec.Get("line")
		if lineStr, ok := line.(string); ok {
			if len(lineStr) >= 11 {
				level := lineStr[11:]
				if len(level) >= 5 && level[:5] == "ERROR" {
					errors++
				}
			}
		}
	}
	src.Close()

	if errors != 2 {
		t.Errorf("expected 2 errors in log, got %d", errors)
	}
}

// TestRealLife_JSONLValidation validates real JSONL data
func TestRealLife_JSONLValidation(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlIn := filepath.Join(tmpDir, "events.jsonl")
	jsonlOut := filepath.Join(tmpDir, "valid_events.jsonl")

	// Real events data with some invalid entries
	events := []map[string]any{
		{"event_id": "evt1", "user_id": "user1", "timestamp": "2024-01-01T10:00:00Z"},
		{"event_id": "evt2", "user_id": "", "timestamp": "2024-01-01T10:01:00Z"},  // invalid user_id
		{"event_id": "", "user_id": "user3", "timestamp": "2024-01-01T10:02:00Z"}, // invalid event_id
		{"event_id": "evt4", "user_id": "user4", "timestamp": "2024-01-01T10:03:00Z"},
	}

	// Write events
	f, _ := os.Create(jsonlIn)
	for _, e := range events {
		enc, _ := json.Marshal(e)
		f.Write(append(enc, '\n'))
	}
	f.Close()

	// Process with validation
	p := intake.New().
		From(source.JSONL(jsonlIn)).
		Validate(
			validate.Required("event_id"),
			validate.Required("user_id"),
		).
		To(sink.JSONL(jsonlOut))

	p.Run(context.Background())

	stats := p.Stats()
	t.Logf("Valid events: %d, Invalid: %d", stats.Written, stats.Invalid)

	if stats.Written != 2 {
		t.Errorf("expected 2 valid events, got %d", stats.Written)
	}
	if stats.Invalid != 2 {
		t.Errorf("expected 2 invalid events, got %d", stats.Invalid)
	}
}

// TestRealLife_RetryWithBackoff tests retry policy
func TestRealLife_RetryWithBackoff(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")

	os.WriteFile(csvFile, []byte("id\n1\n2\n"), 0644)

	attempts := 0
	policy := retry.NewPolicy(3, retry.WithBackoff(1*time.Millisecond))

	// Simulate transient failure
	err := policy.Do(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return os.ErrNotExist
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success after retries, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestRealLife_Enrichment tests data enrichment
func TestRealLife_Enrichment(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "countries.csv")

	os.WriteFile(csvFile, []byte("code\nUS\nUK\nFR\n"), 0644)

	enricher := enrich.NewStaticMapEnrich("code", "country", map[string]string{
		"US": "United States",
		"UK": "United Kingdom",
	})

	rec := make(intake.Record)
	rec["code"] = "US"

	result, err := enricher.Apply(context.Background(), rec)
	if err != nil {
		t.Fatal(err)
	}

	val, ok := result.Get("country")
	if !ok || val != "United States" {
		t.Errorf("expected United States, got %v", val)
	}
}
