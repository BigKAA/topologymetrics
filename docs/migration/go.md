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

2. Замените import paths во всех файлах:

```bash
# Массовая замена (Linux/macOS)
find . -name '*.go' -exec sed -i '' \
  's|github.com/BigKAA/topologymetrics/dephealth|github.com/BigKAA/topologymetrics/sdk-go/dephealth|g' {} +
```

3. Обновите `go.mod` — удалите старую зависимость:

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
