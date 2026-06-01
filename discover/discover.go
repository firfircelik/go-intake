// Package discover inspects a record-oriented Source and produces a
// schema inference and data-quality profile.
//
// discover is best-effort and streaming-friendly. It samples at most
// Options.SampleSize records, so memory use is bounded regardless
// of the size of the upstream dataset. It does not guarantee
// accuracy over the full dataset; for that, integrate discover with
// a real data-quality system, and always validate important fields
// explicitly with validate.* in a pipeline.
//
// The package exposes a small, stable set of types:
//
//   - FieldType: the inferred type of a field (string, int, float,
//     bool, date, datetime, or unknown).
//   - FieldProfile: per-field inference and statistics.
//   - DatasetProfile: the overall profile plus a list of Issues.
//   - Issue: a data-quality problem surfaced during inspection.
//   - Options: knobs for sample size, example caps, and the
//     thresholds above which the high-null-ratio and mixed-types
//     issues are emitted.
//   - InspectSource: the entry point that opens a Source, samples
//     records, and returns a *DatasetProfile.
package discover

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/firfircelik/go-intake"
)

// FieldType is the inferred type of a field in a record dataset.
type FieldType string

const (
	// TypeUnknown indicates the field had no non-null values and
	// therefore no inferable type.
	TypeUnknown FieldType = "unknown"
	// TypeString is the fallback type when no narrower type matches.
	// It is also the result when a field is mixed (with reduced
	// TypeConfidence and a mixed_types issue).
	TypeString FieldType = "string"
	// TypeInt indicates every non-null value parses as an integer
	// (JSON int, or a string that parses via strconv.ParseInt).
	TypeInt FieldType = "int"
	// TypeFloat indicates every non-null value is a number (int or
	// float) and at least one value is fractional.
	TypeFloat FieldType = "float"
	// TypeBool indicates every non-null value is a JSON bool or a
	// recognised boolean string ("true", "false", "yes", "no", "y",
	// "n", "t", "f", case-insensitive). The strings "1" and "0" are
	// intentionally classified as int, not bool.
	TypeBool FieldType = "bool"
	// TypeDate indicates every non-null value parses as a date with
	// no time component.
	TypeDate FieldType = "date"
	// TypeDatetime indicates every non-null value parses as a date
	// with a time component.
	TypeDatetime FieldType = "datetime"
)

// Issue severities.
const (
	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
)

// Issue codes. Use the constants for matching rather than the
// string literals so that consumers can switch on a typed value.
const (
	// CodeNoRecordsSampled is emitted (info) when the source
	// produced no records within Options.SampleSize reads.
	CodeNoRecordsSampled = "no_records_sampled"
	// CodeEmptyFieldName is emitted (error) when at least one
	// record contributed a key that is the empty string.
	CodeEmptyFieldName = "empty_field_name"
	// CodeHighNullRatio is emitted (warning) when a field's
	// NullRatio exceeds Options.NullIssueThreshold.
	CodeHighNullRatio = "high_null_ratio"
	// CodeMixedTypes is emitted (warning) when a field's
	// TypeConfidence is below Options.MixedTypeIssueThreshold.
	CodeMixedTypes = "mixed_types"
	// CodeDuplicateFieldName is reserved for future use. None of
	// the bundled Sources currently expose duplicate header names
	// to discover; the constant is exported so downstream tooling
	// can switch on it.
	CodeDuplicateFieldName = "duplicate_field_name"
)

// Issue is a data-quality problem found by InspectSource. The Code
// is the stable, machine-readable identifier; Message is for humans.
type Issue struct {
	// Field is the name of the field the issue refers to, or the
	// empty string for dataset-level issues such as
	// no_records_sampled.
	Field string
	// Code is the machine-readable identifier (one of the Code*
	// constants).
	Code string
	// Message is a human-readable description of the issue.
	Message string
	// Severity is one of the Severity* constants: info, warning,
	// or error.
	Severity string
}

