*[English version](api-reference.md)*

# C# SDK: Справочник API

## DepHealthMonitor

Основной класс SDK. Управляет мониторингом состояния зависимостей, экспортом
метрик и жизненным циклом динамических эндпоинтов.

### Фабричный метод

```csharp
DepHealthMonitor.CreateBuilder(string name, string group)
```

| Параметр | Тип | Описание |
| --- | --- | --- |
| `name` | `string` | Имя приложения (или env `DEPHEALTH_NAME`). Формат: `[a-z][a-z0-9-]{0,62}` |
| `group` | `string` | Группа приложения (или env `DEPHEALTH_GROUP`). Тот же формат |

### Методы Builder

#### Конфигурация

```csharp
builder.WithCheckInterval(TimeSpan interval)   // По умолчанию: 15s
builder.WithCheckTimeout(TimeSpan timeout)      // По умолчанию: 5s
builder.WithInitialDelay(TimeSpan delay)        // По умолчанию: 0s
builder.WithRegistry(CollectorRegistry registry)
builder.WithLogger(ILogger logger)
```

#### Добавление зависимостей

```csharp
builder.AddHttp(name, url, healthPath: "/health", critical: null, labels: null,
    headers: null, bearerToken: null, basicAuthUsername: null, basicAuthPassword: null)

builder.AddGrpc(name, host, port, tlsEnabled: false, critical: null, labels: null,
    metadata: null, bearerToken: null, basicAuthUsername: null, basicAuthPassword: null)

builder.AddTcp(name, host, port, critical: null, labels: null)

builder.AddPostgres(name, url, critical: null, labels: null)

builder.AddMySql(name, url, critical: null, labels: null)

builder.AddRedis(name, url, critical: null, labels: null)

builder.AddAmqp(name, url, critical: null, labels: null)

builder.AddKafka(name, url, critical: null, labels: null)

builder.AddLdap(name, host, port,
    checkMethod: LdapCheckMethod.RootDse,
    bindDN: "", bindPassword: "",
    baseDN: "", searchFilter: "(objectClass=*)",
    searchScope: LdapSearchScope.Base,
    useTls: false, startTls: false, tlsSkipVerify: false,
    critical: null, labels: null)

builder.AddCustom(name, type, host, port, checker, critical: null, labels: null)
```

#### Сборка

```csharp
DepHealthMonitor Build()
```

### Методы жизненного цикла

#### `Start()`

Запуск периодических проверок здоровья. Создаёт один `async Task` на каждый
эндпоинт.

#### `Stop()`

Остановка всех задач проверки.

#### `Dispose()`

Освобождение ресурсов (вызывает `Stop()` при необходимости).

### Методы запроса состояния

#### `Health() -> Dictionary<string, bool>`

Текущее состояние здоровья, сгруппированное по имени зависимости. Зависимость
считается здоровой, если хотя бы один её эндпоинт здоров.

#### `HealthDetails() -> Dictionary<string, EndpointStatus>`

Детальный статус каждого эндпоинта. Ключи в формате `"dependency:host:port"`.

### Динамическое управление эндпоинтами

Добавлено в v0.6.0. Все методы требуют запущенного планировщика (через
`Start()`).

#### `AddEndpoint(depName, depType, critical, ep, checker) -> void`

Добавление нового мониторируемого эндпоинта в рантайме.

```csharp
public void AddEndpoint(
    string depName,
    DependencyType depType,
    bool critical,
    Endpoint ep,
    IHealthChecker checker)
```

| Параметр | Тип | Описание |
| --- | --- | --- |
| `depName` | `string` | Имя зависимости. Формат: `[a-z][a-z0-9-]{0,62}` |
| `depType` | `DependencyType` | Тип зависимости (`Http`, `Postgres` и т.д.) |
| `critical` | `bool` | Критичность зависимости |
| `ep` | `Endpoint` | Эндпоинт для мониторинга |
| `checker` | `IHealthChecker` | Реализация проверки здоровья |

