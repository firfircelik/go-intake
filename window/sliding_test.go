// Package window provides windowing and aggregation functions.
package window

import (
	"testing"
	"time"

	"github.com/firfircelik/go-intake"
)

func TestSlidingWindow_AddRecord(t *testing.T) {
	w := NewSlidingWindow(1*time.Second, 100*time.Millisecond)

	rec := make(intake.Record)
	rec["test"] = "value"

	w.AddRecord(rec)

	// Window should be empty initially
	select {
	case batch := <-w.Window():
		if len(batch) != 0 {
			t.Errorf("expected no records immediately")
		}
	default:
		// Expected - no output yet
	}
}

func TestCountWindow_Complete(t *testing.T) {
	w := NewCountWindow(3)

	for i := 0; i < 3; i++ {
		rec := make(intake.Record)
		rec["id"] = i
		w.AddRecord(rec)
	}

	select {
	case batch := <-w.Window():
		if len(batch) != 3 {
			t.Errorf("expected 3 records, got %d", len(batch))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected batch after 3 records")
	}
}
