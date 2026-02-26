*[Русская версия](authentication.ru.md)*

# Authentication

This guide covers authentication options for all checkers that support
credentials — HTTP, gRPC, database/cache checkers, and LDAP.

## Overview

| Checker | Auth methods | How credentials are passed |
| --- | --- | --- |
| HTTP | Bearer token, Basic auth, custom headers | `bearerToken:`, `basicAuthUsername:`/`basicAuthPassword:`, `headers:` |
| gRPC | Bearer token, Basic auth, custom metadata | `bearerToken:`, `basicAuthUsername:`/`basicAuthPassword:`, `metadata:` |
| PostgreSQL | Username/password | Credentials in URL |
| MySQL | Username/password | Credentials in URL |
| Redis | Password | Password in URL or `IConnectionMultiplexer` |
| AMQP | Username/password | `AmqpUsername()`, `AmqpPassword()` or credentials in URL |
| LDAP | Bind DN/password | `bindDN:`, `bindPassword:` parameters |
| TCP | — | No authentication |
| Kafka | — | No authentication |

## HTTP Authentication

Only one authentication method per dependency is allowed. Specifying
both `bearerToken` and `basicAuthUsername`/`basicAuthPassword` causes a
`ValidationException`.

### Bearer Token

```csharp
.AddHttp("secure-api", "http://api.svc:8080",
    critical: true,
    bearerToken: Environment.GetEnvironmentVariable("API_TOKEN"))
```

Sends `Authorization: Bearer <token>` header with each health check request.

### Basic Auth

```csharp
.AddHttp("basic-api", "http://api.svc:8080",
    critical: true,
    basicAuthUsername: "admin",
    basicAuthPassword: Environment.GetEnvironmentVariable("API_PASSWORD"))
```

Sends `Authorization: Basic <base64(user:pass)>` header.

### Custom Headers

For non-standard authentication schemes (API keys, custom tokens):

```csharp
.AddHttp("custom-auth-api", "http://api.svc:8080",
    critical: true,
    headers: new Dictionary<string, string>
    {
        ["X-API-Key"] = Environment.GetEnvironmentVariable("API_KEY")!,
        ["X-API-Secret"] = Environment.GetEnvironmentVariable("API_SECRET")!
    })
```

## gRPC Authentication

Only one authentication method per dependency is allowed, same as HTTP.

### Bearer Token

```csharp
.AddGrpc("grpc-backend",
    host: "backend.svc",
    port: "9090",
    critical: true,
    bearerToken: Environment.GetEnvironmentVariable("GRPC_TOKEN"))
```

Sends `authorization: Bearer <token>` as gRPC metadata.

### Basic Auth

```csharp
.AddGrpc("grpc-backend",
    host: "backend.svc",
    port: "9090",
    critical: true,
    basicAuthUsername: "admin",
    basicAuthPassword: Environment.GetEnvironmentVariable("GRPC_PASSWORD"))
```

Sends `authorization: Basic <base64(user:pass)>` as gRPC metadata.

### Custom Metadata

```csharp
.AddGrpc("grpc-backend",
    host: "backend.svc",
    port: "9090",
    critical: true,
    metadata: new Dictionary<string, string>
    {
        ["x-api-key"] = Environment.GetEnvironmentVariable("GRPC_API_KEY")!
    })
```

## Database Credentials

### PostgreSQL

Credentials are included in the connection URL:

```csharp
// Via URL
.AddPostgres("postgres-main",
    url: "postgresql://myuser:mypass@pg.svc:5432/mydb",
    critical: true)

// Via environment variable
.AddPostgres("postgres-main",
    url: Environment.GetEnvironmentVariable("DATABASE_URL")!,
    critical: true)
```

The SDK extracts username, password, host, port, and database name from the URL automatically.

### MySQL

Same approach — credentials in the URL:

```csharp
.AddMySql("mysql-main",
    url: "mysql://myuser:mypass@mysql.svc:3306/mydb",
    critical: true)
```

### Redis

Password can be included in the URL or passed via `IConnectionMultiplexer` in pool mode:

```csharp
// Via URL
.AddRedis("redis-cache",
    url: "redis://:mypassword@redis.svc:6379/0",
    critical: false)

// No password
.AddRedis("redis-cache",
    url: "redis://redis.svc:6379",
    critical: false)
```

Pool mode with `IConnectionMultiplexer` — the multiplexer is already authenticated
by the application:

```csharp
using DepHealth.Checks;
using StackExchange.Redis;

IConnectionMultiplexer multiplexer =
    await ConnectionMultiplexer.ConnectAsync("redis.svc:6379,password=secret");

var checker = new RedisChecker(multiplexer);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("redis-cache", DependencyType.Redis, "redis.svc", "6379", checker,
        critical: false)
    .Build();
```

### AMQP (RabbitMQ)

