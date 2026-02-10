[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# Integration Guide: dephealth in an Existing Go Service

Step-by-step instructions for adding dependency monitoring
to a running microservice.

## Migration from v0.2 to v0.3

### Breaking change: new module path

In v0.3.0, the module path has changed from `github.com/BigKAA/topologymetrics`
to `github.com/BigKAA/topologymetrics/sdk-go`.

This fixes `go get` functionality — the standard approach for Go modules
in monorepos where `go.mod` is located in a subdirectory.

### Migration steps

1. Update the dependency:

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

1. Replace import paths in all files:

```bash
# Bulk replacement (Linux/macOS)
find . -name '*.go' -exec sed -i '' \
  's|github.com/BigKAA/topologymetrics/dephealth|github.com/BigKAA/topologymetrics/sdk-go/dephealth|g' {} +
```

1. Update `go.mod` — remove the old dependency:

```bash
go mod tidy
```

### Import replacement examples

```go
// v0.2
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
)
```

The API and SDK behavior remain unchanged — only the module path has changed.

---

## Migration from v0.1 to v0.2

### API Changes

| v0.1 | v0.2 | Description |
| --- | --- | --- |
| `dephealth.New(...)` | `dephealth.New("my-service", ...)` | Required first argument `name` |
| `dephealth.Critical(true)` (optional) | `dephealth.Critical(true/false)` (required) | For each dependency |
| `Endpoint.Metadata` | `Endpoint.Labels` | Field renamed |
| `dephealth.WithMetadata(map)` | `dephealth.WithLabel("key", "value")` | Custom labels |
| `WithOptionalLabels(...)` | removed | Custom labels via `WithLabel` |

### Required Changes

1. Add `name` as the first argument to `dephealth.New()`:

```go
// v0.1
dh, err := dephealth.New(
    dephealth.Postgres("postgres-main", ...),
)

// v0.2
dh, err := dephealth.New("my-service",
    dephealth.Postgres("postgres-main", ...),
)
```

1. Specify `Critical()` for each dependency:

```go
// v0.1 — Critical is optional
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
)

// v0.2 — Critical is required
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
    dephealth.Critical(false),
)
```

1. Replace `WithMetadata` with `WithLabel` (if used):

```go
// v0.1
dephealth.WithMetadata(map[string]string{"role": "primary"})

// v0.2
dephealth.WithLabel("role", "primary")
```

### New metric labels

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Update PromQL queries and Grafana dashboards to include the `name` and `critical` labels.

## Prerequisites

- Go 1.21+
- Service already exports Prometheus metrics via `promhttp.Handler()`
- Access to dependencies (databases, caches, other services) from the service

## Step 1. Install Dependencies

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

## Step 2. Import Packages

Add imports to your service initialization file:

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"

    // Register built-in checkers — required blank import
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
)
```

If using connection pool integration (recommended):

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"     // for *sql.DB
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"  // for *redis.Client
)
```

## Step 3. Create DepHealth Instance

### Option A: Standalone mode (simple)

The SDK creates temporary connections for health checks. Suitable for HTTP/gRPC
services and situations where connection pools are unavailable.

```go
func initDepHealth() (*dephealth.DepHealth, error) {
    return dephealth.New("my-service",
        dephealth.Postgres("postgres-main",
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),
        dephealth.Redis("redis-cache",
            dephealth.FromURL(os.Getenv("REDIS_URL")),
            dephealth.Critical(false),
        ),
        dephealth.HTTP("payment-api",
            dephealth.FromURL(os.Getenv("PAYMENT_SERVICE_URL")),
            dephealth.Critical(true),
        ),
    )
}
```

### Option B: Connection pool integration (recommended)

The SDK uses the service's existing connections. Benefits:

- Reflects the service's actual ability to work with dependencies
- Does not create additional load on databases/caches
- Detects pool-related issues (exhaustion, leaks)

