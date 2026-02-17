*[English version](java.md)*

# Быстрый старт: Java SDK

Руководство по подключению dephealth к Java-сервису за несколько минут.

## Установка

### Maven

Core-модуль:

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-core</artifactId>
    <version>0.4.2</version>
</dependency>
```

Spring Boot Starter (включает core):

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.4.2</version>
</dependency>
```

## Минимальный пример

Подключение одной HTTP-зависимости с экспортом метрик:

```java
import biz.kryukov.dev.dephealth.*;
import io.micrometer.prometheus.PrometheusMeterRegistry;

public class Main {
    public static void main(String[] args) {
        PrometheusMeterRegistry registry = new PrometheusMeterRegistry(
            PrometheusConfig.DEFAULT);

        DepHealth depHealth = DepHealth.builder("my-service", registry)
            .dependency("payment-api", DependencyType.HTTP, d -> d
                .url("http://payment.svc:8080")
                .critical(true))
            .build();

        depHealth.start();

        // ... HTTP-сервер экспортирует registry.scrape() на /metrics ...

        // Graceful shutdown
        Runtime.getRuntime().addShutdownHook(new Thread(depHealth::stop));
    }
}
```

После запуска на `/metrics` появятся метрики:

```text
app_dependency_health{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
app_dependency_status{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",status="healthy"} 1
app_dependency_status_detail{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",detail=""} 1
```

## Несколько зависимостей

```java
DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    // Глобальные настройки
    .checkInterval(Duration.ofSeconds(30))
    .timeout(Duration.ofSeconds(3))

    // PostgreSQL — standalone check (новое соединение)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))

    // Redis — standalone check
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .url("redis://:password@redis.svc:6379/0")
        .critical(false))

    // HTTP-сервис
    .dependency("auth-service", DependencyType.HTTP, d -> d
        .url("http://auth.svc:8080")
        .httpHealthPath("/healthz")
        .critical(true))

    // gRPC-сервис
    .dependency("user-service", DependencyType.GRPC, d -> d
        .host("user.svc")
        .port("9090")
        .critical(false))

    // RabbitMQ
    .dependency("rabbitmq", DependencyType.AMQP, d -> d
        .host("rabbitmq.svc")
        .port("5672")
        .amqpUsername("user")
        .amqpPassword("pass")
        .amqpVirtualHost("/")
        .critical(false))

    // Kafka
    .dependency("kafka", DependencyType.KAFKA, d -> d
        .host("kafka.svc")
        .port("9092")
        .critical(false))

    .build();
```

## Произвольные метки

Добавляйте произвольные метки через `.label()`:

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgres://user:pass@pg.svc:5432/mydb")
    .critical(true)
    .label("role", "primary")
    .label("shard", "eu-west"))
```

Результат в метриках:

```text
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",role="primary",shard="eu-west"} 1
```

## Интеграция с connection pool

Предпочтительный режим: SDK использует существующий connection pool
сервиса вместо создания новых соединений.

### PostgreSQL через DataSource

```java
import javax.sql.DataSource;
// HikariCP, Tomcat JDBC или любой другой пул

DataSource dataSource = ...; // существующий pool из приложения

DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .dataSource(dataSource)
        .critical(true))
    .build();
```

### Redis через JedisPool

```java
import redis.clients.jedis.JedisPool;

JedisPool jedisPool = ...; // существующий pool из приложения

DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .jedisPool(jedisPool)
        .critical(false))
    .build();
```

## Spring Boot интеграция

Добавьте зависимость `dephealth-spring-boot-starter` и настройте
`application.yml`:

```yaml
dephealth:
  name: my-service
  interval: 15s
  timeout: 5s

  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      critical: true

    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false

    auth-service:
      type: http
      url: http://auth.svc:8080
      health-path: /healthz
      critical: true
      labels:
        role: primary

    user-service:
      type: grpc
      host: user.svc
      port: "9090"
      critical: false

    rabbitmq:
      type: amqp
      url: amqp://user:pass@rabbitmq.svc:5672/
      amqp-username: user
      amqp-password: pass
      critical: false

    kafka:
      type: kafka
      host: kafka.svc
      port: "9092"
      critical: false
