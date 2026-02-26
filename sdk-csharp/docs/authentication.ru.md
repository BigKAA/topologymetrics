*[English version](authentication.md)*

# Аутентификация

Руководство по настройке аутентификации для всех чекеров, поддерживающих
учётные данные — HTTP, gRPC, чекеров баз данных/кэшей и LDAP.

## Обзор

| Чекер | Методы аутентификации | Как передаются учётные данные |
| --- | --- | --- |
| HTTP | Bearer token, Basic auth, произвольные заголовки | `bearerToken:`, `basicAuthUsername:`/`basicAuthPassword:`, `headers:` |
| gRPC | Bearer token, Basic auth, произвольные метаданные | `bearerToken:`, `basicAuthUsername:`/`basicAuthPassword:`, `metadata:` |
| PostgreSQL | Логин/пароль | Учётные данные в URL |
| MySQL | Логин/пароль | Учётные данные в URL |
| Redis | Пароль | Пароль в URL или `IConnectionMultiplexer` |
| AMQP | Логин/пароль | `AmqpUsername()`, `AmqpPassword()` или учётные данные в URL |
| LDAP | Bind DN/пароль | Параметры `bindDN:`, `bindPassword:` |
| TCP | — | Без аутентификации |
| Kafka | — | Без аутентификации |

## HTTP аутентификация

Для каждой зависимости допускается только один метод аутентификации.
Одновременное указание `bearerToken` и `basicAuthUsername`/`basicAuthPassword`
вызывает `ValidationException`.

### Bearer Token

```csharp
.AddHttp("secure-api", "http://api.svc:8080",
    critical: true,
    bearerToken: Environment.GetEnvironmentVariable("API_TOKEN"))
```

Отправляет заголовок `Authorization: Bearer <token>` с каждым запросом проверки.

### Basic Auth

```csharp
.AddHttp("basic-api", "http://api.svc:8080",
    critical: true,
    basicAuthUsername: "admin",
    basicAuthPassword: Environment.GetEnvironmentVariable("API_PASSWORD"))
```

Отправляет заголовок `Authorization: Basic <base64(user:pass)>`.

### Произвольные заголовки

Для нестандартных схем аутентификации (API-ключи, пользовательские токены):

```csharp
.AddHttp("custom-auth-api", "http://api.svc:8080",
    critical: true,
    headers: new Dictionary<string, string>
    {
        ["X-API-Key"] = Environment.GetEnvironmentVariable("API_KEY")!,
        ["X-API-Secret"] = Environment.GetEnvironmentVariable("API_SECRET")!
    })
```

## gRPC аутентификация

Для каждой зависимости допускается только один метод, как и для HTTP.

### Bearer Token

```csharp
.AddGrpc("grpc-backend",
    host: "backend.svc",
    port: "9090",
    critical: true,
    bearerToken: Environment.GetEnvironmentVariable("GRPC_TOKEN"))
```

Отправляет `authorization: Bearer <token>` как gRPC-метаданные.

### Basic Auth

```csharp
.AddGrpc("grpc-backend",
    host: "backend.svc",
    port: "9090",
    critical: true,
    basicAuthUsername: "admin",
    basicAuthPassword: Environment.GetEnvironmentVariable("GRPC_PASSWORD"))
```

Отправляет `authorization: Basic <base64(user:pass)>` как gRPC-метаданные.

### Произвольные метаданные

```csharp
.AddGrpc("grpc-backend",
    host: "backend.svc",
    port: "9090",
    critical: true,
    metadata: new Dictionary<string, string>
    {
        ["x-api-key"] = Environment.GetEnvironmentVariable("GRPC_API_KEY")!
    })
```

## Учётные данные баз данных

### PostgreSQL

Учётные данные включаются в строку подключения URL:

```csharp
// Через URL
.AddPostgres("postgres-main",
    url: "postgresql://myuser:mypass@pg.svc:5432/mydb",
    critical: true)

// Через переменную окружения
.AddPostgres("postgres-main",
    url: Environment.GetEnvironmentVariable("DATABASE_URL")!,
    critical: true)
```

SDK автоматически извлекает имя пользователя, пароль, хост, порт и имя базы данных из URL.

### MySQL

Аналогичный подход — учётные данные в URL:

```csharp
.AddMySql("mysql-main",
    url: "mysql://myuser:mypass@mysql.svc:3306/mydb",
    critical: true)
```

### Redis

Пароль можно включить в URL или передать через `IConnectionMultiplexer` в режиме пула:

```csharp
// Через URL
.AddRedis("redis-cache",
    url: "redis://:mypassword@redis.svc:6379/0",
    critical: false)

// Без пароля
.AddRedis("redis-cache",
    url: "redis://redis.svc:6379",
    critical: false)
```

Режим пула через `IConnectionMultiplexer` — мультиплексор уже аутентифицирован
самим приложением:

```csharp
using DepHealth.Checks;
using StackExchange.Redis;

IConnectionMultiplexer multiplexer =
    await ConnectionMultiplexer.ConnectAsync("redis.svc:6379,password=secret");

var checker = new RedisChecker(multiplexer);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("redis-cache", DependencyType.Redis, "redis.svc", "6379", checker,
        critical: false)
    .Build();
```

