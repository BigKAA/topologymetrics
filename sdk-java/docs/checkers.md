*[Русская версия](checkers.ru.md)*

# Health Checkers

The Java SDK includes 9 built-in health checkers for common dependency types.
Each checker implements the `HealthChecker` interface and can be used via
the high-level API (`DepHealth.builder().dependency(...)`) or directly via
its builder.

## HTTP

Checks HTTP endpoints by sending a GET request and expecting a 2xx response.

### Registration

```java
.dependency("api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true))
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `.httpHealthPath(path)` | `/health` | Path for the health check endpoint |
| `.httpTls(enabled)` | `false` | Use HTTPS instead of HTTP (auto-detected from `https://` URLs) |
| `.httpTlsSkipVerify(skip)` | `false` | Skip TLS certificate verification |
| `.httpHeaders(headers)` | -- | Custom HTTP headers (`Map<String, String>`) |
| `.httpBearerToken(token)` | -- | Set `Authorization: Bearer <token>` header |
| `.httpBasicAuth(user, pass)` | -- | Set `Authorization: Basic <base64>` header |

### Full Example

```java
import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.DependencyType;
import io.micrometer.prometheus.PrometheusMeterRegistry;

import java.util.Map;

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("payment-api", DependencyType.HTTP, d -> d
        .url("https://payment.svc:443")
        .critical(true)
        .httpHealthPath("/healthz")
        .httpTls(true)
        .httpTlsSkipVerify(true)
        .httpHeaders(Map.of("X-Request-Source", "dephealth")))
    .build();

dh.start();
// ...
dh.stop();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Response 2xx | `ok` | `ok` |
| Response 401 or 403 | `auth_error` | `auth_error` |
| Response other non-2xx | `unhealthy` | `http_<code>` (e.g., `http_500`) |
| Timeout | `timeout` | `timeout` |
| Connection refused | `connection_error` | `connection_error` |
| DNS resolution failed | `dns_error` | `dns_error` |
| TLS handshake failed | `tls_error` | `tls_error` |
| Other network error | classified by core | depends on error type |

### Direct Checker Usage

```java
import biz.kryukov.dev.dephealth.checks.HttpHealthChecker;

HttpHealthChecker checker = HttpHealthChecker.builder()
    .healthPath("/healthz")
    .tlsEnabled(true)
    .tlsSkipVerify(false)
    .build();
```

### Behavior Notes

- Uses `java.net.http.HttpClient` with `NORMAL` redirect policy (follows 3xx)
- Creates a new HTTP client for each check
- Sends `User-Agent: dephealth/0.5.0` header
- Custom headers are applied after User-Agent and can override it
- Only one auth method is allowed: `bearerToken`, `basicAuth`, or an
  `Authorization` key in custom headers

---

## gRPC

Checks gRPC services using the
[gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

### Registration

```java
.dependency("user-service", DependencyType.GRPC, d -> d
    .host("user.svc")
    .port("9090")
    .critical(true))
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `.grpcServiceName(name)` | `""` (empty) | Service name to check; empty checks overall server health |
| `.grpcTls(enabled)` | `false` | Enable TLS |
| `.grpcMetadata(md)` | -- | Custom gRPC metadata (`Map<String, String>`) |
| `.grpcBearerToken(token)` | -- | Set `authorization: Bearer <token>` metadata |
| `.grpcBasicAuth(user, pass)` | -- | Set `authorization: Basic <base64>` metadata |

### Full Example

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Check a specific gRPC service
    .dependency("user-service", DependencyType.GRPC, d -> d
        .host("user.svc")
        .port("9090")
        .critical(true)
        .grpcServiceName("user.v1.UserService")
        .grpcTls(true)
        .grpcMetadata(Map.of("x-request-id", "dephealth")))

    // Check overall server health (empty service name)
    .dependency("grpc-gateway", DependencyType.GRPC, d -> d
        .host("gateway.svc")
        .port("9090")
        .critical(false))
    .build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Response SERVING | `ok` | `ok` |
