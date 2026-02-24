# Plan: Go SDK — Dynamic Endpoint Management (v0.7.0)

## Goal

Allow applications to dynamically add, remove, and update health-checked endpoints
at runtime on a running `DepHealth` instance. This enables use cases like artstore's
Admin Module where Storage Elements are registered/removed via REST API while the
application is serving traffic.

## Current state

- Dependencies are registered via functional options in `New()` and frozen at `Start()`
- `Scheduler` iterates `deps []scheduledDep` once in `Start()`, launching one goroutine
  per endpoint — no mutations possible after that
- `MetricsExporter` already has `DeleteMetrics(dep, ep)` method (unused) that correctly
  removes all 4 metric series for a label combination
- `Health()` and `HealthDetails()` grab the `states` map reference under `s.mu` then
  iterate without holding the lock — unsafe for concurrent map mutations
- Global `CheckConfig` (interval, timeout) is resolved per-dependency in `buildDependency()`
  but not stored anywhere for later reuse
- Goroutines all share a single cancellable context from `Start()` — no per-endpoint
  cancellation

## Target API

Three new methods on `DepHealth`:

```go
func (dh *DepHealth) AddEndpoint(depName string, depType DependencyType,
    critical bool, ep Endpoint, checker HealthChecker) error

func (dh *DepHealth) RemoveEndpoint(depName, host, port string) error

func (dh *DepHealth) UpdateEndpoint(depName, oldHost, oldPort string,
    newEp Endpoint, checker HealthChecker) error
```

## Files changed

| File | Change |
|------|--------|
| `dephealth/scheduler.go` | New fields (`globalConfig`, `ctx`, per-endpoint cancel); 3 new methods; fix `Health`/`HealthDetails` locking; new sentinel `ErrEndpointNotFound`; new option `WithGlobalCheckConfig` |
| `dephealth/dephealth.go` | 3 new facade methods with validation; pass `globalConfig` to scheduler in `New()` |
| `dephealth/scheduler_test.go` | New tests for dynamic Add/Remove/Update + race tests |
| `dephealth/dephealth_test.go` | New integration tests for facade methods |
| `dephealth/version.go` | Bump `0.6.0` → `0.7.0` |
| `sdk-go/docs/api-reference.md` | Document new methods (EN) |
| `sdk-go/docs/api-reference.ru.md` | Document new methods (RU) |
| `sdk-go/README.md` | Add dynamic endpoints section |
| `docs/migration/v060-to-v070.md` | Migration guide (EN) |
| `docs/migration/v060-to-v070.ru.md` | Migration guide (RU) |
| `CHANGELOG.md` | Add `[0.7.0]` section |

## Phases

---

### Phase 1: Scheduler infrastructure — fields, per-endpoint cancel, global config

Prepare the `Scheduler` for dynamic mutations without changing the public API yet.

**Modify `dephealth/scheduler.go`:**

- [x] Add `globalConfig CheckConfig` field to `Scheduler` struct
- [x] Add `ctx context.Context` field to `Scheduler` struct (stores parent context from `Start()`)
- [x] Add `cancel context.CancelFunc` field to `endpointState` struct
- [x] Add `ErrEndpointNotFound` sentinel to existing `var` block
- [x] Add `WithGlobalCheckConfig(cfg CheckConfig) SchedulerOption`
- [x] Add `globalConfig *CheckConfig` field to `schedulerConfig` struct
- [x] Update `NewScheduler()`: apply `globalConfig` from options (with `DefaultCheckConfig()` fallback)
- [x] Update `Start()`: store `ctx` in `s.ctx`; derive per-endpoint child context via `context.WithCancel(ctx)` and save `epCancel` in `st.cancel`
- [x] Fix `Health()`: hold `s.mu` with `defer s.mu.Unlock()` during entire map iteration
- [x] Fix `HealthDetails()`: same locking pattern

**Modify `dephealth/dephealth.go`:**

