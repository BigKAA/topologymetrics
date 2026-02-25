*[English version](checkers.md)*

# Чекеры

Java SDK включает 9 встроенных чекеров для распространённых типов зависимостей.
Каждый чекер реализует интерфейс `HealthChecker` и может использоваться через
высокоуровневый API (`DepHealth.builder().dependency(...)`) или напрямую
через свой билдер.

## HTTP

Проверяет HTTP-эндпоинты, отправляя GET-запрос и ожидая ответ с кодом 2xx.

### Регистрация

```java
.dependency("api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true))
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `.httpHealthPath(path)` | `/health` | Путь для эндпоинта проверки |
| `.httpTls(enabled)` | `false` | Использовать HTTPS вместо HTTP (определяется автоматически из `https://` URL) |
| `.httpTlsSkipVerify(skip)` | `false` | Пропустить проверку TLS-сертификата |
| `.httpHeaders(headers)` | -- | Пользовательские HTTP-заголовки (`Map<String, String>`) |
| `.httpBearerToken(token)` | -- | Установить заголовок `Authorization: Bearer <token>` |
| `.httpBasicAuth(user, pass)` | -- | Установить заголовок `Authorization: Basic <base64>` |

### Полный пример

```java
import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.DependencyType;
import io.micrometer.prometheus.PrometheusMeterRegistry;

import java.util.Map;

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("payment-api", DependencyType.HTTP, d -> d
        .url("https://payment.svc:443")
        .critical(true)
        .httpHealthPath("/healthz")
        .httpTls(true)
        .httpTlsSkipVerify(true)
        .httpHeaders(Map.of("X-Request-Source", "dephealth")))
    .build();

dh.start();
// ...
dh.stop();
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

```java
import biz.kryukov.dev.dephealth.checks.HttpHealthChecker;

HttpHealthChecker checker = HttpHealthChecker.builder()
    .healthPath("/healthz")
    .tlsEnabled(true)
    .tlsSkipVerify(false)
    .build();
```

### Особенности поведения

- Использует `java.net.http.HttpClient` с политикой редиректов `NORMAL`
  (автоматически следует 3xx)
- Создаёт новый HTTP-клиент для каждой проверки
- Отправляет заголовок `User-Agent: dephealth/0.5.0`
- Пользовательские заголовки применяются после User-Agent и могут его
  перезаписать
- Допускается только один метод аутентификации: `bearerToken`, `basicAuth`
  или ключ `Authorization` в пользовательских заголовках

---

## gRPC

Проверяет gRPC-сервисы через
[gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

### Регистрация

```java
.dependency("user-service", DependencyType.GRPC, d -> d
    .host("user.svc")
    .port("9090")
    .critical(true))
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `.grpcServiceName(name)` | `""` (пусто) | Имя сервиса; пустое -- проверка всего сервера |
| `.grpcTls(enabled)` | `false` | Включить TLS |
| `.grpcMetadata(md)` | -- | Пользовательские метаданные gRPC (`Map<String, String>`) |
| `.grpcBearerToken(token)` | -- | Установить метаданные `authorization: Bearer <token>` |
| `.grpcBasicAuth(user, pass)` | -- | Установить метаданные `authorization: Basic <base64>` |

### Полный пример

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Проверка конкретного gRPC-сервиса
    .dependency("user-service", DependencyType.GRPC, d -> d
        .host("user.svc")
        .port("9090")
        .critical(true)
        .grpcServiceName("user.v1.UserService")
        .grpcTls(true)
        .grpcMetadata(Map.of("x-request-id", "dephealth")))

    // Проверка состояния всего сервера (пустое имя сервиса)
    .dependency("grpc-gateway", DependencyType.GRPC, d -> d
        .host("gateway.svc")
        .port("9090")
        .critical(false))
    .build();
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

```java
import biz.kryukov.dev.dephealth.checks.GrpcHealthChecker;

GrpcHealthChecker checker = GrpcHealthChecker.builder()
    .serviceName("user.v1.UserService")
    .tlsEnabled(true)
    .build();
```

