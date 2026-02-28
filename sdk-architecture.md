# Архитектура SDK мониторинга зависимостей

## 1. Архитектурное решение: нативные SDK + общая спецификация

### Почему не единая бинарная библиотека

Рассматривались три подхода:

| Подход | Суть | Вердикт |
| ------ | ---- | ------- |
| Единое ядро (Go/Rust) + FFI-обёртки | JNI, P/Invoke, CGo, ctypes | Отклонён |
| Нативная библиотека на каждом языке | Общая спецификация, независимые реализации | **Выбран** |
| Sidecar-процесс | Внешний агент проверяет endpoint-ы | Отклонён |

**Причина выбора**: основная ценность SDK — не логика проверки (она тривиальна), а **глубокая интеграция с экосистемой каждого языка**:

- Java — Spring Boot auto-configuration, Micrometer, JDBC DataSource
- C# — ASP.NET Health Checks, IHealthCheck, EF Core, prometheus-net
- Go — `prometheus/client_golang`, `database/sql`, стандартные паттерны
- Python — `prometheus_client`, SQLAlchemy, FastAPI/Django

FFI-библиотека не может реализовать эти интеграции. Sidecar не видит внутреннее состояние connection pool.

---

## 2. Структура монорепо

```text
dephealth/
├── spec/                          # Спецификация (единая для всех языков)
│   ├── metric-contract.md         # Имена метрик, метки, значения
│   ├── check-behavior.md          # Поведение проверок (интервалы, таймауты)
│   └── config-contract.md         # Форматы конфигурации соединений
│
├── conformance/                   # Conformance-тесты
│   ├── docker-compose.yml         # Тестовая инфраструктура (PG, Redis, RabbitMQ...)
│   ├── scenarios/                 # Тестовые сценарии (YAML)
│   └── runner/                    # Запуск тестов, проверка метрик через /metrics
│
├── sdk-java/                      # Java SDK
│   ├── dephealth-core/            # Ядро (проверки, метрики)
│   ├── dephealth-spring-boot/     # Spring Boot starter
│   └── dephealth-micrometer/      # Мост Micrometer → Prometheus
│
├── sdk-go/                        # Go SDK
│   ├── dephealth/                 # Основной модуль
│   └── dephealth/contrib/         # Интеграции (database/sql, etc.)
│
├── sdk-csharp/                    # C# SDK
│   ├── DepHealth.Core/            # Ядро
│   ├── DepHealth.AspNetCore/      # ASP.NET интеграция
│   └── DepHealth.EntityFramework/ # EF Core интеграция
│
├── sdk-python/                    # Python SDK
│   ├── dephealth/                 # Ядро (пакет)
│   ├── dephealth-fastapi/         # FastAPI интеграция
│   └── dephealth-django/          # Django интеграция
│
└── docs/                          # Общая документация
    ├── migration/                 # Общие cross-SDK миграции
    ├── code-style/                # Общие принципы и тестирование
    └── alerting/                  # Алертинг и мониторинг
```

Каждый `sdk-*` публикуется как независимый пакет в соответствующем реестре
(Maven Central, NuGet, Go modules, PyPI), но версионируется координированно.

---

## 3. Общая спецификация

Спецификация — единый источник правды. Все языковые SDK **обязаны** ей следовать.
Соответствие проверяется conformance-тестами.

### 3.1. Контракт метрик

**Метрика здоровья**:

- Имя: `app_dependency_health`
- Тип: Gauge
- Значение: `1` (доступен) или `0` (недоступен)
- Обязательные метки:

| Метка | Описание | Пример |
| ----- | -------- | ------ |
| `name` | Уникальное имя приложения | `order-service` |
| `group` | Логическая группа | `billing-team` |
| `dependency` | Логическое имя зависимости | `payment-service`, `postgres-main` |
| `type` | Тип соединения | `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka`, `ldap` |
| `host` | Адрес endpoint-а | `pg-master.db.svc.cluster.local` |
| `port` | Порт endpoint-а | `5432` |
| `critical` | Критичность зависимости | `yes`, `no` |

Опциональные метки: произвольные через `WithLabel(key, value)`.

**Метрика латентности**:

- Имя: `app_dependency_latency_seconds`
- Тип: Histogram
- Buckets: `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]`
- Метки: те же, что у `app_dependency_health`

**Метрика статуса**:

- Имя: `app_dependency_status`
- Тип: Gauge (enum-паттерн)
- Значения: `1` (активный статус), `0` (неактивный статус)
- Значения `status`: `ok`, `timeout`, `connection_error`, `dns_error`, `auth_error`, `tls_error`, `unhealthy`, `error`
- Метки: те же, что у `app_dependency_health` + `status`
- Все 8 серий status всегда экспортируются для каждого endpoint

**Метрика детализации статуса**:

- Имя: `app_dependency_status_detail`
- Тип: Gauge (info-паттерн)
- Значение: всегда `1`
- Метки: те же, что у `app_dependency_health` + `detail`
- Одна серия на endpoint; при смене detail старая серия удаляется

### 3.2. Контракт поведения проверок

| Параметр | Значение по умолчанию | Описание |
| -------- | --------------------- | -------- |
| `checkInterval` | 15s | Интервал между проверками |
| `timeout` | 5s | Таймаут одной проверки |
| `initialDelay` | 5s | Задержка перед первой проверкой (после старта сервиса) |
| `failureThreshold` | 1 | Сколько неудачных проверок подряд для перехода в `0` |
| `successThreshold` | 1 | Сколько успешных проверок для возврата в `1` |

**Типы проверок**:

| Тип | Метод проверки |
| --- | -------------- |
| `http` | HTTP GET к `healthPath` (по умолчанию `/health`), ожидается 2xx |
| `grpc` | gRPC Health Check Protocol (grpc.health.v1) |
| `tcp` | Установка TCP-соединения |
| `postgres` | `SELECT 1` через connection pool |
| `mysql` | `SELECT 1` через connection pool |
| `redis` | Команда `PING` |
| `amqp` | Проверка соединения с брокером |
| `kafka` | Metadata request к брокеру |
| `ldap` | LDAP bind или поисковая операция |

### 3.3. Контракт конфигурации

SDK принимает параметры соединений в нескольких форматах:

**Полный URL**:

```text
postgres://user:pass@pg-master.db.svc:5432/orders
http://payment-svc.payments.svc:8080
redis://redis-0.cache.svc:6379/0
amqp://user:pass@rabbit-0.mq.svc:5672/vhost
```

**Отдельные параметры**:

```text
host=pg-master.db.svc port=5432
# или переменные окружения:
DB_HOST=pg-master.db.svc DB_PORT=5432
```

**Connection string** (специфичные для технологии):

```text
Host=pg-master.db.svc;Port=5432;Database=orders;Username=app
Server=pg-master.db.svc,5432;Database=orders
```

SDK автоматически извлекает `host` и `port` из любого формата.
Разработчик указывает только:

- Логическое имя зависимости (`dependency`)
- Тип соединения (`type`) — если не определяется автоматически из URL-схемы
- Признак критичности (`critical`) — влияет ли на readiness

---

## 4. Внутренняя архитектура SDK (общая для всех языков)

### Слои

```text
┌─────────────────────────────────────────────┐
│         Framework Integration               │  Spring Boot / ASP.NET / FastAPI / ...
│         (auto-configuration, middleware)     │
├─────────────────────────────────────────────┤
│         Metrics Exporter                    │  Prometheus / Micrometer / prometheus-net
│         (gauges, histograms, /metrics)       │
├─────────────────────────────────────────────┤
│         Check Scheduler                     │  Периодический запуск проверок
│         (goroutines / threads / asyncio)     │
├─────────────────────────────────────────────┤
│         Health Checkers                     │  HTTP, gRPC, TCP, Postgres, MySQL, Redis, AMQP, Kafka, LDAP
│         (реализации проверок по типам)       │
├─────────────────────────────────────────────┤
│         Connection Config Parser            │  URL → (host, port, type)
│         (парсинг разных форматов)            │  Params → (host, port)
├─────────────────────────────────────────────┤
│         Core Abstractions                   │  Dependency, Endpoint, HealthCheck interface
└─────────────────────────────────────────────┘
```

### Ключевые абстракции

