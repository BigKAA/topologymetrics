*[Русская версия](connection-pools.ru.md)*

# Connection Pool Integration

dephealth supports two modes for checking dependencies:

- **Standalone mode** — SDK creates a new connection for each health check
- **Pool mode** — SDK uses the existing connection pool of your service

Pool mode is preferred because it reflects the actual ability of the service
to work with the dependency. If the connection pool is exhausted, the health
check will detect it.

## Standalone vs Pool Mode

| Aspect | Standalone | Pool |
| --- | --- | --- |
| Connection | New per check | Reuses existing pool |
| Reflects real health | Partially | Yes |
| Setup | Simple — just URL | Requires passing pool object |
| External dependencies | None (uses checker's driver) | Your application's driver |
| Detects pool exhaustion | No | Yes |

## PostgreSQL via DataSource

Pass your existing `DataSource` (HikariCP, Tomcat JDBC, etc.) to the
dependency builder:

```java
import javax.sql.DataSource;

DataSource dataSource = ...; // existing pool from the application

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .dataSource(dataSource)
        .critical(true))
    .build();
```

When using `dataSource()`, the host and port are extracted from the
DataSource metadata. You can also provide them explicitly if needed:

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .dataSource(dataSource)
    .host("pg.svc")
    .port("5432")
    .critical(true))
```

The checker executes `SELECT 1` (default) on a connection borrowed from
the pool. Custom query via `.dbQuery()`:

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .dataSource(dataSource)
    .dbQuery("SELECT 1 FROM pg_stat_activity LIMIT 1")
    .critical(true))
```

## MySQL via DataSource

Same approach as PostgreSQL — pass a `DataSource`:

```java
DataSource dataSource = ...; // HikariCP, Tomcat JDBC, etc.

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("mysql-main", DependencyType.MYSQL, d -> d
        .dataSource(dataSource)
        .critical(true))
    .build();
```

## Redis via JedisPool

Pass your existing `JedisPool`:

```java
import redis.clients.jedis.JedisPool;

JedisPool jedisPool = ...; // existing pool from the application

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .jedisPool(jedisPool)
        .critical(false))
    .build();
```

The checker borrows a `Jedis` connection from the pool, sends `PING`,
and returns it. Host and port are extracted from the pool configuration.

## LDAP via LDAPConnection

Pass your existing UnboundID `LDAPConnection`:

```java
import com.unboundid.ldap.sdk.LDAPConnection;

LDAPConnection conn = ...; // existing connection

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("directory", DependencyType.LDAP, d -> d
        .ldapConnection(conn)
        .ldapCheckMethod("root_dse")
        .critical(true))
    .build();
```

When using an existing connection, TLS settings (startTLS, useTLS) are
managed by the connection itself, not by the checker.

## Direct Checker Pool Mode

If you need more control, you can create checkers directly with pool
objects and register them via `addEndpoint()`:

### PostgreSQL

```java
import biz.kryukov.dev.dephealth.checks.PostgresHealthChecker;
import biz.kryukov.dev.dephealth.model.Endpoint;

DataSource dataSource = ...;

var checker = PostgresHealthChecker.builder()
    .dataSource(dataSource)
    .query("SELECT 1")
    .build();

// After dh.start()
dh.addEndpoint("postgres-main", DependencyType.POSTGRES, true,
    new Endpoint("pg.svc", "5432"),
    checker);
```

### Redis

```java
import biz.kryukov.dev.dephealth.checks.RedisHealthChecker;

JedisPool jedisPool = ...;

var checker = RedisHealthChecker.builder()
    .jedisPool(jedisPool)
    .build();

dh.addEndpoint("redis-cache", DependencyType.REDIS, false,
    new Endpoint("redis.svc", "6379"),
    checker);
```

## Standalone vs Pool: When to Use Which

| Use case | Recommendation |
| --- | --- |
| Standard setup, one pool per dependency | Pool mode via `dataSource()` / `jedisPool()` |
| No existing pool (external services) | Standalone via `url()` |
| HTTP and gRPC services | Standalone only (no pool needed) |
| Custom health check query | Pool mode with `.dbQuery()` |
| LDAP with managed connection lifecycle | Pool mode via `.ldapConnection()` |

## Full Example: Mixed Modes

```java
import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.model.DependencyType;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;
import javax.sql.DataSource;
import redis.clients.jedis.JedisPool;

public class Main {
    public static void main(String[] args) {
        var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

        // Existing connection pools
        DataSource dataSource = ...; // HikariCP
        JedisPool jedisPool = ...;

        var dh = DepHealth.builder("my-service", "my-team", registry)
            // Pool mode — PostgreSQL
            .dependency("postgres-main", DependencyType.POSTGRES, d -> d
                .dataSource(dataSource)
                .critical(true))

            // Pool mode — Redis
            .dependency("redis-cache", DependencyType.REDIS, d -> d
                .jedisPool(jedisPool)
                .critical(false))

            // Standalone mode — HTTP (no pool needed)
            .dependency("payment-api", DependencyType.HTTP, d -> d
                .url("http://payment.svc:8080")
                .critical(true))

            .build();

        dh.start();
        Runtime.getRuntime().addShutdownHook(new Thread(dh::stop));
    }
}
```

## Spring Boot Pool Integration

When using Spring Boot, you can define a custom `DepHealth` bean that
uses auto-configured connection pools:

```java
@Configuration
public class DepHealthConfig {
    @Bean
    public DepHealth depHealth(MeterRegistry registry, DataSource dataSource) {
        return DepHealth.builder("my-service", "my-team", registry)
            .dependency("postgres-main", DependencyType.POSTGRES, d -> d
                .dataSource(dataSource)
                .critical(true))
            .dependency("auth-service", DependencyType.HTTP, d -> d
                .url("http://auth.svc:8080")
                .critical(true))
            .build();
    }
}
```

This overrides the YAML-based auto-configuration but still gets
lifecycle management and actuator integration from the starter.

## See Also

- [Checkers](checkers.md) — all checker details including pool options
- [Spring Boot Integration](spring-boot.md) — auto-configuration with pools
- [Configuration](configuration.md) — DataSource and JedisPool options
- [API Reference](api-reference.md) — builder methods reference
- [Troubleshooting](troubleshooting.md) — common issues and solutions