### Особенности поведения

- Использует `passthrough:///` resolver (по умолчанию для
  `ManagedChannelBuilder.forTarget()`) для обхода DNS SRV-запросов;
  критично в Kubernetes, где `ndots:5` вызывает высокую задержку
  при использовании `dns:///` resolver
- Создаёт новый gRPC-канал для каждой проверки; канал закрывается
  сразу после вызова
- Пустое имя сервиса проверяет состояние всего сервера
- Допускается только один метод аутентификации: `bearerToken`, `basicAuth`
  или ключ `authorization` в пользовательских метаданных

---

## TCP

Проверяет TCP-подключение: устанавливает сокет-соединение и немедленно
закрывает. Простейший чекер -- без протокола прикладного уровня.

### Регистрация

```java
.dependency("memcached", DependencyType.TCP, d -> d
    .host("memcached.svc")
    .port("11211")
    .critical(false))
```

Или с использованием URL:

```java
.dependency("memcached", DependencyType.TCP, d -> d
    .url("tcp://memcached.svc:11211")
    .critical(false))
```

### Опции

Нет специфичных опций. TCP-чекер не имеет состояния.

### Полный пример

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("memcached", DependencyType.TCP, d -> d
        .host("memcached.svc")
        .port("11211")
        .critical(false))
    .dependency("custom-service", DependencyType.TCP, d -> d
        .host("custom.svc")
        .port("5555")
        .critical(true))
    .build();
```

### Классификация ошибок

TCP-чекер не производит специфичных ошибок. Все ошибки (отказ соединения,
DNS-ошибки, таймауты) классифицируются ядром.

### Прямое использование чекера

```java
import biz.kryukov.dev.dephealth.checks.TcpHealthChecker;

TcpHealthChecker checker = new TcpHealthChecker();
```

### Особенности поведения

- Выполняет только TCP-рукопожатие (SYN/ACK) -- данные не отправляются
  и не принимаются
- Соединение закрывается сразу после установки
- Использует `java.net.Socket` с настроенным таймаутом
- Подходит для сервисов без протокола проверки здоровья

---

## PostgreSQL

Проверяет PostgreSQL, выполняя запрос (по умолчанию `SELECT 1`).
Поддерживает автономный режим (новое JDBC-соединение) и режим пула
(существующий `DataSource`).

### Регистрация

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://user:pass@pg.svc:5432/mydb")
    .critical(true))
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `.dbUsername(user)` | -- | Имя пользователя БД (также извлекается из URL) |
| `.dbPassword(pass)` | -- | Пароль БД (также извлекается из URL) |
| `.dbDatabase(name)` | -- | Имя базы данных (также извлекается из пути URL) |
| `.dbQuery(query)` | `SELECT 1` | Пользовательский SQL-запрос для проверки |
| `.dataSource(ds)` | -- | Пул соединений DataSource (предпочтительно) |

### Полный пример

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Автономный режим -- создаёт новое JDBC-соединение для каждой проверки
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true)
        .dbQuery("SELECT 1"))
    .build();
```

### Режим пула

Использование существующего пула соединений:

```java
import javax.sql.DataSource;

// Предположим, 'dataSource' -- пул вашего приложения (HikariCP и т.д.)
DataSource dataSource = ...;

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true)
        .dataSource(dataSource))
    .build();
```

Или с предварительно созданным чекером:

