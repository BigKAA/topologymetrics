*[Русская версия](java.ru.md)*

# Quick Start: Java SDK

A guide to connecting dephealth to a Java service in just a few minutes.

## Installation

### Maven

Core module:

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-core</artifactId>
    <version>0.2.2</version>
</dependency>
```

Spring Boot Starter (includes core):

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.2.2</version>
</dependency>
```

## Minimal Example

Connecting a single HTTP dependency with metrics export:

```java
import biz.kryukov.dev.dephealth.*;
import io.micrometer.prometheus.PrometheusMeterRegistry;

public class Main {
    public static void main(String[] args) {
        PrometheusMeterRegistry registry = new PrometheusMeterRegistry(
            PrometheusConfig.DEFAULT);

        DepHealth depHealth = DepHealth.builder("my-service", registry)
            .dependency("payment-api", DependencyType.HTTP, d -> d
                .url("http://payment.svc:8080")
                .critical(true))
            .build();

        depHealth.start();

        // ... HTTP server exports registry.scrape() at /metrics ...

        // Graceful shutdown
        Runtime.getRuntime().addShutdownHook(new Thread(depHealth::stop));
    }
}
```

After startup, metrics will appear at `/metrics`:

```text
app_dependency_health{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Multiple Dependencies

```java
DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    // Global settings
    .checkInterval(Duration.ofSeconds(30))
    .timeout(Duration.ofSeconds(3))

    // PostgreSQL — standalone check (new connection)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))

    // Redis — standalone check
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .url("redis://:password@redis.svc:6379/0")
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

    // RabbitMQ
    .dependency("rabbitmq", DependencyType.AMQP, d -> d
        .host("rabbitmq.svc")
        .port("5672")
        .amqpUsername("user")
        .amqpPassword("pass")
        .amqpVirtualHost("/")
        .critical(false))

    // Kafka
    .dependency("kafka", DependencyType.KAFKA, d -> d
        .host("kafka.svc")
        .port("9092")
        .critical(false))

    .build();
```

## Custom Labels

Add custom labels via `.label()`:

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgres://user:pass@pg.svc:5432/mydb")
    .critical(true)
    .label("role", "primary")
    .label("shard", "eu-west"))
```

Result in metrics:

```text
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",role="primary",shard="eu-west"} 1
```

## Connection Pool Integration

Preferred mode: SDK uses the service's existing connection pool
instead of creating new connections.

### PostgreSQL via DataSource

```java
import javax.sql.DataSource;
// HikariCP, Tomcat JDBC or any other pool

DataSource dataSource = ...; // existing pool from the application

DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .dataSource(dataSource)
        .critical(true))
    .build();
```

### Redis via JedisPool

```java
import redis.clients.jedis.JedisPool;

JedisPool jedisPool = ...; // existing pool from the application

DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .jedisPool(jedisPool)
        .critical(false))
    .build();
```

## Spring Boot Integration

Add the `dephealth-spring-boot-starter` dependency and configure
`application.yml`:

```yaml
dephealth:
  name: my-service
  interval: 15s
  timeout: 5s

  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      critical: true

    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false

    auth-service:
      type: http
      url: http://auth.svc:8080
      health-path: /healthz
      critical: true
      labels:
        role: primary

    user-service:
      type: grpc
      host: user.svc
      port: "9090"
      critical: false

    rabbitmq:
      type: amqp
      url: amqp://user:pass@rabbitmq.svc:5672/
      amqp-username: user
      amqp-password: pass
      critical: false

    kafka:
      type: kafka
      host: kafka.svc
      port: "9092"
      critical: false
```

Spring Boot automatically:

- Creates and starts `DepHealth` bean
- Registers Health Indicator (`/actuator/health`)
- Adds endpoint `/actuator/dependencies`
- Exports Prometheus metrics at `/actuator/prometheus`

### Actuator endpoints

