*[Русская версия](ingress-gateway.ru.md)*

# Health Checks Through Ingress / Gateway API

When a service connects to a dependency **by IP address** through an ingress
controller or Gateway API, the dependency may use **Host-based routing** to
direct traffic. In this case, a plain request to the IP address fails (404 or
connection rejected) because the ingress does not know which backend to route
the request to.

dephealth provides two options to solve this:

| Protocol | Option | Effect |
| --- | --- | --- |
| HTTP | `hostHeader` | Sets the `Host` header + TLS SNI |
| gRPC | `grpcAuthority` | Sets the `:authority` pseudo-header + TLS SNI |

Both options also set **TLS SNI** (`ServerName`) when TLS is enabled, so that
TLS handshake succeeds when connecting by IP with a domain-based certificate.

> The `host` label in Prometheus metrics always reflects the **real endpoint
> address** (IP or hostname), not the value of `hostHeader` / `grpcAuthority`.

## Typical Architecture

```text
Service ──► 192.168.218.180 (Gateway VIP)
              │
              ├─ Host: api.example.com  ──►  api-backend
              ├─ Host: auth.example.com ──►  auth-backend
              └─ Host: other.com        ──►  404
```

The service connects to the gateway VIP by IP. Without the `Host` header,
the gateway returns 404. With `hostHeader: "api.example.com"`, the gateway
routes the request to the correct backend.

## Go

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"
)

dh, err := dephealth.New("my-service", "my-team",
    // HTTP through gateway by IP
    dephealth.HTTP("payment-api",
        dephealth.FromParams("192.168.218.180", "443"),
        dephealth.Critical(true),
        dephealth.WithHTTPTLS(true),
        dephealth.WithHTTPTLSSkipVerify(true),
        dephealth.WithHTTPHostHeader("payment.example.com"),
        dephealth.WithHTTPHealthPath("/healthz"),
    ),

    // gRPC through gateway by IP
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
    // HTTP through gateway by IP
    .dependency("payment-api", DependencyType.HTTP, d -> d
        .host("192.168.218.180")
        .port("443")
        .critical(true)
        .httpTls(true)
        .httpTlsSkipVerify(true)
        .httpHostHeader("payment.example.com")
        .httpHealthPath("/healthz"))

    // gRPC through gateway by IP
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
    // HTTP through gateway by IP
    .AddHttp(
        name: "payment-api",
        url: "https://192.168.218.180:443",
        healthPath: "/healthz",
        critical: true,
        httpHostHeader: "payment.example.com")

    // gRPC through gateway by IP
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

# HTTP through gateway by IP
http_check("payment-api",
    host="192.168.218.180",
    port="443",
    tls=True,
    tls_skip_verify=True,
    http_host_header="payment.example.com",
    health_path="/healthz",
    critical=True,
)

# gRPC through gateway by IP
grpc_check("user-service",
    host="192.168.218.180",
    port="443",
    tls=True,
    tls_skip_verify=True,
    grpc_authority="user.example.com",
    critical=True,
)
```

## Environment Variables

The same configuration can be provided via environment variables:

```bash
# HTTP host header override
export DEPHEALTH_PAYMENT_API_HOST_HEADER=payment.example.com

# gRPC authority override
export DEPHEALTH_USER_SERVICE_GRPC_AUTHORITY=user.example.com
```

## Important Notes

- **Metric labels are not affected** — the `host` label always shows the real
  endpoint address (`192.168.218.180`), not the virtual host name
- **TLS SNI is set automatically** — when TLS is enabled and `hostHeader` /
  `grpcAuthority` is set, TLS SNI (`ServerName`) is set to the same value
- **Works with `tlsSkipVerify`** — even when certificate verification is
  skipped, SNI is still sent (the gateway may need it for routing)
- **Combines with authentication** — `hostHeader` / `grpcAuthority` can be
  used together with `bearerToken`, `basicAuth`, custom `headers` / `metadata`
- **Conflict validation** — setting `hostHeader` and `Host` in custom headers
  simultaneously causes a validation error; same for `grpcAuthority` and
  `:authority` in metadata

## See Also

- [Go SDK — Checkers](../sdk-go/docs/checkers.md)
- [Java SDK — Checkers](../sdk-java/docs/checkers.md)
- [C# SDK — Checkers](../sdk-csharp/docs/checkers.md)
- [Python SDK — Checkers](../sdk-python/docs/checkers.md)
