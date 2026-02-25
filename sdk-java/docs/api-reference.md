*[Русская версия](api-reference.ru.md)*

# API Reference

Complete reference of all public classes, interfaces, and methods in the
Java SDK.

## Package Structure

```text
biz.kryukov.dev.dephealth           -- Core API
biz.kryukov.dev.dephealth.model     -- Model classes (Dependency, Endpoint, etc.)
biz.kryukov.dev.dephealth.checks    -- HealthChecker implementations
biz.kryukov.dev.dephealth.metrics   -- MetricsExporter
biz.kryukov.dev.dephealth.scheduler -- CheckScheduler
biz.kryukov.dev.dephealth.parser    -- ConfigParser
biz.kryukov.dev.dephealth.spring    -- Spring Boot integration
```

---

## DepHealth

**Package:** `biz.kryukov.dev.dephealth`

Main entry point class. Combines metrics export and check scheduling.

| Method | Signature | Description |
| --- | --- | --- |
| `builder` | `static Builder builder(String name, String group, MeterRegistry registry)` | Create a new builder |
| `start` | `void start()` | Start periodic health checks |
| `stop` | `void stop()` | Stop all checks and clean up |
| `health` | `Map<String, Boolean> health()` | Quick health map (key: `dep/host:port`) |
| `healthDetails` | `Map<String, EndpointStatus> healthDetails()` | Detailed status per endpoint |
| `addEndpoint` | `void addEndpoint(String depName, DependencyType depType, boolean critical, Endpoint ep, HealthChecker checker)` | Add endpoint at runtime |
| `removeEndpoint` | `void removeEndpoint(String depName, String host, String port)` | Remove endpoint at runtime |
| `updateEndpoint` | `void updateEndpoint(String depName, String oldHost, String oldPort, Endpoint newEp, HealthChecker checker)` | Replace endpoint atomically |

### DepHealth.Builder

Builder for `DepHealth` instances. Created via `DepHealth.builder()`.

#### Global Options

| Method | Signature | Description |
| --- | --- | --- |
| `checkInterval` | `Builder checkInterval(Duration interval)` | Global check interval (default 15s) |
| `timeout` | `Builder timeout(Duration timeout)` | Global check timeout (default 5s) |
| `dependency` | `Builder dependency(String name, DependencyType type, Consumer<DependencyBuilder> config)` | Add dependency with full configuration |
| `build` | `DepHealth build()` | Validate configuration and create instance |

#### Shortcut Methods

Convenience methods that create a dependency with a single endpoint
parsed from the URL.

| Method | Signature | Description |
| --- | --- | --- |
| `postgres` | `Builder postgres(String name, String url, boolean critical)` | Add PostgreSQL dependency |
| `redis` | `Builder redis(String name, String url, boolean critical)` | Add Redis dependency |
| `http` | `Builder http(String name, String url, boolean critical)` | Add HTTP dependency (default checker) |
| `http` | `Builder http(String name, String url, boolean critical, HealthChecker checker)` | Add HTTP dependency (custom checker) |
| `grpc` | `Builder grpc(String name, String host, int port, boolean critical)` | Add gRPC dependency (default checker) |
| `grpc` | `Builder grpc(String name, String host, int port, boolean critical, HealthChecker checker)` | Add gRPC dependency (custom checker) |

### DepHealth.DependencyBuilder

Per-dependency configuration builder. Received via the `Consumer` callback
in `Builder.dependency()`.

#### Connection

| Method | Signature | Description |
| --- | --- | --- |
| `url` | `DependencyBuilder url(String url)` | Parse host/port from URL |
| `jdbcUrl` | `DependencyBuilder jdbcUrl(String jdbcUrl)` | Parse host/port from JDBC URL |
| `host` | `DependencyBuilder host(String host)` | Set host explicitly |
| `port` | `DependencyBuilder port(String port)` | Set port as string |
| `port` | `DependencyBuilder port(int port)` | Set port as integer |

#### General

| Method | Signature | Description |
| --- | --- | --- |
| `critical` | `DependencyBuilder critical(boolean critical)` | Mark as critical dependency |
| `label` | `DependencyBuilder label(String key, String value)` | Add custom Prometheus label |
| `interval` | `DependencyBuilder interval(Duration interval)` | Per-dependency check interval |
| `timeout` | `DependencyBuilder timeout(Duration timeout)` | Per-dependency check timeout |

#### HTTP Options

