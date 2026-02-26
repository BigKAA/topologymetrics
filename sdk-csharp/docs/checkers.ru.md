*[English version](checkers.md)*

# Чекеры

C# SDK включает 9 встроенных чекеров для распространённых типов зависимостей.
Каждый чекер реализует интерфейс `IHealthChecker` и может использоваться через
высокоуровневый builder API (`DepHealthMonitor.CreateBuilder(...).AddHttp(...)`)
или путём непосредственного создания экземпляра класса чекера и передачи его
в `AddCustom(...)` или `AddEndpoint(...)`.

## HTTP

Проверяет HTTP-эндпоинты, отправляя GET-запрос и ожидая ответ с кодом 2xx.

### Регистрация

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddHttp("api", "http://api.svc:8080", critical: true)
    .Build();
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `healthPath` | `/health` | Путь для эндпоинта проверки |
| `critical` | `null` | Критичность зависимости (также через `DEPHEALTH_<NAME>_CRITICAL`) |
| `headers` | `null` | Пользовательские HTTP-заголовки (`Dictionary<string, string>`) |
| `bearerToken` | `null` | Установить заголовок `Authorization: Bearer <token>` |
| `basicAuthUsername` | `null` | Имя пользователя для HTTP Basic-аутентификации |
| `basicAuthPassword` | `null` | Пароль для HTTP Basic-аутентификации |

TLS определяется автоматически по наличию `https://` в URL.

### Полный пример

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddHttp(
        name: "payment-api",
        url: "https://payment.svc:443",
        healthPath: "/healthz",
        critical: true,
        headers: new Dictionary<string, string>
        {
            ["X-Request-Source"] = "dephealth"
        })
    .Build();

dh.Start();
// ...
dh.Stop();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Ответ 2xx | `ok` | `ok` |
| Ответ 401 или 403 | `auth_error` | `auth_error` |
| Другой не-2xx ответ | `unhealthy` | `http_<код>` (напр., `http_500`) |
| Таймаут | `timeout` | `timeout` |
| Отказ соединения | `connection_error` | `connection_error` |
| Ошибка DNS-разрешения | `dns_error` | `dns_error` |
| Ошибка TLS-рукопожатия | `tls_error` | `tls_error` |
| Другая сетевая ошибка | классифицируется ядром | зависит от типа ошибки |

### Прямое использование чекера

```csharp
using DepHealth.Checks;

var checker = new HttpChecker(
    healthPath: "/healthz",
    tlsEnabled: true,
    tlsSkipVerify: false,
    bearerToken: "my-token");

// Использование через AddCustom:
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("payment-api", DependencyType.Http, "payment.svc", "443", checker,
        critical: true)
    .Build();
```

### Особенности поведения

- Использует `System.Net.Http.HttpClient`; новый экземпляр создаётся для каждой проверки
- Отправляет заголовок `User-Agent: dephealth/0.5.0`
- Пользовательские заголовки применяются после User-Agent и могут его перезаписать
- TLS включается при `tlsEnabled: true` или если URL начинается с `https://`
- Допускается только один метод аутентификации: `bearerToken`, `basicAuthUsername`/`basicAuthPassword`
  или ключ `Authorization` в пользовательских заголовках — совместное использование
  выбрасывает `ValidationException`

---

## gRPC

