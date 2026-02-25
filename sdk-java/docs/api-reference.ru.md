*[English version](api-reference.md)*

# API Reference

Полный справочник всех публичных классов, интерфейсов и методов
Java SDK.

## Структура пакетов

```text
biz.kryukov.dev.dephealth           -- Core API
biz.kryukov.dev.dephealth.model     -- Классы моделей (Dependency, Endpoint и др.)
biz.kryukov.dev.dephealth.checks    -- Реализации HealthChecker
biz.kryukov.dev.dephealth.metrics   -- MetricsExporter
biz.kryukov.dev.dephealth.scheduler -- CheckScheduler
biz.kryukov.dev.dephealth.parser    -- ConfigParser
biz.kryukov.dev.dephealth.spring    -- Интеграция со Spring Boot
```

---

## DepHealth

**Пакет:** `biz.kryukov.dev.dephealth`

Главная точка входа. Объединяет экспорт метрик и планирование проверок.

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `builder` | `static Builder builder(String name, String group, MeterRegistry registry)` | Создать новый билдер |
| `start` | `void start()` | Запустить периодические проверки |
| `stop` | `void stop()` | Остановить проверки и освободить ресурсы |
| `health` | `Map<String, Boolean> health()` | Быстрая карта здоровья (ключ: `dep/host:port`) |
| `healthDetails` | `Map<String, EndpointStatus> healthDetails()` | Детальный статус по каждому эндпоинту |
| `addEndpoint` | `void addEndpoint(String depName, DependencyType depType, boolean critical, Endpoint ep, HealthChecker checker)` | Добавить эндпоинт в рантайме |
| `removeEndpoint` | `void removeEndpoint(String depName, String host, String port)` | Удалить эндпоинт в рантайме |
| `updateEndpoint` | `void updateEndpoint(String depName, String oldHost, String oldPort, Endpoint newEp, HealthChecker checker)` | Заменить эндпоинт атомарно |

### DepHealth.Builder

Билдер для экземпляров `DepHealth`. Создаётся через `DepHealth.builder()`.

#### Глобальные опции

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `checkInterval` | `Builder checkInterval(Duration interval)` | Глобальный интервал проверок (по умолчанию 15s) |
| `timeout` | `Builder timeout(Duration timeout)` | Глобальный тайм-аут проверок (по умолчанию 5s) |
| `dependency` | `Builder dependency(String name, DependencyType type, Consumer<DependencyBuilder> config)` | Добавить зависимость с полной конфигурацией |
| `build` | `DepHealth build()` | Валидировать конфигурацию и создать экземпляр |

#### Методы-сокращения

Удобные методы, создающие зависимость с одним эндпоинтом,
распарсенным из URL.

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `postgres` | `Builder postgres(String name, String url, boolean critical)` | Добавить зависимость PostgreSQL |
| `redis` | `Builder redis(String name, String url, boolean critical)` | Добавить зависимость Redis |
| `http` | `Builder http(String name, String url, boolean critical)` | Добавить HTTP-зависимость (чекер по умолчанию) |
| `http` | `Builder http(String name, String url, boolean critical, HealthChecker checker)` | Добавить HTTP-зависимость (пользовательский чекер) |
| `grpc` | `Builder grpc(String name, String host, int port, boolean critical)` | Добавить gRPC-зависимость (чекер по умолчанию) |
| `grpc` | `Builder grpc(String name, String host, int port, boolean critical, HealthChecker checker)` | Добавить gRPC-зависимость (пользовательский чекер) |

### DepHealth.DependencyBuilder

Билдер конфигурации для конкретной зависимости. Передаётся через
колбэк `Consumer` в `Builder.dependency()`.

#### Подключение

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `url` | `DependencyBuilder url(String url)` | Распарсить host/port из URL |
| `jdbcUrl` | `DependencyBuilder jdbcUrl(String jdbcUrl)` | Распарсить host/port из JDBC URL |
| `host` | `DependencyBuilder host(String host)` | Указать host явно |
| `port` | `DependencyBuilder port(String port)` | Указать port как строку |
| `port` | `DependencyBuilder port(int port)` | Указать port как число |

#### Общие

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `critical` | `DependencyBuilder critical(boolean critical)` | Пометить как критическую зависимость |
| `label` | `DependencyBuilder label(String key, String value)` | Добавить пользовательскую метку Prometheus |
| `interval` | `DependencyBuilder interval(Duration interval)` | Интервал проверок для зависимости |
| `timeout` | `DependencyBuilder timeout(Duration timeout)` | Тайм-аут проверок для зависимости |

