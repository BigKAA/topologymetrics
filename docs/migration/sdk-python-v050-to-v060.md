*[Русская версия](sdk-python-v050-to-v060.ru.md)*

# Python SDK: Migration from v0.5.0 to v0.6.0

Migration guide for the Python SDK v0.6.0 release.

> This release affects **Python SDK only**. Go SDK remains at v0.7.0;
> Java SDK remains at v0.6.0; C# SDK remains at v0.5.0.

## What Changed

Three new async methods (plus sync variants) on `DependencyHealth` allow
dynamic endpoint management at runtime:

| Method | Description |
| --- | --- |
| `add_endpoint` / `add_endpoint_sync` | Add a monitored endpoint after `start()` |
| `remove_endpoint` / `remove_endpoint_sync` | Remove an endpoint (cancels task, deletes metrics) |
| `update_endpoint` / `update_endpoint_sync` | Atomically replace an endpoint with a new one |

A new exception `EndpointNotFoundError` is thrown by `update_endpoint` when
the old endpoint does not exist.

---

## Do I Need to Change My Code?

**No.** This is a fully backward-compatible release. All existing code
continues to work without modification.

---

## New Feature: Dynamic Endpoints

Prior to v0.6.0, all dependencies had to be registered upfront via factory
functions (`http_check()`, `postgres_check()`, etc.) passed to the
`DependencyHealth(...)` constructor. Once `start()` was called, the set of
monitored endpoints was frozen.

Starting with v0.6.0, you can add, remove, and update endpoints on a running
`DependencyHealth` instance:

```python
from dephealth import DependencyType, Endpoint, EndpointNotFoundError, HealthChecker
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

### Synchronous Mode

For applications using the threading mode (`start_sync()`):

```python
dh.add_endpoint_sync("api-backend", DependencyType.HTTP, True,
    Endpoint(host="backend-2.svc", port="8080"), HTTPChecker())

dh.remove_endpoint_sync("api-backend", "backend-2.svc", "8080")

dh.update_endpoint_sync("api-backend", "backend-1.svc", "8080",
    Endpoint(host="backend-3.svc", port="8080"), HTTPChecker())
```

### Key Behaviors

- **Thread-safe:** all methods use `threading.Lock` and can be called from
  multiple threads or tasks.
- **Idempotent:** `add_endpoint` returns silently if the endpoint already
  exists. `remove_endpoint` returns silently if the endpoint is not found.
- **Global config inheritance:** dynamically added endpoints use the global
  `check_interval` and `timeout` configured in the constructor.
- **Metrics lifecycle:** `remove_endpoint` and `update_endpoint` delete all
  Prometheus metrics for the old endpoint (health, latency, status,
  status\_detail).

### Validation

`add_endpoint` and `update_endpoint` validate inputs before proceeding:

- `dep_name` must match `[a-z][a-z0-9-]*`, max 63 chars
- `dep_type` must be a valid `DependencyType`
- `endpoint.host` and `endpoint.port` must be non-empty
- `endpoint.labels` must not use reserved label names

Invalid inputs raise `ValueError`.

### Error Handling

```python
from dephealth import EndpointNotFoundError

try:
    await dh.update_endpoint("api", "old-host", "8080", new_ep, checker)
except EndpointNotFoundError:
    # old endpoint does not exist — use add_endpoint instead
    pass
except RuntimeError:
    # scheduler not started or already stopped
    pass
```

---

## Internal Changes

- `CheckScheduler` now stores a `threading.Lock` for safe state mutations.
- Per-endpoint `asyncio.Task` tracking via `_ep_tasks` dict for cancellation.
- Per-endpoint `threading.Thread` + `threading.Event` tracking for sync mode.
- `_states` dict (`dict[str, _EndpointState]`) replaces iteration over
  `_entries` in `health()` and `health_details()`, protected by lock.

---

## Version Update

```toml
# v0.5.0
version = "0.5.0"

# v0.6.0
version = "0.6.0"
```