```java
import biz.kryukov.dev.dephealth.checks.PostgresHealthChecker;

PostgresHealthChecker checker = PostgresHealthChecker.builder()
    .dataSource(dataSource)
    .build();

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("postgres-main", DependencyType.POSTGRES, checker, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true))
    .build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Запрос успешен | `ok` | `ok` |
| SQLSTATE 28000 (неверная авторизация) | `auth_error` | `auth_error` |
| SQLSTATE 28P01 (ошибка аутентификации) | `auth_error` | `auth_error` |
| "password authentication failed" в ошибке | `auth_error` | `auth_error` |
| Таймаут соединения | `timeout` | `timeout` |
| Отказ соединения | `connection_error` | `connection_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```java
import biz.kryukov.dev.dephealth.checks.PostgresHealthChecker;

// Автономный режим
PostgresHealthChecker checker = PostgresHealthChecker.builder()
    .username("user")
    .password("pass")
    .database("mydb")
    .build();

// Режим пула
PostgresHealthChecker poolChecker = PostgresHealthChecker.builder()
    .dataSource(dataSource)
    .build();
```

### Особенности поведения

- В автономном режиме строит JDBC URL `jdbc:postgresql://host:port/database`
- Режим пула переиспользует существующий пул -- отражает реальную
  способность сервиса работать с зависимостью
- Использует `DriverManager.getConnection()` для автономного режима
- Учётные данные из URL извлекаются автоматически, если не заданы явно

---

## MySQL

Проверяет MySQL, выполняя запрос (по умолчанию `SELECT 1`). Поддерживает
автономный режим и режим пула.

### Регистрация

```java
.dependency("mysql-main", DependencyType.MYSQL, d -> d
    .url("mysql://user:pass@mysql.svc:3306/mydb")
    .critical(true))
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `.dbUsername(user)` | -- | Имя пользователя БД (также извлекается из URL) |
| `.dbPassword(pass)` | -- | Пароль БД (также извлекается из URL) |
| `.dbDatabase(name)` | -- | Имя базы данных (также извлекается из пути URL) |
| `.dbQuery(query)` | `SELECT 1` | Пользовательский SQL-запрос для проверки |
| `.dataSource(ds)` | -- | Пул соединений DataSource (предпочтительно) |

### Полный пример

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("mysql-main", DependencyType.MYSQL, d -> d
        .url("mysql://user:pass@mysql.svc:3306/mydb")
        .critical(true))
    .build();
```

### Режим пула

```java
import javax.sql.DataSource;

DataSource dataSource = ...;

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("mysql-main", DependencyType.MYSQL, d -> d
        .url("mysql://mysql.svc:3306/mydb")
        .critical(true)
        .dataSource(dataSource))
    .build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Запрос успешен | `ok` | `ok` |
| MySQL ошибка 1045 (Access Denied) | `auth_error` | `auth_error` |
| "Access denied" в сообщении ошибки | `auth_error` | `auth_error` |
| Таймаут соединения | `timeout` | `timeout` |
| Отказ соединения | `connection_error` | `connection_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```java
import biz.kryukov.dev.dephealth.checks.MysqlHealthChecker;

// Автономный режим
MysqlHealthChecker checker = MysqlHealthChecker.builder()
    .username("user")
    .password("pass")
    .database("mydb")
    .build();

// Режим пула
MysqlHealthChecker poolChecker = MysqlHealthChecker.builder()
    .dataSource(dataSource)
    .build();
```

### Особенности поведения

- В автономном режиме строит JDBC URL `jdbc:mysql://host:port/database`
- Использует `DriverManager.getConnection()` для автономного режима
- Тот же интерфейс, что и у PostgreSQL-чекера (оба используют `DataSource`
  для режима пула)
- Учётные данные из URL извлекаются автоматически, если не заданы явно

---

## Redis

Проверяет Redis, выполняя команду `PING` и ожидая ответ `PONG`.
Поддерживает автономный режим и режим пула.

### Регистрация

```java
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://:password@redis.svc:6379/0")
    .critical(false))
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `.redisPassword(password)` | -- | Пароль для автономного режима (также извлекается из URL) |
| `.redisDb(db)` | `0` | Индекс базы данных для автономного режима (также извлекается из пути URL) |
| `.jedisPool(pool)` | -- | JedisPool для интеграции с пулом соединений (предпочтительно) |

### Полный пример

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Пароль из URL
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .url("redis://:mypassword@redis.svc:6379/0")
        .critical(false))

    // Пароль через опцию
    .dependency("redis-sessions", DependencyType.REDIS, d -> d
        .host("redis-sessions.svc")
        .port("6379")
        .redisPassword("secret")
        .redisDb(1)
        .critical(true))
    .build();
```