| Method | Signature | Description |
| --- | --- | --- |
| `httpHealthPath` | `DependencyBuilder httpHealthPath(String path)` | Health check path (default `/health`) |
| `httpTls` | `DependencyBuilder httpTls(boolean enabled)` | Enable HTTPS (auto for `https://`) |
| `httpTlsSkipVerify` | `DependencyBuilder httpTlsSkipVerify(boolean skip)` | Skip TLS certificate verification |
| `httpHeaders` | `DependencyBuilder httpHeaders(Map<String, String> headers)` | Custom HTTP headers |
| `httpBearerToken` | `DependencyBuilder httpBearerToken(String token)` | Bearer token authentication |
| `httpBasicAuth` | `DependencyBuilder httpBasicAuth(String username, String password)` | Basic authentication |

#### gRPC Options

| Method | Signature | Description |
| --- | --- | --- |
| `grpcServiceName` | `DependencyBuilder grpcServiceName(String name)` | Service name (empty = server health) |
| `grpcTls` | `DependencyBuilder grpcTls(boolean enabled)` | Enable TLS |
| `grpcTlsSkipVerify` | `DependencyBuilder grpcTlsSkipVerify(boolean skip)` | Skip TLS certificate verification |
| `grpcMetadata` | `DependencyBuilder grpcMetadata(Map<String, String> metadata)` | Custom gRPC metadata |
| `grpcBearerToken` | `DependencyBuilder grpcBearerToken(String token)` | Bearer token authentication |
| `grpcBasicAuth` | `DependencyBuilder grpcBasicAuth(String username, String password)` | Basic authentication |

#### Database Options

| Method | Signature | Description |
| --- | --- | --- |
| `dbUsername` | `DependencyBuilder dbUsername(String username)` | Database username |
| `dbPassword` | `DependencyBuilder dbPassword(String password)` | Database password |
| `dbDatabase` | `DependencyBuilder dbDatabase(String database)` | Database name |
| `dbQuery` | `DependencyBuilder dbQuery(String query)` | Health check query (default `SELECT 1`) |
| `dataSource` | `DependencyBuilder dataSource(DataSource ds)` | Use existing connection pool |

#### Redis Options

| Method | Signature | Description |
| --- | --- | --- |
| `redisPassword` | `DependencyBuilder redisPassword(String password)` | Password (standalone mode) |
| `redisDb` | `DependencyBuilder redisDb(int db)` | Database number (standalone mode) |
| `jedisPool` | `DependencyBuilder jedisPool(JedisPool pool)` | Use existing Jedis pool |

#### AMQP Options

| Method | Signature | Description |
| --- | --- | --- |
| `amqpUrl` | `DependencyBuilder amqpUrl(String url)` | Full AMQP URL |
| `amqpUsername` | `DependencyBuilder amqpUsername(String username)` | AMQP username |
| `amqpPassword` | `DependencyBuilder amqpPassword(String password)` | AMQP password |
| `amqpVirtualHost` | `DependencyBuilder amqpVirtualHost(String vhost)` | AMQP virtual host |

#### LDAP Options

| Method | Signature | Description |
| --- | --- | --- |
| `ldapCheckMethod` | `DependencyBuilder ldapCheckMethod(String method)` | Check method: `anonymous_bind`, `simple_bind`, `root_dse`, `search` |
| `ldapBindDN` | `DependencyBuilder ldapBindDN(String dn)` | DN for simple bind |
| `ldapBindPassword` | `DependencyBuilder ldapBindPassword(String password)` | Password for simple bind |
| `ldapBaseDN` | `DependencyBuilder ldapBaseDN(String baseDN)` | Base DN for search method |
| `ldapSearchFilter` | `DependencyBuilder ldapSearchFilter(String filter)` | LDAP search filter (default `(objectClass=*)`) |
| `ldapSearchScope` | `DependencyBuilder ldapSearchScope(String scope)` | Search scope: `base`, `one`, `sub` |
| `ldapStartTLS` | `DependencyBuilder ldapStartTLS(boolean enabled)` | Use StartTLS (only with `ldap://`) |
| `ldapTlsSkipVerify` | `DependencyBuilder ldapTlsSkipVerify(boolean skip)` | Skip TLS certificate verification |
| `ldapConnection` | `DependencyBuilder ldapConnection(LDAPConnection conn)` | Use existing LDAP connection (pool mode) |

---

## Model Classes

**Package:** `biz.kryukov.dev.dephealth.model`

### DependencyType

```java
public enum DependencyType {
    HTTP, GRPC, TCP, POSTGRES, MYSQL, REDIS, AMQP, KAFKA, LDAP
}
```

