# Plan: Java SDK — Dynamic Endpoint Management (v0.6.0)

## Goal

Allow applications to dynamically add, remove, and update health-checked endpoints
at runtime on a running `DepHealthMonitor` instance. This enables use cases like artstore's
Admin Module where Storage Elements are registered/removed via REST API while the
application is serving traffic.

## Current state

- Dependencies are registered via `DepHealth.builder().dependency(...)` and frozen at `build()`
- `CheckScheduler.addDependency()` throws `IllegalStateException` if called after `start()`
- `ScheduledExecutorService` with `scheduleAtFixedRate()` per endpoint — fixed thread pool
  sized to total endpoint count; no dynamic resizing
- `MetricsExporter` uses Micrometer `MeterRegistry`; status detail has delete-on-change
  pattern (`registry.remove(oldMeter)`), but no full `deleteMetrics()` for all 4 metric
  families
- `EndpointState` is thread-safe (all methods `synchronized`)
- `Health()` and `HealthDetails()` iterate `states` map without global lock — safe only
  because `states` is not mutated after start
- `CheckConfig` (interval, timeout, thresholds) stored per-`Dependency` but not globally
  accessible for dynamic endpoints
- No per-endpoint `ScheduledFuture` tracking — futures are fire-and-forget

## Target API

Three new methods on `DepHealthMonitor`:

```java
public void addEndpoint(String depName, DependencyType depType,
    boolean critical, Endpoint ep, HealthChecker checker)
    throws ValidationException

public void removeEndpoint(String depName, String host, String port)

public void updateEndpoint(String depName, String oldHost, String oldPort,
    Endpoint newEp, HealthChecker checker)
    throws EndpointNotFoundException, ValidationException
```

New exception: `EndpointNotFoundException extends RuntimeException`

## Files changed

| File | Change |
|------|--------|
| `dephealth-core/.../scheduler/CheckScheduler.java` | Store `globalConfig`, per-endpoint `ScheduledFuture` tracking, `ConcurrentHashMap` for states; 3 new methods; `deleteMetrics()` support |
| `dephealth-core/.../scheduler/EndpointState.java` | Add `ScheduledFuture` field for cancellation |
| `dephealth-core/.../metrics/MetricsExporter.java` | Add full `deleteMetrics(dep, ep)` method removing all 4 metric families |
| `dephealth-core/.../DepHealth.java` | 3 new facade methods with validation |
| `dephealth-core/.../EndpointNotFoundException.java` | New exception class |
| `dephealth-core/src/test/.../CheckSchedulerTest.java` | New tests for dynamic Add/Remove/Update + concurrency |
| `dephealth-core/src/test/.../DepHealthTest.java` | New integration tests for facade methods |
| `pom.xml` (parent) | Bump `0.5.0` → `0.6.0` |
| `sdk-java/docs/api-reference.md` | Document new methods (EN) |
| `sdk-java/docs/api-reference.ru.md` | Document new methods (RU) |
| `sdk-java/README.md` | Add dynamic endpoints section |
| `docs/migration/sdk-java-v050-to-v060.md` | Migration guide (EN) |
| `docs/migration/sdk-java-v050-to-v060.ru.md` | Migration guide (RU) |
| `CHANGELOG.md` | Add `[sdk-java 0.6.0]` section |

## Phases

---

### Phase 1: Scheduler infrastructure — global config, per-endpoint Future tracking, ConcurrentHashMap

Prepare `CheckScheduler` for dynamic mutations without changing the public API yet.

**Modify `dephealth-core/.../scheduler/CheckScheduler.java`:**

- [x] Add `CheckConfig globalConfig` field (stored at construction, used for dynamic endpoints)
- [x] Change `states` from `HashMap` to `ConcurrentHashMap<String, EndpointState>`
- [x] Add `ScheduledFuture<?>` field to `EndpointState` (or a parallel `Map<String, ScheduledFuture<?>>`)
- [x] In `start()`: store each `scheduleAtFixedRate()` result in the state/map
- [x] Switch `ScheduledExecutorService` from fixed-size to `ScheduledThreadPoolExecutor`
  (supports dynamic resizing via `setCorePoolSize()`)
