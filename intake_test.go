package intake

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
)

// mockSource is a programmable Source for tests.
type mockSource struct {
	records     []Record
	readErr     error
	openErr     error
	closeErr    error
	openCalls   int
	readCalls   int
	closed      bool
	cancelAfter int
	onRead      func()
}

func (m *mockSource) Open(_ context.Context) error {
	m.openCalls++
	return m.openErr
}

func (m *mockSource) Read(_ context.Context) (Record, error) {
	m.readCalls++
	if m.onRead != nil && m.readCalls == m.cancelAfter {
		m.onRead()
	}
	if m.readErr != nil {
		return nil, m.readErr
	}
	if len(m.records) == 0 {
		return nil, io.EOF
	}
	rec := m.records[0]
	m.records = m.records[1:]
	return rec, nil
}

func (m *mockSource) Close() error {
	m.closed = true
	return m.closeErr
}

// mockSink is a programmable Sink for tests.
type mockSink struct {
	written   []Record
	openErr   error
	writeErr  error
	closeErr  error
	openCalls int
	closed    bool
}

func (m *mockSink) Open(_ context.Context) error {
	m.openCalls++
	return m.openErr
}

func (m *mockSink) Write(_ context.Context, r Record) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	// Defensive copy so caller mutations don't leak in.
	cp := make(Record, len(r))
	for k, v := range r {
		cp[k] = v
	}
	m.written = append(m.written, cp)
	return nil
}

func (m *mockSink) Close() error {
	m.closed = true
	return m.closeErr
}

// mockQuarantine records every InvalidRecord it receives.
type mockQuarantine struct {
	written  []InvalidRecord
	openErr  error
	writeErr error
	closed   bool
}

func (m *mockQuarantine) Open(_ context.Context) error { return m.openErr }

func (m *mockQuarantine) Write(_ context.Context, info InvalidRecord) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	cp := make(Record, len(info.Record))
	for k, v := range info.Record {
		cp[k] = v
	}
	errs := make([]error, len(info.Errors))
	copy(errs, info.Errors)
	m.written = append(m.written, InvalidRecord{
		Record:    cp,
		Errors:    errs,
		Stage:     info.Stage,
		Timestamp: info.Timestamp,
	})
	return nil
}

func (m *mockQuarantine) Close() error {
	m.closed = true
	return nil
}

// transformFunc is a test helper that wraps a function as a
// Transformer.
type transformFunc func(ctx context.Context, r Record) (Record, error)

func (f transformFunc) Apply(ctx context.Context, r Record) (Record, error) {
	return f(ctx, r)
}

// validateFunc is a test helper that wraps a function as a
// Validator.
type validateFunc func(ctx context.Context, r Record) error

func (f validateFunc) Validate(ctx context.Context, r Record) error {
	return f(ctx, r)
}

func TestPipelineHappyPath(t *testing.T) {
	src := &mockSource{records: []Record{
		{"a": "1", "b": "2"},
		{"a": "3", "b": "4"},
	}}
	snk := &mockSink{}

	p := New().From(src).To(snk)
	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if p.Stats().Read != 2 {
		t.Errorf("Read = %d, want 2", p.Stats().Read)
	}
	if p.Stats().Written != 2 {
		t.Errorf("Written = %d, want 2", p.Stats().Written)
	}
	if p.Stats().Invalid != 0 {
		t.Errorf("Invalid = %d, want 0", p.Stats().Invalid)
	}
	if p.Stats().Failed != 0 {
		t.Errorf("Failed = %d, want 0", p.Stats().Failed)
	}
	if len(snk.written) != 2 {
		t.Fatalf("sink received %d records, want 2", len(snk.written))
	}
	if !src.closed {
		t.Error("source was not closed")
	}
	if !snk.closed {
		t.Error("sink was not closed")
	}
}

func TestPipelineEOFCleanExit(t *testing.T) {
	src := &mockSource{records: nil}
	snk := &mockSink{}
	if err := New().From(src).To(snk).Run(context.Background()); err != nil {
		t.Fatalf("Run on empty source returned error: %v", err)
	}
	// Also confirm that a source returning io.EOF cleanly ends the run.
	src2 := &mockSource{records: nil}
	snk2 := &mockSink{}
	if err := New().From(src2).To(snk2).Run(context.Background()); err != nil {
		t.Fatalf("Run with io.EOF should return nil, got %v", err)
	}
}