| Value | `label()` |
| --- | --- |
| `HTTP` | `"http"` |
| `GRPC` | `"grpc"` |
| `TCP` | `"tcp"` |
| `POSTGRES` | `"postgres"` |
| `MYSQL` | `"mysql"` |
| `REDIS` | `"redis"` |
| `AMQP` | `"amqp"` |
| `KAFKA` | `"kafka"` |
| `LDAP` | `"ldap"` |

| Method | Signature | Description |
| --- | --- | --- |
| `label` | `String label()` | Lowercase string representation |
| `fromLabel` | `static DependencyType fromLabel(String label)` | Parse from lowercase string |

### Dependency

```java
public final class Dependency { /* ... */ }
```

Immutable representation of a monitored dependency. Contains name, type,
critical flag, list of endpoints, and check configuration.

### Endpoint

```java
public final class Endpoint {
    public Endpoint(String host, String port)
    public Endpoint(String host, String port, Map<String, String> labels)
}
```

Network endpoint for a dependency.

| Method | Signature | Description |
| --- | --- | --- |
| `host` | `String host()` | Hostname or IP address |
| `port` | `String port()` | Port as string |
| `portAsInt` | `int portAsInt()` | Port as integer |
| `labels` | `Map<String, String> labels()` | Custom Prometheus labels |

### EndpointStatus

```java
public final class EndpointStatus { /* ... */ }
```

Detailed health check state for a single endpoint.

| Method | Signature | Description |
| --- | --- | --- |
| `isHealthy` | `Boolean isHealthy()` | `null` before first check, `true`/`false` after |
| `getStatus` | `String getStatus()` | Status category string |
| `getDetail` | `String getDetail()` | Detail string (e.g. `http_503`, `grpc_not_serving`) |
| `getLatency` | `Duration getLatency()` | Check latency |
| `getLatencyMillis` | `double getLatencyMillis()` | Latency in milliseconds |
| `getType` | `DependencyType getType()` | Dependency type |
| `getName` | `String getName()` | Dependency name |
| `getHost` | `String getHost()` | Endpoint host |
| `getPort` | `String getPort()` | Endpoint port |
| `isCritical` | `boolean isCritical()` | Whether the dependency is critical |
| `getLastCheckedAt` | `Instant getLastCheckedAt()` | Timestamp of last check (`null` before first check) |
| `getLabels` | `Map<String, String> getLabels()` | Custom labels |

### CheckConfig

```java
public final class CheckConfig { /* ... */ }
```

Configuration for check scheduling.

| Field | Type | Default |
| --- | --- | --- |
| `interval` | `Duration` | 15s |
| `timeout` | `Duration` | 5s |
| `initialDelay` | `Duration` | 5s |
| `failureThreshold` | `int` | 1 |
| `successThreshold` | `int` | 1 |

### CheckResult

```java
public class CheckResult {
    public String category()
    public String detail()
}
```

Classification of a health check outcome.

| Field | Description |
| --- | --- |
| `OK` | Static constant for a successful result |

---

## StatusCategory

**Package:** `biz.kryukov.dev.dephealth.model`

Constants class defining status categories.

| Constant | Value | Description |
| --- | --- | --- |
| `OK` | `"ok"` | Healthy |
| `TIMEOUT` | `"timeout"` | Check timed out |
| `CONNECTION_ERROR` | `"connection_error"` | Connection refused or reset |
| `DNS_ERROR` | `"dns_error"` | DNS resolution failed |
| `AUTH_ERROR` | `"auth_error"` | Authentication/authorization failure |
| `TLS_ERROR` | `"tls_error"` | TLS handshake failure |
| `UNHEALTHY` | `"unhealthy"` | Reachable but unhealthy |
| `ERROR` | `"error"` | Other error |
| `UNKNOWN` | `"unknown"` | Not yet checked |

---

## HealthChecker

**Package:** `biz.kryukov.dev.dephealth.checks`

```java
public interface HealthChecker {
    void check(Endpoint endpoint, Duration timeout) throws Exception;
    DependencyType type();
}
```

Interface for dependency health checks. `check()` completes normally if
the dependency is healthy, or throws an exception describing the failure.
`type()` returns the dependency type.

### Health Checker Implementations

| Class | Type | Builder Shortcut |
| --- | --- | --- |
| `HttpHealthChecker` | `HTTP` | `http(...)` |
| `GrpcHealthChecker` | `GRPC` | `grpc(...)` |
| `TcpHealthChecker` | `TCP` | -- |
| `PostgresHealthChecker` | `POSTGRES` | `postgres(...)` |
| `MysqlHealthChecker` | `MYSQL` | -- |
| `RedisHealthChecker` | `REDIS` | `redis(...)` |
| `AmqpHealthChecker` | `AMQP` | -- |
| `KafkaHealthChecker` | `KAFKA` | -- |
| `LdapHealthChecker` | `LDAP` | -- |

