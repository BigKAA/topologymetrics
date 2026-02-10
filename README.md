[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# Topology Metrics (dephealth)

SDK for monitoring microservice dependencies. Each service exports
Prometheus metrics about the health of its dependencies (databases, caches,
queues, HTTP/gRPC services). VictoriaMetrics collects data, Grafana visualizes.

**Supported languages**: Go, Python, Java, C#

## Problem

A system of hundreds of microservices faces three problems:

- **Slow root cause analysis** — when a failure occurs, it is unclear which service is the source
- **No dependency map** — nobody sees the full picture of connections between services
- **Cascading failures** — one service going down triggers a flood of alerts from dependents

## Solution

Each microservice exports metrics about the health of its connections:

```text
# Health: 1 = available, 0 = unavailable
app_dependency_health{name="order-service",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes"} 1

# Check latency
app_dependency_latency_seconds_bucket{name="order-service",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.01"} 42
```

From these metrics a dependency graph is automatically built, alerting is configured
with cascade suppression, and the degradation level of each service is displayed.

## Quick Start

### Go

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
)

dh, err := dephealth.New("order-service",
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
    dephealth.Redis("redis-cache",
        dephealth.FromURL(os.Getenv("REDIS_URL")),
        dephealth.Critical(false),
    ),
)
dh.Start(ctx)
defer dh.Stop()

http.Handle("/metrics", promhttp.Handler())
```

### Python (FastAPI)

```python
from dephealth.api import postgres_check, redis_check
from dephealth_fastapi import dephealth_lifespan, DepHealthMiddleware

app = FastAPI(
    lifespan=dephealth_lifespan("order-service",
        postgres_check("postgres-main", url=os.environ["DATABASE_URL"], critical=True),
        redis_check("redis-cache", url=os.environ["REDIS_URL"], critical=False),
    )
)
app.add_middleware(DepHealthMiddleware)
```

### Java (Spring Boot)

```yaml
# application.yml
dephealth:
  name: order-service
  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      critical: true
    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false
```

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.2.2</version>
</dependency>
```

### C# (ASP.NET Core)

```csharp
builder.Services.AddDepHealth("order-service", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
);

app.UseDepHealth(); // /metrics + /health/dependencies
```

## Architecture

A native library for each language, unified by a common specification.
Not a sidecar, not FFI — deep integration with each language's ecosystem.

```text
┌─────────────────────────────────────────────┐
│         Framework Integration               │  Spring Boot / ASP.NET / FastAPI
├─────────────────────────────────────────────┤
│         Metrics Exporter                    │  Prometheus gauges + histograms
├─────────────────────────────────────────────┤
│         Check Scheduler                     │  Periodic health checks
├─────────────────────────────────────────────┤
│         Health Checkers                     │  HTTP, gRPC, TCP, Postgres, MySQL,
│                                             │  Redis, AMQP, Kafka
├─────────────────────────────────────────────┤
│         Connection Config Parser            │  URL / params / connection string
├─────────────────────────────────────────────┤
│         Core Abstractions                   │  Dependency, Endpoint, HealthChecker
└─────────────────────────────────────────────┘
```

### Two Check Modes

