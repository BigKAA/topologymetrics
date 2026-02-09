# Руководство по интеграции dephealth в существующий Java-сервис

Пошаговая инструкция по добавлению мониторинга зависимостей
в работающий микросервис.

## Миграция с v0.1 на v0.2

### Изменения API

| v0.1 | v0.2 | Описание |
| --- | --- | --- |
| `DepHealth.builder(registry)` | `DepHealth.builder("my-service", registry)` | Обязательный первый аргумент `name` |
| `.critical(true)` (необязателен) | `.critical(true/false)` (обязателен) | Для каждой зависимости |
| нет | `.label("key", "value")` | Произвольные метки |
| `dephealth.name` (нет) | `dephealth.name: my-service` | В application.yml |

### Обязательные изменения

1. Добавьте `name` в builder:

```java
// v0.1
DepHealth depHealth = DepHealth.builder(meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))
    .build();

// v0.2
DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))
    .build();
```

1. Укажите `.critical()` для каждой зависимости:

```java
// v0.1 — critical необязателен
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://redis.svc:6379"))

// v0.2 — critical обязателен
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://redis.svc:6379")
    .critical(false))
```

1. Обновите `application.yml` (Spring Boot):

```yaml
# v0.1
dephealth:
  dependencies:
    redis-cache:
      type: redis
      url: ${REDIS_URL}

# v0.2
dephealth:
  name: my-service
  dependencies:
    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false
```

1. Обновите версию зависимости:

```xml
<!-- v0.1 -->
<version>0.1.0</version>

<!-- v0.2 -->
<version>0.2.1</version>
```

### Новые метки в метриках

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Обновите PromQL-запросы и дашборды Grafana, добавив метки `name` и `critical`.

## Предварительные требования

- Java 21+
- Spring Boot 3.x (рекомендуется) или любой фреймворк с Micrometer
- Доступ к зависимостям (БД, кэш, другие сервисы) из сервиса

## Шаг 1. Установка зависимости

### Spring Boot (рекомендуется)

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.2.1</version>
</dependency>
```

### Без Spring Boot

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-core</artifactId>
    <version>0.2.1</version>
</dependency>
```

Также убедитесь, что драйверы зависимостей присутствуют в classpath
(postgresql, jedis, amqp-client, kafka-clients и т.д.).

## Шаг 2. Конфигурация (Spring Boot)

Добавьте настройки в `application.yml`:

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

    payment-api:
      type: http
      url: http://payment.svc:8080
      health-path: /health
      critical: true
      labels:
        role: primary

    user-service:
      type: grpc
      host: user.svc
      port: "9090"
      critical: false
```

Spring Boot автоматически создаст и запустит `DepHealth` bean.

## Шаг 3. Конфигурация (без Spring Boot)

### Вариант A: Standalone-режим (простой)

SDK создаёт временные соединения для проверок:

```java
import biz.kryukov.dev.dephealth.*;
import io.micrometer.prometheus.PrometheusMeterRegistry;

PrometheusMeterRegistry meterRegistry = ...;

DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true))
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .url(System.getenv("REDIS_URL"))
        .critical(false))
    .dependency("payment-api", DependencyType.HTTP, d -> d
        .url(System.getenv("PAYMENT_SERVICE_URL"))
        .critical(true))
    .build();
```

### Вариант B: Интеграция с connection pool (рекомендуется)

SDK использует существующие подключения сервиса. Преимущества:

- Отражает реальную способность сервиса работать с зависимостью
- Не создаёт дополнительную нагрузку на БД/кэш
- Обнаруживает проблемы с пулом (исчерпание, утечки)

```java
import javax.sql.DataSource;
import redis.clients.jedis.JedisPool;

DataSource dataSource = ...; // HikariCP, Tomcat JDBC и т.д.
JedisPool jedisPool = ...;

DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
    .checkInterval(Duration.ofSeconds(15))

    // PostgreSQL через существующий DataSource
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .dataSource(dataSource)
        .critical(true))

    // Redis через существующий JedisPool
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .jedisPool(jedisPool)
        .critical(false))

    // Для HTTP/gRPC — только standalone
    .dependency("payment-api", DependencyType.HTTP, d -> d
        .url(System.getenv("PAYMENT_SERVICE_URL"))
        .critical(true))

    .dependency("auth-service", DependencyType.GRPC, d -> d
        .host(System.getenv("AUTH_HOST"))
        .port(System.getenv("AUTH_PORT"))
        .critical(true))

    .build();
```

## Шаг 4. Запуск и остановка

### Spring Boot

Управление автоматическое через `DepHealthLifecycle` (SmartLifecycle).

### Без Spring Boot

Встройте `start()` и `stop()` в жизненный цикл сервиса:

```java
public class Main {
    public static void main(String[] args) {
        DepHealth depHealth = DepHealth.builder("my-service", meterRegistry)
            // ... зависимости ...
            .build();

        depHealth.start();

        // ... запуск HTTP-сервера ...

        // Graceful shutdown
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            depHealth.stop();
            // ... остановка HTTP-сервера ...
        }));
    }
}
```

## Шаг 5. Экспорт метрик

### Spring Boot

Метрики доступны автоматически на `/actuator/prometheus`.

Убедитесь, что в `application.yml`:

```yaml
management:
  endpoints:
    web:
      exposure:
        include: health, prometheus, dependencies
