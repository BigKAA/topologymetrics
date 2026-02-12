package dephealth

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// registerMockFactory registers a mock factory for the specified dependency type.
func registerMockFactory(t *testing.T, depType DependencyType, checker HealthChecker) {
	t.Helper()
	old := checkerFactories[depType]
	checkerFactories[depType] = func(_ *DependencyConfig) HealthChecker {
		return checker
	}
	t.Cleanup(func() {
		if old != nil {
			checkerFactories[depType] = old
		} else {
			delete(checkerFactories, depType)
		}
	})
}

func TestNew_ValidHTTP(t *testing.T) {
	reg := prometheus.NewRegistry()
	checker := &mockChecker{}
	registerMockFactory(t, TypeHTTP, checker)

	dh, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(true)),
	)
	if err != nil {
		t.Fatalf("failed to create DepHealth: %v", err)
	}
	if dh == nil {
		t.Fatal("DepHealth should not be nil")
	}
}

func TestNew_NoDependencies(t *testing.T) {
	reg := prometheus.NewRegistry()

	dh, err := New("test-app", WithRegisterer(reg))
	if err != nil {
		t.Fatalf("zero deps should be allowed: %v", err)
	}
	if dh == nil {
		t.Fatal("DepHealth should not be nil")
	}

	// Health() before Start — empty collection.
	health := dh.Health()
	if len(health) != 0 {
		t.Fatalf("expected empty Health(), got %d entries", len(health))
	}

	// Start()/Stop() — no-op, no panic.
	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("Start() should not return an error: %v", err)
	}
	dh.Stop()
}

func TestNew_MissingName(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	_, err := New("",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(true)),
	)
	if err == nil {
		t.Fatal("expected error when name is missing")
	}
}

func TestNew_NameFromEnv(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	t.Setenv("DEPHEALTH_NAME", "env-app")

	dh, err := New("",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(true)),
	)
	if err != nil {
		t.Fatalf("expected success with name from env: %v", err)
	}
	if dh == nil {
		t.Fatal("DepHealth should not be nil")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	_, err := New("test-app",
		WithRegisterer(reg),
		HTTP("bad", FromURL("://invalid"), Critical(true)),
	)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNew_MissingHostPort(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	_, err := New("test-app",
		WithRegisterer(reg),
		HTTP("no-addr", Critical(true)),
	)
	if err == nil {
		t.Fatal("expected error when URL and host/port are missing")
	}
}

func TestNew_MissingCritical(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	_, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080")),
	)
	if err == nil {
		t.Fatal("expected error when critical is missing")
	}
}

func TestNew_FromParams(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeTCP, &mockChecker{})

	dh, err := New("test-app",
		WithRegisterer(reg),
		TCP("my-tcp", FromParams("127.0.0.1", "8080"), Critical(false)),
	)
	if err != nil {
		t.Fatalf("failed to create DepHealth: %v", err)
	}
	if dh == nil {
		t.Fatal("DepHealth should not be nil")
	}
}

func TestNew_CriticalFlag(t *testing.T) {
	reg := prometheus.NewRegistry()
	checker := &mockChecker{}
	registerMockFactory(t, TypeHTTP, checker)

	dh, err := New("test-app",
		WithRegisterer(reg),
		HTTP("critical-api",
			FromURL("http://api.svc:8080"),
			Critical(true),
		),
	)
	if err != nil {
		t.Fatalf("creation error: %v", err)
	}

	// Verify that the dependency is marked as critical.
	if dh.scheduler.deps[0].dep.Critical == nil || !*dh.scheduler.deps[0].dep.Critical {
		t.Error("expected Critical=true")
	}
}

func TestNew_CriticalFromEnv(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	t.Setenv("DEPHEALTH_WEB_API_CRITICAL", "yes")

	dh, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080")),
	)
	if err != nil {
		t.Fatalf("expected success with critical from env: %v", err)
	}
	if dh.scheduler.deps[0].dep.Critical == nil || !*dh.scheduler.deps[0].dep.Critical {
		t.Error("expected Critical=true from env")
	}
}

func TestNew_WithLabel(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	dh, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api",
			FromURL("http://api.svc:8080"),
			Critical(true),
			WithLabel("role", "primary"),
		),
	)
	if err != nil {
		t.Fatalf("creation error: %v", err)
	}

	ep := dh.scheduler.deps[0].dep.Endpoints[0]
	if ep.Labels["role"] != "primary" {
		t.Errorf("expected label role=primary, got %q", ep.Labels["role"])
	}
}

func TestNew_ReservedLabel(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	_, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api",
			FromURL("http://api.svc:8080"),
			Critical(true),
			WithLabel("dependency", "bad"),
		),
	)
	if err == nil {
		t.Fatal("expected error for reserved label")
	}
}

func TestNew_LabelFromEnv(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	t.Setenv("DEPHEALTH_WEB_API_LABEL_ROLE", "replica")

	dh, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api",
			FromURL("http://api.svc:8080"),
			Critical(true),
		),
	)
	if err != nil {
		t.Fatalf("creation error: %v", err)
	}

	ep := dh.scheduler.deps[0].dep.Endpoints[0]
	if ep.Labels["role"] != "replica" {
		t.Errorf("expected label role=replica from env, got %q", ep.Labels["role"])
	}
}

