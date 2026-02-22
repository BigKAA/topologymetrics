# Plan: Go SDK — Full Documentation (sdk-go/docs/)

## Goal

Create comprehensive documentation for the Go SDK inside `sdk-go/docs/`.
10 EN + 10 RU files covering the complete public API, all 8 checkers,
connection pools, custom checkers, authentication, configuration,
Prometheus metrics, selective imports, and troubleshooting.

The documentation complements (not replaces) existing `docs/quickstart/go.md`
and `docs/migration/` guides, with cross-references between them.

## Current state

- `sdk-go/README.md` — brief overview (~138 lines), outdated code examples (v0.5 API)
- `docs/quickstart/go.md` + `.ru.md` — quickstart with examples (up to date for v0.6.0)
- `docs/migration/go.md`, `v050-to-v060.md` + `.ru.md` — migration guides
- `docs/code-style/go.md` + `.ru.md` — code style conventions
- No API reference, no detailed checker guides, no troubleshooting

## Target structure

```
sdk-go/docs/
├── getting-started.md          (+.ru.md)
├── api-reference.md            (+.ru.md)
├── checkers.md                 (+.ru.md)
├── connection-pools.md         (+.ru.md)
├── custom-checkers.md          (+.ru.md)
├── authentication.md           (+.ru.md)
├── configuration.md            (+.ru.md)
├── metrics.md                  (+.ru.md)
├── selective-imports.md        (+.ru.md)
└── troubleshooting.md          (+.ru.md)
```

**Total: 20 new files + README.md update**

## Documentation conventions

- EN file: base name (e.g., `checkers.md`)
- RU file: `.ru.md` suffix (e.g., `checkers.ru.md`)
- EN header: `*[Русская версия](file.ru.md)*`
- RU header: `*[English version](file.md)*`
- Code examples: runnable Go snippets with proper imports
- Cross-references: relative links to `docs/quickstart/`, `docs/migration/`, `spec/`
- Tables for options, defaults, comparison
- No emojis, technical professional tone
- Markdownlint: MD013 disabled, MD024 siblings_only

## Phases

---

### Phase 1: Getting Started + Selective Imports (EN + RU)

Create the entry-point documentation and the selective imports guide.

**Files to create:**

- [x] `sdk-go/docs/getting-started.md`
- [x] `sdk-go/docs/getting-started.ru.md`
- [x] `sdk-go/docs/selective-imports.md`
- [x] `sdk-go/docs/selective-imports.ru.md`

**getting-started.md content:**

- Installation (`go get`)
- Module path: `github.com/BigKAA/topologymetrics/sdk-go/dephealth`
- Minimal example (HTTP checker, Prometheus export, graceful shutdown)
- Checker registration (blank import pattern)
- Link to selective imports for optimization
- Link to `docs/quickstart/go.md` for extended examples
- Link to API reference for full details

**selective-imports.md content:**

- Problem: importing `checks` pulls all 8 checker dependencies
- Solution: v0.6.0 split packages
- Full import (`checks`) vs selective (`checks/httpcheck`, etc.)
- Available sub-packages table (8 packages with import paths)
- Binary size impact explanation
- Backward compatibility: `checks` package still works
- Deprecated type aliases in `checks/compat.go`
- Link to migration guide `docs/migration/v050-to-v060.md`

**Validation:**

- [x] `markdownlint` on all 4 files — 0 issues
- [x] All code examples compile (mentally verified against actual API)
- [x] Cross-references point to existing files

**Status:** done

---

### Phase 2: Checkers Guide (EN + RU)

Detailed guide for all 8 built-in checkers with examples.

**Files to create:**

- [x] `sdk-go/docs/checkers.md`
- [x] `sdk-go/docs/checkers.ru.md`

**Content per checker (8 sections):**

For each of HTTP, gRPC, TCP, Postgres, MySQL, Redis, AMQP, Kafka:

1. **Overview** — what it checks, protocol used
2. **Registration** — `dephealth.HTTP(name, opts...)` syntax
3. **Options table** — all checker-specific `DependencyOption` functions
4. **Full example** — runnable code with proper imports
5. **Error classification** — what errors map to which `StatusCategory`
6. **Direct checker usage** — `httpcheck.New()` with `httpcheck.Option`

**Checker details:**

| Checker | Key points to document |
| --- | --- |
| HTTP | health path, TLS, skip verify, redirects, auth (bearer/basic/headers), status codes → status |
| gRPC | service name, TLS, metadata, auth, gRPC health protocol, status codes |
| TCP | simplest checker, just connection test |
| Postgres | `SELECT 1` default, custom query, DSN, pool mode via `WithDB` |
| MySQL | `SELECT 1` default, custom query, DSN, `URLToDSN()` helper, pool mode |
| Redis | `PING`, password, DB number, pool mode via `WithClient` |
| AMQP | connection check, custom URL, default guest:guest |
| Kafka | metadata request, multi-host support |

