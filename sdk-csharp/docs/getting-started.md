*[Русская версия](getting-started.ru.md)*

# Getting Started

This guide covers installation, basic setup, and your first health check
with the dephealth C# SDK.

## Prerequisites

- .NET 8 or later
- ASP.NET Core (Minimal API or MVC)
- A running dependency to monitor (PostgreSQL, Redis, HTTP service, etc.)

## Installation

Core package (programmatic API):

```bash
dotnet add package DepHealth.Core
```

ASP.NET Core integration (includes Core):

```bash
dotnet add package DepHealth.AspNetCore
```

Entity Framework integration:

```bash
dotnet add package DepHealth.EntityFramework
```

## Minimal Example

Monitor a single HTTP dependency and expose Prometheus metrics:

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddHttp("payment-api", "http://payment.svc:8080", critical: true)
    .Build();

dh.Start();

// Metrics are available via prometheus-net at /metrics

// Graceful shutdown
Console.CancelKeyPress += (_, _) => dh.Stop();
```

After startup, Prometheus metrics appear at `/metrics`:

```text
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Key Concepts

### Name and Group

Every `DepHealthMonitor` instance requires two identifiers:

- **name** — unique application name (e.g., `"my-service"`)
- **group** — logical group the service belongs to (e.g., `"my-team"`, `"payments"`)

Both appear as labels in all exported metrics. Validation rules:
`[a-z][a-z0-9-]*`, 1-63 characters.

If not passed as arguments, the SDK reads `DEPHEALTH_NAME` and
`DEPHEALTH_GROUP` environment variables as fallback.

### Dependencies

Each dependency is registered via the builder's `Add*()` methods
with a `DependencyType`:

| DependencyType | Description |
| --- | --- |
| `Http` | HTTP service |
| `Grpc` | gRPC service |
| `Tcp` | TCP endpoint |
| `Postgres` | PostgreSQL database |
| `MySql` | MySQL database |
| `Redis` | Redis server |
| `Amqp` | RabbitMQ (AMQP broker) |
| `Kafka` | Apache Kafka broker |
| `Ldap` | LDAP directory server |

Each dependency requires:

- A **name** (first argument) — identifies the dependency in metrics
- **Endpoint** — specified via URL string or `host` + `port`
- **Critical** flag — `critical: true` or `critical: false`

### Lifecycle

1. **Create** — `DepHealthMonitor.CreateBuilder(...).Build()`
2. **Start** — `dh.Start()` launches periodic health checks
3. **Run** — checks execute at the configured interval (default 15s)
4. **Stop** — `dh.Stop()` stops checks and disposes resources

## Multiple Dependencies

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Global settings
    .WithCheckInterval(TimeSpan.FromSeconds(30))
    .WithCheckTimeout(TimeSpan.FromSeconds(3))

    // PostgreSQL
    .AddPostgres("postgres-main",
        Environment.GetEnvironmentVariable("DATABASE_URL")!,
        critical: true)

    // Redis
    .AddRedis("redis-cache",
        Environment.GetEnvironmentVariable("REDIS_URL")!,
        critical: false)

    // HTTP service
    .AddHttp("auth-service", "http://auth.svc:8080",
        healthPath: "/healthz",
        critical: true)

    // gRPC service
    .AddGrpc("user-service", "user.svc", "9090",
        critical: false)

    .Build();
```

## Checking Health Status

### Simple Status

```csharp
Dictionary<string, bool> health = dh.Health();
// {"postgres-main": true, "redis-cache": true}

// Use for readiness probe
bool allHealthy = health.Values.All(v => v);
```

### Detailed Status

```csharp
Dictionary<string, EndpointStatus> details = dh.HealthDetails();
foreach (var (key, ep) in details)
{
    Console.WriteLine($"{key}: healthy={ep.Healthy} status={ep.Status} " +
        $"latency={ep.LatencyMillis:F1}ms");
}
```

`HealthDetails()` returns an `EndpointStatus` object with health state,
status category, latency, timestamps, and custom labels. Before the first
check completes, `Healthy` is `null` and `Status` is `"unknown"`.

## Next Steps

- [Checkers](checkers.md) — detailed guide for all 9 built-in checkers
- [Configuration](configuration.md) — all options, defaults, and environment variables
- [Connection Pools](connection-pools.md) — integration with existing connection pools
- [ASP.NET Core Integration](aspnetcore.md) — DI registration, hosted service, health endpoints
- [Entity Framework](entity-framework.md) — DbContext-based health checks
- [Authentication](authentication.md) — auth for HTTP, gRPC, and database checkers
- [Metrics](metrics.md) — Prometheus metrics reference and PromQL examples
- [API Reference](api-reference.md) — complete reference of all public classes
- [Troubleshooting](troubleshooting.md) — common issues and solutions
- [Migration Guide](migration.md) — version upgrade instructions
- [Code Style](code-style.md) — C# code conventions for this project
- [Examples](examples/) — complete runnable examples
