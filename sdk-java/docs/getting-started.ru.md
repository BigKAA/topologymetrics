*[English version](getting-started.md)*

# Начало работы

Руководство по установке, базовой настройке и первой проверке состояния
зависимости с помощью Java SDK dephealth.

## Требования

- Java 21 или новее
- Maven или Gradle
- Работающая зависимость для мониторинга (PostgreSQL, Redis, HTTP-сервис и т.д.)

## Установка

### Maven

Core-модуль (программный API):

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-core</artifactId>
    <version>0.8.0</version>
</dependency>
```

Spring Boot Starter (включает core):

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.8.0</version>
</dependency>
```

### Gradle

```groovy
// Core-модуль
implementation 'biz.kryukov.dev:dephealth-core:0.8.0'

// Или Spring Boot Starter
implementation 'biz.kryukov.dev:dephealth-spring-boot-starter:0.8.0'
```

## Минимальный пример

Мониторинг одной HTTP-зависимости с экспортом метрик Prometheus:

```java
import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.model.DependencyType;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;

public class Main {
    public static void main(String[] args) {
        var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

        var dh = DepHealth.builder("my-service", "my-team", registry)
            .dependency("payment-api", DependencyType.HTTP, d -> d
                .url("http://payment.svc:8080")
                .critical(true))
            .build();

        dh.start();

        // Экспортируйте registry.scrape() на /metrics через HTTP-сервер

        // Graceful shutdown
        Runtime.getRuntime().addShutdownHook(new Thread(dh::stop));
    }
}
```

После запуска метрики Prometheus доступны по адресу `/metrics`:

```text
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Ключевые концепции

### Имя и группа

Каждый экземпляр `DepHealth` требует два идентификатора:

- **name** — уникальное имя приложения (например, `"my-service"`)
- **group** — логическая группа сервиса (например, `"my-team"`, `"payments"`)

Оба значения появляются как метки во всех экспортируемых метриках.
Правила валидации: `[a-z][a-z0-9-]*`, от 1 до 63 символов.

Если не переданы как аргументы, SDK использует переменные окружения
`DEPHEALTH_NAME` и `DEPHEALTH_GROUP` как запасной вариант.

### Зависимости

Каждая зависимость регистрируется через метод `.dependency()` билдера
с указанием `DependencyType`:

| DependencyType | Описание |
| --- | --- |
| `HTTP` | HTTP-сервис |
| `GRPC` | gRPC-сервис |
| `TCP` | TCP-эндпоинт |
| `POSTGRES` | База данных PostgreSQL |
| `MYSQL` | База данных MySQL |
| `REDIS` | Сервер Redis |
| `AMQP` | RabbitMQ (AMQP-брокер) |
| `KAFKA` | Брокер Apache Kafka |
| `LDAP` | LDAP-сервер каталогов |

Для каждой зависимости обязательны:

- **Имя** (первый аргумент) — идентификатор зависимости в метриках
- **Эндпоинт** — через `.url()`, `.jdbcUrl()` или `.host()` + `.port()`
- **Флаг критичности** — `.critical(true)` или `.critical(false)` (обязателен)

### Жизненный цикл

1. **Создание** — `DepHealth.builder(...).build()`
2. **Запуск** — `dh.start()` запускает периодические проверки
3. **Работа** — проверки выполняются с заданным интервалом (по умолчанию 15 сек)
4. **Остановка** — `dh.stop()` останавливает проверки и завершает планировщик

## Несколько зависимостей

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    // Глобальные настройки
    .checkInterval(Duration.ofSeconds(30))
    .timeout(Duration.ofSeconds(3))

    // PostgreSQL
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true))

    // Redis
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .url(System.getenv("REDIS_URL"))
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

    .build();
```

## Проверка состояния

### Простой статус

```java
Map<String, Boolean> health = dh.health();
// {"postgres-main:pg.svc:5432": true, "redis-cache:redis.svc:6379": true}

// Использование для readiness probe
boolean allHealthy = health.values().stream().allMatch(Boolean::booleanValue);
```

### Подробный статус

```java
Map<String, EndpointStatus> details = dh.healthDetails();
details.forEach((key, ep) ->
    System.out.printf("%s: healthy=%s status=%s latency=%.1fms%n",
        key, ep.isHealthy(), ep.getStatus(), ep.getLatencyMillis()));
```

`healthDetails()` возвращает объект `EndpointStatus` с состоянием
здоровья, категорией статуса, задержкой, временными метками и
пользовательскими метками. До завершения первой проверки `healthy`
равен `null`, а `status` — `"unknown"`.

## Дальнейшие шаги

- [Чекеры](checkers.ru.md) — подробное руководство по всем 9 встроенным чекерам
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и переменные окружения
- [Пулы соединений](connection-pools.ru.md) — интеграция с существующими пулами соединений
- [Spring Boot интеграция](spring-boot.ru.md) — авто-конфигурация и actuator
- [Аутентификация](authentication.ru.md) — авторизация для HTTP, gRPC и чекеров БД
- [Метрики](metrics.ru.md) — справочник по метрикам Prometheus и примеры PromQL
- [API Reference](api-reference.ru.md) — полный справочник по всем публичным классам
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
- [Руководство по миграции](migration.ru.md) — инструкции по обновлению версий
- [Стиль кода](code-style.ru.md) — соглашения по стилю кода Java
- [Примеры](examples/) — полные рабочие примеры
