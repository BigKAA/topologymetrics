*[Русская версия](config-contract.ru.md)*

# Configuration Contract

> Specification version: **3.0-draft**
>
> This document describes the input data formats for dependency configuration,
> parsing rules, programmatic API, and configuration via environment variables.
> All SDKs must support the described formats. Compliance is verified
> by conformance tests.

---

## 1. Supported Input Formats

SDK accepts connection information for a dependency in four formats:

| # | Format | Example | Priority |
| --- | --- | --- | --- |
| 1 | Full URL | `postgres://user:pass@host:5432/db` | Primary |
| 2 | Separate parameters | `host=pg.svc, port=5432` | Alternative |
| 3 | Connection string | `Host=pg.svc;Port=5432;Database=db` | For .NET/JDBC |
| 4 | JDBC URL | `jdbc:postgresql://host:5432/db` | For Java |

SDK must correctly extract from any format:

- `host` — endpoint address
- `port` — endpoint port
- `type` — dependency type (if determined from URL scheme)

---

## 2. Format 1: Full URL

### 2.1. Supported Schemes

| Scheme | Determined `type` | Default Port |
| --- | --- | --- |
| `postgres://`, `postgresql://` | `postgres` | `5432` |
| `mysql://` | `mysql` | `3306` |
| `redis://` | `redis` | `6379` |
| `rediss://` | `redis` | `6379` |
| `amqp://` | `amqp` | `5672` |
| `amqps://` | `amqp` | `5671` |
| `http://` | `http` | `80` |
| `https://` | `http` | `443` |
| `grpc://` | `grpc` | `443` |
| `kafka://` | `kafka` | `9092` |
| `ldap://` | `ldap` | `389` |
| `ldaps://` | `ldap` | `636` |

### 2.2. URL Parsing Rules

General format:

```text
scheme://[user:password@]host[:port][/path][?query]
```

**Extracting host**:

- From the `host` component of the URL.
- IPv6 addresses in URLs are enclosed in square brackets: `[::1]`.
- In the metric's `host` label, IPv6 is written **without** square brackets: `::1`.

**Extracting port**:

- From the `port` component of the URL.
- If port is not specified, the default port for the given scheme is used
  (see table above).

**Extracting type**:

- Automatically from the URL scheme (see table above).
- If the developer explicitly specified `type`, it takes priority.

**Extracting additional data**:

| Data | Source | Used in |
| --- | --- | --- |
| `vhost` | Path component (`amqp://host/vhost`) | `vhost` label, connection to AMQP |
| `database` | Path component (`redis://host/0`) | Redis database selection |
| Credentials | Userinfo (`user:pass@`) | Authentication during autonomous check |

If credentials are specified in the URL, they **MUST** be passed to the checker
for autonomous checking.
Priority: explicit API parameters > credentials from URL > default values.

### 2.3. URLs with Multiple Hosts

Some dependencies support specifying multiple hosts in a URL:

```text
# Kafka brokers
kafka://broker-0:9092,broker-1:9092,broker-2:9092

# PostgreSQL failover (libpq format)
postgres://user:pass@primary:5432,replica:5432/db?target_session_attrs=read-write
```

**Rules**:

- Each `host:port` pair creates a separate `Endpoint`.
- All endpoints are combined into one `Dependency`.
- If port is specified only for the first host, the rest use the same port.
- If port is not specified for any — the default port is used.

### 2.4. URL Parsing Examples