```go
func initDepHealth(db *sql.DB, rdb *redis.Client) (*dephealth.DepHealth, error) {
    return dephealth.New("my-service",
        dephealth.WithCheckInterval(15 * time.Second),
        dephealth.WithLogger(slog.Default()),

        // PostgreSQL via existing *sql.DB
        sqldb.FromDB("postgres-main", db,
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),

        // Redis via existing *redis.Client
        // Host:port extracted automatically
        redispool.FromClient("redis-cache", rdb,
            dephealth.Critical(false),
        ),

        // For HTTP/gRPC — standalone only
        dephealth.HTTP("payment-api",
            dephealth.FromURL(os.Getenv("PAYMENT_SERVICE_URL")),
            dephealth.Critical(true),
        ),

        dephealth.GRPC("auth-service",
            dephealth.FromParams(os.Getenv("AUTH_HOST"), os.Getenv("AUTH_PORT")),
            dephealth.Critical(true),
        ),
    )
}
```

## Step 4. Start and Stop

Integrate `dh.Start()` and `dh.Stop()` into your service lifecycle:

```go
func main() {
    // ... initialize DB, Redis, etc. ...

    dh, err := initDepHealth(db, rdb)
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := dh.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // ... start HTTP server ...

    // Graceful shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    <-sigCh

    dh.Stop() // stop checks before server shutdown
    server.Shutdown(context.Background())
}
```

## Step 5. Dependency Status Endpoint (optional)

Add an endpoint for Kubernetes readiness probe or debugging:

```go
func handleDependencies(dh *dephealth.DepHealth) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        health := dh.Health()

        w.Header().Set("Content-Type", "application/json")

        // If there are unhealthy dependencies — return 503
        for _, ok := range health {
            if !ok {
                w.WriteHeader(http.StatusServiceUnavailable)
                json.NewEncoder(w).Encode(health)
                return
            }
        }

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(health)
    }
}

// Register
mux.HandleFunc("/health/dependencies", handleDependencies(dh))
```

## Typical Configurations

### Web service with PostgreSQL and Redis

```go
dh, _ := dephealth.New("my-service",
    sqldb.FromDB("postgres", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
    redispool.FromClient("redis", rdb,
        dephealth.Critical(false),
    ),
)
```

### API Gateway with upstream services

```go
dh, _ := dephealth.New("api-gateway",
    dephealth.WithCheckInterval(10 * time.Second),

    dephealth.HTTP("user-service",
        dephealth.FromURL("http://user-svc:8080"),
        dephealth.WithHTTPHealthPath("/healthz"),
        dephealth.Critical(true),
    ),
    dephealth.HTTP("order-service",
        dephealth.FromURL("http://order-svc:8080"),
        dephealth.Critical(true),
    ),
    dephealth.GRPC("auth-service",
        dephealth.FromParams("auth-svc", "9090"),
        dephealth.Critical(true),
    ),
)
```

### Event processor with Kafka and RabbitMQ

```go
dh, _ := dephealth.New("event-processor",
    dephealth.Kafka("kafka-main",
        dephealth.FromParams("kafka.svc", "9092"),
        dephealth.Critical(true),
    ),
    dephealth.AMQP("rabbitmq",
        dephealth.FromParams("rabbitmq.svc", "5672"),
        dephealth.WithAMQPURL("amqp://user:pass@rabbitmq.svc:5672/"),
        dephealth.Critical(true),
    ),
    sqldb.FromDB("postgres", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(false),
    ),
)
```

### Service with TLS dependencies

```go
dh, _ := dephealth.New("my-service",
    dephealth.HTTP("external-api",
        dephealth.FromURL("https://api.example.com"),
        dephealth.WithHTTPHealthPath("/status"),
        dephealth.Timeout(10 * time.Second),
        dephealth.Critical(true),
        // TLS enabled automatically for https://
    ),
    dephealth.GRPC("secure-service",
        dephealth.FromParams("secure.svc", "443"),
        dephealth.WithGRPCTLS(true),
        dephealth.WithGRPCTLSSkipVerify(true), // for self-signed certificates
        dephealth.Critical(false),
    ),
)
```

## Troubleshooting

### `no checker factory registered for type "..."`

**Cause**: the `checks` package is not imported.

**Solution**: add blank import:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

### Metrics do not appear on `/metrics`

**Check:**

1. `dh.Start(ctx)` called without errors
2. `promhttp.Handler()` registered on the `/metrics` route
3. Enough time has passed for the first check (default `initialDelay` = 0
   in the public API, first check runs immediately)

### All dependencies show `0` (unhealthy)

**Check:**

