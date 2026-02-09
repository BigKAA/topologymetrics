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

// mockChecker — мок-чекер для тестов планировщика.
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

// panicChecker — чекер, вызывающий панику.
type panicChecker struct{}

func (p *panicChecker) Check(_ context.Context, _ Endpoint) error {
	panic("test panic")
}

func (p *panicChecker) Type() string { return "panic" }

func testDep(name string, interval, timeout, initialDelay time.Duration) Dependency { //nolint:unparam // name параметризован для читаемости тестов
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
		t.Fatalf("не удалось создать MetricsExporter: %v", err)
	}
	return NewScheduler(metrics), reg
}

// addTestDep добавляет зависимость без валидации (для тестов с быстрыми интервалами).
func addTestDep(s *Scheduler, dep Dependency, checker HealthChecker) {
	s.deps = append(s.deps, scheduledDep{dep: dep, checker: checker})
}

func TestScheduler_StartStop(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)

	if err := sched.Start(context.Background()); err != nil {
		t.Fatalf("ошибка запуска: %v", err)
	}

	// Ждём чтобы прошла хотя бы одна проверка.
	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	if checker.callCount.Load() == 0 {
		t.Error("ожидали хотя бы один вызов чекера")
	}
}

func TestScheduler_DoubleStart(t *testing.T) {
	sched, _ := newTestScheduler(t)

	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, &mockChecker{})

	if err := sched.Start(context.Background()); err != nil {
		t.Fatalf("первый Start не должен вернуть ошибку: %v", err)
	}
	defer sched.Stop()

	if err := sched.Start(context.Background()); !errors.Is(err, ErrAlreadyStarted) {
		t.Errorf("ожидали ErrAlreadyStarted, получили: %v", err)
	}
}

func TestScheduler_DoubleStop(t *testing.T) {
	sched, _ := newTestScheduler(t)

	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, &mockChecker{})
	_ = sched.Start(context.Background())

	// Повторный Stop — no-op, не должно быть паники.
	sched.Stop()
	sched.Stop()
}

func TestScheduler_AddAfterStart(t *testing.T) {
	sched, _ := newTestScheduler(t)

	dep := testDep("test-dep", 2*time.Second, 500*time.Millisecond, 0)
	addTestDep(sched, dep, &mockChecker{})
	_ = sched.Start(context.Background())
	defer sched.Stop()

	// Добавление после старта через Add — должно вернуть ErrAlreadyStarted.
	crit := false
	dep2 := Dependency{
		Name:     "test-dep-2",
		Type:     TypeTCP,
		Critical: &crit,
		Endpoints: []Endpoint{
			{Host: "127.0.0.1", Port: "5678"},
		},
		Config: DefaultCheckConfig(),
	}
	if err := sched.Add(dep2, &mockChecker{}); !errors.Is(err, ErrAlreadyStarted) {
		t.Errorf("ожидали ErrAlreadyStarted при добавлении после Start, получили: %v", err)
	}
}

func TestScheduler_InitialDelay(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{}
	dep := testDep("test-dep", 200*time.Millisecond, 50*time.Millisecond, 150*time.Millisecond)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	// Через 100ms (до initialDelay 150ms) проверка ещё не должна пройти.
	time.Sleep(100 * time.Millisecond)
	if checker.callCount.Load() > 0 {
		t.Error("проверка прошла до истечения initialDelay")
	}

	// Через ещё 150ms (суммарно 250ms) первая проверка должна пройти.
	time.Sleep(150 * time.Millisecond)
	sched.Stop()

	if checker.callCount.Load() == 0 {
		t.Error("проверка не прошла после initialDelay")
	}
}

