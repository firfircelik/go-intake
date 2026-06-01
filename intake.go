// Package intake provides a Go-native, library-first toolkit for
// turning unknown or messy input data into validated, transformed,
// record-oriented output.
//
// The package is organised around five small interfaces:
//
//   - Source: produces a stream of Records.
//   - Sink: consumes a stream of Records.
//   - Transformer: converts or enriches a Record.
//   - Validator: inspects a Record and reports whether it is
//     acceptable.
//   - Quarantine: receives records that a transformer or validator
//     rejected, bundled with the structured reason for rejection.
//
// A Pipeline wires a Source, optional Transformers and Validators,
// an optional Quarantine, and a Sink into a single executable
// unit. See the Pipeline type for the fluent builder.
//
// Errors raised during a run are wrapped in *Error with a stable
// Code and an ErrorKind (source, transform, validation, sink,
// quarantine, config). When multiple validators reject the same
// record, the pipeline collects the errors and exposes them as an
// *MultiError so callers can inspect every reason at once.
package intake

import "context"

// Source produces a stream of Records.
//
// A Source is opened once with Open, then read repeatedly with Read
// until it returns io.EOF, signalling the natural end of the
// stream. After Read returns io.EOF, the Source is considered
// exhausted and the pipeline terminates cleanly. Any other non-nil
// error from Open or Read is a fatal pipeline error that aborts the
// run.
//
// Close must be safe to call even when Open was never called or
// failed. The pipeline defers Close before calling Open, so an
// implementation that does anything non-trivial in Close (such as
// closing a file) must guard against a nil underlying resource.
type Source interface {
	// Open prepares the source for reading. It is called once before
	// the first Read.
	Open(ctx context.Context) error
	// Read returns the next record. When the stream is exhausted it
	// returns (nil, io.EOF). Any other non-nil error aborts the run.
	Read(ctx context.Context) (Record, error)
	// Close releases any resources held by the source. It is safe to
	// call even when Open failed.
	Close() error
}

// Sink consumes a stream of Records.
//
// A Sink is opened once with Open, then written to repeatedly with
// Write. Close must be safe to call even when Open was never called
// or failed; the pipeline defers Close before Open.
type Sink interface {
	// Open prepares the sink for writing. It is called once before
	// the first Write.
	Open(ctx context.Context) error
	// Write appends a single record to the sink.
	Write(ctx context.Context, r Record) error
	// Close flushes and releases any resources held by the sink. It
	// is safe to call even when Open failed.
	Close() error
}

// Transformer converts or enriches a Record.
//
// Contract:
//
//   - Apply must not mutate the input record. It always returns a
//     fresh Record, even on no-op, on error, and when the
//     transformation target field is missing.
//   - A non-nil error signals the record should be rejected. The
//     pipeline routes the original (untransformed) record to the
//     Quarantine (if configured) and continues with the next
//     record.
//
// All transformers provided by the transform package honour this
// contract; custom implementations should do the same so that
// parallel or replay pipelines stay safe.
type Transformer interface {
	// Apply runs the transformation. It returns the transformed
	// record and any error encountered. The returned record is a
	// fresh copy of the input even on no-op or error; the input is
	// never mutated. A non-nil error signals the record should be
	// rejected.
	Apply(ctx context.Context, r Record) (Record, error)
}

// Validator inspects a Record and reports whether it is acceptable.
//
// Contract:
//
//   - Validate must not mutate the input record. It is read-only.
//   - A non-nil error sends the record to the Quarantine (if
//     configured) and skips it. The pipeline calls every configured
//     validator on every record, then collects the errors from all
//     failing validators into a single InvalidRecord event so
//     downstream consumers see every reason at once.
//
// All validators provided by the validate package honour this
// contract; custom implementations should do the same.
type Validator interface {
	// Validate returns nil if the record is acceptable, or a
	// descriptive error otherwise. It must not mutate the record.
	Validate(ctx context.Context, r Record) error
}

// Quarantine is a destination for records rejected by a transformer
// or validator. Unlike a Sink, it receives a structured
// InvalidRecord bundling the original record, the errors that
// caused rejection, the pipeline stage, and a timestamp.
//
// The bundled JSONL Quarantine reserves three keys for its own
// metadata:
//
//	_errors     (QuarantineKeyErrors)
//	_stage      (QuarantineKeyStage)
//	_timestamp  (QuarantineKeyTimestamp)
//
// Custom Quarantine implementations are free to choose their own
// schema, but if they want to remain interchangeable with
// consumers of the bundled JSONL sink, they should reserve the
// same three keys and overwrite any matching keys on the input
// record.
//
// Close must be safe to call even when Open was never called or
// failed.
type Quarantine interface {
	// Open prepares the quarantine for writing.
	Open(ctx context.Context) error
	// Write records a rejected record along with the structured
	// reason for rejection. info.Record is the original record
	// (untransformed for StageTransform, post-transform for
	// StageValidation); info.Errors lists every error that
	// caused rejection (one per failing validator or one wrapped
	// error from a failing transformer); info.Stage is
	// StageTransform or StageValidation; info.Timestamp is
	// populated by the pipeline with time.Now() at the moment
	// of the call.
	Write(ctx context.Context, info InvalidRecord) error
	// Close flushes and releases any resources held by the
	// quarantine. It is safe to call even when Open failed.
	Close() error
}

// Stats summarises a pipeline run. All counters are cumulative and
// monotonic over the life of a single Pipeline.
type Stats struct {
	// Read is the number of records successfully read from the
	// source.
	Read uint64
	// Written is the number of records successfully written to the
	// sink.
	Written uint64
	// Invalid is the number of records that were rejected by a
	// transformer or validator and either sent to quarantine or
	// skipped.
	Invalid uint64
	// Failed is the number of fatal source, sink, or quarantine
	// errors that caused the pipeline to abort.
	Failed uint64
}

// New returns a fresh, empty Pipeline.
func New() *Pipeline { return &Pipeline{} }
