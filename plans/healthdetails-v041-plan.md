# Implementation Plan: HealthDetails() API + Release v0.4.1

> Source: `.tasks/sdk_feature_healthdetails_requirements.md`
>
> Status: **IN PROGRESS**

---

## Overview

Add `HealthDetails()` public API to all 4 SDKs, add conformance tests,
update documentation, and release v0.4.1.

**Branch**: `feature/health-details`
**Target version**: 0.4.1 (all SDKs)

---

## Phase 1: Specification Update ✅

> Scope: Add HealthDetails() API contract to `spec/`
> Estimated effort: Small (2 files)
> **Status: COMPLETED**

### Tasks

1. ✅ Add new section to `spec/check-behavior.md`:
   - Section 8 "Programmatic Health Details API" (10 subsections)
   - Define `EndpointStatus` structure (11 fields, language-agnostic)
   - Define `StatusCategory` typed alias with values (8 existing + `unknown`)
   - Define key format: `"dependency:host:port"`
   - Define behavior: before Start(), UNKNOWN state, after Stop()
   - Define JSON serialization contract

2. ✅ Add same section to `spec/check-behavior.ru.md` (Russian translation)

3. ✅ Lint markdown files — clean

### Acceptance criteria
- [x] Spec clearly defines all 11 fields of EndpointStatus
- [x] StatusCategory type documented with all 9 values
- [x] Behavior for UNKNOWN state documented
- [x] JSON serialization example included
- [x] Both EN and RU versions

---

## Phase 2: Go SDK Implementation (reference) ✅

> Scope: `sdk-go/dephealth/`
> Estimated effort: Medium
> **Status: COMPLETED**

### Tasks

1. ✅ **Add `StatusCategory` type** (`check_result.go`):
   - `type StatusCategory string` with typed constants
   - `ClassifiedError` interface and `ClassifiedCheckError` updated to use `StatusCategory`
   - Added `StatusUnknown StatusCategory = "unknown"`
   - `AllStatusCategories` now `[]StatusCategory` (excludes `StatusUnknown`)

2. ✅ **Add `EndpointStatus` struct** (new file `endpoint_status.go`):
   - 11 fields with Go-idiomatic types
   - `LatencyMillis() float64` helper
   - Custom `MarshalJSON` / `UnmarshalJSON` for `latency_ms` and nullable `last_checked_at`

3. ✅ **Extend `endpointState`** (`scheduler.go`):
   - Dynamic fields: `lastStatus`, `lastDetail`, `lastLatency`, `lastCheckedAt`
   - Static fields: `depName`, `depType`, `host`, `port`, `critical`, `labels`

4. ✅ **Store results in `executeCheck()`** — after `classifyError()`

5. ✅ **Set static fields in `Start()`** — with `StatusUnknown` / `"unknown"` defaults

6. ✅ **Add `Scheduler.HealthDetails()`** — same lock pattern as `Health()`

7. ✅ **Add `DepHealth.HealthDetails()`** — delegate to scheduler

8. ✅ **Unit tests** (`endpoint_status_test.go` — 14 tests):
   - BeforeStart, UnknownState, HealthyEndpoint, UnhealthyEndpoint
   - KeysMatchHealth, ConcurrentAccess, AfterStop, LabelsEmpty
   - ResultMapIndependent, DepHealth facade
   - JSON Healthy, JSON Unknown, JSON Roundtrip, LatencyMillis

9. ✅ **Lint**: `make lint` — 0 issues; `make test` — all pass with `-race`

### Acceptance criteria
- [x] `HealthDetails()` returns `map[string]EndpointStatus`
- [x] All 11 fields populated correctly
- [x] UNKNOWN endpoints included with `Status="unknown"`
- [x] Keys match `Health()` output
- [x] All tests pass (4 packages, including `-race`)
- [x] Linter clean (0 issues)

---

## Phase 3: Java SDK Implementation ✅

> Scope: `sdk-java/dephealth-core/`
> Estimated effort: Medium
> **Status: COMPLETED**

### Tasks

1. ✅ **Add `StatusCategory.UNKNOWN` constant** (`StatusCategory.java`):
   - Values: OK, TIMEOUT, CONNECTION_ERROR, DNS_ERROR, AUTH_ERROR,
     TLS_ERROR, UNHEALTHY, ERROR, UNKNOWN
   - `String value()` method returning lowercase string