```bash
# Dependency status
curl http://localhost:8080/actuator/dependencies

# Response:
{
    "postgres-main:pg.svc:5432": true,
    "redis-cache:redis.svc:6379": true,
    "auth-service:auth.svc:8080": false
}
```

```bash
# Health indicator (integrated into the main /actuator/health)
curl http://localhost:8080/actuator/health

# Response:
{
    "status": "DOWN",
    "components": {
        "dephealth": {
            "status": "DOWN",
            "details": {
                "postgres-main:pg.svc:5432": "UP",
                "auth-service:auth.svc:8080": "DOWN"
            }
        }
    }
}
```

## Global Options

```java
DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    // Check interval (default 15s)
    .checkInterval(Duration.ofSeconds(30))

    // Timeout for each check (default 5s)
    .timeout(Duration.ofSeconds(3))

    // ...dependencies
    .build();
```

## Dependency Options

Each dependency can override global settings:

```java
.dependency("slow-service", DependencyType.HTTP, d -> d
    .url("http://slow.svc:8080")
    .httpHealthPath("/ready")           // health check path
    .httpTls(true)                      // HTTPS
    .httpTlsSkipVerify(true)            // skip certificate verification
    .interval(Duration.ofSeconds(60))   // custom interval
    .timeout(Duration.ofSeconds(10))    // custom timeout
    .critical(true))                    // critical dependency
```

## Configuration via Environment Variables

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Application name (overridden by API) | `my-service` |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality | `yes` / `no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label | `primary` |

`<DEP>` — dependency name in uppercase, hyphens replaced with `_`.

Examples:

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
```

Priority: values from API/application.yml > environment variables.

## Behavior When Required Parameters Are Missing

| Situation | Behavior |
| --- | --- |
| No `name` specified and no `DEPHEALTH_NAME` | Error on creation: `missing name` |
| No `.critical()` specified for dependency | Error on creation: `missing critical` |
| Invalid label name | Error on creation: `invalid label name` |
| Label conflicts with required label | Error on creation: `reserved label` |

## Checking Dependency Status

The `health()` method returns the current status of all endpoints:

```java
Map<String, Boolean> health = depHealth.health();
// {"postgres-main:pg.svc:5432": true, "redis-cache:redis.svc:6379": true}

// Use for readiness probe
boolean allHealthy = health.values().stream().allMatch(Boolean::booleanValue);
```

## Metrics Export

dephealth exports four Prometheus metrics via Micrometer:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = available, `0` = unavailable |
| `app_dependency_latency_seconds` | Histogram | Check latency (seconds) |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed reason: e.g. `http_503`, `auth_error` |

Labels: `name`, `dependency`, `type`, `host`, `port`, `critical`.
Additional: `status` (on `app_dependency_status`), `detail` (on `app_dependency_status_detail`).

For Spring Boot: metrics are available at `/actuator/prometheus`.

## Supported Dependency Types

| DependencyType | Type | Check Method |
| --- | --- | --- |
| `HTTP` | `http` | HTTP GET to health endpoint, expecting 2xx |
| `GRPC` | `grpc` | gRPC Health Check Protocol |
| `TCP` | `tcp` | Establish TCP connection |
| `POSTGRES` | `postgres` | `SELECT 1` via JDBC |
| `MYSQL` | `mysql` | `SELECT 1` via JDBC |
| `REDIS` | `redis` | `PING` command via Jedis |
| `AMQP` | `amqp` | Check connection to RabbitMQ |
| `KAFKA` | `kafka` | Metadata request to broker |

## Default Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `checkInterval` | 15s | Interval between checks |
| `timeout` | 5s | Timeout for a single check |
| `initialDelay` | 5s | Delay before first check |
| `failureThreshold` | 1 | Number of failures before transitioning to unhealthy |
| `successThreshold` | 1 | Number of successes before transitioning to healthy |

## Next Steps

- [Integration Guide](../migration/java.md) — step-by-step integration
  with an existing service
- [Specification Overview](../specification.md) — details of metrics contracts and behavior