1. Network accessibility of dependencies from the service container/pod
2. DNS resolution of service names
3. Correct URL/host/port in configuration
4. Timeout (default `5s`) — is it sufficient for this dependency
5. Logs: `dephealth.WithLogger(slog.Default())` will show error causes

### High latency in PostgreSQL/MySQL checks

**Cause**: standalone mode creates a new connection each time.

**Solution**: use the contrib module `sqldb.FromDB()` with an existing
connection pool. This eliminates the connection setup overhead.

### gRPC: error `context deadline exceeded`

**Check:**

1. gRPC service is accessible at the specified address
2. Service implements `grpc.health.v1.Health/Check`
3. For gRPC use `FromParams(host, port)`, not `FromURL()` —
   the URL parser may incorrectly handle bare `host:port`
4. If TLS is needed: `dephealth.WithGRPCTLS(true)`

### AMQP: connection error to RabbitMQ

**For AMQP, you must provide the full URL via `WithAMQPURL()`:**

```go
dephealth.AMQP("rabbitmq",
    dephealth.FromParams("rabbitmq.svc", "5672"),   // for metric labels
    dephealth.WithAMQPURL("amqp://user:pass@rabbitmq.svc:5672/vhost"),
    dephealth.Critical(false),
)
```

`FromParams` defines the `host`/`port` metric labels, while `WithAMQPURL` defines
the connection string with credentials.

### Dependency Naming

Names must conform to the following rules:

- Length: 1-63 characters
- Format: `[a-z][a-z0-9-]*` (lowercase letters, digits, hyphens)
- Starts with a letter
- Examples: `postgres-main`, `redis-cache`, `auth-service`

## Next Steps

- [Quick Start](../quickstart/go.md) — minimal examples
- [Specification Overview](../specification.md) — details on metrics contracts and behavior

---

<a id="russian"></a>

# Руководство по интеграции dephealth в существующий Go-сервис

Пошаговая инструкция по добавлению мониторинга зависимостей
в работающий микросервис.

## Миграция с v0.2 на v0.3

### Breaking change: новый module path

В v0.3.0 module path изменён с `github.com/BigKAA/topologymetrics`
на `github.com/BigKAA/topologymetrics/sdk-go`.

Это исправляет работу `go get` — стандартный подход для Go-модулей
в монорепозиториях, где `go.mod` находится в поддиректории.

### Шаги миграции

1. Обновите зависимость:

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

1. Замените import paths во всех файлах:

```bash
# Массовая замена (Linux/macOS)
find . -name '*.go' -exec sed -i '' \
  's|github.com/BigKAA/topologymetrics/dephealth|github.com/BigKAA/topologymetrics/sdk-go/dephealth|g' {} +
```

1. Обновите `go.mod` — удалите старую зависимость:

```bash
go mod tidy
```

### Примеры замены импортов

```go
// v0.2
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
)
```

API и поведение SDK не изменились — только module path.

---

## Миграция с v0.1 на v0.2

### Изменения API

| v0.1 | v0.2 | Описание |
| --- | --- | --- |
| `dephealth.New(...)` | `dephealth.New("my-service", ...)` | Обязательный первый аргумент `name` |
| `dephealth.Critical(true)` (необязателен) | `dephealth.Critical(true/false)` (обязателен) | Для каждой зависимости |
| `Endpoint.Metadata` | `Endpoint.Labels` | Переименование поля |
| `dephealth.WithMetadata(map)` | `dephealth.WithLabel("key", "value")` | Произвольные метки |
| `WithOptionalLabels(...)` | удалён | Произвольные метки через `WithLabel` |

### Обязательные изменения

1. Добавьте `name` первым аргументом в `dephealth.New()`:

```go
// v0.1
dh, err := dephealth.New(
    dephealth.Postgres("postgres-main", ...),
)

// v0.2
dh, err := dephealth.New("my-service",
    dephealth.Postgres("postgres-main", ...),
)
```

1. Укажите `Critical()` для каждой зависимости:

```go
// v0.1 — Critical необязателен
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
)

// v0.2 — Critical обязателен
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
    dephealth.Critical(false),
)
```

1. Замените `WithMetadata` на `WithLabel` (если используется):

```go
// v0.1
dephealth.WithMetadata(map[string]string{"role": "primary"})

// v0.2
dephealth.WithLabel("role", "primary")
```

