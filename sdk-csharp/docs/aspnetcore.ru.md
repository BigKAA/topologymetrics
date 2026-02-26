*[English version](aspnetcore.md)*

# Интеграция с ASP.NET Core

Пакет `DepHealth.AspNetCore` обеспечивает интеграцию с приложениями
ASP.NET Core. Он автоматически управляет жизненным циклом
`DepHealthMonitor` через `IHostedService` и регистрирует эндпоинты
health-проверок и метрик.

## Установка

```bash
dotnet add package DepHealth.AspNetCore
```

`DepHealth.AspNetCore` транзитивно подключает `DepHealth.Core`, поэтому
добавлять его отдельно не нужно.

## Регистрация сервисов

Зарегистрируйте DepHealth в DI-контейнере с помощью метода расширения
`AddDepHealth`:

```csharp
using DepHealth;
using DepHealth.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("postgres-main",
        builder.Configuration["DATABASE_URL"]!,
        critical: true)
    .AddRedis("redis-cache",
        builder.Configuration["REDIS_URL"]!,
        critical: false)
    .AddHttp("auth-service", "http://auth.svc:8080",
        critical: true)
);

var app = builder.Build();
app.MapDepHealthEndpoints();
app.Run();
```

## Регистрируемые сервисы

`AddDepHealth` регистрирует в DI-контейнере следующие сервисы:

| Сервис | Описание |
| --- | --- |
| `DepHealthMonitor` | Основной экземпляр, регистрируется как singleton |
| `DepHealthHostedService` | `IHostedService` — запускается при старте приложения, останавливается при завершении |
| `DepHealthHealthCheck` | `IHealthCheck` — интегрируется со стандартным эндпоинтом ASP.NET Core `/health` |

## Middleware и эндпоинты

`app.MapDepHealthEndpoints()` регистрирует следующий маршрут:

| Эндпоинт | Описание |
| --- | --- |
| `GET /health/dependencies` | JSON-карта статусов всех зависимостей |

**Запрос:**

```bash
curl -s http://localhost:5000/health/dependencies | jq .
```

**Ответ:**

```json
{
  "postgres-main:pg.svc:5432": true,
  "redis-cache:redis.svc:6379": true,
  "auth-service:auth.svc:8080": true
}
```

## Интеграция с Health Checks

`DepHealthHealthCheck` реализует `IHealthCheck` и интегрируется со
стандартной инфраструктурой health-проверок ASP.NET Core. Зарегистрируйте
его вместе с другими проверками:

```csharp
var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("postgres-main",
        builder.Configuration["DATABASE_URL"]!,
        critical: true)
    .AddRedis("redis-cache",
        builder.Configuration["REDIS_URL"]!,
        critical: false)
);

builder.Services.AddHealthChecks();

var app = builder.Build();
app.MapDepHealthEndpoints();
app.MapHealthChecks("/health");
app.Run();
```

`DepHealthHealthCheck` возвращает `Healthy`, когда все эндпоинты исправны,
и `Unhealthy`, если один или несколько эндпоинтов недоступны. Ответ
содержит статус каждой зависимости в поле `data`:

**Запрос:**

```bash
curl -s http://localhost:5000/health | jq .
```

**Ответ:**

```json
{
  "status": "Healthy",
  "results": {
    "dephealth": {
      "status": "Healthy",
      "data": {
        "postgres-main:pg.svc:5432": "healthy",
        "redis-cache:redis.svc:6379": "healthy"
      }
    }
  }
}
```

Для отображения полных деталей в ответе настройте параметры health check:

```csharp
app.MapHealthChecks("/health", new HealthCheckOptions
{
    ResponseWriter = UIResponseWriter.WriteHealthCheckUIResponse
});
```

## Метрики Prometheus

DepHealth экспортирует метрики через `prometheus-net`. Для регистрации
эндпоинта Prometheus в ASP.NET Core добавьте пакет
`prometheus-net.AspNetCore`:

```bash
dotnet add package prometheus-net.AspNetCore
```

Затем подключите middleware:

```csharp
using Prometheus;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("postgres-main",
        builder.Configuration["DATABASE_URL"]!,
        critical: true)
);

var app = builder.Build();
app.UseHttpMetrics();
app.MapDepHealthEndpoints();
app.MapMetrics();
app.Run();
```

Метрики DepHealth доступны на `/metrics`:

```bash
curl -s http://localhost:5000/metrics | grep app_dependency
```

```text
app_dependency_health{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",le="0.01"} 42
```

## Конфигурация через appsettings.json

Используйте `IConfiguration` для чтения строк подключения и параметров
из `appsettings.json`:

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .WithCheckInterval(TimeSpan.FromSeconds(
        builder.Configuration.GetValue<int>("DepHealth:Interval", 15)))
    .AddPostgres("postgres-main",
        builder.Configuration["ConnectionStrings:Postgres"]!,
        critical: true)
);
```

`appsettings.json`:

```json
{
  "DepHealth": {
    "Interval": 30
  },
  "ConnectionStrings": {
    "Postgres": "Host=pg.svc;Port=5432;Database=mydb;Username=user;Password=pass"
  }
}
```

Чувствительные данные следует хранить в переменных окружения или
[.NET User Secrets](https://learn.microsoft.com/ru-ru/aspnet/core/security/app-secrets)
и читать через `IConfiguration`:

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("postgres-main",
        builder.Configuration.GetConnectionString("Postgres")!,
        critical: true)
);
```

## Собственный DepHealthMonitor

Для переопределения стандартной регистрации — например, для интеграции
с существующим пулом соединений — зарегистрируйте собственный singleton
`DepHealthMonitor` до вызова `AddHostedService`:

```csharp
builder.Services.AddSingleton(sp =>
{
    var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
        .AddPostgres("db", connStr, critical: true)
        .Build();
    return dh;
});
builder.Services.AddHostedService<DepHealthHostedService>();
```

При наличии пользовательского singleton `DepHealthHostedService` и
`DepHealthHealthCheck` автоматически используют его.

## Переменные окружения

Используйте `Environment.GetEnvironmentVariable` или систему конфигурации
ASP.NET Core для чтения значений из переменных окружения. Оператор `??`
задаёт значение по умолчанию при отсутствии переменной:

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("postgres-main",
        Environment.GetEnvironmentVariable("DATABASE_URL")!,
        critical: true)
    .AddHttp("auth-service",
        Environment.GetEnvironmentVariable("AUTH_SERVICE_URL") ?? "http://auth.svc:8080",
        critical: true)
);
```

ASP.NET Core также автоматически отображает переменные окружения в
`IConfiguration` — переменные, использующие двойное подчёркивание в
качестве разделителя, доступны через `builder.Configuration`:

```csharp
// Переменная окружения: ConnectionStrings__Postgres=Host=pg.svc;...
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("postgres-main",
        builder.Configuration.GetConnectionString("Postgres")!,
        critical: true)
);
```

## Смотрите также

- [Начало работы](getting-started.ru.md) — установка и первый пример
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и переменные окружения
- [Проверки зависимостей](checkers.ru.md) — подробное руководство по всем 9 встроенным чекерам
- [Пулы соединений](connection-pools.ru.md) — интеграция с NpgsqlDataSource, IConnectionMultiplexer
- [Справочник API](api-reference.ru.md) — полный справочник всех публичных классов
