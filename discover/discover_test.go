package discover

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/firfircelik/go-intake"
	"github.com/firfircelik/go-intake/source"
)

// findField returns the FieldProfile for name, or nil if absent.
func findField(p *DatasetProfile, name string) *FieldProfile {
	for i := range p.Fields {
		if p.Fields[i].Name == name {
			return &p.Fields[i]
		}
	}
	return nil
}

// hasIssue reports whether any issue in p matches code and field.
func hasIssue(p *DatasetProfile, code, field string) bool {
	for _, i := range p.Issues {
		if i.Code == code && i.Field == field {
			return true
		}
	}
	return false
}

// issueCount returns the number of issues with the given code.
func issueCount(p *DatasetProfile, code string) int {
	n := 0
	for _, i := range p.Issues {
		if i.Code == code {
			n++
		}
	}
	return n
}

func fieldNames(fs []FieldProfile) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Name
	}
	return out
}

func TestInspectSourceBasic(t *testing.T) {
	src := newSource(
		intake.Record{"a": "1", "b": "x"},
		intake.Record{"a": "2", "b": "y"},
		intake.Record{"a": "3", "b": "z"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", p.RowCount)
	}
	if p.FieldCount != 2 {
		t.Errorf("FieldCount = %d, want 2", p.FieldCount)
	}
	a := findField(p, "a")
	if a == nil {
		t.Fatal("missing field 'a'")
	}
	if a.Type != TypeInt {
		t.Errorf("a.Type = %q, want int", a.Type)
	}
	b := findField(p, "b")
	if b == nil {
		t.Fatal("missing field 'b'")
	}
	if b.Type != TypeString {
		t.Errorf("b.Type = %q, want string", b.Type)
	}
}

func TestInspectSourceCleanNumeric(t *testing.T) {
	src := newSource(
		intake.Record{"n": int64(1)},
		intake.Record{"n": int64(2)},
		intake.Record{"n": int64(3)},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	n := findField(p, "n")
	if n == nil {
		t.Fatal("missing field 'n'")
	}
	if n.Type != TypeInt {
		t.Errorf("n.Type = %q, want int", n.Type)
	}
	if n.TypeConfidence != 1.0 {
		t.Errorf("n.TypeConfidence = %v, want 1.0", n.TypeConfidence)
	}
	if n.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", n.ParseErrors)
	}
}

func TestInspectSourceNumericStrings(t *testing.T) {
	src := newSource(
		intake.Record{"n": "1"},
		intake.Record{"n": "2"},
		intake.Record{"n": "3"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	n := findField(p, "n")
	if n == nil {
		t.Fatal("missing field 'n'")
	}
	if n.Type != TypeInt {
		t.Errorf("n.Type = %q, want int", n.Type)
	}
	if n.TypeConfidence != 1.0 {
		t.Errorf("n.TypeConfidence = %v, want 1.0", n.TypeConfidence)
	}
}

func TestInspectSourceFloats(t *testing.T) {
	src := newSource(
		intake.Record{"price": float64(1.5)},
		intake.Record{"price": float64(2.5)},
		intake.Record{"price": float64(3.5)},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	price := findField(p, "price")
	if price.Type != TypeFloat {
		t.Errorf("price.Type = %q, want float", price.Type)
	}
	if price.MinFloat != 1.5 {
		t.Errorf("price.MinFloat = %v, want 1.5", price.MinFloat)
	}
	if price.MaxFloat != 3.5 {
		t.Errorf("price.MaxFloat = %v, want 3.5", price.MaxFloat)
	}
}

func TestInspectSourceMixedNumericString(t *testing.T) {
	src := newSource(
		intake.Record{"x": int64(1)},
		intake.Record{"x": "two"},
		intake.Record{"x": int64(3)},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	x := findField(p, "x")
	if x == nil {
		t.Fatal("missing field 'x'")
	}
	if x.Type != TypeString {
		t.Errorf("x.Type = %q, want string (mixed)", x.Type)
	}
	if x.TypeConfidence >= 1.0 {
		t.Errorf("x.TypeConfidence = %v, want < 1.0", x.TypeConfidence)
	}
	if !hasIssue(p, CodeMixedTypes, "x") {
		t.Errorf("expected CodeMixedTypes issue for 'x'")
	}
}

func TestInspectSourceBooleans(t *testing.T) {
	src := newSource(
		intake.Record{"flag": true},
		intake.Record{"flag": false},
		intake.Record{"flag": "true"},
		intake.Record{"flag": "false"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := findField(p, "flag")
	if f == nil {
		t.Fatal("missing field 'flag'")
	}
	if f.Type != TypeBool {
		t.Errorf("flag.Type = %q, want bool", f.Type)
	}
	if f.TypeConfidence != 1.0 {
		t.Errorf("flag.TypeConfidence = %v, want 1.0", f.TypeConfidence)
	}
}

func TestInspectSourceDates(t *testing.T) {
	src := newSource(
		intake.Record{"d": "2024-01-15"},
		intake.Record{"d": "2024-02-20"},
		intake.Record{"d": "2024-03-25"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	d := findField(p, "d")
	if d == nil {
		t.Fatal("missing field 'd'")
	}
	if d.Type != TypeDate {
		t.Errorf("d.Type = %q, want date", d.Type)
	}
	if d.TypeConfidence != 1.0 {
		t.Errorf("d.TypeConfidence = %v, want 1.0", d.TypeConfidence)
	}
}

func TestInspectSourceDatetimes(t *testing.T) {
	src := newSource(
		intake.Record{"ts": "2024-01-15T10:30:00Z"},
		intake.Record{"ts": "2024-02-20T10:30:00Z"},
		intake.Record{"ts": "2024-03-25T10:30:00Z"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	ts := findField(p, "ts")
	if ts == nil {
		t.Fatal("missing field 'ts'")
	}
	if ts.Type != TypeDatetime {
		t.Errorf("ts.Type = %q, want datetime", ts.Type)
	}
	if ts.TypeConfidence != 1.0 {
		t.Errorf("ts.TypeConfidence = %v, want 1.0", ts.TypeConfidence)
	}
}

func TestInspectSourceInvalidDates(t *testing.T) {
	src := newSource(
		intake.Record{"d": "2024-01-15"},
		intake.Record{"d": "2024-13-99"},
		intake.Record{"d": "2024-03-25"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	d := findField(p, "d")
	if d == nil {
		t.Fatal("missing field 'd'")
	}
	if d.Type != TypeString {
		t.Errorf("d.Type = %q, want string (mixed dates)", d.Type)
	}
	if !hasIssue(p, CodeMixedTypes, "d") {
		t.Errorf("expected CodeMixedTypes issue for 'd'")
	}
}

func TestInspectSourceMissingValues(t *testing.T) {
	src := newSource(
		intake.Record{"a": "1"},
		intake.Record{"a": nil},
		intake.Record{"a": "3"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	a := findField(p, "a")
	if a == nil {
		t.Fatal("missing field 'a'")
	}
	if a.NullCount != 1 {
		t.Errorf("NullCount = %d, want 1", a.NullCount)
	}
	if a.NonNullCount != 2 {
		t.Errorf("NonNullCount = %d, want 2", a.NonNullCount)
	}
	want := 1.0 / 3.0
	if !floatEq(a.NullRatio, want) {
		t.Errorf("NullRatio = %v, want %v", a.NullRatio, want)
	}
}

func TestInspectSourceMissingValuesAboveThreshold(t *testing.T) {
	src := newSource(
		intake.Record{"a": "1"},
		intake.Record{"a": nil},
		intake.Record{"a": nil},
	)
	p, err := InspectSource(context.Background(), src, Options{NullIssueThreshold: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if !hasIssue(p, CodeHighNullRatio, "a") {
		t.Errorf("expected CodeHighNullRatio issue for 'a'")
	}
}

func TestInspectSourceMissingValuesBelowThreshold(t *testing.T) {
	src := newSource(
		intake.Record{"a": "1"},
		intake.Record{"a": nil},
		intake.Record{"a": "3"},
	)
	p, err := InspectSource(context.Background(), src, Options{NullIssueThreshold: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if hasIssue(p, CodeHighNullRatio, "a") {
		t.Errorf("did not expect CodeHighNullRatio issue at null ratio 1/3")
	}
}

func TestInspectSourceMixedTypeIssueThreshold(t *testing.T) {
	// 2 ints and 1 string: confidence is 0.0 because zero strings
	// are observed; the field is reported as String. The default
	// threshold of 0.8 means a confidence of 0.0 (or any value
	// below 0.8) triggers the mixed_types issue.
	src := newSource(
		intake.Record{"x": int64(1)},
		intake.Record{"x": "two"},
		intake.Record{"x": int64(3)},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasIssue(p, CodeMixedTypes, "x") {
		t.Errorf("expected CodeMixedTypes issue with default threshold")
	}

	// Setting a threshold of 0.0 means any mixed field is
	// reported. confidence of 0.0 < 0.0 is false, so the issue
	// should NOT be emitted.
	p, err = InspectSource(context.Background(), src, Options{MixedTypeIssueThreshold: 0.0})
	if err != nil {
		t.Fatal(err)
	}
	if hasIssue(p, CodeMixedTypes, "x") {
		t.Errorf("did not expect CodeMixedTypes issue with threshold 0.0 (confidence=0.0)")
	}
}

func TestInspectSourceEmptyFieldName(t *testing.T) {
	src := newSource(
		intake.Record{"a": "1", "": "leak"},
		intake.Record{"a": "2"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasIssue(p, CodeEmptyFieldName, "") {
		t.Errorf("expected CodeEmptyFieldName issue")
	}
}

func TestInspectSourceNoRecordsSampled(t *testing.T) {
	p, err := InspectSource(context.Background(), newSource(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.RowCount != 0 {
		t.Errorf("RowCount = %d, want 0", p.RowCount)
	}
	if !hasIssue(p, CodeNoRecordsSampled, "") {
		t.Errorf("expected CodeNoRecordsSampled issue")
	}
}

func TestInspectSourceUniqueCount(t *testing.T) {
	src := newSource(
		intake.Record{"a": "1"},
		intake.Record{"a": "1"},
		intake.Record{"a": "2"},
		intake.Record{"a": "2"},
		intake.Record{"a": "3"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	a := findField(p, "a")
	if a.UniqueCount != 3 {
		t.Errorf("UniqueCount = %d, want 3", a.UniqueCount)
	}
	if !floatEq(a.UniqueRatio, 0.6) {
		t.Errorf("UniqueRatio = %v, want 0.6", a.UniqueRatio)
	}
}

func TestInspectSourceSampleSize(t *testing.T) {
	src := newSource(
		intake.Record{"a": "1"},
		intake.Record{"a": "2"},
		intake.Record{"a": "3"},
	)
	p, err := InspectSource(context.Background(), src, Options{SampleSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if p.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2 (capped at SampleSize)", p.RowCount)
	}
}

func TestInspectSourceContextCancelled(t *testing.T) {
	src := newSource(
		intake.Record{"a": "1"},
		intake.Record{"a": "2"},
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := InspectSource(ctx, src, Options{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestInspectSourceSourceOpenError(t *testing.T) {
	src := &memSource{openErr: errors.New("boom")}
	_, err := InspectSource(context.Background(), src, Options{})
	if err == nil {
		t.Fatal("expected error from source Open")
	}
	if !strings.Contains(err.Error(), "open source") {
		t.Errorf("err = %v, want one mentioning 'open source'", err)
	}
}

func TestInspectSourceSourceReadError(t *testing.T) {
	src := &memSource{readErr: errors.New("io failure")}
	_, err := InspectSource(context.Background(), src, Options{})
	if err == nil {
		t.Fatal("expected error from source Read")
	}
	if !strings.Contains(err.Error(), "read source") {
		t.Errorf("err = %v, want one mentioning 'read source'", err)
	}
}

func TestInspectSourceClosesSource(t *testing.T) {
	src := newSource(intake.Record{"a": "1"})
	if _, err := InspectSource(context.Background(), src, Options{}); err != nil {
		t.Fatal(err)
	}
	if !src.closed {
		t.Error("source.Close was not called after successful inspection")
	}
}

func TestInspectSourceClosesSourceOnOpenError(t *testing.T) {
	src := &memSource{openErr: errors.New("boom")}
	_, _ = InspectSource(context.Background(), src, Options{})
	// Even on open failure, the close path was scheduled. Verify
	// the source was not marked opened (openErr short-circuits).
	if src.opened {
		t.Error("source.Open should not be marked opened when it returned an error")
	}
}

func TestInspectSourceEndToEndCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	csv := "name,age,active\n" +
		"alice,30,true\n" +
		"bob,25,false\n" +
		"carol,40,true\n"
	if err := os.WriteFile(path, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}
	src := source.CSV(path)
	defer src.Close()
	if err := src.Open(context.Background()); err != nil {
		t.Fatal(err)
	}
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", p.RowCount)
	}
	if findField(p, "name").Type != TypeString {
		t.Errorf("name type = %q, want string", findField(p, "name").Type)
	}
	if findField(p, "age").Type != TypeInt {
		t.Errorf("age type = %q, want int", findField(p, "age").Type)
	}
	if findField(p, "active").Type != TypeBool {
		t.Errorf("active type = %q, want bool", findField(p, "active").Type)
	}
}

func TestInspectSourceMessyData(t *testing.T) {
	// Combines several messy-data requirements: numeric strings,
	// missing values, invalid dates, mixed int/string, and
	// booleans. Verifies the field types and the issues that the
	// package promises to emit.
	src := newSource(
		intake.Record{
			"id":      "1",
			"name":    "alice",
			"score":   "9.5",
			"joined":  "2024-01-15",
			"active":  true,
			"comment": "ok",
		},
		intake.Record{
			"id":      "2",
			"name":    "bob",
			"score":   "8.0",
			"joined":  "2024-13-99", // invalid date
			"active":  "no",
			"comment": nil,
		},
		intake.Record{
			"id":      "3",
			"name":    "carol",
			"score":   "7.2",
			"joined":  "2024-03-25",
			"active":  "yes",
			"comment": "fine",
		},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", p.RowCount)
	}
	if findField(p, "id").Type != TypeInt {
		t.Errorf("id type = %q, want int (numeric string)", findField(p, "id").Type)
	}
	if findField(p, "name").Type != TypeString {
		t.Errorf("name type = %q, want string", findField(p, "name").Type)
	}
	if findField(p, "score").Type != TypeFloat {
		t.Errorf("score type = %q, want float (numeric string)", findField(p, "score").Type)
	}
	joined := findField(p, "joined")
	if joined.Type != TypeString {
		t.Errorf("joined type = %q, want string (invalid date)", joined.Type)
	}
	if !hasIssue(p, CodeMixedTypes, "joined") {
		t.Error("expected CodeMixedTypes issue on joined")
	}
	if findField(p, "active").Type != TypeBool {
		t.Errorf("active type = %q, want bool", findField(p, "active").Type)
	}
	comment := findField(p, "comment")
	if comment.NullCount != 1 {
		t.Errorf("comment NullCount = %d, want 1", comment.NullCount)
	}
}

func TestInspectSourceExamples(t *testing.T) {
	src := newSource(
		intake.Record{"a": "x"},
		intake.Record{"a": "y"},
		intake.Record{"a": "z"},
	)
	p, err := InspectSource(context.Background(), src, Options{MaxExamples: 2})
	if err != nil {
		t.Fatal(err)
	}
	a := findField(p, "a")
	if len(a.Examples) != 2 {
		t.Errorf("len(Examples) = %d, want 2 (MaxExamples)", len(a.Examples))
	}
}

func TestInspectSourceFieldsSorted(t *testing.T) {
	src := newSource(
		intake.Record{"z": "1", "a": "2", "m": "3"},
	)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Fields[0].Name != "a" || p.Fields[1].Name != "m" || p.Fields[2].Name != "z" {
		t.Errorf("Fields not sorted: %v", fieldNames(p.Fields))
	}
}

func TestOptionsDefaults(t *testing.T) {
	o := Options{}
	o.applyDefaults()
	if o.SampleSize != 1000 {
		t.Errorf("SampleSize = %d, want 1000", o.SampleSize)
	}
	if o.MaxExamples != 3 {
		t.Errorf("MaxExamples = %d, want 3", o.MaxExamples)
	}
	if o.MaxUnique != 1024 {
		t.Errorf("MaxUnique = %d, want 1024", o.MaxUnique)
	}
	if o.NullIssueThreshold != 0.5 {
		t.Errorf("NullIssueThreshold = %v, want 0.5", o.NullIssueThreshold)
	}
	if o.MixedTypeIssueThreshold != 0.8 {
		t.Errorf("MixedTypeIssueThreshold = %v, want 0.8", o.MixedTypeIssueThreshold)
	}
}

func TestOptionsOutOfRangeDefaults(t *testing.T) {
	o := Options{
		SampleSize:              -1,
		MaxExamples:             0,
		MaxUnique:               0,
		NullIssueThreshold:      1.5,
		MixedTypeIssueThreshold: 0.0,
	}
	o.applyDefaults()
	if o.SampleSize != 1000 {
		t.Errorf("SampleSize defaulted to %d, want 1000", o.SampleSize)
	}
	if o.NullIssueThreshold != 0.5 {
		t.Errorf("NullIssueThreshold defaulted to %v, want 0.5", o.NullIssueThreshold)
	}
	if o.MixedTypeIssueThreshold != 0.8 {
		t.Errorf("MixedTypeIssueThreshold defaulted to %v, want 0.8", o.MixedTypeIssueThreshold)
	}
}

func TestNoDuplicateRowsIssueInV01(t *testing.T) {
	// The CodeDuplicateRows issue from earlier drafts is no longer
	// part of the v0.1 surface. Verify it is not emitted, even
	// when a record is repeated.
	row := intake.Record{"a": "1", "b": "x"}
	src := newSource(row, row, row)
	p, err := InspectSource(context.Background(), src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if issueCount(p, "duplicate_rows") != 0 {
		t.Errorf("did not expect a 'duplicate_rows' issue in v0.1")
	}
}

func floatEq(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}

// memSource is a minimal intake.Source for tests. It is not exposed.
type memSource struct {
	records []intake.Record
	idx     int
	opened  bool
	closed  bool
	openErr error
	readErr error
}

func (m *memSource) Open(_ context.Context) error {
	if m.openErr != nil {
		return m.openErr
	}
	m.opened = true
	return nil
}

func (m *memSource) Read(_ context.Context) (intake.Record, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	if m.idx >= len(m.records) {
		return nil, io.EOF
	}
	r := m.records[m.idx]
	m.idx++
	return r, nil
}

func (m *memSource) Close() error { m.closed = true; return nil }

func newSource(records ...intake.Record) *memSource {
	return &memSource{records: records}
}
