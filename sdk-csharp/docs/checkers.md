*[Русская версия](checkers.ru.md)*

# Health Checkers

The C# SDK includes 9 built-in health checkers for common dependency types.
Each checker implements the `IHealthChecker` interface and can be used via
the high-level builder API (`DepHealthMonitor.CreateBuilder(...).AddHttp(...)`)
or by instantiating the checker class directly and passing it to
`AddCustom(...)` or `AddEndpoint(...)`.

## HTTP

Checks HTTP endpoints by sending a GET request and expecting a 2xx response.

### Registration

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddHttp("api", "http://api.svc:8080", critical: true)
    .Build();
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `healthPath` | `/health` | Path for the health check endpoint |
| `critical` | `null` | Whether the dependency is critical (also via `DEPHEALTH_<NAME>_CRITICAL`) |
| `headers` | `null` | Custom HTTP headers (`Dictionary<string, string>`) |
| `bearerToken` | `null` | Set `Authorization: Bearer <token>` header |
| `basicAuthUsername` | `null` | Username for HTTP Basic authentication |
| `basicAuthPassword` | `null` | Password for HTTP Basic authentication |

TLS is detected automatically from `https://` in the URL.

### Full Example

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddHttp(
        name: "payment-api",
        url: "https://payment.svc:443",
        healthPath: "/healthz",
        critical: true,
        headers: new Dictionary<string, string>
        {
            ["X-Request-Source"] = "dephealth"
        })
    .Build();

dh.Start();
// ...
dh.Stop();
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

```csharp
using DepHealth.Checks;

var checker = new HttpChecker(
    healthPath: "/healthz",
    tlsEnabled: true,
    tlsSkipVerify: false,
    bearerToken: "my-token");

// Use with AddCustom:
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("payment-api", DependencyType.Http, "payment.svc", "443", checker,
        critical: true)
    .Build();
```

### Behavior Notes

- Uses `System.Net.Http.HttpClient` with a fresh instance per check
- Sends `User-Agent: dephealth/0.5.0` header
- Custom headers are applied after User-Agent and can override it
- TLS is enabled when `tlsEnabled: true` or when the URL starts with `https://`
- Only one auth method is allowed: `bearerToken`, `basicAuthUsername`/`basicAuthPassword`,
  or an `Authorization` key in custom headers — mixing them throws `ValidationException`

---

## gRPC