- [x] In `New()`: build `globalCfg CheckConfig` from `cfg.interval`/`cfg.timeout` with defaults
- [x] In `New()`: pass `WithGlobalCheckConfig(globalCfg)` to `NewScheduler()`

**Validation:**

- [x] `make build` passes
- [x] `make test` passes (existing tests, no behavioral change)
- [x] `make lint` passes

**Status:** done

---

### Phase 2: Dynamic methods on Scheduler

Implement the three core methods on `Scheduler`.

**Add to `dephealth/scheduler.go`:**

- [ ] `Scheduler.AddEndpoint(depName, depType, critical, ep, checker) error`
  - Lock `s.mu`
  - Check `started` and `!stopped`, else return `ErrNotStarted`
  - Compute key `depName:host:port`, return nil if exists (idempotent)
  - Build `Dependency` with `s.globalConfig`
  - Create `endpointState` with static fields
  - Derive child context `context.WithCancel(s.ctx)`, store cancel in state
  - Insert into `s.states`, `s.wg.Add(1)`, launch `runEndpointLoop`
- [ ] `Scheduler.RemoveEndpoint(depName, host, port) error`
  - Lock `s.mu`
  - Check `started`, else return `ErrNotStarted`
  - Find state by key, return nil if not found (idempotent)
  - Call `st.cancel()`, `delete(s.states, key)`
  - Build minimal `Dependency`/`Endpoint` from state, call `s.metrics.DeleteMetrics()`
- [ ] `Scheduler.UpdateEndpoint(depName, oldHost, oldPort, newEp, checker) error`
  - Lock `s.mu`
  - Check `started` and `!stopped`
  - Find old state, return `ErrEndpointNotFound` if missing
  - Cancel old goroutine, delete old state, delete old metrics
  - Create new state, new child context, insert, launch goroutine

**Validation:**

- [ ] `make build` passes
- [ ] `make lint` passes

**Status:** not started

---

### Phase 3: DepHealth facade methods

Thin wrappers with input validation, delegating to Scheduler.

**Modify `dephealth/dephealth.go`:**

- [ ] Add `func (dh *DepHealth) AddEndpoint(depName, depType, critical, ep, checker) error`
  - Validate `depName` via `ValidateName()`
  - Validate `depType` via `ValidTypes`
  - Validate `ep.Host != ""` and `ep.Port != ""`
  - Validate `ep.Labels` via `ValidateLabels()`
  - Delegate to `dh.scheduler.AddEndpoint()`
- [ ] Add `func (dh *DepHealth) RemoveEndpoint(depName, host, port) error`
  - Passthrough to `dh.scheduler.RemoveEndpoint()`
- [ ] Add `func (dh *DepHealth) UpdateEndpoint(depName, oldHost, oldPort, newEp, checker) error`
  - Validate `newEp.Host`, `newEp.Port`, `newEp.Labels`
  - Delegate to `dh.scheduler.UpdateEndpoint()`

**Validation:**

- [ ] `make build` passes
- [ ] `make lint` passes

**Status:** not started

---

### Phase 4: Tests — Scheduler level

Unit tests for the three new Scheduler methods. Use existing test patterns
(`mockChecker`, `newTestScheduler`, `testutil.CollectAndCompare`).

**Add to `dephealth/scheduler_test.go`:**

