# dephealth

SDK for monitoring microservice dependencies via Prometheus metrics.

## Features

- Automatic health checking for dependencies (PostgreSQL, MySQL, Redis, RabbitMQ, Kafka, HTTP, gRPC, TCP)
- Prometheus metrics export: `app_dependency_health` (Gauge 0/1), `app_dependency_latency_seconds` (Histogram), `app_dependency_status` (enum), `app_dependency_status_detail` (info)
- Async architecture built on `asyncio`
- FastAPI integration (middleware, lifespan, endpoints)
- Connection pool support (preferred) and standalone checks

## Installation

```bash
# Basic installation
pip install dephealth

# With specific checkers
pip install dephealth[postgres,redis]

# All checkers + FastAPI
pip install dephealth[all]
```

## Quick Start

### Standalone

```python
from dephealth import DepHealth

dh = DepHealth()
dh.add("postgres", url="postgresql://user:pass@localhost:5432/mydb")
dh.add("redis", url="redis://localhost:6379")

await dh.start()
# Metrics are available via prometheus_client
await dh.stop()
```

### FastAPI

```python
from fastapi import FastAPI
from dephealth_fastapi import DepHealthFastAPI

app = FastAPI()
dh = DepHealthFastAPI(app)
dh.add("postgres", url="postgresql://user:pass@localhost:5432/mydb")
```

## Dynamic Endpoints

Add, remove, or replace monitored endpoints at runtime on a running instance
(v0.6.0+):

```python
from dephealth import DependencyType, Endpoint
from dephealth.checks.http import HTTPChecker

# After dh.start()...

# Add a new endpoint
await dh.add_endpoint(
    "api-backend",
    DependencyType.HTTP,
    True,
    Endpoint(host="backend-2.svc", port="8080"),
    HTTPChecker(),
)

# Remove an endpoint (cancels check task, deletes metrics)
await dh.remove_endpoint("api-backend", "backend-2.svc", "8080")

# Replace an endpoint atomically
await dh.update_endpoint(
    "api-backend",
    "backend-1.svc", "8080",
    Endpoint(host="backend-3.svc", port="8080"),
    HTTPChecker(),
)
```

Synchronous variants are available: `add_endpoint_sync()`,
`remove_endpoint_sync()`, `update_endpoint_sync()`.

See [migration guide](../docs/migration/sdk-python-v050-to-v060.md) for details.

## Health Details

```python
details = dh.health_details()
for key, ep in details.items():
    print(f"{key}: healthy={ep.healthy} status={ep.status} "
          f"latency={ep.latency_millis():.1f}ms")
```

## Configuration

| Parameter | Default | Description |
| --- | --- | --- |
| `interval` | `15` | Check interval (seconds) |
| `timeout` | `5` | Check timeout (seconds) |

## Supported Dependencies

| Type | Extra | URL Format |
| --- | --- | --- |
| PostgreSQL | `postgres` | `postgresql://user:pass@host:5432/db` |
| MySQL | `mysql` | `mysql://user:pass@host:3306/db` |
| Redis | `redis` | `redis://host:6379` |
| RabbitMQ | `amqp` | `amqp://user:pass@host:5672/vhost` |
| Kafka | `kafka` | `kafka://host1:9092,host2:9092` |
| HTTP | — | `http://host:8080/health` |
| gRPC | `grpc` | `host:50051` (via `FromParams`) |
| TCP | — | `tcp://host:port` |

## Authentication

HTTP and gRPC checkers support Bearer token, Basic Auth, and custom headers/metadata:

```python
http_check("secure-api",
    url="http://api.svc:8080",
    critical=True,
    bearer_token="eyJhbG...",
)

grpc_check("grpc-backend",
    host="backend.svc",
    port=9090,
    critical=True,
    bearer_token="eyJhbG...",
)
```

See [quickstart guide](../docs/quickstart/python.md#authentication) for all options.

## License

Apache License 2.0 — see [LICENSE](https://github.com/BigKAA/topologymetrics/blob/master/LICENSE).
