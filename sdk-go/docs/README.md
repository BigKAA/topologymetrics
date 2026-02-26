*[Русская версия](README.ru.md)*

# Go SDK Documentation

## Overview

dephealth Go SDK provides dependency health monitoring for Go
microservices via Prometheus metrics. It supports selective imports
for minimal binary size and custom checker implementations.

**Current version:** 0.8.0 | **Go:** 1.22+

## Documentation

| Document | Description |
| --- | --- |
| [Getting Started](getting-started.md) | Installation, basic setup, and first example |
| [API Reference](api-reference.md) | Complete reference of all public types and functions |
| [Configuration](configuration.md) | All options, defaults, and environment variables |
| [Health Checkers](checkers.md) | All 9 built-in checkers with examples |
| [Custom Checkers](custom-checkers.md) | Implementing custom health checkers |
| [Selective Imports](selective-imports.md) | Importing only needed checkers |
| [Prometheus Metrics](metrics.md) | Metrics reference and PromQL examples |
| [Authentication](authentication.md) | Auth options for HTTP, gRPC, and database checkers |
| [Connection Pools](connection-pools.md) | Integration with sql.DB, go-redis, pgxpool |
| [Troubleshooting](troubleshooting.md) | Common issues and solutions |
| [Migration Guide](migration.md) | Version upgrade instructions |
| [Code Style](code-style.md) | Go code conventions for this project |
| [Examples](examples/) | Complete runnable examples |

## Quick Links

- [pkg.go.dev](https://pkg.go.dev/github.com/BigKAA/topologymetrics/sdk-go/dephealth) — API documentation
- [Specification](../../spec/) — cross-SDK metric contracts and behavior
- [Grafana Dashboards](../../docs/grafana-dashboards.md) — dashboard configuration
- [Alert Rules](../../docs/alerting/alert-rules.md) — alerting configuration
