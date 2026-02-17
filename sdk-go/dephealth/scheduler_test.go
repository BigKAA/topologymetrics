package dephealth

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// mockChecker is a mock checker for scheduler tests.
type mockChecker struct {
	checkFunc func(ctx context.Context, ep Endpoint) error
	callCount atomic.Int64
}

func (m *mockChecker) Check(ctx context.Context, ep Endpoint) error {
	m.callCount.Add(1)
	if m.checkFunc != nil {
		return m.checkFunc(ctx, ep)
	}
	return nil
}

func (m *mockChecker) Type() string { return "mock" }

// panicChecker is a checker that panics.
type panicChecker struct{}

func (p *panicChecker) Check(_ context.Context, _ Endpoint) error {
	panic("test panic")
}

func (p *panicChecker) Type() string { return "panic" }

func testDep(name string, interval, timeout, initialDelay time.Duration) Dependency { //nolint:unparam // name is parameterized for test readability
	crit := false
	return Dependency{
		Name:     name,
		Type:     TypeTCP,
		Critical: &crit,
		Endpoints: []Endpoint{
			{Host: "127.0.0.1", Port: "1234"},
		},
		Config: CheckConfig{
			Interval:         interval,
			Timeout:          timeout,
			InitialDelay:     initialDelay,
			FailureThreshold: DefaultFailureThreshold,
			SuccessThreshold: DefaultSuccessThreshold,
		},
	}
}

func testDepWithThresholds(name string, failThreshold, successThreshold int) Dependency {
	crit := false
	return Dependency{
		Name:     name,
		Type:     TypeTCP,
		Critical: &crit,
		Endpoints: []Endpoint{
			{Host: "127.0.0.1", Port: "1234"},
		},
		Config: CheckConfig{
			Interval:         100 * time.Millisecond,
			Timeout:          50 * time.Millisecond,
			InitialDelay:     0,
			FailureThreshold: failThreshold,
			SuccessThreshold: successThreshold,
		},
	}
}

func newTestScheduler(t *testing.T) (*Scheduler, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	metrics, err := NewMetricsExporter("test-app", WithMetricsRegisterer(reg))
	if err != nil {
		t.Fatalf("failed to create MetricsExporter: %v", err)
	}
	return NewScheduler(metrics), reg
}

// addTestDep adds a dependency without validation (for tests with fast intervals).
func addTestDep(s *Scheduler, dep Dependency, checker HealthChecker) {
	s.deps = append(s.deps, scheduledDep{dep: dep, checker: checker})
}

func TestScheduler_StartStop(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)

	if err := sched.Start(context.Background()); err != nil {
		t.Fatalf("start error: %v", err)
	}

	// Wait for at least one check to complete.
	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	if checker.callCount.Load() == 0 {
		t.Error("expected at least one checker call")
	}
}

func TestScheduler_DoubleStart(t *testing.T) {
	sched, _ := newTestScheduler(t)

	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, &mockChecker{})

	if err := sched.Start(context.Background()); err != nil {
		t.Fatalf("first Start should not return an error: %v", err)
	}
	defer sched.Stop()

	if err := sched.Start(context.Background()); !errors.Is(err, ErrAlreadyStarted) {
		t.Errorf("expected ErrAlreadyStarted, got: %v", err)
	}
}

func TestScheduler_DoubleStop(t *testing.T) {
	sched, _ := newTestScheduler(t)

	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, &mockChecker{})
	_ = sched.Start(context.Background())

	// Repeated Stop is no-op, should not panic.
	sched.Stop()
	sched.Stop()
}

func TestScheduler_InitialDelay(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 200*time.Millisecond, 50*time.Millisecond, 150*time.Millisecond)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	// After 100ms (before initialDelay of 150ms) the check should not have run yet.
	time.Sleep(100 * time.Millisecond)
	if checker.callCount.Load() > 0 {
		t.Error("check ran before initialDelay expired")
	}

	// After another 150ms (total 250ms) the first check should have run.
	time.Sleep(150 * time.Millisecond)
	sched.Stop()

	if checker.callCount.Load() == 0 {
		t.Error("check did not run after initialDelay")
	}
}

func TestScheduler_HealthyMetric(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{} // Always returns nil (healthy).
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	time.Sleep(150 * time.Millisecond)
	sched.Stop()

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 1
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("health metric mismatch: %v", err)
	}
}

