*[Русская версия](code-style.ru.md)*

# Code Style Guide: Java SDK

This document describes code style conventions for the Java SDK (`sdk-java/`).
See also: [General Principles](../../docs/code-style/overview.md) | [Testing](../../docs/code-style/testing.md)

## Naming Conventions

### Packages

Use reverse domain notation in lowercase:

```java
biz.kryukov.dev.dephealth          // core
biz.kryukov.dev.dephealth.checks   // health checkers
biz.kryukov.dev.dephealth.model    // model classes
biz.kryukov.dev.dephealth.metrics  // metrics exporter
biz.kryukov.dev.dephealth.scheduler // check scheduler
biz.kryukov.dev.dephealth.parser   // config parser
biz.kryukov.dev.dephealth.spring   // Spring Boot integration
```

### Classes and Interfaces

- `PascalCase` for all types
- Interfaces: noun or adjective, no `I` prefix (unlike C#)
- Implementations: descriptive names, not `Impl` suffix

```java
// Good
public interface HealthChecker { }
public class HttpHealthChecker implements HealthChecker { }
public class TcpHealthChecker implements HealthChecker { }

// Bad
public interface IHealthChecker { }     // no I-prefix in Java
public class HttpHealthCheckerImpl { }  // avoid Impl suffix
```

### Methods and Variables

- Methods: `camelCase`, verb-first
- Local variables: `camelCase`
- Constants: `UPPER_SNAKE_CASE`
- Boolean methods: `is`/`has`/`can` prefix

```java
public void check(Endpoint endpoint, Duration timeout) { }
public DependencyType type() { }

private static final Duration DEFAULT_TIMEOUT = Duration.ofSeconds(5);
private static final int MAX_RETRY_COUNT = 3;

public boolean isCritical() { }
public boolean hasEndpoints() { }
```

### Enums

- Type name: `PascalCase` singular
- Values: `UPPER_SNAKE_CASE`

```java
public enum DependencyType {
    HTTP, GRPC, TCP, POSTGRES, MYSQL, REDIS, AMQP, KAFKA, LDAP
}
```

## Package Structure

```text
sdk-java/
├── dephealth-core/
│   └── src/main/java/biz/kryukov/dev/dephealth/
│       ├── DepHealth.java                  // main API, builder
│       ├── model/
│       │   ├── Dependency.java             // model
│       │   ├── Endpoint.java               // model
│       │   ├── DependencyType.java         // enum
│       │   ├── EndpointStatus.java         // health status
│       │   ├── CheckConfig.java            // config model
│       │   ├── CheckResult.java            // check result
│       │   └── StatusCategory.java         // status constants
│       ├── HealthChecker.java              // checker interface
│       ├── checks/
│       │   ├── HttpHealthChecker.java
│       │   ├── GrpcHealthChecker.java
│       │   ├── TcpHealthChecker.java
│       │   ├── PostgresHealthChecker.java
│       │   ├── MysqlHealthChecker.java
│       │   ├── RedisHealthChecker.java
│       │   ├── AmqpHealthChecker.java
│       │   ├── KafkaHealthChecker.java
│       │   └── LdapHealthChecker.java
│       ├── scheduler/
│       │   └── CheckScheduler.java
│       ├── metrics/
│       │   └── MetricsExporter.java
│       ├── parser/
│       │   ├── ConfigParser.java
│       │   └── ParsedConnection.java
│       └── exceptions/
│           ├── CheckException.java         // base exception
│           ├── CheckAuthException.java
│           ├── CheckConnectionException.java
│           ├── UnhealthyException.java
│           ├── ValidationException.java
│           ├── ConfigurationException.java
│           ├── EndpointNotFoundException.java
│           └── DepHealthException.java
│
└── dephealth-spring-boot-starter/
    └── src/main/java/biz/kryukov/dev/dephealth/spring/
        ├── DepHealthAutoConfiguration.java
        ├── DepHealthProperties.java
        ├── DepHealthLifecycle.java
        ├── DepHealthIndicator.java
        └── DependenciesEndpoint.java
```

## Error Handling

### Exception Hierarchy

Use **checked exceptions** for `HealthChecker.check()` (extends `Exception`)
and **unchecked exceptions** for configuration errors. The scheduler catches
all exceptions from checkers.

```java
// CheckException hierarchy (checked)
public class CheckException extends Exception {
    public String statusCategory() { ... }
    public String statusDetail() { ... }
}

public class CheckAuthException extends CheckException { }
public class CheckConnectionException extends CheckException { }

// Configuration exceptions (unchecked)
public class ValidationException extends RuntimeException { }
public class ConfigurationException extends RuntimeException { }
```

### Rules

- Configuration errors: throw immediately during `build()`
- Check failures: throw specific `CheckException` subtypes, caught by the scheduler
- Never swallow exceptions — always log before suppressing
- Always include the cause chain: `new CheckException("msg", cause)`

```java
// Good — informative message with cause
throw new CheckAuthException(
    String.format("Authentication failed for %s:%s",
        endpoint.host(), endpoint.port()),
    cause);

// Bad — loses context
throw new CheckException("auth error");
```

## JavaDoc

### What to Document

- All `public` and `protected` types and members
- All interfaces and their contracts
- Non-obvious behavior, side effects, thread safety guarantees

### Format

```java
/**
 * Performs a health check against the given endpoint.
 *
 * <p>Implementations must be thread-safe.</p>
 *
 * @param endpoint the endpoint to check
 * @param timeout  maximum wait time
 * @throws CheckException if the check fails
 */
void check(Endpoint endpoint, Duration timeout) throws Exception;
```

Rules:

- First sentence: summary in English (shown in IDE tooltips)
- Use `@param`, `@return`, `@throws` for all parameters, return values, and exceptions
- Use `{@code}` for inline code, `{@link}` for cross-references
- Thread safety guarantees in `<p>` block

## Builder Pattern

Use the builder pattern for `DepHealth` configuration:

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .checkInterval(Duration.ofSeconds(15))
    .timeout(Duration.ofSeconds(5))
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true))
    .build();

