*[Русская версия](migration.ru.md)*

# Migration Guide

Version-by-version upgrade instructions for the dephealth Python SDK.

## v0.5.0 to v0.6.0

> This release affects **Python SDK only**. Go SDK remains at v0.7.0;
> Java SDK remains at v0.6.0; C# SDK remains at v0.5.0.

### Do I Need to Change My Code?

**No.** This is a fully backward-compatible release. All existing code
continues to work without modification.

### New Feature: Dynamic Endpoints

Three new async methods (plus sync variants) on `DependencyHealth` allow
dynamic endpoint management at runtime:

| Method | Description |
| --- | --- |
| `add_endpoint` / `add_endpoint_sync` | Add a monitored endpoint after `start()` |
| `remove_endpoint` / `remove_endpoint_sync` | Remove an endpoint (cancels task, deletes metrics) |
| `update_endpoint` / `update_endpoint_sync` | Atomically replace an endpoint with a new one |

A new exception `EndpointNotFoundError` is thrown by `update_endpoint` when
the old endpoint does not exist.

```python
from dephealth import DependencyType, Endpoint, EndpointNotFoundError
from dephealth.checks.http import HTTPChecker

# After dh.start()...

# Add a new endpoint
await dh.add_endpoint(
    "api-backend",
    DependencyType.HTTP,
    True,
    Endpoint(host="backend-2.svc", port="8080"),
    HTTPChecker(),
)

# Remove an endpoint
await dh.remove_endpoint("api-backend", "backend-2.svc", "8080")

# Replace an endpoint atomically
await dh.update_endpoint(
    "api-backend",
    "backend-1.svc", "8080",
    Endpoint(host="backend-3.svc", port="8080"),
    HTTPChecker(),
)
```

#### Synchronous Mode

For applications using the threading mode (`start_sync()`):

```python
dh.add_endpoint_sync("api-backend", DependencyType.HTTP, True,
    Endpoint(host="backend-2.svc", port="8080"), HTTPChecker())

dh.remove_endpoint_sync("api-backend", "backend-2.svc", "8080")

dh.update_endpoint_sync("api-backend", "backend-1.svc", "8080",
    Endpoint(host="backend-3.svc", port="8080"), HTTPChecker())
```

#### Key Behaviors

- **Thread-safe:** all methods use `threading.Lock` and can be called from
  multiple threads or tasks.
- **Idempotent:** `add_endpoint` returns silently if the endpoint already
  exists. `remove_endpoint` returns silently if the endpoint is not found.
- **Global config inheritance:** dynamically added endpoints use the global
  `check_interval` and `timeout` configured in the constructor.
- **Metrics lifecycle:** `remove_endpoint` and `update_endpoint` delete all
  Prometheus metrics for the old endpoint.

#### Error Handling

```python
from dephealth import EndpointNotFoundError

try:
    await dh.update_endpoint("api", "old-host", "8080", new_ep, checker)
except EndpointNotFoundError:
    # old endpoint does not exist -- use add_endpoint instead
    pass
except RuntimeError:
    # scheduler not started or already stopped
    pass
```

---

## v0.4.x to v0.5.0

See also: [cross-SDK migration guide](../../docs/migration/v042-to-v050.md)

### Breaking: mandatory `group` parameter

v0.5.0 adds a mandatory `group` parameter (logical grouping: team,
subsystem, project).

```python
# v0.4.x
dh = DependencyHealth("my-service",
    postgres_check("postgres-main", ...),
)

# v0.5.0
dh = DependencyHealth("my-service", "my-team",
    postgres_check("postgres-main", ...),
)
```

FastAPI:

```python
# v0.5.0
app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres-main", ...),
    )
)
```

Alternative: set `DEPHEALTH_GROUP` environment variable (API takes precedence).

Validation: same rules as `name` — `[a-z][a-z0-9-]*`, 1-63 chars.

---

## v0.4.0 to v0.4.1

### New: health_details() API

v0.4.1 adds the `health_details()` method that returns detailed status for each
endpoint. No existing API changes — this is a purely additive feature.

```python
details = dh.health_details()
# dict[str, EndpointStatus]

for key, ep in details.items():
    print(f"{key}: healthy={ep.healthy} status={ep.status} "
          f"detail={ep.detail} latency={ep.latency_millis():.1f}ms")
```

`EndpointStatus` fields: `dependency`, `type`, `host`, `port`,
`healthy` (`bool | None`, `None` = unknown), `status`, `detail`,
`latency`, `last_checked_at`, `critical`, `labels`.

> **Note:** `health_details()` uses per-endpoint keys (`"dep:host:port"`),
> while `health()` uses aggregated keys (`"dep"`). The `to_dict()` method
> serializes `EndpointStatus` to a JSON-compatible dict.

---

## v0.3.x to v0.4.0

### New Status Metrics (no code changes required)

v0.4.0 adds two new automatically exported Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason: e.g. `http_503`, `auth_error` |

**No code changes are needed** — the SDK exports these metrics automatically
alongside the existing `app_dependency_health` and
`app_dependency_latency_seconds`.

### Storage Impact

Each endpoint now produces 9 additional time series (8 for
`app_dependency_status` + 1 for `app_dependency_status_detail`). For a service
with 5 endpoints, this adds 45 series.

### New PromQL Queries

```promql
# Status category for a dependency
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Detailed failure reason
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Alert on authentication errors
app_dependency_status{status="auth_error"} == 1
```

For the full list of status values, see
[Specification](../../spec/).

---

## v0.1 to v0.2

### API Changes

| v0.1 | v0.2 | Description |
| --- | --- | --- |
| `DependencyHealth(...)` | `DependencyHealth("my-service", ...)` | Required first argument `name` |
| `dephealth_lifespan(...)` | `dephealth_lifespan("my-service", ...)` | Required first argument `name` |
| `critical=True` (optional) | `critical=True/False` (required) | For each factory |
| none | `labels={"key": "value"}` | Arbitrary labels |

### Required Changes

1. Add `name` as the first argument:

   ```python
   # v0.1
   dh = DependencyHealth(
       postgres_check("postgres-main", url="postgresql://..."),
   )

   # v0.2
   dh = DependencyHealth("my-service",
       postgres_check("postgres-main", url="postgresql://...", critical=True),
   )
   ```

2. Specify `critical` for each dependency:

   ```python
   # v0.1 -- critical is optional
   redis_check("redis-cache", url="redis://redis.svc:6379")

   # v0.2 -- critical is required
   redis_check("redis-cache", url="redis://redis.svc:6379", critical=False)
   ```

3. Update `dephealth_lifespan` (FastAPI):

   ```python
   # v0.1
   app = FastAPI(
       lifespan=dephealth_lifespan(
           http_check("api", url="http://api:8080"),
       )
   )

   # v0.2
   app = FastAPI(
       lifespan=dephealth_lifespan("my-service",
           http_check("api", url="http://api:8080", critical=True),
       )
   )
   ```

### New Labels in Metrics

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Update PromQL queries and Grafana dashboards to include `name` and `critical`
labels.

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Configuration](configuration.md) — all options and defaults
- [API Reference](api-reference.md) — complete reference of all public classes
