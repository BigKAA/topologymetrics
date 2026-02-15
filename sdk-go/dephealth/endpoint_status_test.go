package dephealth

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHealthDetails_BeforeStart(t *testing.T) {
	sched, _ := newTestScheduler(t)

	details := sched.HealthDetails()
	if details != nil {
		t.Errorf("expected nil before Start(), got %v", details)
	}
}

func TestHealthDetails_UnknownState(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics, _ := NewMetricsExporter("test-app", WithMetricsRegisterer(reg))
	sched := NewScheduler(metrics)

	// Use a checker that blocks until we signal it.
	started := make(chan struct{})
	checker := &mockChecker{
		checkFunc: func(_ context.Context, _ Endpoint) error {
			<-started // Block forever until test finishes.
			return nil
		},
	}

	crit := true
	dep := Dependency{
		Name:     "test-dep",
		Type:     TypePostgres,
		Critical: &crit,
		Endpoints: []Endpoint{
			{Host: "pg.svc", Port: "5432", Labels: map[string]string{"role": "primary"}},
		},
		Config: CheckConfig{
			Interval:         10 * time.Second,
			Timeout:          5 * time.Second,
			InitialDelay:     0,
			FailureThreshold: 1,
			SuccessThreshold: 1,
		},
	}
	addTestDep(sched, dep, checker)

	if err := sched.Start(context.Background()); err != nil {
		t.Fatalf("start error: %v", err)
	}
	defer func() {
		close(started) // Unblock the checker.
		sched.Stop()
	}()

	// Give a moment for goroutine to start (but check won't complete).
	time.Sleep(50 * time.Millisecond)

	details := sched.HealthDetails()
	if len(details) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(details))
	}

	key := "test-dep:pg.svc:5432"
	es, ok := details[key]
	if !ok {
		t.Fatalf("key %q not found", key)
	}

	// UNKNOWN state checks.
	if es.Healthy != nil {
		t.Errorf("Healthy: expected nil (UNKNOWN), got %v", *es.Healthy)
	}
	if es.Status != StatusUnknown {
		t.Errorf("Status: expected %q, got %q", StatusUnknown, es.Status)
	}
	if es.Detail != "unknown" {
		t.Errorf("Detail: expected %q, got %q", "unknown", es.Detail)
	}
	if es.Latency != 0 {
		t.Errorf("Latency: expected 0, got %v", es.Latency)
	}
	if !es.LastCheckedAt.IsZero() {
		t.Errorf("LastCheckedAt: expected zero, got %v", es.LastCheckedAt)
	}

	// Static fields should be populated.
	if es.Type != TypePostgres {
		t.Errorf("Type: expected %q, got %q", TypePostgres, es.Type)
	}
	if es.Name != "test-dep" {
		t.Errorf("Name: expected %q, got %q", "test-dep", es.Name)
	}
	if es.Host != "pg.svc" {
		t.Errorf("Host: expected %q, got %q", "pg.svc", es.Host)
	}
	if es.Port != "5432" {
		t.Errorf("Port: expected %q, got %q", "5432", es.Port)
	}
	if !es.Critical {
		t.Error("Critical: expected true, got false")
	}
	if es.Labels["role"] != "primary" {
		t.Errorf("Labels[role]: expected %q, got %q", "primary", es.Labels["role"])
	}
}

func TestHealthDetails_HealthyEndpoint(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{} // Always healthy.
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	time.Sleep(150 * time.Millisecond)

	details := sched.HealthDetails()
	key := "test-dep:127.0.0.1:1234"
	es, ok := details[key]
	if !ok {
		t.Fatalf("key %q not found", key)
	}

	if es.Healthy == nil || !*es.Healthy {
		t.Errorf("Healthy: expected true, got %v", es.Healthy)
	}
	if es.Status != StatusOK {
		t.Errorf("Status: expected %q, got %q", StatusOK, es.Status)
	}
	if es.Detail != "ok" {
		t.Errorf("Detail: expected %q, got %q", "ok", es.Detail)
	}
	if es.Latency <= 0 {
		t.Errorf("Latency: expected > 0, got %v", es.Latency)
	}
	if es.LastCheckedAt.IsZero() {
		t.Error("LastCheckedAt: expected non-zero timestamp")
	}
	if es.Type != TypeTCP {
		t.Errorf("Type: expected %q, got %q", TypeTCP, es.Type)
	}

	sched.Stop()
}

