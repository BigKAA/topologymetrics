*[Русская версия](README.ru.md)*

# Python SDK Documentation

## Overview

dephealth Python SDK provides dependency health monitoring for Python
microservices via Prometheus metrics. It supports async and threading modes,
with first-class FastAPI integration.

**Current version:** 0.8.0 | **Python:** 3.11+ | **FastAPI:** 0.110+

## Documentation

| Document | Description |
| --- | --- |
| [Getting Started](getting-started.md) | Installation, basic setup, and first example |
| [API Reference](api-reference.md) | Complete reference of all public classes and functions |
| [Configuration](configuration.md) | All options, defaults, and environment variables |
| [Health Checkers](checkers.md) | All 9 built-in checkers with examples |
| [FastAPI Integration](fastapi.md) | Lifespan, middleware, and health endpoint |
| [Prometheus Metrics](metrics.md) | Metrics reference and PromQL examples |
| [Authentication](authentication.md) | Auth options for HTTP, gRPC, LDAP, and database checkers |
| [Connection Pools](connection-pools.md) | Integration with asyncpg, redis-py, aiomysql, ldap3 |
| [Troubleshooting](troubleshooting.md) | Common issues and solutions |
| [Migration Guide](migration.md) | Version upgrade instructions |
| [Code Style](code-style.md) | Python code conventions for this project |
| [Examples](examples/) | Complete runnable examples |

## Quick Links

- [pdoc API docs](_build/) (generated via `make docs`)
- [Specification](../../spec/) — cross-SDK metric contracts and behavior
- [Grafana Dashboards](../../docs/grafana-dashboards.md) — dashboard configuration
- [Alert Rules](../../docs/alerting/alert-rules.md) — alerting configuration
