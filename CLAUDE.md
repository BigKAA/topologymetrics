# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**dephealth** — SDK для мониторинга зависимостей микросервисов. Каждый сервис экспортирует Prometheus-метрики о состоянии своих зависимостей (БД, кэши, очереди, HTTP/gRPC-сервисы). VictoriaMetrics собирает данные, Grafana визуализирует.

## Language and Communication

- Общение — **на русском языке**.
- Комментарии, commit-сообщения — **на английском языке**.
- Документация - на английском и на русском языке. Отдельные файлы для каждого языка.
- Код (имена переменных, функций, классов) — на английском.

## Repository Structure

```text
proposal.md          # Бизнес-предложение: проблема, решение, план внедрения
sdk-architecture.md  # Техническая архитектура SDK (все 4 языка)
AGENTS.md            # Правила для AI-агентов
GIT-WORKFLOW.md      # Git workflow (GitHub Flow + Conventional Commits + Semver Tags)
```

Целевая структура монорепо (из sdk-architecture.md):

- `spec/` — единая спецификация метрик, поведения, конфигурации
- `conformance/` — conformance-тесты (Docker + YAML-сценарии)
- `sdk-java/`, `sdk-go/`, `sdk-csharp/`, `sdk-python/` — нативные SDK
- `docs/` — документация (quickstart, migration guides)

## Architecture Essentials

**Подход**: нативная библиотека на каждом языке, объединённая общей спецификацией. Не sidecar, не FFI.

**Ключевые метрики**:

- `app_dependency_health` (Gauge, 0/1) — с метками `dependency`, `type`, `host`, `port`
- `app_dependency_latency_seconds` (Histogram) — те же метки

**Слои SDK** (одинаковы для всех языков):

1. Core Abstractions — `Dependency`, `Endpoint`, `HealthChecker` interface
2. Connection Config Parser — парсинг URL, host/port, connection string
3. Health Checkers — HTTP, gRPC, TCP, Postgres, MySQL, Redis, AMQP, Kafka
4. Check Scheduler — периодический запуск проверок (default 15s)
5. Metrics Exporter — Prometheus gauges и histograms
6. Framework Integration — Spring Boot / ASP.NET / FastAPI / Django

**Два режима проверки**: автономный (новое соединение) и интеграция с connection pool сервиса (предпочтительный).

## Development Environment

- Разработка, отладка и тестирование — **только через Docker/Kubernetes**
- Тестовый k8s-кластер (amd64): Gateway API (без Ingress), MetalLB, cert-manager (ClusterIssuer: `dev-ca-issuer`)
- Container registry: `harbor.kryukov.lan` (admin/password), проект `library` (собственные образы), проект `homelab` (прокси-кэш Docker Hub)
- Тестовые домены: `test1.kryukov.lan`, `test2.kryukov.lan` → 192.168.218.180 (Gateway API)
- Доменные имена для тестов — добавлять в hosts (просить пользователя)
- Доступные инструменты: `kubectl`, `helm`, `docker`
- docker — сборка контейнеров **только для linux/amd64** (`--platform linux/amd64`)

## Git Workflow

GitHub Flow + Conventional Commits + Semver Tags. Подробности в `GIT-WORKFLOW.md`.

- Основная ветка: `master` (всегда deployable)
- Ветки: `feature/`, `bugfix/`, `docs/`, `refactor/`, `test/`, `hotfix/`
- Commit формат: `<type>(<scope>): <subject>` (feat, fix, docs, style, refactor, test, chore)
- Быстрые правки (опечатки) можно коммитить напрямую в `master`
- **Перед commit спросить пользователя**, перед merge предложить варианты (локальный merge / GitHub PR)
- После merge — удалить временную ветку

### Версионирование и релизы (ВАЖНО)

- **Каждый SDK версионируется независимо** — теги per-SDK: `sdk-go/vX.Y.Z`, `sdk-java/vX.Y.Z`, `sdk-python/vX.Y.Z`, `sdk-csharp/vX.Y.Z`
- **НЕ создавать общие теги** вида `vX.Y.Z` — только per-SDK теги
- Go **требует** формат `sdk-go/vX.Y.Z` для работы `go get` с модулем в поддиректории
- Semver применяется к каждому SDK отдельно: breaking change в Go SDK не бампит Java/Python/C#
- GitHub Release создаётся per-SDK (один Release = один тег)

## Linting

- Markdown: markdownlint с отключённым `MD013` (line length). Конфиг в `.markdownlint.json`
- Все файлы для языков программирования и md проверять соответствующим linter

## Plans

- Планы разработки хранить в `plans/`
- План должен быть подробным, разбит на фазы
- Одна фаза должна помещаться в один контекст AI
- Выполненные фазы отмечать как завершённые в файле плана
