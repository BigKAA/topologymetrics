*[Русская версия](sdk-csharp-v050-to-v060.ru.md)*

# C# SDK: Migration from v0.5.0 to v0.6.0

Migration guide for the C# SDK v0.6.0 release.

> This release affects **C# SDK only**. Go SDK remains at v0.7.0;
> Java SDK remains at v0.6.0; Python SDK remains at v0.6.0.

## What Changed

Three new methods on `DepHealthMonitor` allow dynamic endpoint management
at runtime:

| Method | Description |
| --- | --- |
| `AddEndpoint` | Add a monitored endpoint after `Start()` |
| `RemoveEndpoint` | Remove an endpoint (cancels task, deletes metrics) |
| `UpdateEndpoint` | Atomically replace an endpoint with a new one |

A new exception `EndpointNotFoundException` (extends `InvalidOperationException`)
is thrown by `UpdateEndpoint` when the old endpoint does not exist.

---

## Do I Need to Change My Code?

**No.** This is a fully backward-compatible release. All existing code
continues to work without modification.

---

## New Feature: Dynamic Endpoints

Prior to v0.6.0, all dependencies had to be registered upfront via
builder methods (`AddHttp()`, `AddPostgres()`, etc.). Once `Build()` was
called, the set of monitored endpoints was frozen.

Starting with v0.6.0, you can add, remove, and update endpoints on a
running `DepHealthMonitor` instance:

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

### Key Behaviors

- **Thread-safe:** all three methods use `lock` and can be called from
  multiple threads. Read operations (`Health()`, `HealthDetails()`) remain
  lock-free via `ConcurrentDictionary`.
- **Idempotent:** `AddEndpoint` returns silently if the endpoint already
  exists. `RemoveEndpoint` returns silently if the endpoint is not found.
- **Global config inheritance:** dynamically added endpoints use the
  global check interval and timeout configured in the builder.
- **Metrics lifecycle:** `RemoveEndpoint` and `UpdateEndpoint` delete
  all Prometheus metrics for the old endpoint (health, latency, status,
  status\_detail).

### Validation

`AddEndpoint` and `UpdateEndpoint` validate inputs before proceeding:

- `depName` must match `[a-z][a-z0-9-]*`, max 63 chars
- `depType` must be a defined `DependencyType` enum value
- `ep.Host` and `ep.Port` must be non-empty
- `ep.Labels` must not use reserved label names

Invalid inputs throw `ValidationException`.

### Error Handling

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

---

## Internal Changes

- `CheckScheduler` now stores a global `CheckConfig` for dynamic endpoints.
- `_states` changed from `Dictionary` to `ConcurrentDictionary` for safe
  concurrent iteration in `Health()` / `HealthDetails()`.
- Per-endpoint `CancellationTokenSource` tracking via `_cancellations`
  dict (keyed by `"name:host:port"`), linked to a global CTS.
- `object _mutationLock` synchronizes add/remove/update operations while
  keeping read operations lock-free.

---

## Version Update

```xml
<!-- v0.5.0 -->
<Version>0.5.0</Version>

<!-- v0.6.0 -->
<Version>0.6.0</Version>
```