| gRPC UNAUTHENTICATED | `auth_error` | `auth_error` |
| gRPC PERMISSION_DENIED | `auth_error` | `auth_error` |
| Response NOT_SERVING | `unhealthy` | `grpc_not_serving` |
| Other gRPC status | `unhealthy` | `grpc_unknown` |
| Dial/RPC error | classified by core | depends on error type |

### Direct Checker Usage

```java
import biz.kryukov.dev.dephealth.checks.GrpcHealthChecker;

GrpcHealthChecker checker = GrpcHealthChecker.builder()
    .serviceName("user.v1.UserService")
    .tlsEnabled(true)
    .build();
```

### Behavior Notes

- Uses `passthrough:///` resolver (default for `ManagedChannelBuilder.forTarget()`)
  to avoid DNS SRV lookups; important in Kubernetes where `ndots:5` causes
  high latency with `dns:///` resolver
- Creates a new gRPC channel for each check; channel is shut down immediately
  after the call
- Empty service name checks overall server health
- Only one auth method is allowed: `bearerToken`, `basicAuth`, or an
  `authorization` key in custom metadata

---

## TCP

Checks TCP connectivity by establishing a socket connection and closing it
immediately. The simplest checker -- no application-level protocol involved.

### Registration

```java
.dependency("memcached", DependencyType.TCP, d -> d
    .host("memcached.svc")
    .port("11211")
    .critical(false))
```

Or using a URL:

```java
.dependency("memcached", DependencyType.TCP, d -> d
    .url("tcp://memcached.svc:11211")
    .critical(false))
```

### Options

No checker-specific options. TCP checker is stateless.

### Full Example

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("memcached", DependencyType.TCP, d -> d
        .host("memcached.svc")
        .port("11211")
        .critical(false))
    .dependency("custom-service", DependencyType.TCP, d -> d
        .host("custom.svc")
        .port("5555")
        .critical(true))
    .build();
```

### Error Classification

TCP checker does not produce checker-specific errors. All failures (connection
refused, DNS errors, timeouts) are classified by the core error classifier.

### Direct Checker Usage

```java
import biz.kryukov.dev.dephealth.checks.TcpHealthChecker;

TcpHealthChecker checker = new TcpHealthChecker();
```

### Behavior Notes

- Only performs TCP handshake (SYN/ACK) -- no data is sent or received
- Connection is closed immediately after establishment
- Uses `java.net.Socket` with the configured timeout
- Useful for services that don't have a health check protocol

---

## PostgreSQL

Checks PostgreSQL by executing a query (default: `SELECT 1`). Supports
both standalone mode (new JDBC connection) and pool mode (existing
`DataSource`).

### Registration

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://user:pass@pg.svc:5432/mydb")
    .critical(true))
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `.dbUsername(user)` | -- | Database username (also extracted from URL) |
| `.dbPassword(pass)` | -- | Database password (also extracted from URL) |
| `.dbDatabase(name)` | -- | Database name (also extracted from URL path) |
| `.dbQuery(query)` | `SELECT 1` | Custom SQL query for health check |
| `.dataSource(ds)` | -- | Connection pool DataSource (preferred over standalone) |

### Full Example

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Standalone mode -- creates new JDBC connection for each check
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true)
        .dbQuery("SELECT 1"))
    .build();
```

### Pool Mode

Use the existing connection pool for health checks:

```java
import javax.sql.DataSource;

// Assume 'dataSource' is your application's connection pool (HikariCP, etc.)
DataSource dataSource = ...;

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true)
        .dataSource(dataSource))
    .build();
```

Or with a pre-built checker:

