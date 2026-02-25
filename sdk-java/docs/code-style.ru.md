*[English version](code-style.md)*

# Руководство по стилю кода: Java SDK

Этот документ описывает соглашения по стилю кода для Java SDK (`sdk-java/`).
См. также: [Общие принципы](../../docs/code-style/overview.ru.md) | [Тестирование](../../docs/code-style/testing.ru.md)

## Соглашения об именовании

### Пакеты

Используйте обратную доменную нотацию в нижнем регистре:

```java
biz.kryukov.dev.dephealth          // ядро
biz.kryukov.dev.dephealth.checks   // чекеры
biz.kryukov.dev.dephealth.model    // классы моделей
biz.kryukov.dev.dephealth.metrics  // экспортёр метрик
biz.kryukov.dev.dephealth.scheduler // планировщик проверок
biz.kryukov.dev.dephealth.parser   // парсер конфигурации
biz.kryukov.dev.dephealth.spring   // Spring Boot интеграция
```

### Классы и интерфейсы

- `PascalCase` для всех типов
- Интерфейсы: существительное или прилагательное, без префикса `I` (в отличие от C#)
- Реализации: описательные имена, без суффикса `Impl`

```java
// Хорошо
public interface HealthChecker { }
public class HttpHealthChecker implements HealthChecker { }
public class TcpHealthChecker implements HealthChecker { }

// Плохо
public interface IHealthChecker { }     // без I-префикса в Java
public class HttpHealthCheckerImpl { }  // без суффикса Impl
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
    HTTP, GRPC, TCP, POSTGRES, MYSQL, REDIS, AMQP, KAFKA, LDAP
}
```

## Структура пакетов

```text
sdk-java/
├── dephealth-core/
│   └── src/main/java/biz/kryukov/dev/dephealth/
│       ├── DepHealth.java                  // основной API, builder
│       ├── model/
│       │   ├── Dependency.java             // модель
│       │   ├── Endpoint.java               // модель
│       │   ├── DependencyType.java         // enum
│       │   ├── EndpointStatus.java         // статус здоровья
│       │   ├── CheckConfig.java            // модель конфигурации
│       │   ├── CheckResult.java            // результат проверки
│       │   └── StatusCategory.java         // константы статуса
│       ├── HealthChecker.java              // интерфейс чекера
│       ├── checks/
│       │   ├── HttpHealthChecker.java
│       │   ├── GrpcHealthChecker.java
│       │   ├── TcpHealthChecker.java
│       │   ├── PostgresHealthChecker.java
│       │   ├── MysqlHealthChecker.java
│       │   ├── RedisHealthChecker.java
│       │   ├── AmqpHealthChecker.java
│       │   ├── KafkaHealthChecker.java
│       │   └── LdapHealthChecker.java
│       ├── scheduler/
│       │   └── CheckScheduler.java
│       ├── metrics/
│       │   └── MetricsExporter.java
│       ├── parser/
│       │   ├── ConfigParser.java
│       │   └── ParsedConnection.java
│       └── exceptions/
│           ├── CheckException.java         // базовое исключение
│           ├── CheckAuthException.java
│           ├── CheckConnectionException.java
│           ├── UnhealthyException.java
│           ├── ValidationException.java
│           ├── ConfigurationException.java
│           ├── EndpointNotFoundException.java
│           └── DepHealthException.java
│
└── dephealth-spring-boot-starter/
    └── src/main/java/biz/kryukov/dev/dephealth/spring/
        ├── DepHealthAutoConfiguration.java
        ├── DepHealthProperties.java
        ├── DepHealthLifecycle.java
        ├── DepHealthIndicator.java
        └── DependenciesEndpoint.java
```

## Обработка ошибок

### Иерархия исключений

Используйте **checked-исключения** для `HealthChecker.check()` (наследники `Exception`)
и **unchecked-исключения** для ошибок конфигурации. Планировщик перехватывает
все исключения от чекеров.

```java
// Иерархия CheckException (checked)
public class CheckException extends Exception {
    public String statusCategory() { ... }
    public String statusDetail() { ... }
}

public class CheckAuthException extends CheckException { }
public class CheckConnectionException extends CheckException { }

// Исключения конфигурации (unchecked)
public class ValidationException extends RuntimeException { }
public class ConfigurationException extends RuntimeException { }
```

### Правила

- Ошибки конфигурации: бросать немедленно во время `build()`
- Ошибки проверки: бросать конкретные подтипы `CheckException`, перехватываемые планировщиком
- Никогда не проглатывать исключения — всегда логировать перед подавлением
- Всегда включать цепочку причин: `new CheckException("msg", cause)`

```java
// Хорошо — информативное сообщение с причиной
throw new CheckAuthException(
    String.format("Authentication failed for %s:%s",
        endpoint.host(), endpoint.port()),
    cause);

// Плохо — теряет контекст
throw new CheckException("auth error");
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
 * @throws CheckException if the check fails
 */
void check(Endpoint endpoint, Duration timeout) throws Exception;
```

Правила:

- Первое предложение: краткое описание на английском (показывается в тултипах IDE)
- Используйте `@param`, `@return`, `@throws` для всех параметров, значений и исключений
- Используйте `{@code}` для inline-кода, `{@link}` для перекрёстных ссылок
- Гарантии потокобезопасности — в блоке `<p>`

## Builder Pattern

Используйте builder pattern для конфигурации `DepHealth`:

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .checkInterval(Duration.ofSeconds(15))
    .timeout(Duration.ofSeconds(5))
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL"))
        .critical(true))
    .build();

