*[English version](README.md)*

# Документация C# SDK

## Обзор

dephealth C# SDK обеспечивает мониторинг здоровья зависимостей для .NET
микросервисов через метрики Prometheus. Поддерживает как программный API,
так и интеграцию с ASP.NET Core и Entity Framework.

**Текущая версия:** 0.8.0 | **.NET:** 8 LTS | **ASP.NET Core:** 8.x

## Документация

| Документ | Описание |
| --- | --- |
| [Начало работы](getting-started.ru.md) | Установка, базовая настройка и первый пример |
| [Справочник API](api-reference.ru.md) | Полный справочник всех публичных классов и интерфейсов |
| [Конфигурация](configuration.ru.md) | Все опции, значения по умолчанию и переменные окружения |
| [Чекеры](checkers.ru.md) | Все 9 встроенных чекеров с примерами |
| [Интеграция с ASP.NET Core](aspnetcore.ru.md) | DI-регистрация, hosted service, health endpoints |
| [Интеграция с Entity Framework](entity-framework.ru.md) | Health check на основе DbContext |
| [Метрики Prometheus](metrics.ru.md) | Справочник метрик и примеры PromQL |
| [Аутентификация](authentication.ru.md) | Опции аутентификации для HTTP, gRPC и БД чекеров |
| [Connection Pools](connection-pools.ru.md) | Интеграция с NpgsqlDataSource, IConnectionMultiplexer, ILdapConnection |
| [Решение проблем](troubleshooting.ru.md) | Частые проблемы и решения |
| [Руководство по миграции](migration.ru.md) | Инструкции по обновлению версий |
| [Стиль кода](code-style.ru.md) | Соглашения по стилю кода C# для этого проекта |
| [Примеры](examples/) | Полные рабочие примеры |

## Полезные ссылки

- [XML-документация](_build/) (генерируется через `make docs`)
- [Спецификация](../../spec/) — общие контракты метрик и поведения для всех SDK
- [Дашборды Grafana](../../docs/grafana-dashboards.ru.md) — настройка дашбордов
- [Правила алертинга](../../docs/alerting/alert-rules.ru.md) — настройка алертинга
