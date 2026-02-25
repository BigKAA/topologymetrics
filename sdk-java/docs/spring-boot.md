*[Русская версия](spring-boot.ru.md)*

# Spring Boot Integration

The `dephealth-spring-boot-starter` provides auto-configuration for
Spring Boot 3.x applications. It automatically creates, configures,
starts and stops the `DepHealth` instance based on your
`application.yml` properties.

## Installation

### Maven

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.8.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'biz.kryukov.dev:dephealth-spring-boot-starter:0.8.0'
```

The starter transitively includes `dephealth-core`, so you do not need
to add it separately.

## Auto-Configuration

The starter automatically registers the following beans when the
application starts:

| Bean | Description |
| --- | --- |
| `DepHealth` | Main health checker instance. Created from `dephealth.*` properties unless you define your own `@Bean` |
| `DepHealthLifecycle` | Implements Spring `Lifecycle` interface — starts `DepHealth` on application startup and stops it on shutdown |
| `DepHealthIndicator` | Integrates into Spring Boot Actuator `/actuator/health` endpoint |
| `DependenciesEndpoint` | Custom actuator endpoint at `/actuator/dependencies` returning dependency statuses |

Auto-configuration is conditional:

- `@ConditionalOnClass(DepHealth.class)` — the starter activates only
  when `dephealth-core` is on the classpath.
- `@ConditionalOnMissingBean(DepHealth.class)` — if you define your own
  `DepHealth` bean, the auto-configured one is skipped.

## Configuration Properties

All properties use the `dephealth` prefix. Below is a complete YAML
example with all supported options:

```yaml
dephealth:
  name: my-service
  group: my-team
  interval: 15s
  timeout: 5s
  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      critical: true
      labels:
        role: primary
    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false
    auth-service:
      type: http
      url: http://auth.svc:8080
      health-path: /healthz
      critical: true
      http-bearer-token: ${API_TOKEN}
    user-service:
      type: grpc
      host: user.svc
      port: "9090"
      critical: false
    rabbitmq:
      type: amqp
      host: rabbitmq.svc
      port: "5672"
      amqp-username: user
      amqp-password: pass
      virtual-host: /
      critical: false
    kafka:
      type: kafka
      host: kafka.svc
      port: "9092"
      critical: false
    directory:
      type: ldap
      url: ldap://ldap.svc:389
      critical: true
      ldap-check-method: root_dse
