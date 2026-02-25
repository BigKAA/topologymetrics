# Documentation Restructure Plan

Переработка документации: перенос SDK-специфичной документации из `docs/` в
директории SDK, добавление автогенерации API-документации, устранение
дублирования, создание подробных примеров.

## Текущее состояние

### Проблемы

| # | Проблема | Детали |
|---|----------|--------|
| 1 | **Неравномерность документации** | Go SDK: 20 файлов в `sdk-go/docs/`, Python: 2 файла, Java: 0 файлов (только README), C#: 2 файла |
| 2 | **SDK-документация в общем `docs/`** | Quickstart, migration, code-style для каждого SDK хранятся в `docs/`, а не внутри SDK |
| 3 | **Дублирование** | `docs/quickstart/go.md` ≈ `sdk-go/docs/getting-started.md` (оба содержат установку и минимальный пример); `docs/migration/go.md` содержит version-specific migration информацию, которая специфична для Go SDK |
| 4 | **Нет автогенерации API-документации** | Javadoc настроен в pom.xml но аннотации не интегрированы; Python без Sphinx/pdoc; C# без DocFX; Go без формального godoc |
| 5 | **Нет примеров** | Нет `examples/` ни в одном SDK; примеры только inline в markdown |

### Текущая структура документации

```
docs/
├── quickstart/         # 8 файлов: go, python, java, csharp (en + ru)
├── migration/          # 16 файлов: общие миграции + per-SDK миграции
├── code-style/         # 12 файлов: overview + per-SDK стили + testing
├── alerting/           # 10 файлов: общая документация по алертингу
├── specification.md/ru # Ссылки на spec/
├── grafana-dashboards.md/ru
└── comparison.md/ru

sdk-go/docs/            # 20 файлов: полная документация
sdk-python/docs/        # 2 файла: только api-reference (en + ru)
sdk-java/               # 0 файлов docs/, только README.md
sdk-csharp/docs/        # 2 файла: только api-reference (en + ru)
```

### Целевая структура