func TestHealthDetails_UnhealthyEndpoint(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{
		checkFunc: func(_ context.Context, _ Endpoint) error {
			return errors.New("connection refused")
		},
	}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	time.Sleep(150 * time.Millisecond)

	details := sched.HealthDetails()
	key := "test-dep:127.0.0.1:1234"
	es, ok := details[key]
	if !ok {
		t.Fatalf("key %q not found", key)
	}

	if es.Healthy == nil || *es.Healthy {
		t.Errorf("Healthy: expected false, got %v", es.Healthy)
	}
	if es.Status != StatusError {
		t.Errorf("Status: expected %q, got %q", StatusError, es.Status)
	}
	if es.Detail != "error" {
		t.Errorf("Detail: expected %q, got %q", "error", es.Detail)
	}
	if es.Latency <= 0 {
		t.Errorf("Latency: expected > 0, got %v", es.Latency)
	}

	sched.Stop()
}

func TestHealthDetails_KeysMatchHealth(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	crit := false
	dep := Dependency{
		Name:     "multi-ep",
		Type:     TypeTCP,
		Critical: &crit,
		Endpoints: []Endpoint{
			{Host: "host-1", Port: "1111"},
			{Host: "host-2", Port: "2222"},
		},
		Config: CheckConfig{
			Interval:         100 * time.Millisecond,
			Timeout:          50 * time.Millisecond,
			InitialDelay:     0,
			FailureThreshold: 1,
			SuccessThreshold: 1,
		},
	}
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	time.Sleep(200 * time.Millisecond)

	health := sched.Health()
	details := sched.HealthDetails()

	// All keys from Health() must be present in HealthDetails().
	for key := range health {
		if _, ok := details[key]; !ok {
			t.Errorf("key %q in Health() but not in HealthDetails()", key)
		}
	}

	// HealthDetails() includes all endpoints (same or more than Health()).
	if len(details) < len(health) {
		t.Errorf("HealthDetails() has fewer entries (%d) than Health() (%d)", len(details), len(health))
	}

	sched.Stop()
}

func TestHealthDetails_ConcurrentAccess(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	time.Sleep(150 * time.Millisecond)

	// Launch concurrent readers.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				details := sched.HealthDetails()
				if details == nil {
					t.Error("HealthDetails() returned nil during running state")
					return
				}
			}
		}()
	}

	wg.Wait()
	sched.Stop()
}

func TestHealthDetails_AfterStop(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	time.Sleep(150 * time.Millisecond)
	sched.Stop()

	// After Stop, HealthDetails should return last known state.
	details := sched.HealthDetails()
	if details == nil {
		t.Fatal("HealthDetails() returned nil after Stop()")
	}

	key := "test-dep:127.0.0.1:1234"
	es, ok := details[key]
	if !ok {
		t.Fatalf("key %q not found after Stop()", key)
	}
	if es.Healthy == nil || !*es.Healthy {
		t.Errorf("Healthy: expected true after Stop(), got %v", es.Healthy)
	}
}

func TestHealthDetails_LabelsEmpty(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	time.Sleep(150 * time.Millisecond)

	details := sched.HealthDetails()
	key := "test-dep:127.0.0.1:1234"
	es := details[key]

	// Labels should be empty map, not nil.
	if es.Labels == nil {
		t.Error("Labels: expected empty map, got nil")
	}
	if len(es.Labels) != 0 {
		t.Errorf("Labels: expected empty, got %v", es.Labels)
	}

	sched.Stop()
}

func TestHealthDetails_ResultMapIndependent(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	time.Sleep(150 * time.Millisecond)

	details := sched.HealthDetails()
	key := "test-dep:127.0.0.1:1234"

	// Modify the returned map — should not affect internal state.
	details[key] = EndpointStatus{Name: "modified"}
	delete(details, key)

	// Get fresh details — should be unaffected.
	details2 := sched.HealthDetails()
	es, ok := details2[key]
	if !ok {
		t.Fatal("internal state corrupted by external modification")
	}
	if es.Name != "test-dep" {
		t.Errorf("Name: expected %q, got %q", "test-dep", es.Name)
	}

	sched.Stop()
}

func TestDepHealth_HealthDetails(t *testing.T) {
	// Test HealthDetails() on DepHealth facade using internal scheduler directly.
	reg := prometheus.NewRegistry()
	metrics, err := NewMetricsExporter("test-app", WithMetricsRegisterer(reg))
	if err != nil {
		t.Fatalf("metrics error: %v", err)
	}
	sched := NewScheduler(metrics)

	checker := &mockChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)

	dh := &DepHealth{scheduler: sched, metrics: metrics}

	// Before Start.
	if dh.HealthDetails() != nil {
		t.Error("expected nil before Start()")
	}

	_ = dh.Start(context.Background())
	time.Sleep(200 * time.Millisecond)

	details := dh.HealthDetails()
	if details == nil {
		t.Fatal("expected non-nil HealthDetails()")
	}
	if len(details) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(details))
	}

	dh.Stop()
}