```java
import biz.kryukov.dev.dephealth.checks.PostgresHealthChecker;

PostgresHealthChecker checker = PostgresHealthChecker.builder()
    .dataSource(dataSource)
    .build();

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("postgres-main", DependencyType.POSTGRES, checker, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true))
    .build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Query succeeds | `ok` | `ok` |
| SQLSTATE 28000 (Invalid Authorization) | `auth_error` | `auth_error` |
| SQLSTATE 28P01 (Authentication Failed) | `auth_error` | `auth_error` |
| "password authentication failed" in error | `auth_error` | `auth_error` |
| Connection timeout | `timeout` | `timeout` |
| Connection refused | `connection_error` | `connection_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```java
import biz.kryukov.dev.dephealth.checks.PostgresHealthChecker;

// Standalone mode
PostgresHealthChecker checker = PostgresHealthChecker.builder()
    .username("user")
    .password("pass")
    .database("mydb")
    .build();

// Pool mode
PostgresHealthChecker poolChecker = PostgresHealthChecker.builder()
    .dataSource(dataSource)
    .build();
```

### Behavior Notes

- Standalone mode builds JDBC URL `jdbc:postgresql://host:port/database`
- Pool mode reuses the existing connection pool -- reflects the actual
  ability of the service to work with the dependency
- Uses `DriverManager.getConnection()` for standalone mode
- Credentials from URL are extracted automatically if not set explicitly

---

## MySQL

Checks MySQL by executing a query (default: `SELECT 1`). Supports both
standalone mode and pool mode.

### Registration

```java
.dependency("mysql-main", DependencyType.MYSQL, d -> d
    .url("mysql://user:pass@mysql.svc:3306/mydb")
    .critical(true))
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `.dbUsername(user)` | -- | Database username (also extracted from URL) |
| `.dbPassword(pass)` | -- | Database password (also extracted from URL) |
| `.dbDatabase(name)` | -- | Database name (also extracted from URL path) |
| `.dbQuery(query)` | `SELECT 1` | Custom SQL query for health check |
| `.dataSource(ds)` | -- | Connection pool DataSource (preferred over standalone) |

### Full Example

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("mysql-main", DependencyType.MYSQL, d -> d
        .url("mysql://user:pass@mysql.svc:3306/mydb")
        .critical(true))
    .build();
```

### Pool Mode

```java
import javax.sql.DataSource;

DataSource dataSource = ...;

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("mysql-main", DependencyType.MYSQL, d -> d
        .url("mysql://mysql.svc:3306/mydb")
        .critical(true)
        .dataSource(dataSource))
    .build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Query succeeds | `ok` | `ok` |
| MySQL error 1045 (Access Denied) | `auth_error` | `auth_error` |
| "Access denied" in error message | `auth_error` | `auth_error` |
| Connection timeout | `timeout` | `timeout` |
| Connection refused | `connection_error` | `connection_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```java
import biz.kryukov.dev.dephealth.checks.MysqlHealthChecker;

// Standalone mode
MysqlHealthChecker checker = MysqlHealthChecker.builder()
    .username("user")
    .password("pass")
    .database("mydb")
    .build();

// Pool mode
MysqlHealthChecker poolChecker = MysqlHealthChecker.builder()
    .dataSource(dataSource)
    .build();
```

### Behavior Notes

- Standalone mode builds JDBC URL `jdbc:mysql://host:port/database`
- Uses `DriverManager.getConnection()` for standalone mode
- Same interface as PostgreSQL checker (both use `DataSource` for pool mode)
- Credentials from URL are extracted automatically if not set explicitly

---

## Redis

Checks Redis by executing the `PING` command and expecting `PONG`. Supports
both standalone mode and pool mode.

### Registration

```java
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://:password@redis.svc:6379/0")
    .critical(false))
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `.redisPassword(password)` | -- | Password for standalone mode (also extracted from URL) |
| `.redisDb(db)` | `0` | Database index for standalone mode (also extracted from URL path) |
| `.jedisPool(pool)` | -- | JedisPool for connection pool integration (preferred) |

### Full Example

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Password from URL
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .url("redis://:mypassword@redis.svc:6379/0")
        .critical(false))

    // Password via option
    .dependency("redis-sessions", DependencyType.REDIS, d -> d
        .host("redis-sessions.svc")
        .port("6379")
        .redisPassword("secret")
        .redisDb(1)
        .critical(true))
    .build();
```

