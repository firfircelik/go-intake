package intake

import (
	"context"
	"errors"
	"io"
	"time"
)

// Pipeline wires a Source, optional transforms and validators, an
// optional quarantine, and a Sink into an executable unit.
//
// Build a Pipeline with the fluent API:
//
//	p := intake.New().
//	    From(source.CSV("in.csv")).
//	    Transform(transform.TrimStrings()).
//	    Validate(validate.Required("id")).
//	    OnInvalid(quarantine.JSONL("bad.jsonl")).
//	    To(sink.JSONL("out.jsonl"))
//
//	if err := p.Run(ctx); err != nil { ... }
//
// A Pipeline is single-use: Run may only be called once per
// Pipeline. Build a new Pipeline for additional runs.
type Pipeline struct {
	src        Source
	snk        Sink
	quarantine Quarantine
	transforms []Transformer
	validators []Validator
	stats      Stats
	run        bool
}

// From attaches the source that produces records for the pipeline.
// It panics if s is nil; a nil source is a programmer error.
func (p *Pipeline) From(s Source) *Pipeline {
	if s == nil {
		panic("intake: nil Source passed to From")
	}
	p.src = s
	return p
}

// Transform appends one or more transformers to the pipeline.
// Transformers run in the order they are added and before any
// validator. It panics if any t is nil.
func (p *Pipeline) Transform(ts ...Transformer) *Pipeline {
	for _, t := range ts {
		if t == nil {
			panic("intake: nil Transformer passed to Transform")
		}
		p.transforms = append(p.transforms, t)
	}
	return p
}

// Validate appends one or more validators to the pipeline.
// Validators run after all transformers and in the order they are
// added. It panics if any v is nil.
func (p *Pipeline) Validate(vs ...Validator) *Pipeline {
	for _, v := range vs {
		if v == nil {
			panic("intake: nil Validator passed to Validate")
		}
		p.validators = append(p.validators, v)
	}
	return p
}

// OnInvalid attaches an optional quarantine sink for records
// rejected by transformers or validators. If not set, invalid
// records are silently skipped (and counted). It panics if q is
// nil.
func (p *Pipeline) OnInvalid(q Quarantine) *Pipeline {
	if q == nil {
		panic("intake: nil Quarantine passed to OnInvalid")
	}
	p.quarantine = q
	return p
}

// To attaches the destination sink for valid records. It panics if
// s is nil.
func (p *Pipeline) To(s Sink) *Pipeline {
	if s == nil {
		panic("intake: nil Sink passed to To")
	}
	p.snk = s
	return p
}

// Stats returns a snapshot of the counters accumulated by the most
// recent Run. Calling Stats before Run returns zero values.
func (p *Pipeline) Stats() Stats {
	return p.stats
}

