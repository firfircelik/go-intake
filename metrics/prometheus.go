// Package metrics provides observability for pipelines.
package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/firfircelik/go-intake"
)

// PrometheusExporter exposes pipeline stats via Prometheus format.
// Use as an http.Handler with your own HTTP server.
type PrometheusExporter struct {
	stats *intake.Stats
	mu    sync.RWMutex
}

// NewPrometheusExporter creates a Prometheus metrics exporter.
// The address parameter is reserved for future use.
func NewPrometheusExporter(_ string) *PrometheusExporter {
	return &PrometheusExporter{
		stats: &intake.Stats{},
	}
}

// UpdateStats is called by the pipeline to record metrics.
func (e *PrometheusExporter) UpdateStats(s intake.Stats) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats = &s
}

// ServeHTTP implements http.Handler for /metrics endpoint.
func (e *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	stats := *e.stats
	e.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# HELP intake_records_total Total number of records processed\n")
	fmt.Fprintf(w, "# TYPE intake_records_total counter\n")
	fmt.Fprintf(w, "intake_records_total{status=\"read\"} %d\n", stats.Read)
	fmt.Fprintf(w, "intake_records_total{status=\"validated\"} %d\n", stats.Read-stats.Invalid)
	fmt.Fprintf(w, "intake_records_total{status=\"written\"} %d\n", stats.Written)
	fmt.Fprintf(w, "intake_records_total{status=\"invalid\"} %d\n", stats.Invalid)
	fmt.Fprintf(w, "intake_records_total{status=\"failed\"} %d\n", stats.Failed)
}

// Stats returns a copy of current stats.
func (e *PrometheusExporter) Stats() intake.Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return *e.stats
}

// HealthCheck creates an HTTP health endpoint.
type HealthCheck struct {
	mu    sync.RWMutex
	ready bool
}

// NewHealthCheck creates a health check server.
func NewHealthCheck() *HealthCheck {
	return &HealthCheck{ready: true}
}

// SetReady updates the ready status.
func (h *HealthCheck) SetReady(ready bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ready = ready
}

// ServeHTTP implements http.Handler for /health endpoint.
func (h *HealthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	ready := h.ready
	h.mu.RUnlock()

	if ready {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "UNHEALTHY")
	}
}

// MetricsCollector wraps multiple metric exporters.
type MetricsCollector struct {
	exporters []MetricsExporter
	ticker    *time.Ticker
	stop      chan struct{}
}

// MetricsExporter interface for metric implementations.
type MetricsExporter interface {
	UpdateStats(s intake.Stats)
}

// NewMetricsCollector creates a collector that periodically exports metrics.
func NewMetricsCollector(exporters ...MetricsExporter) *MetricsCollector {
	return &MetricsCollector{
		exporters: exporters,
		ticker:    time.NewTicker(5 * time.Second),
		stop:      make(chan struct{}),
	}
}

// Collect starts the periodic collection loop.
func (c *MetricsCollector) Collect(fn func() intake.Stats) {
	go func() {
		for {
			select {
			case <-c.ticker.C:
				stats := fn()
				for _, exp := range c.exporters {
					exp.UpdateStats(stats)
				}
			case <-c.stop:
				return
			}
		}
	}()
}

// Stop stops the collector.
func (c *MetricsCollector) Stop() {
	close(c.stop)
}
