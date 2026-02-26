*[English version](connection-pools.md)*

# Интеграция с пулами подключений

dephealth поддерживает два режима проверки зависимостей:

- **Автономный режим** — SDK создаёт новое подключение для каждой проверки
- **Режим пула** — SDK использует существующий пул подключений вашего сервиса

Режим пула предпочтительнее, так как отражает реальную способность сервиса
работать с зависимостью. Если пул подключений исчерпан, проверка состояния
обнаружит это.

## Автономный режим vs режим пула

| Аспект | Автономный | Пул |
| --- | --- | --- |
| Подключение | Новое при каждой проверке | Использует существующий пул |
| Отражает реальное состояние | Частично | Да |
| Настройка | Простая — только URL | Требует передачи объекта пула |
| Внешние зависимости | Нет (использует драйвер чекера) | Драйвер вашего приложения |
| Обнаруживает исчерпание пула | Нет | Да |

## PostgreSQL через NpgsqlDataSource

Передайте существующий `NpgsqlDataSource` в builder зависимостей:

```csharp
using Npgsql;

NpgsqlDataSource dataSource = ...; // existing from DI

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("postgres-main", DependencyType.Postgres,
        "pg.svc", "5432",
        new PostgresChecker(dataSource),
        critical: true)
    .Build();
```

Чекер вызывает `NpgsqlDataSource.OpenConnectionAsync()`, выполняет `SELECT 1`
и закрывает подключение. Хост и порт указываются явно в `AddCustom`.

Можно также передать `NpgsqlDataSource` напрямую, когда `AddNpgsqlFromContext`
недоступен — например, при работе вне DI-контейнера ASP.NET Core:

```csharp
using Npgsql;
using DepHealth.Checks;

NpgsqlDataSource dataSource = NpgsqlDataSource.Create(connectionString);

var checker = new PostgresChecker(dataSource);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("postgres-main", DependencyType.Postgres,
        "pg.svc", "5432", checker, critical: true)
    .Build();
```

## PostgreSQL через Entity Framework

При использовании ASP.NET Core с Entity Framework Core используйте
`AddNpgsqlFromContext<TContext>()`:

```csharp
using DepHealth.EntityFramework;

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext<AppDbContext>("postgres-main", critical: true)
);
```

Расширение разрешает `AppDbContext` из DI-контейнера, извлекает строку
подключения и создаёт `PostgresChecker`, использующий тот же пул.
Подробности — в [Интеграция с Entity Framework](entity-framework.ru.md).

## Redis через IConnectionMultiplexer

Передайте существующий `IConnectionMultiplexer` (StackExchange.Redis) в
builder зависимостей:

```csharp
using StackExchange.Redis;

IConnectionMultiplexer redis = ...; // existing from DI

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("redis-cache", DependencyType.Redis,
        "redis.svc", "6379",
        new RedisChecker(redis),
        critical: false)
    .Build();
```

Чекер вызывает `IConnectionMultiplexer.GetDatabase().PingAsync()` на
переданном экземпляре. Это отражает реальную способность приложения
обращаться к Redis через тот же мультиплексер, который оно использует
для всех остальных операций.

## LDAP через ILdapConnection

Передайте существующий `ILdapConnection` (Novell.Directory.Ldap) в
builder зависимостей:

```csharp
ILdapConnection ldapConn = ...; // existing connection

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("directory", DependencyType.Ldap,
        "ldap.svc", "389",
        new LdapChecker(ldapConn, checkMethod: LdapCheckMethod.RootDse),
        critical: true)
    .Build();
```

При использовании существующего подключения настройки TLS (`useTls`,
`startTls`) управляются самим подключением, а не чекером.

## Прямое создание чекеров в режиме пула

Для более тонкого управления можно создавать чекеры напрямую с объектами
пулов и регистрировать их через `AddEndpoint()`:

### PostgreSQL с NpgsqlDataSource

```csharp
using DepHealth.Checks;
using Npgsql;

NpgsqlDataSource dataSource = ...;

var checker = new PostgresChecker(dataSource);

// После dh.Build() и dh.Start()
dh.AddEndpoint("postgres-main", DependencyType.Postgres, critical: true,
    new Endpoint("pg.svc", "5432"),
    checker);
```

### Redis с IConnectionMultiplexer

