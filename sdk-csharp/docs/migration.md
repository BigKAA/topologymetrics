*[Русская версия](migration.ru.md)*

# Migration Guide

Version upgrade instructions for the C# SDK.

## v0.7.0 to v0.8.0

### New: LDAP Health Checker

v0.8.0 adds the LDAP health checker. No existing API changes — this is a
purely additive feature.

New types: `LdapChecker`, `LdapCheckMethod`, `LdapSearchScope`.

New builder method: `AddLdap()`.

See [Checkers — LDAP](checkers.md#ldap) for details.

---

## v0.6.0 to v0.7.0

### New: Status Metrics

v0.7.0 adds two new automatically exported Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason: e.g. `http_503`, `auth_error` |

**No code changes needed** — the SDK exports these metrics automatically
alongside the existing `app_dependency_health` and
`app_dependency_latency_seconds`.

#### Storage Impact

Each endpoint now produces 9 additional time series (8 for
`app_dependency_status` + 1 for `app_dependency_status_detail`). For a
service with 5 endpoints, this adds 45 series.

#### New PromQL Queries

```promql
# Status category for a dependency
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Detailed failure reason
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Alert on authentication errors
app_dependency_status{status="auth_error"} == 1
```

---

## v0.5.0 to v0.6.0

### New: Dynamic Endpoint Management

v0.6.0 adds three new methods to `DepHealthMonitor` for dynamic endpoint
management at runtime. No existing API changes — this is a fully
backward-compatible release.

| Method | Description |
| --- | --- |
| `AddEndpoint` | Add a monitored endpoint after `Start()` |
| `RemoveEndpoint` | Remove an endpoint (cancels task, deletes metrics) |
| `UpdateEndpoint` | Atomically replace an endpoint with a new one |

```csharp
using DepHealth;
using DepHealth.Checks;

// After dh.Start()...

// Add a new endpoint
dh.AddEndpoint("api-backend", DependencyType.Http, true,
    new Endpoint("backend-2.svc", "8080"),
    new HttpChecker());

// Remove an endpoint
dh.RemoveEndpoint("api-backend", "backend-2.svc", "8080");

// Replace an endpoint atomically
dh.UpdateEndpoint("api-backend", "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    new HttpChecker());
```

#### Key Behaviors

- **Thread-safe:** all three methods use `lock` and can be called from
  multiple threads. Read operations (`Health()`, `HealthDetails()`) remain
  lock-free via `ConcurrentDictionary`.
- **Idempotent:** `AddEndpoint` returns silently if the endpoint already
  exists. `RemoveEndpoint` returns silently if the endpoint is not found.
- **Global config inheritance:** dynamically added endpoints use the
  global check interval and timeout configured in the builder.
- **Metrics lifecycle:** `RemoveEndpoint` and `UpdateEndpoint` delete
  all Prometheus metrics for the old endpoint.

#### Validation

`AddEndpoint` and `UpdateEndpoint` validate inputs:

- `depName` must match `[a-z][a-z0-9-]*`, max 63 chars
- `depType` must be a defined `DependencyType` enum value
- `ep.Host` and `ep.Port` must be non-empty
- `ep.Labels` must not use reserved label names

Invalid inputs throw `ValidationException`.

#### Error Handling

```csharp
using DepHealth.Exceptions;

try
{
    dh.UpdateEndpoint("api", "old-host", "8080", newEp, checker);
}
catch (EndpointNotFoundException)
{
    // old endpoint does not exist — use AddEndpoint instead
}
catch (InvalidOperationException)
{
    // scheduler not started or already stopped
}
```

New exception: `EndpointNotFoundException`.

#### Version Update

```xml
<!-- v0.5.0 -->
<Version>0.5.0</Version>

<!-- v0.6.0 -->
<Version>0.6.0</Version>
```

---

## v0.4.1 to v0.5.0

### Breaking: Mandatory `group` Parameter

v0.5.0 adds a mandatory `group` parameter (logical grouping: team,
subsystem, project).

```csharp
// v0.4.x
builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency(...)
);

// v0.5.0
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddDependency(...)
);
```

For standalone API:

```csharp
// v0.4.x
DepHealthMonitor.CreateBuilder("my-service")

// v0.5.0
DepHealthMonitor.CreateBuilder("my-service", "my-team")
```

Alternative: set `DEPHEALTH_GROUP` environment variable (API takes
precedence).

Validation: same rules as `name` — `[a-z][a-z0-9-]*`, 1-63 chars.

---

## v0.4.0 to v0.4.1

### New: HealthDetails() API

v0.4.1 adds the `HealthDetails()` method that returns detailed status for
each endpoint. No existing API changes.

```csharp
Dictionary<string, EndpointStatus> details = dh.HealthDetails();

foreach (var (key, ep) in details)
{
    Console.WriteLine($"{key}: healthy={ep.Healthy} status={ep.Status} " +
        $"detail={ep.Detail} latency={ep.LatencyMillis:F1}ms");
}
```

`EndpointStatus` properties: `Dependency`, `Type`, `Host`, `Port`,
`Healthy` (`bool?`, `null` = unknown), `Status`, `Detail`,
`Latency`, `LastCheckedAt`, `Critical`, `Labels`.

JSON serialization uses `System.Text.Json` with snake_case naming.

---

## v0.3.x to v0.4.0

### New: Status Metrics (No Code Changes Required)

v0.4.0 adds two new automatically exported Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason: e.g. `http_503`, `auth_error` |

**No code changes needed** — the SDK exports these metrics automatically
alongside the existing `app_dependency_health` and
`app_dependency_latency_seconds`.

#### Storage Impact

Each endpoint now produces 9 additional time series (8 for
`app_dependency_status` + 1 for `app_dependency_status_detail`). For a
service with 5 endpoints, this adds 45 series.

#### New PromQL Queries

```promql
# Status category for a dependency
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Detailed failure reason
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Alert on authentication errors
app_dependency_status{status="auth_error"} == 1
```

---

## v0.1 to v0.2

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

Update your PromQL queries and Grafana dashboards to include `name` and
`critical` labels.

## See Also

- [Getting Started](getting-started.md) — installation and basic setup
- [Configuration](configuration.md) — all options, defaults, and validation
- [API Reference](api-reference.md) — complete reference of all public classes
- [Troubleshooting](troubleshooting.md) — common issues and solutions