### Режим пула

```java
import redis.clients.jedis.JedisPool;

JedisPool jedisPool = new JedisPool("redis.svc", 6379);

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .host("redis.svc")
        .port("6379")
        .critical(false)
        .jedisPool(jedisPool))
    .build();
```

Или с предварительно созданным чекером:

```java
import biz.kryukov.dev.dephealth.checks.RedisHealthChecker;

RedisHealthChecker checker = RedisHealthChecker.builder()
    .jedisPool(jedisPool)
    .build();

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("redis-cache", DependencyType.REDIS, checker, d -> d
        .host("redis.svc")
        .port("6379")
        .critical(false))
    .build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| PING возвращает PONG | `ok` | `ok` |
| "NOAUTH" в ошибке | `auth_error` | `auth_error` |
| "WRONGPASS" в ошибке | `auth_error` | `auth_error` |
| Отказ соединения | `connection_error` | `connection_error` |
| Таймаут соединения | `connection_error` | `connection_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```java
import biz.kryukov.dev.dephealth.checks.RedisHealthChecker;

// Автономный режим
RedisHealthChecker checker = RedisHealthChecker.builder()
    .password("pass")
    .database(0)
    .build();

// Режим пула
RedisHealthChecker poolChecker = RedisHealthChecker.builder()
    .jedisPool(jedisPool)
    .build();
```

### Особенности поведения

- Использует библиотеку Jedis для связи с Redis
- В автономном режиме создаёт новое `Jedis`-соединение с настроенным
  таймаутом
- В режиме пула использует `jedisPool.getResource()` и закрывает ресурс
  после проверки
- Пароль из опций имеет приоритет над паролем из URL
- Индекс БД из опций имеет приоритет над индексом из URL

---

## AMQP (RabbitMQ)

Проверяет AMQP-брокеры, устанавливая соединение и немедленно закрывая.
Поддерживается только автономный режим.

### Регистрация

```java
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .host("rabbitmq.svc")
    .port("5672")
    .critical(false))
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `.amqpUrl(url)` | -- | Полный AMQP URL (переопределяет host/port/учётные данные) |
| `.amqpUsername(user)` | -- | Имя пользователя AMQP (также извлекается из URL) |
| `.amqpPassword(pass)` | -- | Пароль AMQP (также извлекается из URL) |
| `.amqpVirtualHost(vhost)` | -- | Виртуальный хост AMQP (также извлекается из пути URL) |

### Полный пример

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Учётные данные по умолчанию (guest:guest)
    .dependency("rabbitmq", DependencyType.AMQP, d -> d
        .host("rabbitmq.svc")
        .port("5672")
        .critical(false))

    // Пользовательские учётные данные
    .dependency("rabbitmq-prod", DependencyType.AMQP, d -> d
        .host("rmq-prod.svc")
        .port("5672")
        .amqpUrl("amqp://myuser:mypass@rmq-prod.svc:5672/myvhost")
        .critical(true))
    .build();
```

Или с учётными данными из URL:

```java
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .url("amqp://user:pass@rabbitmq.svc:5672/myvhost")
    .critical(true))
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Соединение установлено и открыто | `ok` | `ok` |
| "403" в ошибке | `auth_error` | `auth_error` |
| "ACCESS_REFUSED" в ошибке | `auth_error` | `auth_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```java
import biz.kryukov.dev.dephealth.checks.AmqpHealthChecker;

AmqpHealthChecker checker = AmqpHealthChecker.builder()
    .username("myuser")
    .password("mypass")
    .virtualHost("/myvhost")
    .build();
```

### Особенности поведения

- Использует RabbitMQ `ConnectionFactory` из `com.rabbitmq.client`
- Нет режима пула -- всегда создаёт новое соединение
- Соединение закрывается сразу после успешного установления
- Когда установлен `amqpUrl`, он переопределяет host/port/учётные данные
  из `ConnectionFactory`