2. ✅ **Add `EndpointStatus` class** (new file `EndpointStatus.java`):
   - 11 fields matching Go struct
   - Java types: `Boolean` healthy, `Duration` latency, `Instant` lastCheckedAt,
     `Map<String, String>` labels
   - Immutable final class with public constructor, `latencyMillis()` helper

3. ✅ **Extend `EndpointState`**:
   - Add synchronized fields: `lastStatus`, `lastDetail`, `lastLatency`, `lastCheckedAt`
   - Add static fields: `depName`, `depType`, `host`, `port`, `critical`, `labels`
   - Add `toEndpointStatus()` method, `setStaticFields()`, `storeCheckResult()`

4. ✅ **Store results in `runCheck()`** (`CheckScheduler.java`):
   - After `ErrorClassifier.classify()`: `state.storeCheckResult(category, detail, duration)`

5. ✅ **Set static fields** in `addDependency()` via `state.setStaticFields()`

6. ✅ **Add `CheckScheduler.healthDetails()`**:
   - Return `Map<String, EndpointStatus>` (LinkedHashMap)
   - Include UNKNOWN endpoints

7. ✅ **Add `DepHealth.healthDetails()`**:
   - Delegate to scheduler

8. ✅ **Unit tests** (`HealthDetailsTest.java` — 10 tests + 1 in `DepHealthTest`):
   - emptyBeforeAddingDependencies, unknownStateBeforeFirstCheck, healthyEndpoint
   - unhealthyEndpoint, keysMatchHealth, concurrentAccess, afterStop
   - labelsEmptyWhenNotSet, resultMapIsIndependent, latencyMillis
   - DepHealth.healthDetailsFacade

9. ✅ **Lint**: Checkstyle 0 violations, SpotBugs 0 bugs

### Acceptance criteria
- [x] Same behavior as Go SDK
- [x] All tests pass (176 total, 0 failures)
- [x] Checkstyle + SpotBugs clean (0 violations, 0 bugs)

---

## Phase 4: Python SDK Implementation

> Scope: `sdk-python/dephealth/`
> Estimated effort: Medium

### Tasks

1. **Add `StatusCategory` type** (str enum or Literal type):
   - In `check_result.py` or new `endpoint_status.py`

2. **Add `EndpointStatus` frozen dataclass**:
   - 11 fields with Python types: `bool | None`, `float` (seconds),
     `datetime | None`, `dict[str, str]`

3. **Extend `_EndpointState`** (`scheduler.py`):
   - Add fields: `last_status`, `last_detail`, `last_latency`, `last_checked_at`
   - Add static fields: `dep_name`, `dep_type`, `host`, `port`, `critical`, `labels`

4. **Store results in `_check_once()`**:
   - After `classify_error()`: store category, detail, duration, datetime.now(UTC)

5. **Set static fields** when creating `_EndpointState`