- **Standalone** — SDK creates a temporary connection for the check
- **Pool integration** — SDK uses the service's existing connection pool
  (preferred, reflects the service's actual ability to work with the dependency)

## Supported Check Types

| Type | Check Method |
| --- | --- |
| `http` | HTTP GET to `healthPath`, follows redirects, expects 2xx |
| `grpc` | gRPC Health Check Protocol (grpc.health.v1) |
| `tcp` | TCP connection establishment |
| `postgres` | `SELECT 1` via connection pool or new connection |
| `mysql` | `SELECT 1` via connection pool or new connection |
| `redis` | `PING` command |
| `amqp` | Broker connection check |
| `kafka` | Metadata request to broker |

## Repository Structure

```text
spec/                           # Unified specification (metric, behavior, config contracts)
conformance/                    # Conformance tests (Kubernetes, 8 scenarios × 4 languages)
sdk-go/                         # Go SDK
sdk-python/                     # Python SDK
sdk-java/                       # Java SDK (Maven multi-module)
sdk-csharp/                     # C# SDK (.NET 8)
test-services/                  # Test microservices for each language
deploy/                         # Monitoring: Grafana, Alertmanager, VictoriaMetrics
docs/                           # Documentation (quickstart, migration, specification)
plans/                          # Development plans
```

## Specification

The single source of truth for all SDKs — the `spec/` directory:

- [`spec/metric-contract.md`](spec/metric-contract.md) — metric format, labels, values
- [`spec/check-behavior.md`](spec/check-behavior.md) — check lifecycle, thresholds, timeouts
- [`spec/config-contract.md`](spec/config-contract.md) — connection configuration formats

### Key Metrics

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | Availability: `1` / `0` |
| `app_dependency_latency_seconds` | Histogram | Check latency |

Required labels: `name`, `dependency`, `type`, `host`, `port`, `critical`.

### Default Parameters

| Parameter | Value |
| --- | --- |
| `checkInterval` | 15s |
| `timeout` | 5s |
| `failureThreshold` | 1 |
| `successThreshold` | 1 |

## Conformance Tests

Infrastructure for verifying SDK compliance with the specification (`conformance/`):

- Kubernetes manifests for dependencies (PostgreSQL, Redis, RabbitMQ, Kafka)
- Managed HTTP and gRPC stubs
- 8 test scenarios: basic-health, partial-failure, full-failure, recovery,
  latency, labels, timeout, initial-state
- All 4 SDKs pass 8/8 scenarios (32 tests total)

## Documentation

### Quick Start

- [Go](docs/quickstart/go.md)
- [Python](docs/quickstart/python.md)
- [Java](docs/quickstart/java.md)
- [C#](docs/quickstart/csharp.md)

### Integration Guide

- [Go](docs/migration/go.md)
- [Python](docs/migration/python.md)
- [Java](docs/migration/java.md)
- [C#](docs/migration/csharp.md)

### Additional

- [SDK Comparison](docs/comparison.md) — all languages side-by-side
- [Specification Overview](docs/specification.md) — metric, behavior, config contracts

## Development

Detailed developer guide — [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT License](LICENSE) - Copyright (c) 2026 Artur Kryukov

---

<a id="russian"></a>

# Topology Metrics (dephealth)

SDK для мониторинга зависимостей микросервисов. Каждый сервис экспортирует
Prometheus-метрики о состоянии своих зависимостей (БД, кэши, очереди,
HTTP/gRPC-сервисы). VictoriaMetrics собирает данные, Grafana визуализирует.

**Поддерживаемые языки**: Go, Python, Java, C#

## Проблема

Система из сотен микросервисов сталкивается с тремя проблемами:

- **Долгий поиск первопричины** — при сбое неясно, какой сервис является источником
- **Нет карты зависимостей** — никто не видит полную картину связей между сервисами
- **Каскадные сбои** — падение одного сервиса вызывает шквал алертов от зависимых

## Решение

Каждый микросервис экспортирует метрики о состоянии своих соединений:

```text
# Здоровье: 1 = доступен, 0 = недоступен
app_dependency_health{name="order-service",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes"} 1

# Латентность проверки
app_dependency_latency_seconds_bucket{name="order-service",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.01"} 42
```

Из метрик автоматически строится граф зависимостей, настраивается алертинг
с подавлением каскадов, отображается степень деградации каждого сервиса.

## Быстрый старт

### Go

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
)

dh, err := dephealth.New("order-service",
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
    dephealth.Redis("redis-cache",
        dephealth.FromURL(os.Getenv("REDIS_URL")),
        dephealth.Critical(false),
    ),
)
dh.Start(ctx)
defer dh.Stop()

http.Handle("/metrics", promhttp.Handler())
```

### Python (FastAPI)

```python
from dephealth.api import postgres_check, redis_check
from dephealth_fastapi import dephealth_lifespan, DepHealthMiddleware

app = FastAPI(
    lifespan=dephealth_lifespan("order-service",
        postgres_check("postgres-main", url=os.environ["DATABASE_URL"], critical=True),
        redis_check("redis-cache", url=os.environ["REDIS_URL"], critical=False),
    )
)
app.add_middleware(DepHealthMiddleware)
```

### Java (Spring Boot)

```yaml
# application.yml
dephealth:
  name: order-service
  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      critical: true
    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false
```

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.2.2</version>
</dependency>
```

### C# (ASP.NET Core)

```csharp
builder.Services.AddDepHealth("order-service", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
);

app.UseDepHealth(); // /metrics + /health/dependencies
```

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
| `http` | HTTP GET к `healthPath`, следует редиректам, ожидание 2xx |
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
conformance/                    # Conformance-тесты (Kubernetes, 8 сценариев × 4 языка)
sdk-go/                         # Go SDK
sdk-python/                     # Python SDK
sdk-java/                       # Java SDK (Maven multi-module)
sdk-csharp/                     # C# SDK (.NET 8)
test-services/                  # Тестовые микросервисы для каждого языка
deploy/                         # Мониторинг: Grafana, Alertmanager, VictoriaMetrics
docs/                           # Документация (quickstart, migration, specification)
plans/                          # Планы разработки
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

Обязательные метки: `name`, `dependency`, `type`, `host`, `port`, `critical`.

### Параметры по умолчанию

| Параметр | Значение |
| --- | --- |
| `checkInterval` | 15s |
| `timeout` | 5s |
| `failureThreshold` | 1 |
| `successThreshold` | 1 |

## Conformance-тесты

Инфраструктура для проверки соответствия SDK спецификации (`conformance/`):

- Kubernetes-манифесты для зависимостей (PostgreSQL, Redis, RabbitMQ, Kafka)
- Управляемые HTTP и gRPC заглушки
- 8 тестовых сценариев: basic-health, partial-failure, full-failure, recovery,
  latency, labels, timeout, initial-state
- Все 4 SDK проходят 8/8 сценариев (32 теста суммарно)

## Документация

### Быстрый старт

- [Go](docs/quickstart/go.md)
- [Python](docs/quickstart/python.md)
- [Java](docs/quickstart/java.md)
- [C#](docs/quickstart/csharp.md)

### Руководство по интеграции

- [Go](docs/migration/go.md)
- [Python](docs/migration/python.md)
- [Java](docs/migration/java.md)
- [C#](docs/migration/csharp.md)

### Дополнительно

- [Сравнение SDK](docs/comparison.md) — все языки side-by-side
- [Обзор спецификации](docs/specification.md) — контракты метрик, поведения, конфигурации

## Разработка

Подробное руководство для разработчиков — [CONTRIBUTING.md](CONTRIBUTING.md).

## Лицензия

[MIT License](LICENSE) - Copyright (c) 2026 Artur Kryukov
