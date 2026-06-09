// Package window provides windowing and aggregation functions.
package window

import (
	"context"
	"sync"
	"time"

	"github.com/firfircelik/go-intake"
)

// SlidingWindow aggregates records over a sliding time window.
type SlidingWindow struct {
	duration   time.Duration
	interval   time.Duration
	records    []intake.Record
	mu         sync.Mutex
	resultChan chan []intake.Record
	stopChan   chan struct{}
}

// NewSlidingWindow creates a window that emits aggregated records.
func NewSlidingWindow(duration, interval time.Duration) *SlidingWindow {
	return &SlidingWindow{
		duration:   duration,
		interval:   interval,
		resultChan: make(chan []intake.Record, 10),
		stopChan:   make(chan struct{}),
	}
}

// Window returns a channel that receives aggregated records.
func (w *SlidingWindow) Window() <-chan []intake.Record {
	return w.resultChan
}

// AddRecord adds a record to the window.
func (w *SlidingWindow) AddRecord(rec intake.Record) {
	w.mu.Lock()
	w.records = append(w.records, rec)
	w.mu.Unlock()
}

// Start begins the sliding window process.
func (w *SlidingWindow) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopChan:
				return
			case <-ticker.C:
				w.flushWindow()
			}
		}
	}()
}

// Stop stops the window.
func (w *SlidingWindow) Stop() {
	close(w.stopChan)
}

func (w *SlidingWindow) flushWindow() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.records) > 0 {
		w.resultChan <- w.records
		w.records = nil
	}
}

// CountWindow emits a batch after N records.
type CountWindow struct {
	count      int
	records    []intake.Record
	resultChan chan []intake.Record
}

// NewCountWindow creates a window that emits after N records.
func NewCountWindow(count int) *CountWindow {
	return &CountWindow{
		count:      count,
		resultChan: make(chan []intake.Record, 10),
	}
}

// Window returns a channel for batched records.
func (w *CountWindow) Window() <-chan []intake.Record {
	return w.resultChan
}

// AddRecord adds to window and flushes when full.
func (w *CountWindow) AddRecord(rec intake.Record) {
	w.records = append(w.records, rec)
	if len(w.records) >= w.count {
		w.resultChan <- w.records
		w.records = nil
	}
}