**Идемпотентность:** возвращает управление без ошибки, если эндпоинт уже существует.

**Исключения:**

- `ValidationException` — некорректный `depName`, `depType`, или пустой `host`/`port`
- `InvalidOperationException` — планировщик не запущен или уже остановлен

#### `RemoveEndpoint(depName, host, port) -> void`

Удаление мониторируемого эндпоинта. Отменяет задачу проверки и удаляет
все метрики Prometheus для эндпоинта.

```csharp
public void RemoveEndpoint(
    string depName,
    string host,
    string port)
```

**Идемпотентность:** возвращает управление без ошибки, если эндпоинт не найден.

**Исключения:** `InvalidOperationException` — планировщик не запущен.

#### `UpdateEndpoint(depName, oldHost, oldPort, newEp, checker) -> void`

Атомарная замена эндпоинта. Удаляет старый эндпоинт (отменяет задачу,
удаляет метрики) и добавляет новый.

```csharp
public void UpdateEndpoint(
    string depName,
    string oldHost,
    string oldPort,
    Endpoint newEp,
    IHealthChecker checker)
```

**Исключения:**

- `EndpointNotFoundException` — старый эндпоинт не найден
- `ValidationException` — некорректный новый эндпоинт (пустой `host`/`port`, зарезервированные метки)
- `InvalidOperationException` — планировщик не запущен или уже остановлен

---

## Типы

### `Endpoint`

```csharp
public sealed class Endpoint
{
    public string Host { get; }
    public string Port { get; }
    public IReadOnlyDictionary<string, string> Labels { get; }

    public int PortAsInt()
}
```

### `DependencyType`

Enum: `Http`, `Grpc`, `Tcp`, `Postgres`, `MySql`, `Redis`, `Amqp`, `Kafka`, `Ldap`.

### `LdapCheckMethod`

Enum (пространство имён `DepHealth.Checks`): `AnonymousBind`, `SimpleBind`, `RootDse`, `Search`.

### `LdapSearchScope`

Enum (пространство имён `DepHealth.Checks`): `Base`, `One`, `Sub`.

### `EndpointStatus`

```csharp
public sealed class EndpointStatus
{
    public string Name { get; }
    public string Type { get; }
    public string Host { get; }
    public string Port { get; }
    public bool? Healthy { get; }
    public string Status { get; }
    public string Detail { get; }
    public TimeSpan Latency { get; }
    public double LatencyMillis { get; }
    public DateTimeOffset? LastCheckedAt { get; }
    public bool Critical { get; }
    public IReadOnlyDictionary<string, string> Labels { get; }
}
```

Свойства:

- `Latency` — сырое значение `TimeSpan`; игнорируется при JSON-сериализации
- `LatencyMillis` — задержка в миллисекундах (`Latency.TotalMilliseconds`); JSON-свойство `latency_ms`
- JSON-сериализация использует `System.Text.Json` с именованием snake_case

### `IHealthChecker`

```csharp
public interface IHealthChecker
{
    Task CheckAsync(Endpoint endpoint, CancellationToken cancellationToken);
}
```

---

## Исключения

| Исключение | Описание |
| --- | --- |
| `DepHealthException` | Базовый класс всех ошибок проверки (пространство имён `DepHealth.Exceptions`) |
| `CheckTimeoutException` | Тайм-аут проверки |
| `ConnectionRefusedException` | Соединение отклонено |
| `CheckDnsException` | Ошибка DNS-разрешения |
| `CheckTlsException` | Ошибка TLS-рукопожатия |
| `CheckAuthException` | Ошибка аутентификации/авторизации |
| `UnhealthyException` | Эндпоинт сообщил о нездоровом статусе |
| `ValidationException` | Ошибка валидации входных данных (пространство имён `DepHealth`) |
| `EndpointNotFoundException` | Целевой эндпоинт не найден при динамическом обновлении/удалении (v0.6.0, пространство имён `DepHealth.Exceptions`) |
| `ConfigurationException` | Ошибка разбора URL или строки подключения (пространство имён `DepHealth`) |
