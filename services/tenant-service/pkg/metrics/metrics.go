package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector holds all Prometheus metrics for the tenant service
type MetricsCollector struct {
	// Provisioning metrics
	provisioningTotal    *prometheus.CounterVec
	provisioningDuration prometheus.Histogram
	provisioningErrors   *prometheus.CounterVec

	// Tenant metrics
	tenantsTotal         prometheus.Gauge
	tenantsActiveTotal   prometheus.Gauge
	tenantsByTier        *prometheus.GaugeVec

	// Storage metrics
	storageQuotaTotal    *prometheus.GaugeVec
	storageUsedTotal     *prometheus.GaugeVec
	storageUsagePercent  *prometheus.GaugeVec

	// Request metrics
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec

	// Registry for testing
	registry prometheus.Registerer
}

// NewMetricsCollector creates a new metrics collector for tenant service
// Uses the default global registry
func NewMetricsCollector() *MetricsCollector {
	return NewMetricsCollectorWithRegistry(nil)
}

// NewMetricsCollectorWithRegistry creates a new metrics collector with a custom registry
// If registry is nil, uses promauto (global registry). For tests, pass a custom registry.
func NewMetricsCollectorWithRegistry(registry prometheus.Registerer) *MetricsCollector {
	if registry == nil {
		// Use promauto for production (global registry)
		return newMetricsCollectorPromauto()
	}

	// Use custom registry for tests
	return newMetricsCollectorWithReg(registry)
}

func newMetricsCollectorPromauto() *MetricsCollector {
	return &MetricsCollector{
		// AC7: Provisioning metrics (must complete within 2 minutes)
		provisioningTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tenant_provisioning_total",
				Help: "Total number of tenant provisioning attempts",
			},
			[]string{"status"}, // initiated, completed, failed
		),
		provisioningDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "tenant_provisioning_duration_seconds",
				Help:    "Duration of tenant provisioning in seconds (AC7: must be < 120s)",
				Buckets: []float64{5, 10, 20, 30, 60, 90, 120, 180, 300},
			},
		),
		provisioningErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tenant_provisioning_errors_total",
				Help: "Total number of tenant provisioning errors by type",
			},
			[]string{"error_type"},
		),

		// Tenant metrics
		tenantsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenants_total",
				Help: "Total number of tenants in the system",
			},
		),
		tenantsActiveTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenants_active_total",
				Help: "Total number of active tenants",
			},
		),
		tenantsByTier: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenants_by_tier",
				Help: "Number of tenants by subscription tier",
			},
			[]string{"tier"},
		),

		// AC6: Storage quota metrics
		storageQuotaTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenant_storage_quota_bytes",
				Help: "Storage quota in bytes for each tenant",
			},
			[]string{"tenant_id", "tier"},
		),
		storageUsedTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenant_storage_used_bytes",
				Help: "Storage used in bytes for each tenant",
			},
			[]string{"tenant_id", "tier"},
		),
		storageUsagePercent: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenant_storage_usage_percentage",
				Help: "Storage usage percentage for each tenant",
			},
			[]string{"tenant_id", "tier"},
		),

		// Request metrics
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "tenant_service_request_duration_seconds",
				Help:    "Duration of tenant service requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "status"},
		),
		requestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tenant_service_requests_total",
				Help: "Total number of tenant service requests",
			},
			[]string{"operation", "status"},
		),
	}
}

