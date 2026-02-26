*[English version](code-style.md)*

# Code Style Guide: C# SDK

Этот документ описывает соглашения по стилю кода для C# SDK (`sdk-csharp/`).
См. также: [Общие принципы](../../docs/code-style/overview.ru.md) | [Тестирование](../../docs/code-style/testing.ru.md)

## Соглашения об именовании

### Пространства имён

- `PascalCase`, соответствуют структуре проекта

```csharp
namespace DepHealth;           // ядро
namespace DepHealth.Checks;    // health checkers
namespace DepHealth.AspNetCore; // ASP.NET интеграция
```

### Классы, интерфейсы, записи

- `PascalCase` для всех типов
- Интерфейсы: префикс `I` (соглашение C#)
- Records для неизменяемых моделей

```csharp
// Интерфейсы — I-префикс
public interface IHealthChecker { }
public interface ICheckScheduler { }

// Классы
public class HttpChecker : IHealthChecker { }
public class CheckScheduler : ICheckScheduler { }

// Records для моделей
public record Dependency(string Name, DependencyType Type, bool Critical, IReadOnlyList<Endpoint> Endpoints);
public record Endpoint(string Host, int Port, IReadOnlyDictionary<string, string> Metadata);
```

### Методы и свойства

- `PascalCase` для всех публичных методов и свойств
- Async-методы: суффикс `Async`

```csharp
public Task CheckAsync(Endpoint endpoint, CancellationToken ct);
public DependencyType Type { get; }
public bool IsCritical { get; }

// Суффикс Async для всех async-методов
public Task StartAsync(CancellationToken ct);
public Task StopAsync();
```

### Поля и переменные

- `_camelCase` для приватных полей (префикс подчёркивание)
- `camelCase` для локальных переменных и параметров
- Без префиксов `m_` или `s_`

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

### Константы и перечисления

- `PascalCase` для констант (без `UPPER_SNAKE_CASE` в C#)
- Тип enum: `PascalCase` единственное число, значения: `PascalCase`

```csharp
public static class Defaults
{
    public const int CheckIntervalSeconds = 15;
    public const int TimeoutSeconds = 5;
    public const int FailureThreshold = 1;
}

public enum DependencyType
{
    Http, Grpc, Tcp, Postgres, MySql, Redis, Amqp, Kafka, Ldap
}
```

## Структура проекта

```text
sdk-csharp/
├── DepHealth.Core/
│   ├── DependencyHealthBuilder.cs   // builder, основной API
│   ├── Dependency.cs                // модель (record)
│   ├── Endpoint.cs                  // модель (record)
│   ├── DependencyType.cs            // enum
│   ├── IHealthChecker.cs            // интерфейс проверки
│   ├── CheckScheduler.cs            // планировщик
│   ├── ConnectionParser.cs          // парсер URL/параметров
│   ├── PrometheusExporter.cs        // prometheus-net метрики
│   ├── Exceptions/
│   │   ├── DepHealthException.cs
│   │   ├── CheckTimeoutException.cs
│   │   ├── ConnectionRefusedException.cs
│   │   ├── CheckDnsException.cs
│   │   ├── CheckAuthException.cs
│   │   ├── CheckTlsException.cs
│   │   ├── EndpointNotFoundException.cs
│   │   └── UnhealthyException.cs
│   └── Checks/
│       ├── HttpChecker.cs
│       ├── GrpcChecker.cs
│       ├── TcpChecker.cs
│       ├── PostgresChecker.cs
│       ├── RedisChecker.cs
│       ├── AmqpChecker.cs
│       ├── KafkaChecker.cs
│       └── LdapChecker.cs
│
├── DepHealth.AspNetCore/
│   ├── ServiceCollectionExtensions.cs  // AddDepHealth()
│   └── ApplicationBuilderExtensions.cs // UseDepHealth()
│
├── DepHealth.EntityFramework/
│   └── ...
│
└── DepHealth.Core.Tests/
    └── ...
```

## Обработка ошибок

### Иерархия исключений

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

public class CheckDnsException : DepHealthException
{
    public CheckDnsException(string message, Exception inner)
        : base(message, inner) { }
}

public class CheckAuthException : DepHealthException
{
    public CheckAuthException(string message, Exception inner)
        : base(message, inner) { }
}

public class CheckTlsException : DepHealthException
{
    public CheckTlsException(string message, Exception inner)
        : base(message, inner) { }
}

public class EndpointNotFoundException : DepHealthException
{
    public EndpointNotFoundException(string message) : base(message) { }
    public EndpointNotFoundException(string message, Exception inner)
        : base(message, inner) { }
}

public class UnhealthyException : DepHealthException
{
    public UnhealthyException(string message) : base(message) { }
    public UnhealthyException(string message, Exception inner)
        : base(message, inner) { }
}
```

### Правила

- Ошибки конфигурации: бросать `ArgumentException` / `ArgumentNullException` немедленно
- Ошибки проверки: бросать конкретные подтипы, перехватываемые планировщиком
- Всегда включать `innerException` для сохранения стека вызовов
- Использовать `nameof()` для валидации аргументов

```csharp
// Хорошо — чёткая валидация с nameof
public void AddDependency(string name, DependencyType type, Endpoint endpoint)
{
    ArgumentException.ThrowIfNullOrWhiteSpace(name);
    ArgumentNullException.ThrowIfNull(endpoint);

    // ...
}

// Хорошо — исключение с inner
catch (OperationCanceledException ex)
{
    throw new CheckTimeoutException(
        $"Health check timed out for {endpoint.Host}:{endpoint.Port}", ex);
}
```

## XML-документация

### Формат

```csharp
/// <summary>
/// Interface for dependency health checks.
/// Implementations must be thread-safe.
/// </summary>
public interface IHealthChecker
{
    /// <summary>
    /// Performs a health check against the given endpoint.
    /// </summary>
    /// <param name="endpoint">The endpoint to check.</param>
    /// <param name="ct">Cancellation token (used as timeout).</param>
    /// <exception cref="CheckTimeoutException">If the check exceeded the timeout.</exception>
    /// <exception cref="ConnectionRefusedException">If the connection was refused.</exception>
    Task CheckAsync(Endpoint endpoint, CancellationToken ct);

    /// <summary>
    /// The dependency type.
    /// </summary>
    DependencyType Type { get; }
}
```

Правила:

- `<summary>` для всех публичных типов и членов (на английском)
- `<param>` для всех параметров
- `<exception cref="">` для бросаемых исключений
- `<returns>` для не-void возвращаемых значений
- `<inheritdoc/>` для реализаций интерфейсов где уместно

## Async/Await

### ConfigureAwait(false)

**Всегда** используйте `ConfigureAwait(false)` в библиотечном коде. Это предотвращает
дедлоки при вызове библиотеки из контекста синхронизации (например, ASP.NET):

```csharp
// Хорошо — библиотечный код
public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
{
    using var response = await _client.GetAsync(url, ct)
        .ConfigureAwait(false);

    response.EnsureSuccessStatusCode();
}

// В ASP.NET контроллерах — ConfigureAwait не нужен (нет SynchronizationContext в .NET 6+)
```

### CancellationToken

Принимайте `CancellationToken` во всех async-методах и передавайте далее:

```csharp
public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
{
    // Передаём ct во все async-вызовы
    await using var conn = new NpgsqlConnection(_connectionString);
    await conn.OpenAsync(ct).ConfigureAwait(false);
    await using var cmd = new NpgsqlCommand("SELECT 1", conn);
    await cmd.ExecuteScalarAsync(ct).ConfigureAwait(false);
}
```

## IDisposable / IAsyncDisposable

Реализуйте освобождение ресурсов для типов, владеющих unmanaged-ресурсами или долгоживущими соединениями:

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

Правила:

- Предпочитайте `IAsyncDisposable` вместо `IDisposable` для async-ресурсов
- Используйте `await using` для потребления disposable-объектов
- Вызывайте `Dispose`/`DisposeAsync` на всех собственных ресурсах

## Nullable Reference Types

Включите nullable reference types на уровне проекта (`<Nullable>enable</Nullable>`):

```csharp
// Non-nullable по умолчанию — компилятор предупреждает при присвоении null
public record Endpoint(string Host, int Port);

// Явный nullable где нужно
public string? ParseHost(string? connectionString)
{
    if (string.IsNullOrEmpty(connectionString))
        return null;
    // ...
}
```

Правила:

- Избегайте `null!` (null-forgiving оператор) — исправьте тип или добавьте проверку на null
- Используйте `ArgumentNullException.ThrowIfNull()` для параметров публичного API
- Предпочитайте `string.IsNullOrEmpty()` / `string.IsNullOrWhiteSpace()` вместо `== null`

## File-Scoped Namespaces

Используйте file-scoped namespaces (C# 10+) для уменьшения вложенности:

```csharp
// Хорошо — file-scoped
namespace DepHealth;

public interface IHealthChecker
{
    Task CheckAsync(Endpoint endpoint, CancellationToken ct);
    DependencyType Type { get; }
}

// Плохо — block-scoped (лишняя вложенность)
namespace DepHealth
{
    public interface IHealthChecker
    {
        Task CheckAsync(Endpoint endpoint, CancellationToken ct);
        DependencyType Type { get; }
    }
}
```

## Линтер

### dotnet format

Конфигурация через `.editorconfig` в корне проекта.

Основные правила:

- Отступы: 4 пробела
- File-scoped namespaces обязательны
- `var` предпочтительно где тип очевиден
- Expression-bodied members для простых геттеров
- IDE1006 naming rules

### Запуск

```bash
cd sdk-csharp && make lint    # dotnet format --verify-no-changes в Docker
cd sdk-csharp && make fmt     # dotnet format
```

## Дополнительные соглашения

- **Версия .NET**: 8 LTS
- **Target framework**: `net8.0`
- **Метрики**: prometheus-net для регистрации метрик
- **Records**: используйте для неизменяемых моделей (`Dependency`, `Endpoint`)
- **Primary constructors** (C# 12): используйте для простых DI-классов
- **Collection expressions** (C# 12): `[]` вместо `new List<T>()`
- **Pattern matching**: используйте `is`, `switch` expressions где улучшают читаемость
- **Один тип на файл**: каждый публичный тип в своём файле (совпадающем с именем класса)
