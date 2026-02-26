*[English version](README.md)*

# Документация Java SDK

## Обзор

dephealth Java SDK обеспечивает мониторинг здоровья зависимостей для Java
микросервисов через метрики Prometheus. Поддерживает как программный API,
так и auto-configuration для Spring Boot.

**Текущая версия:** 0.8.0 | **Java:** 21 LTS | **Spring Boot:** 3.x

## Документация

| Документ | Описание |
| --- | --- |
| [Начало работы](getting-started.ru.md) | Установка, базовая настройка и первый пример |
| [Справочник API](api-reference.ru.md) | Полный справочник всех публичных классов и интерфейсов |
| [Конфигурация](configuration.ru.md) | Все опции, значения по умолчанию и переменные окружения |
| [Чекеры](checkers.ru.md) | Все 9 встроенных чекеров с примерами |
| [Интеграция со Spring Boot](spring-boot.ru.md) | Auto-configuration, actuator, properties |
| [Метрики Prometheus](metrics.ru.md) | Справочник метрик и примеры PromQL |
| [Аутентификация](authentication.ru.md) | Опции аутентификации для HTTP, gRPC и БД чекеров |
| [Connection Pools](connection-pools.ru.md) | Интеграция с DataSource, JedisPool, LDAPConnection |
| [Решение проблем](troubleshooting.ru.md) | Частые проблемы и решения |
| [Руководство по миграции](migration.ru.md) | Инструкции по обновлению версий |
| [Стиль кода](code-style.ru.md) | Соглашения по стилю кода Java для этого проекта |
| [Примеры](examples/) | Полные рабочие примеры |

## Полезные ссылки

- [Javadoc](../dephealth-core/target/reports/apidocs/) (генерируется через `make docs`)
- [Спецификация](../../spec/) — общие контракты метрик и поведения для всех SDK
- [Дашборды Grafana](../../docs/grafana-dashboards.ru.md) — настройка дашбордов
- [Правила алертинга](../../docs/alerting/alert-rules.ru.md) — настройка алертинга