- [ ] `TestScheduler_AddEndpoint` — add endpoint after Start, wait, verify Health() includes it
- [ ] `TestScheduler_AddEndpoint_Idempotent` — add same endpoint twice, verify no error, single entry
- [ ] `TestScheduler_AddEndpoint_BeforeStart` — returns `ErrNotStarted`
- [ ] `TestScheduler_AddEndpoint_AfterStop` — returns `ErrNotStarted`
- [ ] `TestScheduler_AddEndpoint_Metrics` — verify health gauge appears with correct labels
- [ ] `TestScheduler_RemoveEndpoint` — remove after Start, verify disappears from Health()
- [ ] `TestScheduler_RemoveEndpoint_Idempotent` — remove non-existent, returns nil
- [ ] `TestScheduler_RemoveEndpoint_MetricsDeleted` — verify health/latency/status metrics gone
- [ ] `TestScheduler_RemoveEndpoint_BeforeStart` — returns `ErrNotStarted`
- [ ] `TestScheduler_UpdateEndpoint` — update, verify old gone and new appears in Health()
- [ ] `TestScheduler_UpdateEndpoint_NotFound` — returns `ErrEndpointNotFound`
- [ ] `TestScheduler_UpdateEndpoint_MetricsSwap` — old metrics deleted, new metrics present
- [ ] `TestScheduler_StopAfterDynamicAdd` — add endpoint, then Stop(), verify no goroutine leak
- [ ] `TestScheduler_ConcurrentAddRemoveHealth` — run Add/Remove/Health in parallel goroutines with `-race`

**Validation:**

- [ ] `make test` passes (all tests, including new ones)
- [ ] `make test` with `-race` passes (critical for concurrency tests)
- [ ] `make lint` passes

**Status:** not started

---

### Phase 5: Tests — DepHealth facade level

Integration tests for the public API.

**Add to `dephealth/dephealth_test.go`:**

- [ ] `TestDepHealth_AddEndpoint` — create DepHealth, Start, AddEndpoint, verify Health()
- [ ] `TestDepHealth_AddEndpoint_InvalidName` — invalid dep name, returns validation error
- [ ] `TestDepHealth_AddEndpoint_InvalidType` — unknown type, returns error
- [ ] `TestDepHealth_AddEndpoint_MissingHost` — empty host, returns error
- [ ] `TestDepHealth_AddEndpoint_ReservedLabel` — reserved label, returns error
- [ ] `TestDepHealth_RemoveEndpoint` — remove, verify gone from Health()
- [ ] `TestDepHealth_UpdateEndpoint` — update, verify old gone and new present
- [ ] `TestDepHealth_UpdateEndpoint_MissingNewHost` — empty new host, returns error
- [ ] `TestDepHealth_AddEndpoint_InheritsGlobalConfig` — verify dynamic endpoint uses global interval/timeout

**Validation:**

- [ ] `make test` passes
- [ ] `make lint` passes

**Status:** not started

---

### Phase 6: Version bump, documentation, changelog

**Version bump:**

- [ ] `dephealth/version.go` → `Version = "0.7.0"`

**Documentation (EN):**

- [ ] Update `sdk-go/docs/api-reference.md` — add `AddEndpoint`, `RemoveEndpoint`, `UpdateEndpoint` to API reference
- [ ] Update `sdk-go/README.md` — add "Dynamic Endpoints" section with usage example
- [ ] Create `docs/migration/v060-to-v070.md` — migration guide

**Documentation (RU):**

- [ ] Update `sdk-go/docs/api-reference.ru.md` — same as EN
- [ ] Create `docs/migration/v060-to-v070.ru.md` — migration guide

**Changelog:**

- [ ] Update `CHANGELOG.md` — add `[0.7.0]` section

**Validation:**

- [ ] `make build && make test && make lint` — all pass
- [ ] `markdownlint` on all new/changed `.md` files — 0 issues

**Status:** not started

---

### Phase 7: Merge, tag, release

**Pre-merge checklist:**

- [ ] All unit tests pass (including new dynamic endpoint tests)
- [ ] `go test -race` passes
- [ ] All linters pass
- [ ] Backward compatibility verified (existing API unchanged)
- [ ] Docs complete (EN + RU)
- [ ] CHANGELOG updated

**Actions:**

- Merge to master (or PR — ask user)
- Tag: `sdk-go/v0.7.0`
- GitHub Release: sdk-go/v0.7.0
- Move this plan to `plans/archive/`

**Status:** not started
