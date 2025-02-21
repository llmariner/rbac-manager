package monitoring

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/llmariner/rbac-manager/server/internal/cache"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricNamespace = "llmariner"

	metricsNameSinceLastCacheSyncSec = "rbac_server_since_last_cache_sync_sec"
)

// MetricsMonitor holds and updates Prometheus metrics.
type MetricsMonitor struct {
	cstore *cache.Store
	logger logr.Logger

	sinceLastCacheSyncSecGauge prometheus.Gauge
}

// NewMetricsMonitor returns a new MetricsMonitor.
func NewMetricsMonitor(cstore *cache.Store, logger logr.Logger) *MetricsMonitor {
	sinceLastCacheSyncSecGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Name:      metricsNameSinceLastCacheSyncSec,
		},
	)

	m := &MetricsMonitor{
		cstore:                     cstore,
		logger:                     logger.WithName("monitor"),
		sinceLastCacheSyncSecGauge: sinceLastCacheSyncSecGauge,
	}

	prometheus.MustRegister(
		m.sinceLastCacheSyncSecGauge,
	)

	return m
}

// Run updates the metrics periodically.
func (m *MetricsMonitor) Run(ctx context.Context, interval time.Duration) error {
	m.logger.Info("Starting metrics monitor...", "interval", interval)
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			t := time.Since(m.cstore.GetLastSuccessfulSyncTime())
			m.sinceLastCacheSyncSecGauge.Set(float64(t.Seconds()))
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// UnregisterAllCollectors unregisters all connectors.
func (m *MetricsMonitor) UnregisterAllCollectors() {
	prometheus.Unregister(m.sinceLastCacheSyncSecGauge)
}
