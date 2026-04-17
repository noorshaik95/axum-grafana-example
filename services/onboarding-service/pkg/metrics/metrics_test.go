package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.registrations)
	assert.NotNil(t, metrics.logins)
	assert.NotNil(t, metrics.requestDuration)
	assert.NotNil(t, metrics.dbConnections)
}

func TestIncrementRegistrations(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	t.Run("successful registration", func(t *testing.T) {
		metrics.IncrementRegistrations(true)

		count := testutil.ToFloat64(metrics.registrations.WithLabelValues("success"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("failed registration", func(t *testing.T) {
		metrics.IncrementRegistrations(false)

		count := testutil.ToFloat64(metrics.registrations.WithLabelValues("failure"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("multiple registrations", func(t *testing.T) {
		metrics.IncrementRegistrations(true)
		metrics.IncrementRegistrations(true)

		count := testutil.ToFloat64(metrics.registrations.WithLabelValues("success"))
		assert.Equal(t, float64(3), count) // 1 from first test + 2 from this test
	})
}

func TestIncrementLogins(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	t.Run("successful login", func(t *testing.T) {
		metrics.IncrementLogins(true)

		count := testutil.ToFloat64(metrics.logins.WithLabelValues("success"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("failed login", func(t *testing.T) {
		metrics.IncrementLogins(false)

		count := testutil.ToFloat64(metrics.logins.WithLabelValues("failure"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("multiple logins", func(t *testing.T) {
		metrics.IncrementLogins(true)
		metrics.IncrementLogins(false)
		metrics.IncrementLogins(true)

		successCount := testutil.ToFloat64(metrics.logins.WithLabelValues("success"))
		failureCount := testutil.ToFloat64(metrics.logins.WithLabelValues("failure"))

		assert.Equal(t, float64(3), successCount) // 1 from first test + 2 from this test
		assert.Equal(t, float64(2), failureCount) // 1 from second test + 1 from this test
	})
}

func TestObserveRequestDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	t.Run("observe durations", func(t *testing.T) {
		// Test that observing durations doesn't panic
		metrics.ObserveRequestDuration("register", 0.5)
		metrics.ObserveRequestDuration("login", 0.1)
		metrics.ObserveRequestDuration("login", 0.2)
		metrics.ObserveRequestDuration("validate_token", 0.05)

		// Verify the histogram exists and is not nil
		assert.NotNil(t, metrics.requestDuration)

		// Note: Testing histogram values requires more complex assertions
		// In production, these would be verified via Prometheus queries
	})
}

func TestSetDBConnections(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	t.Run("set initial value", func(t *testing.T) {
		metrics.SetDBConnections(10)

		count := testutil.ToFloat64(metrics.dbConnections)
		assert.Equal(t, float64(10), count)
	})

	t.Run("update value", func(t *testing.T) {
		metrics.SetDBConnections(25)

		count := testutil.ToFloat64(metrics.dbConnections)
		assert.Equal(t, float64(25), count)
	})

	t.Run("set to zero", func(t *testing.T) {
		metrics.SetDBConnections(0)

		count := testutil.ToFloat64(metrics.dbConnections)
		assert.Equal(t, float64(0), count)
	})
}

func TestGetters(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	t.Run("GetRegistrations", func(t *testing.T) {
		counter := metrics.GetRegistrations()
		require.NotNil(t, counter)
		assert.Equal(t, metrics.registrations, counter)
	})

	t.Run("GetLogins", func(t *testing.T) {
		counter := metrics.GetLogins()
		require.NotNil(t, counter)
		assert.Equal(t, metrics.logins, counter)
	})

	t.Run("GetRequestDuration", func(t *testing.T) {
		histogram := metrics.GetRequestDuration()
		require.NotNil(t, histogram)
		assert.Equal(t, metrics.requestDuration, histogram)
	})

	t.Run("GetDBConnections", func(t *testing.T) {
		gauge := metrics.GetDBConnections()
		require.NotNil(t, gauge)
		assert.Equal(t, metrics.dbConnections, gauge)
	})
}

func TestMetricsRegistration(t *testing.T) {
	t.Run("metrics are registered with custom registry", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		// Use the metrics to ensure they're initialized
		metrics.IncrementRegistrations(true)
		metrics.IncrementLogins(true)
		metrics.ObserveRequestDuration("test", 0.1)
		metrics.SetDBConnections(5)

		// Verify metrics are registered by collecting them
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		// Should have at least 4 metric families (registrations, logins, duration, connections)
		assert.GreaterOrEqual(t, len(metricFamilies), 4)

		// Verify metric names
		metricNames := make(map[string]bool)
		for _, mf := range metricFamilies {
			metricNames[*mf.Name] = true
		}

		assert.True(t, metricNames["user_registrations_total"])
		assert.True(t, metricNames["user_logins_total"])
		assert.True(t, metricNames["request_duration_seconds"])
		assert.True(t, metricNames["db_connections"])
	})
}