**Dependency** — описание зависимости:

- `name: string` — логическое имя (`postgres-main`)
- `type: string` — тип соединения (`postgres`)
- `critical: bool` — влияет на readiness
- `endpoints: []Endpoint` — список endpoint-ов

**Endpoint** — один endpoint зависимости:

- `host: string`
- `port: int`
- `metadata: map[string]string` — дополнительные метки (role, shard, etc.)

**HealthChecker** — интерфейс проверки:

- `check(ctx, endpoint) → bool` — выполнить проверку
- Реализации: `HTTPChecker`, `GRPCChecker`, `TCPChecker`, `PostgresChecker`, `MySQLChecker`, `RedisChecker`, `AMQPChecker`, `KafkaChecker`, `LDAPChecker`

**ConnectionParser** — парсер конфигурации:

- `parse(input) → (host, port, type)` — извлечь параметры из URL, connection string или отдельных параметров

---

## 5. SDK для каждого языка

### 5.1. Go

**Модули**:

- `github.com/BigKAA/topologymetrics/sdk-go` — ядро
- `github.com/BigKAA/topologymetrics/sdk-go/contrib/sqldb` — интеграция с `database/sql`
- `github.com/BigKAA/topologymetrics/sdk-go/contrib/redispool` — интеграция с `go-redis`

**API инициализации**:

```go
package main

import (
    "github.com/BigKAA/topologymetrics/sdk-go"
    "github.com/BigKAA/topologymetrics/sdk-go/contrib/sqldb"
)

func main() {
    checker := dephealth.New(
        // Зависимость из URL
        dephealth.HTTP("payment-service",
            dephealth.FromURL(os.Getenv("PAYMENT_SERVICE_URL")),
            dephealth.Critical(true),
        ),

        // Зависимость из отдельных параметров
        dephealth.Postgres("postgres-main",
            dephealth.FromParams(os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
            dephealth.Critical(true),
        ),

        // Зависимость из connection string
        dephealth.Redis("redis-cache",
            dephealth.FromURL(os.Getenv("REDIS_URL")),
        ),

        // Интеграция с существующим *sql.DB
        // (использует connection pool сервиса, а не создаёт новое соединение)
        sqldb.FromDB("postgres-main", db,
            dephealth.Critical(true),
        ),
    )

    // Метрики регистрируются автоматически в prometheus.DefaultRegisterer
    checker.Start(ctx)

    // HTTP handler для /metrics уже есть (promhttp)
    http.Handle("/metrics", promhttp.Handler())
}
```

**Внутренняя структура**:

```text
dephealth/
├── dephealth.go          # New(), Option pattern
├── dependency.go         # Dependency, Endpoint structs
├── checker.go            # HealthChecker interface, Check Scheduler
├── parser.go             # ConnectionParser (URL, params, connstring)
├── metrics.go            # Prometheus gauges и histograms
├── checks/
│   ├── http.go           # HTTPChecker
│   ├── grpc.go           # GRPCChecker
│   ├── tcp.go            # TCPChecker
│   ├── postgres.go       # PostgresChecker (SELECT 1)
│   ├── redis.go          # RedisChecker (PING)
│   ├── amqp.go           # AMQPChecker
│   ├── kafka.go          # KafkaChecker
│   ├── mysql.go          # MySQLChecker
│   └── ldap.go           # LDAPChecker
└── contrib/
    ├── sqldb/            # Интеграция с database/sql
    └── redispool/        # Интеграция с go-redis
```

---

### 5.2. Java

**Модули (Maven)**:

- `biz.kryukov.dev:dephealth-core` — ядро, проверки, Prometheus-метрики
- `biz.kryukov.dev:dephealth-spring-boot-starter` — Spring Boot auto-configuration
- `biz.kryukov.dev:dephealth-micrometer` — мост для Micrometer (если используется вместо Prometheus напрямую)

**API инициализации (Spring Boot)**:

```java
// application.yml — ничего нового, используем существующие настройки
spring:
  datasource:
    url: jdbc:postgresql://pg-master.db.svc:5432/orders
  redis:
    url: redis://redis-0.cache.svc:6379/0

// DependencyHealthConfig.java
@Configuration
public class DependencyHealthConfig {

    @Bean
    public DependencyHealth dependencyHealth(
            DataSource dataSource,           // существующий бин
            RedisConnectionFactory redis     // существующий бин
    ) {
        return DependencyHealth.builder()
            // Зависимость из URL
            .http("payment-service",
                  fromEnv("PAYMENT_SERVICE_URL"),
                  critical(true))

            // Интеграция с существующим DataSource
            // (проверяет через тот же connection pool)
            .jdbc("postgres-main", dataSource,
                  critical(true))

            // Интеграция с существующим RedisConnectionFactory
            .redis("redis-cache", redis)

            // Зависимость из отдельных параметров
            .amqp("rabbitmq",
                  fromParams(env("RABBIT_HOST"), env("RABBIT_PORT")))

            .build();
    }
}
```

**Spring Boot auto-configuration**:

```java
// При подключении dephealth-spring-boot-starter автоматически:
// 1. Регистрирует Prometheus-метрики (через MeterRegistry или напрямую)
// 2. Запускает check scheduler
// 3. Добавляет /actuator/health/dependencies endpoint
// 4. Влияет на readiness probe (для critical-зависимостей)

@SpringBootApplication
@EnableDependencyHealth  // активирует auto-configuration
public class OrderServiceApplication {
    public static void main(String[] args) {
        SpringApplication.run(OrderServiceApplication.class, args);
    }
}
```

**Внутренняя структура**:

```text
sdk-java/
├── dephealth-core/
│   └── src/main/java/biz/kryukov/dev/dephealth/
│       ├── DependencyHealth.java        # Builder, основной API
│       ├── Dependency.java              # Модель зависимости
│       ├── Endpoint.java                # Модель endpoint-а
│       ├── HealthChecker.java           # Интерфейс проверки
│       ├── CheckScheduler.java          # Планировщик проверок
│       ├── ConnectionParser.java        # Парсер URL/params/connstring
│       ├── PrometheusExporter.java      # Экспорт метрик
│       └── checkers/
│           ├── HttpChecker.java
│           ├── GrpcChecker.java
│           ├── TcpChecker.java
│           ├── JdbcChecker.java         # Использует DataSource.getConnection()
│           ├── RedisChecker.java        # Использует RedisConnectionFactory
│           ├── AmqpChecker.java
│           ├── KafkaChecker.java
│           ├── MySqlChecker.java
│           └── LdapChecker.java
│
├── dephealth-spring-boot/
│   └── src/main/java/biz/kryukov/dev/dephealth/spring/
│       ├── DependencyHealthAutoConfiguration.java
│       ├── DependencyHealthIndicator.java  # → /actuator/health
│       ├── DependencyReadinessContributor.java
│       └── EnableDependencyHealth.java     # Аннотация
│
└── dephealth-micrometer/
    └── src/main/java/com/github/bigkaa/dephealth/micrometer/
        └── MicrometerExporter.java     # Мост: пишет метрики через MeterRegistry
```

---

### 5.3. C\#

**NuGet-пакеты**:

- `DepHealth.Core` — ядро, проверки, prometheus-net метрики
- `DepHealth.AspNetCore` — ASP.NET middleware, IHealthCheck
- `DepHealth.EntityFramework` — интеграция с EF Core DbContext

**API инициализации (ASP.NET)**:

```csharp
// Program.cs
var builder = WebApplication.CreateBuilder(args);

// Подключение через стандартный DI
builder.Services.AddDependencyHealth(health =>
{
    // Зависимость из URL (из существующей конфигурации)
    health.AddHttp("payment-service",
        fromUrl: builder.Configuration["PaymentService:Url"],
        critical: true);

    // Интеграция с существующим DbContext
    // (проверяет через тот же connection pool)
    health.AddNpgsql<OrderDbContext>("postgres-main",
        critical: true);

    // Зависимость из отдельных параметров
    health.AddRedis("redis-cache",
        fromUrl: builder.Configuration.GetConnectionString("Redis"));

    // AMQP из отдельных параметров
    health.AddAmqp("rabbitmq",
        host: builder.Configuration["RabbitMQ:Host"],
        port: int.Parse(builder.Configuration["RabbitMQ:Port"]));
});

var app = builder.Build();

// Prometheus-метрики (prometheus-net)
app.UseMetricServer();  // /metrics
app.MapHealthChecks("/health/dependencies");  // ASP.NET Health Checks

app.Run();
```

