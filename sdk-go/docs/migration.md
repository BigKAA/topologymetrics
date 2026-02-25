*[Русская версия](migration.ru.md)*

# Migration Guide

Version-by-version migration instructions for the dephealth Go SDK.

## v0.7.0 → v0.8.0 (LDAP Checker)

v0.8.0 adds the LDAP health checker. No breaking changes.

New import for LDAP support:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck"
```

Usage:

```go
dephealth.LDAP("directory",
    dephealth.FromURL("ldap://ldap.svc:389"),
    dephealth.Critical(true),
    dephealth.WithLDAPCheckMethod("root_dse"),
)
```

Check methods: `anonymous_bind`, `simple_bind`, `root_dse` (default), `search`.

See the [cross-SDK migration guide](../../docs/migration/v070-to-v080.md) for full details.

---

## v0.6.0 → v0.7.0 (Dynamic Endpoints)

v0.7.0 adds runtime endpoint management. No breaking changes.

New methods on a running `DepHealth` instance:

```go
// Add a new endpoint after Start()
err := dh.AddEndpoint("api-backend", dephealth.TypeHTTP, true,
    dephealth.Endpoint{Host: "backend-2.svc", Port: "8080"},
    httpcheck.New(),
)

// Remove an endpoint (cancels goroutine, deletes metrics)
err = dh.RemoveEndpoint("api-backend", "backend-2.svc", "8080")

// Atomically replace an endpoint
err = dh.UpdateEndpoint("api-backend", "backend-1.svc", "8080",
    dephealth.Endpoint{Host: "backend-3.svc", Port: "8080"},
    httpcheck.New(),
)
```

See the [cross-SDK migration guide](../../docs/migration/v060-to-v070.md) for full details.

---

## v0.5.0 → v0.6.0 (Split Checkers)

v0.6.0 introduces selective checker imports to reduce binary size.

### Breaking: checker registration required

Before v0.6.0, all checkers were registered automatically. Starting from v0.6.0,
you must explicitly import checker packages.

**Import all checkers (backward-compatible):**

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

**Import only what you need:**

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)
```

Available sub-packages: `tcpcheck`, `httpcheck`, `grpccheck`, `pgcheck`,
`mysqlcheck`, `redischeck`, `amqpcheck`, `kafkacheck`.

See [Selective Imports](selective-imports.md) for details and the
[cross-SDK migration guide](../../docs/migration/v050-to-v060.md) for full details.

---

## v0.4.x → v0.5.0

### Breaking: mandatory `group` parameter

v0.5.0 adds a mandatory `group` parameter (logical grouping: team, subsystem, project).

```go
// v0.4.x
dh, err := dephealth.New("my-service",
    dephealth.Postgres("postgres-main", ...),
)

// v0.5.0
dh, err := dephealth.New("my-service", "my-team",
    dephealth.Postgres("postgres-main", ...),
)
```

Alternative: set `DEPHEALTH_GROUP` environment variable (API takes precedence).

Validation: same rules as `name` — `[a-z][a-z0-9-]*`, 1-63 chars.

See the [cross-SDK migration guide](../../docs/migration/v042-to-v050.md) for full details.

---

## v0.4.0 → v0.4.1

### New: HealthDetails() API

v0.4.1 adds the `HealthDetails()` method that returns detailed status for each
endpoint. No existing API changes — this is a purely additive feature.

```go
details := dh.HealthDetails()
// map[string]dephealth.EndpointStatus

for key, ep := range details {
    fmt.Printf("%s: healthy=%v status=%s detail=%s latency=%v\n",
        key, ep.Healthy, ep.Status, ep.Detail, ep.Latency)
}
```

`EndpointStatus` fields: `Dependency`, `Type`, `Host`, `Port`, `Healthy` (`*bool`),
`Status`, `Detail`, `Latency`, `LastCheckedAt`, `Critical`, `Labels`.

Before the first check, `Healthy` is `nil` and `Status` is `"unknown"`.

---

## v0.3.x → v0.4.0

### New Status Metrics (no code changes required)

v0.4.0 adds two new automatically exported Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason: e.g. `http_503`, `auth_error` |

**No code changes are needed** — the SDK exports these metrics automatically alongside the existing `app_dependency_health` and `app_dependency_latency_seconds`.

### Storage Impact

Each endpoint now produces 9 additional time series (8 for `app_dependency_status` + 1 for `app_dependency_status_detail`). For a service with 5 endpoints, this adds 45 series.

### New PromQL Queries

```promql
# Status category for a dependency
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Detailed failure reason
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Alert on authentication errors
app_dependency_status{status="auth_error"} == 1
```

For the full list of status values, see [Metric Contract](../../spec/metric-contract.md).

---

## v0.2 → v0.3.0

### Breaking: new module path

In v0.3.0, the module path has changed from `github.com/BigKAA/topologymetrics`
to `github.com/BigKAA/topologymetrics/sdk-go`.

This fixes `go get` functionality — the standard approach for Go modules
in monorepos where `go.mod` is located in a subdirectory.

### Migration steps

1. Update the dependency:

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

1. Replace import paths in all files:

```bash
# Bulk replacement (Linux/macOS)
find . -name '*.go' -exec sed -i '' \
  's|github.com/BigKAA/topologymetrics/dephealth|github.com/BigKAA/topologymetrics/sdk-go/dephealth|g' {} +
```

1. Update `go.mod` — remove the old dependency:

```bash
go mod tidy
```

The API and SDK behavior remain unchanged — only the module path has changed.

---

## v0.1 → v0.2

### API Changes

| v0.1 | v0.2 | Description |
| --- | --- | --- |
| `dephealth.New(...)` | `dephealth.New("my-service", ...)` | Required first argument `name` |
| `dephealth.Critical(true)` (optional) | `dephealth.Critical(true/false)` (required) | For each dependency |
| `Endpoint.Metadata` | `Endpoint.Labels` | Field renamed |
| `dephealth.WithMetadata(map)` | `dephealth.WithLabel("key", "value")` | Custom labels |
| `WithOptionalLabels(...)` | removed | Custom labels via `WithLabel` |

### Required Changes

1. Add `name` as the first argument to `dephealth.New()`:

```go
// v0.1
dh, err := dephealth.New(
    dephealth.Postgres("postgres-main", ...),
)

// v0.2
dh, err := dephealth.New("my-service",
    dephealth.Postgres("postgres-main", ...),
)
```

1. Specify `Critical()` for each dependency:

```go
// v0.1 — Critical is optional
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
)

// v0.2 — Critical is required
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
    dephealth.Critical(false),
)
```

1. Replace `WithMetadata` with `WithLabel` (if used):

```go
// v0.1
dephealth.WithMetadata(map[string]string{"role": "primary"})

// v0.2
dephealth.WithLabel("role", "primary")
```

### New metric labels

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Update PromQL queries and Grafana dashboards to include the `name` and `critical` labels.
