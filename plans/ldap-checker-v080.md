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
| C# | Novell LDAP | `Novell.Directory.Ldap.NETStandard` |

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

- [ ] Add `github.com/go-ldap/ldap/v3` to `sdk-go/go.mod`

**Modify `sdk-go/dephealth/options.go`:**

- [ ] Add `TypeLDAP DependencyType = "ldap"` constant
- [ ] Add `LDAP()` factory function
- [ ] Add LDAP-specific options: `WithLDAPCheckMethod`, `WithLDAPBindDN`, `WithLDAPBindPassword`, `WithLDAPBaseDN`, `WithLDAPSearchFilter`, `WithLDAPSearchScope`, `WithLDAPStartTLS`
- [ ] Add LDAP fields to `DependencyConfig`

**Modify `sdk-go/dephealth/parser.go`:**

- [ ] Add `ldap` → `TypeLDAP`, `ldaps` → `TypeLDAP` to scheme mapping
- [ ] Add default ports: `ldap` → 389, `ldaps` → 636

**Create `sdk-go/dephealth/checks/ldapcheck/ldapcheck.go`:**

- [ ] Implement `LdapChecker` struct with `Check(ctx, Endpoint) error` and `Type() string`
- [ ] Factory `NewFromConfig(cfg dephealth.DependencyConfig) dephealth.HealthChecker`
- [ ] `init()` with `RegisterCheckerFactory(TypeLDAP, NewFromConfig)`
- [ ] Standalone mode: connect, optional StartTLS, execute check method, close
- [ ] Pool mode: accept existing `*ldap.Conn`
- [ ] Error classification: auth errors (LDAP result code 49, 50), TLS errors, connection errors
- [ ] Options: `WithConn`, `WithCheckMethod`, `WithBindDN`, `WithBindPassword`, etc.

**Create `sdk-go/dephealth/checks/ldapcheck/ldapcheck_test.go`:**

- [ ] Test RootDSE check (default) — mock or real LDAP
- [ ] Test anonymous bind
- [ ] Test simple bind with valid credentials
- [ ] Test simple bind with invalid credentials (auth_error)
- [ ] Test search method with baseDN
- [ ] Test StartTLS
- [ ] Test connection refused
- [ ] Test `NewFromConfig` with URL parsing
- [ ] Test validation: `simple_bind` without credentials, `search` without baseDN, `startTLS` with `ldaps://`

**Validation:**

- [ ] `make build` passes (in sdk-go/)
- [ ] `make test` passes
- [ ] `make lint` passes

**Status:** todo

---

### Phase 3: Java SDK — LDAP checker implementation

Implement LDAP checker for Java SDK.

**Add dependency:**

- [ ] Add `com.unboundid:unboundid-ldapsdk` to `sdk-java/dephealth-core/pom.xml`

**Modify `DependencyType.java`:**

- [ ] Add `LDAP("ldap")` enum value

**Modify `ConfigParser.java`:**

- [ ] Add `ldap` → `LDAP`, `ldaps` → `LDAP` to scheme mapping
- [ ] Add default ports: `ldap` → 389, `ldaps` → 636

**Create `LdapHealthChecker.java`:**

- [ ] Implement `HealthChecker` interface
- [ ] Builder pattern for configuration (checkMethod, bindDN, bindPassword, baseDN, etc.)
- [ ] Standalone mode: `LDAPConnection`, optional StartTLS, check method, close
- [ ] Pool mode: accept existing `LDAPConnection`
- [ ] Error classification: `CheckAuthException`, `CheckConnectionException`

**Create `LdapHealthCheckerTest.java`:**

- [ ] Unit tests with Mockito (mock LDAP connection)
- [ ] Test all 4 check methods
- [ ] Test error classification
- [ ] Test validation rules

**Version bump:**

- [ ] `sdk-java/pom.xml` → `0.8.0`
- [ ] All child `pom.xml` → `0.8.0`

**Validation:**

- [ ] `mvn compile` passes
- [ ] `mvn test` passes
- [ ] `mvn checkstyle:check` passes (if configured)

**Status:** todo

---

### Phase 4: Python SDK — LDAP checker implementation

Implement LDAP checker for Python SDK.

**Add dependency:**

- [ ] Add `ldap3` to `sdk-python/pyproject.toml` dependencies

**Modify `sdk-python/dephealth/dependency.py`:**

- [ ] Add `LDAP = "ldap"` to `DependencyType`

**Modify `sdk-python/dephealth/parser.py`:**

- [ ] Add `ldap` → `LDAP`, `ldaps` → `LDAP` to scheme mapping
- [ ] Add default ports: `ldap` → 389, `ldaps` → 636

**Create `sdk-python/dephealth/checks/ldap.py`:**

- [ ] `LdapChecker` class with `async check(endpoint)` and `checker_type() -> str`
- [ ] Constructor: `timeout`, `check_method`, `bind_dn`, `bind_password`, `base_dn`, `search_filter`, `search_scope`, `start_tls`, `tls_skip_verify`, `client`
- [ ] Standalone: create `ldap3.Connection` in executor (ldap3 is synchronous)
- [ ] Pool mode: accept existing connection
- [ ] Error classification: `CheckAuthError`, `CheckConnectionRefusedError`, `CheckTlsError`

**Modify `sdk-python/dephealth/checks/__init__.py`:**

- [ ] Export `LdapChecker`

**Create `sdk-python/tests/test_ldap.py`:**

- [ ] Unit tests for all 4 check methods
- [ ] Test error classification
- [ ] Test validation rules

**Version bump:**

- [ ] `sdk-python/pyproject.toml` version → `0.8.0`

