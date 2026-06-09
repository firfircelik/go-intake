// Package retry provides retry logic for pipeline components.
package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPolicy_Do_Success(t *testing.T) {
	policy := NewPolicy(3)

	callCount := 0
	err := policy.Do(context.Background(), func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestPolicy_Do_Retry(t *testing.T) {
	policy := NewPolicy(3, WithBackoff(10*time.Millisecond))

	callCount := 0
	err := policy.Do(context.Background(), func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success after retries, got: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestPolicy_Do_MaxRetries(t *testing.T) {
	policy := NewPolicy(2, WithBackoff(10*time.Millisecond))

	callCount := 0
	err := policy.Do(context.Background(), func() error {
		callCount++
		return errors.New("persistent error")
	})

	if err == nil {
		t.Error("expected error after max retries")
	}
	if callCount != 3 { // initial + 2 retries
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}