Checks gRPC services using the
[gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

### Registration

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddGrpc("user-service", host: "user.svc", port: "9090", critical: true)
    .Build();
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `tlsEnabled` | `false` | Enable TLS (HTTPS) for the gRPC channel |
| `critical` | `null` | Whether the dependency is critical |
| `metadata` | `null` | Custom gRPC metadata (`Dictionary<string, string>`) |
| `bearerToken` | `null` | Set `authorization: Bearer <token>` metadata |
| `basicAuthUsername` | `null` | Username for Basic authentication metadata |
| `basicAuthPassword` | `null` | Password for Basic authentication metadata |

### Full Example

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Check a specific gRPC service with TLS
    .AddGrpc(
        name: "user-service",
        host: "user.svc",
        port: "9090",
        tlsEnabled: true,
        critical: true,
        metadata: new Dictionary<string, string>
        {
            ["x-request-id"] = "dephealth"
        })

    // Check a service without TLS (plain HTTP/2)
    .AddGrpc("grpc-gateway", host: "gateway.svc", port: "9090", critical: false)
    .Build();
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

```csharp
using DepHealth.Checks;

var checker = new GrpcChecker(
    tlsEnabled: true,
    bearerToken: "my-token");

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("user-service", DependencyType.Grpc, "user.svc", "9090", checker,
        critical: true)
    .Build();
```

### Behavior Notes

- Uses `GrpcChannel.ForAddress()` from `Grpc.Net.Client`
- Creates a new gRPC channel for each check; channel is disposed immediately after the call
- Checks overall server health (empty service name in `HealthCheckRequest`)
- Only one auth method is allowed: `bearerToken`, `basicAuthUsername`/`basicAuthPassword`,
  or an `authorization` key in custom metadata — mixing them throws `ValidationException`

---

## TCP

Checks TCP connectivity by establishing a socket connection and closing it
immediately. The simplest checker — no application-level protocol involved.

### Registration

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddTcp("memcached", host: "memcached.svc", port: "11211", critical: false)
    .Build();
```

### Options

No checker-specific options. TCP checker is stateless.

### Full Example

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddTcp("memcached", host: "memcached.svc", port: "11211", critical: false)
    .AddTcp("custom-service", host: "custom.svc", port: "5555", critical: true)
    .Build();
```

### Error Classification

TCP checker does not produce checker-specific errors. All failures (connection
refused, DNS errors, timeouts) are classified by the core error classifier.

### Direct Checker Usage

```csharp
using DepHealth.Checks;

var checker = new TcpChecker();

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("memcached", DependencyType.Tcp, "memcached.svc", "11211", checker,
        critical: false)
    .Build();
```

### Behavior Notes

- Uses `System.Net.Sockets.TcpClient.ConnectAsync()` — only TCP handshake, no data sent
- Connection is closed immediately after establishment
- Useful for services that do not expose a health check protocol

---

## PostgreSQL

Checks PostgreSQL by executing `SELECT 1`. Supports both standalone mode
(new connection from connection string) and pool mode (existing
`NpgsqlDataSource`).

### Registration

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddPostgres("postgres-main", url: "postgresql://user:pass@pg.svc:5432/mydb",
        critical: true)
    .Build();
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | required | PostgreSQL connection URL; credentials and database extracted automatically |
| `critical` | `null` | Whether the dependency is critical |
| `labels` | `null` | Custom Prometheus labels |

### Full Example

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Standalone mode — connection string built from URL
    .AddPostgres(
        name: "postgres-main",
        url: Environment.GetEnvironmentVariable("DATABASE_URL")!,
        critical: true)
    .Build();
```

### Pool Mode

Use an existing `NpgsqlDataSource` for health checks:

```csharp
using DepHealth;
using DepHealth.Checks;
using Npgsql;

// Build or inject the application's NpgsqlDataSource
NpgsqlDataSource dataSource = NpgsqlDataSource.Create(connectionString);

var checker = new PostgresChecker(dataSource);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("postgres-main", DependencyType.Postgres,
        "pg.svc", "5432", checker, critical: true)
    .Build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Query succeeds | `ok` | `ok` |
| SQL state 28000 (Invalid Authorization) | `auth_error` | `auth_error` |
| SQL state 28P01 (Authentication Failed) | `auth_error` | `auth_error` |
| Connection timeout | `timeout` | `timeout` |
| Connection refused | `connection_error` | `connection_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```csharp
using DepHealth.Checks;
using Npgsql;

// Standalone mode
var checker = new PostgresChecker(
    connectionString: "Host=pg.svc;Port=5432;Username=user;Password=pass;Database=mydb");

// Pool mode
NpgsqlDataSource dataSource = NpgsqlDataSource.Create(connectionString);
var poolChecker = new PostgresChecker(dataSource);
```

### Behavior Notes

- Standalone mode builds a Npgsql connection string from the URL: host, port, username,
  password, and database are extracted automatically
- Pool mode calls `NpgsqlDataSource.OpenConnectionAsync()` — reflects the real
  ability of the application to obtain a connection from its pool
- Uses `Npgsql` library (`Npgsql.NpgsqlConnection`)
- Auth errors are detected via `PostgresException.SqlState`

---

## MySQL

Checks MySQL by executing `SELECT 1`. Only standalone mode is supported;
use `AddCustom` with a pre-built `MySqlChecker` for custom connection strings.

### Registration

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddMySql("mysql-main", url: "mysql://user:pass@mysql.svc:3306/mydb",
        critical: true)
    .Build();
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | required | MySQL connection URL; credentials and database extracted automatically |
| `critical` | `null` | Whether the dependency is critical |
| `labels` | `null` | Custom Prometheus labels |

### Full Example

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddMySql(
        name: "mysql-main",
        url: "mysql://user:pass@mysql.svc:3306/mydb",
        critical: true)
    .Build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Query succeeds | `ok` | `ok` |
| `MySqlErrorCode.AccessDenied` | `auth_error` | `auth_error` |
| Connection timeout | `timeout` | `timeout` |
| Connection refused | `connection_error` | `connection_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```csharp
using DepHealth.Checks;

var checker = new MySqlChecker(
    connectionString: "Server=mysql.svc;Port=3306;User=user;Password=pass;Database=mydb");

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("mysql-main", DependencyType.MySql, "mysql.svc", "3306", checker,
        critical: true)
    .Build();
```

### Behavior Notes

- Uses `MySqlConnector` library (`MySqlConnector.MySqlConnection`)
- Standalone mode builds a connection string from the URL; credentials and database
  are extracted automatically
- Auth errors detected via `MySqlException.ErrorCode == MySqlErrorCode.AccessDenied`
- Same interface as `PostgresChecker` for standalone mode

---

## Redis

Checks Redis by executing the `PING` command and expecting `PONG`. Supports
both standalone mode and pool mode via `IConnectionMultiplexer`.

### Registration

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddRedis("redis-cache", url: "redis://:password@redis.svc:6379/0",
        critical: false)
    .Build();
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | required | Redis connection URL; password extracted automatically |
| `critical` | `null` | Whether the dependency is critical |
| `labels` | `null` | Custom Prometheus labels |

### Full Example

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Password from URL
    .AddRedis("redis-cache", url: "redis://:mypassword@redis.svc:6379", critical: false)

    // No password
    .AddRedis("redis-sessions", url: "redis://redis-sessions.svc:6379", critical: true)
    .Build();
```

### Pool Mode

```csharp
using DepHealth;
using DepHealth.Checks;
using StackExchange.Redis;

IConnectionMultiplexer multiplexer =
    await ConnectionMultiplexer.ConnectAsync("redis.svc:6379");

var checker = new RedisChecker(multiplexer);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("redis-cache", DependencyType.Redis, "redis.svc", "6379", checker,
        critical: false)
    .Build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| PING returns PONG | `ok` | `ok` |
| "NOAUTH" in error message | `auth_error` | `auth_error` |
| "WRONGPASS" in error message | `auth_error` | `auth_error` |
| "AUTH" in error message | `auth_error` | `auth_error` |
| Connection refused | `connection_error` | `connection_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```csharp
using DepHealth.Checks;
using StackExchange.Redis;

// Standalone mode
var checker = new RedisChecker(
    connectionString: "redis.svc:6379,connectTimeout=5000,abortConnect=true,password=secret");

// Pool mode
IConnectionMultiplexer mux = await ConnectionMultiplexer.ConnectAsync("redis.svc:6379");
var poolChecker = new RedisChecker(mux);
```

### Behavior Notes

- Uses `StackExchange.Redis` library (`IConnectionMultiplexer`)
- Standalone mode creates a new `ConnectionMultiplexer`, issues `PING`, then disposes it
- Pool mode calls `IConnectionMultiplexer.GetDatabase().PingAsync()` on the provided instance
- Auth errors are detected by inspecting the exception message for `NOAUTH`, `WRONGPASS`,
  or `AUTH` substrings

---

## AMQP (RabbitMQ)

Checks AMQP brokers by establishing a connection and closing it immediately.
Only standalone mode is supported.

### Registration

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddAmqp("rabbitmq", url: "amqp://user:pass@rabbitmq.svc:5672/myvhost",
        critical: false)
    .Build();
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | required | AMQP connection URL; username, password, and virtual host extracted automatically |
| `critical` | `null` | Whether the dependency is critical |
| `labels` | `null` | Custom Prometheus labels |

### Full Example

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Default virtual host "/"
    .AddAmqp("rabbitmq", url: "amqp://user:pass@rabbitmq.svc:5672", critical: false)

    // Custom virtual host
    .AddAmqp("rabbitmq-prod",
        url: "amqp://myuser:mypass@rmq-prod.svc:5672/myvhost",
        critical: true)
    .Build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Connection established and open | `ok` | `ok` |
| "403" in error message | `auth_error` | `auth_error` |
| "ACCESS_REFUSED" in error message | `auth_error` | `auth_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```csharp
using DepHealth.Checks;

var checker = new AmqpChecker(
    username: "myuser",
    password: "mypass",
    vhost: "myvhost");

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("rabbitmq", DependencyType.Amqp, "rabbitmq.svc", "5672", checker,
        critical: false)
    .Build();
```

### Behavior Notes

- Uses `RabbitMQ.Client` library (`RabbitMQ.Client.ConnectionFactory`)
- No pool mode — always creates a new connection
- Connection is closed immediately after successful establishment
- Default virtual host is `/` when not specified in the URL
- Auth errors are detected by inspecting the exception message for `403` or
  `ACCESS_REFUSED` substrings

---

## Kafka

Checks Kafka brokers by connecting and requesting cluster metadata.
Stateless checker — no configuration options beyond the URL.

### Registration

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddKafka("kafka", url: "kafka://kafka.svc:9092", critical: true)
    .Build();
```

### Options

No checker-specific options. Kafka checker is stateless.

### Full Example

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddKafka("kafka", url: "kafka://kafka.svc:9092", critical: true)
    .Build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Metadata returns brokers | `ok` | `ok` |
| No brokers in metadata | `unhealthy` | `no_brokers` |
| Connection/metadata error | classified by core | depends on error type |

### Direct Checker Usage

```csharp
using DepHealth.Checks;

var checker = new KafkaChecker();

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("kafka", DependencyType.Kafka, "kafka.svc", "9092", checker,
        critical: true)
    .Build();
```

### Behavior Notes

- Uses `Confluent.Kafka` library (`Confluent.Kafka.AdminClientBuilder`)
- Creates a new `AdminClient`, calls `GetMetadata(timeout: 5s)`, then disposes the client
- Verifies that at least one broker is present in the metadata response
- `SocketTimeoutMs` is set to 5000 ms
- No authentication support (plain TCP only)

---

## LDAP

Checks LDAP directory servers. Supports 4 check methods, 3 connection
protocols, and both standalone and pool modes.

### Registration

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddLdap("ldap", host: "ldap.svc", port: "389", critical: true)
    .Build();
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `checkMethod` | `LdapCheckMethod.RootDse` | Check method (see below) |
| `bindDN` | `""` | Bind DN for `SimpleBind` or `Search` |
| `bindPassword` | `""` | Bind password |
| `baseDN` | `""` | Base DN for `Search` (required for `Search` method) |
| `searchFilter` | `(objectClass=*)` | LDAP search filter |
| `searchScope` | `LdapSearchScope.Base` | Search scope: `Base`, `One`, or `Sub` |
| `useTls` | `false` | Use LDAPS (TLS from connection start) |
| `startTls` | `false` | Use StartTLS (upgrade plain connection to TLS) |
| `tlsSkipVerify` | `false` | Skip TLS certificate verification |
| `critical` | `null` | Whether the dependency is critical |
| `labels` | `null` | Custom Prometheus labels |

### Check Methods

| Method | Description |
| --- | --- |
| `AnonymousBind` | Performs an anonymous LDAP bind (empty DN and password) |
| `SimpleBind` | Performs a bind with `bindDN` and `bindPassword` (both required) |
| `RootDse` | Queries the Root DSE entry (default; works without authentication) |
| `Search` | Performs an LDAP search with `baseDN`, filter, and scope |

### Connection Protocols

| Protocol | Port | `useTls` | `startTls` | TLS |
| --- | --- | --- | --- | --- |
| Plain LDAP | 389 | `false` | `false` | No |
| LDAPS | 636 | `true` | `false` | Yes (from connection start) |
| StartTLS | 389 | `false` | `true` | Yes (upgraded after connect) |

### Full Examples

**RootDse (default)** — simplest check, no credentials required:

```csharp
.AddLdap("ldap", host: "ldap.svc", port: "389", critical: true)
```

**AnonymousBind** — test that anonymous access is allowed:

```csharp
.AddLdap("ldap", host: "ldap.svc", port: "389",
    checkMethod: LdapCheckMethod.AnonymousBind,
    critical: true)
```

**SimpleBind** — verify credentials:

```csharp
.AddLdap("ldap", host: "ldap.svc", port: "389",
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=admin,dc=example,dc=org",
    bindPassword: "secret",
    critical: true)
```

**Search** — perform an authenticated search:

```csharp
.AddLdap("ldap", host: "ldap.svc", port: "389",
    checkMethod: LdapCheckMethod.Search,
    bindDN: "cn=readonly,dc=example,dc=org",
    bindPassword: "pass",
    baseDN: "ou=users,dc=example,dc=org",
    searchFilter: "(uid=healthcheck)",
    searchScope: LdapSearchScope.One,
    critical: true)
```

**LDAPS** — TLS from connection start:

```csharp
.AddLdap("ldap-secure", host: "ldap.svc", port: "636",
    useTls: true,
    tlsSkipVerify: true,
    critical: true)
```

**StartTLS** — upgrade plain connection to TLS:

```csharp
.AddLdap("ldap-starttls", host: "ldap.svc", port: "389",
    startTls: true,
    critical: true)
```

### Pool Mode

Use an existing `ILdapConnection` for health checks:

```csharp
using DepHealth;
using DepHealth.Checks;
using Novell.Directory.Ldap;

var ldapConn = new LdapConnection();
ldapConn.Connect("ldap.svc", 389);

var checker = new LdapChecker(
    connection: ldapConn,
    checkMethod: LdapCheckMethod.RootDse);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("ldap", DependencyType.Ldap, "ldap.svc", "389", checker, critical: true)
    .Build();
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Check succeeds | `ok` | `ok` |
| ResultCode 49 (INVALID_CREDENTIALS) | `auth_error` | `auth_error` |
| ResultCode 50 (INSUFFICIENT_ACCESS_RIGHTS) | `auth_error` | `auth_error` |
| ResultCode 51 (BUSY) | `unhealthy` | `unhealthy` |
| ResultCode 52 (UNAVAILABLE) | `unhealthy` | `unhealthy` |
| ResultCode 53 (UNWILLING_TO_PERFORM) | `unhealthy` | `unhealthy` |
| ResultCode 81 (SERVER_DOWN) | `connection_error` | `connection_error` |
| ResultCode 91 (CONNECT_ERROR) | `connection_error` | `connection_error` |
| Connection refused | `connection_error` | `connection_error` |
| TLS / SSL / certificate error | `tls_error` | `tls_error` |
| ResultCode 85 (LDAP_TIMEOUT) | `timeout` | `timeout` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```csharp
using DepHealth.Checks;

// Standalone mode
var checker = new LdapChecker(
    checkMethod: LdapCheckMethod.RootDse);

// Standalone mode with SimpleBind
var bindChecker = new LdapChecker(
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=admin,dc=example,dc=org",
    bindPassword: "secret");
```

### Config Validation

The `LdapChecker` constructor validates configuration:

- `SimpleBind` requires both `bindDN` and `bindPassword` to be non-empty
- `Search` requires `baseDN` to be non-empty
- `startTls: true` is incompatible with `useTls: true` (they throw `ValidationException`)

### Behavior Notes

- Uses `Novell.Directory.Ldap` library (`Novell.Directory.Ldap.NETStandard`)
- Standalone mode creates a new `LdapConnection` for each check; disconnects after check
- Pool mode uses the provided `ILdapConnection` (caller manages lifecycle)
- `RootDse` queries `namingContexts` and `subschemaSubentry` attributes with `MaxResults = 1`
- `Search` method limits results to 1 (`MaxResults = 1`)
- `SearchWithConfig` performs a bind before search when `bindDN` is non-empty

---

## Error Classification Summary

All checkers classify errors into status categories. The core error classifier
handles common error types (timeouts, DNS errors, TLS errors, connection refused).
Checker-specific classification adds protocol-level detail:

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

- [API Reference](api-reference.md) — full API reference for `DepHealthMonitor` and types
- [API Reference (Russian)](api-reference.ru.md) — full API reference in Russian
