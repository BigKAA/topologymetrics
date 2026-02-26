*[Русская версия](aspnetcore.ru.md)*

# ASP.NET Core Integration

The `DepHealth.AspNetCore` package provides integration with ASP.NET Core
applications. It automatically manages the `DepHealthMonitor` lifecycle
via `IHostedService` and exposes health and metrics endpoints.

## Installation

```bash
dotnet add package DepHealth.AspNetCore
```

`DepHealth.AspNetCore` transitively includes `DepHealth.Core`, so you do
not need to add it separately.

## Service Registration

Register DepHealth in the DI container using the `AddDepHealth` extension
method:

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

## Registered Services

`AddDepHealth` registers the following services in the DI container:

| Service | Description |
| --- | --- |
| `DepHealthMonitor` | Main instance, registered as singleton |
| `DepHealthHostedService` | `IHostedService` — starts on application start, stops on shutdown |
| `DepHealthHealthCheck` | `IHealthCheck` — integrates with ASP.NET Core `/health` |

## Middleware and Endpoints

`app.MapDepHealthEndpoints()` registers the following route:

| Endpoint | Description |
| --- | --- |
| `GET /health/dependencies` | JSON map of all dependency health statuses |

**Request:**

```bash
curl -s http://localhost:5000/health/dependencies | jq .
```

**Response:**

```json
{
  "postgres-main:pg.svc:5432": true,
  "redis-cache:redis.svc:6379": true,
  "auth-service:auth.svc:8080": true
}
```

## Health Check Integration

`DepHealthHealthCheck` implements `IHealthCheck` and integrates with the
standard ASP.NET Core health check infrastructure. Map it alongside other
health checks:

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

`DepHealthHealthCheck` reports `Healthy` when all endpoints are healthy,
and `Unhealthy` when one or more endpoints have failed. The response
includes the status of each dependency in the `data` field:

**Request:**

```bash
curl -s http://localhost:5000/health | jq .
```

**Response:**

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

To display full health details in the response, configure the health
check options:

```csharp
app.MapHealthChecks("/health", new HealthCheckOptions
{
    ResponseWriter = UIResponseWriter.WriteHealthCheckUIResponse
});
```

## Prometheus Metrics

DepHealth exports metrics via `prometheus-net`. To expose a Prometheus
scrape endpoint in your ASP.NET Core application, add the
`prometheus-net.AspNetCore` package:

```bash
dotnet add package prometheus-net.AspNetCore
```

Then register the middleware:

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

DepHealth metrics are available at `/metrics`:

```bash
curl -s http://localhost:5000/metrics | grep app_dependency
```

```text
app_dependency_health{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",le="0.01"} 42
```

## Configuration via appsettings.json

Use `IConfiguration` to read connection strings and tuning parameters
from `appsettings.json`:

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

Sensitive values should be stored in environment variables or
[.NET User Secrets](https://learn.microsoft.com/en-us/aspnet/core/security/app-secrets)
and referenced via `IConfiguration`:

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("postgres-main",
        builder.Configuration.GetConnectionString("Postgres")!,
        critical: true)
);
```

## Custom DepHealthMonitor

To override the default registration — for example, to integrate with an
existing connection pool — register your own singleton `DepHealthMonitor`
before calling `AddHostedService`:

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

When a custom singleton is present, `DepHealthHostedService` and
`DepHealthHealthCheck` use it automatically.

## Environment Variables

Use `Environment.GetEnvironmentVariable` or the ASP.NET Core
configuration system to read values from environment variables. The
`??` operator provides a default when the variable is not set:

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

ASP.NET Core also maps environment variables to `IConfiguration`
automatically — variables prefixed with the application name or using
double underscore as separator are available via `builder.Configuration`:

```csharp
// Environment variable: ConnectionStrings__Postgres=Host=pg.svc;...
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("postgres-main",
        builder.Configuration.GetConnectionString("Postgres")!,
        critical: true)
);
```

## See Also

- [Getting Started](getting-started.md) — installation and first example
- [Configuration](configuration.md) — all options, defaults, and environment variables
- [Health Checkers](checkers.md) — detailed guide for all 9 built-in checkers
- [Connection Pools](connection-pools.md) — integration with NpgsqlDataSource, IConnectionMultiplexer
- [API Reference](api-reference.md) — complete reference of all public classes