**Validation:**

- [x] `markdownlint` — 0 issues
- [x] All 8 checkers documented with examples

**Status:** done

---

### Phase 3: Authentication + Connection Pools (EN + RU)

**Files to create:**

- [x] `sdk-go/docs/authentication.md`
- [x] `sdk-go/docs/authentication.ru.md`
- [x] `sdk-go/docs/connection-pools.md`
- [x] `sdk-go/docs/connection-pools.ru.md`

**authentication.md content:**

- Overview: which checkers support auth
- HTTP authentication:
  - Bearer token (`WithHTTPBearerToken`)
  - Basic auth (`WithHTTPBasicAuth`)
  - Custom headers (`WithHTTPHeaders`)
  - Mutual exclusion rule (only one auth method per dependency)
- gRPC authentication:
  - Bearer token (`WithGRPCBearerToken`)
  - Basic auth (`WithGRPCBasicAuth`)
  - Custom metadata (`WithGRPCMetadata`)
  - Mutual exclusion rule
- Database credentials:
  - PostgreSQL: credentials in URL
  - MySQL: credentials in URL or DSN
  - Redis: `WithRedisPassword`
  - AMQP: credentials in URL (`amqp://user:pass@host/`)
- Auth error classification:
  - HTTP 401/403 → `StatusAuthError`
  - gRPC UNAUTHENTICATED/PERMISSION_DENIED → `StatusAuthError`
  - PostgreSQL SQLSTATE 28000/28P01 → `StatusAuthError`
  - MySQL error 1045 → `StatusAuthError`
  - Redis NOAUTH/WRONGPASS → `StatusAuthError`
  - AMQP 403 ACCESS_REFUSED → `StatusAuthError`
- Security: credentials from env vars, not hardcoded
- Full example with all auth types

**connection-pools.md content:**

- Standalone vs pool mode comparison
- Why pool mode is preferred
- `contrib/sqldb` package:
  - `sqldb.FromDB(name, db, opts...)` — PostgreSQL via `*sql.DB`
  - `sqldb.FromMySQLDB(name, db, opts...)` — MySQL via `*sql.DB`
  - Full example with `pgx/stdlib` driver
  - Full example with `go-sql-driver/mysql`
- `contrib/redispool` package:
  - `redispool.FromClient(name, client, opts...)` — Redis via `*redis.Client`
  - Auto-extraction of host:port from client options
  - Full example with `go-redis/v9`
- Direct checker pool integration:
  - `pgcheck.New(pgcheck.WithDB(db))` with `AddDependency`
  - `mysqlcheck.New(mysqlcheck.WithDB(db))` with `AddDependency`
  - `redischeck.New(redischeck.WithClient(client))` with `AddDependency`
- When to use contrib vs direct checker

**Validation:**

- [x] `markdownlint` — 0 issues
- [x] All code examples match actual API signatures

**Status:** done

---

### Phase 4: Configuration + Custom Checkers (EN + RU)

**Files to create:**

- [x] `sdk-go/docs/configuration.md`
- [x] `sdk-go/docs/configuration.ru.md`
- [x] `sdk-go/docs/custom-checkers.md`
- [x] `sdk-go/docs/custom-checkers.ru.md`

**configuration.md content:**

- `New(name, group, opts...)` — name and group parameters
  - Validation rules: `[a-z][a-z0-9-]*`, 1-63 chars
  - `DEPHEALTH_NAME` and `DEPHEALTH_GROUP` env vars
  - Priority: API > env var
- Global options table:
  - `WithCheckInterval(d)` — min 1s, max 10m, default 15s
  - `WithTimeout(d)` — min 100ms, max 30s, default 5s
  - `WithRegisterer(r)` — custom Prometheus registerer
  - `WithLogger(l)` — `*slog.Logger`
- Common dependency options table:
  - `FromURL(url)` / `FromParams(host, port)`
  - `Critical(bool)` — mandatory
  - `WithLabel(key, value)`
  - `CheckInterval(d)` / `Timeout(d)` — per-dependency overrides
- Checker-specific options reference table (all `With*` options grouped by checker)
- Environment variables:
  - `DEPHEALTH_NAME`, `DEPHEALTH_GROUP`
  - `DEPHEALTH_<DEP>_CRITICAL`
  - `DEPHEALTH_<DEP>_LABEL_<KEY>`
  - Name transformation rules (uppercase, hyphens → underscores)
- Validation rules and error messages:
  - Required parameters (name, group, critical)
  - Invalid label names (reserved labels)
  - Range validation for intervals and timeouts
- Default values table (all defaults in one place)

**custom-checkers.md content:**

