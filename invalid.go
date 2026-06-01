package intake

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Stage identifies which pipeline stage produced an invalid record.
// Quarantine sinks and downstream tooling use this to distinguish
// records that failed transformation from those that failed
// validation. The pipeline always sets InvalidRecord.Stage to one of
// these constants; custom Quarantine implementations can rely on
// it.
const (
	// StageTransform marks records rejected by a Transformer. The
	// InvalidRecord.Record is the original, untransformed record.
	StageTransform = "transform"
	// StageValidation marks records rejected by one or more
	// Validators. The InvalidRecord.Record is the record after all
	// transformers have run.
	StageValidation = "validation"
)

// Reserved keys added by the bundled JSONL Quarantine sink.
// Custom Quarantine implementations are free to choose their own
// schema, but if they want to remain interchangeable with consumers
// of quarantine.JSONL, they should reserve these keys.
const (
	// QuarantineKeyErrors is the JSONL key that holds the list of
	// error objects for the rejected record.
	QuarantineKeyErrors = "_errors"
	// QuarantineKeyStage is the JSONL key that holds the pipeline
	// stage that produced the rejection (StageTransform or
	// StageValidation).
	QuarantineKeyStage = "_stage"
	// QuarantineKeyTimestamp is the JSONL key that holds the
	// RFC 3339 timestamp captured at quarantine time.
	QuarantineKeyTimestamp = "_timestamp"
)

// InvalidRecord is the structured payload passed to a Quarantine
// sink. It bundles the original record with the errors that caused
// its rejection, the stage that produced those errors, and a
// timestamp captured at the moment the pipeline decided to
// quarantine the record.
//
// Quarantine implementations decide how to serialise this payload.
// The bundled JSONL quarantine, for example, spreads the record
// fields and adds _errors, _stage, and _timestamp (QuarantineKeyErrors,
// QuarantineKeyStage, QuarantineKeyTimestamp).
//
// Field semantics:
//
//   - Record: the record as it appeared at the pipeline stage that
//     produced the error(s). For transform failures it is the
//     untransformed record. For validation failures it is the record
//     after all transforms have run. The pipeline never mutates it
//     before handing it to Quarantine.Write.
//   - Errors: the list of errors that caused rejection. For
//     validation failures this may contain multiple
//     *validate.ValidationError entries (one per failing
//     validator); for transform failures it typically contains a
//     single wrapped error. Each element is a distinct error; the
//     slice is never wrapped in MultiError here.
//   - Stage: StageTransform or StageValidation. Set by the pipeline.
//   - Timestamp: the moment the record was quarantined. The
//     pipeline always populates this with time.Now() at the moment
//     of the call to Quarantine.Write; callers should not populate
//     it themselves.
type InvalidRecord struct {
	Record Record
	Errors []error
	Stage  string
	// Timestamp is set by the pipeline at the moment the record
	// was handed to Quarantine.Write.
	Timestamp time.Time
}

// MultiError collects multiple errors into a single value that
// implements the error interface. The pipeline uses it to combine
// the errors returned by every validator that rejected a record so
// downstream consumers see them together rather than only the first.
//
// MultiError implements Unwrap() []error so errors.As can recover
// each individual error, and Is so errors.Is can search the
// collected slice. The slice returned by Unwrap is a fresh copy of
// the receiver's slice so callers cannot mutate the pipeline's
// internal state.
//
// Example: extract and inspect every error from a pipeline result.
//
//	if err := p.Run(ctx); err != nil {
//	    var me *intake.MultiError
//	    if errors.As(err, &me) {
//	        for _, e := range me.Errors {
//	            // handle e individually
//	        }
//	    }
//	}
type MultiError struct {
	// Errors is the list of collected errors, in the order the
	// pipeline observed them. The slice is never nil when a
	// MultiError is returned by the pipeline; an empty slice is
	// represented as a nil MultiError instead.
	Errors []error
}

// Error implements the error interface. It renders one error per
// line so the message is readable in logs.
func (m *MultiError) Error() string {
	if m == nil || len(m.Errors) == 0 {
		return "no errors"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d errors:", len(m.Errors))
	for _, e := range m.Errors {
		if e == nil {
			continue
		}
		fmt.Fprintf(&b, "\n  - %s", e.Error())
	}
	return b.String()
}

// Unwrap returns the underlying errors so errors.As and errors.Is
// can inspect each one. The slice is a fresh copy of the receiver's
// slice to keep the contract that callers cannot mutate the
// pipeline's internal state.
func (m *MultiError) Unwrap() []error {
	if m == nil {
		return nil
	}
	out := make([]error, len(m.Errors))
	copy(out, m.Errors)
	return out
}

// Is supports errors.Is across a MultiError's collected slice. It
// returns true when target matches any collected error (using
// errors.Is on each element). This lets callers write
// errors.Is(me, mySentinel) without iterating manually.
func (m *MultiError) Is(target error) bool {
	if m == nil || target == nil {
		return false
	}
	for _, e := range m.Errors {
		if e == nil {
			continue
		}
		if errors.Is(e, target) {
			return true
		}
	}
	return false
}
