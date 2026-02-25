*[Русская версия](check-behavior.ru.md)*

# Check Behavior Contract

> Specification version: **2.0-draft**
>
> This document describes the behavior of dependency health checks:
> lifecycle, threshold logic, check types, operating modes, and error handling.
> All SDKs must implement the described behavior. Compliance is verified
> by conformance tests.

---

## 1. Default Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `checkInterval` | `15s` | Interval between the start of consecutive checks |
| `timeout` | `5s` | Maximum time to wait for a response from the dependency |
| `initialDelay` | `5s` | Delay before the first check after SDK startup |
| `failureThreshold` | `1` | Number of consecutive failures to transition to unhealthy |
| `successThreshold` | `1` | Number of consecutive successes to return to healthy |

All parameters are configurable:

- Globally — for all dependencies.
- Individually — for a specific dependency (takes priority over global settings).

**Value constraints**:

| Parameter | Minimum | Maximum |
| --- | --- | --- |
| `checkInterval` | `1s` | `10m` |
| `timeout` | `100ms` | `30s` |
| `initialDelay` | `0s` | `5m` |
| `failureThreshold` | `1` | `10` |
| `successThreshold` | `1` | `10` |

The `timeout` value must be less than `checkInterval`. If `timeout >= checkInterval` is specified,
the SDK must return a configuration error during initialization.

---

## 2. Check Lifecycle

### 2.1. State Diagram

```text
Start()
  │
  ▼
[INIT] ── initialDelay ──► [CHECKING]
                               │
                    ┌──────────┤
                    │          │
                    ▼          ▼
              [HEALTHY]   [UNHEALTHY]
                    │          │
                    └────◄─────┘
                 (threshold logic)
                         │
                    Stop()│
                         ▼
                      [STOPPED]
```

### 2.2. Phases

#### INIT — initialization

1. SDK creates a goroutine / thread / task for each dependency.
2. Wait for `initialDelay`.
3. Metrics for this dependency are **not exported**.

#### CHECKING — active checks

1. First check is performed.
2. Based on the result, initial state is set: HEALTHY or UNHEALTHY.
3. Metrics start being exported.
4. Subsequent checks are performed at `checkInterval` intervals.

#### HEALTHY / UNHEALTHY — stable state

- State changes only when threshold is reached
  (see section 3 "Threshold Logic").
- Each check updates `app_dependency_latency_seconds`.
- When state changes, `app_dependency_health` is updated.

#### STOPPED — shutdown

1. Call `Stop()` / `close()` / graceful shutdown.
2. Context cancellation / thread interruption.
3. Wait for current checks to complete (with `timeout` deadline).
4. Metrics **remain** with last values (are not reset).

**Rationale**: resetting metrics on shutdown creates false alerts.
Metrics will disappear on next scrape if the process has terminated.

### 2.3. Check Interval

The `checkInterval` is measured from the **start** of the previous check,
not from its completion.

```text
t=0s     t=15s    t=30s    t=45s
│        │        │        │
▼        ▼        ▼        ▼
[check]  [check]  [check]  [check]
 3ms      5ms      2ms      4ms
```

If a check takes longer than `checkInterval`, the next check
starts immediately after the current one completes (without skipping).

---

## 3. Threshold Logic

### 3.1. Transition HEALTHY -> UNHEALTHY

```text
failureThreshold = 3

Check:     OK   OK   FAIL  FAIL  FAIL  → UNHEALTHY
Counter:   0    0    1     2     3
Metric:    1    1    1     1     0
```

- On each failed check, the `consecutiveFailures` counter increments by 1.
- On a successful check, the `consecutiveFailures` counter is reset to 0.
- When `consecutiveFailures >= failureThreshold`, state transitions to UNHEALTHY.
- The `app_dependency_health` metric changes to `0` **at the moment the threshold is reached**.

### 3.2. Transition UNHEALTHY -> HEALTHY

```text
successThreshold = 2

Check:     FAIL  FAIL  OK   OK   → HEALTHY
Counter:   0     0     1    2
Metric:    0     0     0    1
```

- On each successful check, the `consecutiveSuccesses` counter increments by 1.
- On a failed check, the `consecutiveSuccesses` counter is reset to 0.
- When `consecutiveSuccesses >= successThreshold`, state transitions to HEALTHY.
- The `app_dependency_health` metric changes to `1` **at the moment the threshold is reached**.

