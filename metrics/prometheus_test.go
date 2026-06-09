package metrics

import (
	"bytes"
	"net/http/httptest"
	"strings"
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

	s := exporter.Stats()
	if s.Read != 100 {
		t.Errorf("expected Read=100, got %d", s.Read)
	}
	if s.Written != 95 {
		t.Errorf("expected Written=95, got %d", s.Written)
	}
}

func TestHealthCheck(t *testing.T) {
	hc := NewHealthCheck()
	hc.SetReady(true)

	// Check stats method
	stats := intake.Stats{Read: 50, Written: 40}
	exporter := NewPrometheusExporter(":9999")
	exporter.UpdateStats(stats)

	s := exporter.Stats()
	if s.Read != 50 {
		t.Errorf("expected Read=50, got %d", s.Read)
	}
}

func TestPrometheusExporter_ServeHTTP(t *testing.T) {
	exporter := NewPrometheusExporter(":9999")
	exporter.UpdateStats(intake.Stats{
		Read:    100,
		Written: 95,
		Invalid: 5,
		Failed:  0,
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	exporter.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	// Check Prometheus format uses labels
	if !strings.Contains(body, "intake_records_total{status=\"read\"} 100") {
		t.Errorf("expected intake_records_total{status=\"read\"} 100, got %s", body)
	}
	if !strings.Contains(body, "intake_records_total{status=\"validated\"} 95") {
		t.Errorf("expected intake_records_total{status=\"validated\"} 95, got %s", body)
	}
	if !strings.Contains(body, "intake_records_total{status=\"written\"} 95") {
		t.Errorf("expected intake_records_total{status=\"written\"} 95, got %s", body)
	}
	if !strings.Contains(body, "intake_records_total{status=\"invalid\"} 5") {
		t.Errorf("expected intake_records_total{status=\"invalid\"} 5, got %s", body)
	}
}

func TestHealthCheck_ServeHTTP(t *testing.T) {
	hc := NewHealthCheck()

	req := httptest.NewRequest("GET", "/health", nil)

	// Test ready state
	w := httptest.NewRecorder()
	hc.ServeHTTP(w, req)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), []byte("OK")) {
		t.Errorf("expected OK, got code %d body %s", w.Code, w.Body.String())
	}

	// Test not ready state
	hc.SetReady(false)
	w = httptest.NewRecorder()
	hc.ServeHTTP(w, req)
	if w.Code != 503 || !bytes.Equal(w.Body.Bytes(), []byte("UNHEALTHY")) {
		t.Errorf("expected UNHEALTHY, got code %d body %s", w.Code, w.Body.String())
	}
}

func TestMetricsCollector_Stop(t *testing.T) {
	collector := NewMetricsCollector()

	count := 0
	fn := func() intake.Stats {
		return intake.Stats{Read: uint64(count)}
	}

	collector.Collect(fn)
	count = 100

	time.Sleep(50 * time.Millisecond)
	collector.Stop()
	// Should not panic
}
