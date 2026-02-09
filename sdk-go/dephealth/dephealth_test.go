package dephealth

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// registerMockFactory регистрирует мок-фабрику для указанного типа зависимости.
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
		t.Fatalf("ошибка создания DepHealth: %v", err)
	}
	if dh == nil {
		t.Fatal("DepHealth не должен быть nil")
	}
}

func TestNew_NoDependencies(t *testing.T) {
	reg := prometheus.NewRegistry()

	dh, err := New("test-app", WithRegisterer(reg))
	if err != nil {
		t.Fatalf("zero deps должен быть допустим: %v", err)
	}
	if dh == nil {
		t.Fatal("DepHealth не должен быть nil")
	}

	// Health() до Start — пустая коллекция.
	health := dh.Health()
	if len(health) != 0 {
		t.Fatalf("ожидали пустую Health(), получили %d записей", len(health))
	}

	// Start()/Stop() — no-op, без паники.
	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("Start() не должен возвращать ошибку: %v", err)
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
		t.Fatal("ожидали ошибку при отсутствии name")
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
		t.Fatalf("ожидали успех с name из env: %v", err)
	}
	if dh == nil {
		t.Fatal("DepHealth не должен быть nil")
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
		t.Fatal("ожидали ошибку для невалидного URL")
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
		t.Fatal("ожидали ошибку при отсутствии URL и host/port")
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
		t.Fatal("ожидали ошибку при отсутствии critical")
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
		t.Fatalf("ошибка создания DepHealth: %v", err)
	}
	if dh == nil {
		t.Fatal("DepHealth не должен быть nil")
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
		t.Fatalf("ошибка создания: %v", err)
	}

	// Проверяем, что зависимость помечена как critical.
	if dh.scheduler.deps[0].dep.Critical == nil || !*dh.scheduler.deps[0].dep.Critical {
		t.Error("ожидали Critical=true")
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
		t.Fatalf("ожидали успех с critical из env: %v", err)
	}
	if dh.scheduler.deps[0].dep.Critical == nil || !*dh.scheduler.deps[0].dep.Critical {
		t.Error("ожидали Critical=true из env")
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
		t.Fatalf("ошибка создания: %v", err)
	}

	ep := dh.scheduler.deps[0].dep.Endpoints[0]
	if ep.Labels["role"] != "primary" {
		t.Errorf("ожидали label role=primary, получили %q", ep.Labels["role"])
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
		t.Fatal("ожидали ошибку для зарезервированной метки")
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
		t.Fatalf("ошибка создания: %v", err)
	}

	ep := dh.scheduler.deps[0].dep.Endpoints[0]
	if ep.Labels["role"] != "replica" {
		t.Errorf("ожидали label role=replica из env, получили %q", ep.Labels["role"])
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
		t.Fatalf("ошибка создания: %v", err)
	}

	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("ошибка запуска: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	dh.Stop()

	if checker.callCount.Load() == 0 {
		t.Error("ожидали хотя бы один вызов чекера")
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
		t.Fatalf("ошибка создания: %v", err)
	}

	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("ошибка запуска: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	health := dh.Health()
	if len(health) != 1 {
		t.Fatalf("ожидали 1 запись, получили %d", len(health))
	}

	key := "web-api:api.svc:8080"
	val, ok := health[key]
	if !ok {
		t.Fatalf("ключ %q не найден в Health()", key)
	}
	if !val {
		t.Errorf("ожидали healthy=true")
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
		t.Fatalf("ошибка: %v", err)
	}

	got := dh.scheduler.deps[0].dep.Config.Interval
	if got != 30*time.Second {
		t.Errorf("ожидали интервал 30s, получили %v", got)
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
		t.Fatalf("ошибка: %v", err)
	}

	got := dh.scheduler.deps[0].dep.Config.Interval
	if got != 10*time.Second {
		t.Errorf("per-dep интервал должен перекрыть глобальный: ожидали 10s, получили %v", got)
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
		t.Fatalf("ошибка: %v", err)
	}

	got := dh.scheduler.deps[0].dep.Config.Timeout
	if got != 3*time.Second {
		t.Errorf("ожидали таймаут 3s, получили %v", got)
	}
}

func TestNew_HTTPSAutoTLS(t *testing.T) {
	reg := prometheus.NewRegistry()

	// Специальный мок, который проверяет что TLS включен.
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
		t.Fatalf("ошибка: %v", err)
	}

	if gotConfig == nil || gotConfig.HTTPTLS == nil || !*gotConfig.HTTPTLS {
		t.Error("ожидали автоматическое включение TLS для https:// URL")
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
		t.Fatalf("ошибка: %v", err)
	}

	if len(dh.scheduler.deps) != 1 {
		t.Fatalf("ожидали 1 зависимость, получили %d", len(dh.scheduler.deps))
	}
	if dh.scheduler.deps[0].dep.Name != "custom-dep" {
		t.Errorf("ожидали имя custom-dep, получили %q", dh.scheduler.deps[0].dep.Name)
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
		t.Fatalf("ошибка: %v", err)
	}

	if len(dh.scheduler.deps) != 2 {
		t.Fatalf("ожидали 2 зависимости, получили %d", len(dh.scheduler.deps))
	}
}

func TestNew_NoFactoryRegistered(t *testing.T) {
	reg := prometheus.NewRegistry()
	// Не регистрируем фабрику — должна быть ошибка.

	_, err := New("test-app",
		WithRegisterer(reg),
		HTTP("web-api", FromURL("http://api.svc:8080"), Critical(false)),
	)
	if err == nil {
		t.Fatal("ожидали ошибку при отсутствии зарегистрированной фабрики")
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
		t.Fatalf("ошибка: %v", err)
	}

	if gotConfig.HTTPHealthPath != "/ready" {
		t.Errorf("ожидали healthPath=/ready, получили %q", gotConfig.HTTPHealthPath)
	}
	if gotConfig.HTTPTLSSkipVerify == nil || !*gotConfig.HTTPTLSSkipVerify {
		t.Error("ожидали TLSSkipVerify=true")
	}
}