func TestPipelineTransformErrorGoesToQuarantine(t *testing.T) {
	src := &mockSource{records: []Record{
		{"a": "ok"},
		{"a": "bad"},
		{"a": "ok2"},
	}}
	snk := &mockSink{}
	q := &mockQuarantine{}

	tr := transformFunc(func(_ context.Context, r Record) (Record, error) {
		if r["a"] == "bad" {
			return r, errors.New("nope")
		}
		return r, nil
	})

	p := New().From(src).Transform(tr).OnInvalid(q).To(snk)
	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if p.Stats().Read != 3 {
		t.Errorf("Read = %d, want 3", p.Stats().Read)
	}
	if p.Stats().Written != 2 {
		t.Errorf("Written = %d, want 2", p.Stats().Written)
	}
	if p.Stats().Invalid != 1 {
		t.Errorf("Invalid = %d, want 1", p.Stats().Invalid)
	}
	if len(q.written) != 1 {
		t.Fatalf("quarantine received %d records, want 1", len(q.written))
	}
	if len(q.written[0].Errors) == 0 {
		t.Error("quarantine entry has no errors")
	}
}

func TestPipelineValidationErrorGoesToQuarantine(t *testing.T) {
	src := &mockSource{records: []Record{
		{"a": "ok"},
		{"a": ""},
	}}
	snk := &mockSink{}
	q := &mockQuarantine{}
	v := validateFunc(func(_ context.Context, r Record) error {
		if s, _ := r["a"].(string); s == "" {
			return errors.New("a is empty")
		}
		return nil
	})
	p := New().From(src).Validate(v).OnInvalid(q).To(snk)
	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if p.Stats().Written != 1 {
		t.Errorf("Written = %d, want 1", p.Stats().Written)
	}
	if p.Stats().Invalid != 1 {
		t.Errorf("Invalid = %d, want 1", p.Stats().Invalid)
	}
	if len(q.written) != 1 {
		t.Errorf("quarantine received %d, want 1", len(q.written))
	}
}

func TestPipelineInvalidSkippedWithoutQuarantine(t *testing.T) {
	src := &mockSource{records: []Record{
		{"a": "ok"},
		{"a": "bad"},
	}}
	snk := &mockSink{}
	tr := transformFunc(func(_ context.Context, r Record) (Record, error) {
		if r["a"] == "bad" {
			return r, errors.New("nope")
		}
		return r, nil
	})
	p := New().From(src).Transform(tr).To(snk)
	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if p.Stats().Read != 2 {
		t.Errorf("Read = %d, want 2", p.Stats().Read)
	}
	if p.Stats().Written != 1 {
		t.Errorf("Written = %d, want 1", p.Stats().Written)
	}
	if p.Stats().Invalid != 1 {
		t.Errorf("Invalid = %d, want 1", p.Stats().Invalid)
	}
}

func TestPipelineSourceOpenError(t *testing.T) {
	src := &mockSource{openErr: errors.New("boom")}
	snk := &mockSink{}
	p := New().From(src).To(snk)
	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("Run should have returned an error")
	}
	if !IsKind(err, ErrKindSource) {
		t.Errorf("error kind = %v, want source", err)
	}
	if p.Stats().Failed != 1 {
		t.Errorf("Failed = %d, want 1", p.Stats().Failed)
	}
}

func TestPipelineSinkOpenError(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "x"}}}
	snk := &mockSink{openErr: errors.New("boom")}
	p := New().From(src).To(snk)
	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("Run should have returned an error")
	}
	if !IsKind(err, ErrKindSink) {
		t.Errorf("error kind = %v, want sink", err)
	}
	if p.Stats().Failed != 1 {
		t.Errorf("Failed = %d, want 1", p.Stats().Failed)
	}
}

func TestPipelineSinkWriteError(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "x"}}}
	snk := &mockSink{writeErr: errors.New("boom")}
	p := New().From(src).To(snk)
	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("Run should have returned an error")
	}
	if !IsKind(err, ErrKindSink) {
		t.Errorf("error kind = %v, want sink", err)
	}
	if p.Stats().Failed != 1 {
		t.Errorf("Failed = %d, want 1", p.Stats().Failed)
	}
}

func TestPipelineSourceReadError(t *testing.T) {
	src := &mockSource{readErr: errors.New("read fail")}
	snk := &mockSink{}
	p := New().From(src).To(snk)
	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("Run should have returned an error")
	}
	if !IsKind(err, ErrKindSource) {
		t.Errorf("error kind = %v, want source", err)
	}
}

