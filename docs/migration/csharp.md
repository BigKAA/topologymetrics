[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# Guide to Integrating dephealth into an Existing .NET Service

Step-by-step instructions for adding dependency monitoring
to a running microservice.

## Migration from v0.1 to v0.2

### API Changes

| v0.1 | v0.2 | Description |
| --- | --- | --- |
| `AddDepHealth(dh => ...)` | `AddDepHealth("my-service", dh => ...)` | Required first argument `name` |
| `CreateBuilder()` | `CreateBuilder("my-service")` | Required `name` argument |
| `.Critical(true)` (optional) | `.Critical(true/false)` (required) | For each dependency |
| none | `.Label("key", "value")` | Arbitrary labels |

### Required Changes

1. Add `name` to `AddDepHealth`:

```csharp
// v0.1
builder.Services.AddDepHealth(dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url("postgres://user:pass@pg.svc:5432/mydb")
        .Critical(true))
);

// v0.2
builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url("postgres://user:pass@pg.svc:5432/mydb")
        .Critical(true))
);
```

1. Specify `.Critical()` for each dependency:

```csharp
// v0.1 — Critical is optional
.AddDependency("redis-cache", DependencyType.Redis, d => d
    .Url("redis://redis.svc:6379"))

// v0.2 — Critical is required
.AddDependency("redis-cache", DependencyType.Redis, d => d
    .Url("redis://redis.svc:6379")
    .Critical(false))
```

### New Labels in Metrics

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Update your PromQL queries and Grafana dashboards to include `name` and `critical` labels.

## Prerequisites

- .NET 8+
- ASP.NET Core (Minimal API or MVC)
- Access to dependencies (databases, cache, other services) from the service

## Step 1. Installing Dependencies

```bash
dotnet add package DepHealth.AspNetCore
```

For Entity Framework integration:

```bash
dotnet add package DepHealth.EntityFramework
```

## Step 2. Service Registration

Add dephealth to `Program.cs`:

```csharp
using DepHealth;
using DepHealth.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
    .AddDependency("payment-api", DependencyType.Http, d => d
        .Url("http://payment.svc:8080")
        .Critical(true))
);
```

## Step 3. Choosing the Mode

### Option A: Standalone Mode (Simple)

The SDK creates temporary connections for health checks:

```csharp
builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
    .AddDependency("payment-api", DependencyType.Http, d => d
        .Url("http://payment.svc:8080")
        .HttpHealthPath("/healthz")
        .Critical(true))
);
```

### Option B: Entity Framework Integration (Recommended)

The SDK uses an existing DbContext. Benefits:

- Reflects the actual ability of the service to work with the dependency
- Does not create additional load on the database
- Detects pool issues (exhaustion, leaks)

```csharp
using DepHealth.EntityFramework;

builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseNpgsql(builder.Configuration["DATABASE_URL"]));

builder.Services.AddDepHealth("my-service", dh => dh
    .CheckInterval(TimeSpan.FromSeconds(15))

    // PostgreSQL via EF Core DbContext
    .AddEntityFrameworkDependency<AppDbContext>("postgres-main",
        critical: true)

    // Redis — standalone
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))

    // HTTP — standalone only
    .AddDependency("payment-api", DependencyType.Http, d => d
        .Url("http://payment.svc:8080")
        .Critical(true))

    // gRPC — standalone only
    .AddDependency("auth-service", DependencyType.Grpc, d => d
        .Host("auth.svc")
        .Port("9090")
        .Critical(true))
);
```

### Option C: Connection String Integration

```csharp
.AddDependency("postgres-main", DependencyType.Postgres, d => d
    .ConnectionString("Host=pg.svc;Port=5432;Database=mydb;Username=user;Password=pass")
    .Critical(true))
```

## Step 4. Middleware and Endpoints

```csharp
var app = builder.Build();

// Registers /metrics (Prometheus) and /health/dependencies
app.UseDepHealth();

app.MapGet("/", () => "OK");
app.Run();
```

`UseDepHealth()` registers:

| Endpoint | Description |
| --- | --- |
| `/metrics` | Prometheus metrics (text format) |
| `/health/dependencies` | JSON status of all dependencies |

## Step 5. Verifying Functionality

### Prometheus Metrics

```bash
curl http://localhost:8080/metrics

# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
app_dependency_health{name="my-service",dependency="redis-cache",type="redis",host="redis.svc",port="6379",critical="no"} 1
```

### Dependency Status

```bash
curl http://localhost:8080/health/dependencies

{
    "status": "healthy",
    "dependencies": {
        "postgres-main": true,
        "redis-cache": true,
        "payment-api": false
    }
}
```

Status code: `200` (all healthy) or `503` (has unhealthy).

## Step 6. Accessing DepHealth from Code

```csharp
// Via DI
app.MapGet("/info", (IDepHealth depHealth) =>
{
    var health = depHealth.Health();
    return Results.Ok(health);
});
```

## Typical Configurations

### Web Service with PostgreSQL and Redis

```csharp
builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("postgres", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
);
```

### API Gateway with Upstream Services

```csharp
builder.Services.AddDepHealth("api-gateway", dh => dh
    .CheckInterval(TimeSpan.FromSeconds(10))

    .AddDependency("user-service", DependencyType.Http, d => d
        .Url("http://user-svc:8080")
        .HttpHealthPath("/healthz")
        .Critical(true))
    .AddDependency("order-service", DependencyType.Http, d => d
        .Url("http://order-svc:8080")
        .Critical(true))
    .AddDependency("auth-service", DependencyType.Grpc, d => d
        .Host("auth-svc")
        .Port("9090")
        .Critical(true))
);
```

### Event Processor with Kafka and RabbitMQ

```csharp
builder.Services.AddDepHealth("event-processor", dh => dh
    .AddDependency("kafka-main", DependencyType.Kafka, d => d
        .Url("kafka://kafka.svc:9092")
        .Critical(true))
    .AddDependency("rabbitmq", DependencyType.Amqp, d => d
        .Host("rabbitmq.svc")
        .Port("5672")
        .AmqpUsername("user")
        .AmqpPassword("pass")
        .Critical(true))
    .AddDependency("postgres", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(false))
);
```

## Troubleshooting

### Metrics Don't Appear on `/metrics`

**Check:**

1. `app.UseDepHealth()` is called in the pipeline
2. The `DepHealth.AspNetCore` package is installed
3. The application started without errors

### All Dependencies Show `0` (unhealthy)

**Check:**

1. Network accessibility of dependencies from the service's container/pod
2. DNS resolution of service names
3. Correctness of URL/host/port in the configuration
4. Timeout (default `5s`) — is it sufficient for this dependency
5. Logs: configure `Logging:LogLevel:DepHealth=Debug` in `appsettings.json`

### High Latency for PostgreSQL Checks

**Cause**: Standalone mode creates a new connection each time.

**Solution**: Use Entity Framework integration or connection string
with pooling.

### gRPC: `DeadlineExceeded` Error

**Check:**

1. The gRPC service is accessible at the specified address
2. The service implements `grpc.health.v1.Health/Check`
3. For gRPC use `Host()` + `Port()`, not `Url()`
4. If TLS is needed: `.GrpcTls(true)`

### Kafka: "unsupported URL scheme" Error

**Kafka URL requires a scheme**: `kafka://host:port`

```csharp
.AddDependency("kafka", DependencyType.Kafka, d => d
    .Url("kafka://kafka.svc:9092")
    .Critical(false))
```

### AMQP: RabbitMQ Connection Error

**Use explicit parameters:**

```csharp
.AddDependency("rabbitmq", DependencyType.Amqp, d => d
    .Host("rabbitmq.svc")
    .Port("5672")
    .AmqpUsername("user")
    .AmqpPassword("pass")
    .AmqpVhost("/")
    .Critical(false))
```

### Dependency Naming

Names must follow these rules:

- Length: 1-63 characters
- Format: `[a-z][a-z0-9-]*` (lowercase letters, digits, hyphens)
- Starts with a letter
- Examples: `postgres-main`, `redis-cache`, `auth-service`

## Next Steps

- [Quick Start](../quickstart/csharp.md) — minimal examples
- [Specification Overview](../specification.md) — details of metric contracts and behavior

---

<a id="russian"></a>

# Руководство по интеграции dephealth в существующий .NET-сервис

Пошаговая инструкция по добавлению мониторинга зависимостей
в работающий микросервис.

## Миграция с v0.1 на v0.2

### Изменения API

| v0.1 | v0.2 | Описание |
| --- | --- | --- |
| `AddDepHealth(dh => ...)` | `AddDepHealth("my-service", dh => ...)` | Обязательный первый аргумент `name` |
| `CreateBuilder()` | `CreateBuilder("my-service")` | Обязательный аргумент `name` |
| `.Critical(true)` (необязателен) | `.Critical(true/false)` (обязателен) | Для каждой зависимости |
| нет | `.Label("key", "value")` | Произвольные метки |

### Обязательные изменения

1. Добавьте `name` в `AddDepHealth`:

```csharp
// v0.1
builder.Services.AddDepHealth(dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url("postgres://user:pass@pg.svc:5432/mydb")
        .Critical(true))
);

// v0.2
builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url("postgres://user:pass@pg.svc:5432/mydb")
        .Critical(true))
);
```

1. Укажите `.Critical()` для каждой зависимости:

```csharp
// v0.1 — Critical необязателен
.AddDependency("redis-cache", DependencyType.Redis, d => d
    .Url("redis://redis.svc:6379"))

// v0.2 — Critical обязателен
.AddDependency("redis-cache", DependencyType.Redis, d => d
    .Url("redis://redis.svc:6379")
    .Critical(false))
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

- .NET 8+
- ASP.NET Core (Minimal API или MVC)
- Доступ к зависимостям (БД, кэш, другие сервисы) из сервиса

## Шаг 1. Установка зависимостей

```bash
dotnet add package DepHealth.AspNetCore
```

Для Entity Framework интеграции:

```bash
dotnet add package DepHealth.EntityFramework
```

## Шаг 2. Регистрация сервисов

Добавьте dephealth в `Program.cs`:

```csharp
using DepHealth;
using DepHealth.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
    .AddDependency("payment-api", DependencyType.Http, d => d
        .Url("http://payment.svc:8080")
        .Critical(true))
);
```

## Шаг 3. Выбор режима

### Вариант A: Standalone-режим (простой)

SDK создаёт временные соединения для проверок:

```csharp
builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
    .AddDependency("payment-api", DependencyType.Http, d => d
        .Url("http://payment.svc:8080")
        .HttpHealthPath("/healthz")
        .Critical(true))
);
```

### Вариант B: Интеграция с Entity Framework (рекомендуется)

SDK использует существующий DbContext. Преимущества:

- Отражает реальную способность сервиса работать с зависимостью
- Не создаёт дополнительную нагрузку на БД
- Обнаруживает проблемы с пулом (исчерпание, утечки)

```csharp
using DepHealth.EntityFramework;

builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseNpgsql(builder.Configuration["DATABASE_URL"]));

builder.Services.AddDepHealth("my-service", dh => dh
    .CheckInterval(TimeSpan.FromSeconds(15))

    // PostgreSQL через EF Core DbContext
    .AddEntityFrameworkDependency<AppDbContext>("postgres-main",
        critical: true)

    // Redis — standalone
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))

    // HTTP — только standalone
    .AddDependency("payment-api", DependencyType.Http, d => d
        .Url("http://payment.svc:8080")
        .Critical(true))

    // gRPC — только standalone
    .AddDependency("auth-service", DependencyType.Grpc, d => d
        .Host("auth.svc")
        .Port("9090")
        .Critical(true))
);
```

### Вариант C: Интеграция с connection string

```csharp
.AddDependency("postgres-main", DependencyType.Postgres, d => d
    .ConnectionString("Host=pg.svc;Port=5432;Database=mydb;Username=user;Password=pass")
    .Critical(true))
```

## Шаг 4. Middleware и endpoints

```csharp
var app = builder.Build();

// Регистрирует /metrics (Prometheus) и /health/dependencies
app.UseDepHealth();

app.MapGet("/", () => "OK");
app.Run();
```

`UseDepHealth()` регистрирует:

| Endpoint | Описание |
| --- | --- |
| `/metrics` | Prometheus-метрики (text format) |
| `/health/dependencies` | JSON-статус всех зависимостей |

## Шаг 5. Проверка работоспособности

### Prometheus-метрики

```bash
curl http://localhost:8080/metrics

# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
app_dependency_health{name="my-service",dependency="redis-cache",type="redis",host="redis.svc",port="6379",critical="no"} 1
```

### Состояние зависимостей

```bash
curl http://localhost:8080/health/dependencies

{
    "status": "healthy",
    "dependencies": {
        "postgres-main": true,
        "redis-cache": true,
        "payment-api": false
    }
}
```

Статус-код: `200` (все healthy) или `503` (есть unhealthy).

## Шаг 6. Доступ к DepHealth из кода

```csharp
// Через DI
app.MapGet("/info", (IDepHealth depHealth) =>
{
    var health = depHealth.Health();
    return Results.Ok(health);
});
```

## Типичные конфигурации

### Веб-сервис с PostgreSQL и Redis

```csharp
builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("postgres", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
);
```

### API Gateway с upstream-сервисами

```csharp
builder.Services.AddDepHealth("api-gateway", dh => dh
    .CheckInterval(TimeSpan.FromSeconds(10))

    .AddDependency("user-service", DependencyType.Http, d => d
        .Url("http://user-svc:8080")
        .HttpHealthPath("/healthz")
        .Critical(true))
    .AddDependency("order-service", DependencyType.Http, d => d
        .Url("http://order-svc:8080")
        .Critical(true))
    .AddDependency("auth-service", DependencyType.Grpc, d => d
        .Host("auth-svc")
        .Port("9090")
        .Critical(true))
);
```

### Обработчик событий с Kafka и RabbitMQ

```csharp
builder.Services.AddDepHealth("event-processor", dh => dh
    .AddDependency("kafka-main", DependencyType.Kafka, d => d
        .Url("kafka://kafka.svc:9092")
        .Critical(true))
    .AddDependency("rabbitmq", DependencyType.Amqp, d => d
        .Host("rabbitmq.svc")
        .Port("5672")
        .AmqpUsername("user")
        .AmqpPassword("pass")
        .Critical(true))
    .AddDependency("postgres", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(false))
);
```

## Troubleshooting

### Метрики не появляются на `/metrics`

**Проверьте:**

1. `app.UseDepHealth()` вызван в pipeline
2. Пакет `DepHealth.AspNetCore` установлен
3. Приложение стартовало без ошибок

### Все зависимости показывают `0` (unhealthy)

**Проверьте:**

1. Сетевая доступность зависимостей из контейнера/пода сервиса
2. DNS-резолвинг имён сервисов
3. Правильность URL/host/port в конфигурации
4. Таймаут (`5s` по умолчанию) — достаточен ли для данной зависимости
5. Логи: настройте `Logging:LogLevel:DepHealth=Debug` в `appsettings.json`

### Высокая латентность проверок PostgreSQL

**Причина**: standalone-режим создаёт новое соединение каждый раз.

**Решение**: используйте Entity Framework интеграцию или connection string
с пулом.

### gRPC: ошибка `DeadlineExceeded`

**Проверьте:**

1. gRPC-сервис доступен по указанному адресу
2. Сервис реализует `grpc.health.v1.Health/Check`
3. Для gRPC используйте `Host()` + `Port()`, а не `Url()`
4. Если нужен TLS: `.GrpcTls(true)`

### Kafka: ошибка «unsupported URL scheme»

**Kafka URL нуждается в схеме**: `kafka://host:port`

```csharp
.AddDependency("kafka", DependencyType.Kafka, d => d
    .Url("kafka://kafka.svc:9092")
    .Critical(false))
```

### AMQP: ошибка подключения к RabbitMQ

**Используйте явные параметры:**

```csharp
.AddDependency("rabbitmq", DependencyType.Amqp, d => d
    .Host("rabbitmq.svc")
    .Port("5672")
    .AmqpUsername("user")
    .AmqpPassword("pass")
    .AmqpVhost("/")
    .Critical(false))
```

### Именование зависимостей

Имена должны соответствовать правилам:

- Длина: 1-63 символа
- Формат: `[a-z][a-z0-9-]*` (строчные буквы, цифры, дефисы)
- Начинается с буквы
- Примеры: `postgres-main`, `redis-cache`, `auth-service`

## Следующие шаги

- [Быстрый старт](../quickstart/csharp.md) — минимальные примеры
- [Обзор спецификации](../specification.md) — детали контрактов метрик и поведения
