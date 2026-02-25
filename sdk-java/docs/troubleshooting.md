*[Русская версия](troubleshooting.ru.md)*

# Troubleshooting

Common issues and solutions when using the dephealth Java SDK.

## Empty Metrics / No Metrics Exported

**Symptom:** The `/metrics` or `/actuator/prometheus` endpoint returns no
`app_dependency_*` metrics.

**Possible causes:**

1. **`start()` not called.** Metrics are only registered and updated after
   `dh.start()` is called. Check that `start()` is invoked and returns
   without error.

   For Spring Boot: verify that `dephealth-spring-boot-starter` is on
   the classpath — it calls `start()` automatically via `DepHealthLifecycle`.

2. **Wrong Prometheus endpoint.** For Spring Boot, metrics are at
   `/actuator/prometheus`, not `/metrics`. Ensure the endpoint is exposed:

   ```yaml
   management:
     endpoints:
       web:
         exposure:
           include: health, prometheus, dependencies
   ```

3. **Micrometer registry not connected.** For programmatic API, verify
   that you pass the same `MeterRegistry` to both `DepHealth.builder()`
   and your metrics endpoint:

   ```java
   var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);
   var dh = DepHealth.builder("my-service", "my-team", registry)
       // ...
       .build();
   dh.start();

   // Same registry for /metrics handler
   String metrics = registry.scrape();
   ```

## All Dependencies Show 0 (Unhealthy)

**Symptom:** `app_dependency_health` is `0` for all dependencies.

**Possible causes:**

1. **Network accessibility** — verify the target services are reachable
   from the service container/pod.

2. **DNS resolution** — check that service names resolve correctly.

3. **Wrong URL/host/port** — double-check configuration values.

4. **Timeout too low** — default is 5s. For slow dependencies, increase:

   ```java
   .dependency("slow-db", DependencyType.POSTGRES, d -> d
       .url(System.getenv("DATABASE_URL"))
       .timeout(Duration.ofSeconds(10))
       .critical(true))
   ```

5. **Debug logging** — enable SDK debug output:

   ```yaml
   # Spring Boot
   logging:
     level:
       biz.kryukov.dev.dephealth: DEBUG
   ```

   ```java
   // Programmatic — SLF4J is used internally
   // Configure your SLF4J implementation to DEBUG for biz.kryukov.dev.dephealth
   ```

## High Latency for PostgreSQL/MySQL Checks

**Symptom:** `app_dependency_latency_seconds` shows high values (100ms+)
for database checks.

**Cause:** Standalone mode creates a new JDBC connection for each check.
This includes TCP handshake, TLS negotiation, and authentication.

**Solution:** Use connection pool integration:

```java
// Instead of
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://user:pass@pg.svc:5432/mydb")
    .critical(true))

// Use existing DataSource
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .dataSource(dataSource)
    .critical(true))
```

See [Connection Pools](connection-pools.md) for details.

## gRPC: DEADLINE_EXCEEDED Error

**Symptom:** gRPC checks fail with timeout or show high latency.

**Possible causes:**

1. **gRPC service not accessible** at the specified address.

2. **Service does not implement** `grpc.health.v1.Health/Check` — the
   gRPC Health Checking Protocol must be enabled on the target service.

3. **Use `host()` + `port()`**, not `url()` for gRPC:

   ```java
   .dependency("user-service", DependencyType.GRPC, d -> d
       .host("user.svc")
       .port("9090")
       .critical(true))
   ```

4. **TLS mismatch** — if the service uses TLS, set `.grpcTls(true)`.

5. **DNS resolution in Kubernetes** — use FQDN with trailing dot to avoid
   search domain iteration:

   ```java
   .host("user-service.namespace.svc.cluster.local.")
   ```

## Connection Refused Errors

**Symptom:** `app_dependency_status{status="connection_error"}` is `1`.

**Possible causes:**

1. **Service not running** — verify the target service is up and listening
   on the expected host and port.

2. **Wrong host or port** — double-check the URL or host/port values.

3. **Kubernetes network policies** — ensure traffic is allowed from the
   checker pod to the target service.

4. **Firewall rules** — on non-Kubernetes setups, check firewall rules.

## Timeout Errors

**Symptom:** `app_dependency_status{status="timeout"}` is `1`.

**Possible causes:**