**Внутренняя структура**:

```text
sdk-csharp/
├── DepHealth.Core/
│   ├── DependencyHealthBuilder.cs
│   ├── Dependency.cs
│   ├── Endpoint.cs
│   ├── IHealthChecker.cs              # Интерфейс проверки
│   ├── CheckScheduler.cs
│   ├── ConnectionParser.cs
│   ├── PrometheusExporter.cs          # prometheus-net gauges/histograms
│   └── Checkers/
│       ├── HttpChecker.cs
│       ├── GrpcChecker.cs
│       ├── TcpChecker.cs
│       ├── NpgsqlChecker.cs           # Использует NpgsqlConnection
│       ├── RedisChecker.cs            # Использует StackExchange.Redis
│       ├── AmqpChecker.cs
│       ├── KafkaChecker.cs
│       ├── MySqlChecker.cs
│       └── LdapChecker.cs
│
├── DepHealth.AspNetCore/
│   ├── ServiceCollectionExtensions.cs  # AddDependencyHealth()
│   ├── DependencyHealthCheck.cs        # Реализует IHealthCheck
│   └── DependencyReadinessCheck.cs     # Влияет на /health/ready
│
└── DepHealth.EntityFramework/
    └── EfCoreHealthExtensions.cs       # AddNpgsql<TContext>()
```

---

### 5.4. Python

**PyPI-пакеты**:

- `dephealth` — ядро, проверки, prometheus_client метрики
- `dephealth-fastapi` — FastAPI интеграция (lifespan, middleware)
- `dephealth-django` — Django app (планируется)

**API инициализации (FastAPI)**:

```python
# main.py
from dephealth import DependencyHealth, http_check, postgres_check, redis_check
from dephealth.fastapi import DependencyHealthMiddleware
import os

health = DependencyHealth(
    # Зависимость из URL
    http_check(
        "payment-service",
        url=os.environ["PAYMENT_SERVICE_URL"],
        critical=True,
    ),

    # Зависимость из отдельных параметров
    postgres_check(
        "postgres-main",
        host=os.environ["DB_HOST"],
        port=os.environ["DB_PORT"],
        critical=True,
    ),

    # Зависимость из connection string
    redis_check(
        "redis-cache",
        url=os.environ["REDIS_URL"],
    ),

    # Интеграция с существующим SQLAlchemy engine
    # (проверяет через тот же connection pool)
    postgres_check(
        "postgres-main",
        engine=existing_engine,   # SQLAlchemy Engine
        critical=True,
    ),
)

app = FastAPI(lifespan=health.lifespan)
# Метрики на /metrics через prometheus_client
app.add_middleware(DependencyHealthMiddleware, health=health)
```

**API инициализации (Django)**:

```python
# settings.py
INSTALLED_APPS = [
    "dephealth.django",
    ...
]

DEPHEALTH = {
    "dependencies": [
        {
            "name": "postgres-main",
            "type": "postgres",
            "url": os.environ["DATABASE_URL"],
            "critical": True,
        },
        {
            "name": "redis-cache",
            "type": "redis",
            "url": os.environ["REDIS_URL"],
        },
    ]
}

# urls.py
urlpatterns = [
    path("metrics", dephealth_metrics_view),
]
```

**Внутренняя структура**:

```text
sdk-python/
├── dephealth/
│   ├── __init__.py               # DependencyHealth, публичный API
│   ├── dependency.py             # Dependency, Endpoint dataclasses
│   ├── checker.py                # HealthChecker protocol (typing.Protocol)
│   ├── scheduler.py              # asyncio / threading scheduler
│   ├── parser.py                 # ConnectionParser (URL, params)
│   ├── metrics.py                # prometheus_client gauges/histograms
│   └── checkers/
│       ├── http.py
│       ├── grpc.py
│       ├── tcp.py
│       ├── postgres.py           # psycopg / asyncpg / SQLAlchemy
│       ├── redis.py              # redis-py
│       ├── amqp.py               # aio-pika / pika
│       ├── kafka.py              # confluent-kafka / aiokafka
│       ├── mysql.py              # aiomysql
│       └── ldap.py               # python-ldap / bonsai
│
├── dephealth-fastapi/
│   ├── middleware.py              # DependencyHealthMiddleware
│   └── lifespan.py               # Интеграция с FastAPI lifespan
│
└── dephealth-django/
    ├── apps.py                    # Django AppConfig
    ├── views.py                   # /metrics view
    └── health_checks.py           # Django Health Checks
```