Credentials are passed in the AMQP URL:

```csharp
// Via URL
.AddAmqp("rabbitmq",
    url: "amqp://user:pass@rabbitmq.svc:5672/",
    critical: false)

// Custom virtual host
.AddAmqp("rabbitmq",
    url: "amqp://user:pass@rabbitmq.svc:5672/myvhost",
    critical: false)
```

For direct checker instantiation, credentials can be passed explicitly:

```csharp
using DepHealth.Checks;

var checker = new AmqpChecker(
    username: "user",
    password: "pass",
    vhost: "/");
```

### LDAP

For `SimpleBind` check method, bind credentials are required:

```csharp
.AddLdap("directory",
    host: "ldap.svc",
    port: "389",
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=monitor,dc=corp,dc=com",
    bindPassword: Environment.GetEnvironmentVariable("LDAP_PASSWORD"),
    critical: true)
```

For LDAPS (TLS):

```csharp
.AddLdap("directory-secure",
    host: "ldap.svc",
    port: "636",
    useTls: true,
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=monitor,dc=corp,dc=com",
    bindPassword: Environment.GetEnvironmentVariable("LDAP_PASSWORD"),
    critical: true)
```

## Auth Error Classification

When a dependency rejects credentials, the checker classifies the error
as `auth_error`. This enables specific alerting on authentication failures
without mixing them with connectivity or health issues.

| Checker | Trigger | Status | Detail |
| --- | --- | --- | --- |
| HTTP | Response 401 (Unauthorized) | `auth_error` | `auth_error` |
| HTTP | Response 403 (Forbidden) | `auth_error` | `auth_error` |
| gRPC | Code UNAUTHENTICATED | `auth_error` | `auth_error` |
| gRPC | Code PERMISSION_DENIED | `auth_error` | `auth_error` |
| PostgreSQL | SQLSTATE 28000/28P01 | `auth_error` | `auth_error` |
| MySQL | Error AccessDenied (1045) | `auth_error` | `auth_error` |
| Redis | NOAUTH/WRONGPASS error | `auth_error` | `auth_error` |
| AMQP | 403 ACCESS_REFUSED | `auth_error` | `auth_error` |
| LDAP | Code 49 (Invalid Credentials) | `auth_error` | `auth_error` |
| LDAP | Code 50 (Insufficient Access) | `auth_error` | `auth_error` |

### PromQL Example: Auth Error Alerting

```promql
# Alert when any dependency has auth_error status
app_dependency_status{status="auth_error"} == 1
```

## Security Best Practices

1. **Never hardcode credentials** — always use environment variables or
   secret management systems
2. **Use short-lived tokens** — if your auth system supports token rotation,
   the checker will use the latest token value passed during creation
3. **Prefer TLS** — enable TLS for HTTP (`https://`) and gRPC (`tlsEnabled: true`)
   checkers when using authentication to protect credentials in transit
4. **Limit permissions** — use read-only credentials for health checks;
   `SELECT 1` does not require write permissions

## Full Example

```csharp
using DepHealth;
using DepHealth.Checks;
using StackExchange.Redis;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // HTTP with Bearer token
    .AddHttp("payment-api", "https://payment.svc:443",
        critical: true,
        bearerToken: Environment.GetEnvironmentVariable("PAYMENT_TOKEN"))

    // gRPC with custom metadata
    .AddGrpc("user-service",
        host: "user.svc",
        port: "9090",
        critical: true,
        metadata: new Dictionary<string, string>
        {
            ["x-api-key"] = Environment.GetEnvironmentVariable("USER_API_KEY")!
        })

    // PostgreSQL with credentials in URL
    .AddPostgres("postgres-main",
        url: Environment.GetEnvironmentVariable("DATABASE_URL")!,
        critical: true)

    // Redis with password in URL
    .AddRedis("redis-cache",
        url: $"redis://:{Environment.GetEnvironmentVariable("REDIS_PASSWORD")}@redis.svc:6379",
        critical: false)

    // AMQP with credentials in URL
    .AddAmqp("rabbitmq",
        url: Environment.GetEnvironmentVariable("AMQP_URL")!,
        critical: false)

    // LDAP with bind credentials
    .AddLdap("directory",
        host: "ad.corp",
        port: "636",
        useTls: true,
        checkMethod: LdapCheckMethod.SimpleBind,
        bindDN: "cn=monitor,dc=corp,dc=com",
        bindPassword: Environment.GetEnvironmentVariable("LDAP_PASSWORD"),
        critical: true)

    .Build();

dh.Start();
Console.CancelKeyPress += (_, _) => dh.Stop();
Console.ReadLine();
```

## See Also

- [Checkers](checkers.md) — detailed guide for all 9 checkers
- [Configuration](configuration.md) — environment variables and options
- [API Reference](api-reference.md) — complete reference of all public classes