1. **Default timeout too low.** Default is 5s. Increase for slow deps:

   ```java
   // Global
   .timeout(Duration.ofSeconds(10))

   // Per-dependency
   .dependency("slow-service", DependencyType.HTTP, d -> d
       .url("http://slow.svc:8080")
       .timeout(Duration.ofSeconds(10))
       .critical(true))
   ```

2. **Network latency** — check round-trip time to the target service.

3. **Target overloaded** — the service may be too slow to respond.

## Unexpected Auth Errors

**Symptom:** `app_dependency_status{status="auth_error"}` is `1` when
credentials should be valid.

**Possible causes:**

1. **Credentials not set or incorrect** — verify token or username/password:

   ```java
   .httpBearerToken(System.getenv("API_TOKEN"))
   .grpcBearerToken(System.getenv("GRPC_TOKEN"))
   ```

2. **Token expired** — bearer tokens have a limited lifetime.

3. **Wrong auth method** — some services expect Basic auth, not Bearer.

4. **Database credentials** — for PostgreSQL, MySQL, and AMQP, verify
   credentials are correct in the URL:

   ```java
   .url("postgresql://user:password@host:5432/dbname")
   ```

See [Authentication](authentication.md) for all auth options.

## AMQP: Connection Error to RabbitMQ

**Symptom:** AMQP checker fails to connect.

**Important**: path `/` in URL means vhost `/` (not empty).

```java
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .host("rabbitmq.svc")
    .port("5672")
    .amqpUsername("user")
    .amqpPassword("pass")
    .amqpVirtualHost("/")
    .critical(false))
```

## LDAP: Configuration Errors

**Symptom:** LDAP checker throws `ConfigurationException` on startup.

**Common causes:**

1. **`simple_bind` without credentials:**

   ```java
   // Wrong — missing bindDN and bindPassword
   .ldapCheckMethod("simple_bind")

   // Correct
   .ldapCheckMethod("simple_bind")
   .ldapBindDN("cn=monitor,dc=corp,dc=com")
   .ldapBindPassword("secret")
   ```

2. **`search` without baseDN:**

   ```java
   // Wrong — missing baseDN
   .ldapCheckMethod("search")

   // Correct
   .ldapCheckMethod("search")
   .ldapBaseDN("dc=example,dc=com")
   ```

3. **startTLS with ldaps://** — these are incompatible:

   ```java
   // Wrong — cannot use both
   .url("ldaps://ldap.svc:636")
   .ldapStartTLS(true)

   // Correct — use one or the other
   .url("ldaps://ldap.svc:636")        // implicit TLS
   // OR
   .url("ldap://ldap.svc:389")
   .ldapStartTLS(true)                 // upgrade to TLS
   ```

## Custom Labels Not Appearing

**Symptom:** Labels added via `.label()` are not visible in metrics.

**Possible causes:**

1. **Invalid label name.** Must match `[a-zA-Z_][a-zA-Z0-9_]*` and not be
   a reserved name.

   Reserved: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`.

2. **Inconsistent labels across dependencies.** If custom labels are used,
   all endpoints must use the same label names. The SDK collects all label
   names from all dependencies and applies them to all metrics.

## health() Returns Empty Map

**Symptom:** `dh.health()` returns an empty map immediately after `start()`.

**Cause:** The first check has not completed yet. There is an initial delay
(default 5s) before the first check runs.

**Solution:** Use `healthDetails()`. Before the first check completes, it
returns entries with `healthy = null` and `status = "unknown"`:

```java
var details = dh.healthDetails();
details.forEach((key, ep) -> {
    if (ep.isHealthy() == null) {
        System.out.printf("%s: not yet checked%n", key);
    } else {
        System.out.printf("%s: healthy=%s%n", key, ep.isHealthy());
    }
});
```

## Spring Boot: Metrics Not at /actuator/prometheus

**Check:**

1. Dependency `spring-boot-starter-actuator` is present
2. `management.endpoints.web.exposure.include` includes `prometheus`
3. `dephealth-spring-boot-starter` is in classpath
4. `io.micrometer:micrometer-registry-prometheus` is on classpath

## See Also

- [Getting Started](getting-started.md) — installation and basic setup
- [Configuration](configuration.md) — all options, defaults, and validation rules
- [Checkers](checkers.md) — detailed guide for all 9 checkers
- [Metrics](metrics.md) — Prometheus metrics reference and PromQL examples
- [Authentication](authentication.md) — auth options for HTTP, gRPC, and databases
- [Connection Pools](connection-pools.md) — connection pool integration
- [Spring Boot Integration](spring-boot.md) — auto-configuration details