- Имя соединения устанавливается как `"dephealth-check"` для удобной
  идентификации в консоли управления RabbitMQ

---

## Kafka

Проверяет Kafka-брокеры, подключаясь и запрашивая метаданные кластера через
`AdminClient.describeCluster().nodes()`. Чекер без состояния, без опций
конфигурации.

### Регистрация

```java
.dependency("kafka", DependencyType.KAFKA, d -> d
    .host("kafka.svc")
    .port("9092")
    .critical(true))
```

### Опции

Нет специфичных опций. Kafka-чекер не имеет состояния.

### Полный пример

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("kafka", DependencyType.KAFKA, d -> d
        .host("kafka.svc")
        .port("9092")
        .critical(true))
    .build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Метаданные содержат узлы | `ok` | `ok` |
| Нет узлов в метаданных | `unhealthy` | `no_brokers` |
| Ошибка соединения/метаданных | классифицируется ядром | зависит от типа ошибки |

### Прямое использование чекера

```java
import biz.kryukov.dev.dephealth.checks.KafkaHealthChecker;

KafkaHealthChecker checker = new KafkaHealthChecker();
```

### Особенности поведения

- Использует Apache Kafka `AdminClient` из `org.apache.kafka.clients.admin`
- Создаёт новый `AdminClient`, запрашивает метаданные кластера, закрывает
  клиент
- Проверяет наличие хотя бы одного узла в ответе `describeCluster`
- `REQUEST_TIMEOUT_MS` и `DEFAULT_API_TIMEOUT_MS` устанавливаются равными
  таймауту проверки
- Нет поддержки аутентификации (только plain TCP)

---

## LDAP

Проверяет LDAP-серверы каталогов. Поддерживает 4 метода проверки, 3
протокола соединения и как автономный режим, так и режим пула.
Добавлен в v0.8.0.

### Регистрация

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .critical(true))
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `.ldapCheckMethod(method)` | `ROOT_DSE` | Метод проверки (см. ниже) |
| `.ldapBindDN(dn)` | -- | Bind DN для SIMPLE_BIND или аутентификации SEARCH |
| `.ldapBindPassword(pass)` | -- | Пароль bind |
| `.ldapBaseDN(dn)` | -- | Базовый DN для операций SEARCH (обязателен для метода SEARCH) |
| `.ldapSearchFilter(filter)` | `(objectClass=*)` | LDAP-фильтр поиска |
| `.ldapSearchScope(scope)` | `BASE` | Область поиска: `BASE`, `ONE` или `SUB` |
| `.ldapStartTLS(enabled)` | `false` | Включить StartTLS (несовместимо с `ldaps://`) |
| `.ldapTlsSkipVerify(skip)` | `false` | Пропустить проверку TLS-сертификата |
| `.ldapConnection(conn)` | -- | Существующий `LDAPConnection` для интеграции с пулом |

### Методы проверки

| Метод | Описание |
| --- | --- |
| `ANONYMOUS_BIND` | Выполняет анонимный LDAP bind (пустые DN и пароль) |
| `SIMPLE_BIND` | Выполняет bind с `bindDN` и `bindPassword` (оба обязательны) |
| `ROOT_DSE` | Запрашивает запись Root DSE (по умолчанию; работает без аутентификации) |
| `SEARCH` | Выполняет LDAP-поиск с `baseDN`, фильтром и областью |

### Протоколы соединения

| Протокол | Схема URL | Порт по умолчанию | TLS |
| --- | --- | --- | --- |
| Plain LDAP | `ldap://` | 389 | Нет |
| LDAPS | `ldaps://` | 636 | Да (с начала соединения) |
| StartTLS | `ldap://` + `.ldapStartTLS(true)` | 389 | Обновление после соединения |

### Полные примеры