```
docs/                           # Общая документация (не SDK-специфичная)
├── alerting/                   # Без изменений
├── grafana-dashboards.md/ru    # Без изменений
├── comparison.md/ru            # Без изменений
├── specification.md/ru         # Без изменений
├── migration/                  # Только общие cross-SDK миграции
│   ├── v042-to-v050.md/ru
│   ├── v050-to-v060.md/ru
│   ├── v060-to-v070.md/ru
│   └── v070-to-v080.md/ru
└── code-style/
    ├── overview.md/ru          # Общие принципы
    └── testing.md/ru           # Общие принципы тестирования

sdk-go/docs/                    # Полная документация Go SDK
├── README.md                   # Обзор + навигация по документации
├── getting-started.md/ru       # Уже есть
├── api-reference.md/ru         # Уже есть
├── configuration.md/ru         # Уже есть
├── checkers.md/ru              # Уже есть
├── custom-checkers.md/ru       # Уже есть
├── metrics.md/ru               # Уже есть
├── authentication.md/ru        # Уже есть
├── connection-pools.md/ru      # Уже есть
├── selective-imports.md/ru     # Уже есть
├── troubleshooting.md/ru       # Уже есть
├── migration.md/ru             # НОВЫЙ: объединение из docs/migration/go.md
├── code-style.md/ru            # ПЕРЕНОС из docs/code-style/go.md
└── examples/                   # НОВЫЙ: подробные примеры
    ├── basic-http/
    ├── postgres-pool/
    ├── multi-dependency/
    └── dynamic-endpoints/

sdk-java/docs/                  # Полная документация Java SDK (НОВОЕ)
├── README.md                   # Обзор + навигация
├── getting-started.md/ru       # НОВЫЙ: на основе README.md quick start
├── api-reference.md/ru         # НОВЫЙ: автогенерация из Javadoc
├── configuration.md/ru         # НОВЫЙ: конфигурация, connection strings
├── checkers.md/ru              # НОВЫЙ: все health checkers
├── spring-boot.md/ru           # НОВЫЙ: Spring Boot интеграция
├── metrics.md/ru               # НОВЫЙ: Prometheus метрики
├── authentication.md/ru        # НОВЫЙ: аутентификация, TLS
├── connection-pools.md/ru      # НОВЫЙ: connection pool интеграция
├── troubleshooting.md/ru       # НОВЫЙ: частые проблемы
├── migration.md/ru             # ПЕРЕНОС из docs/migration/java.md + sdk-java миграция
├── code-style.md/ru            # ПЕРЕНОС из docs/code-style/java.md
└── examples/                   # НОВЫЙ: подробные примеры
    ├── basic-spring-boot/
    ├── programmatic-api/
    ├── multi-dependency/
    └── dynamic-endpoints/

sdk-python/docs/                # Полная документация Python SDK (РАСШИРЕНИЕ)
├── README.md                   # Обзор + навигация
├── getting-started.md/ru       # НОВЫЙ
├── api-reference.md/ru         # Уже есть
├── configuration.md/ru         # НОВЫЙ
├── checkers.md/ru              # НОВЫЙ
├── fastapi.md/ru               # НОВЫЙ: FastAPI интеграция
├── metrics.md/ru               # НОВЫЙ
├── authentication.md/ru        # НОВЫЙ
├── connection-pools.md/ru      # НОВЫЙ
├── troubleshooting.md/ru       # НОВЫЙ
├── migration.md/ru             # ПЕРЕНОС из docs/migration/python.md
├── code-style.md/ru            # ПЕРЕНОС из docs/code-style/python.md
└── examples/                   # НОВЫЙ: подробные примеры
    ├── basic-fastapi/
    ├── async-checks/
    ├── multi-dependency/
    └── dynamic-endpoints/

sdk-csharp/docs/                # Полная документация C# SDK (РАСШИРЕНИЕ)
├── README.md                   # Обзор + навигация
├── getting-started.md/ru       # НОВЫЙ
├── api-reference.md/ru         # Уже есть
├── configuration.md/ru         # НОВЫЙ
├── checkers.md/ru              # НОВЫЙ
├── aspnetcore.md/ru            # НОВЫЙ: ASP.NET Core интеграция
├── entity-framework.md/ru      # НОВЫЙ: EF integration
├── metrics.md/ru               # НОВЫЙ
├── authentication.md/ru        # НОВЫЙ
├── connection-pools.md/ru      # НОВЫЙ
├── troubleshooting.md/ru       # НОВЫЙ
├── migration.md/ru             # ПЕРЕНОС из docs/migration/csharp.md
├── code-style.md/ru            # ПЕРЕНОС из docs/code-style/csharp.md
└── examples/                   # НОВЫЙ: подробные примеры
    ├── basic-aspnetcore/
    ├── entity-framework/
    ├── multi-dependency/
    └── dynamic-endpoints/
```

---

## Фаза 0: Подготовка

- [ ] 0.1 Создать ветку `docs/restructure-sdk-docs`
- [ ] 0.2 Проанализировать содержимое каждого SDK (публичные API, типы, функции) для понимания объёма документации
- [ ] 0.3 Составить матрицу соответствия: какой код какой документацией покрыт

## Фаза 1: Go SDK — эталонная документация

Go SDK уже имеет наиболее полную документацию. Используем его как эталон и дополняем.

- [x] 1.1 Перенести `docs/migration/go.md` / `go.ru.md` → `sdk-go/docs/migration.md` / `migration.ru.md`
  - Объединить с информацией из `docs/migration/v042-to-v050.md` (Go-специфичные части)
- [x] 1.2 Перенести `docs/code-style/go.md` / `go.ru.md` → `sdk-go/docs/code-style.md` / `code-style.ru.md`
- [x] 1.3 Удалить `docs/quickstart/go.md` / `go.ru.md` — дублирует `sdk-go/docs/getting-started.md`
  - Проверено: уникальный контент quickstart (env vars, auth, pool) уже покрыт в sdk-go/docs/
  - Удаление произойдёт в Фазе 5 (очистка общего docs/)
- [x] 1.4 Создать `sdk-go/docs/examples/` с подробными рабочими примерами:
  - `basic-http/main.go` — минимальный HTTP мониторинг
  - `postgres-pool/main.go` — интеграция с connection pool
  - `multi-dependency/main.go` — несколько зависимостей разных типов
  - `dynamic-endpoints/main.go` — динамическое управление endpoints
- [x] 1.5 Обновить навигацию в getting-started.md / getting-started.ru.md — ссылки на migration, code-style, examples
  - Примечание: отдельного `sdk-go/docs/README.md` не существует; навигация обновлена в getting-started