// FieldProfile is the inferred profile of a single field across the
// sampled records.
type FieldProfile struct {
	// Name is the record key this profile describes.
	Name string
	// Type is the inferred FieldType. See the FieldType constants
	// for the inference rules.
	Type FieldType
	// TypeConfidence is the fraction of non-null values whose
	// primary type matches Type. A value of 1.0 means every
	// non-null value matched cleanly; 0 means the field had no
	// non-null values (TypeUnknown) or no values matched (a mixed
	// field where Type was forced to string).
	TypeConfidence float64
	// NullCount is the number of records where the field was nil,
	// missing, or an empty string.
	NullCount int
	// NonNullCount is the number of records where the field had a
	// value of any kind.
	NonNullCount int
	// NullRatio is NullCount / (NullCount + NonNullCount). It is 0
	// for a field that was always present.
	NullRatio float64
	// UniqueCount is the number of distinct non-null values seen,
	// capped at Options.MaxUnique. Once the cap is reached, this
	// value is approximate.
	UniqueCount int
	// UniqueRatio is UniqueCount / NonNullCount.
	UniqueRatio float64
	// Examples is up to Options.MaxExamples non-null values, in
	// first-seen order, deduplicated.
	Examples []string
	// MinFloat and MaxFloat are populated when Type is int or
	// float. They are zero otherwise.
	MinFloat float64
	MaxFloat float64
	// MinLength and MaxLength are populated when Type is string.
	// They are zero otherwise.
	MinLength int
	MaxLength int
	// ParseErrors is the number of non-null values whose primary
	// type did not match Type. For TypeString and TypeUnknown it
	// is always 0.
	ParseErrors int
}

// DatasetProfile is the inferred profile of a dataset sampled by
// InspectSource.
type DatasetProfile struct {
	// RowCount is the number of records that were read.
	RowCount int
	// FieldCount is the number of distinct keys observed across all
	// records.
	FieldCount int
	// Fields is sorted alphabetically by Name for stable output.
	Fields []FieldProfile
	// Issues lists all data-quality problems detected during the
	// sample. Order is stable but not otherwise meaningful.
	Issues []Issue
}

// Options controls the behaviour of InspectSource. The zero value
// selects the defaults documented on each field.
type Options struct {
	// SampleSize is the maximum number of records to read. The
	// default is 1000.
	SampleSize int
	// MaxExamples is the maximum number of example values to retain
	// per field. The default is 3.
	MaxExamples int
	// MaxUnique is the cap on the per-field distinct-value counter
	// used to compute UniqueCount. The default is 1024. Larger
	// values are more accurate but use more memory.
	MaxUnique int
	// NullIssueThreshold is the NullRatio above which a field
	// triggers a CodeHighNullRatio issue. Must be in (0, 1]. The
	// default is 0.5.
	NullIssueThreshold float64
	// MixedTypeIssueThreshold is the TypeConfidence below which a
	// field triggers a CodeMixedTypes issue. Must be in (0, 1].
	// The default is 0.8.
	MixedTypeIssueThreshold float64
}

func (o *Options) applyDefaults() {
	if o.SampleSize <= 0 {
		o.SampleSize = 1000
	}
	if o.MaxExamples <= 0 {
		o.MaxExamples = 3
	}
	if o.MaxUnique <= 0 {
		o.MaxUnique = 1024
	}
	if o.NullIssueThreshold <= 0 || o.NullIssueThreshold > 1 {
		o.NullIssueThreshold = 0.5
	}
	if o.MixedTypeIssueThreshold <= 0 || o.MixedTypeIssueThreshold > 1 {
		o.MixedTypeIssueThreshold = 0.8
	}
}