- `HealthChecker` interface: `Check(ctx, endpoint) error` + `Type() string`
- Creating a custom checker (full example: Elasticsearch health check)
- Registering via `AddDependency(name, depType, checker, opts...)`
- Error classification:
  - `ClassifiedError` interface
  - `ClassifiedCheckError` struct
  - Built-in `StatusCategory` constants
  - Returning classified errors from `Check()`
- Registering checker factory for URL-based creation:
  - `CheckerFactory` type
  - `RegisterCheckerFactory(depType, factory)`
  - Custom `DependencyType` constant
- Full example: custom checker with status classification
- Testing custom checkers

**Validation:**

- [x] `markdownlint` — 0 issues
- [x] Custom checker example is compilable

**Status:** done

---

### Phase 5: Prometheus Metrics (EN + RU)

**Files to create:**

- [x] `sdk-go/docs/metrics.md`
- [x] `sdk-go/docs/metrics.ru.md`

**Content:**

- Overview: 4 Prometheus metrics exported by dephealth
- `app_dependency_health` (Gauge):
  - Labels: name, group, dependency, type, host, port, critical, [custom]
  - Values: 1 = healthy, 0 = unhealthy
  - Example metric output
- `app_dependency_latency_seconds` (Histogram):
  - Same labels
  - Buckets: 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0
  - Example metric output
- `app_dependency_status` (Gauge, enum pattern):
  - Additional label: `status` (one of 8 categories)
  - 8 series per endpoint, exactly one = 1
  - Status categories: ok, timeout, connection_error, dns_error, auth_error, tls_error, unhealthy, error
  - Example metric output
- `app_dependency_status_detail` (Gauge, info pattern):
  - Additional label: `detail`
  - 1 series per unique detail value
  - Possible detail values per checker
  - Example metric output
- Label order and naming:
  - Fixed labels: name, group, dependency, type, host, port, critical
  - Custom labels appended in alphabetical order
  - `critical` values: "yes" / "no"
- Custom Prometheus registerer:
  - `WithRegisterer(r)` option
  - Use case: separate registry, push gateway
- PromQL examples:
  - Health status query
  - Latency percentiles
  - Status category filtering
  - Unhealthy dependencies alert
  - Auth error rate
- Integration with Grafana dashboards
- Link to `docs/grafana-dashboards.md` and `docs/alerting/`
- Link to `spec/metric-contract.md` for full specification

**Validation:**

- [x] `markdownlint` — 0 issues
- [x] PromQL examples are valid
- [x] Metric examples match actual SDK output

**Status:** done

---

### Phase 6: API Reference (EN + RU)

The largest document — full reference of all public symbols.

**Files to create:**

- [x] `sdk-go/docs/api-reference.md`
- [x] `sdk-go/docs/api-reference.ru.md`

**Structure (organized by package):**

**Package `dephealth`:**

- Constants section:
  - `Version`
  - `DefaultCheckInterval`, `DefaultTimeout`, etc.
  - Min/Max bounds
  - `DependencyType` constants (TypeHTTP, TypeGRPC, etc.)
  - `StatusCategory` constants (StatusOK, StatusTimeout, etc.)

- Types section:
  - `DepHealth` — main struct, methods: `Start`, `Stop`, `Health`, `HealthDetails`
  - `Endpoint` — struct fields
  - `Dependency` — struct fields, `Validate()` method
  - `EndpointStatus` — struct fields, `LatencyMillis()`, JSON marshaling
  - `CheckConfig` — struct fields, `DefaultCheckConfig()`, `Validate()`
  - `CheckResult` — struct fields
  - `DependencyConfig` — all fields (HTTP, gRPC, DB options)
  - `ParsedConnection` — struct fields

- Interfaces section:
  - `HealthChecker` — `Check(ctx, endpoint) error`, `Type() string`
  - `ClassifiedError` — `Error()`, `StatusCategory()`, `StatusDetail()`

- Functions section:
  - `New(name, group, opts...) (*DepHealth, error)`
  - Dependency factories: `HTTP()`, `GRPC()`, `TCP()`, `Postgres()`, `MySQL()`, `Redis()`, `AMQP()`, `Kafka()`
  - `AddDependency(name, depType, checker, opts...) Option`
  - URL parsers: `ParseURL()`, `ParseConnectionString()`, `ParseJDBC()`, `ParseParams()`
  - Validators: `ValidateName()`, `ValidateLabelName()`, `ValidateLabels()`
  - `RegisterCheckerFactory(depType, factory)`
  - `BoolToYesNo(v) string`

