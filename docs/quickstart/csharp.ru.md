*[English version](csharp.md)*

# Быстрый старт: C# SDK

Руководство по подключению dephealth к .NET-сервису за несколько минут.

## Установка

Core-пакет:

```bash
dotnet add package DepHealth.Core
```

ASP.NET Core интеграция (включает Core):

```bash
dotnet add package DepHealth.AspNetCore
```

Entity Framework интеграция (connection pool):

```bash
dotnet add package DepHealth.EntityFramework
```

## Минимальный пример

Подключение одной HTTP-зависимости с экспортом метрик (ASP.NET Minimal API):

```csharp
using DepHealth;
using DepHealth.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddDependency("payment-api", DependencyType.Http, d => d
        .Url("http://payment.svc:8080")
        .Critical(true))
);

var app = builder.Build();

app.UseDepHealth();          // Регистрирует /metrics и /health/dependencies
app.MapGet("/", () => "OK");
app.Run();
```

После запуска на `/metrics` появятся метрики:

```text
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
app_dependency_status{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",status="healthy"} 1
app_dependency_status_detail{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",detail=""} 1
```

## Несколько зависимостей

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    // Глобальные настройки
    .CheckInterval(TimeSpan.FromSeconds(30))
    .Timeout(TimeSpan.FromSeconds(3))

    // PostgreSQL — standalone check (новое соединение)
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url("postgres://user:pass@pg.svc:5432/mydb")
        .Critical(true))

    // Redis — standalone check
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url("redis://:password@redis.svc:6379/0")
        .Critical(false))

    // HTTP-сервис
    .AddDependency("auth-service", DependencyType.Http, d => d
        .Url("http://auth.svc:8080")
        .HttpHealthPath("/healthz")
        .Critical(true))

    // gRPC-сервис
    .AddDependency("user-service", DependencyType.Grpc, d => d
        .Host("user.svc")
        .Port("9090")
        .Critical(false))

    // RabbitMQ
    .AddDependency("rabbitmq", DependencyType.Amqp, d => d
        .Host("rabbitmq.svc")
        .Port("5672")
        .AmqpUsername("user")
        .AmqpPassword("pass")
        .AmqpVhost("/")
        .Critical(false))

    // Kafka
    .AddDependency("kafka", DependencyType.Kafka, d => d
        .Url("kafka://kafka.svc:9092")
        .Critical(false))
);
```

## Произвольные метки

Добавляйте произвольные метки через `.Label()`:

```csharp
.AddDependency("postgres-main", DependencyType.Postgres, d => d
    .Url("postgres://user:pass@pg.svc:5432/mydb")
    .Critical(true)
    .Label("role", "primary")
    .Label("shard", "eu-west"))
```

Результат в метриках:

```text
app_dependency_health{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",role="primary",shard="eu-west"} 1
```

## Интеграция с connection pool

Предпочтительный режим: SDK использует существующий connection pool
сервиса вместо создания новых соединений.

### PostgreSQL через Entity Framework

```csharp
using DepHealth.EntityFramework;

builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseNpgsql(connectionString));

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddEntityFrameworkDependency<AppDbContext>("postgres-main",
        critical: true)
);
```

### PostgreSQL через connection string

```csharp
.AddDependency("postgres-main", DependencyType.Postgres, d => d
    .ConnectionString("Host=pg.svc;Port=5432;Database=mydb;Username=user;Password=pass")
    .Critical(true))
```

## ASP.NET Core интеграция

### Minimal API

```csharp
var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
    .AddDependency("auth-service", DependencyType.Http, d => d
        .Url("http://auth.svc:8080")
        .HttpHealthPath("/healthz")
        .Critical(true))
);

var app = builder.Build();

app.UseDepHealth();  // Prometheus /metrics + /health/dependencies

app.MapGet("/", () => "OK");
app.Run();
```

### Endpoints

```bash
# Prometheus-метрики
GET /metrics

# Состояние зависимостей
GET /health/dependencies

# Ответ:
{
    "status": "healthy",
    "dependencies": {
        "postgres-main": true,
        "redis-cache": true,
        "auth-service": false
    }
}
```

Статус-код: `200` (все healthy) или `503` (есть unhealthy).

## Глобальные опции

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    // Интервал проверки (по умолчанию 15s)
    .CheckInterval(TimeSpan.FromSeconds(30))

    // Таймаут каждой проверки (по умолчанию 5s)
    .Timeout(TimeSpan.FromSeconds(3))

    // ...зависимости
);
```

## Опции зависимостей

Каждая зависимость может переопределить глобальные настройки:

```csharp
.AddDependency("slow-service", DependencyType.Http, d => d
    .Url("http://slow.svc:8080")
    .HttpHealthPath("/ready")                    // путь health check
    .HttpTls(true)                               // HTTPS
    .HttpTlsSkipVerify(true)                     // пропустить проверку сертификата
    .Interval(TimeSpan.FromSeconds(60))          // свой интервал
    .Timeout(TimeSpan.FromSeconds(10))           // свой таймаут
    .Critical(true))                             // критическая зависимость
```

## Аутентификация

HTTP и gRPC чекеры поддерживают аутентификацию. Для каждой зависимости
допускается только один метод — смешивание вызывает ошибку валидации.

### HTTP Bearer Token

```csharp
.AddDependency("secure-api", DependencyType.Http, d => d
    .Url("http://api.svc:8080")
    .Critical(true)
    .HttpBearerToken("eyJhbG..."))
```

### HTTP Basic Auth

```csharp
.AddDependency("secure-api", DependencyType.Http, d => d
    .Url("http://api.svc:8080")
    .Critical(true)
    .HttpBasicAuth("admin", "secret"))
```

### HTTP произвольные заголовки

```csharp
.AddDependency("secure-api", DependencyType.Http, d => d
    .Url("http://api.svc:8080")
    .Critical(true)
    .HttpHeaders(new Dictionary<string, string>
    {
        ["X-API-Key"] = "my-key",
    }))
```

### gRPC Bearer Token

```csharp
.AddDependency("grpc-backend", DependencyType.Grpc, d => d
    .Host("backend.svc")
    .Port(9090)
    .Critical(true)
    .GrpcBearerToken("eyJhbG..."))
```

### gRPC произвольные метаданные

```csharp
.AddDependency("grpc-backend", DependencyType.Grpc, d => d
    .Host("backend.svc")
    .Port(9090)
    .Critical(true)
    .GrpcMetadata(new Dictionary<string, string>
    {
        ["x-api-key"] = "my-key",
    }))
```

### Классификация ошибок аутентификации

Когда сервер возвращает ошибку аутентификации, чекер классифицирует
её как `auth_error`:

- HTTP 401/403 → `status="auth_error"`, `detail="auth_error"`
- gRPC UNAUTHENTICATED/PERMISSION_DENIED → `status="auth_error"`, `detail="auth_error"`

## Конфигурация через переменные окружения

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Имя приложения (перекрывается API) | `my-service` |
| `DEPHEALTH_GROUP` | Логическая группа (метка `group`) | `my-team` |
| `DEPHEALTH_<DEP>_CRITICAL` | Критичность зависимости | `yes` / `no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Произвольная метка | `primary` |

`<DEP>` — имя зависимости в верхнем регистре, дефисы заменены на `_`.

### Полный пример с переменными окружения

```bash
# URL подключений
export DATABASE_URL=postgres://user:pass@pg.svc:5432/mydb
export REDIS_URL=redis://:password@redis.svc:6379/0

# Токены аутентификации
export API_BEARER_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
export GRPC_BEARER_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# Конфигурация зависимостей
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

### Использование переменных окружения в коде

```csharp
using DepHealth;
using DepHealth.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

// Чтение конфигурации из переменных окружения
var dbUrl = Environment.GetEnvironmentVariable("DATABASE_URL");
var redisUrl = Environment.GetEnvironmentVariable("REDIS_URL");
var apiToken = Environment.GetEnvironmentVariable("API_BEARER_TOKEN");
var grpcToken = Environment.GetEnvironmentVariable("GRPC_BEARER_TOKEN");

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .CheckInterval(TimeSpan.FromSeconds(15))

    // PostgreSQL из переменной окружения
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(dbUrl!)
        .Critical(true))

    // Redis из переменной окружения
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(redisUrl!)
        .Critical(false))

    // HTTP с Bearer-токеном из переменной окружения
    .AddDependency("api-service", DependencyType.Http, d => d
        .Url("http://api.svc:8080")
        .HttpBearerToken(apiToken!)
        .Critical(true))

    // gRPC с Bearer-токеном из переменной окружения
    .AddDependency("grpc-backend", DependencyType.Grpc, d => d
        .Host("backend.svc")
        .Port("9090")
        .GrpcBearerToken(grpcToken!)
        .Critical(true))
);

var app = builder.Build();
app.UseDepHealth();
app.Run();
```

### ASP.NET Core конфигурация через appsettings.json

ASP.NET Core автоматически разрешает переменные окружения в конфигурации.
Создайте `appsettings.json`:

```json
{
  "DepHealth": {
    "Name": "my-service",
    "Group": "my-team",
    "Dependencies": {
      "postgres-main": {
        "Type": "Postgres",
        "Url": "placeholder-will-be-overridden",
        "Critical": true
      }
    }
  }
}
```

Затем используйте Configuration:

```csharp
var builder = WebApplication.CreateBuilder(args);

var dbUrl = builder.Configuration["DATABASE_URL"];
var redisUrl = builder.Configuration["REDIS_URL"];
var apiToken = builder.Configuration["API_BEARER_TOKEN"];

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(dbUrl!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(redisUrl!)
        .Critical(false))
    .AddDependency("api-service", DependencyType.Http, d => d
        .Url("http://api.svc:8080")
        .HttpBearerToken(apiToken!)
        .Critical(true))
);
```

Приоритет: значения из API > переменные окружения.

## Поведение при отсутствии обязательных параметров

| Ситуация | Поведение |
| --- | --- |
| Не указан `name` и нет `DEPHEALTH_NAME` | Ошибка при создании: `missing name` |
| Не указан `group` и нет `DEPHEALTH_GROUP` | Ошибка при создании: `missing group` |
| Не указан `.Critical()` для зависимости | Ошибка при создании: `missing critical` |
| Недопустимое имя метки | Ошибка при создании: `invalid label name` |
| Метка совпадает с обязательной | Ошибка при создании: `reserved label` |

## Проверка состояния зависимостей

```csharp
// Через DI
var depHealth = app.Services.GetRequiredService<IDepHealth>();

var health = depHealth.Health();
// Dictionary<string, bool>:
// {"postgres-main": true, "redis-cache": true, "auth-service": false}

bool allHealthy = health.Values.All(v => v);
```

## Детальный статус зависимостей

Метод `HealthDetails()` возвращает подробную информацию о каждом endpoint-е,
включая категорию статуса, причину сбоя, латентность и пользовательские метки:

```csharp
Dictionary<string, EndpointStatus> details = depHealth.HealthDetails();
// {"postgres-main:pg.svc:5432": EndpointStatus {
//     Dependency = "postgres-main", Type = "postgres",
//     Host = "pg.svc", Port = "5432",
//     Healthy = true, Status = "ok", Detail = "ok",
//     Latency = TimeSpan.FromMilliseconds(15),
//     LastCheckedAt = DateTimeOffset.UtcNow,
//     Critical = true, Labels = {{"role", "primary"}}
// }}

// Сериализация в JSON
var json = JsonSerializer.Serialize(details);
```

В отличие от `Health()`, который возвращает `Dictionary<string, bool>`, `HealthDetails()`
предоставляет полный объект `EndpointStatus` для каждого endpoint-а. До завершения
первой проверки `Healthy` равен `null` (неизвестно), а `Status` — `"unknown"`.

## Экспорт метрик

dephealth экспортирует четыре метрики Prometheus через prometheus-net:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = доступен, `0` = недоступен |
| `app_dependency_latency_seconds` | Histogram | Латентность проверки (секунды) |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина: напр. `http_503`, `auth_error` |

Метки: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`.
Дополнительные: `status` (на `app_dependency_status`), `detail` (на `app_dependency_status_detail`).

## Поддерживаемые типы зависимостей

| DependencyType | Тип | Метод проверки |
| --- | --- | --- |
| `Http` | `http` | HTTP GET к health endpoint, ожидание 2xx |
| `Grpc` | `grpc` | gRPC Health Check Protocol |
| `Tcp` | `tcp` | Установка TCP-соединения |
| `Postgres` | `postgres` | `SELECT 1` через Npgsql |
| `MySql` | `mysql` | `SELECT 1` через MySqlConnector |
| `Redis` | `redis` | Команда `PING` через StackExchange.Redis |
| `Amqp` | `amqp` | Проверка соединения с RabbitMQ |
| `Kafka` | `kafka` | Metadata request через Confluent.Kafka |

## Параметры по умолчанию

| Параметр | Значение | Описание |
| --- | --- | --- |
| `CheckInterval` | 15s | Интервал между проверками |
| `Timeout` | 5s | Таймаут одной проверки |
| `FailureThreshold` | 1 | Число неудач до перехода в unhealthy |
| `SuccessThreshold` | 1 | Число успехов до перехода в healthy |

## Следующие шаги

- [Руководство по интеграции](../migration/csharp.ru.md) — пошаговое подключение
  к существующему сервису
- [Обзор спецификации](../specification.ru.md) — детали контрактов метрик и поведения
