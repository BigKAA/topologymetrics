# Быстрый старт: Go SDK

Руководство по подключению dephealth к Go-сервису за несколько минут.

## Установка

```bash
go get github.com/BigKAA/topologymetrics@latest
```

## Минимальный пример

Подключение одной HTTP-зависимости с экспортом метрик:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/BigKAA/topologymetrics/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    _ "github.com/BigKAA/topologymetrics/dephealth/checks" // регистрация чекеров
)

func main() {
    dh, err := dephealth.New(
        dephealth.HTTP("payment-api",
            dephealth.FromURL("http://payment.svc:8080"),
            dephealth.Critical(true),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := dh.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer dh.Stop()

    http.Handle("/metrics", promhttp.Handler())
    go http.ListenAndServe(":8080", nil)

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    <-sigCh
}
```

После запуска на `/metrics` появятся метрики:

```text
app_dependency_health{dependency="payment-api",type="http",host="payment.svc",port="8080"} 1
app_dependency_latency_seconds_bucket{dependency="payment-api",type="http",host="payment.svc",port="8080",le="0.01"} 42
```

## Несколько зависимостей

```go
dh, err := dephealth.New(
    // Глобальные настройки
    dephealth.WithCheckInterval(30 * time.Second),
    dephealth.WithTimeout(3 * time.Second),

    // PostgreSQL — standalone check (новое соединение)
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),

    // Redis — standalone check
    dephealth.Redis("redis-cache",
        dephealth.FromURL(os.Getenv("REDIS_URL")),
    ),

    // HTTP-сервис
    dephealth.HTTP("auth-service",
        dephealth.FromURL("http://auth.svc:8080"),
        dephealth.WithHTTPHealthPath("/healthz"),
        dephealth.Critical(true),
    ),

    // gRPC-сервис
    dephealth.GRPC("user-service",
        dephealth.FromParams("user.svc", "9090"),
    ),

    // RabbitMQ
    dephealth.AMQP("rabbitmq",
        dephealth.FromParams("rabbitmq.svc", "5672"),
        dephealth.WithAMQPURL("amqp://user:pass@rabbitmq.svc:5672/"),
    ),

    // Kafka
    dephealth.Kafka("kafka",
        dephealth.FromParams("kafka.svc", "9092"),
    ),
)
```

## Интеграция с connection pool (contrib)

Предпочтительный режим: SDK использует существующий connection pool
сервиса вместо создания новых соединений. Это отражает реальную
способность сервиса работать с зависимостью.

### PostgreSQL через `*sql.DB`

```go
import (
    "database/sql"

    "github.com/BigKAA/topologymetrics/dephealth"
    "github.com/BigKAA/topologymetrics/dephealth/contrib/sqldb"
    _ "github.com/BigKAA/topologymetrics/dephealth/checks"
    _ "github.com/jackc/pgx/v5/stdlib"
)

// Используем существующий connection pool
db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

dh, err := dephealth.New(
    sqldb.FromDB("postgres-main", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
)
```

### MySQL через `*sql.DB`

```go
import "github.com/BigKAA/topologymetrics/dephealth/contrib/sqldb"

db, _ := sql.Open("mysql", "user:pass@tcp(mysql.svc:3306)/mydb")

dh, err := dephealth.New(
    sqldb.FromMySQLDB("mysql-main", db,
        dephealth.FromParams("mysql.svc", "3306"),
        dephealth.Critical(true),
    ),
)
```

### Redis через go-redis `*redis.Client`

```go
import (
    "github.com/BigKAA/topologymetrics/dephealth/contrib/redispool"
    "github.com/redis/go-redis/v9"
)

client := redis.NewClient(&redis.Options{Addr: "redis.svc:6379"})

dh, err := dephealth.New(
    // Host и port извлекаются автоматически из client.Options().Addr
    redispool.FromClient("redis-cache", client,
        dephealth.Critical(true),
    ),
)
```

## Глобальные опции

```go
dh, err := dephealth.New(
    // Интервал проверки (по умолчанию 15s)
    dephealth.WithCheckInterval(30 * time.Second),

    // Таймаут каждой проверки (по умолчанию 5s)
    dephealth.WithTimeout(3 * time.Second),

    // Кастомный Prometheus Registerer
    dephealth.WithRegisterer(customRegisterer),

    // Логгер (slog)
    dephealth.WithLogger(slog.Default()),

    // ...зависимости
)
```

## Опции зависимостей

Каждая зависимость может переопределить глобальные настройки:

```go
dephealth.HTTP("slow-service",
    dephealth.FromURL("http://slow.svc:8080"),
    dephealth.CheckInterval(60 * time.Second),  // свой интервал
    dephealth.Timeout(10 * time.Second),         // свой таймаут
    dephealth.Critical(true),                    // критическая зависимость
    dephealth.WithHTTPHealthPath("/ready"),       // путь health check
    dephealth.WithHTTPTLS(true),                  // TLS
    dephealth.WithHTTPTLSSkipVerify(true),        // пропустить проверку сертификата
)
```

## Проверка состояния зависимостей

Метод `Health()` возвращает текущее состояние всех endpoint-ов:

```go
health := dh.Health()
// map[string]bool{
//   "postgres-main:pg.svc:5432":   true,
//   "redis-cache:redis.svc:6379":  true,
//   "auth-service:auth.svc:8080":  false,
// }

// Использование для Kubernetes readiness probe
allHealthy := true
for _, ok := range health {
    if !ok {
        allHealthy = false
        break
    }
}
```

## Экспорт метрик

dephealth экспортирует две метрики Prometheus:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = доступен, `0` = недоступен |
| `app_dependency_latency_seconds` | Histogram | Латентность проверки (секунды) |

Метки: `dependency`, `type`, `host`, `port`.

Для экспорта используйте стандартный `promhttp.Handler()`:

```go
http.Handle("/metrics", promhttp.Handler())
```

## Поддерживаемые типы зависимостей

| Функция | Тип | Метод проверки |
| --- | --- | --- |
| `dephealth.HTTP()` | `http` | HTTP GET к health endpoint, ожидание 2xx |
| `dephealth.GRPC()` | `grpc` | gRPC Health Check Protocol |
| `dephealth.TCP()` | `tcp` | Установка TCP-соединения |
| `dephealth.Postgres()` | `postgres` | `SELECT 1` |
| `dephealth.MySQL()` | `mysql` | `SELECT 1` |
| `dephealth.Redis()` | `redis` | Команда `PING` |
| `dephealth.AMQP()` | `amqp` | Проверка соединения с брокером |
| `dephealth.Kafka()` | `kafka` | Metadata request к брокеру |

## Параметры по умолчанию

| Параметр | Значение | Описание |
| --- | --- | --- |
| `checkInterval` | 15s | Интервал между проверками |
| `timeout` | 5s | Таймаут одной проверки |
| `failureThreshold` | 1 | Число неудач до перехода в unhealthy |
| `successThreshold` | 1 | Число успехов до перехода в healthy |

## Следующие шаги

- [Руководство по интеграции](../migration/go.md) — пошаговое подключение
  к существующему сервису
- [Обзор спецификации](../specification.md) — детали контрактов метрик и поведения
