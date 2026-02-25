*[Русская версия](getting-started.ru.md)*

# Getting Started

This guide covers installation, basic setup, and your first health check
with the dephealth Java SDK.

## Prerequisites

- Java 21 or later
- Maven or Gradle
- A running dependency to monitor (PostgreSQL, Redis, HTTP service, etc.)

## Installation

### Maven

Core module (programmatic API):

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-core</artifactId>
    <version>0.8.0</version>
</dependency>
```

Spring Boot Starter (includes core):

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.8.0</version>
</dependency>
```

### Gradle

```groovy
// Core module
implementation 'biz.kryukov.dev:dephealth-core:0.8.0'

// Or Spring Boot Starter
implementation 'biz.kryukov.dev:dephealth-spring-boot-starter:0.8.0'
```

## Minimal Example

Monitor a single HTTP dependency and expose Prometheus metrics:

```java
import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.model.DependencyType;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;

public class Main {
    public static void main(String[] args) {
        var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

        var dh = DepHealth.builder("my-service", "my-team", registry)
            .dependency("payment-api", DependencyType.HTTP, d -> d
                .url("http://payment.svc:8080")
                .critical(true))
            .build();

        dh.start();

        // Expose registry.scrape() at /metrics via your HTTP server

        // Graceful shutdown
        Runtime.getRuntime().addShutdownHook(new Thread(dh::stop));
    }
}
```

After startup, Prometheus metrics appear at `/metrics`:

```text
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Key Concepts

### Name and Group

Every `DepHealth` instance requires two identifiers:

- **name** — unique application name (e.g., `"my-service"`)
- **group** — logical group the service belongs to (e.g., `"my-team"`, `"payments"`)

Both appear as labels in all exported metrics. Validation rules:
`[a-z][a-z0-9-]*`, 1-63 characters.

If not passed as arguments, the SDK reads `DEPHEALTH_NAME` and
`DEPHEALTH_GROUP` environment variables as fallback.

### Dependencies

Each dependency is registered via the builder's `.dependency()` method
with a `DependencyType`:

| DependencyType | Description |
| --- | --- |
| `HTTP` | HTTP service |
| `GRPC` | gRPC service |
| `TCP` | TCP endpoint |
| `POSTGRES` | PostgreSQL database |
| `MYSQL` | MySQL database |
| `REDIS` | Redis server |
| `AMQP` | RabbitMQ (AMQP broker) |
| `KAFKA` | Apache Kafka broker |
| `LDAP` | LDAP directory server |

Each dependency requires:

- A **name** (first argument) — identifies the dependency in metrics
- **Endpoint** — specified via `.url()`, `.jdbcUrl()`, or `.host()` + `.port()`
- **Critical** flag — `.critical(true)` or `.critical(false)` (mandatory)

### Lifecycle

1. **Create** — `DepHealth.builder(...).build()`
2. **Start** — `dh.start()` launches periodic health checks
3. **Run** — checks execute at the configured interval (default 15s)
4. **Stop** — `dh.stop()` stops checks and shuts down the scheduler

## Multiple Dependencies

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Global settings
    .checkInterval(Duration.ofSeconds(30))
    .timeout(Duration.ofSeconds(3))

    // PostgreSQL
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true))

    // Redis
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .url(System.getenv("REDIS_URL"))
        .critical(false))

    // HTTP service
    .dependency("auth-service", DependencyType.HTTP, d -> d
        .url("http://auth.svc:8080")
        .httpHealthPath("/healthz")
        .critical(true))

    // gRPC service
    .dependency("user-service", DependencyType.GRPC, d -> d
        .host("user.svc")
        .port("9090")
        .critical(false))

    .build();
```

## Checking Health Status

### Simple Status

```java
Map<String, Boolean> health = dh.health();
// {"postgres-main:pg.svc:5432": true, "redis-cache:redis.svc:6379": true}

// Use for readiness probe
boolean allHealthy = health.values().stream().allMatch(Boolean::booleanValue);
```

### Detailed Status

```java
Map<String, EndpointStatus> details = dh.healthDetails();
details.forEach((key, ep) ->
    System.out.printf("%s: healthy=%s status=%s latency=%.1fms%n",
        key, ep.isHealthy(), ep.getStatus(), ep.getLatencyMillis()));
```

`healthDetails()` returns an `EndpointStatus` object with health state,
status category, latency, timestamps, and custom labels. Before the first
check completes, `healthy` is `null` and `status` is `"unknown"`.

## Next Steps

- [Checkers](checkers.md) — detailed guide for all 9 built-in checkers
- [Configuration](configuration.md) — all options, defaults, and environment variables
- [Connection Pools](connection-pools.md) — integration with existing connection pools
- [Spring Boot Integration](spring-boot.md) — auto-configuration and actuator
- [Authentication](authentication.md) — auth for HTTP, gRPC, and database checkers
- [Metrics](metrics.md) — Prometheus metrics reference and PromQL examples
- [API Reference](api-reference.md) — complete reference of all public classes
- [Troubleshooting](troubleshooting.md) — common issues and solutions
- [Migration Guide](migration.md) — version upgrade instructions
- [Code Style](code-style.md) — Java code conventions for this project
- [Examples](examples/) — complete runnable examples
