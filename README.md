# Topology Metrics (dephealth)

SDK для мониторинга зависимостей микросервисов. Каждый сервис экспортирует
Prometheus-метрики о состоянии своих зависимостей (БД, кэши, очереди,
HTTP/gRPC-сервисы). VictoriaMetrics собирает данные, Grafana визуализирует.

## Проблема

Система из сотен микросервисов сталкивается с тремя проблемами:

- **Долгий поиск первопричины** — при сбое неясно, какой сервис является источником
- **Нет карты зависимостей** — никто не видит полную картину связей между сервисами
- **Каскадные сбои** — падение одного сервиса вызывает шквал алертов от зависимых

## Решение

Каждый микросервис экспортирует метрики о состоянии своих соединений:

```text
# Здоровье: 1 = доступен, 0 = недоступен
app_dependency_health{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432"} 1

# Латентность проверки
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="0.01"} 42
```

Из метрик автоматически строится граф зависимостей, настраивается алертинг
с подавлением каскадов, отображается степень деградации каждого сервиса.

## Архитектура

Нативная библиотека на каждом языке, объединённая общей спецификацией.
Не sidecar, не FFI — глубокая интеграция с экосистемой каждого языка.

```text
┌─────────────────────────────────────────────┐
│         Framework Integration               │  Spring Boot / ASP.NET / FastAPI
├─────────────────────────────────────────────┤
│         Metrics Exporter                    │  Prometheus gauges + histograms
├─────────────────────────────────────────────┤
│         Check Scheduler                     │  Периодический запуск проверок
├─────────────────────────────────────────────┤
│         Health Checkers                     │  HTTP, gRPC, TCP, Postgres, MySQL,
│                                             │  Redis, AMQP, Kafka
├─────────────────────────────────────────────┤
│         Connection Config Parser            │  URL / params / connection string
├─────────────────────────────────────────────┤
│         Core Abstractions                   │  Dependency, Endpoint, HealthChecker
└─────────────────────────────────────────────┘
```

### Два режима проверки

- **Автономный** — SDK создаёт временное соединение для проверки
- **Интеграция с pool** — SDK использует существующий connection pool сервиса
  (предпочтительный, отражает реальную способность сервиса работать с зависимостью)

## Поддерживаемые типы проверок

| Тип | Метод проверки |
| --- | --- |
| `http` | HTTP GET к `healthPath`, ожидание 2xx |
| `grpc` | gRPC Health Check Protocol (grpc.health.v1) |
| `tcp` | Установка TCP-соединения |
| `postgres` | `SELECT 1` через connection pool или новое соединение |
| `mysql` | `SELECT 1` через connection pool или новое соединение |
| `redis` | Команда `PING` |
| `amqp` | Проверка соединения с брокером |
| `kafka` | Metadata request к брокеру |

## Структура репозитория

```text
spec/                           # Единая спецификация (контракты метрик, поведения, конфигурации)
conformance/                    # Conformance-тесты (Kubernetes, сценарии, runner)
sdk-go/                         # Go SDK (пилотный язык)
  dephealth/                    #   Core: абстракции, парсер, метрики, планировщик, публичный API
  dephealth/checks/             #   Чекеры: TCP, HTTP, gRPC, Postgres, MySQL, Redis, AMQP, Kafka
  dephealth/contrib/sqldb/      #   Contrib: интеграция с *sql.DB (Postgres/MySQL pool)
  dephealth/contrib/redispool/  #   Contrib: интеграция с *redis.Client
plans/                          # Планы разработки
```

Планируемые SDK: Go (пилотный, реализован), Java, C#, Python.

## Go SDK

Пилотный SDK на Go. Публичный API, все 8 чекеров, метрики, планировщик и contrib-модули реализованы.

### Быстрый старт

```go
import (
    "github.com/company/dephealth/dephealth"
    _ "github.com/company/dephealth/dephealth/checks" // регистрация чекеров
)

dh, err := dephealth.New(
    dephealth.HTTP("payment-service",
        dephealth.FromURL(os.Getenv("PAYMENT_SERVICE_URL")),
        dephealth.Critical(true),
    ),
    dephealth.Postgres("postgres-main",
        dephealth.FromParams(os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
        dephealth.Critical(true),
    ),
    dephealth.Redis("redis-cache",
        dephealth.FromURL(os.Getenv("REDIS_URL")),
    ),
)
if err != nil {
    log.Fatal(err)
}

dh.Start(ctx)
defer dh.Stop()

http.Handle("/metrics", promhttp.Handler())
```

### Интеграция с connection pool

