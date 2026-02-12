*[Русская версия](csharp.ru.md)*

# Code Style Guide: C# SDK

This document describes code style conventions for the C# SDK (`sdk-csharp/`).
See also: [General Principles](overview.md) | [Testing](testing.md)

## Naming Conventions

### Namespaces

- `PascalCase`, matching project structure

```csharp
namespace DepHealth;           // core
namespace DepHealth.Checks;    // health checkers
namespace DepHealth.AspNetCore; // ASP.NET integration
```

### Classes, Interfaces, Records

- `PascalCase` for all types
- Interfaces: `I` prefix (C# convention)
- Records for immutable models

```csharp
// Interfaces — I-prefix
public interface IHealthChecker { }
public interface ICheckScheduler { }

// Classes
public class HttpChecker : IHealthChecker { }
public class CheckScheduler : ICheckScheduler { }

// Records for models
public record Dependency(string Name, DependencyType Type, bool Critical, IReadOnlyList<Endpoint> Endpoints);
public record Endpoint(string Host, int Port, IReadOnlyDictionary<string, string> Metadata);
```

### Methods and Properties

- `PascalCase` for all public methods and properties
- Async methods: `Async` suffix

```csharp
public Task CheckAsync(Endpoint endpoint, CancellationToken ct);
public DependencyType Type { get; }
public bool IsCritical { get; }

// Async suffix for all async methods
public Task StartAsync(CancellationToken ct);
public Task StopAsync();
```

### Fields and Variables

- `_camelCase` for private fields (underscore prefix)
- `camelCase` for local variables and parameters
- No `m_` or `s_` prefixes

```csharp
public class CheckScheduler : ICheckScheduler
{
    private readonly List<Dependency> _dependencies;
    private readonly TimeSpan _checkInterval;
    private bool _running;

    public CheckScheduler(TimeSpan checkInterval)
    {
        _checkInterval = checkInterval;
        _dependencies = new List<Dependency>();
    }
}
```

### Constants and Enums

- `PascalCase` for constants (no `UPPER_SNAKE_CASE` in C#)
- Enum type: `PascalCase` singular, values: `PascalCase`

```csharp
public static class Defaults
{
    public const int CheckIntervalSeconds = 15;
    public const int TimeoutSeconds = 5;
    public const int FailureThreshold = 1;
}

public enum DependencyType
{
    Http, Grpc, Tcp, Postgres, MySql, Redis, Amqp, Kafka
}
```

## Project Structure

```text
sdk-csharp/
├── DepHealth.Core/
│   ├── DependencyHealthBuilder.cs   // builder, main API
│   ├── Dependency.cs                // model (record)
│   ├── Endpoint.cs                  // model (record)
│   ├── DependencyType.cs            // enum
│   ├── IHealthChecker.cs            // checker interface
│   ├── CheckScheduler.cs            // scheduler
│   ├── ConnectionParser.cs          // URL/params parser
│   ├── PrometheusExporter.cs        // prometheus-net metrics
│   ├── Exceptions/
│   │   ├── DepHealthException.cs
│   │   ├── CheckTimeoutException.cs
│   │   └── ConnectionRefusedException.cs
│   └── Checks/
│       ├── HttpChecker.cs
│       ├── GrpcChecker.cs
│       ├── TcpChecker.cs
│       ├── PostgresChecker.cs
│       ├── RedisChecker.cs
│       ├── AmqpChecker.cs
│       └── KafkaChecker.cs
│
├── DepHealth.AspNetCore/
│   ├── ServiceCollectionExtensions.cs  // AddDepHealth()
│   └── ApplicationBuilderExtensions.cs // UseDepHealth()
│
└── DepHealth.Core.Tests/
    └── ...
```

## Error Handling

### Exception Hierarchy

```csharp
public class DepHealthException : Exception
{
    public DepHealthException(string message) : base(message) { }
    public DepHealthException(string message, Exception inner) : base(message, inner) { }
}

public class CheckTimeoutException : DepHealthException
{
    public CheckTimeoutException(string message, Exception inner)
        : base(message, inner) { }
}

public class ConnectionRefusedException : DepHealthException
{
    public ConnectionRefusedException(string message, Exception inner)
        : base(message, inner) { }
}
```

### Rules

- Configuration errors: throw `ArgumentException` / `ArgumentNullException` immediately
- Check failures: throw specific exception subtypes, caught by the scheduler
- Always include `innerException` to preserve the stack trace
- Use `nameof()` for argument validation

```csharp
// Good — clear validation with nameof
public void AddDependency(string name, DependencyType type, Endpoint endpoint)
{
    ArgumentException.ThrowIfNullOrWhiteSpace(name);
    ArgumentNullException.ThrowIfNull(endpoint);

    // ...
}

// Good — exception with inner
catch (OperationCanceledException ex)
{
    throw new CheckTimeoutException(
        $"Health check timed out for {endpoint.Host}:{endpoint.Port}", ex);
}
```

## XML Documentation Comments

### Format

```csharp
/// <summary>
/// Интерфейс проверки здоровья зависимости.
/// Реализации должны быть thread-safe.
/// </summary>
public interface IHealthChecker
{
    /// <summary>
    /// Выполняет проверку здоровья эндпоинта.
    /// </summary>
    /// <param name="endpoint">Эндпоинт для проверки.</param>
    /// <param name="ct">Токен отмены (используется как таймаут).</param>
    /// <exception cref="CheckTimeoutException">Если проверка превысила таймаут.</exception>
    /// <exception cref="ConnectionRefusedException">Если соединение отклонено.</exception>
    Task CheckAsync(Endpoint endpoint, CancellationToken ct);

    /// <summary>
    /// Тип зависимости.
    /// </summary>
    DependencyType Type { get; }
}
```

Rules:

- `<summary>` for all public types and members (in Russian)
- `<param>` for all parameters
- `<exception cref="">` for thrown exceptions
- `<returns>` for non-void return values
- `<inheritdoc/>` for interface implementations where appropriate

## Async/Await

### ConfigureAwait(false)

**Always** use `ConfigureAwait(false)` in library code. This avoids deadlocks
when the library is called from a synchronization context (e.g., ASP.NET):

```csharp
// Good — library code
public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
{
    using var response = await _client.GetAsync(url, ct)
        .ConfigureAwait(false);

    response.EnsureSuccessStatusCode();
}

// In ASP.NET controllers — ConfigureAwait not needed (no SynchronizationContext in .NET 6+)
```

### CancellationToken

Accept `CancellationToken` in all async methods and pass it through:

```csharp
public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
{
    // Pass ct to all async calls
    await using var conn = new NpgsqlConnection(_connectionString);
    await conn.OpenAsync(ct).ConfigureAwait(false);
    await using var cmd = new NpgsqlCommand("SELECT 1", conn);
    await cmd.ExecuteScalarAsync(ct).ConfigureAwait(false);
}
```

## IDisposable / IAsyncDisposable

Implement disposal for types that hold unmanaged resources or long-lived connections:

```csharp
public class CheckScheduler : IAsyncDisposable
{
    private readonly CancellationTokenSource _cts = new();
    private readonly List<Task> _tasks = [];

    public async Task StartAsync(CancellationToken ct)
    {
        foreach (var dep in _dependencies)
        {
            _tasks.Add(Task.Run(() => CheckLoopAsync(dep, _cts.Token), ct));
        }
    }

    public async ValueTask DisposeAsync()
    {
        await _cts.CancelAsync().ConfigureAwait(false);
        await Task.WhenAll(_tasks).ConfigureAwait(false);
        _cts.Dispose();
    }
}
```

Rules:

- Prefer `IAsyncDisposable` over `IDisposable` for async resources
- Use `await using` for consuming disposable objects
- Call `Dispose`/`DisposeAsync` on all owned resources

## Nullable Reference Types

Enable nullable reference types project-wide (`<Nullable>enable</Nullable>`):

```csharp
// Non-nullable by default — compiler warns on null assignment
public record Endpoint(string Host, int Port);

// Explicit nullable where needed
public string? ParseHost(string? connectionString)
{
    if (string.IsNullOrEmpty(connectionString))
        return null;
    // ...
}
```

Rules:

- Avoid `null!` (null-forgiving operator) — fix the type or add a null check instead
- Use `ArgumentNullException.ThrowIfNull()` for public API parameters
- Prefer `string.IsNullOrEmpty()` / `string.IsNullOrWhiteSpace()` over `== null`

## File-Scoped Namespaces

Use file-scoped namespaces (C# 10+) to reduce nesting:

```csharp
// Good — file-scoped
namespace DepHealth;

public interface IHealthChecker
{
    Task CheckAsync(Endpoint endpoint, CancellationToken ct);
    DependencyType Type { get; }
}

// Bad — block-scoped (extra nesting)
namespace DepHealth
{
    public interface IHealthChecker
    {
        Task CheckAsync(Endpoint endpoint, CancellationToken ct);
        DependencyType Type { get; }
    }
}
```

## Linter

### dotnet format

Configuration via `.editorconfig` in the project root.

Key rules:

- Indentation: 4 spaces
- File-scoped namespaces required
- `var` preferred where type is obvious
- Expression-bodied members for simple getters
- IDE1006 naming rules enforced

### Running

```bash
cd sdk-csharp && make lint    # dotnet format --verify-no-changes in Docker
cd sdk-csharp && make fmt     # dotnet format
```

## Additional Conventions

- **.NET version**: 8 LTS
- **Target framework**: `net8.0`
- **Metrics**: prometheus-net for metric registration
- **Records**: use for immutable models (`Dependency`, `Endpoint`)
- **Primary constructors** (C# 12): use for simple DI classes
- **Collection expressions** (C# 12): `[]` instead of `new List<T>()`
- **Pattern matching**: use `is`, `switch` expressions where they improve readability
- **One type per file**: each public type in its own file (matching class name)
