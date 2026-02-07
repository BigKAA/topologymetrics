# Быстрый старт: Java SDK

Руководство по подключению dephealth к Java-сервису за несколько минут.

## Установка

### Maven

Core-модуль:

```xml
<dependency>
    <groupId>com.github.bigkaa</groupId>
    <artifactId>dephealth-core</artifactId>
    <version>0.1.0-SNAPSHOT</version>
</dependency>
```

Spring Boot Starter (включает core):

```xml
<dependency>
    <groupId>com.github.bigkaa</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.1.0-SNAPSHOT</version>
</dependency>
```

## Минимальный пример

Подключение одной HTTP-зависимости с экспортом метрик:

```java
import com.github.bigkaa.dephealth.*;
import io.micrometer.prometheus.PrometheusMeterRegistry;

public class Main {
    public static void main(String[] args) {
        PrometheusMeterRegistry registry = new PrometheusMeterRegistry(
            PrometheusConfig.DEFAULT);

        DepHealth depHealth = DepHealth.builder(registry)
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
app_dependency_health{dependency="payment-api",type="http",host="payment.svc",port="8080"} 1
app_dependency_latency_seconds_bucket{dependency="payment-api",type="http",host="payment.svc",port="8080",le="0.01"} 42
```

## Несколько зависимостей

```java
DepHealth depHealth = DepHealth.builder(meterRegistry)
    // Глобальные настройки
    .checkInterval(Duration.ofSeconds(30))
    .timeout(Duration.ofSeconds(3))

    // PostgreSQL — standalone check (новое соединение)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))

    // Redis — standalone check
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .url("redis://:password@redis.svc:6379/0"))

    // HTTP-сервис
    .dependency("auth-service", DependencyType.HTTP, d -> d
        .url("http://auth.svc:8080")
        .httpHealthPath("/healthz")
        .critical(true))

    // gRPC-сервис
    .dependency("user-service", DependencyType.GRPC, d -> d
        .host("user.svc")
        .port("9090"))

    // RabbitMQ
    .dependency("rabbitmq", DependencyType.AMQP, d -> d
        .host("rabbitmq.svc")
        .port("5672")
        .amqpUsername("user")
        .amqpPassword("pass")
        .amqpVirtualHost("/"))

    // Kafka
    .dependency("kafka", DependencyType.KAFKA, d -> d
        .host("kafka.svc")
        .port("9092"))

    .build();
```

## Интеграция с connection pool

Предпочтительный режим: SDK использует существующий connection pool
сервиса вместо создания новых соединений.

### PostgreSQL через DataSource

```java
import javax.sql.DataSource;
// HikariCP, Tomcat JDBC или любой другой пул

DataSource dataSource = ...; // существующий pool из приложения

DepHealth depHealth = DepHealth.builder(meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .dataSource(dataSource)
        .critical(true))
    .build();
```

### Redis через JedisPool

```java
import redis.clients.jedis.JedisPool;

JedisPool jedisPool = ...; // существующий pool из приложения

DepHealth depHealth = DepHealth.builder(meterRegistry)
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .jedisPool(jedisPool))
    .build();
```

## Spring Boot интеграция

Добавьте зависимость `dephealth-spring-boot-starter` и настройте
`application.yml`:

```yaml
dephealth:
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

    auth-service:
      type: http
      url: http://auth.svc:8080
      health-path: /healthz
      critical: true

    user-service:
      type: grpc
      host: user.svc
      port: "9090"

    rabbitmq:
      type: amqp
      url: amqp://user:pass@rabbitmq.svc:5672/
      amqp-username: user
      amqp-password: pass

    kafka:
      type: kafka
      host: kafka.svc
      port: "9092"
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
DepHealth depHealth = DepHealth.builder(meterRegistry)
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

## Проверка состояния зависимостей

Метод `health()` возвращает текущее состояние всех endpoint-ов:

```java
Map<String, Boolean> health = depHealth.health();
// {"postgres-main:pg.svc:5432": true, "redis-cache:redis.svc:6379": true}

// Использование для readiness probe
boolean allHealthy = health.values().stream().allMatch(Boolean::booleanValue);
```

## Экспорт метрик

dephealth экспортирует две метрики Prometheus через Micrometer:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = доступен, `0` = недоступен |
| `app_dependency_latency_seconds` | Histogram | Латентность проверки (секунды) |

Метки: `dependency`, `type`, `host`, `port`.

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

- [Руководство по интеграции](../migration/java.md) — пошаговое подключение
  к существующему сервису
- [Обзор спецификации](../specification.md) — детали контрактов метрик и поведения