Проверяет gRPC-сервисы через
[gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

### Регистрация

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddGrpc("user-service", host: "user.svc", port: "9090", critical: true)
    .Build();
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `tlsEnabled` | `false` | Включить TLS (HTTPS) для gRPC-канала |
| `critical` | `null` | Критичность зависимости |
| `metadata` | `null` | Пользовательские метаданные gRPC (`Dictionary<string, string>`) |
| `bearerToken` | `null` | Установить метаданные `authorization: Bearer <token>` |
| `basicAuthUsername` | `null` | Имя пользователя для Basic-аутентификации |
| `basicAuthPassword` | `null` | Пароль для Basic-аутентификации |

### Полный пример

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Проверка с TLS
    .AddGrpc(
        name: "user-service",
        host: "user.svc",
        port: "9090",
        tlsEnabled: true,
        critical: true,
        metadata: new Dictionary<string, string>
        {
            ["x-request-id"] = "dephealth"
        })

    // Проверка без TLS (plain HTTP/2)
    .AddGrpc("grpc-gateway", host: "gateway.svc", port: "9090", critical: false)
    .Build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Ответ SERVING | `ok` | `ok` |
| gRPC UNAUTHENTICATED | `auth_error` | `auth_error` |
| gRPC PERMISSION_DENIED | `auth_error` | `auth_error` |
| Ответ NOT_SERVING | `unhealthy` | `grpc_not_serving` |
| Другой gRPC-статус | `unhealthy` | `grpc_unknown` |
| Ошибка соединения/RPC | классифицируется ядром | зависит от типа ошибки |

### Прямое использование чекера

```csharp
using DepHealth.Checks;

var checker = new GrpcChecker(
    tlsEnabled: true,
    bearerToken: "my-token");

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("user-service", DependencyType.Grpc, "user.svc", "9090", checker,
        critical: true)
    .Build();
```

### Особенности поведения

- Использует `GrpcChannel.ForAddress()` из `Grpc.Net.Client`
- Создаёт новый gRPC-канал для каждой проверки; канал освобождается сразу после вызова
- Проверяет состояние всего сервера (пустое имя сервиса в `HealthCheckRequest`)
- Допускается только один метод аутентификации: `bearerToken`, `basicAuthUsername`/`basicAuthPassword`
  или ключ `authorization` в пользовательских метаданных — совместное использование
  выбрасывает `ValidationException`

---

## TCP

Проверяет TCP-подключение: устанавливает соединение и немедленно закрывает.
Простейший чекер — без протокола прикладного уровня.

### Регистрация

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddTcp("memcached", host: "memcached.svc", port: "11211", critical: false)
    .Build();
```

### Опции

Нет специфичных опций. TCP-чекер не имеет состояния.

### Полный пример

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddTcp("memcached", host: "memcached.svc", port: "11211", critical: false)
    .AddTcp("custom-service", host: "custom.svc", port: "5555", critical: true)
    .Build();
```

### Классификация ошибок

TCP-чекер не производит специфичных ошибок. Все ошибки (отказ соединения,
DNS-ошибки, таймауты) классифицируются ядром.

### Прямое использование чекера

```csharp
using DepHealth.Checks;

var checker = new TcpChecker();

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("memcached", DependencyType.Tcp, "memcached.svc", "11211", checker,
        critical: false)
    .Build();
```

### Особенности поведения

- Использует `System.Net.Sockets.TcpClient.ConnectAsync()` — только TCP-рукопожатие,
  данные не передаются
- Соединение закрывается сразу после установки
- Подходит для сервисов без протокола проверки здоровья

---

## PostgreSQL

Проверяет PostgreSQL, выполняя `SELECT 1`. Поддерживает автономный режим
(новое соединение из строки подключения) и режим пула (существующий
`NpgsqlDataSource`).

### Регистрация

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddPostgres("postgres-main", url: "postgresql://user:pass@pg.svc:5432/mydb",
        critical: true)
    .Build();
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | обязательно | URL подключения к PostgreSQL; учётные данные и имя БД извлекаются автоматически |
| `critical` | `null` | Критичность зависимости |
| `labels` | `null` | Пользовательские метки Prometheus |

### Полный пример

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Автономный режим — строка подключения строится из URL
    .AddPostgres(
        name: "postgres-main",
        url: Environment.GetEnvironmentVariable("DATABASE_URL")!,
        critical: true)
    .Build();
```

### Режим пула

Использование существующего `NpgsqlDataSource`:

```csharp
using DepHealth;
using DepHealth.Checks;
using Npgsql;

// Создание или внедрение NpgsqlDataSource приложения
NpgsqlDataSource dataSource = NpgsqlDataSource.Create(connectionString);

var checker = new PostgresChecker(dataSource);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("postgres-main", DependencyType.Postgres,
        "pg.svc", "5432", checker, critical: true)
    .Build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Запрос успешен | `ok` | `ok` |
| SQL state 28000 (Invalid Authorization) | `auth_error` | `auth_error` |
| SQL state 28P01 (Authentication Failed) | `auth_error` | `auth_error` |
| Таймаут соединения | `timeout` | `timeout` |
| Отказ соединения | `connection_error` | `connection_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```csharp
using DepHealth.Checks;
using Npgsql;

// Автономный режим
var checker = new PostgresChecker(
    connectionString: "Host=pg.svc;Port=5432;Username=user;Password=pass;Database=mydb");

// Режим пула
NpgsqlDataSource dataSource = NpgsqlDataSource.Create(connectionString);
var poolChecker = new PostgresChecker(dataSource);
```

### Особенности поведения

- В автономном режиме строит Npgsql-строку подключения из URL: host, port, username,
  password и database извлекаются автоматически
- Режим пула вызывает `NpgsqlDataSource.OpenConnectionAsync()` — отражает реальную
  способность приложения получить соединение из пула
- Использует библиотеку `Npgsql` (`Npgsql.NpgsqlConnection`)
- Ошибки аутентификации определяются по `PostgresException.SqlState`

---

## MySQL

Проверяет MySQL, выполняя `SELECT 1`. Поддерживается только автономный режим;
для использования пользовательской строки подключения используйте `AddCustom`
с предварительно созданным `MySqlChecker`.

### Регистрация

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddMySql("mysql-main", url: "mysql://user:pass@mysql.svc:3306/mydb",
        critical: true)
    .Build();
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | обязательно | URL подключения к MySQL; учётные данные и имя БД извлекаются автоматически |
| `critical` | `null` | Критичность зависимости |
| `labels` | `null` | Пользовательские метки Prometheus |

### Полный пример

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddMySql(
        name: "mysql-main",
        url: "mysql://user:pass@mysql.svc:3306/mydb",
        critical: true)
    .Build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Запрос успешен | `ok` | `ok` |
| `MySqlErrorCode.AccessDenied` | `auth_error` | `auth_error` |
| Таймаут соединения | `timeout` | `timeout` |
| Отказ соединения | `connection_error` | `connection_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```csharp
using DepHealth.Checks;

var checker = new MySqlChecker(
    connectionString: "Server=mysql.svc;Port=3306;User=user;Password=pass;Database=mydb");

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("mysql-main", DependencyType.MySql, "mysql.svc", "3306", checker,
        critical: true)
    .Build();
```

### Особенности поведения

- Использует библиотеку `MySqlConnector` (`MySqlConnector.MySqlConnection`)
- В автономном режиме строит строку подключения из URL; учётные данные и имя базы
  данных извлекаются автоматически
- Ошибки аутентификации определяются по `MySqlException.ErrorCode == MySqlErrorCode.AccessDenied`
- Тот же интерфейс, что и у `PostgresChecker` для автономного режима

---

## Redis

Проверяет Redis, выполняя команду `PING` и ожидая ответ `PONG`. Поддерживает
автономный режим и режим пула через `IConnectionMultiplexer`.

### Регистрация

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddRedis("redis-cache", url: "redis://:password@redis.svc:6379/0",
        critical: false)
    .Build();
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | обязательно | URL подключения к Redis; пароль извлекается автоматически |
| `critical` | `null` | Критичность зависимости |
| `labels` | `null` | Пользовательские метки Prometheus |

### Полный пример

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Пароль из URL
    .AddRedis("redis-cache", url: "redis://:mypassword@redis.svc:6379", critical: false)

    // Без пароля
    .AddRedis("redis-sessions", url: "redis://redis-sessions.svc:6379", critical: true)
    .Build();
```

### Режим пула

```csharp
using DepHealth;
using DepHealth.Checks;
using StackExchange.Redis;

IConnectionMultiplexer multiplexer =
    await ConnectionMultiplexer.ConnectAsync("redis.svc:6379");

var checker = new RedisChecker(multiplexer);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("redis-cache", DependencyType.Redis, "redis.svc", "6379", checker,
        critical: false)
    .Build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| PING возвращает PONG | `ok` | `ok` |
| "NOAUTH" в сообщении ошибки | `auth_error` | `auth_error` |
| "WRONGPASS" в сообщении ошибки | `auth_error` | `auth_error` |
| "AUTH" в сообщении ошибки | `auth_error` | `auth_error` |
| Отказ соединения | `connection_error` | `connection_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```csharp
using DepHealth.Checks;
using StackExchange.Redis;

// Автономный режим
var checker = new RedisChecker(
    connectionString: "redis.svc:6379,connectTimeout=5000,abortConnect=true,password=secret");

// Режим пула
IConnectionMultiplexer mux = await ConnectionMultiplexer.ConnectAsync("redis.svc:6379");
var poolChecker = new RedisChecker(mux);
```

### Особенности поведения

- Использует библиотеку `StackExchange.Redis` (`IConnectionMultiplexer`)
- В автономном режиме создаёт новый `ConnectionMultiplexer`, выполняет `PING`,
  затем освобождает его
- В режиме пула вызывает `IConnectionMultiplexer.GetDatabase().PingAsync()`
  на предоставленном экземпляре
- Ошибки аутентификации определяются по наличию подстрок `NOAUTH`, `WRONGPASS`
  или `AUTH` в сообщении исключения

---

## AMQP (RabbitMQ)

Проверяет AMQP-брокеры, устанавливая соединение и немедленно закрывая.
Поддерживается только автономный режим.

### Регистрация

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddAmqp("rabbitmq", url: "amqp://user:pass@rabbitmq.svc:5672/myvhost",
        critical: false)
    .Build();
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | обязательно | URL подключения к AMQP; имя пользователя, пароль и виртуальный хост извлекаются автоматически |
| `critical` | `null` | Критичность зависимости |
| `labels` | `null` | Пользовательские метки Prometheus |

### Полный пример

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Виртуальный хост по умолчанию "/"
    .AddAmqp("rabbitmq", url: "amqp://user:pass@rabbitmq.svc:5672", critical: false)

    // Пользовательский виртуальный хост
    .AddAmqp("rabbitmq-prod",
        url: "amqp://myuser:mypass@rmq-prod.svc:5672/myvhost",
        critical: true)
    .Build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Соединение установлено и открыто | `ok` | `ok` |
| "403" в сообщении ошибки | `auth_error` | `auth_error` |
| "ACCESS_REFUSED" в сообщении ошибки | `auth_error` | `auth_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```csharp
using DepHealth.Checks;

var checker = new AmqpChecker(
    username: "myuser",
    password: "mypass",
    vhost: "myvhost");

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("rabbitmq", DependencyType.Amqp, "rabbitmq.svc", "5672", checker,
        critical: false)
    .Build();
```

### Особенности поведения

- Использует библиотеку `RabbitMQ.Client` (`RabbitMQ.Client.ConnectionFactory`)
- Нет режима пула — всегда создаётся новое соединение
- Соединение закрывается сразу после успешного установления
- Виртуальный хост по умолчанию — `/`, если не задан в URL
- Ошибки аутентификации определяются по наличию подстрок `403` или `ACCESS_REFUSED`
  в сообщении исключения

---

## Kafka

Проверяет Kafka-брокеры, подключаясь и запрашивая метаданные кластера.
Чекер без состояния — нет опций конфигурации помимо URL.

### Регистрация

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddKafka("kafka", url: "kafka://kafka.svc:9092", critical: true)
    .Build();
```

### Опции

Нет специфичных опций. Kafka-чекер не имеет состояния.

### Полный пример

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddKafka("kafka", url: "kafka://kafka.svc:9092", critical: true)
    .Build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Метаданные содержат брокеры | `ok` | `ok` |
| Нет брокеров в метаданных | `unhealthy` | `no_brokers` |
| Ошибка соединения/метаданных | классифицируется ядром | зависит от типа ошибки |

### Прямое использование чекера

```csharp
using DepHealth.Checks;

var checker = new KafkaChecker();

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("kafka", DependencyType.Kafka, "kafka.svc", "9092", checker,
        critical: true)
    .Build();
```

### Особенности поведения

- Использует библиотеку `Confluent.Kafka` (`Confluent.Kafka.AdminClientBuilder`)
- Создаёт новый `AdminClient`, вызывает `GetMetadata(timeout: 5s)`, затем освобождает клиент
- Проверяет наличие хотя бы одного брокера в ответе метаданных
- `SocketTimeoutMs` установлен равным 5000 мс
- Нет поддержки аутентификации (только plain TCP)

---

## LDAP

Проверяет LDAP-серверы каталогов. Поддерживает 4 метода проверки, 3
протокола соединения и как автономный режим, так и режим пула.

### Регистрация

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddLdap("ldap", host: "ldap.svc", port: "389", critical: true)
    .Build();
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `checkMethod` | `LdapCheckMethod.RootDse` | Метод проверки (см. ниже) |
| `bindDN` | `""` | Bind DN для `SimpleBind` или `Search` |
| `bindPassword` | `""` | Пароль bind |
| `baseDN` | `""` | Базовый DN для `Search` (обязателен для метода `Search`) |
| `searchFilter` | `(objectClass=*)` | LDAP-фильтр поиска |
| `searchScope` | `LdapSearchScope.Base` | Область поиска: `Base`, `One` или `Sub` |
| `useTls` | `false` | Использовать LDAPS (TLS с начала соединения) |
| `startTls` | `false` | Использовать StartTLS (обновление plain-соединения до TLS) |
| `tlsSkipVerify` | `false` | Пропустить проверку TLS-сертификата |
| `critical` | `null` | Критичность зависимости |
| `labels` | `null` | Пользовательские метки Prometheus |

### Методы проверки

| Метод | Описание |
| --- | --- |
| `AnonymousBind` | Выполняет анонимный LDAP bind (пустые DN и пароль) |
| `SimpleBind` | Выполняет bind с `bindDN` и `bindPassword` (оба обязательны) |
| `RootDse` | Запрашивает запись Root DSE (по умолчанию; работает без аутентификации) |
| `Search` | Выполняет LDAP-поиск с `baseDN`, фильтром и областью |

### Протоколы соединения

| Протокол | Порт | `useTls` | `startTls` | TLS |
| --- | --- | --- | --- | --- |
| Plain LDAP | 389 | `false` | `false` | Нет |
| LDAPS | 636 | `true` | `false` | Да (с начала соединения) |
| StartTLS | 389 | `false` | `true` | Да (обновление после соединения) |

### Полные примеры

**RootDse (по умолчанию)** — простейшая проверка, учётные данные не нужны:

```csharp
.AddLdap("ldap", host: "ldap.svc", port: "389", critical: true)
```

**AnonymousBind** — проверка, что анонимный доступ разрешён:

```csharp
.AddLdap("ldap", host: "ldap.svc", port: "389",
    checkMethod: LdapCheckMethod.AnonymousBind,
    critical: true)
```

**SimpleBind** — проверка учётных данных:

```csharp
.AddLdap("ldap", host: "ldap.svc", port: "389",
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=admin,dc=example,dc=org",
    bindPassword: "secret",
    critical: true)
```

**Search** — выполнение аутентифицированного поиска:

```csharp
.AddLdap("ldap", host: "ldap.svc", port: "389",
    checkMethod: LdapCheckMethod.Search,
    bindDN: "cn=readonly,dc=example,dc=org",
    bindPassword: "pass",
    baseDN: "ou=users,dc=example,dc=org",
    searchFilter: "(uid=healthcheck)",
    searchScope: LdapSearchScope.One,
    critical: true)
```

**LDAPS** — TLS с начала соединения:

```csharp
.AddLdap("ldap-secure", host: "ldap.svc", port: "636",
    useTls: true,
    tlsSkipVerify: true,
    critical: true)
```

**StartTLS** — обновление plain-соединения до TLS:

```csharp
.AddLdap("ldap-starttls", host: "ldap.svc", port: "389",
    startTls: true,
    critical: true)
```

### Режим пула

Использование существующего `ILdapConnection`:

```csharp
using DepHealth;
using DepHealth.Checks;
using Novell.Directory.Ldap;

var ldapConn = new LdapConnection();
ldapConn.Connect("ldap.svc", 389);

var checker = new LdapChecker(
    connection: ldapConn,
    checkMethod: LdapCheckMethod.RootDse);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("ldap", DependencyType.Ldap, "ldap.svc", "389", checker, critical: true)
    .Build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Проверка успешна | `ok` | `ok` |
| ResultCode 49 (INVALID_CREDENTIALS) | `auth_error` | `auth_error` |
| ResultCode 50 (INSUFFICIENT_ACCESS_RIGHTS) | `auth_error` | `auth_error` |
| ResultCode 51 (BUSY) | `unhealthy` | `unhealthy` |
| ResultCode 52 (UNAVAILABLE) | `unhealthy` | `unhealthy` |
| ResultCode 53 (UNWILLING_TO_PERFORM) | `unhealthy` | `unhealthy` |
| ResultCode 81 (SERVER_DOWN) | `connection_error` | `connection_error` |
| ResultCode 91 (CONNECT_ERROR) | `connection_error` | `connection_error` |
| Отказ соединения | `connection_error` | `connection_error` |
| Ошибка TLS / SSL / сертификата | `tls_error` | `tls_error` |
| ResultCode 85 (LDAP_TIMEOUT) | `timeout` | `timeout` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```csharp
using DepHealth.Checks;

// Автономный режим
var checker = new LdapChecker(
    checkMethod: LdapCheckMethod.RootDse);

// Автономный режим с SimpleBind
var bindChecker = new LdapChecker(
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=admin,dc=example,dc=org",
    bindPassword: "secret");
```

### Валидация конфигурации

Конструктор `LdapChecker` выполняет валидацию при создании:

- `SimpleBind` требует непустых `bindDN` и `bindPassword`
- `Search` требует непустого `baseDN`
- `startTls: true` несовместим с `useTls: true` — выбрасывается `ValidationException`

### Особенности поведения

- Использует библиотеку `Novell.Directory.Ldap` (`Novell.Directory.Ldap.NETStandard`)
- В автономном режиме создаёт новый `LdapConnection` для каждой проверки;
  отключается после завершения проверки
- В режиме пула использует предоставленный `ILdapConnection` (вызывающий управляет жизненным циклом)
- `RootDse` запрашивает атрибуты `namingContexts` и `subschemaSubentry` с `MaxResults = 1`
- Метод `Search` ограничивает результаты 1 записью (`MaxResults = 1`)
- `SearchWithConfig` выполняет bind перед поиском, если `bindDN` непустой

---

## Сводка классификации ошибок

Все чекеры классифицируют ошибки по категориям статусов. Классификатор
ядра обрабатывает общие типы ошибок (таймауты, DNS-ошибки, TLS-ошибки,
отказ соединения). Чекер-специфичная классификация добавляет детали
на уровне протокола:

| Категория статуса | Значение | Типичные причины |
| --- | --- | --- |
| `ok` | Зависимость здорова | Проверка успешна |
| `timeout` | Таймаут проверки | Медленная сеть, перегруженный сервис |
| `connection_error` | Не удаётся подключиться | Сервис не работает, неверный host/port, firewall |
| `dns_error` | Ошибка DNS-разрешения | Неверный hostname, DNS-авария |
| `auth_error` | Ошибка аутентификации | Неверные учётные данные, истёкший токен |
| `tls_error` | Ошибка TLS-рукопожатия | Невалидный сертификат, ошибка конфигурации TLS |
| `unhealthy` | Подключён, но нездоров | Сервис сообщает о проблеме, возвращает код ошибки |
| `error` | Неожиданная ошибка | Неклассифицированные сбои |

## См. также

- [Справочник API](api-reference.ru.md) — полный справочник API `DepHealthMonitor` и типов
- [API Reference (English)](api-reference.md) — полный справочник API на английском