### 3.3. Initial State

Before the first check, state is **UNKNOWN** (metric is not exported).

The first check determines the initial state:

- Success → immediately HEALTHY (`app_dependency_health = 1`), without considering `successThreshold`.
- Failure → immediately UNHEALTHY (`app_dependency_health = 0`), without considering `failureThreshold`.

**Rationale**: at service startup, it's important to get the actual
dependency status as quickly as possible. Threshold logic protects against brief
failures in steady-state operation.

### 3.4. Counters with Threshold 1

With `failureThreshold = 1` and `successThreshold = 1` (default values),
each check immediately updates the state:

```text
Check:   OK   FAIL  OK   FAIL  OK
Metric:  1    0     1    0     1
```

---

## 4. Check Types

### 4.1. HTTP (`type: http`)

| Parameter | Description | Default Value |
| --- | --- | --- |
| `healthPath` | Path for the check | `/health` |
| `method` | HTTP method | `GET` |
| `expectedStatuses` | Expected HTTP status codes | `200-299` (any 2xx) |
| `tlsSkipVerify` | Skip TLS certificate verification | `false` |
| `headers` | Custom HTTP headers added to every request | `{}` (empty) |
| `bearerToken` | Adds `Authorization: Bearer <token>` header | `""` (disabled) |
| `basicAuth` | Adds `Authorization: Basic <base64(user:pass)>` header | not set |

**Algorithm**:

1. Send `GET` (or configured method) to `http(s)://{host}:{port}{healthPath}`.
2. Add configured headers (custom headers, bearer token, or basic auth) to the request.
3. Wait for response within `timeout`.
4. If response status is in the `expectedStatuses` range — **success**.
5. Otherwise — **failure**.

**Authentication**:

- `headers` — arbitrary key-value pairs added as HTTP headers to every health-check request.
- `bearerToken` — convenience parameter; adds `Authorization: Bearer <token>` header.
- `basicAuth` — convenience parameter with `username` and `password` fields;
  adds `Authorization: Basic <base64(username:password)>` header.
- Only one authentication method is allowed at a time. If more than one of the
  following is specified, the SDK must return a **validation error** during
  initialization:
  - `bearerToken` is set AND `headers` contains an `Authorization` key
  - `basicAuth` is set AND `headers` contains an `Authorization` key
  - `bearerToken` is set AND `basicAuth` is set

**Specifics**:

- Response body is not analyzed (only status code).
- Redirects (3xx) are followed automatically; final response status is checked.
- For `https://`, TLS is used; if certificate is invalid and `tlsSkipVerify = false` — failure.
- `User-Agent: dephealth/<version>` header is set. A custom `User-Agent` in `headers` overrides it.
- HTTP 401 and 403 responses are classified as `auth_error` (see section 6.2.3).

### 4.2. gRPC (`type: grpc`)