#### HTTP-опции

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `httpHealthPath` | `DependencyBuilder httpHealthPath(String path)` | Путь проверки (по умолчанию `/health`) |
| `httpTls` | `DependencyBuilder httpTls(boolean enabled)` | Включить HTTPS (авто для `https://`) |
| `httpTlsSkipVerify` | `DependencyBuilder httpTlsSkipVerify(boolean skip)` | Пропустить проверку TLS-сертификата |
| `httpHeaders` | `DependencyBuilder httpHeaders(Map<String, String> headers)` | Пользовательские HTTP-заголовки |
| `httpBearerToken` | `DependencyBuilder httpBearerToken(String token)` | Bearer-токен |
| `httpBasicAuth` | `DependencyBuilder httpBasicAuth(String username, String password)` | Basic-аутентификация |

#### gRPC-опции

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `grpcServiceName` | `DependencyBuilder grpcServiceName(String name)` | Имя сервиса (пустая строка = здоровье сервера) |
| `grpcTls` | `DependencyBuilder grpcTls(boolean enabled)` | Включить TLS |
| `grpcTlsSkipVerify` | `DependencyBuilder grpcTlsSkipVerify(boolean skip)` | Пропустить проверку TLS-сертификата |
| `grpcMetadata` | `DependencyBuilder grpcMetadata(Map<String, String> metadata)` | Пользовательские метаданные gRPC |
| `grpcBearerToken` | `DependencyBuilder grpcBearerToken(String token)` | Bearer-токен |
| `grpcBasicAuth` | `DependencyBuilder grpcBasicAuth(String username, String password)` | Basic-аутентификация |

#### Опции баз данных

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `dbUsername` | `DependencyBuilder dbUsername(String username)` | Имя пользователя БД |
| `dbPassword` | `DependencyBuilder dbPassword(String password)` | Пароль БД |
| `dbDatabase` | `DependencyBuilder dbDatabase(String database)` | Имя базы данных |
| `dbQuery` | `DependencyBuilder dbQuery(String query)` | SQL-запрос проверки (по умолчанию `SELECT 1`) |
| `dataSource` | `DependencyBuilder dataSource(DataSource ds)` | Использовать существующий пул соединений |

#### Redis-опции

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `redisPassword` | `DependencyBuilder redisPassword(String password)` | Пароль (автономный режим) |
| `redisDb` | `DependencyBuilder redisDb(int db)` | Номер базы данных (автономный режим) |
| `jedisPool` | `DependencyBuilder jedisPool(JedisPool pool)` | Использовать существующий пул Jedis |

#### AMQP-опции

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `amqpUrl` | `DependencyBuilder amqpUrl(String url)` | Полный AMQP URL |
| `amqpUsername` | `DependencyBuilder amqpUsername(String username)` | Имя пользователя AMQP |
| `amqpPassword` | `DependencyBuilder amqpPassword(String password)` | Пароль AMQP |
| `amqpVirtualHost` | `DependencyBuilder amqpVirtualHost(String vhost)` | Виртуальный хост AMQP |

#### LDAP-опции

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `ldapCheckMethod` | `DependencyBuilder ldapCheckMethod(String method)` | Метод проверки: `anonymous_bind`, `simple_bind`, `root_dse`, `search` |
| `ldapBindDN` | `DependencyBuilder ldapBindDN(String dn)` | DN для простой привязки |
| `ldapBindPassword` | `DependencyBuilder ldapBindPassword(String password)` | Пароль для простой привязки |
| `ldapBaseDN` | `DependencyBuilder ldapBaseDN(String baseDN)` | Базовый DN для метода поиска |
| `ldapSearchFilter` | `DependencyBuilder ldapSearchFilter(String filter)` | LDAP-фильтр поиска (по умолчанию `(objectClass=*)`) |
| `ldapSearchScope` | `DependencyBuilder ldapSearchScope(String scope)` | Область поиска: `base`, `one`, `sub` |
| `ldapStartTLS` | `DependencyBuilder ldapStartTLS(boolean enabled)` | Использовать StartTLS (только с `ldap://`) |
| `ldapTlsSkipVerify` | `DependencyBuilder ldapTlsSkipVerify(boolean skip)` | Пропустить проверку TLS-сертификата |
| `ldapConnection` | `DependencyBuilder ldapConnection(LDAPConnection conn)` | Использовать существующее LDAP-соединение (режим пула) |

---

## Классы моделей

**Пакет:** `biz.kryukov.dev.dephealth.model`

### DependencyType

```java
public enum DependencyType {
    HTTP, GRPC, TCP, POSTGRES, MYSQL, REDIS, AMQP, KAFKA, LDAP
}
```

| Значение | `label()` |
| --- | --- |
| `HTTP` | `"http"` |
| `GRPC` | `"grpc"` |
| `TCP` | `"tcp"` |
| `POSTGRES` | `"postgres"` |
| `MYSQL` | `"mysql"` |
| `REDIS` | `"redis"` |
| `AMQP` | `"amqp"` |
| `KAFKA` | `"kafka"` |
| `LDAP` | `"ldap"` |

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `label` | `String label()` | Строковое представление в нижнем регистре |
| `fromLabel` | `static DependencyType fromLabel(String label)` | Парсинг из строки в нижнем регистре |

