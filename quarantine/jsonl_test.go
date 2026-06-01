package quarantine

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/firfircelik/go-intake"
	"github.com/firfircelik/go-intake/validate"
)

func TestJSONLQuarantineBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	q := JSONL(path)
	if err := q.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec := intake.Record{"name": "alice", "age": "30"}
	ts := time.Date(2024, 5, 15, 10, 30, 0, 0, time.UTC)
	errs := []error{
		&validate.ValidationError{Field: "name", Rule: "required", Message: "field is required", Value: nil},
	}
	info := intake.InvalidRecord{
		Record:    rec,
		Errors:    errs,
		Stage:     intake.StageValidation,
		Timestamp: ts,
	}
	if err := q.Write(context.Background(), info); err != nil {
		t.Fatal(err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		t.Fatal("no lines")
	}
	var got map[string]any
	if err := json.Unmarshal(sc.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "alice" {
		t.Errorf("name = %v", got["name"])
	}
	if got["_stage"] != "validation" {
		t.Errorf("_stage = %v, want validation", got["_stage"])
	}
	if got["_timestamp"] != "2024-05-15T10:30:00Z" {
		t.Errorf("_timestamp = %v, want 2024-05-15T10:30:00Z", got["_timestamp"])
	}
	errList, ok := got["_errors"].([]any)
	if !ok {
		t.Fatalf("_errors is not a list: %T", got["_errors"])
	}
	if len(errList) != 1 {
		t.Fatalf("_errors has %d entries, want 1", len(errList))
	}
	entry, ok := errList[0].(map[string]any)
	if !ok {
		t.Fatalf("error entry is not an object: %T", errList[0])
	}
	if entry["field"] != "name" {
		t.Errorf("error.field = %v, want name", entry["field"])
	}
	if entry["rule"] != "required" {
		t.Errorf("error.rule = %v, want required", entry["rule"])
	}
	if !strings.Contains(entry["message"].(string), "field is required") {
		t.Errorf("error.message = %v", entry["message"])
	}
}

func TestJSONLQuarantineMultipleErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	q := JSONL(path)
	if err := q.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	errs := []error{
		&validate.ValidationError{Field: "name", Rule: "required", Message: "missing", Value: nil},
		&validate.ValidationError{Field: "age", Rule: "min", Message: "must be >= 0", Value: -1.0},
	}
	info := intake.InvalidRecord{
		Record:    intake.Record{"name": "", "age": -1.0},
		Errors:    errs,
		Stage:     intake.StageValidation,
		Timestamp: time.Now(),
	}
	if err := q.Write(context.Background(), info); err != nil {
		t.Fatal(err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"_errors"`) {
		t.Errorf("output missing _errors: %s", string(data))
	}
	if !strings.Contains(string(data), `"name"`) || !strings.Contains(string(data), `"age"`) {
		t.Errorf("output missing field names: %s", string(data))
	}
	// Decode and check counts.
	var got map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &got); err != nil {
		t.Fatal(err)
	}
	errList := got["_errors"].([]any)
	if len(errList) != 2 {
		t.Errorf("_errors has %d entries, want 2", len(errList))
	}
}

func TestJSONLQuarantineTransformStage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	q := JSONL(path)
	if err := q.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	info := intake.InvalidRecord{
		Record:    intake.Record{"a": "x"},
		Errors:    []error{errPlain("plain error")},
		Stage:     intake.StageTransform,
		Timestamp: time.Now(),
	}
	if err := q.Write(context.Background(), info); err != nil {
		t.Fatal(err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"_stage":"transform"`) {
		t.Errorf("expected _stage=transform, got %s", string(data))
	}
	if !strings.Contains(string(data), `"message":"plain error"`) {
		t.Errorf("plain error not serialised: %s", string(data))
	}
}

func TestJSONLQuarantineNonValidationError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	q := JSONL(path)
	if err := q.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	info := intake.InvalidRecord{
		Record:    intake.Record{"x": "1"},
		Errors:    []error{errPlain("plain error")},
		Stage:     intake.StageTransform,
		Timestamp: time.Now(),
	}
	if err := q.Write(context.Background(), info); err != nil {
		t.Fatal(err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"message":"plain error"`) {
		t.Errorf("plain error not serialised: %s", string(data))
	}
	if strings.Contains(string(data), `"field"`) {
		t.Errorf("plain error should not have field key: %s", string(data))
	}
}

func TestJSONLQuarantineWriteBeforeOpen(t *testing.T) {
	q := JSONL("/tmp/x.jsonl")
	err := q.Write(context.Background(), intake.InvalidRecord{})
	if err == nil {
		t.Fatal("Write before Open should fail")
	}
}

func TestJSONLQuarantineOpenMissingDir(t *testing.T) {
	q := JSONL("/nonexistent/dir/file.jsonl")
	if err := q.Open(context.Background()); err == nil {
		t.Fatal("Open in missing dir should fail")
	}
}

func TestJSONLQuarantineCloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	q := JSONL(path)
	if err := q.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestJSONLQuarantineContextCancelled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	q := JSONL(path)
	if err := q.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := q.Write(ctx, intake.InvalidRecord{Record: intake.Record{"a": "1"}})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestJSONLQuarantineReservesKeys(t *testing.T) {
	// Existing _stage / _timestamp / _errors keys on the record are
	// overwritten by the quarantine.
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	q := JSONL(path)
	if err := q.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	info := intake.InvalidRecord{
		Record: intake.Record{
			"name":       "alice",
			"_stage":     "user_defined",
			"_timestamp": "user_defined",
			"_errors":    "user_defined",
		},
		Errors:    []error{&validate.ValidationError{Field: "name", Rule: "required", Message: "x", Value: nil}},
		Stage:     intake.StageValidation,
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := q.Write(context.Background(), info); err != nil {
		t.Fatal(err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "user_defined") {
		t.Errorf("reserved keys should be overwritten: %s", string(data))
	}
}

type errPlain string

func (e errPlain) Error() string { return string(e) }
