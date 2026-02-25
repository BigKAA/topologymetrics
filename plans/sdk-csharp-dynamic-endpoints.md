# Plan: C# SDK — Dynamic Endpoint Management (v0.6.0)

## Goal

Allow applications to dynamically add, remove, and update health-checked endpoints
at runtime on a running `DepHealthMonitor` instance. This enables use cases like artstore's
Admin Module where Storage Elements are registered/removed via REST API while the
application is serving traffic.

## Current state

- Dependencies are registered via `DepHealthMonitor.CreateBuilder().AddHttp(...)` etc. and
  frozen at `Build()`
- `CheckScheduler.AddDependency()` throws `InvalidOperationException` if called after `Start()`
- One `async Task` per endpoint with dedicated `CancellationTokenSource` per endpoint —
  good foundation for per-endpoint cancellation
- `PrometheusExporter.DeleteMetrics(dep, ep)` already exists and correctly removes all 4 metric
  families (health gauge, latency histogram, 8 status enum series, status detail)
- `EndpointState` is fully thread-safe (all property access under `lock(_lock)`)
- `Health()` and `HealthDetails()` iterate `_states` dictionary without global lock — safe only
  because `_states` is not mutated after start
- `CheckConfig` stored per-`Dependency` but no global config stored for reuse
- `CancellationTokenSource` list tracked in `_cancellations` — but no per-endpoint mapping
  (can't cancel a specific endpoint)
- `_states` is `Dictionary<string, EndpointState>` — not concurrent-safe for mutations

## Target API

Three new methods on `DepHealthMonitor`:

```csharp
public void AddEndpoint(string depName, DependencyType depType,
    bool critical, Endpoint ep, IHealthChecker checker)

public void RemoveEndpoint(string depName, string host, string port)

public void UpdateEndpoint(string depName, string oldHost, string oldPort,
    Endpoint newEp, IHealthChecker checker)
```

New exception: `EndpointNotFoundException : InvalidOperationException`

## Files changed

| File | Change |
|------|--------|
| `DepHealth.Core/CheckScheduler.cs` | Store `_globalConfig`, `ConcurrentDictionary` for states, per-endpoint `CancellationTokenSource` map; 3 new methods |
| `DepHealth.Core/DepHealth.cs` | 3 new facade methods with validation |
| `DepHealth.Core/Exceptions/EndpointNotFoundException.cs` | New exception class |
| `tests/DepHealth.Core.Tests/CheckSchedulerTests.cs` | New tests for dynamic Add/Remove/Update + concurrency |
| `tests/DepHealth.Core.Tests/DepHealthTests.cs` | New integration tests for facade methods |
| `Directory.Build.props` | Bump `0.5.0` → `0.6.0` |
| `sdk-csharp/docs/api-reference.md` | Document new methods (EN) |
| `sdk-csharp/docs/api-reference.ru.md` | Document new methods (RU) |
| `sdk-csharp/README.md` | Add dynamic endpoints section |
| `docs/migration/sdk-csharp-v050-to-v060.md` | Migration guide (EN) |
| `docs/migration/sdk-csharp-v050-to-v060.ru.md` | Migration guide (RU) |
| `CHANGELOG.md` | Add `[sdk-csharp 0.6.0]` section |

## Phases

---

### Phase 1: Scheduler infrastructure — global config, ConcurrentDictionary, per-endpoint CTS

Prepare `CheckScheduler` for dynamic mutations without changing the public API yet.

**Modify `DepHealth.Core/CheckScheduler.cs`:**

- [x] Add `CheckConfig _globalConfig` field (stored at construction, used for dynamic endpoints)
- [x] Change `_states` from `Dictionary<string, EndpointState>` to
  `ConcurrentDictionary<string, EndpointState>`
- [x] Change `_cancellations` from `List<CancellationTokenSource>` to
  `ConcurrentDictionary<string, CancellationTokenSource>` keyed by `"name:host:port"`
- [x] In `Start()`: store each `CancellationTokenSource` in the new dict with endpoint key
- [x] Store parent `CancellationTokenSource` for global stop — derive per-endpoint CTS via
  `CancellationTokenSource.CreateLinkedTokenSource(_globalCts.Token)` so that `Stop()` still
  cancels everything
- [x] Add `object _lock` for synchronizing mutation operations (add/remove/update)
  while keeping read operations lock-free via `ConcurrentDictionary`

**Modify `DepHealth.Core/DepHealth.cs`:**

- [x] In `Build()`: compute `globalConfig` from builder-level interval/timeout/thresholds
  with `CheckConfig` defaults
- [x] Pass `globalConfig` to `CheckScheduler` constructor

**Validation:**

- [x] `dotnet build` passes
- [x] `dotnet test` passes (existing tests, no behavioral change)

**Status:** done

---

### Phase 2: Dynamic methods on CheckScheduler

Implement the three core methods on `CheckScheduler`.

**Add to `DepHealth.Core/CheckScheduler.cs`:**

- [x] `AddEndpoint(string depName, DependencyType depType, bool critical, Endpoint ep, IHealthChecker checker)`
  - `lock(_lock)` block
  - Check `_started && !_stopped`, else throw `InvalidOperationException`
  - Compute key `depName:host:port`, return if exists (idempotent)
  - Build `Dependency` with `_globalConfig`
  - Create `EndpointState` with static fields
  - Insert into `_states`
  - Create linked `CancellationTokenSource`, store in `_cancellations`
  - Fire-and-forget `RunCheckLoopAsync(...)` with the new CTS token
- [x] `RemoveEndpoint(string depName, string host, string port)`
  - `lock(_lock)` block
  - Check `_started`, else throw `InvalidOperationException`
  - Find state by key, return if not found (idempotent)
  - Cancel CTS (`cts.Cancel()`), dispose it, remove from `_cancellations`
  - Remove from `_states`
  - Call `_metrics.DeleteMetrics(dep, ep)` to clean up Prometheus series
- [x] `UpdateEndpoint(string depName, string oldHost, string oldPort, Endpoint newEp, IHealthChecker checker)`
  - `lock(_lock)` block
  - Check `_started && !_stopped`
  - Find old state, throw `EndpointNotFoundException` if missing
  - Cancel old CTS, remove old state, delete old metrics
  - Create new state, new linked CTS, launch new loop, insert

**Add `DepHealth.Core/Exceptions/EndpointNotFoundException.cs`:**

- [x] `public class EndpointNotFoundException : InvalidOperationException`
  - Constructor: `(string depName, string host, string port)`
  - Message: `$"Endpoint not found: {depName}:{host}:{port}"`

**Validation:**

- [x] `dotnet build` passes
- [x] `dotnet test` passes (existing tests unchanged)

**Status:** done

---

### Phase 3: DepHealthMonitor facade methods

Thin wrappers with input validation, delegating to CheckScheduler.

**Modify `DepHealth.Core/DepHealth.cs`:**

- [x] Add `AddEndpoint(string depName, DependencyType depType, bool critical, Endpoint ep, IHealthChecker checker)`
  - Validate `depName` via existing `Dependency.ValidateName()`
  - Validate `depType` is defined enum value
  - Validate `ep.Host` and `ep.Port` non-empty
  - Validate `ep.Labels` via existing label validation (reserved names, pattern)
  - Delegate to `_scheduler.AddEndpoint()`
- [x] Add `RemoveEndpoint(string depName, string host, string port)`
  - Passthrough to `_scheduler.RemoveEndpoint()`
- [x] Add `UpdateEndpoint(string depName, string oldHost, string oldPort, Endpoint newEp, IHealthChecker checker)`
  - Validate `newEp.Host`, `newEp.Port`, `newEp.Labels`
  - Delegate to `_scheduler.UpdateEndpoint()`

**Validation:**

- [x] `dotnet build` passes

**Status:** done

---

### Phase 4: Tests — CheckScheduler level

Unit tests for the three new CheckScheduler methods. Use existing test patterns
(`FakeChecker`, xUnit `[Fact]`).

**Add to `tests/DepHealth.Core.Tests/CheckSchedulerDynamicTests.cs`:**

- [x] `AddEndpoint_AfterStart_AppearsInHealth` — add endpoint after start, wait, verify `Health()` includes it
- [x] `AddEndpoint_Idempotent` — add same endpoint twice, no exception, single entry
- [x] `AddEndpoint_BeforeStart_Throws` — throws `InvalidOperationException`
- [x] `AddEndpoint_AfterStop_Throws` — throws `InvalidOperationException`
- [x] `AddEndpoint_Metrics` — verify health gauge appears with correct labels
- [x] `RemoveEndpoint_AfterStart_DisappearsFromHealth` — remove after start, verify disappears from `Health()`
- [x] `RemoveEndpoint_Idempotent` — remove non-existent, no exception
- [x] `RemoveEndpoint_MetricsDeleted` — verify all metric series removed
- [x] `RemoveEndpoint_BeforeStart_Throws` — throws `InvalidOperationException`
- [x] `UpdateEndpoint_SwapsEndpoint` — update, verify old gone and new appears in `Health()`
- [x] `UpdateEndpoint_NotFound_Throws` — throws `EndpointNotFoundException`
- [x] `UpdateEndpoint_MetricsSwap` — old metrics deleted, new metrics present
- [x] `StopAfterDynamicAdd_CleansUp` — add endpoint, then `Stop()`, verify no task leak
- [x] `ConcurrentAddRemoveHealth_NoExceptions` — run Add/Remove/Health in parallel tasks

**Validation:**

- [x] `dotnet test` passes (all tests)
- [x] No concurrency warnings

**Status:** done

---

### Phase 5: Tests — DepHealthMonitor facade level

Integration tests for the public API.

**Add to `tests/DepHealth.Core.Tests/DepHealthMonitorDynamicTests.cs`:**

- [x] `AddEndpoint_AfterStart_AppearsInHealth` — create DepHealthMonitor, start, AddEndpoint, verify `Health()`
- [x] `AddEndpoint_InvalidName_Throws` — invalid dep name, throws `ValidationException`
- [x] `AddEndpoint_InvalidType_Throws` — undefined enum value, throws exception
- [x] `AddEndpoint_MissingHost_Throws` — empty host, throws `ValidationException`
- [x] `AddEndpoint_ReservedLabel_Throws` — reserved label, throws `ValidationException`
- [x] `RemoveEndpoint_DisappearsFromHealth` — remove, verify gone from `Health()`
- [x] `UpdateEndpoint_SwapsEndpoint` — update, verify old gone and new present
- [x] `UpdateEndpoint_MissingNewHost_Throws` — empty new host, throws `ValidationException`
- [x] `UpdateEndpoint_NotFound_Throws` — throws `EndpointNotFoundException`
- [x] `AddEndpoint_InheritsGlobalConfig` — verify dynamic endpoint uses global interval/timeout

**Validation:**

- [x] `dotnet test` passes (all tests)

**Status:** done

---

### Phase 6: Version bump, documentation, changelog

**Version bump:**

- [x] `Directory.Build.props` → `<Version>0.6.0</Version>`

**Documentation (EN):**

- [x] Create `sdk-csharp/docs/api-reference.md` — add `AddEndpoint`, `RemoveEndpoint`, `UpdateEndpoint`
- [x] Create `sdk-csharp/README.md` — add "Dynamic Endpoints" section with usage example
- [x] Create `docs/migration/sdk-csharp-v050-to-v060.md` — migration guide

**Documentation (RU):**

- [x] Create `sdk-csharp/docs/api-reference.ru.md` — same as EN
- [x] Create `docs/migration/sdk-csharp-v050-to-v060.ru.md` — migration guide

**Changelog:**

- [x] Update `CHANGELOG.md` — add `[sdk-csharp 0.6.0]` section

**Validation:**

- [x] `dotnet build && dotnet test` — all pass (206 tests, 0 failures)
- [x] `markdownlint` on all new/changed `.md` files — 0 issues

**Status:** done

---

### Phase 7: Merge, tag, release

**Pre-merge checklist:**

- [ ] All unit tests pass (including new dynamic endpoint tests)
- [ ] No concurrency issues in tests
- [ ] All linters pass
- [ ] Backward compatibility verified (existing API unchanged)
- [ ] Docs complete (EN + RU)
- [ ] CHANGELOG updated

**Actions:**

- Merge to master (or PR — ask user)
- Tag: `sdk-csharp/v0.6.0`
- GitHub Release: sdk-csharp/v0.6.0
- Move this plan to `plans/archive/`

**Status:** not started
