# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains a **design document** (`metrics.md`, written in Russian) for a microservice dependency monitoring system. There is no application code, build system, or tests — only documentation.

The document specifies a system for 300 Kubernetes microservices using VictoriaMetrics + Grafana, covering:

- **Metric design**: `app_dependency_health` (Gauge, 0/1 per endpoint) and `app_dependency_latency_seconds` (Histogram) — 2 metric names total, ~6300 time series
- **ConfigMap-based dependency configuration** with YAML schema for declaring endpoints per dependency
- **PromQL recording rules** for aggregation (`service:dependency:health`, `service:health:avg`, etc.)
- **4 Grafana dashboards**: Status Grid, Service Detail, Node Graph (dependency map), Impact Analysis
- **Alerting rules** with Alertmanager inhibition for cascade suppression
- **Reference Go implementation** of the health-check library (`depcheck` package)
- **Rollout plan** across 4 phases

## Working With This Repository

- The primary content is `metrics.md` — treat it as a living design spec
- The document language is Russian
- There are no build commands, linters, or tests to run
- Git remote is on a private server (not GitHub)
