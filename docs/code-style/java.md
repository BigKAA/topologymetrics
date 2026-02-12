*[Русская версия](java.ru.md)*

# Code Style Guide: Java SDK

This document describes code style conventions for the Java SDK (`sdk-java/`).
See also: [General Principles](overview.md) | [Testing](testing.md)

## Naming Conventions

### Packages

Use reverse domain notation in lowercase:

```java
biz.kryukov.dev.dephealth          // core
biz.kryukov.dev.dephealth.checks   // health checkers
biz.kryukov.dev.dephealth.spring   // Spring Boot integration
```

### Classes and Interfaces

- `PascalCase` for all types
- Interfaces: noun or adjective, no `I` prefix (unlike C#)
- Implementations: descriptive names, not `Impl` suffix

```java
// Good
public interface HealthChecker { }
public class HttpChecker implements HealthChecker { }
public class TcpChecker implements HealthChecker { }

// Bad
public interface IHealthChecker { }     // no I-prefix in Java
public class HttpCheckerImpl { }        // avoid Impl suffix
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
    HTTP, GRPC, TCP, POSTGRES, MYSQL, REDIS, AMQP, KAFKA
}
```

## Package Structure

```text
sdk-java/
├── dephealth-core/
│   └── src/main/java/biz/kryukov/dev/dephealth/
│       ├── DependencyHealth.java          // main API, builder
│       ├── Dependency.java                // model
│       ├── Endpoint.java                  // model
│       ├── DependencyType.java            // enum
│       ├── HealthChecker.java             // checker interface
│       ├── CheckScheduler.java            // scheduler
│       ├── ConnectionParser.java          // URL/params parser
│       ├── PrometheusExporter.java        // metrics
│       ├── DepHealthException.java        // base exception
│       └── checks/
│           ├── HttpChecker.java
│           ├── GrpcChecker.java
│           ├── TcpChecker.java
│           ├── PostgresChecker.java
│           ├── RedisChecker.java
│           ├── AmqpChecker.java
│           └── KafkaChecker.java
│
└── dephealth-spring-boot-starter/
    └── src/main/java/biz/kryukov/dev/dephealth/spring/
        ├── DepHealthAutoConfiguration.java
        └── DepHealthProperties.java
```

## Error Handling

### Exception Hierarchy

Use **unchecked exceptions** (extending `RuntimeException`). Checked exceptions create
unnecessary boilerplate for library users.

```java
public class DepHealthException extends RuntimeException {
    public DepHealthException(String message) { super(message); }
    public DepHealthException(String message, Throwable cause) { super(message, cause); }
}

public class CheckTimeoutException extends DepHealthException { }
public class ConnectionRefusedException extends DepHealthException { }
```

### Rules

- Configuration errors: throw `IllegalArgumentException` or `DepHealthException` immediately
- Check failures: throw specific exception subtypes, caught by the scheduler
- Never swallow exceptions — always log before suppressing
- Always include the cause chain: `new DepHealthException("msg", cause)`

```java
// Good — informative message with cause
throw new CheckTimeoutException(
    String.format("Health check timed out for %s:%d after %s",
        endpoint.host(), endpoint.port(), timeout),
    cause);

// Bad — loses context
throw new DepHealthException("timeout");
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
 * @throws CheckTimeoutException if the check did not complete within timeout
 * @throws ConnectionRefusedException if the connection was refused
 */
void check(Endpoint endpoint, Duration timeout);
```

Rules:

- First sentence: summary in English (shown in IDE tooltips)
- Use `@param`, `@return`, `@throws` for all parameters, return values, and exceptions
- Use `{@code}` for inline code, `{@link}` for cross-references
- Thread safety guarantees in `<p>` block

## Builder Pattern

Use the builder pattern for `DependencyHealth` configuration:

```java
DependencyHealth health = DependencyHealth.builder("order-service")
    .dependency("postgres-main", DependencyType.POSTGRES,
        Endpoint.fromUrl(System.getenv("DATABASE_URL")),
        DependencyOptions.builder().critical(true).build())
    .dependency("redis-cache", DependencyType.REDIS,
        Endpoint.fromUrl(System.getenv("REDIS_URL")))
    .checkInterval(Duration.ofSeconds(15))
    .timeout(Duration.ofSeconds(5))
    .build();

health.start();
```

Rules:

- Builder is the **only** way to create `DependencyHealth`
- Builder methods return `this` for chaining
- `build()` validates all parameters and returns an immutable object
- `start()` is separate from `build()` — allows inspection before starting

## Immutability and Null Safety

- **Prefer immutable objects**: `Dependency`, `Endpoint`, configuration — all immutable after creation
- **No `null` in public API**: use `Optional<T>` for optional return values
- **Validate parameters**: use `Objects.requireNonNull()` at method entry

```java
// Good — immutable model
public record Endpoint(String host, int port, Map<String, String> metadata) {
    public Endpoint {
        Objects.requireNonNull(host, "host must not be null");
        if (port <= 0 || port > 65535) {
            throw new IllegalArgumentException("port must be 1-65535, got: " + port);
        }
        metadata = Map.copyOf(metadata); // defensive copy
    }
}
```

- Use `final` for fields that should not change
- Use `Collections.unmodifiable*()` or `Map.copyOf()` for collections in public API

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

Configuration: `sdk-java/checkstyle.xml` (Google-based with project modifications).

Key rules enforced:

- Indentation: 4 spaces (no tabs)
- Max line length: not enforced (IDE wrapping)
- Import order: static first, then `java`, `javax`, third-party, project
- No wildcard imports
- Braces required for all `if`/`else`/`for`/`while` blocks

### SpotBugs

Detects common bugs: null pointer dereferences, resource leaks, concurrency issues.

### Running

```bash
cd sdk-java && make lint    # runs both Checkstyle and SpotBugs in Docker
cd sdk-java && make fmt     # auto-format with google-java-format
```

## Additional Conventions

- **Java version**: 21 LTS — use records, sealed classes, pattern matching where appropriate
- **Dependencies**: minimize external dependencies in `dephealth-core`
- **Metrics**: use Micrometer for metric registration
- **Thread safety**: document thread safety guarantees on every public class
  (`@ThreadSafe`, `@NotThreadSafe`, or comment)
- **Resource management**: use try-with-resources for `Closeable`/`AutoCloseable`