6. **Add `CheckScheduler.health_details()`**:
   - Return `dict[str, EndpointStatus]`
   - Key format: `"dependency:host:port"` (aligned with Go/Java/C#)
   - Include UNKNOWN endpoints

7. **Add `DependencyHealth.health_details()`**:
   - Delegate to scheduler

8. **Export in `__all__`** (`__init__.py`):
   - `EndpointStatus`, `StatusCategory`

9. **Unit tests**

10. **Lint**: `make lint` in `sdk-python/` (ruff + mypy --strict)

### Acceptance criteria
- [ ] Same behavior as Go SDK
- [ ] Key format `"dep:host:port"` (not aggregated like `health()`)
- [ ] Frozen dataclass with type hints
- [ ] All tests pass
- [ ] ruff + mypy clean

---

## Phase 5: C# SDK Implementation

> Scope: `sdk-csharp/DepHealth.Core/`
> Estimated effort: Medium

### Tasks

1. **Add `StatusCategory` type** (enum or static class with constants):
   - Values matching Go/Java

2. **Add `EndpointStatus` record/class**:
   - 11 fields: `bool?` Healthy, `TimeSpan` Latency, `DateTimeOffset?` LastCheckedAt,
     `Dictionary<string, string>` Labels

3. **Extend `EndpointState`**:
   - Add locked fields: LastStatus, LastDetail, LastLatency, LastCheckedAt
   - Add static fields: DepName, DepType, Host, Port, Critical, Labels

4. **Store results in `RunCheck()`** (`CheckScheduler.cs`):
   - After `ErrorClassifier.Classify()`: store results

5. **Set static fields** when creating EndpointState

6. **Add `CheckScheduler.HealthDetails()`**

7. **Add `DepHealthMonitor.HealthDetails()`**

8. **Unit tests**

9. **Lint**: `make lint` in `sdk-csharp/`

### Acceptance criteria
- [ ] Same behavior as Go SDK
- [ ] All tests pass
- [ ] dotnet format clean

---

## Phase 6: Conformance Tests

> Scope: `conformance/`
> Estimated effort: Medium-Large
>
> HealthDetails() is a programmatic API, not a metrics endpoint.
> To test cross-SDK consistency, each test service exposes a
> `/health-details` HTTP endpoint returning JSON.

### Tasks

#### 6.1. Update test services (add `/health-details` endpoint)

**Go** (`conformance/test-service/main.go`):
- Add HTTP handler for `/health-details`
- Call `dh.HealthDetails()`, marshal to JSON, return
- JSON keys: snake_case (matching Go struct tags)

**Java** (`conformance/test-service-java/`):
- Add REST endpoint `GET /health-details`
- Call `depHealth.healthDetails()`, serialize to JSON
- Ensure same JSON key naming as Go (snake_case)

**Python** (`conformance/test-service-python/`):
- Add route `GET /health-details`
- Call `dh.health_details()`, serialize to JSON
- Use `dataclasses.asdict()` or custom serializer

**C#** (`conformance/test-service-csharp/`):
- Add endpoint `GET /health-details`
- Call `monitor.HealthDetails()`, serialize to JSON
- Configure snake_case naming policy

#### 6.2. Add new check types to `verify.py`

New check types for `health-details.yml` scenario:

| Check type | Description |
|---|---|
| `health_details_endpoint_exists` | `/health-details` returns HTTP 200 with JSON |
| `health_details_structure` | Each entry has all 11 required fields |
| `health_details_types` | Field types are correct (healthy=bool/null, latency=number, etc.) |
| `health_details_consistency` | Keys match `/metrics` health metric endpoints |
| `health_details_status_values` | Status values match allowed StatusCategory values |
| `health_details_expected` | Specific endpoints have expected status/detail/healthy values |

#### 6.3. Create scenario `scenarios/health-details.yml`

```yaml
name: "health-details"
description: "HealthDetails() API returns correct endpoint data"

checks:
  - type: health_details_endpoint_exists
  - type: health_details_structure
  - type: health_details_types
  - type: health_details_consistency
  - type: health_details_status_values
  - type: health_details_expected
    endpoints:
      - dependency: postgres-primary
        host: postgres-primary.dephealth-conformance.svc
        port: 5432
        healthy: true
        status: ok
        detail: ok
        type: postgres
        critical: true
      # ... all 7 dependencies
```

#### 6.4. Build updated test service images

- `make build` for each test service (linux/amd64)
- Push to `harbor.kryukov.lan/library`

### Acceptance criteria
- [ ] All 4 test services expose `/health-details`
- [ ] JSON format consistent across all 4 SDKs
- [ ] New scenario passes for all 4 SDKs
- [ ] Existing scenarios still pass (regression)

---

## Phase 7: Run Conformance Tests

> Scope: Deploy and test in k8s cluster
> Estimated effort: Manual execution + debugging

### Tasks

1. Build and push all 4 test service images
2. Run `./conformance/run.sh --lang all` (all existing scenarios)
3. Run `./conformance/run.sh --lang all --scenario health-details`
4. Fix any failures
5. Run cross-verify: `cross_verify.py` (all 4 SDKs produce identical structure)

### Acceptance criteria
- [ ] All 8 existing scenarios: PASS for all 4 SDKs
- [ ] New health-details scenario: PASS for all 4 SDKs
- [ ] Cross-verify: JSON structure identical across SDKs

---

## Phase 8: Documentation

> Scope: Update docs for HealthDetails() API
> Estimated effort: Medium (many files, templated)
> **Prerequisite**: Phase 7 complete (successful testing)

### Tasks

1. **Quickstart guides** (8 files):
   - `docs/quickstart/go.md` + `go.ru.md`
   - `docs/quickstart/java.md` + `java.ru.md`
   - `docs/quickstart/python.md` + `python.ru.md`
   - `docs/quickstart/csharp.md` + `csharp.ru.md`
   - Add section: "Getting detailed health status"
   - Code example for each language

2. **Migration guides** (8 files):
   - `docs/migration/go.md` + `go.ru.md`
   - `docs/migration/java.md` + `java.ru.md`
   - `docs/migration/python.md` + `python.ru.md`
   - `docs/migration/csharp.md` + `csharp.ru.md`
   - Add "v0.4.1: HealthDetails() API" section
   - Note: Python `health_details()` key format differs from `health()`

3. **Specification overview** (`docs/specification.md` + `.ru.md`):
   - Add HealthDetails() to "SDK API" section

4. **Root README.md**:
   - Add HealthDetails() mention in features list

5. **CHANGELOG.md**:
   - v0.4.1 entry for all 4 SDKs

6. **SDK READMEs** (if they exist):
   - `sdk-go/README.md`, `sdk-java/README.md`, etc.
   - Add HealthDetails() usage example

7. Lint all markdown files

### Acceptance criteria
- [ ] All quickstart guides updated (4 langs × 2 languages)
- [ ] All migration guides updated
- [ ] CHANGELOG.md has v0.4.1 entries
- [ ] README.md mentions HealthDetails()
- [ ] Markdownlint clean

---

## Phase 9: Release v0.4.1

> Scope: Version bump, tags, publish
> Estimated effort: Small
> **Prerequisite**: Phase 8 complete

### Tasks

1. **Bump versions**:
   - `sdk-java/pom.xml`: `0.4.0` → `0.4.1`
   - `sdk-python/pyproject.toml`: `0.4.0` → `0.4.1`
   - `sdk-csharp/Directory.Build.props`: `0.4.0` → `0.4.1`
   - Go: no version file (version from git tag)

2. **Merge feature branch to master**:
   - Final review
   - Merge commit or squash (ask user)

3. **Create per-SDK tags**:
   - `sdk-go/v0.4.1`
   - `sdk-java/v0.4.1`
   - `sdk-python/v0.4.1`
   - `sdk-csharp/v0.4.1`
   - **NO common `v0.4.1` tag**

4. **Create GitHub Releases** (4 releases):
   - One per SDK tag
   - Release notes from CHANGELOG.md

5. **Publish**:
   - PyPI: `make publish` in `sdk-python/`
   - Maven Central: `make publish` in `sdk-java/`
   - NuGet: still TODO (no publish target)

6. **Push tags**: `git push origin --tags`

### Acceptance criteria
- [ ] All 4 SDK versions bumped to 0.4.1
- [ ] 4 per-SDK tags created
- [ ] 4 GitHub Releases created
- [ ] PyPI: dephealth 0.4.1 published
- [ ] Maven Central: dephealth-core + spring-boot-starter 0.4.1 published

---

## Phase Dependencies

```
Phase 1 (spec)
    ↓
Phase 2 (Go SDK) ← reference implementation
    ↓
Phase 3 (Java SDK)
    ↓
Phase 4 (Python SDK)
    ↓
Phase 5 (C# SDK)
    ↓
Phase 6 (conformance tests)
    ↓
Phase 7 (run tests) ← manual, in k8s
    ↓
Phase 8 (documentation) ← only after successful tests
    ↓
Phase 9 (release 0.4.1)
```

> Note: Phases 3-5 are sequential (each follows Go pattern)
> but could be parallelized if working with multiple agents.

---

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Python `health_details()` key format differs from `health()` | Document clearly in migration guide |
| Conformance JSON format inconsistency across SDKs | Cross-verify check, strict schema |
| JSON serialization edge cases (Duration, Time) | Define canonical format in spec |
| `StatusCategory` type change may affect existing code | Alias existing constants, backward compatible |
| Conformance test flakiness (k8s timing) | Add wait/retry in verify.py for `/health-details` |