### Dependency

```java
public final class Dependency { /* ... */ }
```

Неизменяемое представление наблюдаемой зависимости. Содержит имя, тип,
флаг критичности, список эндпоинтов и конфигурацию проверки.

### Endpoint

```java
public final class Endpoint {
    public Endpoint(String host, String port)
    public Endpoint(String host, String port, Map<String, String> labels)
}
```

Сетевой эндпоинт зависимости.

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `host` | `String host()` | Имя хоста или IP-адрес |
| `port` | `String port()` | Порт как строка |
| `portAsInt` | `int portAsInt()` | Порт как число |
| `labels` | `Map<String, String> labels()` | Пользовательские метки Prometheus |

### EndpointStatus

```java
public final class EndpointStatus { /* ... */ }
```

Детальное состояние проверки для одного эндпоинта.

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `isHealthy` | `Boolean isHealthy()` | `null` до первой проверки, `true`/`false` после |
| `getStatus` | `String getStatus()` | Строка категории статуса |
| `getDetail` | `String getDetail()` | Строка детализации (например `http_503`, `grpc_not_serving`) |
| `getLatency` | `Duration getLatency()` | Задержка проверки |
| `getLatencyMillis` | `double getLatencyMillis()` | Задержка в миллисекундах |
| `getType` | `DependencyType getType()` | Тип зависимости |
| `getName` | `String getName()` | Имя зависимости |
| `getHost` | `String getHost()` | Хост эндпоинта |
| `getPort` | `String getPort()` | Порт эндпоинта |
| `isCritical` | `boolean isCritical()` | Является ли зависимость критической |
| `getLastCheckedAt` | `Instant getLastCheckedAt()` | Время последней проверки (`null` до первой проверки) |
| `getLabels` | `Map<String, String> getLabels()` | Пользовательские метки |

### CheckConfig

```java
public final class CheckConfig { /* ... */ }
```

Конфигурация планирования проверок.

| Поле | Тип | По умолчанию |
| --- | --- | --- |
| `interval` | `Duration` | 15s |
| `timeout` | `Duration` | 5s |
| `initialDelay` | `Duration` | 5s |
| `failureThreshold` | `int` | 1 |
| `successThreshold` | `int` | 1 |

### CheckResult

```java
public class CheckResult {
    public String category()
    public String detail()
}
```

Классификация результата проверки.

| Поле | Описание |
| --- | --- |
| `OK` | Статическая константа для успешного результата |

---

## StatusCategory

**Пакет:** `biz.kryukov.dev.dephealth.model`

Класс констант, определяющий категории статусов.

| Константа | Значение | Описание |
| --- | --- | --- |
| `OK` | `"ok"` | Зависимость доступна |
| `TIMEOUT` | `"timeout"` | Тайм-аут проверки |
| `CONNECTION_ERROR` | `"connection_error"` | Соединение отклонено или сброшено |
| `DNS_ERROR` | `"dns_error"` | Ошибка DNS-разрешения |
| `AUTH_ERROR` | `"auth_error"` | Ошибка аутентификации/авторизации |
| `TLS_ERROR` | `"tls_error"` | Ошибка TLS-рукопожатия |
| `UNHEALTHY` | `"unhealthy"` | Доступна, но нездорова |
| `ERROR` | `"error"` | Прочие ошибки |
| `UNKNOWN` | `"unknown"` | Ещё не проверялась |

---

## HealthChecker

**Пакет:** `biz.kryukov.dev.dephealth.checks`

```java
public interface HealthChecker {
    void check(Endpoint endpoint, Duration timeout) throws Exception;
    DependencyType type();
}
```

Интерфейс для проверки состояния зависимости. `check()` завершается
нормально, если зависимость здорова, или выбрасывает исключение
с описанием проблемы. `type()` возвращает тип зависимости.

### Реализации чекеров

| Класс | Тип | Метод-сокращение в билдере |
| --- | --- | --- |
| `HttpHealthChecker` | `HTTP` | `http(...)` |
| `GrpcHealthChecker` | `GRPC` | `grpc(...)` |
| `TcpHealthChecker` | `TCP` | -- |
| `PostgresHealthChecker` | `POSTGRES` | `postgres(...)` |
| `MysqlHealthChecker` | `MYSQL` | -- |
| `RedisHealthChecker` | `REDIS` | `redis(...)` |
| `AmqpHealthChecker` | `AMQP` | -- |
| `KafkaHealthChecker` | `KAFKA` | -- |
| `LdapHealthChecker` | `LDAP` | -- |

---

## Иерархия исключений

**Пакет:** `biz.kryukov.dev.dephealth`