**Protocol**: [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
(package `grpc.health.v1`, method `Health/Check`).

| Parameter | Description | Default Value |
| --- | --- | --- |
| `serviceName` | Service name for Health Check | `""` (empty string — overall status) |
| `tlsEnabled` | Use TLS | `false` |
| `tlsSkipVerify` | Skip TLS certificate verification | `false` |
| `metadata` | Custom gRPC metadata added to every Health/Check call | `{}` (empty) |
| `bearerToken` | Adds `authorization: Bearer <token>` metadata | `""` (disabled) |
| `basicAuth` | Adds `authorization: Basic <base64(user:pass)>` metadata | not set |

**Algorithm**:

1. Establish gRPC connection to `{host}:{port}`.
2. Add configured metadata (custom metadata, bearer token, or basic auth) to the call.
3. Call `grpc.health.v1.Health/Check` with the specified `serviceName`.
4. If response is `SERVING` — **success**.
5. Other statuses (`NOT_SERVING`, `UNKNOWN`, `SERVICE_UNKNOWN`) — **failure**.

**Authentication**:

- `metadata` — arbitrary key-value pairs added as gRPC metadata to every Health/Check call.
- `bearerToken` — convenience parameter; adds `authorization: Bearer <token>` metadata.
- `basicAuth` — convenience parameter with `username` and `password` fields;
  adds `authorization: Basic <base64(username:password)>` metadata.
- Same validation rules as HTTP (section 4.1): only one authentication method allowed.
  Conflict between `bearerToken`, `basicAuth`, and custom `authorization` metadata key
  results in a **validation error**.
- gRPC status `UNAUTHENTICATED` and `PERMISSION_DENIED` are classified as `auth_error`
  (see section 6.2.3).

**Specifics**:

- Connection is created anew for each check (standalone mode).
- Timeout is passed via gRPC deadline.

### 4.3. TCP (`type: tcp`)

**Algorithm**:

1. Establish TCP connection to `{host}:{port}` (with `timeout`).
2. If connection is established — **success**.
3. Immediately close the connection.

**Specifics**:

- No data is sent or read.
- Suitable for arbitrary TCP services without a specific check protocol.

### 4.4. PostgreSQL (`type: postgres`)

| Parameter | Description | Default Value |
| --- | --- | --- |
| `query` | SQL query for the check | `SELECT 1` |

**Standalone mode**:

1. Establish new TCP connection to PostgreSQL (`{host}:{port}`).
2. Perform authentication (if required).
3. Execute `SELECT 1`.
4. If result is received — **success**.
5. Close the connection.

**Connection pool mode**:

1. Acquire connection from pool (`db.QueryContext(ctx, "SELECT 1")`).
2. If query succeeds — **success**.
3. Connection is returned to the pool.

**Specifics**:

- In pool mode, not only database availability is checked, but also pool health.
- If pool is exhausted and `timeout` expires before acquiring a connection — **failure**.
- TLS support (if specified in connection string).

### 4.5. MySQL (`type: mysql`)

Similar to PostgreSQL. The only difference is the connection driver.

| Parameter | Description | Default Value |
| --- | --- | --- |
| `query` | SQL query for the check | `SELECT 1` |

### 4.6. Redis (`type: redis`)

**Standalone mode**:

1. Establish TCP connection to Redis (`{host}:{port}`).
2. Perform authentication (if password is specified).
3. Send `PING` command.
4. Wait for `PONG` response — **success**.
5. Close the connection.

**Connection pool mode**:

1. Use existing client (`client.Ping(ctx)`).
2. If response is `PONG` — **success**.

**Specifics**:

- Redis Sentinel and Redis Cluster support is not included in v1.0.
- TLS support (`rediss://`).
- Database selection support (`/0`, `/1`, ...).

### 4.7. AMQP (`type: amqp`)

**Standalone mode**:

1. Establish AMQP connection to `{host}:{port}` (with specified `vhost`).
2. If connection is established (connection.open) — **success**.
3. Close the connection.

**Specifics**:

- Channel is not created (connection-level check is sufficient).
- TLS support (`amqps://`).
- Vhost support (from URL or separate parameter).
- Authentication: username/password from URL or configuration.

### 4.8. Kafka (`type: kafka`)

**Standalone mode**:

1. Create Kafka Admin Client (or minimal client).
2. Send Metadata request to broker `{host}:{port}`.
3. If response with metadata is received — **success**.
4. Close the client.

**Specifics**:

- Each broker is checked independently.
- Only network availability and response capability of the broker are checked.
- Topic-level checks are not included in v1.0.
- SASL authentication support is not included in v1.0.

### 4.9. LDAP (`type: ldap`)

| Parameter | Description | Default Value |
| --- | --- | --- |
| `checkMethod` | Check method | `root_dse` |
| `bindDN` | DN for Simple Bind | `""` |
| `bindPassword` | Password for Simple Bind | `""` |
| `baseDN` | Base DN for search method | `""` |
| `searchFilter` | LDAP filter for search method | `(objectClass=*)` |
| `searchScope` | Search scope: `base`, `one`, `sub` | `base` |
| `startTLS` | Use StartTLS (only with `ldap://`) | `false` |
| `tlsSkipVerify` | Skip TLS certificate verification | `false` |

**Check methods** (`checkMethod` values):

| Value | Description |
| --- | --- |
| `anonymous_bind` | Anonymous Bind operation |
| `simple_bind` | Simple Bind with `bindDN` / `bindPassword` |
| `root_dse` | Search: base=`""`, scope=base, filter=`(objectClass=*)` |
| `search` | Search with `baseDN`, `searchScope`, `searchFilter` |

**Standalone mode**:

1. Establish TCP connection to `{host}:{port}` (with `timeout`).
2. If scheme is `ldaps://` — perform TLS handshake.
3. If `startTLS=true` — send StartTLS extended operation, then TLS handshake.
4. Execute check based on `checkMethod`:
   - `anonymous_bind`: Anonymous Bind operation.
   - `simple_bind`: Simple Bind with `bindDN` / `bindPassword`.
   - `root_dse`: Search with base=`""`, scope=base, filter=`(objectClass=*)`.
   - `search`: Search with `baseDN`, `searchScope`, `searchFilter`.
5. If operation completes without error — **success**.
6. Close the connection.

**Connection pool mode**:

1. Use existing LDAP connection.
2. Execute check method (same as standalone step 4).
3. If operation completes without error — **success**.

**Validation rules**:

| Condition | Result |
| --- | --- |
| `simple_bind` without `bindDN` or `bindPassword` | Configuration error |
| `search` without `baseDN` | Configuration error |
| `startTLS=true` with `ldaps://` scheme | Configuration error (incompatible) |

**Specifics**:

- LDAP referrals are not followed.
- LDAP result code 49 (Invalid Credentials) and 50 (Insufficient Access Rights) are
  classified as `auth_error` (see section 6.2.3).
- LDAP operational errors (server down, busy, unavailable) are classified as `unhealthy`.
- TLS certificate validation respects `tlsSkipVerify` for both `ldaps://` and StartTLS.

---

## 5. Two Operating Modes

### 5.1. Standalone Mode

SDK **creates a new connection** for each check.

```text
SDK → net.Dial / http.Get / sql.Open → check → Close
```

**When to use**:

- Dependency does not have a connection pool in the service (HTTP, gRPC, TCP).
- Developer did not provide a reference to the pool.

**Advantages**:

- Simple configuration — URL is sufficient.
- Independent of the service's pool state.

**Disadvantages**:

- Does not reflect the actual ability of the service to work with the dependency.
- If pool is exhausted, standalone check will still show `healthy`.
- Additional overhead (creating/closing connections).

### 5.2. Connection Pool Mode (Integrated)

SDK **uses the service's existing connection pool**.

```text
SDK → pool.GetConnection() → SELECT 1 / PING → pool.Release()
```

**When to use**:

- Dependency uses a connection pool (DB, Redis).
- Developer provided a reference to the pool / client.

**Advantages**:

- Reflects the actual ability of the service to work with the dependency.
- If pool is exhausted — check shows `unhealthy`.
- No additional connections.

**Disadvantages**:

- Requires passing a pool reference during initialization.
- Depends on the specific library (go-redis, database/sql, etc.).

### 5.3. Priority

Pool mode is **preferred**. SDK should encourage its use
in documentation and examples. Standalone mode is a fallback for cases
when pool is not available.

---

## 6. Error Handling

All listed situations are treated as check **failure**:

| Situation | Behavior |
| --- | --- |
| Timeout (`timeout` expired) | Failure. Latency = actual time until timeout |
| DNS resolution failure | Failure. Latency = time until DNS error |
| Connection refused | Failure. Latency = time until receiving RST |
| Connection reset | Failure. Latency = time until reset |
| TLS handshake failure | Failure. Latency = time until TLS error |
| HTTP 5xx | Failure (not 2xx) |
| HTTP 3xx (redirect) | Follows redirect; result determined by final response |
| HTTP 4xx | Failure (not 2xx) |
| gRPC NOT_SERVING | Failure |
| SQL error (authentication, syntax) | Failure |
| Pool exhausted (connection acquisition timeout) | Failure |
| Panic / unhandled exception | Failure. SDK must catch and log |

### 6.1. Error Logging

SDK logs each check error at `WARN` level:

```text
WARN dephealth: check failed dependency=postgres-main host=pg.svc port=5432 error="connection refused"
```

First transition to unhealthy is logged at `ERROR` level:

```text
ERROR dephealth: dependency unhealthy dependency=postgres-main host=pg.svc port=5432 consecutive_failures=3
```

Return to healthy is logged at `INFO` level:

```text
INFO dephealth: dependency recovered dependency=postgres-main host=pg.svc port=5432
```

### 6.2. Error Classification

Every check error is classified into a **status category** and a **detail value**.
This classification is used to populate the `app_dependency_status` and
`app_dependency_status_detail` metrics (see metric-contract.md, sections 8-9).

#### 6.2.1. Classification Chain

The scheduler classifies errors using the following priority chain:

1. **ClassifiedError interface** — if the error implements the `ClassifiedError`
   interface (Go: `StatusCategory() string` + `StatusDetail() string`;
   Java/C#: base exception class; Python: properties on `CheckError`),
   its category and detail are used directly.
2. **Sentinel errors** — known typed errors (`ErrTimeout`, `ErrConnectionRefused`,
   `ErrUnhealthy`, etc.) are mapped to their corresponding category/detail.
3. **Platform error detection** — standard library error types are inspected:
   - `context.DeadlineExceeded` / `TimeoutException` → `timeout` / `timeout`
   - `*net.DNSError` / `UnknownHostException` / `socket.gaierror` → `dns_error` / `dns_error`
   - `*net.OpError` (connection refused) / `ConnectException` → `connection_error` / `connection_refused`
   - `*tls.CertificateVerificationError` / `SSLException` → `tls_error` / `tls_error`
4. **Fallback** — unrecognized errors → `error` / `error`.

#### 6.2.2. Successful Check

A successful check (no error) always produces:

- status category: `ok`
- detail: `ok`

#### 6.2.3. Detail Values by Checker Type

| Checker Type | Possible detail values |
| --- | --- |
| HTTP | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `http_NNN` (e.g., `http_404`, `http_503`), `error` |
| gRPC | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `grpc_not_serving`, `grpc_unknown`, `error` |
| TCP | `ok`, `timeout`, `connection_refused`, `dns_error`, `error` |
| PostgreSQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| MySQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| Redis | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `unhealthy`, `error` |
| AMQP | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `unhealthy`, `error` |
| Kafka | `ok`, `timeout`, `connection_refused`, `dns_error`, `no_brokers`, `error` |
| LDAP | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `unhealthy`, `error` |

#### 6.2.4. Detail to Status Mapping

| detail | status |
| --- | --- |
| `ok` | `ok` |
| `timeout` | `timeout` |
| `connection_refused`, `network_unreachable`, `host_unreachable` | `connection_error` |
| `dns_error` | `dns_error` |
| `auth_error` | `auth_error` |
| `tls_error` | `tls_error` |
| `http_NNN`, `grpc_not_serving`, `grpc_unknown`, `unhealthy`, `no_brokers` | `unhealthy` |
| `error`, `pool_exhausted`, `query_error` | `error` |

#### 6.2.5. Backward Compatibility

The `HealthChecker` interface is **not changed**. Checkers that do not return
classified errors are handled by the platform error detection and fallback
in the classification chain (steps 3-4). User-implemented custom checkers
will automatically receive classification through this mechanism.

### 6.3. Panic / Unexpected Errors

SDK must catch panic (Go), unhandled exception (Java/C#/Python)
inside the check. Panic must not interrupt the scheduler's operation.
Check is considered failed, error is logged at `ERROR` level.

---

## 7. Concurrency and Thread Safety

- Checks for different dependencies run **in parallel** (each in its own goroutine / thread).
- Checks for different endpoints of the same dependency also run **in parallel**.
- Metric updates are **thread-safe** (guaranteed by Prometheus client).
- Internal state updates (threshold counters) are **thread-safe**
  (atomic / mutex / synchronized).
- `Start()` and `Stop()` can be called only once. Repeated call to `Start()`
  returns an error. Repeated call to `Stop()` is a no-op.

---

## 8. Programmatic Health Details API

The `HealthDetails()` method provides a programmatic API that exposes
the detailed health check state for every registered endpoint. Unlike
`Health()`, which returns a simple `healthy/unhealthy` boolean map,
`HealthDetails()` returns a rich structure with classification, latency,
metadata, and timestamps.

This API enables consumers (status pages, reverse proxies, custom operators)
to build enriched health endpoints without scraping Prometheus metrics.

### 8.1. Public Method

Each SDK's main facade exposes a `HealthDetails()` method:

| SDK | Facade | Method | Return type |
| --- | --- | --- | --- |
| Go | `DepHealth` | `HealthDetails()` | `map[string]EndpointStatus` |
| Java | `DepHealth` | `healthDetails()` | `Map<String, EndpointStatus>` |
| Python | `DependencyHealth` | `health_details()` | `dict[str, EndpointStatus]` |
| C# | `DepHealthMonitor` | `HealthDetails()` | `Dictionary<string, EndpointStatus>` |

The method delegates to the scheduler using the same pattern as `Health()`.

### 8.2. Key Format

Map keys use the format `"dependency:host:port"`, consistent across all SDKs.
Keys in `HealthDetails()` **must match** the keys in `Health()` for
Go, Java, and C#.

> **Note**: Python's existing `health()` aggregates by dependency name only.
> The new `health_details()` intentionally uses per-endpoint keys
> `"dependency:host:port"` for consistency with other SDKs and to provide
> endpoint-level granularity. This is a documented difference.

### 8.3. `EndpointStatus` Structure

The `EndpointStatus` structure contains 11 fields. All fields are required
in every SDK implementation.

| # | Field | Go | Java | Python | C# | Description |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Healthy | `*bool` | `Boolean` | `bool \| None` | `bool?` | `nil`/`null`/`None` = UNKNOWN |
| 2 | Status | `StatusCategory` | `StatusCategory` | `StatusCategory` | `StatusCategory` | Classification category |
| 3 | Detail | `string` | `String` | `str` | `string` | Specific failure reason |
| 4 | Latency | `time.Duration` | `Duration` | `float` (seconds) | `TimeSpan` | Duration of last check |
| 5 | Type | `DependencyType` | `DependencyType` | `DependencyType` | `DependencyType` | Dependency type |
| 6 | Name | `string` | `String` | `str` | `string` | Dependency logical name |
| 7 | Host | `string` | `String` | `str` | `string` | Endpoint host |
| 8 | Port | `string` | `String` | `str` | `string` | Endpoint port |
| 9 | Critical | `bool` | `boolean` | `bool` | `bool` | Whether dependency is critical |
| 10 | LastCheckedAt | `time.Time` | `Instant` | `datetime \| None` | `DateTimeOffset?` | Timestamp of last check |
| 11 | Labels | `map[string]string` | `Map<String, String>` | `dict[str, str]` | `Dictionary<string, string>` | Custom labels |

**Field details**:

- **Healthy**: tri-state — `true` (healthy), `false` (unhealthy),
  `nil`/`null`/`None` (unknown, before first check).
- **Status**: typed category from `StatusCategory` (see section 8.4).
- **Detail**: specific reason string matching the detail values
  from section 6.2.3. For unknown state: `"unknown"`.
- **Latency**: duration of the last completed health check.
  Zero before first check.
- **Type**: the configured dependency type (`"http"`, `"postgres"`, etc.).
- **Name**: the configured logical name of the dependency.
- **Host**, **Port**: the endpoint's host and port.
- **Critical**: whether the dependency was marked as critical.
- **LastCheckedAt**: wall-clock timestamp of when the last check completed.
  Zero value / `null` / `None` before first check.
- **Labels**: custom labels configured on the endpoint.
  Empty map (not null) if no labels are configured.

### 8.4. `StatusCategory` Type

`StatusCategory` is a typed alias for status category string values.
It wraps the existing status constants used by the error classification
system (section 6.2).

**Values** (9 total):

| Value | Description |
| --- | --- |
| `ok` | Check succeeded |
| `timeout` | Check timed out |
| `connection_error` | Connection failed (refused, reset, unreachable) |
| `dns_error` | DNS resolution failed |
| `auth_error` | Authentication failed |
| `tls_error` | TLS handshake failed |
| `unhealthy` | Dependency responded but reported unhealthy status |
| `error` | Unclassified error |
| `unknown` | No check performed yet (initial state) |

**Language-specific implementation**:

| SDK | Implementation |
| --- | --- |
| Go | `type StatusCategory string` with typed constants |
| Java | `enum StatusCategory` with `String value()` method |
| Python | `str` type alias with module-level constants |
| C# | `static class StatusCategory` with `string` constants |

The first 8 values (`ok` through `error`) are aliases for existing
constants used in metrics. The `unknown` value is new, added specifically
for the `HealthDetails()` API to represent the pre-first-check state.

### 8.5. UNKNOWN State

Endpoints that have not yet completed their first check are **included**
in the `HealthDetails()` result with the following values:

| Field | Value |
| --- | --- |
| Healthy | `nil` / `null` / `None` |
| Status | `"unknown"` |
| Detail | `"unknown"` |
| Latency | zero |
| LastCheckedAt | zero value / `null` / `None` |
| Type, Name, Host, Port, Critical, Labels | Populated from configuration |

> This differs from `Health()`, which **excludes** endpoints in UNKNOWN state.
> Rationale: `HealthDetails()` provides a complete view for status pages;
> excluding endpoints hides important information about startup state.

### 8.6. Lifecycle Behavior

| State | `HealthDetails()` returns |
| --- | --- |
| Before `Start()` | `nil` / `null` / empty (no endpoints registered) |
| After `Start()`, before first check | All endpoints with UNKNOWN state (section 8.5) |
| Running | Current state of all endpoints |
| After `Stop()` | Last known state (frozen snapshot) |

### 8.7. Data Sourcing

All data comes from existing internal scheduler state. The following
fields must be stored in the endpoint state during `executeCheck()`:

| Field | Source | When stored |
| --- | --- | --- |
| Healthy | Existing `healthy` field | Already stored |
| Status | `classifyError(err).Category` | After each check |
| Detail | `classifyError(err).Detail` | After each check |
| Latency | Check duration | After each check |
| LastCheckedAt | `time.Now()` / `Instant.now()` / `datetime.now(UTC)` | After each check |
| Type | `dependency.Type` | At state creation |
| Name | `dependency.Name` | At state creation |
| Host | `endpoint.Host` | At state creation |
| Port | `endpoint.Port` | At state creation |
| Critical | `dependency.Critical` | At state creation |
| Labels | `endpoint.Labels` | At state creation |

### 8.8. Thread Safety

`HealthDetails()` **must be safe** to call concurrently from multiple
goroutines / threads. Implementation follows the existing `Health()` pattern:

1. Lock the scheduler mutex to access the states map.
2. Iterate states, locking each endpoint state individually.
3. Copy values into a result map under the lock.
4. Return the result map (caller owns it; modifications do not affect
   internal state).

### 8.9. JSON Serialization

`EndpointStatus` must be serializable to JSON without additional work.
All SDKs must use **snake_case** field names in JSON output.

**Canonical JSON format**:

```json
{
  "postgres-main:pg.svc:5432": {
    "healthy": true,
    "status": "ok",
    "detail": "ok",
    "latency_ms": 2.3,
    "type": "postgres",
    "name": "postgres-main",
    "host": "pg.svc",
    "port": "5432",
    "critical": true,
    "last_checked_at": "2026-02-14T10:30:00Z",
    "labels": {"role": "primary"}
  },
  "redis-cache:redis.svc:6379": {
    "healthy": null,
    "status": "unknown",
    "detail": "unknown",
    "latency_ms": 0,
    "type": "redis",
    "name": "redis-cache",
    "host": "redis.svc",
    "port": "6379",
    "critical": false,
    "last_checked_at": null,
    "labels": {}
  }
}
```

**Field serialization rules**:

| Field | JSON type | Notes |
| --- | --- | --- |
| `healthy` | `boolean` or `null` | `null` for UNKNOWN state |
| `status` | `string` | One of 9 StatusCategory values |
| `detail` | `string` | Detail value from error classification |
| `latency_ms` | `number` | Milliseconds as float (e.g., `2.3`) |
| `type` | `string` | Dependency type |
| `name` | `string` | Dependency name |
| `host` | `string` | Endpoint host |
| `port` | `string` | Endpoint port (always string) |
| `critical` | `boolean` | Never null |
| `last_checked_at` | `string` or `null` | ISO 8601 UTC format; `null` before first check |
| `labels` | `object` | Empty `{}` if no labels (never null) |

### 8.10. Backward Compatibility

- `Health()` remains unchanged in all SDKs.
- `EndpointStatus` is a new exported type.
- `StatusCategory` is a new exported type.
- `HealthDetails()` is a new method on existing facades.
- No changes to metrics behavior.
- No changes to configuration.
