# Plan: Python SDK — Dynamic Endpoint Management (v0.6.0)

## Goal

Allow applications to dynamically add, remove, and update health-checked endpoints
at runtime on a running `DependencyHealth` instance. This enables use cases like artstore's
Admin Module where Storage Elements are registered/removed via REST API while the
application is serving traffic.

## Current state

- Dependencies are registered via factory functions (`http_check()`, `redis_check()`, etc.)
  passed to `DependencyHealth(...)` constructor and frozen at init
- `CheckScheduler.add(dep, checker)` is called during `__init__` only — not exposed publicly
- Asyncio mode: one `asyncio.Task` per endpoint via `asyncio.create_task()` in `start()`
- Threading mode: one `threading.Thread` per endpoint with its own event loop
- `MetricsExporter.delete_metrics(dep, ep)` already exists and correctly removes all 4 metric
  families (health gauge, latency histogram, 8 status enum series, status detail)
- `health()` and `health_details()` iterate `_entries` list without lock — safe only because
  `_entries` is not mutated after init
- `_EndpointState` fields are updated without explicit lock — relies on GIL for atomic updates
- `CheckConfig` stored per-`Dependency` but no global config stored for reuse
- No per-endpoint cancellation — tasks run until `stop()` cancels all via `task.cancel()`
- Async and sync dual-mode support (`start()`/`stop()` + `start_sync()`/`stop_sync()`)

## Target API

Three new methods on `DependencyHealth`:

```python
async def add_endpoint(self, dep_name: str, dep_type: DependencyType,
    critical: bool, endpoint: Endpoint, checker: HealthChecker) -> None

async def remove_endpoint(self, dep_name: str, host: str, port: str) -> None

async def update_endpoint(self, dep_name: str, old_host: str, old_port: str,
    new_endpoint: Endpoint, checker: HealthChecker) -> None
```

Synchronous variants:

```python
def add_endpoint_sync(self, dep_name: str, dep_type: DependencyType,
    critical: bool, endpoint: Endpoint, checker: HealthChecker) -> None

def remove_endpoint_sync(self, dep_name: str, host: str, port: str) -> None

def update_endpoint_sync(self, dep_name: str, old_host: str, old_port: str,
    new_endpoint: Endpoint, checker: HealthChecker) -> None
```

New exception: `EndpointNotFoundError(Exception)`

## Files changed

| File | Change |
|------|--------|
| `dephealth/scheduler.py` | Store `global_config`, per-endpoint task tracking, `threading.Lock` for state mutations; 3 new async methods + 3 sync wrappers; `EndpointNotFoundError` |
| `dephealth/dependency.py` | Export `EndpointNotFoundError` if needed |
| `dephealth/api.py` | 3 new async facade methods + 3 sync wrappers with validation |
| `dephealth/__init__.py` | Export new methods and exception |
| `tests/test_scheduler.py` | New tests for dynamic Add/Remove/Update + concurrency |
| `tests/test_api.py` | New integration tests for facade methods |
| `pyproject.toml` | Bump `0.5.0` → `0.6.0` |
| `sdk-python/docs/api-reference.md` | Document new methods (EN) |
| `sdk-python/docs/api-reference.ru.md` | Document new methods (RU) |
| `sdk-python/README.md` | Add dynamic endpoints section |
| `docs/migration/sdk-python-v050-to-v060.md` | Migration guide (EN) |
| `docs/migration/sdk-python-v050-to-v060.ru.md` | Migration guide (RU) |
| `CHANGELOG.md` | Add `[sdk-python 0.6.0]` section |

## Phases

---

### Phase 1: Scheduler infrastructure — global config, per-endpoint task tracking, lock

Prepare `CheckScheduler` for dynamic mutations without changing the public API yet.

**Modify `dephealth/scheduler.py`:**

- [x] Add `_global_config: CheckConfig` field to `CheckScheduler` (passed at construction)
- [x] Add `_lock: threading.Lock` field — protects `_entries` and `_states` dict during mutations
- [x] Refactor internal state: add `_states: dict[str, _EndpointState]` keyed by `"name:host:port"`
  alongside existing `_entries` (or replace `_entries` entirely with flat state dict)