```csharp
using DepHealth.Checks;
using StackExchange.Redis;

IConnectionMultiplexer multiplexer = ...;

var checker = new RedisChecker(multiplexer);

dh.AddEndpoint("redis-cache", DependencyType.Redis, critical: false,
    new Endpoint("redis.svc", "6379"),
    checker);
```

## Автономный режим vs режим пула: когда что использовать

| Сценарий | Рекомендация |
| --- | --- |
| Стандартная настройка, один пул на зависимость | Режим пула через `NpgsqlDataSource` / `IConnectionMultiplexer` |
| Нет существующего пула (внешние сервисы) | Автономный режим через URL |
| HTTP и gRPC сервисы | Только автономный режим (пул не нужен) |
| Приложение на EF Core | Интеграция с Entity Framework |
| LDAP с управляемым подключением | Режим пула через `ILdapConnection` |

## Полный пример: смешанные режимы

```csharp
using DepHealth;
using DepHealth.Checks;
using Npgsql;
using StackExchange.Redis;

// Existing connection pools
NpgsqlDataSource dataSource = NpgsqlDataSource.Create(
    Environment.GetEnvironmentVariable("DATABASE_URL")!);

IConnectionMultiplexer redis =
    await ConnectionMultiplexer.ConnectAsync(
        Environment.GetEnvironmentVariable("REDIS_URL")!);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Pool mode — PostgreSQL
    .AddCustom("postgres-main", DependencyType.Postgres,
        "pg.svc", "5432",
        new PostgresChecker(dataSource),
        critical: true)

    // Pool mode — Redis
    .AddCustom("redis-cache", DependencyType.Redis,
        "redis.svc", "6379",
        new RedisChecker(redis),
        critical: false)

    // Standalone mode — HTTP (no pool needed)
    .AddHttp("payment-api", "http://payment.svc:8080", critical: true)

    // Standalone mode — gRPC (no pool needed)
    .AddGrpc("user-service", host: "user.svc", port: "9090", critical: true)

    .Build();

dh.Start();
```

## Интеграция с пулами через ASP.NET Core

При использовании DI-контейнера ASP.NET Core получите пулы из провайдера
сервисов и передайте их в builder:

```csharp
using DepHealth;
using DepHealth.AspNetCore;
using DepHealth.Checks;
using Npgsql;
using StackExchange.Redis;

var builder = WebApplication.CreateBuilder(args);

// Регистрация NpgsqlDataSource в DI
builder.Services.AddNpgsqlDataSource(
    builder.Configuration["DATABASE_URL"]!);

// Регистрация IConnectionMultiplexer в DI
builder.Services.AddSingleton<IConnectionMultiplexer>(_ =>
    ConnectionMultiplexer.Connect(
        builder.Configuration["REDIS_URL"]!));

// Регистрация DepHealth с использованием пулов из DI
builder.Services.AddSingleton(sp =>
{
    var dataSource = sp.GetRequiredService<NpgsqlDataSource>();
    var redis = sp.GetRequiredService<IConnectionMultiplexer>();

    return DepHealthMonitor.CreateBuilder("my-service", "my-team")
        .AddCustom("postgres-main", DependencyType.Postgres,
            "pg.svc", "5432",
            new PostgresChecker(dataSource),
            critical: true)
        .AddCustom("redis-cache", DependencyType.Redis,
            "redis.svc", "6379",
            new RedisChecker(redis),
            critical: false)
        .AddHttp("auth-service", "http://auth.svc:8080", critical: true)
        .Build();
});

builder.Services.AddHostedService<DepHealthHostedService>();

var app = builder.Build();
app.MapDepHealthEndpoints();
app.Run();
```

Это заменяет сокращённую форму `AddDepHealth`, но по-прежнему использует
управление жизненным циклом и интеграцию эндпоинтов из `DepHealth.AspNetCore`.

## Смотрите также

- [Чекеры](checkers.ru.md) — подробности о всех чекерах, включая опции пула
- [Интеграция с Entity Framework](entity-framework.ru.md) — интеграция пула через EF Core
- [Интеграция с ASP.NET Core](aspnetcore.ru.md) — регистрация в DI с пулами
- [Конфигурация](configuration.ru.md) — параметры строки подключения и интервала
- [Справочник API](api-reference.ru.md) — справочник методов builder