- [x] Use `ConcurrentHashMap` for `states` — iteration is weakly consistent, safe for
  `health()` and `healthDetails()`; `runCheck()` guards against removed states with null check

**Modify `dephealth-core/.../DepHealth.java`:**

- [x] In `builder().build()`: compute `globalConfig` from builder-level interval/timeout/thresholds
  with defaults from `CheckConfig.defaults()` (includes timeout-capping logic)
- [x] Pass `globalConfig` to `CheckScheduler` constructor

**Modify `dephealth-core/.../metrics/MetricsExporter.java`:**

- [x] Add `deleteMetrics(Dependency dep, Endpoint ep)` method:
  - Remove `app_dependency_health` gauge via `registry.remove()`
  - Remove `app_dependency_latency_seconds` distribution summary
  - Remove all 8 `app_dependency_status` gauges (one per category)
  - Remove `app_dependency_status_detail` gauge (with prev detail tracking cleanup)

**Validation:**

- [x] `mvn compile` passes
- [x] `mvn test` passes (existing tests, no behavioral change)
- [x] Checkstyle/SpotBugs pass

**Status:** done

---

### Phase 2: Dynamic methods on CheckScheduler

Implement the three core methods on `CheckScheduler`.

**Add to `dephealth-core/.../scheduler/CheckScheduler.java`:**

- [ ] `addEndpoint(String depName, DependencyType depType, boolean critical, Endpoint ep, HealthChecker checker)`
  - `synchronized` block
  - Check `started && !stopped`, else throw `IllegalStateException`
  - Compute key `depName:host:port`, return silently if exists (idempotent)
  - Build `Dependency` with `globalConfig`
  - Create `EndpointState` with static fields
  - Insert into `states`
  - Schedule check via `executor.scheduleAtFixedRate()`, store `ScheduledFuture`
- [ ] `removeEndpoint(String depName, String host, String port)`
  - `synchronized` block
  - Check `started`, else throw `IllegalStateException`
  - Find state by key, return silently if not found (idempotent)
  - Cancel `ScheduledFuture` (`future.cancel(false)`)
  - Remove from `states`
  - Call `metrics.deleteMetrics()` to clean up Prometheus series
- [ ] `updateEndpoint(String depName, String oldHost, String oldPort, Endpoint newEp, HealthChecker checker)`
  - `synchronized` block
  - Check `started && !stopped`
  - Find old state, throw `EndpointNotFoundException` if missing
  - Cancel old future, remove old state, delete old metrics
  - Create new state, schedule new check, insert

**Add `dephealth-core/.../EndpointNotFoundException.java`:**

- [ ] `public class EndpointNotFoundException extends RuntimeException`
  - Constructor: `(String depName, String host, String port)`
  - Message: `"Endpoint not found: depName:host:port"`

**Validation:**

- [ ] `mvn compile` passes
- [ ] `mvn test` passes (existing tests unchanged)

**Status:** not started

---

### Phase 3: DepHealthMonitor facade methods

Thin wrappers with input validation, delegating to CheckScheduler.

**Modify `dephealth-core/.../DepHealth.java`:**

- [ ] Add `addEndpoint(String depName, DependencyType depType, boolean critical, Endpoint ep, HealthChecker checker)`
  - Validate `depName` via existing `Dependency.validateName()`
  - Validate `depType != null`
  - Validate `ep.host()` and `ep.port()` non-empty
  - Validate `ep.labels()` via existing label validation
  - Delegate to `scheduler.addEndpoint()`
- [ ] Add `removeEndpoint(String depName, String host, String port)`
  - Passthrough to `scheduler.removeEndpoint()`
- [ ] Add `updateEndpoint(String depName, String oldHost, String oldPort, Endpoint newEp, HealthChecker checker)`
  - Validate `newEp.host()`, `newEp.port()`, `newEp.labels()`
  - Delegate to `scheduler.updateEndpoint()`

**Validation:**

- [ ] `mvn compile` passes
- [ ] Checkstyle passes

**Status:** not started

---

### Phase 4: Tests — CheckScheduler level