| URL | host | port | type |
| --- | --- | --- | --- |
| `postgres://app:pass@pg.svc:5432/orders` | `pg.svc` | `5432` | `postgres` |
| `postgres://app:pass@pg.svc/orders` | `pg.svc` | `5432` | `postgres` |
| `redis://redis.svc:6379/0` | `redis.svc` | `6379` | `redis` |
| `redis://redis.svc` | `redis.svc` | `6379` | `redis` |
| `rediss://redis.svc:6380/0` | `redis.svc` | `6380` | `redis` |
| `http://payment.svc:8080/health` | `payment.svc` | `8080` | `http` |
| `https://payment.svc/health` | `payment.svc` | `443` | `http` |
| `amqp://user:pass@rabbit.svc/orders` | `rabbit.svc` | `5672` | `amqp` |
| `kafka://broker-0.svc:9092` | `broker-0.svc` | `9092` | `kafka` |
| `postgres://[::1]:5432/db` | `::1` | `5432` | `postgres` |
| `ldap://ldap.svc:389` | `ldap.svc` | `389` | `ldap` |
| `ldap://ldap.svc` | `ldap.svc` | `389` | `ldap` |
| `ldaps://ldap.svc:636` | `ldap.svc` | `636` | `ldap` |
| `ldaps://ldap.svc` | `ldap.svc` | `636` | `ldap` |

---

## 3. Format 2: Separate Parameters

Minimum configuration: `host` + `port`.

```go
dephealth.Postgres("postgres-main", dephealth.FromParams("pg.svc", "5432"))
```

**Rules**:

- `host` — required. String (hostname, IPv4 or IPv6).
- `port` — required. String with number 1-65535.
- `type` — must be specified explicitly (via factory method or parameter).
- For invalid port (not a number, out of range) — configuration error.

---

## 4. Format 3: Connection String

Format `Key=Value;Key=Value`, common in .NET ecosystem.

### 4.1. Supported Keys for host

Search (case-insensitive, first matching result):

| Key | Description |
| --- | --- |
| `Host` | Primary |
| `Server` | Alternative (SQL Server) |
| `Data Source` | Alternative (Oracle, SQLite) |
| `Address` | Alternative |
| `Addr` | Alternative |
| `Network Address` | Alternative |

### 4.2. Supported Keys for port

| Key | Description |
| --- | --- |
| `Port` | Primary |

If port is not found as a separate key, the format `Host=hostname,port`
(SQL Server convention) and `Host=hostname:port` are checked.

### 4.3. Connection String Parsing Examples

| Connection string | host | port |
| --- | --- | --- |
| `Host=pg.svc;Port=5432;Database=orders` | `pg.svc` | `5432` |
| `Server=pg.svc,5432;Database=orders` | `pg.svc` | `5432` |
| `Host=pg.svc:5432;Database=orders` | `pg.svc` | `5432` |
| `Data Source=pg.svc;Port=5432` | `pg.svc` | `5432` |

### 4.4. Limitations

- `type` is not determined automatically from connection string.
  Developer must specify type explicitly.
- If `host` is not found — configuration error.
- If `port` is not found — default port for specified `type` is used.

---

## 5. Format 4: JDBC URL

Format specific to Java ecosystem.

### 5.1. General Format

```text
jdbc:<subprotocol>://host[:port][/database][?parameters]
```

### 5.2. Supported Subprotocols

| Subprotocol | Determined `type` | Default Port |
| --- | --- | --- |
| `postgresql` | `postgres` | `5432` |
| `mysql` | `mysql` | `3306` |

### 5.3. Parsing Rules

1. Remove `jdbc:` prefix.
2. Parse the remaining part as a standard URL.
3. Extract `host`, `port`, `type` using the same rules.

### 5.4. Examples

| JDBC URL | host | port | type |
| --- | --- | --- | --- |
| `jdbc:postgresql://pg.svc:5432/orders` | `pg.svc` | `5432` | `postgres` |
| `jdbc:postgresql://pg.svc/orders` | `pg.svc` | `5432` | `postgres` |
| `jdbc:mysql://mysql.svc:3306/orders` | `mysql.svc` | `3306` | `mysql` |

---

## 6. Default Port Table

| Type | Default Port | Protocol |
| --- | --- | --- |
| `postgres` | `5432` | TCP |
| `mysql` | `3306` | TCP |
| `redis` | `6379` | TCP |
| `amqp` | `5672` | TCP |
| `amqp` (TLS) | `5671` | TCP + TLS |
| `http` | `80` | TCP |
| `http` (TLS) | `443` | TCP + TLS |
| `grpc` | `443` | TCP (HTTP/2) |
| `kafka` | `9092` | TCP |
| `ldap` | `389` | TCP |
| `ldap` (TLS) | `636` | TCP + TLS |
| `tcp` | — | TCP (port is required) |

