package metrics

import (
	"testing"
	"time"

	"github.com/firfircelik/go-intake"
)

func TestPrometheusExporter_UpdateStats(t *testing.T) {
	exporter := NewPrometheusExporter(":9999")

	stats := intake.Stats{
		Read:    100,
		Written: 95,
		Invalid: 5,
		Failed:  0,
	}

	exporter.UpdateStats(stats)

	exporter.mu.RLock()
	if exporter.stats.Read != 100 {
		t.Errorf("expected Read=100, got %d", exporter.stats.Read)
	}
	if exporter.stats.Written != 95 {
		t.Errorf("expected Written=95, got %d", exporter.stats.Written)
	}
	exporter.mu.RUnlock()
}

func TestHealthCheck(t *testing.T) {
	// Skip in parallel test environment
	t.Skip("requires network, tested manually")
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	count := 0
	fn := func() intake.Stats {
		return intake.Stats{Read: uint64(count)}
	}

	collector.Collect(fn)
	count = 100

	time.Sleep(100 * time.Millisecond)
	// Collector runs in background, metrics would be updated
}

func TestUint64ToStr(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{999999, "999999"},
	}

	for _, tt := range tests {
		result := uint64ToStr(tt.input)
		if result != tt.expected {
			t.Errorf("uint64ToStr(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}