// Run executes the pipeline, opening the source, sink, and
// quarantine, streaming records through transforms and validators,
// and tearing everything down.
//
// The first call to Run on a Pipeline is the only valid call;
// subsequent calls return a config error. A pipeline with no source
// or no sink is a config error.
//
// Error semantics:
//   - io.EOF from Source.Read ends the run cleanly (nil error).
//   - Any other source read error fails the run with that error.
//   - Transform and validation errors are recorded in Stats.Invalid
//     and, if a quarantine is configured, forwarded to it. They do
//     not fail the run.
//   - Sink and quarantine write errors fail the run with that error.
func (p *Pipeline) Run(ctx context.Context) error {
	if p.run {
		return NewError(ErrKindConfig, "pipeline_already_run", "Pipeline.Run may only be called once per Pipeline")
	}
	p.run = true
	p.stats = Stats{}

	if p.src == nil {
		return NewError(ErrKindConfig, "missing_source", "Pipeline has no Source; call From(...) before Run")
	}
	if p.snk == nil {
		return NewError(ErrKindConfig, "missing_sink", "Pipeline has no Sink; call To(...) before Run")
	}

	// Defer Close calls before Open so that any failed Open still
	// triggers the teardown of the previously-opened components. Each
	// component is required to make Close safe to call even when
	// Open was never invoked.
	defer func() { _ = p.src.Close() }()
	defer func() { _ = p.snk.Close() }()
	if p.quarantine != nil {
		defer func() { _ = p.quarantine.Close() }()
	}

	if err := p.src.Open(ctx); err != nil {
		p.stats.Failed++
		return Wrap(ErrKindSource, "source_open_failed", "failed to open source", err)
	}

	if err := p.snk.Open(ctx); err != nil {
		p.stats.Failed++
		return Wrap(ErrKindSink, "sink_open_failed", "failed to open sink", err)
	}

	if p.quarantine != nil {
		if err := p.quarantine.Open(ctx); err != nil {
			p.stats.Failed++
			return Wrap(ErrKindQuarantine, "quarantine_open_failed", "failed to open quarantine", err)
		}
	}

	for {
		if err := ctx.Err(); err != nil {
			p.stats.Failed++
			return Wrap(ErrKindSource, "context_cancelled", "context cancelled", err)
		}

		rec, err := p.src.Read(ctx)
		if err != nil {
			if isEOF(err) {
				return nil
			}
			p.stats.Failed++
			return Wrap(ErrKindSource, "source_read_failed", "failed to read from source", err)
		}
		p.stats.Read++

		out, terrs := p.applyTransforms(ctx, rec)
		if len(terrs) > 0 {
			p.stats.Invalid++
			if qerr := p.handleInvalid(ctx, rec, StageTransform, terrs); qerr != nil {
				p.stats.Failed++
				return qerr
			}
			continue
		}

		if verrs := p.applyValidators(ctx, out); len(verrs) > 0 {
			p.stats.Invalid++
			if qerr := p.handleInvalid(ctx, out, StageValidation, verrs); qerr != nil {
				p.stats.Failed++
				return qerr
			}
			continue
		}

		if err := p.snk.Write(ctx, out); err != nil {
			p.stats.Failed++
			return Wrap(ErrKindSink, "sink_write_failed", "failed to write record to sink", err)
		}
		p.stats.Written++
	}
}

// applyTransforms runs every transformer in order against r. The
// first transformer that returns an error aborts the chain; the
// returned record is the original (untransformed) record and the
// returned slice contains the single wrapped error. This mirrors
// the previous behaviour: a partially transformed record is not
// useful for downstream debugging, so the pipeline surfaces the
// original.
func (p *Pipeline) applyTransforms(ctx context.Context, r Record) (Record, []error) {
	for _, t := range p.transforms {
		next, err := t.Apply(ctx, r)
		if err != nil {
			return r, []error{Wrap(ErrKindTransform, "transform_failed", "transformer rejected record", err)}
		}
		r = next
	}
	return r, nil
}

// applyValidators runs every validator against r and returns a
// slice containing every validator that rejected the record. The
// pipeline uses the slice to build a single InvalidRecord event,
// so consumers see all failures together rather than only the
// first.
func (p *Pipeline) applyValidators(ctx context.Context, r Record) []error {
	var errs []error
	for _, v := range p.validators {
		if err := v.Validate(ctx, r); err != nil {
			errs = append(errs, Wrap(ErrKindValidation, "validation_failed", "validator rejected record", err))
		}
	}
	return errs
}

func (p *Pipeline) handleInvalid(ctx context.Context, r Record, stage string, errs []error) error {
	if p.quarantine == nil {
		return nil
	}
	info := InvalidRecord{
		Record:    r,
		Errors:    errs,
		Stage:     stage,
		Timestamp: time.Now(),
	}
	if err := p.quarantine.Write(ctx, info); err != nil {
		return Wrap(ErrKindQuarantine, "quarantine_write_failed", "failed to write record to quarantine", err)
	}
	return nil
}

func isEOF(err error) bool {
	return errors.Is(err, io.EOF)
}