**Validation:**

- [ ] `pytest` passes
- [ ] `ruff check` passes
- [ ] `mypy` passes (if configured)

**Status:** todo

---

### Phase 5: C# SDK — LDAP checker implementation

Implement LDAP checker for C# SDK.

**Add dependency:**

- [ ] Add `Novell.Directory.Ldap.NETStandard` to `DepHealth.Core.csproj`

**Modify `DependencyType.cs`:**

- [ ] Add `Ldap` enum value
- [ ] Update `Label()` and `FromLabel()` extension methods

**Modify `ConfigParser.cs`:**

- [ ] Add `ldap` → `Ldap`, `ldaps` → `Ldap` to scheme mapping
- [ ] Add default ports: `ldap` → 389, `ldaps` → 636

**Create `DepHealth.Core/Checks/LdapChecker.cs`:**

- [ ] Implement `IHealthChecker` with `CheckAsync(Endpoint, CancellationToken)` and `Type`
- [ ] Standalone mode: `LdapConnection.ConnectAsync`, optional StartTLS, check, disconnect
- [ ] Pool mode: accept existing `ILdapConnection`
- [ ] Error classification: `CheckAuthException`, `ConnectionRefusedException`
- [ ] Constructor overloads for standalone (config) and pool (connection) modes

**Create `tests/.../LdapCheckerTests.cs`:**

- [ ] xUnit tests
- [ ] Test type property
- [ ] Test all 4 check methods (with mocked connection)
- [ ] Test error classification
- [ ] Test validation rules

**Version bump:**

- [ ] `sdk-csharp/Directory.Build.props` → `0.8.0`

**Validation:**

- [ ] `dotnet build` passes (in Docker: `harbor.kryukov.lan/mcr/dotnet/sdk:8.0`)
- [ ] `dotnet test` passes
- [ ] No warnings

**Status:** todo

---

### Phase 6: Documentation, changelog, version bump Go SDK

Documentation for all SDKs, Go version bump, TODO update.

**Go SDK version bump:**

- [ ] `sdk-go/dephealth/version.go` → `0.8.0`

**Go SDK documentation:**

- [ ] Update `sdk-go/README.md` — add LDAP checker section
- [ ] Update `sdk-go/docs/api-reference.md` — add LDAP types, options, checker API (EN)
- [ ] Update `sdk-go/docs/api-reference.ru.md` — same (RU)

**Java SDK documentation:**

- [ ] Update `sdk-java/README.md` — add LDAP checker section

**Python SDK documentation:**

- [ ] Update `sdk-python/README.md` — add LDAP checker section

**C# SDK documentation:**

- [ ] Update `sdk-csharp/README.md` — add LDAP checker section

**Migration guides:**

- [ ] Create `docs/migration/v070-to-v080.md` (EN) — for Go SDK (0.7.0 → 0.8.0)
- [ ] Create `docs/migration/v070-to-v080.ru.md` (RU)
- [ ] Create `docs/migration/v060-to-v080.md` (EN) — for Java/Python/C# (0.6.0 → 0.8.0)
- [ ] Create `docs/migration/v060-to-v080.ru.md` (RU)

**Changelogs:**

- [ ] Update `CHANGELOG.md` — add `[0.8.0]` section for all SDKs

**TODO:**

- [ ] Update `TODO.md` — mark LDAP checker as done for all SDKs

**Validation:**

- [ ] `markdownlint` passes on all new/modified `.md` files
- [ ] All builds pass: Go, Java, Python, C#

**Status:** todo

---

### Phase 7: Integration tests with 389ds

Integration tests using real 389ds LDAP server in Docker.

**Setup:**

- [ ] Create `tests/ldap/docker-compose.yml` with `389ds/dirsrv:3.1` container
- [ ] Create `tests/ldap/init.ldif` — initialize test entries (OUs, users)
- [ ] Container config: base DN `dc=test,dc=local`, admin `cn=Directory Manager`/`password`, ports 3389/3636

**Integration tests:**

- [ ] Test RootDSE query — successful check (all SDKs)
- [ ] Test anonymous bind — successful check (all SDKs)
- [ ] Test simple bind — valid credentials (all SDKs)
- [ ] Test simple bind — invalid credentials → auth_error (all SDKs)
- [ ] Test search with baseDN (all SDKs)
- [ ] Test LDAPS connection (all SDKs)
- [ ] Test StartTLS connection (all SDKs)
- [ ] Test connection refused (wrong port) (all SDKs)
- [ ] Test TLS error (invalid cert with `tlsSkipVerify=false`) (all SDKs)

**Validation:**

- [ ] All integration tests pass against 389ds container
- [ ] Tests are documented in README

**Status:** todo

---

### Phase 8: Merge, tag, release

**Pre-merge checklist:**

- [ ] All unit tests pass (Go, Java, Python, C#)
- [ ] All integration tests pass
- [ ] All linters pass
- [ ] Spec updated (EN + RU)
- [ ] Docs complete (EN + RU, all SDKs)
- [ ] CHANGELOG updated
- [ ] TODO updated
- [ ] Backward compatibility verified (existing API unchanged in all SDKs)

**Actions:**

- [ ] Merge `feature/ldap-checker` to `master`
- [ ] Tag: `sdk-go/v0.8.0`
- [ ] Tag: `sdk-java/v0.8.0`
- [ ] Tag: `sdk-python/v0.8.0`
- [ ] Tag: `sdk-csharp/v0.8.0`
- [ ] GitHub Releases (4 releases, one per SDK)
- [ ] Move this plan to `plans/archive/`

**Status:** todo