For `type: tcp` the default port is **not defined** — developer must specify it explicitly.

---

## 7. Programmatic Configuration API

### 7.1. Factory Methods (Dependency Creation)

Each dependency type has a factory method:

```go
// Go
dephealth.HTTP(name, source, ...opts)
dephealth.GRPC(name, source, ...opts)
dephealth.TCP(name, source, ...opts)
dephealth.Postgres(name, source, ...opts)
dephealth.MySQL(name, source, ...opts)
dephealth.Redis(name, source, ...opts)
dephealth.AMQP(name, source, ...opts)
dephealth.Kafka(name, source, ...opts)
dephealth.LDAP(name, source, ...opts)
```

**Parameters**:

- `name` (required) — logical dependency name. Format: `[a-z][a-z0-9-]*`,
  length 1-63 characters. For invalid name — configuration error.
- `source` (required) — configuration source (URL, parameters, pool).
- `opts` (optional) — additional settings (including mandatory `critical`).

### 7.2. Configuration Sources (Source)

```go
// From URL
dephealth.FromURL("postgres://user:pass@host:5432/db")

// From separate parameters
dephealth.FromParams("host", "5432")

// From connection string
dephealth.FromConnectionString("Host=pg.svc;Port=5432;Database=orders")

// From JDBC URL
dephealth.FromJDBC("jdbc:postgresql://host:5432/db")
```

### 7.3. Dependency Options (DependencyOption)

```go
// Criticality — required for each dependency, no default value.
// If not specified — configuration error.
dephealth.Critical(true)  // critical="yes"
dephealth.Critical(false) // critical="no"

// Individual check interval
dephealth.WithCheckInterval(30 * time.Second)

// Individual timeout
dephealth.WithTimeout(10 * time.Second)

// Individual initial delay
dephealth.WithInitialDelay(0)

// Thresholds
dephealth.WithFailureThreshold(3)
dephealth.WithSuccessThreshold(2)

// HTTP-specific
dephealth.WithHealthPath("/ready")
dephealth.WithTLSSkipVerify(true)

// HTTP authentication (mutually exclusive)
dephealth.WithHTTPHeaders(map[string]string{"X-API-Key": "my-key"})
dephealth.WithHTTPBearerToken("eyJhbG...")
dephealth.WithHTTPBasicAuth("admin", "secret")

// gRPC authentication (mutually exclusive)
dephealth.WithGRPCMetadata(map[string]string{"x-custom": "value"})
dephealth.WithGRPCBearerToken("eyJhbG...")
dephealth.WithGRPCBasicAuth("admin", "secret")

// LDAP-specific
dephealth.WithLDAPCheckMethod("root_dse")     // anonymous_bind, simple_bind, root_dse, search
dephealth.WithLDAPBindDN("cn=admin,dc=example,dc=com")
dephealth.WithLDAPBindPassword("secret")
dephealth.WithLDAPBaseDN("dc=example,dc=com")
dephealth.WithLDAPSearchFilter("(objectClass=*)")
dephealth.WithLDAPSearchScope("base")         // base, one, sub
dephealth.WithLDAPStartTLS(true)

// Custom labels
dephealth.WithLabel("role", "primary")
dephealth.WithLabel("shard", "shard-01")
dephealth.WithLabel("vhost", "orders")
```

### 7.4. Global Options (When Creating DepHealth)

```go
dh := dephealth.New("order-api", "billing-team",
    // Global options
    dephealth.WithDefaultCheckInterval(30 * time.Second),
    dephealth.WithDefaultTimeout(10 * time.Second),
    dephealth.WithRegisterer(customRegisterer),
    dephealth.WithLogger(slog.Default()),

    // Dependencies (critical is required for each)
    dephealth.Postgres("postgres-main", dephealth.FromURL(url),
        dephealth.Critical(true)),
    dephealth.Redis("redis-cache", dephealth.FromURL(url),
        dephealth.Critical(false)),
)
```

