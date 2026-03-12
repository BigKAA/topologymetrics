*[English version](ingress-gateway.md)*

# Проверки через Ingress / Gateway API

Когда сервис подключается к зависимости **по IP-адресу** через ingress-контроллер
или Gateway API, зависимость может использовать **маршрутизацию по Host** для
направления трафика. В этом случае обычный запрос на IP-адрес завершится ошибкой
(404 или отказ соединения), поскольку ingress не знает, на какой бэкенд
направить запрос.

dephealth предоставляет две опции для решения этой задачи:

| Протокол | Опция | Эффект |
| --- | --- | --- |
| HTTP | `hostHeader` | Устанавливает заголовок `Host` + TLS SNI |
| gRPC | `grpcAuthority` | Устанавливает pseudo-header `:authority` + TLS SNI |

Обе опции также устанавливают **TLS SNI** (`ServerName`) при включённом TLS,
чтобы TLS handshake проходил успешно при подключении по IP с доменным
сертификатом.

> Метка `host` в метриках Prometheus всегда отражает **реальный адрес
> эндпоинта** (IP или hostname), а не значение `hostHeader` / `grpcAuthority`.

## Типичная архитектура

```text
Сервис ──► 192.168.218.180 (Gateway VIP)
              │
              ├─ Host: api.example.com  ──►  api-backend
              ├─ Host: auth.example.com ──►  auth-backend
              └─ Host: other.com        ──►  404
```

Сервис подключается к Gateway VIP по IP. Без заголовка `Host` gateway возвращает
404. С `hostHeader: "api.example.com"` gateway маршрутизирует запрос на нужный
бэкенд.

## Go

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"
)

dh, err := dephealth.New("my-service", "my-team",
    // HTTP через gateway по IP
    dephealth.HTTP("payment-api",
        dephealth.FromParams("192.168.218.180", "443"),
        dephealth.Critical(true),
        dephealth.WithHTTPTLS(true),
        dephealth.WithHTTPTLSSkipVerify(true),
        dephealth.WithHTTPHostHeader("payment.example.com"),
        dephealth.WithHTTPHealthPath("/healthz"),
    ),

    // gRPC через gateway по IP
    dephealth.GRPC("user-service",
        dephealth.FromParams("192.168.218.180", "443"),
        dephealth.Critical(true),
        dephealth.WithGRPCTLS(true),
        dephealth.WithGRPCTLSSkipVerify(true),
        dephealth.WithGRPCAuthority("user.example.com"),
    ),
)
```

## Java

```java
import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.DependencyType;

var dh = DepHealth.builder("my-service", "my-team", registry)
    // HTTP через gateway по IP
    .dependency("payment-api", DependencyType.HTTP, d -> d
        .host("192.168.218.180")
        .port("443")
        .critical(true)
        .httpTls(true)
        .httpTlsSkipVerify(true)
        .httpHostHeader("payment.example.com")
        .httpHealthPath("/healthz"))

    // gRPC через gateway по IP
    .dependency("user-service", DependencyType.GRPC, d -> d
        .host("192.168.218.180")
        .port("443")
        .critical(true)
        .grpcTls(true)
        .grpcAuthority("user.example.com"))
    .build();
```

## C\#

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // HTTP через gateway по IP
    .AddHttp(
        name: "payment-api",
        url: "https://192.168.218.180:443",
        healthPath: "/healthz",
        critical: true,
        httpHostHeader: "payment.example.com")

    // gRPC через gateway по IP
    .AddGrpc(
        name: "user-service",
        host: "192.168.218.180",
        port: "443",
        tlsEnabled: true,
        critical: true,
        grpcAuthority: "user.example.com")
    .Build();
```

## Python

```python
from dephealth.api import http_check, grpc_check

# HTTP через gateway по IP
http_check("payment-api",
    host="192.168.218.180",
    port="443",
    tls=True,
    tls_skip_verify=True,
    http_host_header="payment.example.com",
    health_path="/healthz",
    critical=True,
)

# gRPC через gateway по IP
grpc_check("user-service",
    host="192.168.218.180",
    port="443",
    tls=True,
    tls_skip_verify=True,
    grpc_authority="user.example.com",
    critical=True,
)
```

## Переменные окружения

Ту же конфигурацию можно задать через переменные окружения:

```bash
# HTTP host header override
export DEPHEALTH_PAYMENT_API_HOST_HEADER=payment.example.com

# gRPC authority override
export DEPHEALTH_USER_SERVICE_GRPC_AUTHORITY=user.example.com
```

## Важные замечания

- **Метки метрик не затрагиваются** — метка `host` всегда содержит реальный
  адрес эндпоинта (`192.168.218.180`), а не виртуальное имя хоста
- **TLS SNI устанавливается автоматически** — при включённом TLS и заданном
  `hostHeader` / `grpcAuthority` TLS SNI (`ServerName`) устанавливается в то же
  значение
- **Работает с `tlsSkipVerify`** — даже при отключённой проверке сертификата SNI
  всё равно отправляется (gateway может использовать его для маршрутизации)
- **Совместимо с аутентификацией** — `hostHeader` / `grpcAuthority` можно
  использовать вместе с `bearerToken`, `basicAuth`, пользовательскими
  `headers` / `metadata`
- **Валидация конфликтов** — одновременная установка `hostHeader` и `Host` в
  пользовательских заголовках вызывает ошибку валидации; аналогично для
  `grpcAuthority` и `:authority` в metadata

## См. также

- [Go SDK — Чекеры](../sdk-go/docs/checkers.ru.md)
- [Java SDK — Чекеры](../sdk-java/docs/checkers.ru.md)
- [C# SDK — Чекеры](../sdk-csharp/docs/checkers.ru.md)
- [Python SDK — Чекеры](../sdk-python/docs/checkers.ru.md)
