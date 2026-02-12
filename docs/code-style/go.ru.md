*[English version](go.md)*

# Code Style Guide: Go SDK

Этот документ описывает соглашения по стилю кода для Go SDK (`sdk-go/`).
См. также: [Общие принципы](overview.ru.md) | [Тестирование](testing.ru.md)

## Соглашения об именовании

### Пакеты

- Короткие, строчные, однословные имена
- Без подчёркиваний и mixedCaps
- Имя пакета не должно повторять путь импорта

```go
package dephealth  // хорошо
package checks     // хорошо

package dep_health  // плохо — без подчёркиваний
package depHealth   // плохо — без mixedCaps
```

### Экспортируемые и неэкспортируемые

- Экспортируемые (public): `PascalCase` — видимы за пределами пакета
- Неэкспортируемые (private): `camelCase` — внутренние для пакета

```go
// Экспортируемые
type HealthChecker interface { }
type Dependency struct { }
func New(serviceName string, opts ...Option) (*DepHealth, error) { }

// Неэкспортируемые
type checkResult struct { }
func sanitizeURL(raw string) string { }
```

### Акронимы

Акронимы — полностью заглавные для экспортируемых, строчные для неэкспортируемых:

```go
type HTTPChecker struct { }    // не HttpChecker
type GRPCChecker struct { }    // не GrpcChecker
type TCPChecker struct { }     // не TcpChecker

func parseURL(raw string) { }  // не parseUrl
var httpClient *http.Client     // не hTTPClient
```

### Интерфейсы

- Интерфейсы с одним методом: имя метода + суффикс `er`
- Многометодные интерфейсы: описательное существительное

```go
// Один метод — суффикс "er"
type HealthChecker interface {
    Check(ctx context.Context, endpoint Endpoint) error
    Type() string
}

// Избегайте абстрактных имён
type Doer interface { }  // плохо — слишком размыто
```

## Структура пакетов

```text
sdk-go/
├── dephealth/
│   ├── dephealth.go         // New(), DependencyHealth, Start/Stop
│   ├── options.go           // Option type, functional options
│   ├── dependency.go        // Dependency, Endpoint structs
│   ├── checker.go           // HealthChecker interface, sentinel errors
│   ├── scheduler.go         // планировщик проверок (goroutines)
│   ├── parser.go            // парсер URL/параметров
│   ├── metrics.go           // Prometheus gauges, histograms
│   ├── checks/
│   │   ├── factories.go     // реестр checker-ов, init()
│   │   ├── http.go          // HTTPChecker
│   │   ├── grpc.go          // GRPCChecker
│   │   ├── tcp.go           // TCPChecker
│   │   ├── postgres.go      // PostgresChecker
│   │   ├── redis.go         // RedisChecker
│   │   ├── amqp.go          // AMQPChecker
│   │   └── kafka.go         // KafkaChecker
│   └── contrib/             // опциональные интеграции
│       └── sqldb/           // интеграция с database/sql
```

## Обработка ошибок

### Sentinel errors

Определяйте ошибки уровня пакета для типичных режимов отказа:

```go
var (
    ErrTimeout           = errors.New("health check timeout")
    ErrConnectionRefused = errors.New("connection refused")
    ErrUnhealthy         = errors.New("dependency unhealthy")
)
```

### Оборачивание ошибок

Всегда оборачивайте ошибки с контекстом через `fmt.Errorf` и `%w`:

```go
// Хорошо — оборачивает с контекстом, сохраняет sentinel
func (c *HTTPChecker) Check(ctx context.Context, ep Endpoint) error {
    resp, err := c.client.Get(url)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return fmt.Errorf("HTTP check %s:%d: %w", ep.Host, ep.Port, ErrTimeout)
        }
        return fmt.Errorf("HTTP check %s:%d: %w", ep.Host, ep.Port, err)
    }
    if resp.StatusCode >= 300 {
        return fmt.Errorf("HTTP check %s:%d: status %d: %w",
            ep.Host, ep.Port, resp.StatusCode, ErrUnhealthy)
    }
    return nil
}

// Плохо — теряет контекст
return err
return errors.New("failed")
```

### Правила

- **Нет `panic`** в библиотечном коде — всегда возвращайте ошибки
- Проверяйте ошибки через `errors.Is()` и `errors.As()`, никогда не сравнивайте строки
- Оборачивайте все ошибки через `%w` для сохранения цепочки
- Сообщения ошибок начинаются со строчной буквы, без точки в конце

## GoDoc

Комментарии следуют соглашению Go: начинаются с имени символа.

