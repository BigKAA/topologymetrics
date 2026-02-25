*[Русская версия](troubleshooting.ru.md)*

# Troubleshooting

Common issues and solutions for the dephealth Python SDK.

## No Metrics at /metrics

**Symptoms:** `/metrics` returns empty or no dephealth metrics appear.

**Check:**

1. `DepHealthMiddleware` is added to the FastAPI application:

   ```python
   app.add_middleware(DepHealthMiddleware)
   ```

2. `dephealth_lifespan()` is correctly set as `lifespan`:

   ```python
   app = FastAPI(lifespan=dephealth_lifespan("name", "group", ...))
   ```

3. Application started without errors (check logs)

4. Wait at least one check interval (default 15s) for the first check to complete

5. If using a custom registry, ensure the middleware uses the same registry:

   ```python
   custom_registry = CollectorRegistry()
   dh = DependencyHealth("name", "group", ..., registry=custom_registry)
   app.add_middleware(DepHealthMiddleware, registry=custom_registry)
   ```

## All Dependencies Show Unhealthy (0)

**Symptoms:** `app_dependency_health` is `0` for all dependencies.

**Check:**

1. **Network access**: dependencies are reachable from the container/pod

   ```bash
   # From inside the container
   curl http://payment.svc:8080/health
   nc -zv pg.svc 5432
   ```

2. **DNS resolution**: service names resolve correctly

3. **URL/host/port**: configuration is correct

4. **Timeout**: default 5s may be insufficient for slow dependencies.
   Increase with `timeout=timedelta(seconds=10)`

5. **Logs**: enable debug logging for details:

   ```python
   import logging
   logging.basicConfig(level=logging.DEBUG)
   ```

## High Latency for Database Checks

**Symptoms:** `app_dependency_latency_seconds` shows high values for PostgreSQL/MySQL.

**Cause:** standalone mode creates a new connection per check, including
TCP handshake, TLS negotiation, and authentication.

**Solution:** use pool integration:

```python
# Instead of
postgres_check("db", url="postgresql://...", critical=True)

# Use
pg_pool = await asyncpg.create_pool("postgresql://...")
postgres_check("db", pool=pg_pool, critical=True)
```

This eliminates connection establishment overhead. See
[Connection Pools](connection-pools.md) for details.

## gRPC: context deadline exceeded

**Symptoms:** gRPC checks fail with timeout/deadline exceeded.

**Check:**

1. gRPC service is accessible at the specified address

2. Service implements `grpc.health.v1.Health/Check`

3. Use `host` + `port`, not `url` for gRPC:

   ```python
   # Correct
   grpc_check("grpc-svc", host="grpc.svc", port="9090", critical=True)

   # May not work
   grpc_check("grpc-svc", url="grpc.svc:9090", critical=True)
   ```

4. If TLS is needed: `grpc_check(..., tls=True)`

5. Increase timeout if the service is slow:

   ```python
   grpc_check("grpc-svc", host="grpc.svc", port="9090",
       critical=True, timeout=timedelta(seconds=10))
   ```

## Connection Refused Errors

**Symptoms:** `app_dependency_status{status="connection_error"} == 1`

**Check:**

1. The dependency is running and listening on the expected port
2. Firewall rules allow the connection
3. In Kubernetes: service and pod selectors are correct
4. Port matches the dependency's actual listening port

## Timeout Errors

**Symptoms:** `app_dependency_status{status="timeout"} == 1`

**Check:**

1. Network latency between service and dependency
2. Dependency is under heavy load
3. Default timeout (5s) may be too short — increase per-dependency:

   ```python
   postgres_check("slow-db", url="...", critical=True,
       timeout=timedelta(seconds=15))
   ```

4. DNS resolution may be slow — check DNS configuration

## Authentication Errors

**Symptoms:** `app_dependency_status{status="auth_error"} == 1`

**Check:**

1. Credentials are correct and not expired
2. Bearer tokens are valid and not expired
3. Database user has required permissions
4. Redis password matches server configuration
5. AMQP vhost is accessible with the provided credentials

## AMQP: Connection Error to RabbitMQ

**Symptoms:** AMQP checks fail with connection errors.

**Provide the full URL with all components:**

```python
amqp_check("rabbitmq",
    url="amqp://user:pass@rabbitmq.svc:5672/vhost",
    critical=False,
)
```

Common issues:

- Missing vhost in URL (use `/` for default)
- Wrong port (5672 for AMQP, 5671 for AMQPS)
- URL encoding needed for special characters in password

## LDAP Configuration Errors

**Symptoms:** LDAP checks fail immediately with `ValueError`.

**Common causes:**

1. `SIMPLE_BIND` without credentials:

   ```python
   # Error: LDAP simple_bind requires bind_dn and bind_password
   ldap_check("ldap", url="ldap://...", check_method="SIMPLE_BIND",
       critical=True)

   # Fix: provide bind_dn and bind_password
   ldap_check("ldap", url="ldap://...", check_method="SIMPLE_BIND",
       bind_dn="cn=admin,dc=corp", bind_password="secret", critical=True)
   ```

2. `SEARCH` without `base_dn`:

   ```python
   # Error: LDAP search requires base_dn
   ldap_check("ldap", url="ldap://...", check_method="SEARCH",
       critical=True)

   # Fix: provide base_dn
   ldap_check("ldap", url="ldap://...", check_method="SEARCH",
       base_dn="dc=example,dc=com", critical=True)
   ```

3. `start_tls` with `ldaps://`:

   ```python
   # Error: start_tls and ldaps:// are incompatible
   ldap_check("ldap", url="ldaps://ldap.svc:636",
       start_tls=True, critical=True)

   # Fix: use one or the other
   ldap_check("ldap", url="ldaps://ldap.svc:636", critical=True)
   ldap_check("ldap", url="ldap://ldap.svc:389",
       start_tls=True, critical=True)
   ```

## Custom Labels Not Appearing

**Check:**

1. Labels are passed as a dict:

   ```python
   postgres_check("db", url="...", critical=True,
       labels={"role": "primary"})
   ```

2. Label names are valid: `[a-zA-Z_][a-zA-Z0-9_]*`

3. Label names don't use reserved names: `name`, `group`, `dependency`,
   `type`, `host`, `port`, `critical`

## health() Returns Empty Dict

**Check:**

1. `start()` or `start_sync()` was called before `health()`
2. At least one check interval has passed
3. Dependencies were registered (not an empty `DependencyHealth("name", "group")`)

## Dependency Naming Errors

Names must follow the rules:

- Length: 1-63 characters
- Format: `[a-z][a-z0-9-]*` (lowercase letters, digits, hyphens)
- Must start with a letter

Valid: `postgres-main`, `redis-cache`, `auth-service`

Invalid: `Postgres`, `redis_cache`, `123-service`, `-invalid`

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Configuration](configuration.md) — all options, defaults, and validation
- [Checkers](checkers.md) — all 9 built-in checkers
- [Connection Pools](connection-pools.md) — pool integration guide
- [Metrics](metrics.md) — Prometheus metrics reference
