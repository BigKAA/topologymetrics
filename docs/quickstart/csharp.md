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

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
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
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
app_dependency_status{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",status="healthy"} 1
app_dependency_status_detail{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",detail=""} 1
```

## Multiple Dependencies

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
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
app_dependency_health{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",role="primary",shard="eu-west"} 1
```

## Connection Pool Integration

Preferred mode: SDK uses the service's existing connection pool instead of creating new connections.

### PostgreSQL via Entity Framework

```csharp
using DepHealth.EntityFramework;

builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseNpgsql(connectionString));

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
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
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
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

## Authentication

HTTP and gRPC checkers support authentication. Only one auth method
per dependency is allowed — mixing methods causes a validation error.

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

### HTTP Custom Headers

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

### gRPC Custom Metadata

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

### Auth Error Classification

When a server responds with an authentication error, the checker
classifies it as `auth_error`:

- HTTP 401/403 → `status="auth_error"`, `detail="auth_error"`
- gRPC UNAUTHENTICATED/PERMISSION_DENIED → `status="auth_error"`, `detail="auth_error"`

## Configuration via Environment Variables

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Application name (overridden by API) | `my-service` |
| `DEPHEALTH_GROUP` | Logical group (`group` label) | `my-team` |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality | `yes` / `no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label | `primary` |

`<DEP>` — dependency name in uppercase, dashes replaced with `_`.

### Full Example with Environment Variables

```bash
# Connection URLs
export DATABASE_URL=postgres://user:pass@pg.svc:5432/mydb
export REDIS_URL=redis://:password@redis.svc:6379/0

# Authentication tokens
export API_BEARER_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
export GRPC_BEARER_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# Dependency configuration
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

### Using Environment Variables in Code

```csharp
using DepHealth;
using DepHealth.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

// Read configuration from environment
var dbUrl = Environment.GetEnvironmentVariable("DATABASE_URL");
var redisUrl = Environment.GetEnvironmentVariable("REDIS_URL");
var apiToken = Environment.GetEnvironmentVariable("API_BEARER_TOKEN");
var grpcToken = Environment.GetEnvironmentVariable("GRPC_BEARER_TOKEN");

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .CheckInterval(TimeSpan.FromSeconds(15))

    // PostgreSQL from env var
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(dbUrl!)
        .Critical(true))

    // Redis from env var
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(redisUrl!)
        .Critical(false))

    // HTTP with Bearer token from env var
    .AddDependency("api-service", DependencyType.Http, d => d
        .Url("http://api.svc:8080")
        .HttpBearerToken(apiToken!)
        .Critical(true))

    // gRPC with Bearer token from env var
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

### ASP.NET Core Configuration with appsettings.json

ASP.NET Core automatically resolves environment variables in configuration.
Create `appsettings.json`:

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

Then use Configuration:

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

Priority: API values > environment variables.

## Behavior When Required Parameters Are Missing

| Situation | Behavior |
| --- | --- |
| `name` not specified and no `DEPHEALTH_NAME` | Error at creation: `missing name` |
| No `group` specified and no `DEPHEALTH_GROUP` | Error on creation: `missing group` |
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

## Detailed Health Status

The `HealthDetails()` method returns detailed information about each endpoint,
including status category, failure detail, latency, and custom labels:

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

// Serialize to JSON
var json = JsonSerializer.Serialize(details);
```

Unlike `Health()` which returns `Dictionary<string, bool>`, `HealthDetails()`
provides the full `EndpointStatus` object for each endpoint. Before the first
check completes, `Healthy` is `null` (unknown) and `Status` is `"unknown"`.

## Metrics Export

dephealth exports four Prometheus metrics via prometheus-net:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = available, `0` = unavailable |
| `app_dependency_latency_seconds` | Histogram | Check latency (seconds) |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed reason: e.g. `http_503`, `auth_error` |

Labels: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`.
Additional: `status` (on `app_dependency_status`), `detail` (on `app_dependency_status_detail`).

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