func TestPipelineQuarantineWriteError(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "bad"}}}
	snk := &mockSink{}
	q := &mockQuarantine{writeErr: errors.New("quar fail")}
	tr := transformFunc(func(_ context.Context, r Record) (Record, error) {
		return r, errors.New("nope")
	})
	p := New().From(src).Transform(tr).OnInvalid(q).To(snk)
	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("Run should have returned an error")
	}
	if !IsKind(err, ErrKindQuarantine) {
		t.Errorf("error kind = %v, want quarantine", err)
	}
}

func TestPipelineDoubleRun(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "x"}}}
	snk := &mockSink{}
	p := New().From(src).To(snk)
	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("second Run should fail")
	}
	if !IsKind(err, ErrKindConfig) {
		t.Errorf("error kind = %v, want config", err)
	}
}

func TestPipelineMissingSource(t *testing.T) {
	p := New().To(&mockSink{})
	err := p.Run(context.Background())
	if !IsKind(err, ErrKindConfig) {
		t.Errorf("error kind = %v, want config", err)
	}
}

func TestPipelineMissingSink(t *testing.T) {
	p := New().From(&mockSource{})
	err := p.Run(context.Background())
	if !IsKind(err, ErrKindConfig) {
		t.Errorf("error kind = %v, want config", err)
	}
}

func TestPipelineContextCancelled(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "x"}, {"a": "y"}}}
	snk := &mockSink{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New().From(src).To(snk)
	err := p.Run(ctx)
	if err == nil {
		t.Fatal("Run should have returned an error for cancelled context")
	}
}

func TestPipelineTransformChain(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "1"}}}
	snk := &mockSink{}
	t1 := transformFunc(func(_ context.Context, r Record) (Record, error) {
		r["a"] = r["a"].(string) + "!"
		return r, nil
	})
	t2 := transformFunc(func(_ context.Context, r Record) (Record, error) {
		r["b"] = "added"
		return r, nil
	})
	p := New().From(src).Transform(t1, t2).To(snk)
	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := snk.written[0]["a"]; got != "1!" {
		t.Errorf("a = %v, want 1!", got)
	}
	if got := snk.written[0]["b"]; got != "added" {
		t.Errorf("b = %v, want added", got)
	}
}

func TestPipelineTransformErrorRecordsCurrentStateInQuarantine(t *testing.T) {
	// A transformer that mutates the record in place and then errors
	// still results in the (now-mutated) record being passed to the
	// quarantine. applyTransforms returns the input record on error,
	// so a transformer that mutates the record will see that
	// mutation reflected in the quarantine payload.
	src := &mockSource{records: []Record{{"a": "x"}}}
	snk := &mockSink{}
	q := &mockQuarantine{}
	tr := transformFunc(func(_ context.Context, r Record) (Record, error) {
		r["mutated"] = true
		return r, errors.New("nope")
	})
	p := New().From(src).Transform(tr).OnInvalid(q).To(snk)
	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := q.written[0].Record["mutated"]; !ok {
		t.Errorf("quarantine record missing 'mutated' key: %v", q.written[0].Record)
	}
}

func TestPipelineContextCancelledMidRun(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "x"}, {"a": "y"}, {"a": "z"}}}
	snk := &mockSink{}
	ctx, cancel := context.WithCancel(context.Background())
	src.cancelAfter = 1
	src.onRead = cancel
	p := New().From(src).To(snk)
	err := p.Run(ctx)
	if err == nil {
		t.Fatal("Run should have returned an error")
	}
	if !IsKind(err, ErrKindSource) {
		t.Errorf("error kind = %v, want source", err)
	}
	if !snk.closed {
		t.Error("sink was not closed on cancellation")
	}
}

func TestPipelineSourceOpenErrorCallsAllClose(t *testing.T) {
	// The pipeline defers Close on every component before any Open
	// call so that a failed Open still tears down the rest. Each
	// component is required to make Close safe to call when Open
	// never succeeded.
	src := &mockSource{openErr: errors.New("boom")}
	snk := &mockSink{}
	if err := New().From(src).To(snk).Run(context.Background()); err == nil {
		t.Fatal("Run should fail")
	}
	if !src.closed {
		t.Error("source.Close should be called even when Open fails (no-op)")
	}
	if !snk.closed {
		t.Error("sink.Close should be called even when source.Open fails (no-op)")
	}
}