**Parameters**:

- `name` (required) — unique application name. Format: `[a-z][a-z0-9-]*`,
  length 1-63 characters. Alternatively set via `DEPHEALTH_NAME`
  (API > env var). If missing in both API and env — configuration error.
- `group` (required) — logical group for this service (team, subsystem, project).
  Format: `[a-z][a-z0-9-]*`, length 1-63 characters. Alternatively set via
  `DEPHEALTH_GROUP` (API > env var). If missing in both — configuration error.

### 7.5. Validation

SDK validates configuration on `New()` call and returns error for:

| Issue | Error |
| --- | --- |
| Missing application name | `missing name` |
| Invalid application name | `invalid name: "..."` |
| Missing group | `missing group` |
| Invalid group | `invalid group: "..."` |
| Invalid dependency name | `invalid dependency name: "..."` |
| Missing critical | `missing critical for dependency "..."` |
| Duplicate name + host + port | `duplicate endpoint: "..." host=... port=...` |
| Invalid URL | `invalid URL: "..."` |
| Missing host | `missing host for dependency "..."` |
| Invalid port | `invalid port "..." for dependency "..."` |
| `timeout >= checkInterval` | `timeout must be less than checkInterval for "..."` |
| Unknown type | `unknown dependency type: "..."` |
| Invalid label name | `invalid label name: "..."` |
| Reserved label | `reserved label: "..."` |
| Multiple auth methods | `conflicting auth methods for dependency "..."` |

### 7.6. Valid Configurations

Configuration without dependencies (zero dependencies) is valid.
Leaf services (without outgoing dependencies) are an acceptable pattern
in microservice topology.

**Behavior with zero dependencies**:

| Operation | Behavior |
| --- | --- |
| `New()` / `build()` | Completes without error |
| `Health()` | Returns empty collection |
| `Start()` / `Stop()` | No-op (does not create threads/goroutines) |
| Metrics | Contains no time series |

---

## 8. Configuration via Environment Variables

### 8.1. Format

Instance-level variables:

```text
DEPHEALTH_NAME=<value>
DEPHEALTH_GROUP=<value>
```

Dependency-level variables:

```text
DEPHEALTH_<DEPENDENCY_NAME>_<PARAM>=<value>
```

- `<DEPENDENCY_NAME>` — dependency name in UPPER_SNAKE_CASE.
  Character `-` is replaced with `_`.
- `<PARAM>` — configuration parameter.

**Duration value format**: numbers in seconds (integer or fractional), without unit suffix.
For example: `CHECK_INTERVAL=30`, `TIMEOUT=5`. SDK converts the number to native duration type.

### 8.2. Supported Parameters

#### Instance Level

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Unique application name (`name` label) | `DEPHEALTH_NAME=order-api` |
| `DEPHEALTH_GROUP` | Logical group (`group` label) | `DEPHEALTH_GROUP=billing-team` |

