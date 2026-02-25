# dephealth

SDK for monitoring microservice dependencies via Prometheus metrics.

## Features

- Automatic health checking for dependencies (PostgreSQL, MySQL, Redis, RabbitMQ, Kafka, HTTP, gRPC, TCP, LDAP)
- Prometheus metrics export: `app_dependency_health` (Gauge 0/1), `app_dependency_latency_seconds` (Histogram), `app_dependency_status` (enum), `app_dependency_status_detail` (info)
- Java 21 LTS, Maven multi-module project
- Spring Boot starter with auto-configuration and actuator integration
- Connection pool support (preferred) and standalone checks

## Installation

### Maven

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-core</artifactId>
    <version>0.8.0</version>
</dependency>
```

### Spring Boot

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.8.0</version>
</dependency>
```

## Quick Start

```java
import biz.kryukov.dev.dephealth.DepHealth;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;

var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

var dh = DepHealth.builder("my-service", "my-team", registry)
    .postgres("db", "postgresql://user:pass@localhost:5432/mydb", true)
    .redis("cache", "redis://localhost:6379", false)
    .build();

dh.start();
// Metrics are available via Micrometer registry
dh.stop();
```

## Dynamic Endpoints

Add, remove, or replace monitored endpoints at runtime on a running instance
(v0.6.0+):

```java
import biz.kryukov.dev.dephealth.model.DependencyType;
import biz.kryukov.dev.dephealth.model.Endpoint;
import biz.kryukov.dev.dephealth.checks.HttpHealthChecker;

// After dh.start()...

// Add a new endpoint
dh.addEndpoint("api-backend", DependencyType.HTTP, true,
    new Endpoint("backend-2.svc", "8080"),
    HttpHealthChecker.builder().build());

// Remove an endpoint (cancels scheduled task, deletes metrics)
dh.removeEndpoint("api-backend", "backend-2.svc", "8080");

// Replace an endpoint atomically
dh.updateEndpoint("api-backend", "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    HttpHealthChecker.builder().build());
```

See [migration guide](../docs/migration/sdk-java-v050-to-v060.md) for details.

## Health Details

```java
var details = dh.healthDetails();
details.forEach((key, ep) ->
    System.out.printf("%s: healthy=%s status=%s latency=%.1fms%n",
        key, ep.isHealthy(), ep.getStatus(), ep.getLatencyMillis()));
```

## Supported Dependencies

| Type | URL Format |
| --- | --- |
| PostgreSQL | `postgresql://user:pass@host:5432/db` |
| MySQL | `mysql://user:pass@host:3306/db` |
| Redis | `redis://host:6379` |
| RabbitMQ | `amqp://user:pass@host:5672/vhost` |
| Kafka | `kafka://host1:9092,host2:9092` |
| HTTP | `http://host:8080/health` |
| gRPC | via `host()` + `port()` |
| TCP | `tcp://host:port` |
| LDAP | `ldap://host:389` or `ldaps://host:636` |

## LDAP Checker

LDAP health checker supports four check methods and multiple TLS modes:

```java
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker;
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker.CheckMethod;
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker.LdapSearchScope;

// RootDSE check (default)
var checker = LdapHealthChecker.builder()
    .checkMethod(CheckMethod.ROOT_DSE)
    .build();

// Simple bind with credentials
var checker = LdapHealthChecker.builder()
    .checkMethod(CheckMethod.SIMPLE_BIND)
    .bindDN("cn=monitor,dc=corp,dc=com")
    .bindPassword("secret")
    .useTLS(true)
    .build();

// Search with StartTLS
var checker = LdapHealthChecker.builder()
    .checkMethod(CheckMethod.SEARCH)
    .baseDN("dc=example,dc=com")
    .searchFilter("(objectClass=organizationalUnit)")
    .searchScope(LdapSearchScope.ONE)
    .startTLS(true)
    .build();
```

Check methods: `ANONYMOUS_BIND`, `SIMPLE_BIND`, `ROOT_DSE` (default), `SEARCH`.

## Authentication

HTTP and gRPC checkers support Bearer token, Basic Auth, and custom headers/metadata:

```java
dh.http("secure-api", "http://api.svc:8080", true,
    HttpHealthChecker.builder().bearerToken("eyJhbG...").build());

dh.grpc("grpc-backend", "backend.svc", "9090", true,
    GrpcHealthChecker.builder().bearerToken("eyJhbG...").build());
```

See [quickstart guide](../docs/quickstart/java.md#authentication) for all options.

## License

Apache License 2.0 — see [LICENSE](https://github.com/BigKAA/topologymetrics/blob/master/LICENSE).