```

Spring Boot автоматически:

- Создаёт и стартует `DepHealth` bean
- Регистрирует Health Indicator (`/actuator/health`)
- Добавляет endpoint `/actuator/dependencies`
- Экспортирует Prometheus-метрики на `/actuator/prometheus`

### Actuator endpoints

```bash
# Состояние зависимостей
curl http://localhost:8080/actuator/dependencies

# Ответ:
{
    "postgres-main:pg.svc:5432": true,
    "redis-cache:redis.svc:6379": true,
    "auth-service:auth.svc:8080": false
}
```

```bash
# Health indicator (интегрирован в основной /actuator/health)
curl http://localhost:8080/actuator/health

# Ответ:
{
    "status": "DOWN",
    "components": {
        "dephealth": {
            "status": "DOWN",
            "details": {
                "postgres-main:pg.svc:5432": "UP",
                "auth-service:auth.svc:8080": "DOWN"
            }
        }
    }
}
```

## Глобальные опции

```java
DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    // Интервал проверки (по умолчанию 15s)
    .checkInterval(Duration.ofSeconds(30))

    // Таймаут каждой проверки (по умолчанию 5s)
    .timeout(Duration.ofSeconds(3))

    // ...зависимости
    .build();
```

## Опции зависимостей

Каждая зависимость может переопределить глобальные настройки:

```java
.dependency("slow-service", DependencyType.HTTP, d -> d
    .url("http://slow.svc:8080")
    .httpHealthPath("/ready")           // путь health check
    .httpTls(true)                      // HTTPS
    .httpTlsSkipVerify(true)            // пропустить проверку сертификата
    .interval(Duration.ofSeconds(60))   // свой интервал
    .timeout(Duration.ofSeconds(10))    // свой таймаут
    .critical(true))                    // критическая зависимость
```

## Аутентификация

HTTP и gRPC чекеры поддерживают аутентификацию. Для каждой зависимости
допускается только один метод — смешивание вызывает ошибку валидации.

### HTTP Bearer Token

```java
.dependency("secure-api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true)
    .httpBearerToken("eyJhbG..."))
```

### HTTP Basic Auth

```java
.dependency("secure-api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true)
    .httpBasicAuth("admin", "secret"))
```

### HTTP произвольные заголовки

```java
.dependency("secure-api", DependencyType.HTTP, d -> d
    .url("http://api.svc:8080")
    .critical(true)
    .httpHeaders(Map.of("X-API-Key", "my-key")))
```

### gRPC Bearer Token

```java
.dependency("grpc-backend", DependencyType.GRPC, d -> d
    .host("backend.svc")
    .port(9090)
    .critical(true)
    .grpcBearerToken("eyJhbG..."))
```

### gRPC произвольные метаданные

```java
.dependency("grpc-backend", DependencyType.GRPC, d -> d
    .host("backend.svc")
    .port(9090)
    .critical(true)
    .grpcMetadata(Map.of("x-api-key", "my-key")))
```

### Spring Boot YAML

```yaml
dephealth:
  dependencies:
    secure-api:
      type: http
      url: http://api.svc:8080
      critical: true
      http-bearer-token: ${API_TOKEN}
      # ИЛИ
      # http-basic-username: ${API_USER}
      # http-basic-password: ${API_PASS}
      # ИЛИ
      # http-headers:
      #   X-API-Key: ${API_KEY}

    grpc-backend:
      type: grpc
      host: backend.svc
      port: "9090"
      critical: true
      grpc-bearer-token: ${GRPC_TOKEN}
      # ИЛИ
      # grpc-metadata:
      #   x-api-key: ${GRPC_KEY}
```

### Классификация ошибок аутентификации

Когда сервер возвращает ошибку аутентификации, чекер классифицирует
её как `auth_error`:

- HTTP 401/403 → `status="auth_error"`, `detail="auth_error"`
- gRPC UNAUTHENTICATED/PERMISSION_DENIED → `status="auth_error"`, `detail="auth_error"`

## Конфигурация через переменные окружения

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Имя приложения (перекрывается API) | `my-service` |
| `DEPHEALTH_<DEP>_CRITICAL` | Критичность зависимости | `yes` / `no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Произвольная метка | `primary` |

