package main

import (
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds Prometheus metrics
type Metrics struct {
	processedLogs  prometheus.Counter
	processedBytes prometheus.Counter
}

// Global metrics instance
var metrics *Metrics

// Global atomic counters for thread-safe updates
var (
	totalProcessedLogs  int64
	totalProcessedBytes int64
)

// InitMetrics initializes Prometheus metrics
func InitMetrics() *Metrics {
	m := &Metrics{
		processedLogs: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "syslog_encryptor_processed_logs_total",
			Help: "Total number of log messages processed by the syslog encryptor",
		}),
		processedBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "syslog_encryptor_processed_bytes_total",
			Help: "Total number of bytes processed by the syslog encryptor",
		}),
	}

	// Register metrics with Prometheus
	prometheus.MustRegister(m.processedLogs)
	prometheus.MustRegister(m.processedBytes)

	metrics = m
	return m
}

// RecordProcessedLog increments the processed logs counter
func RecordProcessedLog(messageBytes int) {
	if metrics != nil {
		metrics.processedLogs.Inc()
		metrics.processedBytes.Add(float64(messageBytes))
	}
	
	// Update atomic counters for internal tracking
	atomic.AddInt64(&totalProcessedLogs, 1)
	atomic.AddInt64(&totalProcessedBytes, int64(messageBytes))
}

// GetProcessedLogs returns the current count of processed logs
func GetProcessedLogs() int64 {
	return atomic.LoadInt64(&totalProcessedLogs)
}

// GetProcessedBytes returns the current count of processed bytes
func GetProcessedBytes() int64 {
	return atomic.LoadInt64(&totalProcessedBytes)
}

// StartMetricsServer starts the Prometheus metrics HTTP server
func StartMetricsServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}