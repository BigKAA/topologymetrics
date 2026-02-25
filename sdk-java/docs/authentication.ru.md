*[English version](authentication.md)*

# Аутентификация

Руководство по настройке аутентификации для всех чекеров, поддерживающих
учётные данные — HTTP, gRPC, чекеров баз данных/кэшей и LDAP.

## Обзор

| Чекер | Методы аутентификации | Как передаются учётные данные |
| --- | --- | --- |
| HTTP | Bearer token, Basic auth, произвольные заголовки | `.httpBearerToken()`, `.httpBasicAuth()`, `.httpHeaders()` |
| gRPC | Bearer token, Basic auth, произвольные метаданные | `.grpcBearerToken()`, `.grpcBasicAuth()`, `.grpcMetadata()` |
| PostgreSQL | Логин/пароль | Учётные данные в URL или `.dbUsername()` + `.dbPassword()` |
| MySQL | Логин/пароль | Учётные данные в URL или `.dbUsername()` + `.dbPassword()` |
| Redis | Пароль | `.redisPassword()` или пароль в URL |
| AMQP | Логин/пароль | `.amqpUsername()` + `.amqpPassword()` или `.amqpUrl()` |
| LDAP | Bind DN/пароль | `.ldapBindDN()` + `.ldapBindPassword()` |
| TCP | — | Без аутентификации |
| Kafka | — | Без аутентификации |

## HTTP аутентификация

Для каждой зависимости допускается только один метод аутентификации.
Указание одновременно `.httpBearerToken()` и `.httpBasicAuth()` вызывает
ошибку валидации.

### Bearer Token

```java
.dependency("secure-api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true)
    .httpBearerToken(System.getenv("API_TOKEN")))
```

Отправляет заголовок `Authorization: Bearer <token>` с каждым запросом проверки.

### Basic Auth

```java
.dependency("basic-api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true)
    .httpBasicAuth("admin", System.getenv("API_PASSWORD")))
```

Отправляет заголовок `Authorization: Basic <base64(user:pass)>`.

### Произвольные заголовки

Для нестандартных схем аутентификации (API-ключи, пользовательские токены):

```java
.dependency("custom-auth-api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true)
    .httpHeaders(Map.of(
        "X-API-Key", System.getenv("API_KEY"),
        "X-API-Secret", System.getenv("API_SECRET"))))
```

## gRPC аутентификация

Для каждой зависимости допускается только один метод, как и для HTTP.

### Bearer Token

```java
.dependency("grpc-backend", DependencyType.GRPC, d -> d
    .host("backend.svc")
    .port("9090")
    .critical(true)
    .grpcBearerToken(System.getenv("GRPC_TOKEN")))
```

Отправляет `authorization: Bearer <token>` как gRPC-метаданные.

### Basic Auth

```java
.dependency("grpc-backend", DependencyType.GRPC, d -> d
    .host("backend.svc")
    .port("9090")
    .critical(true)
    .grpcBasicAuth("admin", System.getenv("GRPC_PASSWORD")))
```

Отправляет `authorization: Basic <base64(user:pass)>` как gRPC-метаданные.

### Произвольные метаданные

```java
.dependency("grpc-backend", DependencyType.GRPC, d -> d
    .host("backend.svc")
    .port("9090")
    .critical(true)
    .grpcMetadata(Map.of("x-api-key", System.getenv("GRPC_API_KEY"))))
```

## Учётные данные баз данных

### PostgreSQL

Учётные данные можно включить в URL или задать явно:

```java
// Через URL
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://myuser:mypass@pg.svc:5432/mydb")
    .critical(true))

// Через явные параметры
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://pg.svc:5432/mydb")
    .dbUsername("myuser")
    .dbPassword("mypass")
    .critical(true))
```

Явные параметры перекрывают учётные данные, извлечённые из URL.

### MySQL

Аналогичный подход — учётные данные в URL или явно:

```java
.dependency("mysql-main", DependencyType.MYSQL, d -> d
    .url("mysql://myuser:mypass@mysql.svc:3306/mydb")
    .critical(true))
```

### Redis

Пароль можно задать через опцию или включить в URL:

```java
// Через опцию
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .host("redis.svc")
    .port("6379")
    .redisPassword(System.getenv("REDIS_PASSWORD"))
    .critical(false))

// Через URL
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://:mypassword@redis.svc:6379/0")
    .critical(false))
```

### AMQP (RabbitMQ)

Учётные данные можно задать через опции или в AMQP URL:

```java
// Через опции
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .host("rabbitmq.svc")
    .port("5672")
    .amqpUsername("user")
    .amqpPassword("pass")
    .amqpVirtualHost("/")
    .critical(false))

// Через URL
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .amqpUrl("amqp://user:pass@rabbitmq.svc:5672/")
    .critical(false))
```

### LDAP

Для метода проверки `simple_bind` требуются учётные данные для привязки:

```java
.dependency("directory", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod("simple_bind")
    .ldapBindDN("cn=monitor,dc=corp,dc=com")
    .ldapBindPassword(System.getenv("LDAP_PASSWORD"))
    .critical(true))
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
| MySQL | Ошибка 1045 (Access Denied) | `auth_error` | `auth_error` |
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
3. **Предпочитайте TLS** — включайте TLS для HTTP и gRPC чекеров при
   использовании аутентификации для защиты учётных данных при передаче
4. **Ограничивайте права** — используйте учётные данные только для чтения;
   `SELECT 1` не требует прав на запись

## Полный пример

```java
import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.model.DependencyType;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;

public class Main {
    public static void main(String[] args) {
        var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

        var dh = DepHealth.builder("my-service", "my-team", registry)
            // HTTP с Bearer-токеном
            .dependency("payment-api", DependencyType.HTTP, d -> d
                .url("https://payment.svc:443")
                .httpTls(true)
                .httpBearerToken(System.getenv("PAYMENT_TOKEN"))
                .critical(true))

            // gRPC с произвольными метаданными
            .dependency("user-service", DependencyType.GRPC, d -> d
                .host("user.svc")
                .port("9090")
                .grpcMetadata(Map.of("x-api-key", System.getenv("USER_API_KEY")))
                .critical(true))

            // PostgreSQL с учётными данными в URL
            .dependency("postgres-main", DependencyType.POSTGRES, d -> d
                .url(System.getenv("DATABASE_URL"))
                .critical(true))

            // Redis с паролем
            .dependency("redis-cache", DependencyType.REDIS, d -> d
                .host("redis.svc")
                .port("6379")
                .redisPassword(System.getenv("REDIS_PASSWORD"))
                .critical(false))

            // AMQP с учётными данными
            .dependency("rabbitmq", DependencyType.AMQP, d -> d
                .amqpUrl(System.getenv("AMQP_URL"))
                .critical(false))

            // LDAP с учётными данными привязки
            .dependency("directory", DependencyType.LDAP, d -> d
                .url("ldaps://ad.corp:636")
                .ldapCheckMethod("simple_bind")
                .ldapBindDN("cn=monitor,dc=corp,dc=com")
                .ldapBindPassword(System.getenv("LDAP_PASSWORD"))
                .critical(true))

            .build();

        dh.start();
        Runtime.getRuntime().addShutdownHook(new Thread(dh::stop));
    }
}
```

## См. также

- [Чекеры](checkers.ru.md) — подробное руководство по всем 9 чекерам
- [Конфигурация](configuration.ru.md) — переменные окружения и опции
- [Метрики](metrics.ru.md) — метрики Prometheus включая категории статуса
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