`<DEP>` — имя зависимости в верхнем регистре, дефисы заменены на `_`.

Примеры:

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
```

Приоритет: значения из API/application.yml > переменные окружения.

## Поведение при отсутствии обязательных параметров

| Ситуация | Поведение |
| --- | --- |
| Не указан `name` и нет `DEPHEALTH_NAME` | Ошибка при создании: `missing name` |
| Не указан `.critical()` для зависимости | Ошибка при создании: `missing critical` |
| Недопустимое имя метки | Ошибка при создании: `invalid label name` |
| Метка совпадает с обязательной | Ошибка при создании: `reserved label` |

## Проверка состояния зависимостей

Метод `health()` возвращает текущее состояние всех endpoint-ов:

```java
Map<String, Boolean> health = depHealth.health();
// {"postgres-main:pg.svc:5432": true, "redis-cache:redis.svc:6379": true}

// Использование для readiness probe
boolean allHealthy = health.values().stream().allMatch(Boolean::booleanValue);
```

## Детальный статус зависимостей

Метод `healthDetails()` возвращает подробную информацию о каждом endpoint-е,
включая категорию статуса, причину сбоя, латентность и пользовательские метки:

```java
Map<String, EndpointStatus> details = depHealth.healthDetails();
// {"postgres-main:pg.svc:5432": EndpointStatus{
//     dependency="postgres-main", type="postgres",
//     host="pg.svc", port="5432",
//     healthy=true, status="ok", detail="ok",
//     latency=Duration.ofMillis(15),
//     lastCheckedAt=Instant.now(),
//     critical=true, labels={"role": "primary"}
// }}
```

В отличие от `health()`, который возвращает `Map<String, Boolean>`, `healthDetails()`
предоставляет полный объект `EndpointStatus` для каждого endpoint-а. До завершения
первой проверки `healthy` равен `null` (неизвестно), а `status` — `"unknown"`.

## Экспорт метрик

dephealth экспортирует четыре метрики Prometheus через Micrometer:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = доступен, `0` = недоступен |
| `app_dependency_latency_seconds` | Histogram | Латентность проверки (секунды) |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина: напр. `http_503`, `auth_error` |

Метки: `name`, `dependency`, `type`, `host`, `port`, `critical`.
Дополнительные: `status` (на `app_dependency_status`), `detail` (на `app_dependency_status_detail`).

Для Spring Boot: метрики доступны на `/actuator/prometheus`.

## Поддерживаемые типы зависимостей

| DependencyType | Тип | Метод проверки |
| --- | --- | --- |
| `HTTP` | `http` | HTTP GET к health endpoint, ожидание 2xx |
| `GRPC` | `grpc` | gRPC Health Check Protocol |
| `TCP` | `tcp` | Установка TCP-соединения |
| `POSTGRES` | `postgres` | `SELECT 1` через JDBC |
| `MYSQL` | `mysql` | `SELECT 1` через JDBC |
| `REDIS` | `redis` | Команда `PING` через Jedis |
| `AMQP` | `amqp` | Проверка соединения с RabbitMQ |
| `KAFKA` | `kafka` | Metadata request к брокеру |

## Параметры по умолчанию

| Параметр | Значение | Описание |
| --- | --- | --- |
| `checkInterval` | 15s | Интервал между проверками |
| `timeout` | 5s | Таймаут одной проверки |
| `initialDelay` | 5s | Задержка перед первой проверкой |
| `failureThreshold` | 1 | Число неудач до перехода в unhealthy |
| `successThreshold` | 1 | Число успехов до перехода в healthy |

## Следующие шаги

- [Руководство по интеграции](../migration/java.ru.md) — пошаговое подключение
  к существующему сервису
- [Обзор спецификации](../specification.ru.md) — детали контрактов метрик и поведения
