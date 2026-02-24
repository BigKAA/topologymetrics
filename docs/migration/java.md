*[Русская версия](java.ru.md)*

# Guide to Integrating dephealth into an Existing Java Service

Step-by-step instructions for adding dependency monitoring
to a running microservice.

## Migration to v0.6.0

### New: Dynamic Endpoint Management

v0.6.0 adds three methods for managing endpoints at runtime. No existing API
changes — this is a purely additive feature.

```java
// Add a new endpoint after start()
depHealth.addEndpoint("api-backend", DependencyType.HTTP, true,
    new Endpoint("backend-2.svc", "8080"),
    new HttpHealthChecker());

// Remove an endpoint (idempotent)
depHealth.removeEndpoint("api-backend", "backend-2.svc", "8080");

// Replace an endpoint atomically
depHealth.updateEndpoint("api-backend", "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    new HttpHealthChecker());
```

Key behaviors:

- **Thread-safe** — all three methods are synchronized.
- **Idempotent** — `addEndpoint` is no-op if endpoint exists;
  `removeEndpoint` is no-op if endpoint is not found.
- Dynamic endpoints inherit the global check interval and timeout.
- `removeEndpoint` / `updateEndpoint` delete all Prometheus metrics for
  the old endpoint.
- `updateEndpoint` throws `EndpointNotFoundException` if the old endpoint
  does not exist.

For the full migration guide, see
[Java SDK v0.5.0 to v0.6.0](sdk-java-v050-to-v060.md).

---

## Migration to v0.5.0

### Breaking: mandatory `group` parameter

v0.5.0 adds a mandatory `group` parameter (logical grouping: team, subsystem, project).

Programmatic API:

```java
// v0.4.x
DepHealth dh = DepHealth.builder("my-service", meterRegistry)
    .dependency(...)
    .build();

// v0.5.0
DepHealth dh = DepHealth.builder("my-service", "my-team", meterRegistry)
    .dependency(...)
    .build();
```

Spring Boot YAML:

```yaml
# v0.5.0 — add group
dephealth:
  name: my-service
  group: my-team
  dependencies: ...
```

Alternative: set `DEPHEALTH_GROUP` environment variable (API takes precedence).

Validation: same rules as `name` — `[a-z][a-z0-9-]*`, 1-63 chars.

---

## Migration to v0.4.1

### New: healthDetails() API

v0.4.1 adds the `healthDetails()` method that returns detailed status for each
endpoint. No existing API changes — this is a purely additive feature.

```java
Map<String, EndpointStatus> details = depHealth.healthDetails();

for (var entry : details.entrySet()) {
    EndpointStatus ep = entry.getValue();
    System.out.printf("%s: healthy=%s status=%s detail=%s latency=%s%n",
        entry.getKey(), ep.healthy(), ep.status(), ep.detail(),
        ep.latencyMillis());
}
```

`EndpointStatus` fields: `dependency()`, `type()`, `host()`, `port()`,
`healthy()` (`Boolean`, `null` = unknown), `status()`, `detail()`,
`latency()`, `lastCheckedAt()`, `critical()`, `labels()`.

Before the first check, `healthy()` is `null` and `status()` is `"unknown"`.

---

## Migration to v0.4.0

### New Status Metrics (no code changes required)

v0.4.0 adds two new automatically exported Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason: e.g. `http_503`, `auth_error` |

**No code changes are needed** — the SDK exports these metrics automatically alongside the existing `app_dependency_health` and `app_dependency_latency_seconds`.

### Storage Impact

Each endpoint now produces 9 additional time series (8 for `app_dependency_status` + 1 for `app_dependency_status_detail`). For a service with 5 endpoints, this adds 45 series.

### New PromQL Queries

```promql
# Status category for a dependency
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Detailed failure reason
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Alert on authentication errors
app_dependency_status{status="auth_error"} == 1
```

For the full list of status values, see [Specification — Status Metrics](../specification.md).

## Migration from v0.1 to v0.2

### API Changes

| v0.1 | v0.2 | Description |
| --- | --- | --- |
| `DepHealth.builder(registry)` | `DepHealth.builder("my-service", registry)` | Required first argument `name` |
| `.critical(true)` (optional) | `.critical(true/false)` (required) | For each dependency |
| none | `.label("key", "value")` | Custom labels |
| `dephealth.name` (none) | `dephealth.name: my-service` | In application.yml |

