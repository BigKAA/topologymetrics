# Plan: LDAP Health Checker — All SDKs (v0.8.0)

## Goal

Implement LDAP health checker across all 4 SDKs (Go, Java, Python, C#) with full
protocol support (LDAP, LDAPS, StartTLS), four check methods (anonymous bind,
simple bind, RootDSE query, search), and bump all SDK versions to 0.8.0.

## Current state

- All 4 SDKs have 8 identical health checkers (HTTP, gRPC, TCP, Postgres, MySQL, Redis, AMQP, Kafka)
- Go SDK: v0.7.0, Java/Python/C#: v0.6.0
- LDAP checker listed in TODO.md for all SDKs (not implemented)
- No LDAP-related code exists anywhere in the codebase
- Spec files (`check-behavior.md`, `config-contract.md`) don't mention LDAP

## LDAP libraries

| SDK | Library | Package/Dependency |
|-----|---------|-------------------|
| Go | `go-ldap/ldap/v3` | `github.com/go-ldap/ldap/v3` |
| Java | UnboundID LDAP SDK | `com.unboundid:unboundid-ldapsdk` |
| Python | ldap3 | `ldap3` |
| C# | Novell LDAP | `Novell.Directory.Ldap.NETStandard` (4.0.0) |

## DependencyType

- New type: `ldap`
- Scheme mapping: `ldap://` → `ldap`, `ldaps://` → `ldap`
- Default ports: `ldap://` → `389`, `ldaps://` → `636`

## LDAP checker configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `checkMethod` | enum | `root_dse` | Check method: `anonymous_bind`, `simple_bind`, `root_dse`, `search` |
| `bindDN` | string | `""` | DN for Simple Bind |
| `bindPassword` | string | `""` | Password for Simple Bind |
| `baseDN` | string | `""` | Base DN for search method |
| `searchFilter` | string | `(objectClass=*)` | LDAP filter for search method |
| `searchScope` | enum | `base` | Scope: `base`, `one`, `sub` |
| `startTLS` | bool | `false` | Use StartTLS (only with `ldap://`) |
| `tlsSkipVerify` | bool | `false` | Skip TLS certificate verification |

## Check algorithm (standalone mode)

1. Connect TCP to `{host}:{port}`
2. If `ldaps://` — TLS handshake
3. If `startTLS=true` — send StartTLS extended operation, then TLS handshake
4. Execute check based on `checkMethod`:
   - `anonymous_bind`: Anonymous Bind operation
   - `simple_bind`: Simple Bind with `bindDN`/`bindPassword`
   - `root_dse`: Search base=`""`, scope=base, filter=`(objectClass=*)`
   - `search`: Search with `baseDN`, `searchScope`, `searchFilter`
5. Success if operation completes without error
6. Close connection

## Error classification

| Detail | Condition |
|--------|-----------|
| `ok` | Check succeeded |
| `timeout` | Connection/operation timeout |
| `connection_refused` | TCP connection refused |
| `dns_error` | DNS resolution failed |
| `auth_error` | LDAP error 49 (Invalid Credentials), error 50 (Insufficient Access Rights) |
| `tls_error` | TLS/StartTLS handshake failure |
| `unhealthy` | LDAP operational error (server down, busy, unavailable) |
| `error` | Other errors |

## Validation rules

- `simple_bind` requires `bindDN` + `bindPassword` — otherwise config error
- `search` requires `baseDN` — otherwise config error
- `startTLS=true` with `ldaps://` — config error (incompatible)
- LDAP referrals: do not follow

## Test container

- Image: `389ds/dirsrv:3.1` (via `harbor.kryukov.lan/docker/389ds/dirsrv:3.1`)
- Base DN: `dc=test,dc=local`
- Admin DN: `cn=Directory Manager`
- Admin password: `password`
- Ports: 3389 (LDAP), 3636 (LDAPS)

## Files changed (per SDK)

### Spec

| File | Change |
|------|--------|
| `spec/check-behavior.md` | Add section 4.9 LDAP checker |
| `spec/check-behavior.ru.md` | Add section 4.9 LDAP checker (RU) |
| `spec/config-contract.md` | Add `ldap://`, `ldaps://` to scheme table, default ports, env vars |
| `spec/config-contract.ru.md` | Same (RU) |

### Go SDK

| File | Change |
|------|--------|
| `sdk-go/dephealth/options.go` | Add `TypeLDAP`, `LDAP()` factory, LDAP-specific options |
| `sdk-go/dephealth/parser.go` | Add `ldap`/`ldaps` scheme mapping and default ports |
| `sdk-go/dephealth/checks/ldapcheck/ldapcheck.go` | New LDAP checker |
| `sdk-go/dephealth/checks/ldapcheck/ldapcheck_test.go` | Unit tests |
| `sdk-go/go.mod` | Add `github.com/go-ldap/ldap/v3` |
| `sdk-go/dephealth/version.go` | Bump `0.7.0` → `0.8.0` |

### Java SDK

| File | Change |
|------|--------|
| `sdk-java/dephealth-core/pom.xml` | Add UnboundID dependency |
| `sdk-java/pom.xml` | Version `0.6.0` → `0.8.0` |
| `sdk-java/.../DependencyType.java` | Add `LDAP("ldap")` |
| `sdk-java/.../ConfigParser.java` | Add `ldap`/`ldaps` scheme mapping and default ports |
| `sdk-java/.../checks/LdapHealthChecker.java` | New LDAP checker |
| `sdk-java/.../checks/LdapHealthCheckerTest.java` | Unit tests |

### Python SDK

| File | Change |
|------|--------|
| `sdk-python/pyproject.toml` | Add `ldap3` dependency, version `0.6.0` → `0.8.0` |
| `sdk-python/dephealth/dependency.py` | Add `LDAP = "ldap"` |
| `sdk-python/dephealth/parser.py` | Add `ldap`/`ldaps` scheme mapping and default ports |
| `sdk-python/dephealth/checks/ldap.py` | New LDAP checker |
| `sdk-python/dephealth/checks/__init__.py` | Export `LdapChecker` |
| `sdk-python/tests/test_ldap.py` | Unit tests |

### C# SDK

| File | Change |
|------|--------|
| `sdk-csharp/DepHealth.Core/DepHealth.Core.csproj` | Add `Novell.Directory.Ldap.NETStandard` |
| `sdk-csharp/Directory.Build.props` | Version `0.6.0` → `0.8.0` |
| `sdk-csharp/DepHealth.Core/DependencyType.cs` | Add `Ldap` |
| `sdk-csharp/DepHealth.Core/ConfigParser.cs` | Add `ldap`/`ldaps` scheme mapping and default ports |
| `sdk-csharp/DepHealth.Core/Checks/LdapChecker.cs` | New LDAP checker |
| `sdk-csharp/tests/.../LdapCheckerTests.cs` | Unit tests |

## Phases

---

### Phase 1: Branch, spec update

Create feature branch, update specification files.

**Branch:**

- [x] Create branch `feature/ldap-checker` from `master`

**Modify `spec/check-behavior.md`:**

- [x] Add section `4.9. LDAP (type: ldap)` with parameter table, algorithm, specifics
- [x] Add LDAP to error classification table in section 6.2.3 (detail values)

**Modify `spec/check-behavior.ru.md`:**

- [x] Same changes (Russian translation)

**Modify `spec/config-contract.md`:**

- [x] Add `ldap://` and `ldaps://` to scheme table (section 2.1)
- [x] Add `ldap` to default port table (section 6): port 389 and 636
- [x] Add LDAP URL examples to section 2.4
- [x] Add `LDAP()` to factory methods list (section 7.1)
- [x] Add LDAP-specific options to section 7.3
- [x] Add LDAP env vars to section 8.2

**Modify `spec/config-contract.ru.md`:**

- [x] Same changes (Russian translation)

**Validation:**

- [x] `markdownlint` passes on all modified files

**Status:** done

---

### Phase 2: Go SDK — LDAP checker implementation

Implement LDAP checker for Go SDK.

**Add dependency:**

- [x] Add `github.com/go-ldap/ldap/v3` to `sdk-go/go.mod`

**Modify `sdk-go/dephealth/options.go`:**

- [x] Add `TypeLDAP DependencyType = "ldap"` constant
- [x] Add `LDAP()` factory function
- [x] Add LDAP-specific options: `WithLDAPCheckMethod`, `WithLDAPBindDN`, `WithLDAPBindPassword`, `WithLDAPBaseDN`, `WithLDAPSearchFilter`, `WithLDAPSearchScope`, `WithLDAPStartTLS`
- [x] Add LDAP fields to `DependencyConfig`

**Modify `sdk-go/dephealth/parser.go`:**

- [x] Add `ldap` → `TypeLDAP`, `ldaps` → `TypeLDAP` to scheme mapping
- [x] Add default ports: `ldap` → 389, `ldaps` → 636

**Create `sdk-go/dephealth/checks/ldapcheck/ldapcheck.go`:**

- [x] Implement `LdapChecker` struct with `Check(ctx, Endpoint) error` and `Type() string`
- [x] Factory `NewFromConfig(cfg dephealth.DependencyConfig) dephealth.HealthChecker`
- [x] `init()` with `RegisterCheckerFactory(TypeLDAP, NewFromConfig)`
- [x] Standalone mode: connect, optional StartTLS, execute check method, close
- [x] Pool mode: accept existing `*ldap.Conn`
- [x] Error classification: auth errors (LDAP result code 49, 50), TLS errors, connection errors
- [x] Options: `WithConn`, `WithCheckMethod`, `WithBindDN`, `WithBindPassword`, etc.

**Create `sdk-go/dephealth/checks/ldapcheck/ldapcheck_test.go`:**

- [x] Test RootDSE check (default) — mock or real LDAP
- [x] Test anonymous bind
- [x] Test simple bind with valid credentials
- [x] Test simple bind with invalid credentials (auth_error)
- [x] Test search method with baseDN
- [x] Test StartTLS
- [x] Test connection refused
- [x] Test `NewFromConfig` with URL parsing
- [x] Test validation: `simple_bind` without credentials, `search` without baseDN, `startTLS` with `ldaps://`

**Validation:**

- [x] `make build` passes (in sdk-go/)
- [x] `make test` passes
- [x] `make lint` passes

**Status:** done

---

### Phase 3: Java SDK — LDAP checker implementation

Implement LDAP checker for Java SDK.

**Add dependency:**

- [x] Add `com.unboundid:unboundid-ldapsdk` to `sdk-java/dephealth-core/pom.xml`

**Modify `DependencyType.java`:**

- [x] Add `LDAP("ldap")` enum value

**Modify `ConfigParser.java`:**

- [x] Add `ldap` → `LDAP`, `ldaps` → `LDAP` to scheme mapping
- [x] Add default ports: `ldap` → 389, `ldaps` → 636

**Create `LdapHealthChecker.java`:**

- [x] Implement `HealthChecker` interface
- [x] Builder pattern for configuration (checkMethod, bindDN, bindPassword, baseDN, etc.)
- [x] Standalone mode: `LDAPConnection`, optional StartTLS, check method, close
- [x] Pool mode: accept existing `LDAPConnection`
- [x] Error classification: `CheckAuthException`, `CheckConnectionException`

**Create `LdapHealthCheckerTest.java`:**

- [x] Unit tests with Mockito (mock LDAP connection)
- [x] Test all 4 check methods
- [x] Test error classification
- [x] Test validation rules

**Version bump:**

- [x] `sdk-java/pom.xml` → `0.8.0`
- [x] All child `pom.xml` → `0.8.0`

**Validation:**

- [x] `mvn compile` passes
- [x] `mvn test` passes
- [x] `mvn checkstyle:check` passes (if configured)

**Status:** done

---

### Phase 4: Python SDK — LDAP checker implementation

Implement LDAP checker for Python SDK.

**Add dependency:**

- [x] Add `ldap3` to `sdk-python/pyproject.toml` dependencies

**Modify `sdk-python/dephealth/dependency.py`:**

- [x] Add `LDAP = "ldap"` to `DependencyType`

**Modify `sdk-python/dephealth/parser.py`:**

- [x] Add `ldap` → `LDAP`, `ldaps` → `LDAP` to scheme mapping
- [x] Add default ports: `ldap` → 389, `ldaps` → 636

**Create `sdk-python/dephealth/checks/ldap.py`:**

- [x] `LdapChecker` class with `async check(endpoint)` and `checker_type() -> str`
- [x] Constructor: `timeout`, `check_method`, `bind_dn`, `bind_password`, `base_dn`, `search_filter`, `search_scope`, `start_tls`, `tls_skip_verify`, `client`
- [x] Standalone: create `ldap3.Connection` in executor (ldap3 is synchronous)
- [x] Pool mode: accept existing connection
- [x] Error classification: `CheckAuthError`, `CheckConnectionRefusedError`, `CheckTlsError`

**Modify `sdk-python/dephealth/checks/__init__.py`:**

- [x] Export `LdapChecker`

**Create `sdk-python/tests/test_ldap.py`:**

- [x] Unit tests for all 4 check methods
- [x] Test error classification
- [x] Test validation rules

**Version bump:**

- [x] `sdk-python/pyproject.toml` version → `0.8.0`

**Validation:**

- [x] `pytest` passes
- [x] `ruff check` passes
- [x] `mypy` passes (if configured)

**Status:** done

---

### Phase 5: C# SDK — LDAP checker implementation

Implement LDAP checker for C# SDK.

**Add dependency:**

- [x] Add `Novell.Directory.Ldap.NETStandard` to `DepHealth.Core.csproj`

**Modify `DependencyType.cs`:**

- [x] Add `Ldap` enum value
- [x] Update `Label()` and `FromLabel()` extension methods

**Modify `ConfigParser.cs`:**

- [x] Add `ldap` → `Ldap`, `ldaps` → `Ldap` to scheme mapping
- [x] Add default ports: `ldap` → 389, `ldaps` → 636

**Create `DepHealth.Core/Checks/LdapChecker.cs`:**

- [x] Implement `IHealthChecker` with `CheckAsync(Endpoint, CancellationToken)` and `Type`
- [x] Standalone mode: `LdapConnection.Connect`, optional StartTLS, check, disconnect
- [x] Pool mode: accept existing `LdapConnection`
- [x] Error classification: `CheckAuthException`, `ConnectionRefusedException`
- [x] Constructor overloads for standalone (config) and pool (connection) modes

**Create `tests/.../LdapCheckerTests.cs`:**

- [x] xUnit tests
- [x] Test type property
- [x] Test all 4 check methods (constructor validation)
- [x] Test error classification
- [x] Test validation rules

**Version bump:**

- [x] `sdk-csharp/Directory.Build.props` → `0.8.0`

**Validation:**

- [x] `dotnet build` passes (in Docker: `harbor.kryukov.lan/mcr/dotnet/sdk:8.0`)
- [x] `dotnet test` passes
- [x] No warnings

**Status:** done

---

### Phase 6: Documentation, changelog, version bump Go SDK

Documentation for all SDKs, Go version bump, TODO update.

**Go SDK version bump:**

- [x] `sdk-go/dephealth/version.go` → `0.8.0`

**Go SDK documentation:**

- [x] Update `sdk-go/README.md` — add LDAP checker section
- [x] Update `sdk-go/docs/api-reference.md` — add LDAP types, options, checker API (EN)
- [x] Update `sdk-go/docs/api-reference.ru.md` — same (RU)

**Java SDK documentation:**

- [x] Create `sdk-java/README.md` — add LDAP checker section

**Python SDK documentation:**

- [x] Update `sdk-python/README.md` — add LDAP checker section

**C# SDK documentation:**

- [x] Update `sdk-csharp/README.md` — add LDAP checker section

**Migration guides:**

- [x] Create `docs/migration/v070-to-v080.md` (EN) — for Go SDK (0.7.0 → 0.8.0)
- [x] Create `docs/migration/v070-to-v080.ru.md` (RU)
- [x] Create `docs/migration/v060-to-v080.md` (EN) — for Java/Python/C# (0.6.0 → 0.8.0)
- [x] Create `docs/migration/v060-to-v080.ru.md` (RU)

**Changelogs:**

- [x] Update `CHANGELOG.md` — add `[0.8.0]` section for all SDKs

**TODO:**

- [x] Update `TODO.md` — mark LDAP checker as done for all SDKs

**Validation:**

- [x] `markdownlint` passes on all new/modified `.md` files
- [x] All builds pass: Go, Java, Python, C#

**Status:** done

---

### Phase 7: Conformance testing — LDAP integration

Add LDAP to existing conformance framework: infrastructure (389ds), test services (all 4 SDK),
scenarios, runner, Helm chart. Uses the same approach as existing checkers (Postgres, Redis, etc.).

#### 7.1. Infrastructure — 389ds LDAP server

**Create `conformance/stubs/ldap-stub/` directory:**

- [x] Create `conformance/stubs/ldap-stub/init.ldif` — LDIF for test data:
  - Base suffix: `dc=test,dc=local` (auto-created by 389ds)
  - OU: `ou=People,dc=test,dc=local`
  - Test user: `uid=testuser,ou=People,dc=test,dc=local` (password: `testpassword`)
- [x] Create `conformance/stubs/ldap-stub/Dockerfile` (optional, if custom image needed)
  - Image: `389ds/dirsrv:3.1` (via `${REGISTRY}/389ds/dirsrv:3.1`)
  - Ports: 3389 (LDAP), 3636 (LDAPS)
  - Base DN: `dc=test,dc=local`
  - Admin: `cn=Directory Manager` / `password`
  - Copy `init.ldif` for auto-import on startup
  - **Note:** No custom Dockerfile needed — using stock 389ds image with init.ldif mount

**Add 389ds to Helm subchart (`deploy/helm/dephealth-infra/`):**

- [x] Add LDAP section to `values.yaml`:
  - `ldap.enabled: true`
  - `ldap.image: 389ds/dirsrv`
  - `ldap.tag: "3.1"`
  - `ldap.port: 3389`
  - `ldap.securePort: 3636`
  - `ldap.suffix: dc=test,dc=local`
  - `ldap.adminPassword: password`
- [x] Create `templates/ldap.yaml` — Deployment + Service:
  - Deployment: single replica, ports 3389/3636
  - ConfigMap with `init.ldif` (mounted for auto-import)
  - Service: `ldap.<ns>.svc` exposing ports 3389, 3636
  - Readiness probe: TCP 3389
  - Environment: `DS_SUFFIX_NAME`, `DS_DM_PASSWORD`
- [x] Update `values-homelab.yaml` — no changes needed (uses global.imageRegistry)

**Add 389ds to root `docker-compose.yml`:**

- [x] Add `ldap` service (profile: `full`):
  - Image: `${IMAGE_REGISTRY:-docker.io}/389ds/dirsrv:3.1`
  - Ports: `3389:3389`, `3636:3636`
  - Environment: `DS_SUFFIX_NAME=dc=test,dc=local`, `DS_DM_PASSWORD=password`
  - Volume: `./conformance/stubs/ldap-stub/init.ldif:/data/init.ldif`
  - Healthcheck: TCP check on port 3389

#### 7.2. Test services — add LDAP dependencies

Add LDAP endpoints to all 4 conformance test services. Each service gets 4 LDAP dependencies
covering different check methods and error scenarios:

| Dependency name | Check method | Credentials | Expected state |
|----------------|-------------|-------------|----------------|
| `ldap-rootdse` | `root_dse` | — | healthy |
| `ldap-bind` | `simple_bind` | `cn=Directory Manager` / `password` | healthy |
| `ldap-search` | `search` | `cn=Directory Manager` / `password` | healthy |
| `ldap-invalid-auth` | `simple_bind` | `cn=Directory Manager` / `wrongpassword` | auth_error |

**Modify `conformance/test-service/main.go` (Go):**

- [x] Add `LDAP_HOST` and `LDAP_PORT` to `Config` struct and `loadConfig()`
- [x] Add import for `_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck"`
  - Already imported via `_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"` (compat.go)
- [x] Add 4 LDAP dependencies to `initDepHealth()`:
  - `dephealth.LDAP("ldap-rootdse", ...)` — RootDSE, no auth
  - `dephealth.LDAP("ldap-bind", ...)` — simple bind, valid credentials
  - `dephealth.LDAP("ldap-search", ...)` — search `ou=People,dc=test,dc=local`
  - `dephealth.LDAP("ldap-invalid-auth", ...)` — simple bind, wrong password
- [x] Update dependency count comment (12 → 16)

**Modify `conformance/test-service-python/main.py` (Python):**

- [x] Add `LDAP_HOST`, `LDAP_PORT` to environment config
- [x] Add 4 LDAP dependencies using `ldap_check`:
  - Same 4 endpoints as Go (rootdse, bind, search, invalid-auth)
- [x] Update dependency count

**Modify `conformance/test-service-java/` (Java):**

- [x] Add LDAP dependencies to `application.yml`
- [x] Add `unboundid-ldapsdk` — inherited from dephealth-spring-boot-starter
- [x] Add 4 LDAP dependencies with same configuration
- [x] Update dependency count
- [x] Update dephealth-spring-boot-starter version 0.5.0 → 0.8.0

**Modify `conformance/test-service-csharp/Program.cs` (C#):**

- [x] Add `LDAP_HOST`, `LDAP_PORT` to environment config
- [x] Add 4 LDAP dependencies using `AddLdap` (new builder method)
- [x] Update dependency count

**Additional SDK changes (7.0):**

- [x] Add `AddLdap()` convenience method to C# `DepHealthMonitor.Builder`
- [x] Add LDAP properties to Java Spring Boot `DepHealthProperties`
- [x] Add LDAP mapping to Java `DepHealthAutoConfiguration`

**Rebuild Docker images for all 4 test services.**

#### 7.3. Helm chart — wire LDAP env vars to test services

**Modify `deploy/helm/dephealth-conformance/templates/conformance-services.yml`:**

- [x] Add `LDAP_HOST` and `LDAP_PORT` environment variables to all 4 service deployments:
  - `LDAP_HOST: ldap.{{ $ns }}.svc`
  - `LDAP_PORT: "3389"`
- [x] Credentials hard-coded in test services (conformance values are fixed)

**Update `values.yaml` and `values-homelab.yaml`:**

- [x] Add LDAP credentials to values (for reference):
  - `ldapAdminDN: cn=Directory Manager`
  - `ldapAdminPassword: password`
  - `ldapBaseDN: dc=test,dc=local`
- [x] Add `ldap.enabled: true` to conformance subchart config

#### 7.4. Conformance runner — update `verify.py`

**Modify `conformance/runner/verify.py`:**

- [x] Add `"ldap"` to `VALID_TYPES` set
- [x] Add `"ldap"` entry to `VALID_DETAILS_BY_TYPE`:
  ```python
  "ldap": {"ok", "timeout", "connection_refused", "dns_error",
           "auth_error", "tls_error", "unhealthy", "error"},
  ```
- [x] Verify `DETAIL_TO_STATUS` mapping already covers all LDAP details (it does — no changes needed)

#### 7.5. Conformance scenarios — LDAP

**Create `conformance/scenarios/ldap-basic.yml`:**

- [x] Test all 4 LDAP dependencies in healthy/expected state:
  - `ldap-rootdse`: health=1, status=ok, detail=ok
  - `ldap-bind`: health=1, status=ok, detail=ok
  - `ldap-search`: health=1, status=ok, detail=ok
  - `ldap-invalid-auth`: health=0, status=auth_error, detail=auth_error
- [x] Validate type=ldap label on all LDAP endpoints
- [x] Validate host and port labels

**Create `conformance/scenarios/ldap-failure.yml`:**

- [x] Pre-action: scale LDAP deployment to 0
- [x] Wait 30s for check cycle
- [x] Check all LDAP dependencies: health=0, status=connection_error, detail=connection_refused
- [x] Post-action: scale LDAP deployment back to 1
- [x] Wait for readiness

**Create `conformance/scenarios/ldap-recovery.yml`:**

- [x] Pre-action: scale LDAP to 0, wait, scale back to 1, wait
- [x] Check all healthy LDAP dependencies recover to health=1

**Update existing scenarios:**

- [x] Modify `conformance/scenarios/basic-health.yml`:
  - Add 4 LDAP dependencies to `expected_dependencies` (16 total)
  - Add LDAP endpoints to `expected_status` and `expected_detail`
- [x] Modify `conformance/scenarios/health-details.yml`:
  - Add 4 LDAP endpoints to `health_details_expected`
- [x] Modify `conformance/scenarios/labels.yml`:
  - No changes needed (generic label checks apply to all types)

#### 7.6. Documentation

**Update `conformance/README.md`:**

- [x] Add LDAP to the list of tested dependency types
- [x] Add LDAP dependencies to the dependency table
- [x] Document new LDAP scenarios (`ldap-basic`, `ldap-failure`, `ldap-recovery`)
- [x] Add 389ds to infrastructure components list

#### 7.7. Validation

- [x] `markdownlint` passes on all modified `.md` files
- [x] All 4 test service Docker images build successfully
- [x] Helm chart deploys without errors (including 389ds)
- [x] 389ds starts, accepts connections, test data imported
- [x] `./run.sh --lang all` — all existing scenarios still pass (regression)
- [x] `./run.sh --lang all --scenario ldap-basic` — passes for all 4 SDKs
- [x] `./run.sh --lang all --scenario ldap-failure` — passes for all 4 SDKs
- [x] `./run.sh --lang all --scenario ldap-recovery` — passes for all 4 SDKs
- [x] Cross-verify: `cross_verify.py` shows identical LDAP metrics across all 4 SDKs

**Status:** validation complete

---

### Phase 8: Merge, tag, release

**Pre-merge checklist:**

- [x] All unit tests pass (Go, Java, Python, C#)
- [x] All conformance scenarios pass (`./run.sh --lang all`)
- [x] All linters pass
- [x] Spec updated (EN + RU)
- [x] Docs complete (EN + RU, all SDKs)
- [x] CHANGELOG updated
- [x] TODO updated
- [x] Backward compatibility verified (existing API unchanged in all SDKs)

**Actions:**

- [x] Merge `feature/ldap-checker` to `master`
- [ ] Tag: `sdk-go/v0.8.0`
- [ ] Tag: `sdk-java/v0.8.0`
- [ ] Tag: `sdk-python/v0.8.0`
- [ ] Tag: `sdk-csharp/v0.8.0`
- [ ] GitHub Releases (4 releases, one per SDK)
- [ ] Move this plan to `plans/archive/`

**Status:** todo
