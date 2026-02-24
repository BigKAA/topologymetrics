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

- [x] `addEndpoint(String depName, DependencyType depType, boolean critical, Endpoint ep, HealthChecker checker)`
  - `synchronized` block
  - Check `started && !stopped`, else throw `IllegalStateException`
  - Compute key `depName:host:port`, return silently if exists (idempotent)
  - Build `Dependency` with `globalConfig`
  - Create `EndpointState` with static fields
  - Insert into `states`
  - Schedule check via `executor.scheduleAtFixedRate()`, store `ScheduledFuture`
- [x] `removeEndpoint(String depName, String host, String port)`
  - `synchronized` block
  - Check `started`, else throw `IllegalStateException`
  - Find state by key, return silently if not found (idempotent)
  - Cancel `ScheduledFuture` (`future.cancel(false)`)
  - Remove from `states`
  - Call `metrics.deleteMetrics()` to clean up Prometheus series
- [x] `updateEndpoint(String depName, String oldHost, String oldPort, Endpoint newEp, HealthChecker checker)`
  - `synchronized` block
  - Check `started && !stopped`
  - Find old state, throw `EndpointNotFoundException` if missing
  - Cancel old future, remove old state, delete old metrics
  - Create new state, schedule new check, insert

**Add `dephealth-core/.../EndpointNotFoundException.java`:**

- [x] `public class EndpointNotFoundException extends DepHealthException`
  - Constructor: `(String depName, String host, String port)`
  - Message: `"Endpoint not found: depName:host:port"`

**Validation:**

- [x] `mvn compile` passes
- [x] `mvn test` passes (existing tests unchanged)
- [x] Checkstyle/SpotBugs pass

**Status:** done

---

### Phase 3: DepHealthMonitor facade methods

Thin wrappers with input validation, delegating to CheckScheduler.

**Modify `dephealth-core/.../DepHealth.java`:**

- [x] Add `addEndpoint(String depName, DependencyType depType, boolean critical, Endpoint ep, HealthChecker checker)`
  - Validate `depName` via existing `Dependency.validateName()`
  - Validate `depType != null`
  - Validate `ep.host()` and `ep.port()` non-empty
  - Validate `ep.labels()` via existing label validation
  - Delegate to `scheduler.addEndpoint()`
- [x] Add `removeEndpoint(String depName, String host, String port)`
  - Passthrough to `scheduler.removeEndpoint()`
- [x] Add `updateEndpoint(String depName, String oldHost, String oldPort, Endpoint newEp, HealthChecker checker)`
  - Validate `newEp.host()`, `newEp.port()`, `newEp.labels()`
  - Delegate to `scheduler.updateEndpoint()`

**Validation:**

- [x] `mvn compile` passes
- [x] Checkstyle passes

**Status:** done

---

### Phase 4: Tests — CheckScheduler level

Unit tests for the three new CheckScheduler methods. Use existing test patterns
(`CountDownLatch`, mock `HealthChecker`, `SimpleMeterRegistry`).

**Add to `dephealth-core/src/test/.../CheckSchedulerTest.java`:**

- [x] `testAddEndpoint` — add endpoint after start, wait via latch, verify `health()` includes it
- [x] `testAddEndpoint_Idempotent` — add same endpoint twice, verify no error, single entry
- [x] `testAddEndpoint_BeforeStart` — throws `IllegalStateException`
- [x] `testAddEndpoint_AfterStop` — throws `IllegalStateException`
- [x] `testAddEndpoint_Metrics` — verify health gauge appears with correct labels in `SimpleMeterRegistry`
- [x] `testRemoveEndpoint` — remove after start, verify disappears from `health()`
- [x] `testRemoveEndpoint_Idempotent` — remove non-existent, no error
- [x] `testRemoveEndpoint_MetricsDeleted` — verify all metric series removed from registry
- [x] `testRemoveEndpoint_BeforeStart` — throws `IllegalStateException`
- [x] `testUpdateEndpoint` — update, verify old gone and new appears in `health()`
- [x] `testUpdateEndpoint_NotFound` — throws `EndpointNotFoundException`
- [x] `testUpdateEndpoint_MetricsSwap` — old metrics deleted, new metrics present
- [x] `testStopAfterDynamicAdd` — add endpoint, then `stop()`, verify clean shutdown
- [x] `testConcurrentAddRemoveHealth` — run Add/Remove/Health in parallel threads, no `ConcurrentModificationException`

**Validation:**

- [x] `mvn test` passes (all tests)
- [x] No concurrency warnings or flaky tests

**Status:** done

---

### Phase 5: Tests — DepHealthMonitor facade level

Integration tests for the public API.

**Add to `dephealth-core/src/test/.../DepHealthTest.java`:**

- [x] `testAddEndpoint` — create DepHealthMonitor, start, addEndpoint, verify `health()`
- [x] `testAddEndpoint_InvalidName` — invalid dep name, throws `ValidationException`
- [x] `testAddEndpoint_InvalidType` — null type, throws exception
- [x] `testAddEndpoint_MissingHost` — empty host, throws `ValidationException`
- [x] `testAddEndpoint_MissingPort` — empty port, throws `ValidationException`
- [x] `testAddEndpoint_ReservedLabel` — reserved label, throws `ValidationException`
- [x] `testRemoveEndpoint` — remove, verify gone from `health()`
- [x] `testUpdateEndpoint` — update, verify old gone and new present
- [x] `testUpdateEndpoint_MissingNewHost` — empty new host, throws `ValidationException`
- [x] `testUpdateEndpoint_NotFound` — throws `EndpointNotFoundException`
- [x] `testAddEndpoint_InheritsGlobalConfig` — verify dynamic endpoint uses global interval/timeout

**Validation:**

- [x] `mvn test` passes (all tests)
- [x] Checkstyle passes

**Status:** done

---

### Phase 6: Version bump, documentation, changelog

**Version bump:**

- [x] Parent `pom.xml` → `<version>0.6.0</version>`
- [x] All child modules inherit version

**Documentation (EN):**

- [x] Update `docs/migration/java.md` — add v0.6.0 migration section
- [x] Create `docs/migration/sdk-java-v050-to-v060.md` — migration guide

**Documentation (RU):**

- [x] Update `docs/migration/java.ru.md` — add v0.6.0 migration section
- [x] Create `docs/migration/sdk-java-v050-to-v060.ru.md` — migration guide

**Changelog:**

- [x] Update `CHANGELOG.md` — add `[sdk-java 0.6.0]` section

**Validation:**

- [x] `mvn compile && mvn test` — all pass
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
- Tag: `sdk-java/v0.6.0`
- GitHub Release: sdk-java/v0.6.0
- Publish to maven central
- Move this plan to `plans/archive/`

**Status:** not started
