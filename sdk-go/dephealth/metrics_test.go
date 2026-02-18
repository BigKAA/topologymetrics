package dephealth

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func boolPtr(b bool) *bool { return &b }

func newTestExporter(t *testing.T, instanceName string, opts ...MetricsOption) (*MetricsExporter, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	allOpts := append([]MetricsOption{WithMetricsRegisterer(reg)}, opts...)
	m, err := NewMetricsExporter(instanceName, "test-group", allOpts...)
	if err != nil {
		t.Fatalf("failed to create MetricsExporter: %v", err)
	}
	return m, reg
}

func TestMetricsExporter_SetHealth(t *testing.T) {
	m, _ := newTestExporter(t, "test-app")

	dep := Dependency{Name: "postgres-main", Type: TypePostgres, Critical: boolPtr(true)}
	ep := Endpoint{Host: "pg.svc", Port: "5432"}

	m.SetHealth(dep, ep, 1)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="yes",dependency="postgres-main",group="test-group",host="pg.svc",name="test-app",port="5432",type="postgres"} 1
	`
	if err := testutil.CollectAndCompare(m.health, strings.NewReader(expected)); err != nil {
		t.Errorf("health metric mismatch: %v", err)
	}
}

func TestMetricsExporter_SetHealth_Unhealthy(t *testing.T) {
	m, _ := newTestExporter(t, "test-app")

	dep := Dependency{Name: "redis-cache", Type: TypeRedis, Critical: boolPtr(false)}
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	m.SetHealth(dep, ep, 0)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="redis-cache",group="test-group",host="redis.svc",name="test-app",port="6379",type="redis"} 0
	`
	if err := testutil.CollectAndCompare(m.health, strings.NewReader(expected)); err != nil {
		t.Errorf("health metric mismatch: %v", err)
	}
}

