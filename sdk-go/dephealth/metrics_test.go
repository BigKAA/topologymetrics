package dephealth

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

)

func newTestExporter(t *testing.T, opts ...MetricsOption) (*MetricsExporter, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	allOpts := append([]MetricsOption{WithRegisterer(reg)}, opts...)
	m, err := NewMetricsExporter(allOpts...)
	if err != nil {
		t.Fatalf("не удалось создать MetricsExporter: %v", err)
	}
	return m, reg
}

func TestMetricsExporter_SetHealth(t *testing.T) {
	m, _ := newTestExporter(t)

	dep := Dependency{Name: "postgres-main", Type: TypePostgres}
	ep := Endpoint{Host: "pg.svc", Port: "5432"}

	m.SetHealth(dep, ep, 1)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{dependency="postgres-main",host="pg.svc",port="5432",type="postgres"} 1
	`
	if err := testutil.CollectAndCompare(m.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрика health не совпадает: %v", err)
	}
}

func TestMetricsExporter_SetHealth_Unhealthy(t *testing.T) {
	m, _ := newTestExporter(t)

	dep := Dependency{Name: "redis-cache", Type: TypeRedis}
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	m.SetHealth(dep, ep, 0)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{dependency="redis-cache",host="redis.svc",port="6379",type="redis"} 0
	`
	if err := testutil.CollectAndCompare(m.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрика health не совпадает: %v", err)
	}
}

func TestMetricsExporter_ObserveLatency(t *testing.T) {
	m, reg := newTestExporter(t)

	dep := Dependency{Name: "redis-cache", Type: TypeRedis}
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	m.ObserveLatency(dep, ep, 3*time.Millisecond)

	// Проверяем через Gather, что histogram получил 1 наблюдение.
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("ошибка сбора метрик: %v", err)
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
		t.Error("ожидали 1 наблюдение в histogram")
	}
}

func TestMetricsExporter_MultipleEndpoints(t *testing.T) {
	m, _ := newTestExporter(t, WithOptionalLabels("role"))

	dep := Dependency{Name: "postgres-main", Type: TypePostgres}
	primary := Endpoint{Host: "pg-primary.svc", Port: "5432", Metadata: map[string]string{"role": "primary"}}
	replica := Endpoint{Host: "pg-replica.svc", Port: "5432", Metadata: map[string]string{"role": "replica"}}

	m.SetHealth(dep, primary, 1)
	m.SetHealth(dep, replica, 0)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{dependency="postgres-main",host="pg-primary.svc",port="5432",role="primary",type="postgres"} 1
		app_dependency_health{dependency="postgres-main",host="pg-replica.svc",port="5432",role="replica",type="postgres"} 0
	`
	if err := testutil.CollectAndCompare(m.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрики для нескольких endpoint-ов не совпадают: %v", err)
	}
}

func TestMetricsExporter_OptionalLabels_Sorted(t *testing.T) {
	m, _ := newTestExporter(t, WithOptionalLabels("vhost", "role"))

	// Проверяем что метки отсортированы: role, vhost (после обязательных).
	expectedLabels := []string{"dependency", "type", "host", "port", "role", "vhost"}
	if len(m.allLabelNames) != len(expectedLabels) {
		t.Fatalf("ожидали %d меток, получили %d", len(expectedLabels), len(m.allLabelNames))
	}
	for i, l := range expectedLabels {
		if m.allLabelNames[i] != l {
			t.Errorf("метка[%d] = %q, ожидали %q", i, m.allLabelNames[i], l)
		}
	}
}

func TestMetricsExporter_InvalidLabel(t *testing.T) {
	reg := prometheus.NewRegistry()
	_, err := NewMetricsExporter(WithRegisterer(reg), WithOptionalLabels("invalid"))
	if err == nil {
		t.Fatal("ожидали ошибку для недопустимой метки, получили nil")
	}

	var labelErr *InvalidLabelError
	if !strings.Contains(err.Error(), "invalid optional label") {
		t.Errorf("ожидали InvalidLabelError, получили: %v", err)
	}
	_ = labelErr
}

func TestMetricsExporter_DeleteMetrics(t *testing.T) {
	m, reg := newTestExporter(t)

	dep := Dependency{Name: "redis-cache", Type: TypeRedis}
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	m.SetHealth(dep, ep, 1)
	m.ObserveLatency(dep, ep, 5*time.Millisecond)

	// Удаляем метрики.
	m.DeleteMetrics(dep, ep)

	// После удаления метрик не должно быть серий.
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("ошибка сбора метрик: %v", err)
	}
	for _, mf := range mfs {
		if len(mf.GetMetric()) > 0 {
			t.Errorf("ожидали 0 серий для %s после удаления, получили %d",
				mf.GetName(), len(mf.GetMetric()))
		}
	}
}

func TestMetricsExporter_DuplicateRegister(t *testing.T) {
	reg := prometheus.NewRegistry()
	_, err := NewMetricsExporter(WithRegisterer(reg))
	if err != nil {
		t.Fatalf("первая регистрация не должна вернуть ошибку: %v", err)
	}

	// Повторная регистрация должна вернуть ошибку.
	_, err = NewMetricsExporter(WithRegisterer(reg))
	if err == nil {
		t.Fatal("ожидали ошибку при повторной регистрации, получили nil")
	}
}

func TestMetricsExporter_LatencyBuckets(t *testing.T) {
	m, _ := newTestExporter(t)

	dep := Dependency{Name: "redis-cache", Type: TypeRedis}
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	// Записываем наблюдение 50ms — должно попасть в бакет 0.05.
	m.ObserveLatency(dep, ep, 50*time.Millisecond)

	// Проверяем что histogram содержит правильные бакеты.
	count := testutil.CollectAndCount(m.latency)
	if count == 0 {
		t.Error("histogram не содержит данных")
	}
}

func TestMetricsExporter_MetadataEmptyFallback(t *testing.T) {
	m, _ := newTestExporter(t, WithOptionalLabels("role"))

	dep := Dependency{Name: "redis-cache", Type: TypeRedis}
	// Endpoint без Metadata — опциональная метка должна быть пустой строкой.
	ep := Endpoint{Host: "redis.svc", Port: "6379"}

	m.SetHealth(dep, ep, 1)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{dependency="redis-cache",host="redis.svc",port="6379",role="",type="redis"} 1
	`
	if err := testutil.CollectAndCompare(m.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрика с пустой опциональной меткой не совпадает: %v", err)
	}
}
