*[Русская версия](testing.ru.md)*

# Code Style Guide: Testing

This document describes testing conventions across all dephealth SDKs.
See also: [General Principles](overview.md) | [Go](../../sdk-go/docs/code-style.md) | [Java](../../sdk-java/docs/code-style.md) | [Python](../../sdk-python/docs/code-style.md) | [C#](../../sdk-csharp/docs/code-style.md)

## Test Naming

Each language follows its idiomatic naming convention:

| Language | Convention | Example |
| --- | --- | --- |
| Go | `TestXxx` function in `_test.go` | `TestHTTPChecker_Check_HealthyEndpoint` |
| Java | `@Test` methods, camelCase | `httpCheckerReturnsHealthyForOkResponse` |
| Python | `test_` prefix, snake_case | `test_http_checker_healthy_endpoint` |
| C# | `PascalCase_Should_When` | `HttpChecker_CheckAsync_Should_Succeed_When_EndpointReturns200` |

### Go Test Naming

Use the pattern `TestType_Method_Scenario`:

```go
func TestHTTPChecker_Check_HealthyEndpoint(t *testing.T) { }
func TestHTTPChecker_Check_TimeoutError(t *testing.T) { }
func TestHTTPChecker_Check_ConnectionRefused(t *testing.T) { }
func TestConnectionParser_Parse_PostgresURL(t *testing.T) { }
```

### Java Test Naming

Use descriptive camelCase names that read like sentences:

```java
@Test
void httpCheckerReturnsHealthyForOkResponse() { }

@Test
void httpCheckerThrowsTimeoutOnSlowEndpoint() { }

@Test
void connectionParserExtractsHostAndPort() { }
```

### Python Test Naming

Use `test_` prefix with snake_case:

```python
def test_http_checker_healthy_endpoint() -> None: ...
async def test_http_checker_timeout_error() -> None: ...
def test_connection_parser_postgres_url() -> None: ...
```

### C# Test Naming

Use `Subject_Should_Condition` or `Subject_Method_Should_Condition_When_Scenario`:

```csharp
[Fact]
public async Task CheckAsync_Should_Succeed_When_EndpointReturns200() { }

[Fact]
public async Task CheckAsync_Should_ThrowTimeout_When_EndpointIsSlow() { }

[Fact]
public void ParseUrl_Should_ExtractHostAndPort() { }
```

## Test Structure: AAA

All tests follow the **Arrange-Act-Assert** pattern:

```go
// Go example
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
// Java example
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
# Python example
async def test_http_checker_healthy_endpoint() -> None:
    # Arrange
    checker = HTTPChecker(timeout=5.0)
    endpoint = Endpoint(host="localhost", port=8080)

    # Act & Assert (with mock)
    with aioresponses() as mocked:
        mocked.get("http://localhost:8080/health", status=200)
        await checker.check(endpoint)  # should not raise
```

```csharp
// C# example
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

## What to Test in Checkers

Every health checker must cover these scenarios at minimum:

| Scenario | What to verify |
| --- | --- |
| **Happy path** | Check returns success (nil/void/None) for a healthy endpoint |
| **Connection error** | Check returns/throws appropriate error when endpoint is unreachable |
| **Timeout** | Check respects timeout and returns/throws timeout error |
| **Credentials** | Credentials from URL are not leaked in errors or metrics |
| **Unhealthy response** | Check detects an unhealthy response (e.g., HTTP 5xx, gRPC NOT_SERVING) |

### Additional checker-specific tests

| Checker | Additional tests |
| --- | --- |
| HTTP | Redirect handling, custom health path, non-2xx status codes |
| gRPC | Service name handling, TLS, NOT_SERVING status |
| TCP | Port closed, connection reset |
| Postgres/MySQL | Query execution, connection pool behavior |
| Redis | AUTH required, wrong database |
| AMQP | Virtual host, authentication |
| Kafka | Broker list, topic metadata |

## Mocking

### Go: testify/mock or httptest

```go
// httptest for HTTP checkers
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}))
defer server.Close()

// testify for interfaces
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
// Mockito for interfaces
var checker = mock(HealthChecker.class);
when(checker.check(any(), any())).thenThrow(new CheckTimeoutException("timeout", null));

// MockWebServer for HTTP
var server = new MockWebServer();
server.enqueue(new MockResponse().setResponseCode(503));
```

### Python: unittest.mock + aioresponses

```python
# unittest.mock for interfaces
from unittest.mock import AsyncMock

checker = AsyncMock(spec=HealthChecker)
checker.check.side_effect = CheckTimeoutError("timeout")

# aioresponses for HTTP
from aioresponses import aioresponses

with aioresponses() as mocked:
    mocked.get("http://localhost:8080/health", status=503)
    with pytest.raises(UnhealthyError):
        await checker.check(endpoint)
```

### C#: Moq + MockHttpMessageHandler

```csharp
// Moq for interfaces
var checker = new Mock<IHealthChecker>();
checker.Setup(c => c.CheckAsync(It.IsAny<Endpoint>(), It.IsAny<CancellationToken>()))
    .ThrowsAsync(new CheckTimeoutException("timeout", null));

// MockHttpMessageHandler for HTTP
var handler = new MockHttpMessageHandler();
handler.When("*").Respond(HttpStatusCode.ServiceUnavailable);
```

### When to Mock

- **Do mock**: external dependencies (HTTP servers, databases, message brokers)
- **Do mock**: interfaces between layers (checker interface, metrics exporter)
- **Don't mock**: simple value objects, data structures, pure functions
- **Don't mock**: the code under test itself

## Table-Driven Tests

Preferred style in Go (also applicable in other languages with parameterized tests):

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

Java equivalent with `@ParameterizedTest`:

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

Python equivalent with `@pytest.mark.parametrize`:

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

C# equivalent with `[Theory]`:

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

## Coverage

Target coverage levels:

| Component | Target |
| --- | --- |
| Core abstractions (models, parser) | 90%+ |
| Health checkers | 80%+ (limited by external dependencies) |
| Scheduler | 70%+ (concurrency testing is hard) |
| Framework integration | 60%+ (integration tests preferred) |

Coverage is informational, not a hard gate. 100% coverage is not a goal —
focus on testing meaningful behavior, not hitting coverage numbers.

## Test Infrastructure

### Docker Compose

Unit tests use mocks and don't require external services.
Integration tests use Docker Compose with real services:

```bash
# Start test infrastructure
docker compose up -d  # PostgreSQL + Redis (minimal)
docker compose --profile full up -d  # all 7 dependencies

# Run tests
cd sdk-go && make test
cd sdk-python && make test
cd sdk-java && make test
cd sdk-csharp && make test
```

### Make Targets

Each SDK provides uniform test targets:

| Target | Description |
| --- | --- |
| `make test` | Run unit tests |
| `make test-coverage` | Tests with coverage report |
| `make lint` | Linting (includes test files) |

All test commands run in Docker containers — no local SDK installation required.

## Test File Organization

### Go

```text
dephealth/
├── checker.go
├── checker_test.go          # tests next to source
├── checks/
│   ├── http.go
│   └── http_test.go
```

### Java

```text
dephealth-core/
├── src/main/java/.../HttpChecker.java
└── src/test/java/.../HttpCheckerTest.java    # mirror structure
```

### Python

```text
tests/
├── conftest.py              # shared fixtures
├── test_checker.py
├── test_parser.py
└── checks/
    ├── test_http.py
    └── test_postgres.py
```

### C\#

```text
DepHealth.Core.Tests/
├── HttpCheckerTests.cs      # separate test project
├── ConnectionParserTests.cs
└── Checks/
    └── ...
```
