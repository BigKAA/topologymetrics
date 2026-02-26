*[Русская версия](README.ru.md)*

# Topology Metrics (dephealth)

SDK for monitoring microservice dependencies. Each service exports
Prometheus metrics about the health of its dependencies (databases, caches,
queues, HTTP/gRPC services). VictoriaMetrics collects data, Grafana visualizes.

**Supported languages**: Go, Python, Java, C#

## Problem

A system of hundreds of microservices faces three problems:

- **Slow root cause analysis** — when a failure occurs, it is unclear which service is the source
- **No dependency map** — nobody sees the full picture of connections between services
- **Cascading failures** — one service going down triggers a flood of alerts from dependents

## Solution

Each microservice exports metrics about the health of its connections:

```text
# Health: 1 = available, 0 = unavailable
app_dependency_health{name="order-service",group="orders",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes"} 1

# Check latency
app_dependency_latency_seconds_bucket{name="order-service",group="orders",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.01"} 42

# Status category (enum pattern — all 8 values always exported, exactly one = 1)
app_dependency_status{name="order-service",group="orders",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",status="ok"} 1
app_dependency_status{name="order-service",group="orders",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",status="timeout"} 0
# ... (6 more status values = 0)

# Detailed reason
app_dependency_status_detail{name="order-service",group="orders",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",detail="ok"} 1
```

From these metrics a dependency graph is automatically built, alerting is configured
with cascade suppression, and the degradation level of each service is displayed.

## Quick Start

### Go

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
)

dh, err := dephealth.New("order-service", "orders",
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
    dephealth.Redis("redis-cache",
        dephealth.FromURL(os.Getenv("REDIS_URL")),
        dephealth.Critical(false),
    ),
)
dh.Start(ctx)
defer dh.Stop()

http.Handle("/metrics", promhttp.Handler())
```

### Python (FastAPI)

```python
from dephealth.api import postgres_check, redis_check
from dephealth_fastapi import dephealth_lifespan, DepHealthMiddleware

app = FastAPI(
    lifespan=dephealth_lifespan("order-service", "orders",
        postgres_check("postgres-main", url=os.environ["DATABASE_URL"], critical=True),
        redis_check("redis-cache", url=os.environ["REDIS_URL"], critical=False),
    )
)
app.add_middleware(DepHealthMiddleware)
```

### Java (Spring Boot)

```yaml
# application.yml
dephealth:
  name: order-service
  group: orders
  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      critical: true
    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false
```

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.5.0</version>
</dependency>
```

### C# (ASP.NET Core)

```csharp
builder.Services.AddDepHealth("order-service", "orders", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url(builder.Configuration["DATABASE_URL"]!)
        .Critical(true))
    .AddDependency("redis-cache", DependencyType.Redis, d => d
        .Url(builder.Configuration["REDIS_URL"]!)
        .Critical(false))
);

app.UseDepHealth(); // /metrics + /health/dependencies
```

## Architecture

A native library for each language, unified by a common specification.
Not a sidecar, not FFI — deep integration with each language's ecosystem.

```text
┌─────────────────────────────────────────────┐
│         Framework Integration               │  Spring Boot / ASP.NET / FastAPI
├─────────────────────────────────────────────┤
│         Metrics Exporter                    │  Prometheus gauges + histograms
├─────────────────────────────────────────────┤
│         Check Scheduler                     │  Periodic health checks
├─────────────────────────────────────────────┤
│         Health Checkers                     │  HTTP, gRPC, TCP, Postgres, MySQL,
│                                             │  Redis, AMQP, Kafka
├─────────────────────────────────────────────┤
│         Connection Config Parser            │  URL / params / connection string
├─────────────────────────────────────────────┤
│         Core Abstractions                   │  Dependency, Endpoint, HealthChecker
└─────────────────────────────────────────────┘
```

### Two Check Modes