Unit tests for the three new CheckScheduler methods. Use existing test patterns
(`CountDownLatch`, mock `HealthChecker`, `SimpleMeterRegistry`).

**Add to `dephealth-core/src/test/.../CheckSchedulerTest.java`:**

- [ ] `testAddEndpoint` — add endpoint after start, wait via latch, verify `health()` includes it
- [ ] `testAddEndpoint_Idempotent` — add same endpoint twice, verify no error, single entry
- [ ] `testAddEndpoint_BeforeStart` — throws `IllegalStateException`
- [ ] `testAddEndpoint_AfterStop` — throws `IllegalStateException`
- [ ] `testAddEndpoint_Metrics` — verify health gauge appears with correct labels in `SimpleMeterRegistry`
- [ ] `testRemoveEndpoint` — remove after start, verify disappears from `health()`
- [ ] `testRemoveEndpoint_Idempotent` — remove non-existent, no error
- [ ] `testRemoveEndpoint_MetricsDeleted` — verify all metric series removed from registry
- [ ] `testRemoveEndpoint_BeforeStart` — throws `IllegalStateException`
- [ ] `testUpdateEndpoint` — update, verify old gone and new appears in `health()`
- [ ] `testUpdateEndpoint_NotFound` — throws `EndpointNotFoundException`
- [ ] `testUpdateEndpoint_MetricsSwap` — old metrics deleted, new metrics present
- [ ] `testStopAfterDynamicAdd` — add endpoint, then `stop()`, verify clean shutdown
- [ ] `testConcurrentAddRemoveHealth` — run Add/Remove/Health in parallel threads, no `ConcurrentModificationException`

**Validation:**

- [ ] `mvn test` passes (all tests)
- [ ] No concurrency warnings or flaky tests

**Status:** not started

---

### Phase 5: Tests — DepHealthMonitor facade level

Integration tests for the public API.

**Add to `dephealth-core/src/test/.../DepHealthTest.java`:**

- [ ] `testAddEndpoint` — create DepHealthMonitor, start, addEndpoint, verify `health()`
- [ ] `testAddEndpoint_InvalidName` — invalid dep name, throws `ValidationException`
- [ ] `testAddEndpoint_InvalidType` — null type, throws exception
- [ ] `testAddEndpoint_MissingHost` — empty host, throws `ValidationException`
- [ ] `testAddEndpoint_ReservedLabel` — reserved label, throws `ValidationException`
- [ ] `testRemoveEndpoint` — remove, verify gone from `health()`
- [ ] `testUpdateEndpoint` — update, verify old gone and new present
- [ ] `testUpdateEndpoint_MissingNewHost` — empty new host, throws `ValidationException`
- [ ] `testUpdateEndpoint_NotFound` — throws `EndpointNotFoundException`
- [ ] `testAddEndpoint_InheritsGlobalConfig` — verify dynamic endpoint uses global interval/timeout

**Validation:**

- [ ] `mvn test` passes (all tests)
- [ ] Checkstyle passes

**Status:** not started

---

### Phase 6: Version bump, documentation, changelog

**Version bump:**

- [ ] Parent `pom.xml` → `<version>0.6.0</version>`
- [ ] All child modules inherit version

**Documentation (EN):**

- [ ] Update `sdk-java/docs/api-reference.md` — add `addEndpoint`, `removeEndpoint`, `updateEndpoint`
- [ ] Update `sdk-java/README.md` — add "Dynamic Endpoints" section with usage example
- [ ] Create `docs/migration/sdk-java-v050-to-v060.md` — migration guide

**Documentation (RU):**

- [ ] Update `sdk-java/docs/api-reference.ru.md` — same as EN
- [ ] Create `docs/migration/sdk-java-v050-to-v060.ru.md` — migration guide

**Changelog:**

- [ ] Update `CHANGELOG.md` — add `[sdk-java 0.6.0]` section

**Validation:**

- [ ] `mvn compile && mvn test` — all pass
- [ ] `markdownlint` on all new/changed `.md` files — 0 issues

**Status:** not started

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
- Tag: `sdk-java/v0.6.0`
- GitHub Release: sdk-java/v0.6.0
- Move this plan to `plans/archive/`

**Status:** not started