```

## Properties Reference

### Global Properties

| Property | Type | Default | Description |
| --- | --- | --- | --- |
| `dephealth.name` | `String` | — | Application name (required). Falls back to `DEPHEALTH_NAME` env var |
| `dephealth.group` | `String` | — | Application group (required). Falls back to `DEPHEALTH_GROUP` env var |
| `dephealth.interval` | `Duration` | `15s` | Global health check interval |
| `dephealth.timeout` | `Duration` | `5s` | Global health check timeout |

### Dependency Properties

Each dependency is defined under `dephealth.dependencies.<name>`.

#### Common

| Property | Type | Default | Description |
| --- | --- | --- | --- |
| `.type` | `String` | — | Dependency type: `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka`, `ldap` (required) |
| `.url` | `String` | — | Connection URL (alternative to host/port) |
| `.host` | `String` | — | Hostname (alternative to url) |
| `.port` | `String` | — | Port (alternative to url) |
| `.critical` | `boolean` | — | Whether this dependency is critical (required) |
| `.interval` | `Duration` | global | Per-dependency check interval override |
| `.timeout` | `Duration` | global | Per-dependency check timeout override |
| `.labels` | `Map<String, String>` | — | Custom labels added to Prometheus metrics |
| `.tls` | `boolean` | `false` | Enable TLS for the connection |
| `.tls-skip-verify` | `boolean` | `false` | Skip TLS certificate verification |

#### HTTP

| Property | Type | Default | Description |
| --- | --- | --- | --- |
| `.health-path` | `String` | `/` | HTTP health check path |
| `.http-headers` | `Map<String, String>` | — | Custom HTTP headers |
| `.http-bearer-token` | `String` | — | Bearer token for Authorization header |
| `.http-basic-username` | `String` | — | HTTP Basic auth username |
| `.http-basic-password` | `String` | — | HTTP Basic auth password |

#### gRPC

| Property | Type | Default | Description |
| --- | --- | --- | --- |
| `.service-name` | `String` | `""` | gRPC health check service name |
| `.grpc-metadata` | `Map<String, String>` | — | Custom gRPC metadata headers |
| `.grpc-bearer-token` | `String` | — | Bearer token for gRPC call credentials |
| `.grpc-basic-username` | `String` | — | gRPC Basic auth username |
| `.grpc-basic-password` | `String` | — | gRPC Basic auth password |

#### Database (Postgres, MySQL)

| Property | Type | Default | Description |
| --- | --- | --- | --- |
| `.username` | `String` | — | Database username |
| `.password` | `String` | — | Database password |
| `.database` | `String` | — | Database name |
| `.query` | `String` | `SELECT 1` | Custom health check query |

#### Redis

| Property | Type | Default | Description |
| --- | --- | --- | --- |
| `.redis-password` | `String` | — | Redis password |
| `.redis-db` | `int` | `0` | Redis database number |

#### AMQP

| Property | Type | Default | Description |
| --- | --- | --- | --- |
| `.amqp-url` | `String` | — | AMQP connection URL (alternative to host/port) |
| `.amqp-username` | `String` | `guest` | AMQP username |
| `.amqp-password` | `String` | `guest` | AMQP password |
| `.virtual-host` | `String` | `/` | AMQP virtual host |

#### LDAP

| Property | Type | Default | Description |
| --- | --- | --- | --- |
| `.ldap-check-method` | `String` | `root_dse` | Check method: `root_dse`, `bind`, `search` |
| `.ldap-bind-dn` | `String` | — | Bind DN for `bind` or `search` methods |
| `.ldap-bind-password` | `String` | — | Bind password |
| `.ldap-base-dn` | `String` | — | Base DN for `search` method |
| `.ldap-search-filter` | `String` | `(objectClass=*)` | LDAP search filter |
| `.ldap-search-scope` | `String` | `base` | Search scope: `base`, `one`, `sub` |
| `.ldap-start-tls` | `boolean` | `false` | Use STARTTLS extension |
| `.ldap-tls-skip-verify` | `boolean` | `false` | Skip TLS certificate verification for LDAP |

## Actuator Endpoints

### `/actuator/dependencies`

Custom endpoint returning a map of all dependency statuses.

**Request:**

```bash
curl -s http://localhost:8080/actuator/dependencies | jq .
```

**Response:**

```json
{
  "postgres-main:pg.svc:5432": true,
  "redis-cache:redis.svc:6379": true,
  "auth-service:auth.svc:8080": true,
  "user-service:user.svc:9090": false,
  "rabbitmq:rabbitmq.svc:5672": true,
  "kafka:kafka.svc:9092": true,
  "directory:ldap.svc:389": true
}
```

### `/actuator/health`

`DepHealthIndicator` integrates into the standard Spring Boot health
endpoint. The indicator reports `UP` when all **critical** dependencies
are healthy, and `DOWN` otherwise.

**Request:**

```bash
curl -s http://localhost:8080/actuator/health | jq .
```

**Response:**

```json
{
  "status": "UP",
  "components": {
    "dephealth": {
      "status": "UP",
      "details": {
        "postgres-main:pg.svc:5432": "UP",
        "redis-cache:redis.svc:6379": "UP",
        "auth-service:auth.svc:8080": "UP",
        "user-service:user.svc:9090": "DOWN",
        "rabbitmq:rabbitmq.svc:5672": "UP",
        "kafka:kafka.svc:9092": "UP",
        "directory:ldap.svc:389": "UP"
      }
    }
  }
}
```

> Note: `user-service` is DOWN but the overall status is UP because it
> is not marked as `critical`.

### `/actuator/prometheus`

Standard Prometheus metrics endpoint. DepHealth metrics are exported
automatically via the Micrometer `MeterRegistry`.

**Request:**

```bash
curl -s http://localhost:8080/actuator/prometheus | grep app_dependency
```

**Response:**

```text
app_dependency_health{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
app_dependency_health{name="my-service",group="my-team",dependency="redis-cache",type="redis",host="redis.svc",port="6379",critical="no"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",le="0.01"} 42
```

## Actuator Configuration

Ensure the required endpoints are exposed in your `application.yml`:

```yaml
management:
  endpoints:
    web:
      exposure:
        include: health, prometheus, dependencies
```

To show full health details (including the `dephealth` component):

```yaml
management:
  endpoint:
    health:
      show-details: always
  endpoints:
    web:
      exposure:
        include: health, prometheus, dependencies
```

## Custom DepHealth Bean

To override auto-configuration — for example, to integrate with an
existing connection pool — define your own `DepHealth` bean:

```java
@Configuration
public class DepHealthConfig {

    @Bean
    public DepHealth depHealth(MeterRegistry registry, DataSource dataSource) {
        return DepHealth.builder("my-service", "my-team", registry)
            .dependency("postgres-main", DependencyType.POSTGRES, d -> d
                .dataSource(dataSource)
                .critical(true))
            .build();
    }
}
```

When a custom `@Bean` is present, the starter skips its own
`DepHealth` creation but still registers `DepHealthLifecycle`,
`DepHealthIndicator`, and `DependenciesEndpoint` around your bean.

## Environment Variables

Spring Boot natively supports `${VAR_NAME}` placeholders in YAML files.
Use this to keep sensitive values out of configuration:

```yaml
dephealth:
  name: ${DEPHEALTH_NAME:my-service}
  group: ${DEPHEALTH_GROUP:my-team}
  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      username: ${DB_USERNAME}
      password: ${DB_PASSWORD}
      critical: true
    auth-service:
      type: http
      url: ${AUTH_SERVICE_URL:http://auth.svc:8080}
      http-bearer-token: ${API_TOKEN}
      critical: true
```

The `${VAR:default}` syntax provides a default value when the
environment variable is not set.

## See Also

- [Getting Started](getting-started.md) — installation and first example
- [Configuration](configuration.md) — all options, defaults, and environment variables
- [Connection Pools](connection-pools.md) — DataSource, JedisPool, and LDAPConnection integration
- [Troubleshooting](troubleshooting.md) — common issues and solutions