- **Standalone** — SDK creates a temporary connection for the check
- **Pool integration** — SDK uses the service's existing connection pool
  (preferred, reflects the service's actual ability to work with the dependency)

## Supported Check Types

| Type | Check Method |
| --- | --- |
| `http` | HTTP GET to `healthPath`, follows redirects, expects 2xx |
| `grpc` | gRPC Health Check Protocol (grpc.health.v1) |
| `tcp` | TCP connection establishment |
| `postgres` | `SELECT 1` via connection pool or new connection |
| `mysql` | `SELECT 1` via connection pool or new connection |
| `redis` | `PING` command |
| `amqp` | Broker connection check |
| `kafka` | Metadata request to broker |

## Repository Structure

```text
spec/                           # Unified specification (metric, behavior, config contracts)
conformance/                    # Conformance tests (Kubernetes, 8 scenarios × 4 languages)
sdk-go/                         # Go SDK
sdk-python/                     # Python SDK
sdk-java/                       # Java SDK (Maven multi-module)
sdk-csharp/                     # C# SDK (.NET 8)
test-services/                  # Test microservices for each language
deploy/                         # Monitoring: Grafana, Alertmanager, VictoriaMetrics
docs/                           # Documentation (migration, alerting, specification, code-style)
plans/                          # Development plans
```

## Specification

The single source of truth for all SDKs — the `spec/` directory:

- [`spec/metric-contract.md`](spec/metric-contract.md) — metric format, labels, values
- [`spec/check-behavior.md`](spec/check-behavior.md) — check lifecycle, thresholds, timeouts
- [`spec/config-contract.md`](spec/config-contract.md) — connection configuration formats

### Key Metrics

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | Availability: `1` / `0` |
| `app_dependency_latency_seconds` | Histogram | Check latency |
| `app_dependency_status` | Gauge | Status category (enum pattern): 8 values per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge | Detailed reason (info pattern): e.g. `http_503`, `auth_error` |

Required labels: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`.
Additional label: `status` (on `app_dependency_status`), `detail` (on `app_dependency_status_detail`).

### Default Parameters

| Parameter | Value |
| --- | --- |
| `checkInterval` | 15s |
| `timeout` | 5s |
| `failureThreshold` | 1 |
| `successThreshold` | 1 |

## Conformance Tests

Infrastructure for verifying SDK compliance with the specification (`conformance/`):

- Kubernetes manifests for dependencies (PostgreSQL, Redis, RabbitMQ, Kafka)
- Managed HTTP and gRPC stubs
- 14 test scenarios: basic-health, partial-failure, full-failure, recovery,
  latency, labels, timeout, initial-state, health-details, group-label,
  status-enum, status-detail, critical-flag, concurrent-checks
- All 4 SDKs pass 14/14 scenarios (56 tests total)

## Documentation

### SDK Documentation

- [Go SDK](sdk-go/docs/getting-started.md) — getting started, API reference, examples
- [Python SDK](sdk-python/docs/getting-started.md) — getting started, API reference, examples
- [Java SDK](sdk-java/docs/getting-started.md) — getting started, API reference, examples
- [C# SDK](sdk-csharp/docs/getting-started.md) — getting started, API reference, examples

### Alerting and Monitoring

- [Monitoring Stack Overview](docs/alerting/overview.md) — architecture, scraping, VictoriaMetrics/Prometheus
- [Alert Rules](docs/alerting/alert-rules.md) — 5 built-in rules with PromQL breakdown
- [Noise Reduction](docs/alerting/noise-reduction.md) — scenarios, inhibition, best practices
- [Alertmanager Configuration](docs/alerting/alertmanager.md) — routing, receivers, templates
- [Custom Rules](docs/alerting/custom-rules.md) — writing your own rules on top of dephealth metrics

### Code Style

- [General Principles](docs/code-style/overview.md) — cross-SDK conventions, architecture layers
- [Go](sdk-go/docs/code-style.md) | [Java](sdk-java/docs/code-style.md) | [Python](sdk-python/docs/code-style.md) | [C#](sdk-csharp/docs/code-style.md)
- [Testing](docs/code-style/testing.md) — test naming, AAA pattern, mocking, coverage

### Additional

- [SDK Comparison](docs/comparison.md) — all languages side-by-side
- [Specification Overview](docs/specification.md) — metric, behavior, config contracts
- [Grafana Dashboards](docs/grafana-dashboards.md) — 5 dashboards for dependency monitoring

## Development

Detailed developer guide — [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache License 2.0](LICENSE) - Copyright (c) 2026 Artur Kryukov