### Новые метки в метриках

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Обновите PromQL-запросы и дашборды Grafana, добавив метки `name` и `critical`.

## Предварительные требования

- Go 1.21+
- Сервис уже экспортирует Prometheus-метрики через `promhttp.Handler()`
- Доступ к зависимостям (БД, кэш, другие сервисы) из сервиса

## Шаг 1. Установка зависимостей

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

## Шаг 2. Импорт пакетов

Добавьте импорты в файл с инициализацией сервиса:

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"

    // Регистрация встроенных чекеров — обязательный blank import
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
)
```

Если используете connection pool (рекомендуется):

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"     // для *sql.DB
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"  // для *redis.Client
)
```

## Шаг 3. Создание экземпляра DepHealth

### Вариант A: Standalone-режим (простой)

SDK создаёт временные соединения для проверок. Подходит для HTTP/gRPC
сервисов и ситуаций, когда connection pool недоступен.

```go
func initDepHealth() (*dephealth.DepHealth, error) {
    return dephealth.New("my-service",
        dephealth.Postgres("postgres-main",
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),
        dephealth.Redis("redis-cache",
            dephealth.FromURL(os.Getenv("REDIS_URL")),
            dephealth.Critical(false),
        ),
        dephealth.HTTP("payment-api",
            dephealth.FromURL(os.Getenv("PAYMENT_SERVICE_URL")),
            dephealth.Critical(true),
        ),
    )
}
```

### Вариант B: Интеграция с connection pool (рекомендуется)

SDK использует существующие подключения сервиса. Преимущества:

- Отражает реальную способность сервиса работать с зависимостью
- Не создаёт дополнительную нагрузку на БД/кэш
- Обнаруживает проблемы с пулом (исчерпание, утечки)

```go
func initDepHealth(db *sql.DB, rdb *redis.Client) (*dephealth.DepHealth, error) {
    return dephealth.New("my-service",
        dephealth.WithCheckInterval(15 * time.Second),
        dephealth.WithLogger(slog.Default()),

        // PostgreSQL через существующий *sql.DB
        sqldb.FromDB("postgres-main", db,
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),

        // Redis через существующий *redis.Client
        // Host:port извлекаются автоматически
        redispool.FromClient("redis-cache", rdb,
            dephealth.Critical(false),
        ),

        // Для HTTP/gRPC — только standalone
        dephealth.HTTP("payment-api",
            dephealth.FromURL(os.Getenv("PAYMENT_SERVICE_URL")),
            dephealth.Critical(true),
        ),

        dephealth.GRPC("auth-service",
            dephealth.FromParams(os.Getenv("AUTH_HOST"), os.Getenv("AUTH_PORT")),
            dephealth.Critical(true),
        ),
    )
}
```

## Шаг 4. Запуск и остановка

Встройте `dh.Start()` и `dh.Stop()` в жизненный цикл сервиса:

```go
func main() {
    // ... инициализация DB, Redis и т.д. ...

    dh, err := initDepHealth(db, rdb)
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := dh.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // ... запуск HTTP-сервера ...

    // Graceful shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    <-sigCh

    dh.Stop() // остановить проверки перед остановкой сервера
    server.Shutdown(context.Background())
}
```

## Шаг 5. Endpoint для состояния зависимостей (опционально)

Добавьте endpoint для Kubernetes readiness probe или отладки:

```go
func handleDependencies(dh *dephealth.DepHealth) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        health := dh.Health()

        w.Header().Set("Content-Type", "application/json")

        // Если есть unhealthy зависимости — 503
        for _, ok := range health {
            if !ok {
                w.WriteHeader(http.StatusServiceUnavailable)
                json.NewEncoder(w).Encode(health)
                return
            }
        }

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(health)
    }
}

// Регистрация
mux.HandleFunc("/health/dependencies", handleDependencies(dh))
```

## Типичные конфигурации

### Веб-сервис с PostgreSQL и Redis

```go
dh, _ := dephealth.New("my-service",
    sqldb.FromDB("postgres", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
    redispool.FromClient("redis", rdb,
        dephealth.Critical(false),
    ),
)
```

### API Gateway с upstream-сервисами