- Options section (grouped):
  - Global `Option`: `WithCheckInterval`, `WithTimeout`, `WithRegisterer`, `WithLogger`
  - Common `DependencyOption`: `FromURL`, `FromParams`, `Critical`, `WithLabel`, `CheckInterval`, `Timeout`
  - HTTP `DependencyOption`: `WithHTTPHealthPath`, `WithHTTPTLS`, `WithHTTPTLSSkipVerify`, `WithHTTPHeaders`, `WithHTTPBearerToken`, `WithHTTPBasicAuth`
  - gRPC `DependencyOption`: `WithGRPCServiceName`, `WithGRPCTLS`, `WithGRPCTLSSkipVerify`, `WithGRPCMetadata`, `WithGRPCBearerToken`, `WithGRPCBasicAuth`
  - DB `DependencyOption`: `WithPostgresQuery`, `WithMySQLQuery`, `WithRedisPassword`, `WithRedisDB`, `WithAMQPURL`

- Errors section:
  - `ErrTimeout`, `ErrConnectionRefused`, `ErrUnhealthy`
  - `ErrAlreadyStarted`, `ErrNotStarted`
  - `ClassifiedCheckError` struct

**Package `checks`:**

- Overview: backward compatibility package
- Type aliases table (Deprecated)
- Constructor wrappers table (Deprecated)
- Option wrappers table (Deprecated)

**Sub-packages (`checks/httpcheck`, etc.):**

- For each of 8 packages:
  - `Checker` type
  - `New(opts...) *Checker`
  - `NewFromConfig(dc) HealthChecker`
  - `Option` type and all `With*` functions
  - `Check(ctx, endpoint) error` behavior
  - `Type() string` return value

**Contrib packages:**

- `contrib/sqldb`:
  - `FromDB(name, db, opts...) Option`
  - `FromMySQLDB(name, db, opts...) Option`
- `contrib/redispool`:
  - `FromClient(name, client, opts...) Option`

**Format:** tables for options/constants, code signatures in code blocks,
brief descriptions. No lengthy prose — this is a reference document.

**Validation:**

- [x] `markdownlint` — 0 issues
- [x] All symbols verified against actual source code
- [x] No public symbols missed

**Status:** done

---

### Phase 7: Troubleshooting + README update

**Files to create:**

- [x] `sdk-go/docs/troubleshooting.md`
- [x] `sdk-go/docs/troubleshooting.ru.md`

**troubleshooting.md content:**

- **Empty metrics / no metrics exported**
  - Forgot blank import (`checks` or sub-packages)
  - `Start()` not called
  - Wrong Prometheus handler path
- **High gRPC latency (100-500ms)**
  - DNS resolver issue in Kubernetes (ndots:5)
  - Solution: use FQDN with trailing dot, or passthrough resolver
  - Link to `docs/specification.md` DNS section
- **Connection refused errors**
  - Service not running, wrong host/port
  - Network policies in Kubernetes
  - Firewall rules
- **Timeout errors**
  - Default timeout too low for slow dependencies
  - Increase via `Timeout()` per-dependency option
  - Check network latency
- **Auth errors unexpected**
  - Credentials not passed or incorrect
  - Token expired
  - Wrong auth method (bearer vs basic)
- **"no checker factory registered" panic**
  - Missing blank import of checker package
  - Selective import doesn't include needed checker
- **Duplicate metric registration**
  - Two `DepHealth` instances with same registerer
  - Solution: use `WithRegisterer()` with separate registry
- **Custom labels not appearing**
  - Labels must be registered in the same order across all endpoints
  - Invalid label names (must match `[a-z_][a-z0-9_]*`)
  - Reserved label names (name, group, dependency, type, host, port, critical)
- **Health() returns empty map**
  - Checked before first check completed (initial delay)
  - Use `HealthDetails()` to see `unknown` status
- **Debug logging**
  - `WithLogger(slog.New(...))` for debug output
  - What log messages to expect

**Update sdk-go/README.md:**

- [x] Add "Documentation" section with links to all docs/ files
- [x] Fix outdated code example in "Quick Start" (v0.5 API → v0.6 API)
- [x] Fix "Connection Pool Integration" example (outdated `pgxcheck` reference)
- [x] Ensure all cross-references are correct

**Validation:**

- [x] `markdownlint` on all new/modified files — 0 issues
- [x] All links in README.md resolve to existing files

**Status:** done

---

### Phase 8: Final review + cross-references

**Review all 20 docs:**

- [x] Consistent terminology across EN and RU versions
- [x] All code examples use v0.6.0 API (New with name+group, split checkers)
- [x] Cross-references between docs are bidirectional
- [x] No broken links
- [x] Markdownlint passes on all files

**Update existing docs with cross-references:**

- [x] `docs/quickstart/go.md` — add "See also" links to `sdk-go/docs/`
- [x] `docs/quickstart/go.ru.md` — same
- [x] `CHANGELOG.md` — no changes needed (docs don't bump version)

**Commit:**

- Ask user about commit strategy (one commit or per-phase)
- Suggested message: `docs(go): add comprehensive SDK documentation`

**Status:** done
