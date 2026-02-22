*[English version](custom-checkers.md)*

# Кастомные чекеры

Вы можете реализовать собственный чекер для любого типа зависимости,
не покрытого встроенными чекерами. Руководство охватывает интерфейс
`HealthChecker`, классификацию ошибок и регистрацию.

## Интерфейс HealthChecker

```go
type HealthChecker interface {
    Check(ctx context.Context, endpoint Endpoint) error
    Type() string
}
```

- `Check()` — выполняет проверку состояния для данного эндпоинта.
  Возвращает `nil` если здоров, или ошибку с описанием сбоя. Context
  содержит дедлайн таймаута.
- `Type()` — возвращает строку типа зависимости (напр., `"elasticsearch"`).

## Базовый пример: Elasticsearch-чекер

```go
package escheck

import (
    "context"
    "fmt"
    "net/http"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

// Checker реализует dephealth.HealthChecker для Elasticsearch.
type Checker struct{}

func New() *Checker {
    return &Checker{}
}

func (c *Checker) Type() string {
    return "elasticsearch"
}

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
    url := fmt.Sprintf("http://%s:%s/_cluster/health", endpoint.Host, endpoint.Port)

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("elasticsearch health check: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("elasticsearch returned status %d", resp.StatusCode)
    }

    return nil
}
```

### Регистрация кастомного чекера

Используйте `AddDependency()` для регистрации:

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "myapp/escheck"
)

func main() {
    esChecker := escheck.New()

    dh, err := dephealth.New("my-service", "my-team",
        dephealth.AddDependency("elasticsearch", "elasticsearch", esChecker,
            dephealth.FromParams("es.svc", "9200"),
            dephealth.Critical(true),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    if err := dh.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer dh.Stop()

    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Второй аргумент `AddDependency()` — строка типа зависимости
(появляется как метка `type` в метриках). Можно использовать любую строку.

## Классификация ошибок

По умолчанию ошибки из `Check()` классифицируются ядром (таймауты,
DNS-ошибки, TLS-ошибки, отказ соединения). Для протокол-специфичной
классификации реализуйте интерфейс `ClassifiedError` или возвращайте
`ClassifiedCheckError`.

### Интерфейс ClassifiedError

```go
type ClassifiedError interface {
    error
    StatusCategory() dephealth.StatusCategory
    StatusDetail() string
}
```

### Структура ClassifiedCheckError

SDK предоставляет готовую реализацию:

```go
type ClassifiedCheckError struct {
    Category dephealth.StatusCategory
    Detail   string
    Cause    error
}
```

Методы:

- `Error() string` — возвращает сообщение причинной ошибки
- `Unwrap() error` — возвращает причину для `errors.Is`/`errors.As`
- `StatusCategory() StatusCategory` — возвращает категорию статуса
- `StatusDetail() string` — возвращает строку детализации

### Доступные категории статусов

```go
const (
    StatusOK              StatusCategory = "ok"
    StatusTimeout         StatusCategory = "timeout"
    StatusConnectionError StatusCategory = "connection_error"
    StatusDNSError        StatusCategory = "dns_error"
    StatusAuthError       StatusCategory = "auth_error"
    StatusTLSError        StatusCategory = "tls_error"
    StatusUnhealthy       StatusCategory = "unhealthy"
    StatusError           StatusCategory = "error"
)
```

### Пример: чекер с классификацией ошибок

```go
func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
    url := fmt.Sprintf("http://%s:%s/_cluster/health", endpoint.Host, endpoint.Port)

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        // Сетевые ошибки будут классифицированы ядром
        return fmt.Errorf("elasticsearch check: %w", err)
    }
    defer resp.Body.Close()

    // Ошибки авторизации
    if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
        return &dephealth.ClassifiedCheckError{
            Category: dephealth.StatusAuthError,
            Detail:   "auth_error",
            Cause:    fmt.Errorf("elasticsearch returned %d", resp.StatusCode),
        }
    }

    // Нездоров, но доступен
    if resp.StatusCode != http.StatusOK {
        return &dephealth.ClassifiedCheckError{
            Category: dephealth.StatusUnhealthy,
            Detail:   fmt.Sprintf("es_%d", resp.StatusCode),
            Cause:    fmt.Errorf("elasticsearch returned %d", resp.StatusCode),
        }
    }

    return nil
}
```

### Приоритет классификации

Классификатор ядра проверяет ошибки в следующем порядке:

1. **Интерфейс ClassifiedError** — высший приоритет (ваша классификация)
2. **Sentinel-ошибки** — `ErrTimeout`, `ErrConnectionRefused`, `ErrUnhealthy`
3. **Платформенные ошибки** — `context.DeadlineExceeded`, `*net.DNSError`,
   `*net.OpError` (ECONNREFUSED), `*tls.CertificateVerificationError`
4. **Fallback** — `StatusError` с деталью `"error"`

## Регистрация фабрики чекера

Если вы хотите использовать кастомный чекер с URL-based API
(`dephealth.New()` с пользовательским типом), зарегистрируйте фабрику:

```go
package escheck

import "github.com/BigKAA/topologymetrics/sdk-go/dephealth"

const TypeElasticsearch dephealth.DependencyType = "elasticsearch"

func init() {
    dephealth.RegisterCheckerFactory(TypeElasticsearch, NewFromConfig)
}

func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
    return New()
}
```

После импорта этого пакета можно создать хелпер-функцию:

```go
func Elasticsearch(name string, opts ...dephealth.DependencyOption) dephealth.Option {
    return dephealth.AddDependency(name, TypeElasticsearch, New(), opts...)
}
```

Использование:

```go
import "myapp/escheck"

dh, err := dephealth.New("my-service", "my-team",
    escheck.Elasticsearch("es-cluster",
        dephealth.FromParams("es.svc", "9200"),
        dephealth.Critical(true),
    ),
)
```

## Тестирование кастомных чекеров

```go
package escheck_test

import (
    "context"
    "testing"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "myapp/escheck"
)

func TestChecker_Type(t *testing.T) {
    c := escheck.New()
    if c.Type() != "elasticsearch" {
        t.Errorf("expected type elasticsearch, got %s", c.Type())
    }
}

func TestChecker_Check_Success(t *testing.T) {
    // Запустить тестовый HTTP-сервер, отвечающий 200
    // ...

    c := escheck.New()
    err := c.Check(context.Background(), dephealth.Endpoint{
        Host: "localhost",
        Port: "9200",
    })
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}

func TestChecker_Check_AuthError(t *testing.T) {
    // Запустить тестовый HTTP-сервер, отвечающий 401
    // ...

    c := escheck.New()
    err := c.Check(context.Background(), dephealth.Endpoint{
        Host: "localhost",
        Port: "9200",
    })

    var ce dephealth.ClassifiedError
    if !errors.As(err, &ce) {
        t.Fatal("expected ClassifiedError")
    }
    if ce.StatusCategory() != dephealth.StatusAuthError {
        t.Errorf("expected auth_error, got %s", ce.StatusCategory())
    }
}
```

## См. также

- [Чекеры](checkers.ru.md) — справочник встроенных чекеров
- [Выборочный импорт](selective-imports.ru.md) — регистрация фабрик через `init()`
- [API Reference](api-reference.ru.md) — `HealthChecker`, `ClassifiedError`, `AddDependency`
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