**ROOT_DSE (по умолчанию)** -- простейшая проверка, учётные данные не нужны:

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .critical(true))
```

**ANONYMOUS_BIND** -- проверка, что анонимный доступ разрешён:

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod(LdapHealthChecker.CheckMethod.ANONYMOUS_BIND)
    .critical(true))
```

**SIMPLE_BIND** -- проверка учётных данных:

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod(LdapHealthChecker.CheckMethod.SIMPLE_BIND)
    .ldapBindDN("cn=admin,dc=example,dc=org")
    .ldapBindPassword("secret")
    .critical(true))
```

**SEARCH** -- выполнение аутентифицированного поиска:

```java
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod(LdapHealthChecker.CheckMethod.SEARCH)
    .ldapBindDN("cn=readonly,dc=example,dc=org")
    .ldapBindPassword("pass")
    .ldapBaseDN("ou=users,dc=example,dc=org")
    .ldapSearchFilter("(uid=healthcheck)")
    .ldapSearchScope(LdapHealthChecker.LdapSearchScope.ONE)
    .critical(true))
```

**LDAPS** -- TLS с начала соединения:

```java
.dependency("ldap-secure", DependencyType.LDAP, d -> d
    .url("ldaps://ldap.svc:636")
    .ldapTlsSkipVerify(true)
    .critical(true))
```

**StartTLS** -- обновление plain LDAP-соединения до TLS:

```java
.dependency("ldap-starttls", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapStartTLS(true)
    .critical(true))
```

### Режим пула

Использование существующего LDAP-соединения:

```java
import com.unboundid.ldap.sdk.LDAPConnection;

LDAPConnection ldapConn = new LDAPConnection("ldap.svc", 389);

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("ldap", DependencyType.LDAP, d -> d
        .url("ldap://ldap.svc:389")
        .critical(true)
        .ldapConnection(ldapConn))
    .build();
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Проверка успешна | `ok` | `ok` |
| ResultCode 49 (INVALID_CREDENTIALS) | `auth_error` | `auth_error` |
| ResultCode 50 (INSUFFICIENT_ACCESS_RIGHTS) | `auth_error` | `auth_error` |
| ResultCode 51/52/53 (BUSY/UNAVAILABLE/UNWILLING_TO_PERFORM) | `unhealthy` | `unhealthy` |
| CONNECT_ERROR / SERVER_DOWN | `connection_error` | `connection_error` |
| Отказ соединения | `connection_error` | `connection_error` |
| Ошибка TLS / SSL / сертификата | `tls_error` | `tls_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Прямое использование чекера

```java
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker;

LdapHealthChecker checker = LdapHealthChecker.builder()
    .checkMethod(LdapHealthChecker.CheckMethod.ROOT_DSE)
    .build();
```

### Валидация конфигурации

Билдер выполняет валидацию при сборке:

- `SIMPLE_BIND` требует `bindDN` и `bindPassword`
- `SEARCH` требует `baseDN`
- `startTLS(true)` несовместим с `ldaps://` URL (useTLS)

### Особенности поведения

- Использует UnboundID LDAP SDK (`com.unboundid.ldap.sdk`)
- В автономном режиме создаёт новое LDAP-соединение для каждой проверки
- В режиме пула использует предоставленный `LDAPConnection` (вызывающий
  управляет жизненным циклом)
- `followReferrals` отключён для соединений проверки здоровья
- Операции поиска ограничены 1 результатом (`setSizeLimit(1)`)
- ROOT_DSE запрашивает атрибуты `namingContexts` и `subschemaSubentry`

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

- [Конфигурация](configuration.ru.md) -- все опции, значения по умолчанию и переменные окружения
- [Аутентификация](authentication.ru.md) -- подробное руководство по авторизации для HTTP и gRPC
- [Пулы соединений](connection-pools.ru.md) -- режим пула через DataSource и JedisPool
- [Метрики](metrics.ru.md) -- справочник Prometheus-метрик и примеры PromQL
- [Устранение неполадок](troubleshooting.ru.md) -- типичные проблемы и решения
