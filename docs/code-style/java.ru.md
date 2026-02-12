*[English version](java.md)*

# Code Style Guide: Java SDK

Этот документ описывает соглашения по стилю кода для Java SDK (`sdk-java/`).
См. также: [Общие принципы](overview.ru.md) | [Тестирование](testing.ru.md)

## Соглашения об именовании

### Пакеты

Используйте обратную доменную нотацию в нижнем регистре:

```java
biz.kryukov.dev.dephealth          // ядро
biz.kryukov.dev.dephealth.checks   // health checkers
biz.kryukov.dev.dephealth.spring   // Spring Boot интеграция
```

### Классы и интерфейсы

- `PascalCase` для всех типов
- Интерфейсы: существительное или прилагательное, без префикса `I` (в отличие от C#)
- Реализации: описательные имена, без суффикса `Impl`

```java
// Хорошо
public interface HealthChecker { }
public class HttpChecker implements HealthChecker { }
public class TcpChecker implements HealthChecker { }

// Плохо
public interface IHealthChecker { }     // без I-префикса в Java
public class HttpCheckerImpl { }        // без суффикса Impl
```

### Методы и переменные

- Методы: `camelCase`, начинаются с глагола
- Локальные переменные: `camelCase`
- Константы: `UPPER_SNAKE_CASE`
- Булевы методы: префикс `is`/`has`/`can`

```java
public void check(Endpoint endpoint, Duration timeout) { }
public DependencyType type() { }

private static final Duration DEFAULT_TIMEOUT = Duration.ofSeconds(5);
private static final int MAX_RETRY_COUNT = 3;

public boolean isCritical() { }
public boolean hasEndpoints() { }
```

### Перечисления (enum)

- Имя типа: `PascalCase`, единственное число
- Значения: `UPPER_SNAKE_CASE`

```java
public enum DependencyType {
    HTTP, GRPC, TCP, POSTGRES, MYSQL, REDIS, AMQP, KAFKA
}
```

## Структура пакетов

```text
sdk-java/
├── dephealth-core/
│   └── src/main/java/biz/kryukov/dev/dephealth/
│       ├── DependencyHealth.java          // основной API, builder
│       ├── Dependency.java                // модель
│       ├── Endpoint.java                  // модель
│       ├── DependencyType.java            // enum
│       ├── HealthChecker.java             // интерфейс проверки
│       ├── CheckScheduler.java            // планировщик
│       ├── ConnectionParser.java          // парсер URL/params
│       ├── PrometheusExporter.java        // метрики
│       ├── DepHealthException.java        // базовое исключение
│       └── checks/
│           ├── HttpChecker.java
│           ├── GrpcChecker.java
│           ├── TcpChecker.java
│           ├── PostgresChecker.java
│           ├── RedisChecker.java
│           ├── AmqpChecker.java
│           └── KafkaChecker.java
│
└── dephealth-spring-boot-starter/
    └── src/main/java/biz/kryukov/dev/dephealth/spring/
        ├── DepHealthAutoConfiguration.java
        └── DepHealthProperties.java
```

## Обработка ошибок

### Иерархия исключений

Используйте **unchecked-исключения** (наследники `RuntimeException`). Checked-исключения
создают лишний boilerplate для пользователей библиотеки.

```java
public class DepHealthException extends RuntimeException {
    public DepHealthException(String message) { super(message); }
    public DepHealthException(String message, Throwable cause) { super(message, cause); }
}

public class CheckTimeoutException extends DepHealthException { }
public class ConnectionRefusedException extends DepHealthException { }
```

### Правила

- Ошибки конфигурации: бросать `IllegalArgumentException` или `DepHealthException` немедленно
- Ошибки проверки: бросать конкретные подтипы, перехватываемые планировщиком
- Никогда не проглатывать исключения — всегда логировать перед подавлением
- Всегда включать цепочку причин: `new DepHealthException("msg", cause)`

```java
// Хорошо — информативное сообщение с причиной
throw new CheckTimeoutException(
    String.format("Health check timed out for %s:%d after %s",
        endpoint.host(), endpoint.port(), timeout),
    cause);

// Плохо — теряет контекст
throw new DepHealthException("timeout");
```

## JavaDoc

### Что документировать

- Все `public` и `protected` типы и члены
- Все интерфейсы и их контракты
- Неочевидное поведение, побочные эффекты, гарантии потокобезопасности

### Формат

```java
/**
 * Performs a health check against the given endpoint.
 *
 * <p>Implementations must be thread-safe.</p>
 *
 * @param endpoint the endpoint to check
 * @param timeout  maximum wait time
 * @throws CheckTimeoutException if the check did not complete within timeout
 * @throws ConnectionRefusedException if the connection was refused
 */
void check(Endpoint endpoint, Duration timeout);
```

Правила:

- Первое предложение: краткое описание на английском (показывается в тултипах IDE)
- Используйте `@param`, `@return`, `@throws` для всех параметров, возвращаемых значений и исключений
- Используйте `{@code}` для inline-кода, `{@link}` для перекрёстных ссылок
- Гарантии потокобезопасности — в блоке `<p>`

## Builder Pattern

Используйте builder pattern для конфигурации `DependencyHealth`:

```java
DependencyHealth health = DependencyHealth.builder("order-service")
    .dependency("postgres-main", DependencyType.POSTGRES,
        Endpoint.fromUrl(System.getenv("DATABASE_URL")),
        DependencyOptions.builder().critical(true).build())
    .dependency("redis-cache", DependencyType.REDIS,
        Endpoint.fromUrl(System.getenv("REDIS_URL")))
    .checkInterval(Duration.ofSeconds(15))
    .timeout(Duration.ofSeconds(5))
    .build();

health.start();
```

Правила:

- Builder — **единственный** способ создать `DependencyHealth`
- Методы builder-а возвращают `this` для цепочки вызовов
- `build()` валидирует все параметры и возвращает неизменяемый объект
- `start()` отделён от `build()` — позволяет инспектировать объект перед запуском

## Неизменяемость и null safety

- **Предпочитайте неизменяемые объекты**: `Dependency`, `Endpoint`, конфигурация — неизменяемы после создания
- **Нет `null` в публичном API**: используйте `Optional<T>` для необязательных возвращаемых значений
- **Валидируйте параметры**: используйте `Objects.requireNonNull()` на входе метода

```java
// Хорошо — неизменяемая модель
public record Endpoint(String host, int port, Map<String, String> metadata) {
    public Endpoint {
        Objects.requireNonNull(host, "host must not be null");
        if (port <= 0 || port > 65535) {
            throw new IllegalArgumentException("port must be 1-65535, got: " + port);
        }
        metadata = Map.copyOf(metadata); // защитная копия
    }
}
```

- Используйте `final` для полей, которые не должны изменяться
- Используйте `Collections.unmodifiable*()` или `Map.copyOf()` для коллекций в публичном API

## Логирование

Используйте SLF4J с параметризованными сообщениями:

```java
private static final Logger log = LoggerFactory.getLogger(CheckScheduler.class);

// Хорошо — параметризованное (ленивое вычисление)
log.info("Starting check scheduler, {} dependencies", dependencies.size());
log.warn("Check {} failed: {}", dependency.name(), error.getMessage());
log.debug("Check {} completed in {}ms", dependency.name(), elapsed);

// Плохо — конкатенация строк
log.info("Starting scheduler for " + dependencies.size() + " dependencies");
```

- Используйте `log.isDebugEnabled()` только для дорогих для вычисления сообщений
- Никогда не логируйте учётные данные — санитизируйте URL перед логированием

## Линтеры

### Checkstyle

Конфигурация: `sdk-java/checkstyle.xml` (на основе Google с модификациями проекта).

Основные правила:

- Отступы: 4 пробела (без табов)
- Максимальная длина строки: не ограничена (перенос в IDE)
- Порядок импортов: static первыми, затем `java`, `javax`, сторонние, проектные
- Без wildcard-импортов
- Фигурные скобки обязательны для всех блоков `if`/`else`/`for`/`while`

### SpotBugs

Детектирует распространённые ошибки: разыменование null, утечки ресурсов, проблемы параллелизма.

### Запуск

```bash
cd sdk-java && make lint    # запускает Checkstyle и SpotBugs в Docker
cd sdk-java && make fmt     # автоформатирование google-java-format
```

## Дополнительные соглашения

- **Версия Java**: 21 LTS — используйте records, sealed classes, pattern matching где уместно
- **Зависимости**: минимизируйте внешние зависимости в `dephealth-core`
- **Метрики**: используйте Micrometer для регистрации метрик
- **Потокобезопасность**: документируйте гарантии на каждом публичном классе
  (`@ThreadSafe`, `@NotThreadSafe`, или комментарий)
- **Управление ресурсами**: используйте try-with-resources для `Closeable`/`AutoCloseable`