func TestEndpointStatus_JSON_Healthy(t *testing.T) {
	healthy := true
	now := time.Date(2026, 2, 14, 10, 30, 0, 0, time.UTC)
	es := EndpointStatus{
		Healthy:       &healthy,
		Status:        StatusOK,
		Detail:        "ok",
		Latency:       2300 * time.Microsecond,
		Type:          TypePostgres,
		Name:          "postgres-main",
		Host:          "pg.svc",
		Port:          "5432",
		Critical:      true,
		LastCheckedAt: now,
		Labels:        map[string]string{"role": "primary"},
	}

	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if m["healthy"] != true {
		t.Errorf("healthy: expected true, got %v", m["healthy"])
	}
	if m["status"] != "ok" {
		t.Errorf("status: expected %q, got %v", "ok", m["status"])
	}
	if m["latency_ms"] != 2.3 {
		t.Errorf("latency_ms: expected 2.3, got %v", m["latency_ms"])
	}
	if m["last_checked_at"] != "2026-02-14T10:30:00Z" {
		t.Errorf("last_checked_at: expected %q, got %v", "2026-02-14T10:30:00Z", m["last_checked_at"])
	}
	if m["critical"] != true {
		t.Errorf("critical: expected true, got %v", m["critical"])
	}
	labels := m["labels"].(map[string]interface{})
	if labels["role"] != "primary" {
		t.Errorf("labels.role: expected %q, got %v", "primary", labels["role"])
	}
}

func TestEndpointStatus_JSON_Unknown(t *testing.T) {
	es := EndpointStatus{
		Healthy:  nil,
		Status:   StatusUnknown,
		Detail:   "unknown",
		Latency:  0,
		Type:     TypeRedis,
		Name:     "redis-cache",
		Host:     "redis.svc",
		Port:     "6379",
		Critical: false,
		Labels:   map[string]string{},
	}

	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if m["healthy"] != nil {
		t.Errorf("healthy: expected null, got %v", m["healthy"])
	}
	if m["status"] != "unknown" {
		t.Errorf("status: expected %q, got %v", "unknown", m["status"])
	}
	if m["latency_ms"] != 0.0 {
		t.Errorf("latency_ms: expected 0, got %v", m["latency_ms"])
	}
	if m["last_checked_at"] != nil {
		t.Errorf("last_checked_at: expected null, got %v", m["last_checked_at"])
	}
}

func TestEndpointStatus_JSON_Roundtrip(t *testing.T) {
	healthy := false
	now := time.Date(2026, 2, 14, 10, 30, 0, 0, time.UTC)
	original := EndpointStatus{
		Healthy:       &healthy,
		Status:        StatusTimeout,
		Detail:        "timeout",
		Latency:       5000 * time.Millisecond,
		Type:          TypeHTTP,
		Name:          "api-gw",
		Host:          "api.svc",
		Port:          "8080",
		Critical:      true,
		LastCheckedAt: now,
		Labels:        map[string]string{"env": "prod"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded EndpointStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Healthy == nil || *decoded.Healthy != false {
		t.Errorf("Healthy: expected false, got %v", decoded.Healthy)
	}
	if decoded.Status != StatusTimeout {
		t.Errorf("Status: expected %q, got %q", StatusTimeout, decoded.Status)
	}
	if decoded.Name != "api-gw" {
		t.Errorf("Name: expected %q, got %q", "api-gw", decoded.Name)
	}
	// Latency is approximate due to float conversion.
	if decoded.Latency < 4999*time.Millisecond || decoded.Latency > 5001*time.Millisecond {
		t.Errorf("Latency: expected ~5000ms, got %v", decoded.Latency)
	}
	if decoded.Labels["env"] != "prod" {
		t.Errorf("Labels: expected env=prod, got %v", decoded.Labels)
	}
}

func TestEndpointStatus_LatencyMillis(t *testing.T) {
	es := EndpointStatus{Latency: 2500 * time.Microsecond}
	if es.LatencyMillis() != 2.5 {
		t.Errorf("LatencyMillis: expected 2.5, got %v", es.LatencyMillis())
	}
}