func TestScheduler_HealthyMetric(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{} // Всегда возвращает nil (healthy).
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
		t.Errorf("метрика health не совпадает: %v", err)
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
		t.Errorf("метрика health не совпадает: %v", err)
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
				return nil // Первая проверка — ОК.
			}
			return errors.New("fail") // Далее — ошибки.
		},
	}

	dep := testDepWithThresholds("test-dep", 3, 1)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	// Ждём 1-ю проверку (OK) + 2 ошибки (не достигли порога 3).
	time.Sleep(250 * time.Millisecond)

	// Метрика должна быть 1 — порог ещё не достигнут.
	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 1
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрика должна быть 1 до достижения порога: %v", err)
	}

	// Ждём ещё — порог должен быть достигнут.
	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	expected = `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 0
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрика должна быть 0 после достижения порога: %v", err)
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

	// Ждём чтобы стало unhealthy.
	time.Sleep(150 * time.Millisecond)

	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 0
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрика должна быть 0: %v", err)
	}

	// Включаем «здоровье».
	shouldFail.Store(false)
	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	expected = `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 1
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрика должна быть 1 после восстановления: %v", err)
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

	// Проверяем что histogram содержит данные.
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("ошибка сбора метрик: %v", err)
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
		t.Error("histogram не содержит наблюдений после проверок")
	}
}

func TestScheduler_PanicRecovery(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &panicChecker{}
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	// Паника не должна прервать планировщик.
	time.Sleep(250 * time.Millisecond)
	sched.Stop()

	// Метрика должна быть 0 (паника = ошибка проверки).
	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="test-dep",host="127.0.0.1",name="test-app",port="1234",type="tcp"} 0
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрика должна быть 0 после паники: %v", err)
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

	// Оба endpoint-а должны быть healthy.
	expected := `
		# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
		# TYPE app_dependency_health gauge
		app_dependency_health{critical="no",dependency="multi-ep",host="host-1",name="test-app",port="1111",type="tcp"} 1
		app_dependency_health{critical="no",dependency="multi-ep",host="host-2",name="test-app",port="2222",type="tcp"} 1
	`
	if err := testutil.CollectAndCompare(sched.metrics.health, strings.NewReader(expected)); err != nil {
		t.Errorf("метрики для нескольких endpoint-ов не совпадают: %v", err)
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
	cancel() // Отмена внешнего контекста.

	// Stop должен завершиться без зависания.
	done := make(chan struct{})
	go func() {
		sched.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK.
	case <-time.After(2 * time.Second):
		t.Fatal("Stop не завершился после отмены контекста")
	}
}

func TestScheduler_InvalidDependency(t *testing.T) {
	sched, _ := newTestScheduler(t)

	crit := false
	// Зависимость без endpoint-ов — невалидна.
	dep := Dependency{
		Name:      "bad-dep",
		Type:      TypeTCP,
		Critical:  &crit,
		Endpoints: nil,
		Config:    DefaultCheckConfig(),
	}

	err := sched.Add(dep, &mockChecker{})
	if err == nil {
		t.Error("ожидали ошибку для невалидной зависимости, получили nil")
	}
}

func TestScheduler_ValidAdd(t *testing.T) {
	sched, _ := newTestScheduler(t)

	crit := true
	dep := Dependency{
		Name:     "valid-dep",
		Type:     TypeTCP,
		Critical: &crit,
		Endpoints: []Endpoint{
			{Host: "127.0.0.1", Port: "5432"},
		},
		Config: DefaultCheckConfig(),
	}

	if err := sched.Add(dep, &mockChecker{}); err != nil {
		t.Errorf("ожидали nil для валидной зависимости, получили: %v", err)
	}
}

func TestScheduler_Health(t *testing.T) {
	sched, _ := newTestScheduler(t)

	checker := &mockChecker{} // Всегда healthy.
	dep := testDep("test-dep", 100*time.Millisecond, 50*time.Millisecond, 0)
	addTestDep(sched, dep, checker)
	_ = sched.Start(context.Background())

	// Ждём первую проверку.
	time.Sleep(150 * time.Millisecond)

	health := sched.Health()
	if len(health) != 1 {
		t.Fatalf("ожидали 1 запись в Health(), получили %d", len(health))
	}

	key := "test-dep:127.0.0.1:1234"
	val, ok := health[key]
	if !ok {
		t.Fatalf("ключ %q не найден в Health()", key)
	}
	if !val {
		t.Errorf("ожидали healthy=true для %q, получили false", key)
	}

	sched.Stop()
}

func TestScheduler_Health_BeforeStart(t *testing.T) {
	sched, _ := newTestScheduler(t)

	// До Start — Health() должен вернуть nil.
	health := sched.Health()
	if health != nil {
		t.Errorf("ожидали nil до Start(), получили %v", health)
	}
}