```text
Exception
  └── CheckException                  -- базовое для всех исключений проверки
        ├── CheckAuthException        -- ошибка аутентификации/авторизации
        ├── CheckConnectionException  -- соединение отклонено/сброшено/тайм-аут
        ├── UnhealthyException        -- доступна, но нездорова
        ├── ValidationException       -- невалидная конфигурация
        ├── ConfigurationException    -- отсутствующие или некорректные настройки
        ├── EndpointNotFoundException -- эндпоинт не найден для обновления/удаления
        └── DepHealthException        -- общая ошибка SDK
```

### CheckException

Базовое исключение для ошибок проверки состояния.

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `statusCategory` | `String statusCategory()` | Категория статуса (например `"auth_error"`) |
| `statusDetail` | `String statusDetail()` | Строка детализации (например `"http_503"`) |

---

## ConfigParser

**Пакет:** `biz.kryukov.dev.dephealth.parser`

Статический утилитный класс для парсинга URL подключений и параметров.

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `parseUrl` | `static List<ParsedConnection> parseUrl(String url)` | Распарсить URL в список host/port/type |
| `parseJdbc` | `static List<ParsedConnection> parseJdbc(String jdbcUrl)` | Распарсить JDBC URL |
| `parseParams` | `static ParsedConnection parseParams(String host, String port)` | Создать из явных host и port |

Поддерживаемые схемы URL: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`,
`ldap`, `ldaps`. Kafka multi-broker URL
(`kafka://host1:9092,host2:9092`) возвращает несколько соединений.

### ParsedConnection

```java
public final class ParsedConnection {
    public String host()
    public String port()
    public DependencyType type()
}
```

Результат парсинга URL или строки подключения.

---

## MetricsExporter

**Пакет:** `biz.kryukov.dev.dephealth.metrics`

Создаёт метрики Prometheus (gauge `app_dependency_health` и histogram
`app_dependency_latency_seconds`) с использованием предоставленного
`MeterRegistry`. Класс является внутренним, но требует `MeterRegistry`,
переданный через `DepHealth.builder()`.

---

## Интеграция со Spring Boot

**Пакет:** `biz.kryukov.dev.dephealth.spring`

### DepHealthAutoConfiguration

Класс автоконфигурации. Активируется при наличии
`dephealth-spring-boot-starter` в classpath. Читает
`DepHealthProperties` и создаёт бин `DepHealth`.

### DepHealthProperties

```java
@ConfigurationProperties(prefix = "dephealth")
public class DepHealthProperties { /* ... */ }
```

Маппинг конфигурации `application.yml` / `application.properties`
с префиксом `dephealth.*`.

### DepHealthLifecycle

Реализует `SmartLifecycle`. Вызывает `DepHealth.start()` при запуске
приложения и `DepHealth.stop()` при остановке.

### DepHealthIndicator

Реализует Spring Boot `HealthIndicator`. Предоставляет результаты
проверок через `/actuator/health`.

### DependenciesEndpoint

```java
@Endpoint(id = "dependencies")
public class DependenciesEndpoint { /* ... */ }
```

Пользовательский Actuator-эндпоинт по адресу `/actuator/dependencies`.
Возвращает детальный статус всех наблюдаемых зависимостей.

---

## Динамическое управление эндпоинтами

Методы для добавления, удаления и обновления эндпоинтов в рантайме
на работающем экземпляре `DepHealth`. Все методы потокобезопасны.

### addEndpoint

```java
public void addEndpoint(String depName, DependencyType depType,
    boolean critical, Endpoint ep, HealthChecker checker)
```

Добавляет новый эндпоинт к работающему экземпляру `DepHealth`. Задача
проверки запускается немедленно с глобальным интервалом и тайм-аутом.

**Идемпотентность:** если эндпоинт с таким же ключом `depName:host:port`
уже существует, вызов не производит изменений.

### removeEndpoint

```java
public void removeEndpoint(String depName, String host, String port)
```

Удаляет эндпоинт из работающего экземпляра `DepHealth`. Отменяет задачу
проверки и удаляет все связанные метрики Prometheus.

**Идемпотентность:** если эндпоинт с указанным ключом не существует,
вызов не производит изменений.

### updateEndpoint

```java
public void updateEndpoint(String depName, String oldHost, String oldPort,
    Endpoint newEp, HealthChecker checker)
```

Атомарно заменяет существующий эндпоинт новым. Задача старого эндпоинта
отменяется, его метрики удаляются; для нового эндпоинта запускается
новая задача.

**Ошибки:**

| Условие | Исключение |
| --- | --- |
| Старый эндпоинт не найден | `EndpointNotFoundException` |
| Отсутствует host или port нового эндпоинта | `ValidationException` |

---

## Смотрите также

- [Начало работы](getting-started.ru.md) -- установка и первый пример