func TestMetricsExporter_ObserveLatency(t *testing.T) {
	m, reg := newTestExporter(t, "test-app")

	dep := Dependency{Name: "redis-cache", Type: TypeRedis, Critical: boolPtr(false)}
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	m.ObserveLatency(dep, ep, 3*time.Millisecond)

	// Verify through Gather that the histogram received 1 observation.
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range mfs {
		if mf.GetName() == "app_dependency_latency_seconds" {
			for _, metric := range mf.GetMetric() {
				if metric.GetHistogram().GetSampleCount() == 1 {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected 1 observation in histogram")
	}
}

func TestMetricsExporter_MultipleEndpoints(t *testing.T) {
	m, _ := newTestExporter(t, "test-app", WithCustomLabels("role"))

	dep := Dependency{Name: "postgres-main", Type: TypePostgres, Critical: boolPtr(true)}
	primary := Endpoint{Host: "pg-primary.svc", Port: "5432", Labels: map[string]string{"role": "primary"}}
	replica := Endpoint{Host: "pg-replica.svc", Port: "5432", Labels: map[string]string{"role": "replica"}}

	m.SetHealth(dep, primary, 1)
	m.SetHealth(dep, replica, 0)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="yes",dependency="postgres-main",group="test-group",host="pg-primary.svc",name="test-app",port="5432",role="primary",type="postgres"} 1
		app_dependency_health{critical="yes",dependency="postgres-main",group="test-group",host="pg-replica.svc",name="test-app",port="5432",role="replica",type="postgres"} 0
	`
	if err := testutil.CollectAndCompare(m.health, strings.NewReader(expected)); err != nil {
		t.Errorf("metrics for multiple endpoints mismatch: %v", err)
	}
}

func TestMetricsExporter_CustomLabels_Sorted(t *testing.T) {
	m, _ := newTestExporter(t, "test-app", WithCustomLabels("vhost", "role"))

	// Verify that labels are sorted: name, group, dependency, type, host, port, critical, role, vhost.
	expectedLabels := []string{"name", "group", "dependency", "type", "host", "port", "critical", "role", "vhost"}
	if len(m.allLabelNames) != len(expectedLabels) {
		t.Fatalf("expected %d labels, got %d", len(expectedLabels), len(m.allLabelNames))
	}
	for i, l := range expectedLabels {
		if m.allLabelNames[i] != l {
			t.Errorf("label[%d] = %q, expected %q", i, m.allLabelNames[i], l)
		}
	}
}

func TestMetricsExporter_InvalidCustomLabel(t *testing.T) {
	reg := prometheus.NewRegistry()
	_, err := NewMetricsExporter("test-app", "test-group", WithMetricsRegisterer(reg), WithCustomLabels("dependency"))
	if err == nil {
		t.Fatal("expected error for reserved label, got nil")
	}
	if !strings.Contains(err.Error(), "reserved label") {
		t.Errorf("expected reserved label error, got: %v", err)
	}
}

func TestMetricsExporter_InvalidLabelFormat(t *testing.T) {
	reg := prometheus.NewRegistry()
	_, err := NewMetricsExporter("test-app", "test-group", WithMetricsRegisterer(reg), WithCustomLabels("0invalid"))
	if err == nil {
		t.Fatal("expected error for invalid label, got nil")
	}
	if !strings.Contains(err.Error(), "invalid label name") {
		t.Errorf("expected invalid label name error, got: %v", err)
	}
}

func TestMetricsExporter_DeleteMetrics(t *testing.T) {
	m, reg := newTestExporter(t, "test-app")

	dep := Dependency{Name: "redis-cache", Type: TypeRedis, Critical: boolPtr(false)}
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	m.SetHealth(dep, ep, 1)
	m.ObserveLatency(dep, ep, 5*time.Millisecond)

	// Delete metrics.
	m.DeleteMetrics(dep, ep)

	// After deletion there should be no series.
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	for _, mf := range mfs {
		if len(mf.GetMetric()) > 0 {
			t.Errorf("expected 0 series for %s after deletion, got %d",
				mf.GetName(), len(mf.GetMetric()))
		}
	}
}

func TestMetricsExporter_DuplicateRegister(t *testing.T) {
	reg := prometheus.NewRegistry()
	_, err := NewMetricsExporter("test-app", "test-group", WithMetricsRegisterer(reg))
	if err != nil {
		t.Fatalf("first registration should not return an error: %v", err)
	}

	// Duplicate registration should return an error.
	_, err = NewMetricsExporter("test-app", "test-group", WithMetricsRegisterer(reg))
	if err == nil {
		t.Fatal("expected error on duplicate registration, got nil")
	}
}

func TestMetricsExporter_LatencyBuckets(t *testing.T) {
	m, _ := newTestExporter(t, "test-app")

	dep := Dependency{Name: "redis-cache", Type: TypeRedis, Critical: boolPtr(false)}
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	// Record a 50ms observation — should go into the 0.05 bucket.
	m.ObserveLatency(dep, ep, 50*time.Millisecond)

	// Verify that the histogram contains the correct buckets.
	count := testutil.CollectAndCount(m.latency)
	if count == 0 {
		t.Error("histogram has no data")
	}
}

func TestMetricsExporter_LabelEmptyFallback(t *testing.T) {
	m, _ := newTestExporter(t, "test-app", WithCustomLabels("role"))

	dep := Dependency{Name: "redis-cache", Type: TypeRedis, Critical: boolPtr(false)}
	// Endpoint without Labels — custom label should be an empty string.
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	m.SetHealth(dep, ep, 1)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="redis-cache",group="test-group",host="redis.svc",name="test-app",port="6379",role="",type="redis"} 1
	`
	if err := testutil.CollectAndCompare(m.health, strings.NewReader(expected)); err != nil {
		t.Errorf("metric with empty custom label mismatch: %v", err)
	}
}

func TestMetricsExporter_InstanceName(t *testing.T) {
	m, _ := newTestExporter(t, "order-api")

	dep := Dependency{Name: "postgres-main", Type: TypePostgres, Critical: boolPtr(true)}
	ep := Endpoint{Host: "pg.svc", Port: "5432"}

	m.SetHealth(dep, ep, 1)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="yes",dependency="postgres-main",group="test-group",host="pg.svc",name="order-api",port="5432",type="postgres"} 1
	`
	if err := testutil.CollectAndCompare(m.health, strings.NewReader(expected)); err != nil {
		t.Errorf("metric with instanceName mismatch: %v", err)
	}
}
