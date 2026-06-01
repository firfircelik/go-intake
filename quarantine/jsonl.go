// Package quarantine contains bundled Quarantine implementations
// for intake pipelines.
//
// The bundled JSONL sink writes one JSON object per line, spreading
// the original record fields and adding three reserved metadata
// keys:
//
//	_errors     (intake.QuarantineKeyErrors): a list of error
//	            objects, one per failure. *validate.ValidationError
//	            entries become {field, rule, message, value}; other
//	            errors become {message}.
//	_stage      (intake.QuarantineKeyStage): the pipeline stage that
//	            rejected the record, either
//	            intake.StageTransform or intake.StageValidation.
//	_timestamp  (intake.QuarantineKeyTimestamp): RFC 3339 timestamp
//	            captured at quarantine time (UTC, with nanoseconds).
//
// These three keys are reserved. If a key is already present on
// the input record, the quarantine sink overwrites it. Custom
// Quarantine implementations that want to remain interchangeable
// with this one should reserve the same keys and document them
// the same way.
package quarantine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/firfircelik/go-intake"
	"github.com/firfircelik/go-intake/validate"
)

// JSONL returns a Quarantine that writes rejected records to a
// JSONL file at path. The file is truncated on open. See the
// package-level documentation for the exact output schema and the
// list of reserved keys.
func JSONL(path string) intake.Quarantine {
	return &jsonlSink{path: path}
}

type jsonlSink struct {
	path string
	file *os.File
	enc  *json.Encoder
}

// Open opens the destination file for writing.
func (q *jsonlSink) Open(ctx context.Context) error {
	if q.file != nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.Create(q.path)
	if err != nil {
		return fmt.Errorf("create quarantine: %w", err)
	}
	q.file = f
	q.enc = json.NewEncoder(f)
	q.enc.SetEscapeHTML(false)
	return nil
}

// Write records a rejected record along with the structured
// reason for rejection. See the package-level documentation for
// the output schema.
func (q *jsonlSink) Write(ctx context.Context, info intake.InvalidRecord) error {
	if q.enc == nil {
		return intake.NewError(intake.ErrKindConfig, "quarantine_not_open", "Write called before Open")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	out := make(intake.Record, len(info.Record)+3)
	for k, v := range info.Record {
		out[k] = v
	}
	out[intake.QuarantineKeyErrors] = encodeErrors(info.Errors)
	if info.Stage == "" {
		out[intake.QuarantineKeyStage] = intake.StageTransform
	} else {
		out[intake.QuarantineKeyStage] = info.Stage
	}
	ts := info.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	out[intake.QuarantineKeyTimestamp] = ts.UTC().Format(time.RFC3339Nano)
	if err := q.enc.Encode(out); err != nil {
		return fmt.Errorf("encode quarantine: %w", err)
	}
	return nil
}

// encodeErrors renders each error as a structured object suitable
// for JSON serialisation. *validate.ValidationError entries become
// {field, rule, message, value}; other errors become {message}.
func encodeErrors(errs []error) []map[string]any {
	out := make([]map[string]any, 0, len(errs))
	for _, e := range errs {
		if e == nil {
			continue
		}
		var ve *validate.ValidationError
		if errors.As(e, &ve) {
			entry := map[string]any{
				"field":   ve.Field,
				"rule":    ve.Rule,
				"message": ve.Message,
			}
			if ve.Value != nil {
				entry["value"] = ve.Value
			}
			out = append(out, entry)
			continue
		}
		out = append(out, map[string]any{"message": e.Error()})
	}
	return out
}

// Close flushes and closes the destination file.
func (q *jsonlSink) Close() error {
	var firstErr error
	if q.file != nil {
		if err := q.file.Sync(); err != nil {
			firstErr = err
		}
		if err := q.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		q.file = nil
	}
	q.enc = nil
	return firstErr
}