---

## Exception Hierarchy

**Package:** `biz.kryukov.dev.dephealth`

```text
Exception
  └── CheckException                  -- base for all check exceptions
        ├── CheckAuthException        -- authentication/authorization failure
        ├── CheckConnectionException  -- connection refused/reset/timeout
        ├── UnhealthyException        -- reachable but unhealthy
        ├── ValidationException       -- invalid configuration
        ├── ConfigurationException    -- missing or invalid settings
        ├── EndpointNotFoundException -- endpoint not found for update/remove
        └── DepHealthException        -- general SDK error
```

### CheckException

Base exception for health check failures.

| Method | Signature | Description |
| --- | --- | --- |
| `statusCategory` | `String statusCategory()` | Status category (e.g. `"auth_error"`) |
| `statusDetail` | `String statusDetail()` | Detail string (e.g. `"http_503"`) |

---

## ConfigParser

**Package:** `biz.kryukov.dev.dephealth.parser`

Static utility class for parsing connection URLs and parameters.

| Method | Signature | Description |
| --- | --- | --- |
| `parseUrl` | `static List<ParsedConnection> parseUrl(String url)` | Parse URL into host/port/type list |
| `parseJdbc` | `static List<ParsedConnection> parseJdbc(String jdbcUrl)` | Parse JDBC URL |
| `parseParams` | `static ParsedConnection parseParams(String host, String port)` | Create from explicit host and port |

Supported URL schemes: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`,
`ldap`, `ldaps`. Kafka multi-broker URLs
(`kafka://host1:9092,host2:9092`) return multiple connections.

### ParsedConnection

```java
public final class ParsedConnection {
    public String host()
    public String port()
    public DependencyType type()
}
```

Result of parsing a URL or connection string.

---

## MetricsExporter

**Package:** `biz.kryukov.dev.dephealth.metrics`

Creates Prometheus metrics (`app_dependency_health` gauge and
`app_dependency_latency_seconds` histogram) using the provided
`MeterRegistry`. This class is internal but requires a `MeterRegistry`
to be supplied via `DepHealth.builder()`.

---

## Spring Boot Integration

**Package:** `biz.kryukov.dev.dephealth.spring`

### DepHealthAutoConfiguration

Auto-configuration class. Activated when `dephealth-spring-boot-starter`
is on the classpath. Reads `DepHealthProperties` and creates a `DepHealth`
bean.

### DepHealthProperties

```java
@ConfigurationProperties(prefix = "dephealth")
public class DepHealthProperties { /* ... */ }
```

Maps `application.yml` / `application.properties` configuration under
the `dephealth.*` prefix.

### DepHealthLifecycle

Implements `SmartLifecycle`. Calls `DepHealth.start()` on application
startup and `DepHealth.stop()` on shutdown.

### DepHealthIndicator

Implements Spring Boot `HealthIndicator`. Exposes health check results
via `/actuator/health`.

### DependenciesEndpoint

```java
@Endpoint(id = "dependencies")
public class DependenciesEndpoint { /* ... */ }
```

Custom Actuator endpoint at `/actuator/dependencies`. Returns detailed
health status for all monitored dependencies.

---

## Dynamic Endpoint Management

Methods for adding, removing, and updating endpoints at runtime on a
running `DepHealth` instance. All methods are thread-safe.

### addEndpoint

```java
public void addEndpoint(String depName, DependencyType depType,
    boolean critical, Endpoint ep, HealthChecker checker)
```

Adds a new endpoint to a running `DepHealth` instance. A health check
task starts immediately using the global check interval and timeout.

**Idempotent:** if an endpoint with the same `depName:host:port` key
already exists, the call is a no-op.

### removeEndpoint

```java
public void removeEndpoint(String depName, String host, String port)
```

Removes an endpoint from a running `DepHealth` instance. Cancels the
health check task and deletes all associated Prometheus metrics.

**Idempotent:** if no endpoint with the given key exists, the call is
a no-op.

### updateEndpoint

```java
public void updateEndpoint(String depName, String oldHost, String oldPort,
    Endpoint newEp, HealthChecker checker)
```

Atomically replaces an existing endpoint with a new one. The old
endpoint's task is cancelled and its metrics are deleted; a new task is
started for the new endpoint.

**Errors:**

| Condition | Exception |
| --- | --- |
| Old endpoint not found | `EndpointNotFoundException` |
| Missing new host or port | `ValidationException` |

---

## See Also

- [Getting Started](getting-started.md) -- installation and first example
