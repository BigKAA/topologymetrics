*[Русская версия](README.ru.md)*

# C# SDK Documentation

## Overview

dephealth C# SDK provides dependency health monitoring for .NET
microservices via Prometheus metrics. It supports both programmatic API
and ASP.NET Core integration with Entity Framework support.

**Current version:** 0.8.0 | **.NET:** 8 LTS | **ASP.NET Core:** 8.x

## Documentation

| Document | Description |
| --- | --- |
| [Getting Started](getting-started.md) | Installation, basic setup, and first example |
| [API Reference](api-reference.md) | Complete reference of all public classes and interfaces |
| [Configuration](configuration.md) | All options, defaults, and environment variables |
| [Health Checkers](checkers.md) | All 9 built-in checkers with examples |
| [ASP.NET Core Integration](aspnetcore.md) | DI registration, hosted service, health endpoints |
| [Entity Framework Integration](entity-framework.md) | DbContext-based health checks |
| [Prometheus Metrics](metrics.md) | Metrics reference and PromQL examples |
| [Authentication](authentication.md) | Auth options for HTTP, gRPC, and database checkers |
| [Connection Pools](connection-pools.md) | Integration with NpgsqlDataSource, IConnectionMultiplexer, ILdapConnection |
| [Troubleshooting](troubleshooting.md) | Common issues and solutions |
| [Migration Guide](migration.md) | Version upgrade instructions |
| [Code Style](code-style.md) | C# code conventions for this project |
| [Examples](examples/) | Complete runnable examples |

## Quick Links

- [XML Documentation](_build/) (generated via `make docs`)
- [Specification](../../spec/) — cross-SDK metric contracts and behavior
- [Grafana Dashboards](../../docs/grafana-dashboards.md) — dashboard configuration
- [Alert Rules](../../docs/alerting/alert-rules.md) — alerting configuration