### Pool Mode

```java
import redis.clients.jedis.JedisPool;

JedisPool jedisPool = new JedisPool("redis.svc", 6379);

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .host("redis.svc")
        .port("6379")
        .critical(false)
        .jedisPool(jedisPool))
    .build();
```

Or with a pre-built checker:

```java
import biz.kryukov.dev.dephealth.checks.RedisHealthChecker;

RedisHealthChecker checker = RedisHealthChecker.builder()
    .jedisPool(jedisPool)
    .build();

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("redis-cache", DependencyType.REDIS, checker, d -> d
        .host("redis.svc")
        .port("6379")
        .critical(false))
    .build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| PING returns PONG | `ok` | `ok` |
| "NOAUTH" in error | `auth_error` | `auth_error` |
| "WRONGPASS" in error | `auth_error` | `auth_error` |
| Connection refused | `connection_error` | `connection_error` |
| Connection timeout | `connection_error` | `connection_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```java
import biz.kryukov.dev.dephealth.checks.RedisHealthChecker;

// Standalone mode
RedisHealthChecker checker = RedisHealthChecker.builder()
    .password("pass")
    .database(0)
    .build();

// Pool mode
RedisHealthChecker poolChecker = RedisHealthChecker.builder()
    .jedisPool(jedisPool)
    .build();
```

### Behavior Notes

- Uses Jedis library for Redis communication
- Standalone mode creates a new `Jedis` connection with the configured timeout
- Pool mode uses `jedisPool.getResource()` and closes the resource after check
- Password from options takes precedence over password from URL
- Database index from options takes precedence over index from URL

---

## AMQP (RabbitMQ)

Checks AMQP brokers by establishing a connection and closing it
immediately. Only standalone mode is supported.

### Registration

```java
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .host("rabbitmq.svc")
    .port("5672")
    .critical(false))
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `.amqpUrl(url)` | -- | Full AMQP URL (overrides host/port/credentials) |
| `.amqpUsername(user)` | -- | AMQP username (also extracted from URL) |
| `.amqpPassword(pass)` | -- | AMQP password (also extracted from URL) |
| `.amqpVirtualHost(vhost)` | -- | AMQP virtual host (also extracted from URL path) |

### Full Example

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Default credentials (guest:guest)
    .dependency("rabbitmq", DependencyType.AMQP, d -> d
        .host("rabbitmq.svc")
        .port("5672")
        .critical(false))

    // Custom credentials
    .dependency("rabbitmq-prod", DependencyType.AMQP, d -> d
        .host("rmq-prod.svc")
        .port("5672")
        .amqpUrl("amqp://myuser:mypass@rmq-prod.svc:5672/myvhost")
        .critical(true))
    .build();
```

Or using credentials from URL:

```java
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .url("amqp://user:pass@rabbitmq.svc:5672/myvhost")
    .critical(true))
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Connection established and open | `ok` | `ok` |
| "403" in error | `auth_error` | `auth_error` |
| "ACCESS_REFUSED" in error | `auth_error` | `auth_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```java
import biz.kryukov.dev.dephealth.checks.AmqpHealthChecker;

AmqpHealthChecker checker = AmqpHealthChecker.builder()
    .username("myuser")
    .password("mypass")
    .virtualHost("/myvhost")
    .build();
```

### Behavior Notes

- Uses RabbitMQ `ConnectionFactory` from `com.rabbitmq.client`
- No pool mode -- always creates a new connection
- Connection is closed immediately after successful establishment
- When `amqpUrl` is set, it overrides host/port/credentials from the
  `ConnectionFactory`
- Connection name is set to `"dephealth-check"` for easy identification in
  the RabbitMQ management console

