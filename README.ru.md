*[English version](README.md)*

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
docs/                           # Документация (quickstart, migration, alerting, specification)
plans/                          # Планы разработки
```

## Спецификация

Единый источник правды для всех SDK — каталог `spec/`:

- [`spec/metric-contract.md`](spec/metric-contract.ru.md) — формат метрик, метки, значения
- [`spec/check-behavior.md`](spec/check-behavior.ru.md) — жизненный цикл проверок, пороги, таймауты
- [`spec/config-contract.md`](spec/config-contract.ru.md) — форматы конфигурации соединений

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

- [Go](docs/quickstart/go.ru.md)
- [Python](docs/quickstart/python.ru.md)
- [Java](docs/quickstart/java.ru.md)
- [C#](docs/quickstart/csharp.ru.md)

### Руководство по интеграции

- [Go](docs/migration/go.ru.md)
- [Python](docs/migration/python.ru.md)
- [Java](docs/migration/java.ru.md)
- [C#](docs/migration/csharp.ru.md)

### Алертинг и мониторинг

- [Обзор стека мониторинга](docs/alerting/overview.ru.md) — архитектура, scraping, VictoriaMetrics/Prometheus
- [Правила алертов](docs/alerting/alert-rules.ru.md) — 5 встроенных правил с разбором PromQL
- [Уменьшение шума](docs/alerting/noise-reduction.ru.md) — сценарии, подавление, best practices
- [Конфигурация Alertmanager](docs/alerting/alertmanager.ru.md) — маршрутизация, receivers, шаблоны
- [Кастомные правила](docs/alerting/custom-rules.ru.md) — написание своих правил поверх dephealth-метрик

### Дополнительно

- [Сравнение SDK](docs/comparison.ru.md) — все языки side-by-side
- [Обзор спецификации](docs/specification.ru.md) — контракты метрик, поведения, конфигурации
- [Grafana Dashboards](docs/grafana-dashboards.ru.md) — 5 дашбордов для мониторинга зависимостей

## Разработка

Подробное руководство для разработчиков — [CONTRIBUTING.md](CONTRIBUTING.ru.md).

## Лицензия

[MIT License](LICENSE) - Copyright (c) 2026 Artur Kryukov
