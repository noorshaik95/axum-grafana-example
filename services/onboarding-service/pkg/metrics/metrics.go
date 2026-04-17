package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the user auth service
type Metrics struct {
	// Business metrics
	registrations *prometheus.CounterVec
	logins        *prometheus.CounterVec

	// Performance metrics
	requestDuration *prometheus.HistogramVec

	// System metrics
	dbConnections prometheus.Gauge

	// Rate limiter metrics
	rateLimiterMode prometheus.Gauge
}

// NewMetrics creates a new Metrics instance with all metrics registered
func NewMetrics(registry prometheus.Registerer) *Metrics {
	factory := promauto.With(registry)

	return &Metrics{
		registrations: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_registrations_total",
				Help: "Total number of user registrations",
			},
			[]string{"status"}, // success, failure
		),
		logins: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_logins_total",
				Help: "Total number of user login attempts",
			},
			[]string{"status"}, // success, failure
		),
		requestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_duration_seconds",
				Help:    "Duration of requests in seconds",
				Buckets: prometheus.DefBuckets, // Default buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
			},
			[]string{"operation"}, // register, login, validate_token, etc.
		),
		dbConnections: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections",
				Help: "Current number of database connections",
			},
		),
		rateLimiterMode: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "rate_limiter_mode",
				Help: "Rate limiter mode: 0=redis, 1=memory (fallback)",
			},
		),
	}
}

// IncrementRegistrations increments the registration counter
func (m *Metrics) IncrementRegistrations(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.registrations.WithLabelValues(status).Inc()
}

// IncrementLogins increments the login counter
func (m *Metrics) IncrementLogins(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.logins.WithLabelValues(status).Inc()
}

// ObserveRequestDuration records the duration of a request
func (m *Metrics) ObserveRequestDuration(operation string, durationSeconds float64) {
	m.requestDuration.WithLabelValues(operation).Observe(durationSeconds)
}

// SetDBConnections sets the current number of database connections
func (m *Metrics) SetDBConnections(count int) {
	m.dbConnections.Set(float64(count))
}

// GetRegistrations returns the registrations counter (useful for testing)
func (m *Metrics) GetRegistrations() *prometheus.CounterVec {
	return m.registrations
}

// GetLogins returns the logins counter (useful for testing)
func (m *Metrics) GetLogins() *prometheus.CounterVec {
	return m.logins
}

// GetRequestDuration returns the request duration histogram (useful for testing)
func (m *Metrics) GetRequestDuration() *prometheus.HistogramVec {
	return m.requestDuration
}

// GetDBConnections returns the db connections gauge (useful for testing)
func (m *Metrics) GetDBConnections() prometheus.Gauge {
	return m.dbConnections
}

// SetRateLimiterMode sets the rate limiter mode (0=redis, 1=memory)
func (m *Metrics) SetRateLimiterMode(usingMemory bool) {
	if usingMemory {
		m.rateLimiterMode.Set(1)
	} else {
		m.rateLimiterMode.Set(0)
	}
}

// GetRateLimiterMode returns the rate limiter mode gauge (useful for testing)
func (m *Metrics) GetRateLimiterMode() prometheus.Gauge {
	return m.rateLimiterMode
}