- [x] Add `_tasks: dict[str, asyncio.Task]` for asyncio mode — per-endpoint task tracking
- [x] Add `_threads: dict[str, threading.Thread]` for threading mode — per-endpoint thread tracking
- [x] Add `_stop_events: dict[str, threading.Event]` for threading mode — per-endpoint stop signal
- [x] In `start()`: store each `asyncio.create_task()` result in `_tasks` dict
- [x] In `start_sync()`: store each `threading.Thread` in `_threads` dict, each stop event in `_stop_events`
- [x] Wrap `health()` and `health_details()` state iteration with `self._lock`

**Modify `dephealth/api.py`:**

- [x] In `__init__()`: compute `global_config` from `check_interval`/`check_timeout` parameters
  with `CheckConfig` defaults
- [x] Pass `global_config` to `CheckScheduler` constructor

**Validation:**

- [ ] `pytest` passes (existing tests, no behavioral change)
- [ ] `ruff check` passes
- [ ] `mypy` passes (if configured)

**Status:** done

---

### Phase 2: Dynamic methods on CheckScheduler

Implement the three core methods on `CheckScheduler`, both async and sync variants.

**Add to `dephealth/scheduler.py`:**

- [x] `async def add_endpoint(self, dep_name, dep_type, critical, ep, checker)`
  - Acquire `self._lock`
  - Check `_started and not _stopped`, else raise `RuntimeError("Scheduler not started")`
  - Compute key `dep_name:host:port`, return if exists (idempotent)
  - Build `Dependency` with `self._global_config`
  - Create `_EndpointState` with initial fields
  - Insert into `_states`
  - If asyncio mode: create task via `asyncio.create_task(_run_loop(...))`, store in `_tasks`
  - If threading mode: create thread + stop event, start thread, store in dicts
- [x] `async def remove_endpoint(self, dep_name, host, port)`
  - Acquire `self._lock`
  - Check `_started`, else raise `RuntimeError`
  - Find state by key, return if not found (idempotent)
  - If asyncio mode: cancel task (`task.cancel()`), remove from `_tasks`
  - If threading mode: set stop event, join thread (with timeout), remove from dicts
  - Remove from `_states`
  - Call `self._metrics.delete_metrics(dep, ep)` to clean up Prometheus series
- [x] `async def update_endpoint(self, dep_name, old_host, old_port, new_ep, checker)`
  - Acquire `self._lock`
  - Check `_started and not _stopped`
  - Find old state, raise `EndpointNotFoundError` if missing
  - Cancel old task/thread, remove old state, delete old metrics
  - Create new state, new task/thread, insert
- [x] Sync wrappers: `add_endpoint_sync()`, `remove_endpoint_sync()`, `update_endpoint_sync()`
  - Direct implementation for threading mode

**Add `EndpointNotFoundError`:**

- [x] `class EndpointNotFoundError(Exception)` in `scheduler.py`
  - Constructor: `(dep_name: str, host: str, port: str)`
  - Message: `f"Endpoint not found: {dep_name}:{host}:{port}"`

**Validation:**

- [ ] `pytest` passes (existing tests)
- [ ] `ruff check` passes

**Status:** done

---

### Phase 3: DependencyHealth facade methods

Thin wrappers with input validation, delegating to CheckScheduler.

**Modify `dephealth/api.py`:**

- [x] Add `async def add_endpoint(self, dep_name, dep_type, critical, ep, checker)`
  - Validate `dep_name` via `validate_name()` from dependency.py
  - Validate `dep_type` is valid `DependencyType`
  - Validate `ep.host` and `ep.port` non-empty
  - Validate `ep.labels` via existing label validation
  - Delegate to `self._scheduler.add_endpoint()`
- [x] Add `async def remove_endpoint(self, dep_name, host, port)`
  - Passthrough to `self._scheduler.remove_endpoint()`
- [x] Add `async def update_endpoint(self, dep_name, old_host, old_port, new_ep, checker)`
  - Validate `new_ep.host`, `new_ep.port`, `new_ep.labels`
  - Delegate to `self._scheduler.update_endpoint()`
- [x] Sync variants: `add_endpoint_sync()`, `remove_endpoint_sync()`, `update_endpoint_sync()`

**Modify `dephealth/__init__.py`:**

- [x] Export `EndpointNotFoundError` in `__all__`

**Validation:**

- [ ] `pytest` passes
- [ ] `ruff check` passes

**Status:** done

---

### Phase 4: Tests — CheckScheduler level