dh.start();
```

Rules:

- Builder is the **only** way to create `DepHealth`
- Builder methods return `this` for chaining
- `build()` validates all parameters and returns an immutable object
- `start()` is separate from `build()` — allows inspection before starting

## Immutability and Null Safety

- **Prefer immutable objects**: `Dependency`, `Endpoint`, `CheckConfig`, `EndpointStatus` — all immutable after creation
- **No `null` in public API**: use `Optional<T>` for optional return values, `Boolean` (nullable) for unknown state
- **Validate parameters**: use `Objects.requireNonNull()` at method entry

```java
// Good — immutable model with validation
public final class Endpoint {
    public Endpoint(String host, String port) {
        this.host = Objects.requireNonNull(host, "host must not be null");
        this.port = Objects.requireNonNull(port, "port must not be null");
    }
}
```

- Use `final` for fields that should not change
- Use `Collections.unmodifiableMap()` or `Map.copyOf()` for collections

## Logging

Use SLF4J with parameterized messages:

```java
private static final Logger log = LoggerFactory.getLogger(CheckScheduler.class);

// Good — parameterized (lazy evaluation)
log.info("Starting check scheduler, {} dependencies", dependencies.size());
log.warn("Check {} failed: {}", dependency.name(), error.getMessage());
log.debug("Check {} completed in {}ms", dependency.name(), elapsed);

// Bad — string concatenation
log.info("Starting scheduler for " + dependencies.size() + " dependencies");
```

- Use `log.isDebugEnabled()` only for expensive-to-compute messages
- Never log credentials — sanitize URLs before logging

## Linters

### Checkstyle

Configuration: `sdk-java/dephealth-core/checkstyle.xml` (Google-based with
project modifications).

Key rules enforced:

- Indentation: 4 spaces (no tabs)
- Max line length: not enforced (IDE wrapping)
- Import order: static first, then `java`, `javax`, third-party, project
- No wildcard imports
- Braces required for all `if`/`else`/`for`/`while` blocks

### SpotBugs

Detects common bugs: null pointer dereferences, resource leaks, concurrency
issues.

### Running

```bash
cd sdk-java && make lint    # runs both Checkstyle and SpotBugs in Docker
cd sdk-java && make fmt     # Checkstyle only
```

## Additional Conventions

- **Java version**: 21 LTS — use records, sealed classes, pattern matching where appropriate
- **Dependencies**: minimize external dependencies in `dephealth-core`
- **Metrics**: use Micrometer for metric registration
- **Thread safety**: document thread safety guarantees on every public class
- **Resource management**: use try-with-resources for `Closeable`/`AutoCloseable`
- **Daemon threads**: scheduler uses daemon threads, won't prevent JVM shutdown

## See Also

- [General Principles](../../docs/code-style/overview.md) — cross-SDK code style
- [Testing](../../docs/code-style/testing.md) — testing conventions
