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
app_dependency_health{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432"} 1

# Латентность проверки
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="0.01"} 42
```

Из метрик автоматически строится граф зависимостей, настраивается алертинг
с подавлением каскадов, отображается степень деградации каждого сервиса.

## Быстрый старт

### Go

```go
import (
    "github.com/BigKAA/topologymetrics/dephealth"
    _ "github.com/BigKAA/topologymetrics/dephealth/checks"
)

dh, err := dephealth.New(
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
    dephealth.Redis("redis-cache",
        dephealth.FromURL(os.Getenv("REDIS_URL")),
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
    lifespan=dephealth_lifespan(
        postgres_check("postgres-main", url=os.environ["DATABASE_URL"]),
        redis_check("redis-cache", url=os.environ["REDIS_URL"]),
    )
)
app.add_middleware(DepHealthMiddleware)
```

### Java (Spring Boot)

```yaml
# application.yml
dephealth:
  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      critical: true
    redis-cache:
      type: redis
      url: ${REDIS_URL}
```

```xml
<dependency>
    <groupId>com.github.bigkaa</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.1.0-SNAPSHOT</version>
</dependency>
```

### C# (ASP.NET Core)

```csharp
builder.Services.AddDepHealth(dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!))
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

Обязательные метки: `dependency`, `type`, `host`, `port`.

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