Unit tests for the three new CheckScheduler methods. Use existing test patterns
(`AsyncMock`, `_make_dep()`, `_make_scheduler()`).

**Add to `tests/test_scheduler.py`:**

- [x] `test_add_endpoint` — add endpoint after start, wait, verify `health()` includes it
- [x] `test_add_endpoint_idempotent` — add same endpoint twice, no error, single entry
- [x] `test_add_endpoint_before_start` — raises `RuntimeError`
- [x] `test_add_endpoint_after_stop` — raises `RuntimeError`
- [x] `test_add_endpoint_metrics` — verify health gauge appears with correct labels
- [x] `test_remove_endpoint` — remove after start, verify disappears from `health()`
- [x] `test_remove_endpoint_idempotent` — remove non-existent, no error
- [x] `test_remove_endpoint_metrics_deleted` — verify all metric series removed
- [x] `test_remove_endpoint_before_start` — raises `RuntimeError`
- [x] `test_update_endpoint` — update, verify old gone and new appears in `health()`
- [x] `test_update_endpoint_not_found` — raises `EndpointNotFoundError`
- [x] `test_update_endpoint_metrics_swap` — old metrics deleted, new metrics present
- [x] `test_stop_after_dynamic_add` — add endpoint, then `stop()`, verify clean shutdown
- [x] `test_concurrent_add_remove_health` — run Add/Remove/Health from multiple tasks
- [x] sync mode tests: add/remove/update_endpoint_sync + update_not_found

**Validation:**

- [ ] `pytest` passes (all tests)
- [ ] No warnings or flaky tests

**Status:** done

---

### Phase 5: Tests — DependencyHealth facade level

Integration tests for the public API.

**Add to `tests/test_api.py`:**

- [x] `test_add_endpoint` — create DependencyHealth, start, add_endpoint, verify `health()`
- [x] `test_add_endpoint_invalid_name` — invalid dep name, raises `ValueError`
- [x] `test_add_endpoint_invalid_type` — unknown type, raises `ValueError`
- [x] `test_add_endpoint_missing_host` — empty host, raises `ValueError`
- [x] `test_add_endpoint_missing_port` — empty port, raises `ValueError`
- [x] `test_add_endpoint_reserved_label` — reserved label, raises `ValueError`
- [x] `test_remove_endpoint` — remove, verify gone from `health()`
- [x] `test_update_endpoint` — update, verify old gone and new present
- [x] `test_update_endpoint_missing_new_host` — empty new host, raises `ValueError`
- [x] `test_update_endpoint_not_found` — raises `EndpointNotFoundError`
- [x] `test_add_endpoint_inherits_global_config` — verify dynamic endpoint uses global interval/timeout

**Validation:**

- [ ] `pytest` passes (all tests)
- [ ] `ruff check` passes

**Status:** done

---

### Phase 6: Version bump, documentation, changelog

**Version bump:**

- [ ] `pyproject.toml` → `version = "0.6.0"`

**Documentation (EN):**

- [ ] Update `sdk-python/docs/api-reference.md` — add `add_endpoint`, `remove_endpoint`, `update_endpoint`
- [ ] Update `sdk-python/README.md` — add "Dynamic Endpoints" section with usage example
- [ ] Create `docs/migration/sdk-python-v050-to-v060.md` — migration guide

**Documentation (RU):**

- [ ] Update `sdk-python/docs/api-reference.ru.md` — same as EN
- [ ] Create `docs/migration/sdk-python-v050-to-v060.ru.md` — migration guide

**Changelog:**

- [ ] Update `CHANGELOG.md` — add `[sdk-python 0.6.0]` section

**Validation:**

- [ ] `pytest` — all pass
- [ ] `markdownlint` on all new/changed `.md` files — 0 issues

**Status:** not started

---

### Phase 7: Merge, tag, release

**Pre-merge checklist:**

- [ ] All unit tests pass (including new dynamic endpoint tests)
- [ ] No concurrency issues in tests
- [ ] All linters pass (`ruff`, `mypy`)
- [ ] Backward compatibility verified (existing API unchanged)
- [ ] Docs complete (EN + RU)
- [ ] CHANGELOG updated

**Actions:**

- Merge to master (or PR — ask user)
- Tag: `sdk-python/v0.6.0`
- GitHub Release: sdk-python/v0.6.0
- publish in pypi
- Move this plan to `plans/archive/`

**Status:** not started