```

### Без Spring Boot

Экспортируйте через Micrometer:

```java
import io.micrometer.prometheus.PrometheusMeterRegistry;

// В HTTP-обработчике /metrics:
String metrics = meterRegistry.scrape();
response.setContentType("text/plain; version=0.0.4");
response.getWriter().write(metrics);
```

## Шаг 6. Endpoint для состояния зависимостей (опционально)

### Spring Boot

Уже есть два встроенных endpoint-а:

```bash
# Прямой статус зависимостей
GET /actuator/dependencies

# Ответ:
{
    "postgres-main:pg.svc:5432": true,
    "redis-cache:redis.svc:6379": true,
    "payment-api:payment.svc:8080": false
}

# Интегрирован в Spring Health Indicator
GET /actuator/health
```

### Без Spring Boot

```java
void handleDependencies(HttpServletRequest req, HttpServletResponse resp) {
    Map<String, Boolean> health = depHealth.health();

    boolean allHealthy = health.values().stream()
        .allMatch(Boolean::booleanValue);

    resp.setStatus(allHealthy ? 200 : 503);
    resp.setContentType("application/json");

    // Сериализуйте health в JSON
    new ObjectMapper().writeValue(resp.getWriter(), health);
}
```

## Типичные конфигурации

### Spring Boot + PostgreSQL + Redis

```yaml
dephealth:
  name: my-service
  dependencies:
    postgres:
      type: postgres
      url: ${DATABASE_URL}
      critical: true
    redis:
      type: redis
      url: ${REDIS_URL}
      critical: false
```

### API Gateway с upstream-сервисами

```yaml
dephealth:
  name: api-gateway
  interval: 10s
  dependencies:
    user-service:
      type: http
      url: http://user-svc:8080
      health-path: /healthz
      critical: true
    order-service:
      type: http
      url: http://order-svc:8080
      critical: true
    auth-service:
      type: grpc
      host: auth-svc
      port: "9090"
      critical: true
```

### Обработчик событий с Kafka и RabbitMQ

```yaml
dephealth:
  name: event-processor
  dependencies:
    kafka-main:
      type: kafka
      host: kafka.svc
      port: "9092"
      critical: true
    rabbitmq:
      type: amqp
      url: amqp://user:pass@rabbitmq.svc:5672/
      amqp-username: user
      amqp-password: pass
      critical: true
    postgres:
      type: postgres
      url: ${DATABASE_URL}
      critical: false
```

## Troubleshooting

### Метрики не появляются на `/actuator/prometheus`

**Проверьте:**

1. Зависимость `spring-boot-starter-actuator` присутствует
2. `management.endpoints.web.exposure.include` включает `prometheus`
3. `dephealth-spring-boot-starter` в classpath

### Все зависимости показывают `0` (unhealthy)

**Проверьте:**

1. Сетевая доступность зависимостей из контейнера/пода сервиса
2. DNS-резолвинг имён сервисов
3. Правильность URL/host/port в конфигурации
4. Таймаут (`5s` по умолчанию) — достаточен ли для данной зависимости
5. Логи: настройте `logging.level.biz.kryukov.dev.dephealth=DEBUG`

### Высокая латентность проверок PostgreSQL/MySQL

**Причина**: standalone-режим создаёт новое JDBC-соединение каждый раз.

**Решение**: используйте `DataSource` интеграцию.

### gRPC: ошибка `DEADLINE_EXCEEDED`

**Проверьте:**

1. gRPC-сервис доступен по указанному адресу
2. Сервис реализует `grpc.health.v1.Health/Check`
3. Для gRPC используйте `host` + `port`, а не `url`
4. Если нужен TLS: `tls: true` в конфигурации

### AMQP: ошибка подключения к RabbitMQ

**Важно**: путь `/` в URL означает vhost `/` (не пусто).

```yaml
rabbitmq:
  type: amqp
  host: rabbitmq.svc
  port: "5672"
  amqp-username: user
  amqp-password: pass
  virtual-host: /
  critical: false
```

### Парсинг URL и credentials

SDK автоматически извлекает username/password из URL:

```yaml
postgres:
  type: postgres
  url: postgres://user:pass@host:5432/db
  critical: true
  # username и password извлекаются автоматически
```

Можно переопределить явно:

```yaml
postgres:
  type: postgres
  url: postgres://old:old@host:5432/db
  username: new_user    # перекрывает парсинг из URL
  password: new_pass
  critical: true
```

### Именование зависимостей

Имена должны соответствовать правилам:

- Длина: 1-63 символа
- Формат: `[a-z][a-z0-9-]*` (строчные буквы, цифры, дефисы)
- Начинается с буквы
- Примеры: `postgres-main`, `redis-cache`, `auth-service`

## Следующие шаги

- [Быстрый старт](../quickstart/java.md) — минимальные примеры
- [Обзор спецификации](../specification.md) — детали контрактов метрик и поведения
