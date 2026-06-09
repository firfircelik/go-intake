// Package metrics provides observability for pipelines.
package metrics

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/firfircelik/go-intake"
)

// PrometheusExporter exposes pipeline stats via Prometheus format.
// Usage:
//
//	m := metrics.NewPrometheusExporter(":8080")
//	p := intake.New().
//	    From(source...).
//	    Metrics(m).
//	    To(sink...)
type PrometheusExporter struct {
	port   string
	stats  *intake.Stats
	mu     sync.RWMutex
	server *http.Server
}

// NewPrometheusExporter creates a Prometheus metrics exporter.
func NewPrometheusExporter(port string) *PrometheusExporter {
	return &PrometheusExporter{
		port:  port,
		stats: &intake.Stats{},
	}
}

// UpdateStats is called by the pipeline to record metrics.
func (e *PrometheusExporter) UpdateStats(s intake.Stats) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats = &s
}

// Start launches the HTTP server for /metrics endpoint.
func (e *PrometheusExporter) Start() error {
	e.server = &http.Server{
		Addr: e.port,
	}
	http.HandleFunc("/metrics", e.serveMetrics)
	return e.server.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (e *PrometheusExporter) Stop(ctx context.Context) error {
	if e.server != nil {
		return e.server.Shutdown(ctx)
	}
	return nil
}

func (e *PrometheusExporter) serveMetrics(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("# HELP intake_records_read Number of records read\n"))
	w.Write([]byte("# TYPE intake_records_read counter\n"))
	w.Write([]byte(intakeMetric("read", e.stats.Read)))
	w.Write([]byte("# HELP intake_records_written Number of records written\n"))
	w.Write([]byte("# TYPE intake_records_written counter\n"))
	w.Write([]byte(intakeMetric("written", e.stats.Written)))
	w.Write([]byte("# HELP intake_records_invalid Number of invalid records\n"))
	w.Write([]byte("# TYPE intake_records_invalid counter\n"))
	w.Write([]byte(intakeMetric("invalid", e.stats.Invalid)))
	w.Write([]byte("# HELP intake_records_failed Number of failed records\n"))
	w.Write([]byte("# TYPE intake_records_failed counter\n"))
	w.Write([]byte(intakeMetric("failed", e.stats.Failed)))
}

func intakeMetric(name string, value uint64) []byte {
	return []byte("intake_records_" + name + " " + uint64ToStr(value) + "\n")
}

func uint64ToStr(v uint64) string {
	// Simple conversion without strconv import
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// HealthCheck creates an HTTP health endpoint.
type HealthCheck struct {
	port   string
	status int
	mu     sync.RWMutex
}

// NewHealthCheck creates a health check server.
func NewHealthCheck(port string) *HealthCheck {
	return &HealthCheck{port: port, status: http.StatusOK}
}

// SetStatus updates the health status.
func (h *HealthCheck) SetStatus(code int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = code
}

// Start begins serving health checks.
func (h *HealthCheck) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		status := h.status
		h.mu.RUnlock()
		w.WriteHeader(status)
		if status == http.StatusOK {
			w.Write([]byte("OK"))
		} else {
			w.Write([]byte("UNHEALTHY"))
		}
	})
	return http.ListenAndServe(h.port, mux)
}

// Stop gracefully shuts down the health check server.
func (h *HealthCheck) Stop(ctx context.Context) error {
	// Simple implementation - just mark unhealthy
	h.SetStatus(http.StatusServiceUnavailable)
	return nil
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
	c.stop <- struct{}{}
}