### Required Changes

1. Add `name` to builder:

```java
// v0.1
DepHealth depHealth = DepHealth.builder(meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))
    .build();

// v0.2
DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))
    .build();
```

1. Specify `.critical()` for each dependency:

```java
// v0.1 — critical is optional
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://redis.svc:6379"))

// v0.2 — critical is required
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://redis.svc:6379")
    .critical(false))
```

1. Update `application.yml` (Spring Boot):

```yaml
# v0.1
dephealth:
  dependencies:
    redis-cache:
      type: redis
      url: ${REDIS_URL}

# v0.2
dephealth:
  name: my-service
  dependencies:
    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false
```

1. Update dependency version:

```xml
<!-- v0.1 -->
<version>0.1.0</version>

<!-- v0.2 -->
<version>0.2.2</version>
```

### New Labels in Metrics

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Update PromQL queries and Grafana dashboards by adding `name` and `critical` labels.

## Prerequisites

- Java 21+
- Spring Boot 3.x (recommended) or any framework with Micrometer
- Access to dependencies (DB, cache, other services) from the service

## Step 1. Install Dependency

### Spring Boot (recommended)

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.2.2</version>
</dependency>
```

### Without Spring Boot

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-core</artifactId>
    <version>0.2.2</version>
</dependency>
```

Also ensure that dependency drivers are present in the classpath
(postgresql, jedis, amqp-client, kafka-clients, etc.).

## Step 2. Configuration (Spring Boot)

Add settings to `application.yml`:

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

    payment-api:
      type: http
      url: http://payment.svc:8080
      health-path: /health
      critical: true
      labels:
        role: primary

    user-service:
      type: grpc
      host: user.svc
      port: "9090"
      critical: false
```

Spring Boot will automatically create and start the `DepHealth` bean.

## Step 3. Configuration (without Spring Boot)

### Option A: Standalone mode (simple)

SDK creates temporary connections for checks:

```java
import biz.kryukov.dev.dephealth.*;
import io.micrometer.prometheus.PrometheusMeterRegistry;

PrometheusMeterRegistry meterRegistry = ...;

DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true))
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .url(System.getenv("REDIS_URL"))
        .critical(false))
    .dependency("payment-api", DependencyType.HTTP, d -> d
        .url(System.getenv("PAYMENT_SERVICE_URL"))
        .critical(true))
    .build();
```

### Option B: Connection pool integration (recommended)

SDK uses existing service connections. Advantages:

- Reflects the actual ability of the service to work with the dependency
- Does not create additional load on DB/cache
- Detects pool problems (exhaustion, leaks)

```java
import javax.sql.DataSource;
import redis.clients.jedis.JedisPool;

DataSource dataSource = ...; // HikariCP, Tomcat JDBC, etc.
JedisPool jedisPool = ...;

DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .checkInterval(Duration.ofSeconds(15))

    // PostgreSQL via existing DataSource
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .dataSource(dataSource)
        .critical(true))

    // Redis via existing JedisPool
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .jedisPool(jedisPool)
        .critical(false))

    // For HTTP/gRPC — standalone only
    .dependency("payment-api", DependencyType.HTTP, d -> d
        .url(System.getenv("PAYMENT_SERVICE_URL"))
        .critical(true))

    .dependency("auth-service", DependencyType.GRPC, d -> d
        .host(System.getenv("AUTH_HOST"))
        .port(System.getenv("AUTH_PORT"))
        .critical(true))

    .build();
```

## Step 4. Start and Stop

### Spring Boot

Management is automatic via `DepHealthLifecycle` (SmartLifecycle).

### Without Spring Boot

Integrate `start()` and `stop()` into the service lifecycle:

```java
public class Main {
    public static void main(String[] args) {
        DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
            // ... dependencies ...
            .build();

        depHealth.start();

        // ... start HTTP server ...

        // Graceful shutdown
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            depHealth.stop();
            // ... stop HTTP server ...
        }));
    }
}
```

## Step 5. Metrics Export

### Spring Boot

Metrics are automatically available at `/actuator/prometheus`.

Ensure that in `application.yml`:

```yaml
management:
  endpoints:
    web:
      exposure:
        include: health, prometheus, dependencies
```

### Without Spring Boot

Export via Micrometer:

```java
import io.micrometer.prometheus.PrometheusMeterRegistry;