func newMetricsCollectorWithReg(registry prometheus.Registerer) *MetricsCollector {
	provisioningTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenant_provisioning_total",
			Help: "Total number of tenant provisioning attempts",
		},
		[]string{"status"},
	)
	provisioningDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "tenant_provisioning_duration_seconds",
			Help:    "Duration of tenant provisioning in seconds (AC7: must be < 120s)",
			Buckets: []float64{5, 10, 20, 30, 60, 90, 120, 180, 300},
		},
	)
	provisioningErrors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenant_provisioning_errors_total",
			Help: "Total number of tenant provisioning errors by type",
		},
		[]string{"error_type"},
	)
	tenantsTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tenants_total",
			Help: "Total number of tenants in the system",
		},
	)
	tenantsActiveTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tenants_active_total",
			Help: "Total number of active tenants",
		},
	)
	tenantsByTier := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenants_by_tier",
			Help: "Number of tenants by subscription tier",
		},
		[]string{"tier"},
	)
	storageQuotaTotal := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenant_storage_quota_bytes",
			Help: "Storage quota in bytes for each tenant",
		},
		[]string{"tenant_id", "tier"},
	)
	storageUsedTotal := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenant_storage_used_bytes",
			Help: "Storage used in bytes for each tenant",
		},
		[]string{"tenant_id", "tier"},
	)
	storageUsagePercent := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenant_storage_usage_percentage",
			Help: "Storage usage percentage for each tenant",
		},
		[]string{"tenant_id", "tier"},
	)
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tenant_service_request_duration_seconds",
			Help:    "Duration of tenant service requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "status"},
	)
	requestTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenant_service_requests_total",
			Help: "Total number of tenant service requests",
		},
		[]string{"operation", "status"},
	)

	// Register all metrics
	registry.MustRegister(
		provisioningTotal,
		provisioningDuration,
		provisioningErrors,
		tenantsTotal,
		tenantsActiveTotal,
		tenantsByTier,
		storageQuotaTotal,
		storageUsedTotal,
		storageUsagePercent,
		requestDuration,
		requestTotal,
	)

	return &MetricsCollector{
		provisioningTotal:    provisioningTotal,
		provisioningDuration: provisioningDuration,
		provisioningErrors:   provisioningErrors,
		tenantsTotal:         tenantsTotal,
		tenantsActiveTotal:   tenantsActiveTotal,
		tenantsByTier:        tenantsByTier,
		storageQuotaTotal:    storageQuotaTotal,
		storageUsedTotal:     storageUsedTotal,
		storageUsagePercent:  storageUsagePercent,
		requestDuration:      requestDuration,
		requestTotal:         requestTotal,
		registry:             registry,
	}
}

// IncProvisioningTotal increments the provisioning total counter
func (m *MetricsCollector) IncProvisioningTotal(status string) {
	m.provisioningTotal.WithLabelValues(status).Inc()
}

// RecordProvisioningDuration records the provisioning duration
func (m *MetricsCollector) RecordProvisioningDuration(duration float64) {
	m.provisioningDuration.Observe(duration)
}

// IncProvisioningErrors increments the provisioning errors counter
func (m *MetricsCollector) IncProvisioningErrors(errorType string) {
	m.provisioningErrors.WithLabelValues(errorType).Inc()
}

// SetTenantsTotal sets the total number of tenants
func (m *MetricsCollector) SetTenantsTotal(count float64) {
	m.tenantsTotal.Set(count)
}

// SetTenantsActive sets the number of active tenants
func (m *MetricsCollector) SetTenantsActive(count float64) {
	m.tenantsActiveTotal.Set(count)
}

// SetTenantsByTier sets the number of tenants by tier
func (m *MetricsCollector) SetTenantsByTier(tier string, count float64) {
	m.tenantsByTier.WithLabelValues(tier).Set(count)
}

// SetStorageQuota sets the storage quota for a tenant (AC6)
func (m *MetricsCollector) SetStorageQuota(tenantID, tier string, quota float64) {
	m.storageQuotaTotal.WithLabelValues(tenantID, tier).Set(quota)
}

// SetStorageUsed sets the storage used for a tenant (AC6)
func (m *MetricsCollector) SetStorageUsed(tenantID, tier string, used float64) {
	m.storageUsedTotal.WithLabelValues(tenantID, tier).Set(used)
}

// SetStorageUsagePercent sets the storage usage percentage for a tenant (AC6)
func (m *MetricsCollector) SetStorageUsagePercent(tenantID, tier string, percentage float64) {
	m.storageUsagePercent.WithLabelValues(tenantID, tier).Set(percentage)
}

// RecordRequest records a request duration and increments the counter
func (m *MetricsCollector) RecordRequest(operation, status string, duration float64) {
	m.requestDuration.WithLabelValues(operation, status).Observe(duration)
	m.requestTotal.WithLabelValues(operation, status).Inc()
}