- [x] 1.6 Обновить перекрёстные ссылки в `sdk-go/README.md`
- [ ] 1.7 Создать `sdk-go/README.ru.md` — русская версия README

## Фаза 2: Java SDK — создание документации

Java SDK имеет минимальную документацию. Нужно создать полный набор.

### 2A: Javadoc аннотации

- [x] 2A.1 Добавить Javadoc аннотации во все публичные классы `dephealth-core`:
  - `DepHealth`, `DepHealthBuilder`, `Dependency`, `DependencyType`, `Endpoint`
  - `HealthChecker` interface и все реализации
  - `CheckScheduler`, `MetricsExporter`
  - Все `Option`-классы и конфигурация
- [x] 2A.2 Добавить Javadoc аннотации в `dephealth-spring-boot-starter`:
  - `DepHealthAutoConfiguration`
  - `DepHealthProperties`
  - Все `@Bean` методы
- [x] 2A.3 Проверить сборку Javadoc: `make docs` (или `mvn javadoc:javadoc`)
- [x] 2A.4 Добавить target `docs` в `sdk-java/Makefile` для генерации Javadoc

### 2B: Markdown документация

- [x] 2B.1 Создать `sdk-java/docs/README.md` — обзор и навигация по документации
- [x] 2B.2 Создать `sdk-java/docs/getting-started.md` / `.ru.md` — на основе README.md, но подробнее:
  - Prerequisites (Java 21, Maven)
  - Установка (Maven, Gradle)
  - Минимальный пример с пояснениями
  - Checker Registration
- [x] 2B.3 Создать `sdk-java/docs/api-reference.md` / `.ru.md`:
  - Все публичные классы и интерфейсы
  - Builder API
  - Configuration options
  - (Ссылка на автогенерированный Javadoc)
- [x] 2B.4 Создать `sdk-java/docs/configuration.md` / `.ru.md`:
  - Connection string форматы
  - URL парсинг
  - Environment variables
  - Spring Boot properties (`dephealth.*`)
- [x] 2B.5 Создать `sdk-java/docs/checkers.md` / `.ru.md`:
  - Все типы checkers: HTTP, gRPC, TCP, Postgres, MySQL, Redis, AMQP, Kafka, LDAP
  - Для каждого: описание, параметры, пример использования
- [x] 2B.6 Создать `sdk-java/docs/spring-boot.md` / `.ru.md`:
  - Auto-configuration
  - Actuator integration
  - Properties reference
  - Customization
- [x] 2B.7 Создать `sdk-java/docs/metrics.md` / `.ru.md`
- [x] 2B.8 Создать `sdk-java/docs/authentication.md` / `.ru.md`
- [x] 2B.9 Создать `sdk-java/docs/connection-pools.md` / `.ru.md`
- [x] 2B.10 Создать `sdk-java/docs/troubleshooting.md` / `.ru.md`
- [x] 2B.11 Перенести `docs/migration/java.md` + `sdk-java-v050-to-v060.md` → `sdk-java/docs/migration.md` / `.ru.md`
- [x] 2B.12 Перенести `docs/code-style/java.md` → `sdk-java/docs/code-style.md` / `.ru.md`

### 2C: Примеры

- [ ] 2C.1 Создать `sdk-java/docs/examples/basic-spring-boot/` — Spring Boot starter пример
- [ ] 2C.2 Создать `sdk-java/docs/examples/programmatic-api/` — программный API без Spring
- [ ] 2C.3 Создать `sdk-java/docs/examples/multi-dependency/` — несколько зависимостей
- [ ] 2C.4 Создать `sdk-java/docs/examples/dynamic-endpoints/` — динамические endpoints

### 2D: Обновление ссылок

- [x] 2D.1 Обновить `sdk-java/README.md` — ссылки на `docs/`
- [x] 2D.3 Создать `sdk-java/README.ru.md` — русская версия README
- [ ] 2D.2 Обновить `TODO.md` — отметить Javadoc как завершённый

## Фаза 3: Python SDK — создание документации

### 3A: Docstrings и автогенерация

- [ ] 3A.1 Проверить наличие docstrings во всех публичных классах/функциях
  - `DepHealth`, `DepHealthBuilder`
  - Все checkers
  - FastAPI интеграция