### AMQP (RabbitMQ)

Учётные данные передаются в AMQP URL:

```csharp
// Через URL
.AddAmqp("rabbitmq",
    url: "amqp://user:pass@rabbitmq.svc:5672/",
    critical: false)

// С произвольным virtual host
.AddAmqp("rabbitmq",
    url: "amqp://user:pass@rabbitmq.svc:5672/myvhost",
    critical: false)
```

При прямом создании чекера учётные данные можно передать явно:

```csharp
using DepHealth.Checks;

var checker = new AmqpChecker(
    username: "user",
    password: "pass",
    vhost: "/");
```

### LDAP

Для метода проверки `SimpleBind` требуются учётные данные для привязки:

```csharp
.AddLdap("directory",
    host: "ldap.svc",
    port: "389",
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=monitor,dc=corp,dc=com",
    bindPassword: Environment.GetEnvironmentVariable("LDAP_PASSWORD"),
    critical: true)
```

Для LDAPS (TLS):

```csharp
.AddLdap("directory-secure",
    host: "ldap.svc",
    port: "636",
    useTls: true,
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=monitor,dc=corp,dc=com",
    bindPassword: Environment.GetEnvironmentVariable("LDAP_PASSWORD"),
    critical: true)
```

## Классификация ошибок аутентификации

Когда зависимость отклоняет учётные данные, чекер классифицирует ошибку
как `auth_error`. Это позволяет настроить алертинг на ошибки аутентификации
отдельно от проблем подключения и здоровья.

| Чекер | Триггер | Статус | Детализация |
| --- | --- | --- | --- |
| HTTP | Ответ 401 (Unauthorized) | `auth_error` | `auth_error` |
| HTTP | Ответ 403 (Forbidden) | `auth_error` | `auth_error` |
| gRPC | Код UNAUTHENTICATED | `auth_error` | `auth_error` |
| gRPC | Код PERMISSION_DENIED | `auth_error` | `auth_error` |
| PostgreSQL | SQLSTATE 28000/28P01 | `auth_error` | `auth_error` |
| MySQL | Ошибка AccessDenied (1045) | `auth_error` | `auth_error` |
| Redis | Ошибка NOAUTH/WRONGPASS | `auth_error` | `auth_error` |
| AMQP | 403 ACCESS_REFUSED | `auth_error` | `auth_error` |
| LDAP | Код 49 (Invalid Credentials) | `auth_error` | `auth_error` |
| LDAP | Код 50 (Insufficient Access) | `auth_error` | `auth_error` |

### Пример PromQL: алертинг на ошибки аутентификации

```promql
# Алерт когда любая зависимость имеет статус auth_error
app_dependency_status{status="auth_error"} == 1
```

## Лучшие практики безопасности

1. **Никогда не хардкодьте учётные данные** — всегда используйте переменные
   окружения или системы управления секретами
2. **Используйте короткоживущие токены** — если ваша система аутентификации
   поддерживает ротацию токенов, чекер будет использовать значение токена,
   переданное при создании
3. **Предпочитайте TLS** — включайте TLS для HTTP (`https://`) и gRPC
   (`tlsEnabled: true`) чекеров при использовании аутентификации для защиты
   учётных данных при передаче
4. **Ограничивайте права** — используйте учётные данные только для чтения;
   `SELECT 1` не требует прав на запись

## Полный пример

```csharp
using DepHealth;
using DepHealth.Checks;
using StackExchange.Redis;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // HTTP с Bearer-токеном
    .AddHttp("payment-api", "https://payment.svc:443",
        critical: true,
        bearerToken: Environment.GetEnvironmentVariable("PAYMENT_TOKEN"))

    // gRPC с произвольными метаданными
    .AddGrpc("user-service",
        host: "user.svc",
        port: "9090",
        critical: true,
        metadata: new Dictionary<string, string>
        {
            ["x-api-key"] = Environment.GetEnvironmentVariable("USER_API_KEY")!
        })

    // PostgreSQL с учётными данными в URL
    .AddPostgres("postgres-main",
        url: Environment.GetEnvironmentVariable("DATABASE_URL")!,
        critical: true)

    // Redis с паролем в URL
    .AddRedis("redis-cache",
        url: $"redis://:{Environment.GetEnvironmentVariable("REDIS_PASSWORD")}@redis.svc:6379",
        critical: false)

    // AMQP с учётными данными в URL
    .AddAmqp("rabbitmq",
        url: Environment.GetEnvironmentVariable("AMQP_URL")!,
        critical: false)

    // LDAP с учётными данными для привязки
    .AddLdap("directory",
        host: "ad.corp",
        port: "636",
        useTls: true,
        checkMethod: LdapCheckMethod.SimpleBind,
        bindDN: "cn=monitor,dc=corp,dc=com",
        bindPassword: Environment.GetEnvironmentVariable("LDAP_PASSWORD"),
        critical: true)

    .Build();

dh.Start();
Console.CancelKeyPress += (_, _) => dh.Stop();
Console.ReadLine();
```

## См. также

- [Чекеры](checkers.ru.md) — подробное руководство по всем 9 чекерам
- [Конфигурация](configuration.ru.md) — переменные окружения и опции
- [Справочник API](api-reference.ru.md) — полный справочник всех публичных классов