```go
import (
    "github.com/company/dephealth/dephealth/contrib/sqldb"
    "github.com/company/dephealth/dephealth/contrib/redispool"
)

dh, err := dephealth.New(
    // PostgreSQL через существующий *sql.DB
    sqldb.FromDB("postgres-main", db,
        dephealth.FromParams("pg.svc", "5432"),
        dephealth.Critical(true),
    ),
    // Redis через существующий *redis.Client (host:port извлекается автоматически)
    redispool.FromClient("redis-cache", redisClient),
)
```

### Глобальные опции

```go
dh, err := dephealth.New(
    dephealth.WithCheckInterval(30 * time.Second),
    dephealth.WithTimeout(3 * time.Second),
    dephealth.WithRegisterer(customRegisterer),
    dephealth.WithLogger(slog.Default()),
    // ...зависимости
)
```

### Реализованные компоненты

- **Public API** (`sdk-go/dephealth/dephealth.go`, `options.go`) — `DepHealth`, `New()`, Option pattern
- **Core** (`sdk-go/dephealth/`) — `Dependency`, `Endpoint`, `CheckConfig`, `HealthChecker` interface
- **Parser** (`sdk-go/dephealth/parser.go`) — парсинг URL, connection string, JDBC, прямых параметров
- **Checkers** (`sdk-go/dephealth/checks/`) — TCP, HTTP, gRPC, PostgreSQL, MySQL, Redis, AMQP, Kafka
- **Metrics** (`sdk-go/dephealth/metrics.go`) — Prometheus Gauge + Histogram, functional options
- **Scheduler** (`sdk-go/dephealth/scheduler.go`) — горутина на endpoint, пороги, graceful shutdown
- **Contrib** (`sdk-go/dephealth/contrib/`) — `sqldb` (PostgreSQL/MySQL pool), `redispool` (go-redis pool)

### Сборка и тесты

```bash
# Unit-тесты (81 тест)
cd sdk-go && go test ./... -short

# Линтер
cd sdk-go && golangci-lint run

# Integration-тесты (требуют Docker/Kubernetes)
cd sdk-go && go test ./... -tags integration
```

## Спецификация

Единый источник правды для всех SDK — каталог `spec/`:

- [`spec/metric-contract.md`](spec/metric-contract.md) — формат метрик, метки, значения
- [`spec/check-behavior.md`](spec/check-behavior.md) — жизненный цикл проверок, пороги, таймауты
- [`spec/config-contract.md`](spec/config-contract.md) — форматы конфигурации соединений

### Ключевые метрики

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | Доступность: `1` / `0` |
| `app_dependency_latency_seconds` | Histogram | Латентность проверки |

Обязательные метки: `dependency`, `type`, `host`, `port`.
Опциональные: `role`, `shard`, `vhost`.

### Параметры по умолчанию

| Параметр | Значение |
| --- | --- |
| `checkInterval` | 15s |
| `timeout` | 5s |
| `initialDelay` | 5s |
| `failureThreshold` | 1 |
| `successThreshold` | 1 |

## Conformance-тесты

Инфраструктура для проверки соответствия SDK спецификации (`conformance/`):

- Kubernetes-манифесты для зависимостей (PostgreSQL, Redis, RabbitMQ, Kafka)
- Управляемые HTTP и gRPC заглушки
- 8 тестовых сценариев: basic-health, partial-failure, full-failure, recovery,
  latency, labels, timeout, initial-state
- Python runner для парсинга и проверки Prometheus-метрик

## Статус разработки

| Фаза | Описание | Статус |
| --- | --- | --- |
| 1 | Спецификация | Завершена |
| 2 | Conformance-тесты (Kubernetes) | Завершена |
| 3 | Go SDK: ядро и парсер | Завершена |
| 4 | Go SDK: чекеры (8 типов) | Завершена |
| 5 | Go SDK: метрики и планировщик | Завершена |
| 6 | Go SDK: публичный API и contrib | Завершена |
| 7 | Тестовый сервис на Go | Завершена |
| 8 | Conformance-прогон Go SDK | Завершена |
| 9 | Документация и CI/CD | В процессе |
| 10 | Grafana дашборды и алерты | В планах |

SDK для Java, C# и Python планируются после стабилизации Go SDK.

## Документация

- [Быстрый старт Go SDK](docs/quickstart/go.md) — установка и минимальный пример
- [Руководство по интеграции](docs/migration/go.md) — подключение к существующему сервису
- [Обзор спецификации](docs/specification.md) — контракты метрик, поведения, конфигурации

Детальная спецификация: [`spec/`](spec/)

## Лицензия

[MIT License](LICENSE) - Copyright (c) 2026 Artur Kryukov