- [ ] 3A.2 Добавить конфигурацию pdoc или mkdocstrings в `sdk-python/`:
  - `mkdocs.yml` или `pdoc` конфиг для автогенерации API docs
- [ ] 3A.3 Добавить target `docs` в `sdk-python/Makefile`

### 3B: Markdown документация

- [ ] 3B.1 Создать `sdk-python/docs/README.md` — обзор и навигация
- [ ] 3B.2 Создать `sdk-python/docs/getting-started.md` / `.ru.md`
- [ ] 3B.3 Обновить существующий `sdk-python/docs/api-reference.md` — проверить полноту
- [ ] 3B.4 Создать `sdk-python/docs/configuration.md` / `.ru.md`
- [ ] 3B.5 Создать `sdk-python/docs/checkers.md` / `.ru.md`
- [ ] 3B.6 Создать `sdk-python/docs/fastapi.md` / `.ru.md`
- [ ] 3B.7 Создать `sdk-python/docs/metrics.md` / `.ru.md`
- [ ] 3B.8 Создать `sdk-python/docs/authentication.md` / `.ru.md`
- [ ] 3B.9 Создать `sdk-python/docs/connection-pools.md` / `.ru.md`
- [ ] 3B.10 Создать `sdk-python/docs/troubleshooting.md` / `.ru.md`
- [ ] 3B.11 Перенести `docs/migration/python.md` + `sdk-python-v050-to-v060.md` → `sdk-python/docs/migration.md` / `.ru.md`
- [ ] 3B.12 Перенести `docs/code-style/python.md` → `sdk-python/docs/code-style.md` / `.ru.md`

### 3C: Примеры

- [ ] 3C.1 Создать `sdk-python/docs/examples/basic-fastapi/` — FastAPI интеграция
- [ ] 3C.2 Создать `sdk-python/docs/examples/async-checks/` — асинхронные проверки
- [ ] 3C.3 Создать `sdk-python/docs/examples/multi-dependency/`
- [ ] 3C.4 Создать `sdk-python/docs/examples/dynamic-endpoints/`

### 3D: Обновление ссылок

- [ ] 3D.1 Обновить `sdk-python/README.md` (создать если нет) — ссылки на `docs/`
- [ ] 3D.2 Создать `sdk-python/README.ru.md` — русская версия README

## Фаза 4: C# SDK — создание документации

### 4A: XML-комментарии и автогенерация

- [ ] 4A.1 Проверить наличие XML-комментариев `///` во всех публичных типах
  - `DepHealthMonitor`, `DepHealthBuilder`
  - Все checkers, интерфейсы
  - ASP.NET Core и Entity Framework интеграции
- [ ] 4A.2 Включить генерацию XML-документации в `.csproj`:
  ```xml
  <GenerateDocumentationFile>true</GenerateDocumentationFile>
  ```
- [ ] 4A.3 Опционально: настроить DocFX для генерации HTML-документации
- [ ] 4A.4 Добавить target `docs` в `sdk-csharp/Makefile`

### 4B: Markdown документация

- [ ] 4B.1 Создать `sdk-csharp/docs/README.md` — обзор и навигация
- [ ] 4B.2 Создать `sdk-csharp/docs/getting-started.md` / `.ru.md`
- [ ] 4B.3 Обновить существующий `sdk-csharp/docs/api-reference.md` — проверить полноту
- [ ] 4B.4 Создать `sdk-csharp/docs/configuration.md` / `.ru.md`
- [ ] 4B.5 Создать `sdk-csharp/docs/checkers.md` / `.ru.md`
- [ ] 4B.6 Создать `sdk-csharp/docs/aspnetcore.md` / `.ru.md`
- [ ] 4B.7 Создать `sdk-csharp/docs/entity-framework.md` / `.ru.md`
- [ ] 4B.8 Создать `sdk-csharp/docs/metrics.md` / `.ru.md`
- [ ] 4B.9 Создать `sdk-csharp/docs/authentication.md` / `.ru.md`
- [ ] 4B.10 Создать `sdk-csharp/docs/connection-pools.md` / `.ru.md`
- [ ] 4B.11 Создать `sdk-csharp/docs/troubleshooting.md` / `.ru.md`
- [ ] 4B.12 Перенести `docs/migration/csharp.md` + `sdk-csharp-v050-to-v060.md` → `sdk-csharp/docs/migration.md` / `.ru.md`
- [ ] 4B.13 Перенести `docs/code-style/csharp.md` → `sdk-csharp/docs/code-style.md` / `.ru.md`

