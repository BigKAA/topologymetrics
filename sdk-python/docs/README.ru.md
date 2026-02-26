*[English version](README.md)*

# Документация Python SDK

## Обзор

dephealth Python SDK обеспечивает мониторинг здоровья зависимостей для Python
микросервисов через метрики Prometheus. Поддерживает асинхронный и потоковый
режимы с первоклассной интеграцией FastAPI.

**Текущая версия:** 0.8.0 | **Python:** 3.11+ | **FastAPI:** 0.110+

## Документация

| Документ | Описание |
| --- | --- |
| [Начало работы](getting-started.ru.md) | Установка, базовая настройка и первый пример |
| [Справочник API](api-reference.ru.md) | Полный справочник всех публичных классов и функций |
| [Конфигурация](configuration.ru.md) | Все опции, значения по умолчанию и переменные окружения |
| [Чекеры](checkers.ru.md) | Все 9 встроенных чекеров с примерами |
| [Интеграция с FastAPI](fastapi.ru.md) | Lifespan, middleware и health endpoint |
| [Метрики Prometheus](metrics.ru.md) | Справочник метрик и примеры PromQL |
| [Аутентификация](authentication.ru.md) | Опции аутентификации для HTTP, gRPC, LDAP и БД чекеров |
| [Connection Pools](connection-pools.ru.md) | Интеграция с asyncpg, redis-py, aiomysql, ldap3 |
| [Решение проблем](troubleshooting.ru.md) | Частые проблемы и решения |
| [Руководство по миграции](migration.ru.md) | Инструкции по обновлению версий |
| [Стиль кода](code-style.ru.md) | Соглашения по стилю кода Python для этого проекта |
| [Примеры](examples/) | Полные рабочие примеры |

## Полезные ссылки

- [pdoc API docs](_build/) (генерируется через `make docs`)
- [Спецификация](../../spec/) — общие контракты метрик и поведения для всех SDK
- [Дашборды Grafana](../../docs/grafana-dashboards.ru.md) — настройка дашбордов
- [Правила алертинга](../../docs/alerting/alert-rules.ru.md) — настройка алертинга