func TestPipelineSinkOpenErrorClosesSource(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "x"}}}
	snk := &mockSink{openErr: errors.New("boom")}
	if err := New().From(src).To(snk).Run(context.Background()); err == nil {
		t.Fatal("Run should fail")
	}
	if !src.closed {
		t.Error("source.Close should be called when sink.Open fails (deferred)")
	}
	if !snk.closed {
		t.Error("sink.Close should be called when sink.Open fails (deferred)")
	}
}

func TestPipelineTransformErrorClosesQuarantine(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "x"}}}
	snk := &mockSink{}
	q := &mockQuarantine{}
	tr := transformFunc(func(_ context.Context, r Record) (Record, error) {
		return r, errors.New("nope")
	})
	if err := New().From(src).Transform(tr).OnInvalid(q).To(snk).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !q.closed {
		t.Error("quarantine.Close should be called on successful run that wrote to quarantine")
	}
	if !src.closed {
		t.Error("source.Close should be called")
	}
	if !snk.closed {
		t.Error("sink.Close should be called")
	}
}

func TestPipelineQuarantineOpenErrorClosesSourceAndSink(t *testing.T) {
	src := &mockSource{records: []Record{{"a": "x"}}}
	snk := &mockSink{}
	q := &mockQuarantine{openErr: errors.New("boom")}
	tr := transformFunc(func(_ context.Context, r Record) (Record, error) {
		return r, errors.New("nope")
	})
	err := New().From(src).Transform(tr).OnInvalid(q).To(snk).Run(context.Background())
	if err == nil {
		t.Fatal("Run should fail")
	}
	if !IsKind(err, ErrKindQuarantine) {
		t.Errorf("error kind = %v, want quarantine", err)
	}
	if !src.closed {
		t.Error("source.Close should be called when quarantine.Open fails")
	}
	if !snk.closed {
		t.Error("sink.Close should be called when quarantine.Open fails")
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("inner")
	e := Wrap(ErrKindSink, "x", "y", cause)
	if got := errors.Unwrap(e); got != cause {
		t.Errorf("Unwrap = %v, want %v", got, cause)
	}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should match cause")
	}
}

func TestErrorIsKind(t *testing.T) {
	e := NewError(ErrKindSource, "code", "msg")
	if !IsKind(e, ErrKindSource) {
		t.Error("IsKind should match source")
	}
	if IsKind(e, ErrKindSink) {
		t.Error("IsKind should not match sink")
	}
}

func TestErrorKindCodes(t *testing.T) {
	var wrapped error = Wrap(ErrKindValidation, "validation_failed", "x", errors.New("inner"))
	if !IsKind(wrapped, ErrKindValidation) {
		t.Error("wrapped error should be classified as validation")
	}
	var ie *Error
	if !errors.As(wrapped, &ie) {
		t.Fatal("errors.As should extract *Error")
	}
	if ie.Code != "validation_failed" {
		t.Errorf("Code = %q, want validation_failed", ie.Code)
	}
}

func TestRecordHasGet(t *testing.T) {
	var r Record
	if r.Has("x") {
		t.Error("nil Record should not have keys")
	}
	if v, ok := r.Get("x"); ok || v != nil {
		t.Error("Get on nil Record should return (nil, false)")
	}
	r = Record{"a": 1}
	if !r.Has("a") {
		t.Error("Has should report true for present key")
	}
	if v, ok := r.Get("a"); !ok || v != 1 {
		t.Errorf("Get = (%v, %v), want (1, true)", v, ok)
	}
	// Has/Get must work for zero values too.
	r = Record{"a": ""}
	if !r.Has("a") {
		t.Error("Has should report true for empty-string value")
	}
	if v, ok := r.Get("a"); !ok || v != "" {
		t.Errorf("Get for empty value = (%v, %v), want (\"\", true)", v, ok)
	}
}

func TestErrorErrorString(t *testing.T) {
	e := NewError(ErrKindSource, "x", "y")
	want := "source: x: y"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	wrapped := Wrap(ErrKindSource, "x", "y", fmt.Errorf("inner"))
	want2 := "source: x: y: inner"
	if got := wrapped.Error(); got != want2 {
		t.Errorf("Error() = %q, want %q", got, want2)
	}
}