### 4C: Примеры

- [ ] 4C.1 Создать `sdk-csharp/docs/examples/basic-aspnetcore/` — ASP.NET Core
- [ ] 4C.2 Создать `sdk-csharp/docs/examples/entity-framework/` — EF интеграция
- [ ] 4C.3 Создать `sdk-csharp/docs/examples/multi-dependency/`
- [ ] 4C.4 Создать `sdk-csharp/docs/examples/dynamic-endpoints/`

### 4D: Обновление ссылок

- [ ] 4D.1 Обновить `sdk-csharp/README.md` (создать если нет) — ссылки на `docs/`
- [ ] 4D.2 Создать `sdk-csharp/README.ru.md` — русская версия README

## Фаза 5: Очистка общего `docs/`

После переноса SDK-специфичной документации.

- [ ] 5.1 Удалить `docs/quickstart/` (полностью) — заменено на `sdk-*/docs/getting-started.md`
- [ ] 5.2 Удалить SDK-специфичные файлы из `docs/migration/`:
  - `go.md`, `go.ru.md`
  - `java.md`, `java.ru.md`
  - `python.md`, `python.ru.md`
  - `csharp.md`, `csharp.ru.md`
  - `sdk-java-v050-to-v060.md`, `sdk-java-v050-to-v060.ru.md`
  - `sdk-python-v050-to-v060.md`, `sdk-python-v050-to-v060.ru.md`
  - `sdk-csharp-v050-to-v060.md`, `sdk-csharp-v050-to-v060.ru.md`
  - Оставить: общие cross-SDK миграции `v042-to-v050.md`, `v050-to-v060.md`, `v060-to-v070.md`, `v070-to-v080.md`
- [ ] 5.3 Удалить SDK-специфичные файлы из `docs/code-style/`:
  - `go.md`, `go.ru.md`
  - `java.md`, `java.ru.md`
  - `python.md`, `python.ru.md`
  - `csharp.md`, `csharp.ru.md`
  - Оставить: `overview.md`, `overview.ru.md`, `testing.md`, `testing.ru.md`
- [ ] 5.4 Обновить навигационные ссылки в `docs/` файлах
- [ ] 5.5 Обновить `README.md` (root) — ссылки на документацию SDK

## Фаза 6: Финализация

- [ ] 6.1 Обновить `README.md` и `README.ru.md` — секция документации с новой структурой
- [ ] 6.2 Обновить `sdk-architecture.md` — если есть ссылки на старую структуру
- [ ] 6.3 Проверить все ссылки во всех `.md` файлах (dead link check)
- [ ] 6.4 Запустить markdownlint на все новые и изменённые файлы
- [ ] 6.5 Обновить `TODO.md`
- [ ] 6.6 Создать коммиты (по одному на фазу или логическую группу)
- [ ] 6.7 Перенести этот файл в `plans/archive/`

---

## Принципы

1. **Go SDK — эталон**: структура документации Go SDK используется как шаблон для остальных SDK
2. **Билингвальность**: каждый документ в двух версиях — `.md` (English) и `.ru.md` (Русский)
3. **Автогенерация API docs**: использовать родные инструменты языков (Javadoc, pdoc/mkdocstrings, DocFX/XML docs, godoc)
4. **Примеры — рабочий код**: примеры должны компилироваться/запускаться, а не быть псевдокодом
5. **Минимум дублирования**: один источник истины для каждой темы; ссылки вместо копирования
6. **Перекрёстные ссылки**: для общих тем (метрики spec, alerting) — ссылки на `docs/` или `spec/`

## Оценка объёма

| Фаза | Новые файлы | Перенесённые файлы | Удалённые файлы |
|------|-------------|--------------------|-----------------|
| 1 (Go) | ~8 (examples) | 4 | 2 |
| 2 (Java) | ~28 (docs + examples) | 6 | — |
| 3 (Python) | ~24 (docs + examples) | 4 | — |
| 4 (C#) | ~28 (docs + examples) | 4 | — |
| 5 (Cleanup) | — | — | ~22 |
| 6 (Final) | — | — | — |
| **Итого** | **~88** | **18** | **~24** |