func TestDepHealth_StartStop(t *testing.T) {
	reg := prometheus.NewRegistry()
	checker := &mockChecker{}
	registerMockFactory(t, TypeHTTP, checker)

	dh, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(false)),
	)
	if err != nil {
		t.Fatalf("creation error: %v", err)
	}

	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("start error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	dh.Stop()

	if checker.callCount.Load() == 0 {
		t.Error("expected at least one checker call")
	}
}

func TestDepHealth_Health(t *testing.T) {
	reg := prometheus.NewRegistry()
	checker := &mockChecker{}
	registerMockFactory(t, TypeHTTP, checker)

	dh, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(false)),
	)
	if err != nil {
		t.Fatalf("creation error: %v", err)
	}

	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("start error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	health := dh.Health()
	if len(health) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(health))
	}

	key := "web-api:api.svc:8080"
	val, ok := health[key]
	if !ok {
		t.Fatalf("key %q not found in Health()", key)
	}
	if !val {
		t.Errorf("expected healthy=true")
	}

	dh.Stop()
}

func TestNew_GlobalCheckInterval(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	dh, err := New("test-app",
		WithRegisterer(reg),
		WithCheckInterval(30*time.Second),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(false)),
	)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	got := dh.scheduler.deps[0].dep.Config.Interval
	if got != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", got)
	}
}

func TestNew_PerDepCheckInterval(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	dh, err := New("test-app",
		WithRegisterer(reg),
		WithCheckInterval(30*time.Second),
		HTTP("web-api",
			FromURL("http://api.svc:8080"),
			Critical(false),
			CheckInterval(10*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	got := dh.scheduler.deps[0].dep.Config.Interval
	if got != 10*time.Second {
		t.Errorf("per-dep interval should override global: expected 10s, got %v", got)
	}
}

func TestNew_GlobalTimeout(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})

	dh, err := New("test-app",
		WithRegisterer(reg),
		WithTimeout(3*time.Second),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(false)),
	)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	got := dh.scheduler.deps[0].dep.Config.Timeout
	if got != 3*time.Second {
		t.Errorf("expected timeout 3s, got %v", got)
	}
}

func TestNew_HTTPSAutoTLS(t *testing.T) {
	reg := prometheus.NewRegistry()

	// Special mock that verifies TLS is enabled.
	var gotConfig *DependencyConfig
	old := checkerFactories[TypeHTTP]
	checkerFactories[TypeHTTP] = func(dc *DependencyConfig) HealthChecker {
		gotConfig = dc
		return &mockChecker{}
	}
	t.Cleanup(func() {
		if old != nil {
			checkerFactories[TypeHTTP] = old
		} else {
			delete(checkerFactories, TypeHTTP)
		}
	})

	_, err := New("test-app",
		WithRegisterer(reg),
		HTTP("secure-api", FromURL("https://api.svc:443"), Critical(false)),
	)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if gotConfig == nil || gotConfig.HTTPTLS == nil || !*gotConfig.HTTPTLS {
		t.Error("expected automatic TLS enablement for https:// URL")
	}
}

func TestNew_AddDependency(t *testing.T) {
	reg := prometheus.NewRegistry()
	checker := &mockChecker{}

	dh, err := New("test-app",
		WithRegisterer(reg),
		AddDependency("custom-dep", TypeTCP, checker,
			FromParams("10.0.0.1", "9999"),
			Critical(false),
		),
	)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(dh.scheduler.deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(dh.scheduler.deps))
	}
	if dh.scheduler.deps[0].dep.Name != "custom-dep" {
		t.Errorf("expected name custom-dep, got %q", dh.scheduler.deps[0].dep.Name)
	}
}

func TestNew_MultipleDependencies(t *testing.T) {
	reg := prometheus.NewRegistry()
	registerMockFactory(t, TypeHTTP, &mockChecker{})
	registerMockFactory(t, TypeTCP, &mockChecker{})

	dh, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(true)),
		TCP("cache", FromParams("redis.svc", "6379"), Critical(false)),
	)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(dh.scheduler.deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(dh.scheduler.deps))
	}
}

func TestNew_NoFactoryRegistered(t *testing.T) {
	reg := prometheus.NewRegistry()
	// Don't register a factory — should produce an error.

	_, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(false)),
	)
	if err == nil {
		t.Fatal("expected error when no factory is registered")
	}
}

func TestNew_CheckerWrappers(t *testing.T) {
	reg := prometheus.NewRegistry()

	var gotConfig *DependencyConfig
	old := checkerFactories[TypeHTTP]
	checkerFactories[TypeHTTP] = func(dc *DependencyConfig) HealthChecker {
		gotConfig = dc
		return &mockChecker{}
	}
	t.Cleanup(func() {
		if old != nil {
			checkerFactories[TypeHTTP] = old
		} else {
			delete(checkerFactories, TypeHTTP)
		}
	})

	_, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api",
			FromURL("http://api.svc:8080"),
			Critical(false),
			WithHTTPHealthPath("/ready"),
			WithHTTPTLSSkipVerify(true),
		),
	)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if gotConfig.HTTPHealthPath != "/ready" {
		t.Errorf("expected healthPath=/ready, got %q", gotConfig.HTTPHealthPath)
	}
	if gotConfig.HTTPTLSSkipVerify == nil || !*gotConfig.HTTPTLSSkipVerify {
		t.Error("expected TLSSkipVerify=true")
	}
}
