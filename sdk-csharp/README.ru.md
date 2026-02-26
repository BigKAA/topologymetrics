*[English version](README.md)*

# dephealth

SDK для мониторинга зависимостей микросервисов через метрики Prometheus.

## Возможности

- Автоматическая проверка здоровья зависимостей (PostgreSQL, MySQL, Redis, RabbitMQ, Kafka, HTTP, gRPC, TCP, LDAP)
- Экспорт метрик Prometheus: `app_dependency_health` (Gauge 0/1), `app_dependency_latency_seconds` (Histogram), `app_dependency_status` (enum), `app_dependency_status_detail` (info)
- Асинхронная архитектура на базе `async Task`
- Интеграция с ASP.NET Core (hosted service, middleware, health endpoints)
- Интеграция с Entity Framework для проверок через connection pool
- Поддержка connection pool (предпочтительно) и автономных проверок

## Установка

```bash
dotnet add package DepHealth.AspNetCore
```

Для интеграции с Entity Framework:

```bash
dotnet add package DepHealth.EntityFramework
```

## Быстрый старт

### Автономный режим

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddPostgres("db", "postgres://user:pass@localhost:5432/mydb", critical: true)
    .AddRedis("cache", "redis://localhost:6379", critical: false)
    .Build();

dh.Start();
// Метрики доступны через prometheus-net
dh.Stop();
```

### ASP.NET Core

```csharp
using DepHealth;
using DepHealth.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("db", builder.Configuration["DATABASE_URL"]!, critical: true)
    .AddRedis("cache", builder.Configuration["REDIS_URL"]!, critical: false)
);

var app = builder.Build();
app.UseDepHealth();
app.Run();
```

## Динамические эндпоинты

Добавление, удаление и замена мониторируемых эндпоинтов в рантайме на
работающем экземпляре (v0.6.0+):

```csharp
using DepHealth;
using DepHealth.Checks;

// После dh.Start()...

// Добавить новый эндпоинт
dh.AddEndpoint(
    "api-backend",
    DependencyType.Http,
    true,
    new Endpoint("backend-2.svc", "8080"),
    new HttpChecker());

// Удалить эндпоинт (отменяет задачу, удаляет метрики)
dh.RemoveEndpoint("api-backend", "backend-2.svc", "8080");

// Заменить эндпоинт атомарно
dh.UpdateEndpoint(
    "api-backend",
    "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    new HttpChecker());
```

Подробности в [руководстве по миграции](docs/migration.ru.md#v050--v060).

## Детализация здоровья

```csharp
var details = dh.HealthDetails();
foreach (var (key, ep) in details)
{
    Console.WriteLine($"{key}: healthy={ep.Healthy} status={ep.Status} " +
        $"latency={ep.LatencyMillis:F1}ms");
}
```

## Конфигурация

| Параметр | По умолчанию | Описание |
| --- | --- | --- |
| `WithCheckInterval` | `15s` | Интервал проверки |
| `WithCheckTimeout` | `5s` | Таймаут проверки |
| `WithInitialDelay` | `0s` | Начальная задержка перед первой проверкой |

## Поддерживаемые зависимости

| Тип | DependencyType | Формат URL |
| --- | --- | --- |
| PostgreSQL | `Postgres` | `postgres://user:pass@host:5432/db` |
| MySQL | `MySql` | `mysql://user:pass@host:3306/db` |
| Redis | `Redis` | `redis://host:6379` |
| RabbitMQ | `Amqp` | `amqp://user:pass@host:5672/vhost` |
| Kafka | `Kafka` | `kafka://host1:9092,host2:9092` |
| HTTP | `Http` | `http://host:8080/health` |
| gRPC | `Grpc` | `host:50051` (через `Host()` + `Port()`) |
| TCP | `Tcp` | `tcp://host:port` |
| LDAP | `Ldap` | `ldap://host:389` или `ldaps://host:636` |

## LDAP-чекер

LDAP-чекер поддерживает четыре метода проверки и несколько режимов TLS:

```csharp
using DepHealth.Checks;

// Запрос RootDSE (по умолчанию)
var checker = new LdapChecker(checkMethod: LdapCheckMethod.RootDse);

// Простая привязка с учётными данными
var checker = new LdapChecker(
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=monitor,dc=corp,dc=com",
    bindPassword: "secret",
    useTls: true);

// Поиск с StartTLS
var checker = new LdapChecker(
    checkMethod: LdapCheckMethod.Search,
    baseDN: "dc=example,dc=com",
    searchFilter: "(objectClass=organizationalUnit)",
    searchScope: LdapSearchScope.One,
    startTls: true);
```

Методы проверки: `AnonymousBind`, `SimpleBind`, `RootDse` (по умолчанию), `Search`.

## Аутентификация

HTTP и gRPC чекеры поддерживают Bearer token, Basic Auth и пользовательские заголовки/метаданные:

```csharp
builder.AddHttp("secure-api", "http://api.svc:8080",
    critical: true,
    bearerToken: "eyJhbG...");

builder.AddGrpc("grpc-backend", "backend.svc", "9090",
    critical: true,
    bearerToken: "eyJhbG...");
```

Все опции описаны в [руководстве по аутентификации](docs/authentication.ru.md).

## Документация

Полная документация доступна в директории [docs/](docs/README.md).

## Лицензия

Apache License 2.0 — см. [LICENSE](https://github.com/BigKAA/topologymetrics/blob/master/LICENSE).
