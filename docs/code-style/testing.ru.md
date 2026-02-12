*[English version](testing.md)*

# Code Style Guide: тестирование

Этот документ описывает соглашения по тестированию для всех dephealth SDK.
См. также: [Общие принципы](overview.ru.md) | [Java](java.ru.md) | [Go](go.ru.md) | [Python](python.ru.md) | [C#](csharp.ru.md)

## Именование тестов

Каждый язык следует своим идиоматичным соглашениям:

| Язык | Соглашение | Пример |
| --- | --- | --- |
| Go | Функция `TestXxx` в `_test.go` | `TestHTTPChecker_Check_HealthyEndpoint` |
| Java | Методы `@Test`, camelCase | `httpCheckerReturnsHealthyForOkResponse` |
| Python | Префикс `test_`, snake_case | `test_http_checker_healthy_endpoint` |
| C# | `PascalCase_Should_When` | `HttpChecker_CheckAsync_Should_Succeed_When_EndpointReturns200` |

### Go: именование тестов

Используйте паттерн `TestType_Method_Scenario`:

```go
func TestHTTPChecker_Check_HealthyEndpoint(t *testing.T) { }
func TestHTTPChecker_Check_TimeoutError(t *testing.T) { }
func TestHTTPChecker_Check_ConnectionRefused(t *testing.T) { }
func TestConnectionParser_Parse_PostgresURL(t *testing.T) { }
```

### Java: именование тестов

Используйте описательные camelCase-имена, читаемые как предложения:

```java
@Test
void httpCheckerReturnsHealthyForOkResponse() { }

@Test
void httpCheckerThrowsTimeoutOnSlowEndpoint() { }

@Test
void connectionParserExtractsHostAndPort() { }
```

### Python: именование тестов

Используйте префикс `test_` со snake_case:

```python
def test_http_checker_healthy_endpoint() -> None: ...
async def test_http_checker_timeout_error() -> None: ...
def test_connection_parser_postgres_url() -> None: ...
```

### C#: именование тестов

Используйте `Subject_Should_Condition` или `Subject_Method_Should_Condition_When_Scenario`:

```csharp
[Fact]
public async Task CheckAsync_Should_Succeed_When_EndpointReturns200() { }

[Fact]
public async Task CheckAsync_Should_ThrowTimeout_When_EndpointIsSlow() { }

[Fact]
public void ParseUrl_Should_ExtractHostAndPort() { }
```

## Структура тестов: AAA

Все тесты следуют паттерну **Arrange-Act-Assert**:

```go
// Пример на Go
func TestHTTPChecker_Check_HealthyEndpoint(t *testing.T) {
    // Arrange
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    checker := &HTTPChecker{client: server.Client()}
    endpoint := Endpoint{Host: "localhost", Port: 8080}

    // Act
    err := checker.Check(context.Background(), endpoint)

    // Assert
    assert.NoError(t, err)
}
```

```java
// Пример на Java
@Test
void httpCheckerReturnsHealthyForOkResponse() {
    // Arrange
    var server = MockWebServer();
    server.enqueue(new MockResponse().setResponseCode(200));
    var checker = new HttpChecker(server.url("/").toString());
    var endpoint = new Endpoint("localhost", 8080, Map.of());

    // Act & Assert
    assertDoesNotThrow(() -> checker.check(endpoint, Duration.ofSeconds(5)));
}
```

```python
# Пример на Python
async def test_http_checker_healthy_endpoint() -> None:
    # Arrange
    checker = HTTPChecker(timeout=5.0)
    endpoint = Endpoint(host="localhost", port=8080)

    # Act & Assert (с мокой)
    with aioresponses() as mocked:
        mocked.get("http://localhost:8080/health", status=200)
        await checker.check(endpoint)  # не должен бросить исключение
```

```csharp
// Пример на C#
[Fact]
public async Task CheckAsync_Should_Succeed_When_EndpointReturns200()
{
    // Arrange
    var handler = new MockHttpMessageHandler();
    handler.When("*").Respond(HttpStatusCode.OK);
    var checker = new HttpChecker(new HttpClient(handler));
    var endpoint = new Endpoint("localhost", 8080, ImmutableDictionary<string, string>.Empty);

    // Act
    var act = () => checker.CheckAsync(endpoint, CancellationToken.None);

    // Assert
    await act.Should().NotThrowAsync();
}
```

## Что тестировать в checker-ах

Каждый health checker должен покрывать эти сценарии как минимум:

| Сценарий | Что проверять |
| --- | --- |
| **Happy path** | Проверка возвращает успех (nil/void/None) для здорового эндпоинта |
| **Ошибка соединения** | Проверка возвращает/бросает соответствующую ошибку при недоступном эндпоинте |
| **Таймаут** | Проверка уважает таймаут и возвращает/бросает ошибку таймаута |
| **Учётные данные** | Credentials из URL не утекают в ошибки или метрики |
| **Нездоровый ответ** | Проверка обнаруживает нездоровый ответ (HTTP 5xx, gRPC NOT_SERVING) |

### Дополнительные тесты по типам checker-ов

| Checker | Дополнительные тесты |
| --- | --- |
| HTTP | Обработка редиректов, кастомный health path, не-2xx статусы |
| gRPC | Обработка имени сервиса, TLS, статус NOT_SERVING |
| TCP | Закрытый порт, сброс соединения |
| Postgres/MySQL | Выполнение запроса, поведение connection pool |
| Redis | Требуется AUTH, неправильная база данных |
| AMQP | Virtual host, аутентификация |
| Kafka | Список брокеров, metadata топиков |

## Мокирование

### Go: testify/mock или httptest

```go
// httptest для HTTP-checker-ов
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}))
defer server.Close()

// testify для интерфейсов
type mockChecker struct {
    mock.Mock
}

func (m *mockChecker) Check(ctx context.Context, ep Endpoint) error {
    args := m.Called(ctx, ep)
    return args.Error(0)
}
```

### Java: Mockito + MockWebServer

```java
// Mockito для интерфейсов
var checker = mock(HealthChecker.class);
when(checker.check(any(), any())).thenThrow(new CheckTimeoutException("timeout", null));

// MockWebServer для HTTP
var server = new MockWebServer();
server.enqueue(new MockResponse().setResponseCode(503));
```

### Python: unittest.mock + aioresponses

```python
# unittest.mock для интерфейсов
from unittest.mock import AsyncMock

checker = AsyncMock(spec=HealthChecker)
checker.check.side_effect = CheckTimeoutError("timeout")

# aioresponses для HTTP
from aioresponses import aioresponses

with aioresponses() as mocked:
    mocked.get("http://localhost:8080/health", status=503)
    with pytest.raises(UnhealthyError):
        await checker.check(endpoint)
```

### C#: Moq + MockHttpMessageHandler

```csharp
// Moq для интерфейсов
var checker = new Mock<IHealthChecker>();
checker.Setup(c => c.CheckAsync(It.IsAny<Endpoint>(), It.IsAny<CancellationToken>()))
    .ThrowsAsync(new CheckTimeoutException("timeout", null));

// MockHttpMessageHandler для HTTP
var handler = new MockHttpMessageHandler();
handler.When("*").Respond(HttpStatusCode.ServiceUnavailable);
```

### Когда мокировать

- **Мокируйте**: внешние зависимости (HTTP-серверы, базы данных, брокеры сообщений)
- **Мокируйте**: интерфейсы между слоями (интерфейс checker-а, экспортёр метрик)
- **Не мокируйте**: простые value-объекты, структуры данных, чистые функции
- **Не мокируйте**: сам тестируемый код

## Table-Driven Tests

Предпочтительный стиль в Go (также применим в других языках с параметризованными тестами):

```go
func TestConnectionParser_Parse(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        wantHost string
        wantPort int
        wantType string
        wantErr  bool
    }{
        {
            name:     "postgres URL",
            input:    "postgres://user:pass@db.svc:5432/orders",
            wantHost: "db.svc",
            wantPort: 5432,
            wantType: "postgres",
        },
        {
            name:     "redis URL",
            input:    "redis://cache.svc:6379/0",
            wantHost: "cache.svc",
            wantPort: 6379,
            wantType: "redis",
        },
        {
            name:    "invalid URL",
            input:   "not-a-url",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ep, err := ParseURL(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.wantHost, ep.Host)
            assert.Equal(t, tt.wantPort, ep.Port)
        })
    }
}
```

Эквивалент на Java с `@ParameterizedTest`:

```java
@ParameterizedTest
@CsvSource({
    "postgres://user:pass@db.svc:5432/orders, db.svc, 5432, postgres",
    "redis://cache.svc:6379/0, cache.svc, 6379, redis",
})
void connectionParserExtractsHostAndPort(String url, String host, int port, String type) {
    var endpoint = ConnectionParser.parseUrl(url);
    assertThat(endpoint.host()).isEqualTo(host);
    assertThat(endpoint.port()).isEqualTo(port);
}
```

Эквивалент на Python с `@pytest.mark.parametrize`:

```python
@pytest.mark.parametrize("url,host,port,dep_type", [
    ("postgres://user:pass@db.svc:5432/orders", "db.svc", 5432, "postgres"),
    ("redis://cache.svc:6379/0", "cache.svc", 6379, "redis"),
])
def test_parse_url(url: str, host: str, port: int, dep_type: str) -> None:
    endpoint = parse_url(url)
    assert endpoint.host == host
    assert endpoint.port == port
```

Эквивалент на C# с `[Theory]`:

```csharp
[Theory]
[InlineData("postgres://user:pass@db.svc:5432/orders", "db.svc", 5432)]
[InlineData("redis://cache.svc:6379/0", "cache.svc", 6379)]
public void ParseUrl_Should_ExtractHostAndPort(string url, string expectedHost, int expectedPort)
{
    var endpoint = ConnectionParser.ParseUrl(url);
    endpoint.Host.Should().Be(expectedHost);
    endpoint.Port.Should().Be(expectedPort);
}
```

## Покрытие

Целевые уровни покрытия:

| Компонент | Цель |
| --- | --- |
| Базовые абстракции (модели, парсер) | 90%+ |
| Health checkers | 80%+ (ограничено внешними зависимостями) |
| Планировщик | 70%+ (тестирование параллелизма сложно) |
| Интеграция с фреймворком | 60%+ (предпочтительны интеграционные тесты) |

Покрытие — информативная метрика, а не жёсткий порог. 100% покрытие — не цель.
Фокусируйтесь на тестировании осмысленного поведения, а не на достижении числа.

## Тестовая инфраструктура

### Docker Compose

Unit-тесты используют моки и не требуют внешних сервисов.
Интеграционные тесты используют Docker Compose с реальными сервисами:

```bash
# Запуск тестовой инфраструктуры
docker compose up -d  # PostgreSQL + Redis (минимум)
docker compose --profile full up -d  # все 7 зависимостей

# Запуск тестов
cd sdk-go && make test
cd sdk-python && make test
cd sdk-java && make test
cd sdk-csharp && make test
```

### Make-цели

Каждый SDK предоставляет единообразные цели для тестирования:

| Цель | Описание |
| --- | --- |
| `make test` | Запуск unit-тестов |
| `make test-coverage` | Тесты с отчётом о покрытии |
| `make lint` | Линтинг (включая тестовые файлы) |

Все тестовые команды выполняются в Docker-контейнерах — локальная установка SDK не требуется.

## Организация тестовых файлов

### Go

```text
dephealth/
├── checker.go
├── checker_test.go          # тесты рядом с исходниками
├── checks/
│   ├── http.go
│   └── http_test.go
```

### Java

```text
dephealth-core/
├── src/main/java/.../HttpChecker.java
└── src/test/java/.../HttpCheckerTest.java    # зеркальная структура
```

### Python

```text
tests/
├── conftest.py              # общие фикстуры
├── test_checker.py
├── test_parser.py
└── checks/
    ├── test_http.py
    └── test_postgres.py
```

### C\#

```text
DepHealth.Core.Tests/
├── HttpCheckerTests.cs      # отдельный тестовый проект
├── ConnectionParserTests.cs
└── Checks/
    └── ...
```