#### Dependency Level

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_<NAME>_URL` | Dependency URL | `DEPHEALTH_POSTGRES_MAIN_URL=postgres://...` |
| `DEPHEALTH_<NAME>_HOST` | Host | `DEPHEALTH_POSTGRES_MAIN_HOST=pg.svc` |
| `DEPHEALTH_<NAME>_PORT` | Port | `DEPHEALTH_POSTGRES_MAIN_PORT=5432` |
| `DEPHEALTH_<NAME>_TYPE` | Dependency type | `DEPHEALTH_POSTGRES_MAIN_TYPE=postgres` |
| `DEPHEALTH_<NAME>_CHECK_INTERVAL` | Check interval (seconds) | `DEPHEALTH_POSTGRES_MAIN_CHECK_INTERVAL=30` |
| `DEPHEALTH_<NAME>_TIMEOUT` | Timeout (seconds) | `DEPHEALTH_POSTGRES_MAIN_TIMEOUT=10` |
| `DEPHEALTH_<NAME>_CRITICAL` | Criticality (`yes` / `no`) | `DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes` |
| `DEPHEALTH_<NAME>_HEALTH_PATH` | HTTP health path | `DEPHEALTH_PAYMENT_SERVICE_HEALTH_PATH=/ready` |
| `DEPHEALTH_<NAME>_BEARER_TOKEN` | Bearer token (HTTP/gRPC) | `DEPHEALTH_PAYMENT_SERVICE_BEARER_TOKEN=eyJhbG...` |
| `DEPHEALTH_<NAME>_BASIC_USERNAME` | Basic Auth username (HTTP/gRPC) | `DEPHEALTH_PAYMENT_SERVICE_BASIC_USERNAME=admin` |
| `DEPHEALTH_<NAME>_BASIC_PASSWORD` | Basic Auth password (HTTP/gRPC) | `DEPHEALTH_PAYMENT_SERVICE_BASIC_PASSWORD=secret` |
| `DEPHEALTH_<NAME>_LDAP_CHECK_METHOD` | LDAP check method | `DEPHEALTH_LDAP_MAIN_LDAP_CHECK_METHOD=root_dse` |
| `DEPHEALTH_<NAME>_LDAP_BIND_DN` | LDAP Bind DN | `DEPHEALTH_LDAP_MAIN_LDAP_BIND_DN=cn=admin,dc=example,dc=com` |
| `DEPHEALTH_<NAME>_LDAP_BIND_PASSWORD` | LDAP Bind password | `DEPHEALTH_LDAP_MAIN_LDAP_BIND_PASSWORD=secret` |
| `DEPHEALTH_<NAME>_LDAP_BASE_DN` | LDAP Base DN | `DEPHEALTH_LDAP_MAIN_LDAP_BASE_DN=dc=example,dc=com` |
| `DEPHEALTH_<NAME>_LDAP_SEARCH_FILTER` | LDAP search filter | `DEPHEALTH_LDAP_MAIN_LDAP_SEARCH_FILTER=(objectClass=*)` |
| `DEPHEALTH_<NAME>_LDAP_SEARCH_SCOPE` | LDAP search scope | `DEPHEALTH_LDAP_MAIN_LDAP_SEARCH_SCOPE=base` |
| `DEPHEALTH_<NAME>_LDAP_START_TLS` | LDAP StartTLS (`yes` / `no`) | `DEPHEALTH_LDAP_MAIN_LDAP_START_TLS=yes` |

#### Custom Labels

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_<NAME>_LABEL_<KEY>` | Custom label | `DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary` |

`<KEY>` — label name in UPPER_SNAKE_CASE, converted to lower_snake_case
in the metric.

### 8.3. Priority

When values conflict:

1. Programmatic API (highest priority).
2. Environment variables.
3. Default values (lowest priority).

### 8.4. Automatic Discovery

SDK does **not** scan environment variables automatically.
To load configuration from env vars, developer calls:

```go
dephealth.FromEnv("POSTGRES_MAIN") // searches for DEPHEALTH_POSTGRES_MAIN_*
```

This creates a Source, analogous to `FromURL` or `FromParams`,
but reads values from environment variables.

**Rationale**: implicit scanning of env vars creates security issues
and complicates debugging. Explicit `FromEnv` call makes configuration transparent.

---

## 9. Handling Special Cases

### 9.1. IPv6 Addresses

| Input | Extracted host | port |
| --- | --- | --- |
| `postgres://[::1]:5432/db` | `::1` | `5432` |
| `postgres://[2001:db8::1]:5432/db` | `2001:db8::1` | `5432` |
| `Host=[::1];Port=5432` | `::1` | `5432` |
| `FromParams("::1", "5432")` | `::1` | `5432` |

In the metric's `host` label, IPv6 is written **without** square brackets.

### 9.2. URL Without User/Password

```text
postgres://pg.svc:5432/orders
```

Valid URL. Credentials are not required for parsing host/port.
In autonomous mode, credentials for checking can be passed separately.

### 9.3. URL with Empty Path

```text
redis://redis.svc:6379
redis://redis.svc:6379/
```

Both are valid. For Redis, empty path is equivalent to database `0`.

### 9.4. Empty or null URL

Configuration error: `missing URL for dependency "..."`.