// InspectSource opens src, samples up to Options.SampleSize
// records, infers a DatasetProfile, and closes src. Memory use is
// bounded by Options.SampleSize and Options.MaxUnique regardless of
// how many records the source actually contains.
//
// Behaviour:
//
//   - The source is opened and closed exactly once. Close is
//     called even when Open fails.
//   - Context cancellation is respected between reads. A cancelled
//     context aborts the inspection with an error that wraps
//     context.Canceled or context.DeadlineExceeded.
//   - At most Options.SampleSize records are read. The pipeline
//     never reads the whole dataset by default.
//   - Records are never mutated; InspectSource reads them through
//     the Source interface only.
//   - io.EOF ends the inspection cleanly.
//   - Any other read error is returned and the partial profile is
//     discarded.
//
// All observed fields appear in the returned profile, even if they
// only appeared in a single record. Fields with no non-null values
// are present with Type TypeUnknown and TypeConfidence 0.
//
// The returned Issues slice is built in stable order: dataset-level
// issues first, then one entry per field in alphabetical order.
func InspectSource(ctx context.Context, src intake.Source, opts Options) (*DatasetProfile, error) {
	opts.applyDefaults()
	if err := src.Open(ctx); err != nil {
		return nil, fmt.Errorf("discover: open source: %w", err)
	}
	defer func() { _ = src.Close() }()

	accs := make(map[string]*fieldAcc)
	fieldOrder := make([]string, 0)
	var rowCount int

	for i := 0; i < opts.SampleSize; i++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("discover: context cancelled: %w", err)
		}
		rec, err := src.Read(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("discover: read source: %w", err)
		}
		rowCount++
		if rec == nil {
			continue
		}
		for k, v := range rec {
			a, ok := accs[k]
			if !ok {
				a = newFieldAcc(k, opts.MaxExamples, opts.MaxUnique)
				accs[k] = a
				fieldOrder = append(fieldOrder, k)
			}
			a.observe(v)
		}
	}

	sortedFields := append([]string(nil), fieldOrder...)
	sort.Strings(sortedFields)

	profiles := make([]FieldProfile, 0, len(sortedFields))
	for _, name := range sortedFields {
		profiles = append(profiles, accs[name].finalize())
	}

	issues := make([]Issue, 0)

	if _, hasEmpty := accs[""]; hasEmpty {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Code:     CodeEmptyFieldName,
			Field:    "",
			Message:  "at least one record has a field with an empty key",
		})
	}

	for _, p := range profiles {
		issues = append(issues, profileIssues(p, opts.NullIssueThreshold, opts.MixedTypeIssueThreshold)...)
	}

	if rowCount == 0 {
		issues = append(issues, Issue{
			Severity: SeverityInfo,
			Code:     CodeNoRecordsSampled,
			Field:    "",
			Message:  "source produced no records",
		})
	}

	return &DatasetProfile{
		RowCount:   rowCount,
		FieldCount: len(profiles),
		Fields:     profiles,
		Issues:     issues,
	}, nil
}

// profileIssues builds the per-field Issues for a finalized
// FieldProfile. A field is reported as mixed only when there is at
// least one non-null value and the inferred TypeConfidence is below
// MixedTypeIssueThreshold.
func profileIssues(p FieldProfile, nullIssueThreshold, mixedTypeIssueThreshold float64) []Issue {
	var issues []Issue
	if p.NullRatio > nullIssueThreshold {
		issues = append(issues, Issue{
			Severity: SeverityWarning,
			Code:     CodeHighNullRatio,
			Field:    p.Name,
			Message: fmt.Sprintf("field has null ratio %.2f (threshold %.2f)",
				p.NullRatio, nullIssueThreshold),
		})
	}
	if p.NonNullCount > 0 && p.TypeConfidence < mixedTypeIssueThreshold {
		issues = append(issues, Issue{
			Severity: SeverityWarning,
			Code:     CodeMixedTypes,
			Field:    p.Name,
			Message:  fmt.Sprintf("field has mixed types (confidence %.2f)", p.TypeConfidence),
		})
	}
	return issues
}

// hashRecord returns a short, stable hash of a record. The key set
// is sorted so that {a:1, b:2} and {b:2, a:1} hash the same.
func hashRecord(rec intake.Record) string {
	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		fmt.Fprintf(&b, "%v", rec[k])
		b.WriteByte(';')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}