// In HTTP handler /metrics:
String metrics = meterRegistry.scrape();
response.setContentType("text/plain; version=0.0.4");
response.getWriter().write(metrics);
```

## Step 6. Dependency Status Endpoint (optional)

### Spring Boot

Two built-in endpoints are already available:

```bash
# Direct dependency status
GET /actuator/dependencies

# Response:
{
    "postgres-main:pg.svc:5432": true,
    "redis-cache:redis.svc:6379": true,
    "payment-api:payment.svc:8080": false
}

# Integrated into Spring Health Indicator
GET /actuator/health
```

### Without Spring Boot

```java
void handleDependencies(HttpServletRequest req, HttpServletResponse resp) {
    Map<String, Boolean> health = depHealth.health();

    boolean allHealthy = health.values().stream()
        .allMatch(Boolean::booleanValue);

    resp.setStatus(allHealthy ? 200 : 503);
    resp.setContentType("application/json");

    // Serialize health to JSON
    new ObjectMapper().writeValue(resp.getWriter(), health);
}
```

## Typical Configurations

### Spring Boot + PostgreSQL + Redis

```yaml
dephealth:
  name: my-service
  dependencies:
    postgres:
      type: postgres
      url: ${DATABASE_URL}
      critical: true
    redis:
      type: redis
      url: ${REDIS_URL}
      critical: false
```

### API Gateway with upstream services

```yaml
dephealth:
  name: api-gateway
  interval: 10s
  dependencies:
    user-service:
      type: http
      url: http://user-svc:8080
      health-path: /healthz
      critical: true
    order-service:
      type: http
      url: http://order-svc:8080
      critical: true
    auth-service:
      type: grpc
      host: auth-svc
      port: "9090"
      critical: true
```

### Event handler with Kafka and RabbitMQ

```yaml
dephealth:
  name: event-processor
  dependencies:
    kafka-main:
      type: kafka
      host: kafka.svc
      port: "9092"
      critical: true
    rabbitmq:
      type: amqp
      url: amqp://user:pass@rabbitmq.svc:5672/
      amqp-username: user
      amqp-password: pass
      critical: true
    postgres:
      type: postgres
      url: ${DATABASE_URL}
      critical: false
```

## Troubleshooting

### Metrics do not appear at `/actuator/prometheus`

**Check:**

1. Dependency `spring-boot-starter-actuator` is present
2. `management.endpoints.web.exposure.include` includes `prometheus`
3. `dephealth-spring-boot-starter` is in classpath

### All dependencies show `0` (unhealthy)

**Check:**

1. Network accessibility of dependencies from the service container/pod
2. DNS resolution of service names
3. Correctness of URL/host/port in configuration
4. Timeout (`5s` by default) — is it sufficient for this dependency
5. Logs: configure `logging.level.biz.kryukov.dev.dephealth=DEBUG`

### High latency of PostgreSQL/MySQL checks

**Reason**: standalone mode creates a new JDBC connection each time.

**Solution**: use `DataSource` integration.

### gRPC: `DEADLINE_EXCEEDED` error

**Check:**

1. gRPC service is accessible at the specified address
2. Service implements `grpc.health.v1.Health/Check`
3. For gRPC use `host` + `port`, not `url`
4. If TLS is needed: `tls: true` in configuration

### AMQP: connection error to RabbitMQ

**Important**: path `/` in URL means vhost `/` (not empty).

```yaml
rabbitmq:
  type: amqp
  host: rabbitmq.svc
  port: "5672"
  amqp-username: user
  amqp-password: pass
  virtual-host: /
  critical: false
```

### URL and credentials parsing

SDK automatically extracts username/password from URL:

```yaml
postgres:
  type: postgres
  url: postgres://user:pass@host:5432/db
  critical: true
  # username and password are extracted automatically
```

Can be overridden explicitly:

```yaml
postgres:
  type: postgres
  url: postgres://old:old@host:5432/db
  username: new_user    # overrides parsing from URL
  password: new_pass
  critical: true
```

### Dependency Naming

Names must comply with rules:

- Length: 1-63 characters
- Format: `[a-z][a-z0-9-]*` (lowercase letters, digits, hyphens)
- Starts with a letter
- Examples: `postgres-main`, `redis-cache`, `auth-service`

## Next Steps

- [Quick Start](../quickstart/java.md) — minimal examples
- [Specification Overview](../specification.md) — details of metrics and behavior contracts
