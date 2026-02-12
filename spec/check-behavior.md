*[Русская версия](check-behavior.ru.md)*

# Check Behavior Contract

> Specification version: **1.0-draft**
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

**Algorithm**:

1. Send `GET` (or configured method) to `http(s)://{host}:{port}{healthPath}`.
2. Wait for response within `timeout`.
3. If response status is in the `expectedStatuses` range — **success**.
4. Otherwise — **failure**.

**Specifics**:

- Response body is not analyzed (only status code).
- Redirects (3xx) are followed automatically; final response status is checked.
- For `https://`, TLS is used; if certificate is invalid and `tlsSkipVerify = false` — failure.
- `User-Agent: dephealth/<version>` header is set.

### 4.2. gRPC (`type: grpc`)

**Protocol**: [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
(package `grpc.health.v1`, method `Health/Check`).

| Parameter | Description | Default Value |
| --- | --- | --- |
| `serviceName` | Service name for Health Check | `""` (empty string — overall status) |
| `tlsEnabled` | Use TLS | `false` |
| `tlsSkipVerify` | Skip TLS certificate verification | `false` |

**Algorithm**:

1. Establish gRPC connection to `{host}:{port}`.
2. Call `grpc.health.v1.Health/Check` with the specified `serviceName`.
3. If response is `SERVING` — **success**.
4. Other statuses (`NOT_SERVING`, `UNKNOWN`, `SERVICE_UNKNOWN`) — **failure**.

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

### 6.2. Panic / Unexpected Errors

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