---

## Kafka

Checks Kafka brokers by connecting and requesting cluster metadata via
`AdminClient.describeCluster().nodes()`. Stateless checker with no
configuration options.

### Registration

```java
.dependency("kafka", DependencyType.KAFKA, d -> d
    .host("kafka.svc")
    .port("9092")
    .critical(true))
```

### Options

No checker-specific options. Kafka checker is stateless.

### Full Example

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("kafka", DependencyType.KAFKA, d -> d
        .host("kafka.svc")
        .port("9092")
        .critical(true))
    .build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Metadata returns nodes | `ok` | `ok` |
| No nodes in metadata | `unhealthy` | `no_brokers` |
| Connection/metadata error | classified by core | depends on error type |

### Direct Checker Usage

```java
import biz.kryukov.dev.dephealth.checks.KafkaHealthChecker;

KafkaHealthChecker checker = new KafkaHealthChecker();
```

### Behavior Notes

- Uses Apache Kafka `AdminClient` from `org.apache.kafka.clients.admin`
- Creates a new `AdminClient`, requests cluster metadata, closes the client
- Verifies that at least one node is present in the `describeCluster` response
- `REQUEST_TIMEOUT_MS` and `DEFAULT_API_TIMEOUT_MS` are set to the check
  timeout
- No authentication support (plain TCP only)

---

## LDAP

Checks LDAP directory servers. Supports 4 check methods, 3 connection
protocols, and both standalone and pool modes. Added in v0.8.0.

### Registration

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .critical(true))
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `.ldapCheckMethod(method)` | `ROOT_DSE` | Check method (see below) |
| `.ldapBindDN(dn)` | -- | Bind DN for SIMPLE_BIND or SEARCH authentication |
| `.ldapBindPassword(pass)` | -- | Bind password |
| `.ldapBaseDN(dn)` | -- | Base DN for SEARCH operations (required for SEARCH method) |
| `.ldapSearchFilter(filter)` | `(objectClass=*)` | LDAP search filter |
| `.ldapSearchScope(scope)` | `BASE` | Search scope: `BASE`, `ONE`, or `SUB` |
| `.ldapStartTLS(enabled)` | `false` | Enable StartTLS (incompatible with `ldaps://`) |
| `.ldapTlsSkipVerify(skip)` | `false` | Skip TLS certificate verification |
| `.ldapConnection(conn)` | -- | Existing `LDAPConnection` for pool integration |

### Check Methods

| Method | Description |
| --- | --- |
| `ANONYMOUS_BIND` | Performs an anonymous LDAP bind (empty DN and password) |
| `SIMPLE_BIND` | Performs a bind with `bindDN` and `bindPassword` (both required) |
| `ROOT_DSE` | Queries the Root DSE entry (default; works without authentication) |
| `SEARCH` | Performs an LDAP search with `baseDN`, filter, and scope |

### Connection Protocols

| Protocol | URL scheme | Default port | TLS |
| --- | --- | --- | --- |
| Plain LDAP | `ldap://` | 389 | No |
| LDAPS | `ldaps://` | 636 | Yes (from connection start) |
| StartTLS | `ldap://` + `.ldapStartTLS(true)` | 389 | Upgraded after connection |

### Full Examples

**ROOT_DSE (default)** -- simplest check, no credentials required:

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .critical(true))
```

**ANONYMOUS_BIND** -- test that anonymous access is allowed:

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod(LdapHealthChecker.CheckMethod.ANONYMOUS_BIND)
    .critical(true))
```

**SIMPLE_BIND** -- verify credentials:

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod(LdapHealthChecker.CheckMethod.SIMPLE_BIND)
    .ldapBindDN("cn=admin,dc=example,dc=org")
    .ldapBindPassword("secret")
    .critical(true))
