*[Русская версия](authentication.ru.md)*

# Authentication

This guide covers authentication options for all checkers that support
credentials — HTTP, gRPC, database/cache checkers, and LDAP.

## Overview

| Checker | Auth methods | How credentials are passed |
| --- | --- | --- |
| HTTP | Bearer token, Basic auth, custom headers | `.httpBearerToken()`, `.httpBasicAuth()`, `.httpHeaders()` |
| gRPC | Bearer token, Basic auth, custom metadata | `.grpcBearerToken()`, `.grpcBasicAuth()`, `.grpcMetadata()` |
| PostgreSQL | Username/password | Credentials in URL or `.dbUsername()` + `.dbPassword()` |
| MySQL | Username/password | Credentials in URL or `.dbUsername()` + `.dbPassword()` |
| Redis | Password | `.redisPassword()` or password in URL |
| AMQP | Username/password | `.amqpUsername()` + `.amqpPassword()` or `.amqpUrl()` |
| LDAP | Bind DN/password | `.ldapBindDN()` + `.ldapBindPassword()` |
| TCP | — | No authentication |
| Kafka | — | No authentication |

## HTTP Authentication

Only one authentication method per dependency is allowed. Specifying
both `.httpBearerToken()` and `.httpBasicAuth()` causes a validation
error.

### Bearer Token

```java
.dependency("secure-api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true)
    .httpBearerToken(System.getenv("API_TOKEN")))
```

Sends `Authorization: Bearer <token>` header with each health check request.

### Basic Auth

```java
.dependency("basic-api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true)
    .httpBasicAuth("admin", System.getenv("API_PASSWORD")))
```

Sends `Authorization: Basic <base64(user:pass)>` header.

### Custom Headers

For non-standard authentication schemes (API keys, custom tokens):

```java
.dependency("custom-auth-api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true)
    .httpHeaders(Map.of(
        "X-API-Key", System.getenv("API_KEY"),
        "X-API-Secret", System.getenv("API_SECRET"))))
```

## gRPC Authentication

Only one authentication method per dependency is allowed, same as HTTP.

### Bearer Token

```java
.dependency("grpc-backend", DependencyType.GRPC, d -> d
    .host("backend.svc")
    .port("9090")
    .critical(true)
    .grpcBearerToken(System.getenv("GRPC_TOKEN")))
```

Sends `authorization: Bearer <token>` as gRPC metadata.

### Basic Auth

```java
.dependency("grpc-backend", DependencyType.GRPC, d -> d
    .host("backend.svc")
    .port("9090")
    .critical(true)
    .grpcBasicAuth("admin", System.getenv("GRPC_PASSWORD")))
```

Sends `authorization: Basic <base64(user:pass)>` as gRPC metadata.

### Custom Metadata

```java
.dependency("grpc-backend", DependencyType.GRPC, d -> d
    .host("backend.svc")
    .port("9090")
    .critical(true)
    .grpcMetadata(Map.of("x-api-key", System.getenv("GRPC_API_KEY"))))
```

## Database Credentials

### PostgreSQL

Credentials can be included in the URL or set explicitly:

```java
// Via URL
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://myuser:mypass@pg.svc:5432/mydb")
    .critical(true))

// Via explicit parameters
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://pg.svc:5432/mydb")
    .dbUsername("myuser")
    .dbPassword("mypass")
    .critical(true))
```

Explicit parameters override credentials extracted from the URL.

### MySQL

Same approach — credentials in URL or explicit:

```java
.dependency("mysql-main", DependencyType.MYSQL, d -> d
    .url("mysql://myuser:mypass@mysql.svc:3306/mydb")
    .critical(true))
```

### Redis

Password can be set via option or included in the URL:

```java
// Via option
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .host("redis.svc")
    .port("6379")
    .redisPassword(System.getenv("REDIS_PASSWORD"))
    .critical(false))

// Via URL
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://:mypassword@redis.svc:6379/0")
    .critical(false))
```

### AMQP (RabbitMQ)

Credentials can be set via options or in the AMQP URL:

```java
// Via options
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .host("rabbitmq.svc")
    .port("5672")
    .amqpUsername("user")
    .amqpPassword("pass")
    .amqpVirtualHost("/")
    .critical(false))

// Via URL
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .amqpUrl("amqp://user:pass@rabbitmq.svc:5672/")
    .critical(false))
```

### LDAP

For `simple_bind` check method, bind credentials are required:

```java
.dependency("directory", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod("simple_bind")
    .ldapBindDN("cn=monitor,dc=corp,dc=com")
    .ldapBindPassword(System.getenv("LDAP_PASSWORD"))
    .critical(true))
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
| MySQL | Error 1045 (Access Denied) | `auth_error` | `auth_error` |
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
3. **Prefer TLS** — enable TLS for HTTP and gRPC checkers when using
   authentication to protect credentials in transit
4. **Limit permissions** — use read-only credentials for health checks;
   `SELECT 1` does not require write permissions

## Full Example

```java
import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.model.DependencyType;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;

public class Main {
    public static void main(String[] args) {
        var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

        var dh = DepHealth.builder("my-service", "my-team", registry)
            // HTTP with Bearer token
            .dependency("payment-api", DependencyType.HTTP, d -> d
                .url("https://payment.svc:443")
                .httpTls(true)
                .httpBearerToken(System.getenv("PAYMENT_TOKEN"))
                .critical(true))

            // gRPC with custom metadata
            .dependency("user-service", DependencyType.GRPC, d -> d
                .host("user.svc")
                .port("9090")
                .grpcMetadata(Map.of("x-api-key", System.getenv("USER_API_KEY")))
                .critical(true))

            // PostgreSQL with credentials in URL
            .dependency("postgres-main", DependencyType.POSTGRES, d -> d
                .url(System.getenv("DATABASE_URL"))
                .critical(true))

            // Redis with password
            .dependency("redis-cache", DependencyType.REDIS, d -> d
                .host("redis.svc")
                .port("6379")
                .redisPassword(System.getenv("REDIS_PASSWORD"))
                .critical(false))

            // AMQP with credentials
            .dependency("rabbitmq", DependencyType.AMQP, d -> d
                .amqpUrl(System.getenv("AMQP_URL"))
                .critical(false))

            // LDAP with bind credentials
            .dependency("directory", DependencyType.LDAP, d -> d
                .url("ldaps://ad.corp:636")
                .ldapCheckMethod("simple_bind")
                .ldapBindDN("cn=monitor,dc=corp,dc=com")
                .ldapBindPassword(System.getenv("LDAP_PASSWORD"))
                .critical(true))

            .build();

        dh.start();
        Runtime.getRuntime().addShutdownHook(new Thread(dh::stop));
    }
}
```

## See Also

- [Checkers](checkers.md) — detailed guide for all 9 checkers
- [Configuration](configuration.md) — environment variables and options
- [Metrics](metrics.md) — Prometheus metrics including status categories
- [Troubleshooting](troubleshooting.md) — common issues and solutions