func TestScheduler_UnhealthyMetric(t *testing.T) {
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
	sched.Stop()

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 0
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("health metric mismatch: %v", err)
	}
}

func TestScheduler_FailureThreshold(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics, _ := NewMetricsExporter("test-app", WithMetricsRegisterer(reg))
	sched := NewScheduler(metrics)

	callCount := atomic.Int64{}
	checker := &mockChecker{
		checkFunc: func(_ context.Context, _ Endpoint) error {
			n := callCount.Add(1)
			if n == 1 {
				return nil // First check — OK.
			}
			return errors.New("fail") // Subsequent checks — errors.
		},
	}

	dep := testDepWithThresholds("test-dep", 3, 1)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	// Wait for 1st check (OK) + 2 failures (threshold of 3 not reached).
	time.Sleep(250 * time.Millisecond)

	// Metric should be 1 — threshold not yet reached.
	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 1
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("metric should be 1 before reaching threshold: %v", err)
	}

	// Wait more — threshold should be reached.
	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	expected = `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 0
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("metric should be 0 after reaching threshold: %v", err)
	}
}

func TestScheduler_Recovery(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics, _ := NewMetricsExporter("test-app", WithMetricsRegisterer(reg))
	sched := NewScheduler(metrics)

	shouldFail := atomic.Bool{}
	shouldFail.Store(true)

	checker := &mockChecker{
		checkFunc: func(_ context.Context, _ Endpoint) error {
			if shouldFail.Load() {
				return errors.New("fail")
			}
			return nil
		},
	}

	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	// Wait for it to become unhealthy.
	time.Sleep(150 * time.Millisecond)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 0
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("metric should be 0: %v", err)
	}

	// Enable "healthy" state.
	shouldFail.Store(false)
	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	expected = `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 1
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("metric should be 1 after recovery: %v", err)
	}
}

func TestScheduler_LatencyRecorded(t *testing.T) {
	sched, reg := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	time.Sleep(150 * time.Millisecond)
	sched.Stop()

	// Verify that the histogram contains data.
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range mfs {
		if mf.GetName() == "app_dependency_latency_seconds" {
			for _, metric := range mf.GetMetric() {
				if metric.GetHistogram().GetSampleCount() > 0 {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("histogram has no observations after checks")
	}
}

func TestScheduler_PanicRecovery(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &panicChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	// Panic should not crash the scheduler.
	time.Sleep(250 * time.Millisecond)
	sched.Stop()

	// Metric should be 0 (panic = check failure).
	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 0
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("metric should be 0 after panic: %v", err)
	}
}

func TestScheduler_MultipleEndpoints(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics, _ := NewMetricsExporter("test-app", WithMetricsRegisterer(reg))
	sched := NewScheduler(metrics)

	crit := false
	checker := &mockChecker{}
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

	time.Sleep(150 * time.Millisecond)
	sched.Stop()

	// Both endpoints should be healthy.
	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="multi-ep",host="host-1",name="test-app",port="1111",type="tcp"} 1
		app_dependency_health{critical="no",dependency="multi-ep",host="host-2",name="test-app",port="2222",type="tcp"} 1
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("metrics for multiple endpoints mismatch: %v", err)
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)

	ctx, cancel := context.WithCancel(context.Background())
	_ = sched.Start(ctx)

	time.Sleep(150 * time.Millisecond)
	cancel() // Cancel the outer context.

	// Stop should complete without hanging.
	done := make(chan struct{})
	go func() {
		sched.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK.
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not complete after context cancellation")
	}
}

func TestScheduler_Health(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{} // Always healthy.
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	// Wait for the first check.
	time.Sleep(150 * time.Millisecond)

	health := sched.Health()
	if len(health) != 1 {
		t.Fatalf("expected 1 entry in Health(), got %d", len(health))
	}

	key := "test-dep:127.0.0.1:1234"
	val, ok := health[key]
	if !ok {
		t.Fatalf("key %q not found in Health()", key)
	}
	if !val {
		t.Errorf("expected healthy=true for %q, got false", key)
	}

	sched.Stop()
}

func TestScheduler_Health_BeforeStart(t *testing.T) {
	sched, _ := newTestScheduler(t)

	// Before Start — Health() should return nil.
	health := sched.Health()
	if health != nil {
		t.Errorf("expected nil before Start(), got %v", health)
	}
}