```

**SEARCH** -- perform an authenticated search:

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod(LdapHealthChecker.CheckMethod.SEARCH)
    .ldapBindDN("cn=readonly,dc=example,dc=org")
    .ldapBindPassword("pass")
    .ldapBaseDN("ou=users,dc=example,dc=org")
    .ldapSearchFilter("(uid=healthcheck)")
    .ldapSearchScope(LdapHealthChecker.LdapSearchScope.ONE)
    .critical(true))
```

**LDAPS** -- TLS from connection start:

```java
.dependency("ldap-secure", DependencyType.LDAP, d -> d
    .url("ldaps://ldap.svc:636")
    .ldapTlsSkipVerify(true)
    .critical(true))
```

**StartTLS** -- upgrade plain LDAP connection to TLS:

```java
.dependency("ldap-starttls", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapStartTLS(true)
    .critical(true))
```

### Pool Mode

Use an existing LDAP connection for health checks:

```java
import com.unboundid.ldap.sdk.LDAPConnection;

LDAPConnection ldapConn = new LDAPConnection("ldap.svc", 389);

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("ldap", DependencyType.LDAP, d -> d
        .url("ldap://ldap.svc:389")
        .critical(true)
        .ldapConnection(ldapConn))
    .build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Check succeeds | `ok` | `ok` |
| ResultCode 49 (INVALID_CREDENTIALS) | `auth_error` | `auth_error` |
| ResultCode 50 (INSUFFICIENT_ACCESS_RIGHTS) | `auth_error` | `auth_error` |
| ResultCode 51/52/53 (BUSY/UNAVAILABLE/UNWILLING_TO_PERFORM) | `unhealthy` | `unhealthy` |
| CONNECT_ERROR / SERVER_DOWN | `connection_error` | `connection_error` |
| Connection refused | `connection_error` | `connection_error` |
| TLS / SSL / certificate error | `tls_error` | `tls_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```java
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker;

LdapHealthChecker checker = LdapHealthChecker.builder()
    .checkMethod(LdapHealthChecker.CheckMethod.ROOT_DSE)
    .build();
```

### Config Validation

The builder validates configuration at build time:

- `SIMPLE_BIND` requires both `bindDN` and `bindPassword`
- `SEARCH` requires `baseDN`
- `startTLS(true)` is incompatible with `ldaps://` URL (useTLS)

### Behavior Notes

- Uses UnboundID LDAP SDK (`com.unboundid.ldap.sdk`)
- Standalone mode creates a new LDAP connection for each check
- Pool mode uses the provided `LDAPConnection` (caller manages lifecycle)
- `followReferrals` is disabled for health check connections
- Search operations are limited to 1 result (`setSizeLimit(1)`)
- ROOT_DSE queries `namingContexts` and `subschemaSubentry` attributes

---

## Error Classification Summary

All checkers classify errors into status categories. The core error
classifier handles common error types (timeouts, DNS errors, TLS errors,
connection refused). Checker-specific classification adds protocol-level
detail:

| Status Category | Meaning | Common Causes |
| --- | --- | --- |
| `ok` | Dependency is healthy | Check succeeded |
| `timeout` | Check timed out | Slow network, overloaded service |
| `connection_error` | Cannot connect | Service down, wrong host/port, firewall |
| `dns_error` | DNS resolution failed | Wrong hostname, DNS outage |
| `auth_error` | Authentication failed | Wrong credentials, expired token |
| `tls_error` | TLS handshake failed | Invalid certificate, TLS misconfiguration |
| `unhealthy` | Connected but not healthy | Service reports unhealthy, returns error code |
| `error` | Unexpected error | Unclassified failures |

## See Also

- [Configuration](configuration.md) -- all options, defaults, and environment variables
- [Authentication](authentication.md) -- detailed auth guide for HTTP and gRPC
- [Connection Pools](connection-pools.md) -- pool mode via DataSource and JedisPool
- [Metrics](metrics.md) -- Prometheus metrics reference and PromQL examples
- [Troubleshooting](troubleshooting.md) -- common issues and solutions
