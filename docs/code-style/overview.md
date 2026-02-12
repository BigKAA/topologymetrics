*[Русская версия](overview.ru.md)*

# Code Style Guide: General Principles

This document describes the cross-language principles and conventions used in all dephealth SDKs.
Language-specific guides: [Java](java.md) | [Go](go.md) | [Python](python.md) | [C#](csharp.md) | [Testing](testing.md)

## Philosophy

dephealth is a set of **native SDKs** — each language has its own idiomatic implementation
unified by a common [specification](../../spec/specification.md).
Not a sidecar, not FFI — deep integration with each language's ecosystem.

Key principles:

- **Idiomatic code** — each SDK follows the conventions of its language, not a shared "meta-style"
- **Specification is the source of truth** — metric names, labels, check behavior, and configuration
  contracts are defined in `spec/` and must be identical across all SDKs
- **Minimal public API** — expose only what users need, hide implementation details
- **Zero surprises** — sensible defaults, predictable behavior, informative error messages

## Language Conventions

| Aspect | English | Russian |
| --- | --- | --- |
| Code (variables, functions, classes) | Yes | No |
| Comments in code | Yes | No |
| Documentation files | Yes (primary) | Yes (`.ru.md` suffix) |
| Commit messages | Yes | No |
| Logs at runtime | Yes | - |

## Architectural Layers

Every SDK consists of 6 layers. Code should be organized to reflect this structure:

```text
┌─────────────────────────────────────────────┐
│         Framework Integration               │  Spring Boot / ASP.NET / FastAPI
├─────────────────────────────────────────────┤
│         Metrics Exporter                    │  Prometheus gauges + histograms
├─────────────────────────────────────────────┤
│         Check Scheduler                     │  Periodic health checks
├─────────────────────────────────────────────┤
│         Health Checkers                     │  HTTP, gRPC, TCP, Postgres, Redis, ...
├─────────────────────────────────────────────┤
│         Connection Config Parser            │  URL / params / connection string
├─────────────────────────────────────────────┤
│         Core Abstractions                   │  Dependency, Endpoint, HealthChecker
└─────────────────────────────────────────────┘
```

Each layer depends only on the layers below it. Framework Integration (top) depends on
Metrics Exporter; Metrics Exporter depends on Check Scheduler, and so on.
Core Abstractions (bottom) have no internal dependencies.

## Type Mapping

Core types must be consistent across all SDKs:

| Concept | Go | Java | Python | C# |
| --- | --- | --- | --- | --- |
| Dependency model | `Dependency` struct | `Dependency` class | `Dependency` dataclass | `Dependency` record |
| Endpoint model | `Endpoint` struct | `Endpoint` class | `Endpoint` dataclass | `Endpoint` record |
| Health checker | `HealthChecker` interface | `HealthChecker` interface | `HealthChecker` Protocol | `IHealthChecker` interface |
| Dependency type | `string` constant | `DependencyType` enum | `str` literal | `DependencyType` enum |
| Configuration | Functional options | Builder pattern | Constructor kwargs | Builder pattern |
| Check result | `error` (nil = healthy) | `void` (throws on failure) | `None` (raises on failure) | `Task` (throws on failure) |
| Scheduler | goroutines | `ScheduledExecutorService` | `asyncio.Task` | `Task` + `Timer` |

## Extensibility: Adding a New Checker

All SDKs follow the same pattern for adding a new health checker:

1. **Implement the checker interface** — create a type that satisfies `HealthChecker`
   (Go interface, Java interface, Python Protocol, C# interface)
2. **Register in the checker factory** — so the scheduler can instantiate it by `DependencyType`
3. **Add convenience constructor** — a public function/method like `Kafka("name", ...)` that
   creates a `Dependency` with the correct type and checker
4. **Add tests** — at minimum: happy path, connection error, timeout

The checker interface is intentionally minimal — one method to check, one to report the type:

```text
Check(endpoint) → success or error
Type() → string
```

## Thread Safety and Concurrency

All SDK code that runs in the check scheduler **must be thread-safe** (or goroutine-safe,
async-safe depending on the language):

- **Checkers**: called concurrently for different endpoints — must not share mutable state
- **Metrics exporter**: gauge/histogram updates must be atomic (guaranteed by Prometheus client libraries)
- **Scheduler**: manages its own lifecycle (start/stop), must handle graceful shutdown

Language-specific concurrency:

| Language | Mechanism | Key rule |
| --- | --- | --- |
| Go | goroutines + `context.Context` | Pass `ctx` for cancellation; no shared mutable state without sync |
| Java | `ScheduledExecutorService` | Implementations marked thread-safe; prefer immutable objects |
| Python | `asyncio` | Never block the event loop; use `async` checkers |
| C# | `Task` + `CancellationToken` | Use `ConfigureAwait(false)` in library code |

## Configuration

All SDKs use the **builder pattern** (or language equivalent) for configuration:

- **Sensible defaults** — `checkInterval=15s`, `timeout=5s`, `failureThreshold=1`
- **Validation at build time** — fail fast if configuration is invalid (empty name, zero interval, etc.)
- **Immutable after build** — once created, the configuration cannot be changed

```text
// Pseudocode pattern across all languages:
DependencyHealth.builder()
    .dependency("name", type, endpoint_config, options...)
    .dependency(...)
    .checkInterval(15s)
    .build()
    .start()
```

Connection configuration accepts multiple formats:

- Full URL: `postgres://user:pass@host:5432/db`
- Separate parameters: `host`, `port`
- Connection string: `Host=...;Port=...`

The SDK automatically extracts `host` and `port` from any format for metric labels.
**Credentials are never exposed** in metrics or logs.

## Error Handling

Philosophy: **fail fast with informative messages**.

- Configuration errors (invalid URL, missing name) — fail immediately at build/start time
- Check errors (timeout, connection refused) — report via metrics, log at appropriate level,
  do **not** crash the application
- Unexpected errors in checkers — catch, log as error, report endpoint as unhealthy

Each SDK defines a base exception/error type:

| Language | Base type | Common subtypes |
| --- | --- | --- |
| Go | Sentinel errors (`ErrTimeout`, `ErrConnectionRefused`, `ErrUnhealthy`) | Wrapped with `fmt.Errorf("%w", ...)` |
| Java | `DepHealthException` (unchecked) | `CheckTimeoutException`, `ConnectionRefusedException` |
| Python | `CheckError` | `CheckTimeoutError`, `CheckConnectionRefusedError`, `UnhealthyError` |
| C# | `DepHealthException` | `CheckTimeoutException`, `ConnectionRefusedException` |

## Logging

All SDKs use the standard logging framework of their language:

| Language | Framework | Logger name |
| --- | --- | --- |
| Go | `log/slog` | `dephealth` |
| Java | SLF4J | `biz.kryukov.dev.dephealth` |
| Python | `logging` | `dephealth` |
| C# | `ILogger<T>` | `DepHealth.*` |

Log levels:

| Level | Usage |
| --- | --- |
| `ERROR` | Unexpected errors (panic recovery, metric registration failure) |
| `WARN` | Check failures, configuration warnings |
| `INFO` | Scheduler start/stop, dependency registered |
| `DEBUG` | Individual check results, timing |

Rules:

- **Never log credentials** — URLs are sanitized before logging
- **Use structured logging** where available (slog fields, SLF4J MDC, Python extra)
- **Parameterized messages** — no string concatenation for log messages
  (e.g., SLF4J `log.warn("Check failed for {}", name)`, not `log.warn("Check failed for " + name)`)

## Naming Conventions Summary

| Concept | Go | Java | Python | C# |
| --- | --- | --- | --- | --- |
| Package/namespace | `dephealth` | `biz.kryukov.dev.dephealth` | `dephealth` | `DepHealth` |
| Public type | `PascalCase` | `PascalCase` | `PascalCase` | `PascalCase` |
| Public method | `PascalCase` | `camelCase` | `snake_case` | `PascalCase` |
| Private field | `camelCase` | `camelCase` | `_snake_case` | `_camelCase` |
| Constant | `PascalCase` | `UPPER_SNAKE_CASE` | `UPPER_SNAKE_CASE` | `PascalCase` |
| Test method | `TestXxx` | `xxxTest` / `@Test` | `test_xxx` | `Xxx_Should_Yyy` |

See language-specific guides for complete naming rules.

## Documentation Format

All documentation files follow the EN/RU dual format:

- English file: original name (e.g., `overview.md`)
- Russian file: `.ru.md` suffix (e.g., `overview.ru.md`)
- EN starts with `*[Русская версия](overview.ru.md)*`
- RU starts with `*[English version](overview.md)*`
- Internal links in RU files point to `.ru.md` versions
- Markdown must pass `markdownlint` (MD013 line length disabled)
