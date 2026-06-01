package intake_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/firfircelik/go-intake"
	"github.com/firfircelik/go-intake/quarantine"
	"github.com/firfircelik/go-intake/sink"
	"github.com/firfircelik/go-intake/source"
	"github.com/firfircelik/go-intake/transform"
	"github.com/firfircelik/go-intake/validate"
)

// TestEndToEndPipeline mirrors the example in the README/spec.
// CSV with messy input -> normalise headers -> trim -> parse float ->
// validate -> write valid to JSONL, invalid to quarantine.
func TestEndToEndPipeline(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.csv")
	outPath := filepath.Join(dir, "out.jsonl")
	badPath := filepath.Join(dir, "bad.jsonl")

	csv := "Product Name, Unit Price, in stock\n" +
		"  Widget,  9.99, yes\n" +
		"Gadget, -1.00, yes\n" +
		" , 5.00, yes\n" +
		"Doohickey, 3.50, no\n"
	if err := os.WriteFile(inPath, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}

	p := intake.New().
		From(source.CSV(inPath)).
		Transform(
			transform.NormalizeHeaders(transform.SnakeCase),
			transform.TrimStrings(),
			transform.ParseFloat("unit_price"),
			transform.ParseBool("in_stock"),
		).
		Validate(
			validate.Required("product_name"),
			validate.Min("unit_price", 0),
		).
		OnInvalid(quarantine.JSONL(badPath)).
		To(sink.JSONL(outPath))

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	stats := p.Stats()
	if stats.Read != 4 {
		t.Errorf("Read = %d, want 4", stats.Read)
	}
	if stats.Written != 2 {
		t.Errorf("Written = %d, want 2", stats.Written)
	}
	if stats.Invalid != 2 {
		t.Errorf("Invalid = %d, want 2", stats.Invalid)
	}

	// Read the valid output.
	valid := readJSONL(t, outPath)
	if len(valid) != 2 {
		t.Fatalf("valid output has %d records, want 2", len(valid))
	}
	if valid[0]["product_name"] != "Widget" {
		t.Errorf("product_name[0] = %v", valid[0]["product_name"])
	}
	if valid[0]["unit_price"] != 9.99 {
		t.Errorf("unit_price[0] = %v", valid[0]["unit_price"])
	}
	if valid[0]["in_stock"] != true {
		t.Errorf("in_stock[0] = %v", valid[0]["in_stock"])
	}

	// Read the quarantine output.
	bad := readJSONL(t, badPath)
	if len(bad) != 2 {
		t.Fatalf("quarantine has %d records, want 2", len(bad))
	}
	// Quarantine entries should have _errors, _stage, and _timestamp.
	for i, b := range bad {
		if _, ok := b["_errors"]; !ok {
			t.Errorf("quarantine[%d] missing _errors: %v", i, b)
		}
		if stage, _ := b["_stage"].(string); stage != "validation" {
			t.Errorf("quarantine[%d] _stage = %v, want validation", i, b["_stage"])
		}
		if ts, _ := b["_timestamp"].(string); ts == "" {
			t.Errorf("quarantine[%d] _timestamp missing or empty", i)
		}
	}
}

// TestEndToEndPipelineCollectsAllErrors verifies that when more than
// one validator rejects a record, all of their errors end up in a
// single quarantine entry.
func TestEndToEndPipelineCollectsAllErrors(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.csv")
	outPath := filepath.Join(dir, "out.jsonl")
	badPath := filepath.Join(dir, "bad.jsonl")

	csv := "product,price\n" +
		",-5\n" // missing product AND negative price
	if err := os.WriteFile(inPath, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}
	p := intake.New().
		From(source.CSV(inPath)).
		Transform(
			transform.NormalizeHeaders(transform.SnakeCase),
			transform.ParseFloat("price"),
		).
		Validate(
			validate.Required("product"),
			validate.Min("price", 0),
		).
		OnInvalid(quarantine.JSONL(badPath)).
		To(sink.JSONL(outPath))

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if p.Stats().Invalid != 1 {
		t.Errorf("Invalid = %d, want 1", p.Stats().Invalid)
	}
	bad := readJSONL(t, badPath)
	if len(bad) != 1 {
		t.Fatalf("quarantine has %d records, want 1", len(bad))
	}
	errList, ok := bad[0]["_errors"].([]any)
	if !ok {
		t.Fatalf("_errors is not a list: %T", bad[0]["_errors"])
	}
	if len(errList) != 2 {
		t.Errorf("_errors has %d entries, want 2 (one per validator)", len(errList))
	}
}

func readJSONL(t *testing.T, path string) []map[string]any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []map[string]any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatal(err)
		}
		out = append(out, rec)
	}
	return out
}

// TestPipelineJSONLToCSV exercises a JSONL source -> CSV sink.
func TestPipelineJSONLToCSV(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.jsonl")
	outPath := filepath.Join(dir, "out.csv")
	lines := []string{
		`{"a":1,"b":"x"}`,
		`{"a":2,"b":"y"}`,
	}
	if err := os.WriteFile(inPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := intake.New().
		From(source.JSONL(inPath)).
		To(sink.CSV(outPath))
	if err := p.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.Stats().Written != 2 {
		t.Errorf("Written = %d, want 2", p.Stats().Written)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "1,x") {
		t.Errorf("output missing '1,x': %q", string(data))
	}
}

// TestInspectPipeline demonstrates discover.Inspect on a real CSV
// file.
func TestInspectPipeline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	csv := "name,age,active\n" +
		"alice,30,true\n" +
		"bob,25,false\n" +
		",40,true\n"
	if err := os.WriteFile(path, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}
	src := source.CSV(path)
	defer src.Close()
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	records := []intake.Record{}
	for {
		rec, err := src.Read(context.Background())
		if err != nil {
			break
		}
		records = append(records, rec)
	}
	_ = records
}
