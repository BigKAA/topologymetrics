*[Русская версия](csharp.ru.md)*

# Quick Start: C# SDK

Guide to integrating dephealth into a .NET service in just a few minutes.

## Installation

Core package:

```bash
dotnet add package DepHealth.Core
```

ASP.NET Core integration (includes Core):

```bash
dotnet add package DepHealth.AspNetCore
```

Entity Framework integration (connection pool):

```bash
dotnet add package DepHealth.EntityFramework
```

## Minimal Example

Adding a single HTTP dependency with metrics export (ASP.NET Minimal API):

```csharp
using DepHealth;
using DepHealth.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("payment-api", DependencyType.Http, d => d
        .Url("http://payment.svc:8080")
        .Critical(true))
);

var app = builder.Build();

app.UseDepHealth();          // Registers /metrics and /health/dependencies
app.MapGet("/", () => "OK");
app.Run();
```

After starting, `/metrics` will expose:

```text
app_dependency_health{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Multiple Dependencies

```csharp
builder.Services.AddDepHealth("my-service", dh => dh
    // Global settings
    .CheckInterval(TimeSpan.FromSeconds(30))
    .Timeout(TimeSpan.FromSeconds(3))

    // PostgreSQL — standalone check (new connection)
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url("postgres://user:pass@pg.svc:5432/mydb")
        .Critical(true))

    // Redis — standalone check
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url("redis://:password@redis.svc:6379/0")
        .Critical(false))

    // HTTP service
    .AddDependency("auth-service", DependencyType.Http, d => d
        .Url("http://auth.svc:8080")
        .HttpHealthPath("/healthz")
        .Critical(true))

    // gRPC service
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

## Custom Labels

Add custom labels via `.Label()`:

```csharp
.AddDependency("postgres-main", DependencyType.Postgres, d => d
    .Url("postgres://user:pass@pg.svc:5432/mydb")
    .Critical(true)
    .Label("role", "primary")
    .Label("shard", "eu-west"))
```

Result in metrics:

```text
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",role="primary",shard="eu-west"} 1
```

## Connection Pool Integration

Preferred mode: SDK uses the service's existing connection pool instead of creating new connections.

### PostgreSQL via Entity Framework

```csharp
using DepHealth.EntityFramework;

builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseNpgsql(connectionString));

builder.Services.AddDepHealth("my-service", dh => dh
    .AddEntityFrameworkDependency<AppDbContext>("postgres-main",
        critical: true)
);
```

### PostgreSQL via connection string

```csharp
.AddDependency("postgres-main", DependencyType.Postgres, d => d
    .ConnectionString("Host=pg.svc;Port=5432;Database=mydb;Username=user;Password=pass")
    .Critical(true))
```

## ASP.NET Core Integration

### Minimal API

```csharp
var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", dh => dh
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
# Prometheus metrics
GET /metrics

# Dependency status
GET /health/dependencies

# Response:
{
    "status": "healthy",
    "dependencies": {
        "postgres-main": true,
        "redis-cache": true,
        "auth-service": false
    }
}
```

Status code: `200` (all healthy) or `503` (some unhealthy).

## Global Options

```csharp
builder.Services.AddDepHealth("my-service", dh => dh
    // Check interval (default 15s)
    .CheckInterval(TimeSpan.FromSeconds(30))

    // Timeout per check (default 5s)
    .Timeout(TimeSpan.FromSeconds(3))

    // ...dependencies
);
```

## Dependency Options

Each dependency can override global settings:

```csharp
.AddDependency("slow-service", DependencyType.Http, d => d
    .Url("http://slow.svc:8080")
    .HttpHealthPath("/ready")                    // health check path
    .HttpTls(true)                               // HTTPS
    .HttpTlsSkipVerify(true)                     // skip certificate verification
    .Interval(TimeSpan.FromSeconds(60))          // custom interval
    .Timeout(TimeSpan.FromSeconds(10))           // custom timeout
    .Critical(true))                             // critical dependency
```

## Configuration via Environment Variables

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Application name (overridden by API) | `my-service` |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality | `yes` / `no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label | `primary` |

`<DEP>` — dependency name in uppercase, dashes replaced with `_`.

Examples:

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
```

Priority: API values > environment variables.

## Behavior When Required Parameters Are Missing

| Situation | Behavior |
| --- | --- |
| `name` not specified and no `DEPHEALTH_NAME` | Error at creation: `missing name` |
| `.Critical()` not specified for dependency | Error at creation: `missing critical` |
| Invalid label name | Error at creation: `invalid label name` |
| Label conflicts with required label | Error at creation: `reserved label` |

## Checking Dependency Status

```csharp
// Via DI
var depHealth = app.Services.GetRequiredService<IDepHealth>();

var health = depHealth.Health();
// Dictionary<string, bool>:
// {"postgres-main": true, "redis-cache": true, "auth-service": false}

bool allHealthy = health.Values.All(v => v);
```

## Metrics Export

dephealth exports two Prometheus metrics via prometheus-net:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = available, `0` = unavailable |
| `app_dependency_latency_seconds` | Histogram | Check latency (seconds) |

Labels: `name`, `dependency`, `type`, `host`, `port`, `critical`.

## Supported Dependency Types

| DependencyType | Type | Check Method |
| --- | --- | --- |
| `Http` | `http` | HTTP GET to health endpoint, expecting 2xx |
| `Grpc` | `grpc` | gRPC Health Check Protocol |
| `Tcp` | `tcp` | TCP connection establishment |
| `Postgres` | `postgres` | `SELECT 1` via Npgsql |
| `MySql` | `mysql` | `SELECT 1` via MySqlConnector |
| `Redis` | `redis` | `PING` command via StackExchange.Redis |
| `Amqp` | `amqp` | RabbitMQ connection check |
| `Kafka` | `kafka` | Metadata request via Confluent.Kafka |

## Default Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `CheckInterval` | 15s | Interval between checks |
| `Timeout` | 5s | Timeout per check |
| `FailureThreshold` | 1 | Number of failures before unhealthy |
| `SuccessThreshold` | 1 | Number of successes before healthy |

## Next Steps

- [Integration Guide](../migration/csharp.md) — step-by-step integration
  with an existing service
- [Specification Overview](../specification.md) — details on metrics contracts and behavior