```go
// HealthChecker is the interface for dependency health checks.
// Each dependency type (HTTP, gRPC, TCP, Postgres, etc.) implements this interface.
type HealthChecker interface {
    // Check performs a health check against the given endpoint.
    // Returns nil if the endpoint is healthy, or an error describing the failure.
    // The context carries the timeout deadline.
    Check(ctx context.Context, endpoint Endpoint) error

    // Type returns the dependency type this checker handles (e.g. "http", "postgres").
    Type() string
}

// New creates a new DepHealth instance with the given service name and options.
// Returns an error if the configuration is invalid (e.g., empty service name).
func New(serviceName string, opts ...Option) (*DepHealth, error) { }
```

Правила:

- Первое предложение: начинается с имени символа (соглашение GoDoc)
- Комментарии на английском
- Полные предложения с точкой
- Документируйте все экспортируемые символы
- Документируйте неочевидное поведение и гарантии параллелизма

## context.Context

`context.Context` всегда **первый параметр** в функциях, которые его принимают:

```go
// Хорошо
func (c *HTTPChecker) Check(ctx context.Context, endpoint Endpoint) error { }
func (s *Scheduler) Start(ctx context.Context) error { }

// Плохо
func (c *HTTPChecker) Check(endpoint Endpoint, ctx context.Context) error { }
```

Правила:

- Никогда не сохраняйте `ctx` в структуре — передавайте по цепочке вызовов
- Используйте `ctx` для отмены и таймаутов, не для передачи значений
- Уважайте отмену контекста: проверяйте `ctx.Err()` в циклах

## Functional Options

Используйте паттерн functional options для конфигурации:

```go
// Option configures DepHealth.
type Option func(*options)

type options struct {
    checkInterval time.Duration
    timeout       time.Duration
    registry      prometheus.Registerer
}

// WithCheckInterval sets the health check interval. Default: 15s.
func WithCheckInterval(d time.Duration) Option {
    return func(o *options) { o.checkInterval = d }
}

// WithTimeout sets the health check timeout. Default: 5s.
func WithTimeout(d time.Duration) Option {
    return func(o *options) { o.timeout = d }
}
```

Использование:

```go
dh, err := dephealth.New("order-service",
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
    dephealth.WithCheckInterval(30 * time.Second),
    dephealth.WithTimeout(10 * time.Second),
)
```

## defer для очистки ресурсов

Используйте `defer` для освобождения ресурсов. Размещайте сразу после успешного получения ресурса:

```go
func (c *PostgresChecker) Check(ctx context.Context, ep Endpoint) error {
    conn, err := c.pool.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("postgres check %s:%d: %w", ep.Host, ep.Port, err)
    }
    defer conn.Release()

    _, err = conn.Exec(ctx, "SELECT 1")
    if err != nil {
        return fmt.Errorf("postgres check %s:%d: %w", ep.Host, ep.Port, err)
    }
    return nil
}
```

Правила:

- `defer` сразу после успешного получения ресурса
- Помните порядок выполнения `defer` (LIFO) при использовании нескольких defer
- Учитывайте, что `defer` в цикле откладывает до выхода из функции, а не из итерации

## Линтер

### golangci-lint v2

Конфигурация: `sdk-go/.golangci.yml`

Основные включённые линтеры:

- `errcheck` — проверка обработки ошибок
- `govet` — подозрительные конструкции
- `staticcheck` — продвинутый статический анализ
- `revive` — стиль и именование
- `goimports` — форматирование импортов
- `misspell` — опечатки в комментариях
- `gosec` — проблемы безопасности

### Запуск

```bash
cd sdk-go && make lint    # golangci-lint в Docker
cd sdk-go && make fmt     # goimports + gofmt
```

## Дополнительные соглашения

- **Версия Go**: 1.25+ — используйте `log/slog`, range-over-func где уместно
- **Путь модуля**: `github.com/BigKAA/topologymetrics/sdk-go`
- **Нулевые значения**: проектируйте типы так, чтобы нулевые значения были полезны (или документируйте, когда это не так)
- **Порядок полей struct**: экспортируемые первыми, затем неэкспортируемые; группируйте логически
- **Нет `init()` в библиотечном коде** кроме регистрации checker-ов в `checks/factories.go`
- **Table-driven tests**: предпочтительный стиль (см. [Тестирование](testing.ru.md))
- **Строки ошибок**: строчные, без точки в конце, без префикса «failed to»
- **Имена receiver-ов**: короткие (1-2 буквы), единообразные для методов одного типа

```go
// Хорошо — короткий, единообразный receiver
func (s *Scheduler) Start(ctx context.Context) error { }
func (s *Scheduler) Stop() { }
func (s *Scheduler) addDependency(d Dependency) { }

// Плохо — длинный, непоследовательный
func (scheduler *Scheduler) Start(ctx context.Context) error { }
func (sched *Scheduler) Stop() { }
```
