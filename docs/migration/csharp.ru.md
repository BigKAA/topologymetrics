*[English version](csharp.md)*

# Руководство по интеграции dephealth в существующий .NET-сервис

Пошаговая инструкция по добавлению мониторинга зависимостей
в работающий микросервис.

## Миграция на v0.4.1

### Новое: HealthDetails() API

В v0.4.1 добавлен метод `HealthDetails()`, возвращающий детальный статус каждого
endpoint-а. Изменений в существующем API нет — это чисто аддитивная функция.

```csharp
Dictionary<string, EndpointStatus> details = depHealth.HealthDetails();

foreach (var (key, ep) in details)
{
    Console.WriteLine($"{key}: healthy={ep.Healthy} status={ep.Status} " +
        $"detail={ep.Detail} latency={ep.LatencyMillis:F1}ms");
}
```

Свойства `EndpointStatus`: `Dependency`, `Type`, `Host`, `Port`,
`Healthy` (`bool?`, `null` = неизвестно), `Status`, `Detail`,
`Latency`, `LastCheckedAt`, `Critical`, `Labels`.

JSON-сериализация использует `System.Text.Json` с именованием snake_case.

---

## Миграция на v0.4.0

### Новые метрики статуса (изменения кода не требуются)

v0.4.0 добавляет две новые автоматически экспортируемые метрики Prometheus:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя: напр. `http_503`, `auth_error` |

**Изменения кода не требуются** — SDK экспортирует эти метрики автоматически наряду с существующими `app_dependency_health` и `app_dependency_latency_seconds`.

### Влияние на хранилище

Каждый endpoint теперь создаёт 9 дополнительных временных рядов (8 для `app_dependency_status` + 1 для `app_dependency_status_detail`). Для сервиса с 5 endpoint-ами это добавляет 45 рядов.

### Новые PromQL-запросы

```promql
# Категория статуса зависимости
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Детальная причина сбоя
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Алерт на ошибки аутентификации
app_dependency_status{status="auth_error"} == 1
```

Полный список значений статуса см. в [Спецификация — Метрики статуса](../specification.ru.md).

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

- [Быстрый старт](../quickstart/csharp.ru.md) — минимальные примеры
- [Обзор спецификации](../specification.ru.md) — детали контрактов метрик и поведения