---

## 6. Ключевое: интеграция с существующим connection pool

Важная деталь, общая для всех языков. SDK должен уметь работать в двух режимах:

**Режим 1: Автономная проверка** — SDK сам создаёт временное соединение для проверки.
Используется, когда у SDK нет доступа к connection pool сервиса.

```text
SDK → новое TCP/HTTP-соединение → endpoint → ответ → закрытие
```

**Режим 2: Интеграция с connection pool** — SDK использует существующий
connection pool сервиса (DataSource, DbContext, Engine, sql.DB).
Более точный результат: если pool исчерпан — это тоже проблема.

```text
SDK → pool.getConnection() → SELECT 1 → освобождение
```

Режим 2 предпочтительнее, так как отражает реальную способность сервиса
работать с зависимостью. Но требует, чтобы разработчик передал ссылку
на pool при инициализации.

---

## 7. Conformance-тесты

Каждый языковой SDK проверяется на соответствие спецификации:

```text
conformance/
├── docker-compose.yml     # PostgreSQL, Redis, RabbitMQ, HTTP-сервер-заглушка
├── scenarios/
│   ├── basic-health.yml   # Все endpoint-ы доступны → все метрики = 1
│   ├── partial-failure.yml # 1 из 3 реплик PG недоступна → 2 метрики = 1, 1 = 0
│   ├── full-failure.yml   # Остановить Redis → метрика = 0
│   ├── recovery.yml       # Восстановить Redis → метрика возвращается в 1
│   ├── latency.yml        # Проверить наличие histogram-а
│   └── labels.yml         # Проверить правильность меток
└── runner/
    └── verify.py          # Запрашивает /metrics, парсит, сверяет с ожиданиями
```

**Процесс**:

1. `docker-compose up` — поднять инфраструктуру
2. Запустить тестовый сервис на конкретном SDK
3. Дождаться нескольких циклов проверок
4. `GET /metrics` — получить метрики
5. Сверить с ожиданиями из сценария (имена, метки, значения, типы)
6. Изменить состояние инфраструктуры (остановить контейнер)
7. Сверить, что метрика изменилась

Conformance-тесты запускаются в CI при каждом изменении любого SDK.

---

## 8. Стратегия релизов

- Координированное версионирование: все SDK имеют одну мажорную версию
  (совместимы с одной версией спецификации)
- Минорные и патч-версии могут отличаться между языками
- Спецификация версионируется отдельно: `spec v1.0`, `spec v1.1`
- Каждый SDK указывает, какую версию спецификации реализует
- Conformance-тесты привязаны к версии спецификации

Пример: `spec v1.0` → `sdk-java 1.0.3`, `sdk-go 1.0.1`, `sdk-csharp 1.0.0`, `sdk-python 1.0.2`

---

## 9. Порядок реализации

### Шаг 1. Спецификация и conformance-тесты

Зафиксировать контракт метрик, поведения и конфигурации.
Создать conformance-тесты до написания SDK — это гарантия согласованности.

### Шаг 2. SDK на языке пилотных сервисов

Реализовать SDK для языка, на котором написаны пилотные 5–10 сервисов.
Обкатать на пилоте, собрать обратную связь, скорректировать спецификацию.

### Шаг 3. Остальные языки

Параллельная реализация SDK для оставшихся трёх языков.
Каждый SDK проходит conformance-тесты перед релизом.

### Шаг 4. Framework-интеграции

После стабилизации ядра — интеграции со Spring Boot, ASP.NET, FastAPI, Django.
Это отдельные пакеты, чтобы не создавать лишних зависимостей в ядре.