```go
dh, _ := dephealth.New("api-gateway",
    dephealth.WithCheckInterval(10 * time.Second),

    dephealth.HTTP("user-service",
        dephealth.FromURL("http://user-svc:8080"),
        dephealth.WithHTTPHealthPath("/healthz"),
        dephealth.Critical(true),
    ),
    dephealth.HTTP("order-service",
        dephealth.FromURL("http://order-svc:8080"),
        dephealth.Critical(true),
    ),
    dephealth.GRPC("auth-service",
        dephealth.FromParams("auth-svc", "9090"),
        dephealth.Critical(true),
    ),
)
```

### Обработчик событий с Kafka и RabbitMQ

```go
dh, _ := dephealth.New("event-processor",
    dephealth.Kafka("kafka-main",
        dephealth.FromParams("kafka.svc", "9092"),
        dephealth.Critical(true),
    ),
    dephealth.AMQP("rabbitmq",
        dephealth.FromParams("rabbitmq.svc", "5672"),
        dephealth.WithAMQPURL("amqp://user:pass@rabbitmq.svc:5672/"),
        dephealth.Critical(true),
    ),
    sqldb.FromDB("postgres", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(false),
    ),
)
```

### Сервис с TLS-зависимостями

```go
dh, _ := dephealth.New("my-service",
    dephealth.HTTP("external-api",
        dephealth.FromURL("https://api.example.com"),
        dephealth.WithHTTPHealthPath("/status"),
        dephealth.Timeout(10 * time.Second),
        dephealth.Critical(true),
        // TLS включается автоматически для https://
    ),
    dephealth.GRPC("secure-service",
        dephealth.FromParams("secure.svc", "443"),
        dephealth.WithGRPCTLS(true),
        dephealth.WithGRPCTLSSkipVerify(true), // для self-signed сертификатов
        dephealth.Critical(false),
    ),
)
```

## Troubleshooting

### `no checker factory registered for type "..."

**Причина**: не импортирован пакет `checks`.

**Решение**: добавьте blank import:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

### Метрики не появляются на `/metrics`

**Проверьте:**

1. `dh.Start(ctx)` вызван без ошибки
2. `promhttp.Handler()` зарегистрирован на маршруте `/metrics`
3. Прошло достаточно времени для первой проверки (по умолчанию `initialDelay` = 0
   в публичном API, первая проверка запускается сразу)

### Все зависимости показывают `0` (unhealthy)

**Проверьте:**

1. Сетевая доступность зависимостей из контейнера/пода сервиса
2. DNS-резолвинг имён сервисов
3. Правильность URL/host/port в конфигурации
4. Таймаут (`5s` по умолчанию) — достаточен ли для данной зависимости
5. Логи: `dephealth.WithLogger(slog.Default())` покажет причины ошибок

### Высокая латентность проверок PostgreSQL/MySQL

**Причина**: standalone-режим создаёт новое соединение каждый раз.

**Решение**: используйте contrib-модуль `sqldb.FromDB()` с существующим
connection pool. Это исключает overhead на установку соединения.

### gRPC: ошибка `context deadline exceeded`

**Проверьте:**

1. gRPC-сервис доступен по указанному адресу
2. Сервис реализует `grpc.health.v1.Health/Check`
3. Для gRPC используйте `FromParams(host, port)`, а не `FromURL()` —
   URL-парсер может некорректно обработать bare `host:port`
4. Если нужен TLS: `dephealth.WithGRPCTLS(true)`

### AMQP: ошибка подключения к RabbitMQ

**Для AMQP необходимо передать полный URL через `WithAMQPURL()`:**

```go
dephealth.AMQP("rabbitmq",
    dephealth.FromParams("rabbitmq.svc", "5672"),   // для меток метрик
    dephealth.WithAMQPURL("amqp://user:pass@rabbitmq.svc:5672/vhost"),
    dephealth.Critical(false),
)
```

`FromParams` определяет метки `host`/`port` метрик, а `WithAMQPURL` — строку
подключения с credentials.

### Именование зависимостей

Имена должны соответствовать правилам:

- Длина: 1-63 символа
- Формат: `[a-z][a-z0-9-]*` (строчные буквы, цифры, дефисы)
- Начинается с буквы
- Примеры: `postgres-main`, `redis-cache`, `auth-service`

## Следующие шаги

- [Быстрый старт](../quickstart/go.md) — минимальные примеры
- [Обзор спецификации](../specification.md) — детали контрактов метрик и поведения