dh.start();
```

Правила:

- Builder — **единственный** способ создать `DepHealth`
- Методы builder-а возвращают `this` для цепочки вызовов
- `build()` валидирует все параметры и возвращает неизменяемый объект
- `start()` отделён от `build()` — позволяет инспектировать объект перед запуском

## Неизменяемость и null safety

- **Предпочитайте неизменяемые объекты**: `Dependency`, `Endpoint`, `CheckConfig`, `EndpointStatus` — неизменяемы после создания
- **Нет `null` в публичном API**: используйте `Optional<T>` для необязательных возвращаемых значений, `Boolean` (nullable) для неизвестного состояния
- **Валидируйте параметры**: используйте `Objects.requireNonNull()` на входе метода

```java
// Хорошо — неизменяемая модель с валидацией
public final class Endpoint {
    public Endpoint(String host, String port) {
        this.host = Objects.requireNonNull(host, "host must not be null");
        this.port = Objects.requireNonNull(port, "port must not be null");
    }
}
```

- Используйте `final` для полей, которые не должны изменяться
- Используйте `Collections.unmodifiableMap()` или `Map.copyOf()` для коллекций

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

Конфигурация: `sdk-java/dephealth-core/checkstyle.xml` (на основе Google
с модификациями проекта).

Основные правила:

- Отступы: 4 пробела (без табов)
- Максимальная длина строки: не ограничена (перенос в IDE)
- Порядок импортов: static первыми, затем `java`, `javax`, сторонние, проектные
- Без wildcard-импортов
- Фигурные скобки обязательны для всех блоков `if`/`else`/`for`/`while`

### SpotBugs

Детектирует распространённые ошибки: разыменование null, утечки ресурсов,
проблемы параллелизма.

### Запуск

```bash
cd sdk-java && make lint    # запускает Checkstyle и SpotBugs в Docker
cd sdk-java && make fmt     # только Checkstyle
```

## Дополнительные соглашения

- **Версия Java**: 21 LTS — используйте records, sealed classes, pattern matching где уместно
- **Зависимости**: минимизируйте внешние зависимости в `dephealth-core`
- **Метрики**: используйте Micrometer для регистрации метрик
- **Потокобезопасность**: документируйте гарантии на каждом публичном классе
- **Управление ресурсами**: используйте try-with-resources для `Closeable`/`AutoCloseable`
- **Daemon-потоки**: планировщик использует daemon-потоки, не мешает завершению JVM

## См. также

- [Общие принципы](../../docs/code-style/overview.ru.md) — кросс-SDK стиль кода
- [Тестирование](../../docs/code-style/testing.ru.md) — соглашения по тестированию
