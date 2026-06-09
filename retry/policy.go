// Package retry provides retry logic for pipeline components.
package retry

import (
	"context"
	"time"

	"github.com/firfircelik/go-intake"
)

// Policy defines retry behavior.
type Policy struct {
	maxRetries int
	backoff    time.Duration
}

// PolicyOption configures retry policy.
type PolicyOption func(*Policy)

// WithBackoff sets the delay between retries.
func WithBackoff(d time.Duration) PolicyOption {
	return func(p *Policy) {
		p.backoff = d
	}
}

// NewPolicy creates a retry policy.
func NewPolicy(maxRetries int, opts ...PolicyOption) *Policy {
	p := &Policy{
		maxRetries: maxRetries,
		backoff:    100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Do executes fn with retry logic.
func (p *Policy) Do(ctx context.Context, fn func() error) error {
	var lastErr error
	for i := 0; i <= p.maxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.backoff):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err

		// Exponential backoff
		p.backoff *= 2
	}
	return lastErr
}

// RetrySink wraps a sink with retry logic.
// This is a helper type for custom sink implementations.
type RetrySink struct {
	inner  intake.Sink
	policy *Policy
}

// NewRetrySink wraps a sink with retry policy.
func NewRetrySink(inner intake.Sink, policy *Policy) *RetrySink {
	return &RetrySink{
		inner:  inner,
		policy: policy,
	}
}

// Write implements Sink interface with retry.
func (s *RetrySink) Write(ctx context.Context, r intake.Record) error {
	return s.policy.Do(ctx, func() error {
		return s.inner.Write(ctx, r)
	})
}

// Open delegates to inner sink.
func (s *RetrySink) Open(ctx context.Context) error {
	return s.inner.Open(ctx)
}

// Close delegates to inner sink.
func (s *RetrySink) Close() error {
	return s.inner.Close()
}
