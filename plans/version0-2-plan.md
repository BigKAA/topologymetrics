# План реализации dephealth v0.2.0

## Контекст

В v0.1 невозможно построить сквозной граф зависимостей микросервисов: нет метки,
идентифицирующей приложение-источник, а `dependency` не связывается с целевым сервисом.
v0.2 решает это, добавляя `name` (кто я) и обновляя семантику `dependency` (к кому
подключаюсь), а также `critical` (критичность соединения). Опциональные метки
`role`/`shard`/`vhost` заменяются произвольными `WithLabel(key, value)`.

Задача описана в `.tasks/version0-2.md`.

## Обзор фаз

| # | Фаза | Результат | Зависит от |
|---|------|-----------|------------|
| 1 | Спецификация v2.0-draft | `spec/metric-contract.md`, `spec/config-contract.md` | -- |
| 2 | Go SDK v0.2 | `sdk-go/dephealth/` + тесты | Фаза 1 |
| 3 | Java SDK v0.2 | `sdk-java/` + тесты | Фаза 1 |
| 4 | Python SDK v0.2 | `sdk-python/` + тесты | Фаза 1 |
| 5 | C# SDK v0.2 | `sdk-csharp/` + тесты | Фаза 1 |
| 6 | Conformance + тестовые приложения | conformance/, test-services/ | Фазы 2-5 |
| 7 | Документация, CHANGELOG, версии | docs/, CHANGELOG.md, версии в манифестах | Фаза 6 |

---

## Фаза 1: Обновление спецификации

**Цель**: обновить единый источник правды для всех SDK.

**Статус**: [x] Завершена

### `spec/metric-contract.md`

- Версия: `1.0-draft` -> `2.0-draft`
- **Секция 2.3** (обязательные метки): добавить `name` и `critical` в таблицу:
  - `name` -- уникальное имя приложения, формат `[a-z][a-z0-9-]*`, 1-63 символа
  - `critical` -- критичность зависимости, значения `yes`/`no`
  - Обновить описание `dependency`: для сервисов с SDK значение должно совпадать
    с `name` целевого сервиса
- **Секция 2.4** (опциональные метки): удалить таблицу `role`/`shard`/`vhost`,
  заменить описанием `WithLabel(key, value)` с валидацией `[a-zA-Z_][a-zA-Z0-9_]*`
  и запретом переопределения обязательных
- **Секция 4.2** (порядок меток): `name`, `dependency`, `type`, `host`, `port`,
  `critical`, произвольные в алфавитном порядке
- **Секции 4.1, 5, 6** (примеры): обновить все примеры с `name` и `critical`
- **Секция 7** (PromQL): добавить запросы с `name` и `critical`

### `spec/config-contract.md`

- Версия: `1.0-draft` -> `2.0-draft`
- **Секция 7.1**: обновить сигнатуру -- `name` первым обязательным параметром
- **Секция 7.3**: `WithMetadata` -> `WithLabel`, `Critical` обязателен без дефолта
- **Секция 7.5**: добавить ошибки: `missing name`, `missing critical`,
  `invalid label name`, `reserved label`
- **Секция 8**: добавить `DEPHEALTH_NAME`, `DEPHEALTH_<DEP>_CRITICAL` (`yes`/`no`),
  `DEPHEALTH_<DEP>_LABEL_<KEY>`

### Проверка

```bash
npx markdownlint-cli2 "spec/*.md"
```

---

## Фаза 2: Go SDK v0.2

**Цель**: пилотная реализация изменений в Go SDK.

**Статус**: [x] Завершена

### `sdk-go/dephealth/dependency.go`

- `Endpoint.Metadata` -> `Endpoint.Labels` (map[string]string)
- Добавить `ValidateLabelName(name)`, `ValidateLabels(labels)`, `reservedLabels` set
- Обновить `Dependency.Validate()` -- валидация Labels каждого endpoint

### `sdk-go/dephealth/metrics.go`

- `labelNames` -> `requiredLabelNames = []string{"name", "dependency", "type", "host",
  "port", "critical"}`
- Удалить `allowedOptionalLabels`, `WithOptionalLabels()`
- `NewMetricsExporter(instanceName string, opts ...)` -- принимает имя экземпляра
- `labels()` -- добавить `"name": m.instanceName`,
  `"critical": boolToYesNo(dep.Critical)`
- Добавить `WithCustomLabels(labels []string)` MetricsOption для произвольных меток

### `sdk-go/dephealth/dephealth.go`

- `New(opts ...Option)` -> `New(name string, opts ...Option)`
- Валидация `name`, чтение `DEPHEALTH_NAME` (API > env)
- Сбор custom label keys из всех endpoints -> `WithCustomLabels`

### `sdk-go/dephealth/options.go`

- `DependencyConfig.Critical` -- `bool` -> `*bool` (nil = не задано)
- `DependencyConfig.Labels map[string]string`
- `WithLabel(key, value string) DependencyOption` -- новая опция
- `buildDependency()` -- валидация: critical==nil -> ошибка; ValidateLabels;
  мерж labels в endpoints
- Env vars: `DEPHEALTH_<DEP>_CRITICAL`, `DEPHEALTH_<DEP>_LABEL_<KEY>`

### `sdk-go/dephealth/contrib/sqldb/sqldb.go`, `contrib/redispool/redispool.go`

- Без структурных изменений (принимают `...DependencyOption`)

### Тесты (обновить и добавить)

- `dependency_test.go` -- ValidateLabelName, ValidateLabels, reserved labels
- `metrics_test.go` -- instanceName, name/critical в метриках, custom labels, порядок
- `dephealth_test.go` -- `New("test-app", ...)`, MissingName, NameFromEnv,
  MissingCritical, WithLabel, ReservedLabel, CriticalFromEnv, LabelFromEnv
- `options_test.go` -- WithLabel, валидация
- `contrib/*/` -- обновить вызовы New

### Проверка

```bash
cd sdk-go && make test && make lint
```

---

## Фаза 3: Java SDK v0.2

**Цель**: реализация изменений в Java SDK.

**Статус**: [x] Завершена

### `sdk-java/dephealth-core/.../Endpoint.java`

- Добавить `labels()` (алиас для `metadata()`), пометить `metadata()` @Deprecated
- Статические методы валидации: `validateLabelName`, `validateLabels`, `RESERVED_LABELS`

### `sdk-java/dephealth-core/.../Dependency.java`

- `Builder.critical` -> `Boolean criticalExplicit` (null = не задан)
- Валидация: criticalExplicit==null -> ValidationException

### `sdk-java/dephealth-core/.../metrics/MetricsExporter.java`

- Конструктор: `MetricsExporter(MeterRegistry, String instanceName)`
- `buildTags()` -- добавить `"name"`, `"critical"` (yes/no), сортировка custom labels

### `sdk-java/dephealth-core/.../DepHealth.java`

- `builder(MeterRegistry)` -> `builder(String name, MeterRegistry)`
- Валидация name, `DEPHEALTH_NAME` env var
- `DependencyBuilder.label(key, value)` -- новый метод
- `DependencyBuilder.critical(boolean)` -- с `criticalSet` флагом
- Env vars: `DEPHEALTH_<DEP>_CRITICAL`, `DEPHEALTH_<DEP>_LABEL_<KEY>`

### `sdk-java/dephealth-spring-boot-starter/.../DepHealthProperties.java`

- Добавить `name` (String) на верхнем уровне
- `DependencyProperties` -- `Map<String, String> labels`, `Boolean critical` (nullable)

### `sdk-java/dephealth-spring-boot-starter/.../DepHealthAutoConfiguration.java`

- Передать `properties.getName()` в builder
- Передать labels в dependency configurer

### Тесты

- `MetricsExporterTest` -- instanceName, name/critical, custom labels
- `DependencyTest` -- critical обязателен, labels валидация
- `DepHealthTest` -- builder(name, registry), critical обязателен, WithLabel

### Проверка

```bash
cd sdk-java && make test && make lint
```

---

## Фаза 4: Python SDK v0.2

**Цель**: реализация изменений в Python SDK.

**Статус**: [x] Завершена

### `sdk-python/dephealth/dependency.py`

- `Endpoint.metadata` -> `Endpoint.labels` (dict[str, str])
- `RESERVED_LABELS`, `LABEL_NAME_PATTERN`, `validate_label_name()`, `validate_labels()`
- `Dependency.critical: bool` -- обязательный (без default),
  переместить перед `endpoints`

### `sdk-python/dephealth/metrics.py`

- `_LABEL_NAMES` -> `_REQUIRED_LABEL_NAMES = ("name", "dependency", "type", "host",
  "port", "critical")`
- `MetricsExporter.__init__(instance_name, custom_label_names=(), registry=None)`
- `_labels()` -- добавить `"name"`, `"critical"` (yes/no), custom labels

### `sdk-python/dephealth/api.py`

- `DependencyHealth(*specs, ...)` -> `DependencyHealth(name, *specs, ...)`
- Валидация name, `DEPHEALTH_NAME` env var
- Сбор custom label keys -> передать в MetricsExporter
- Все фабрики: `critical: bool` обязателен (убрать `=True`),
  `labels: dict[str, str] | None = None`
- Env vars: `DEPHEALTH_<DEP>_CRITICAL`, `DEPHEALTH_<DEP>_LABEL_<KEY>`

### `sdk-python/dephealth_fastapi/lifespan.py`

- `dephealth_lifespan(name, *specs, ...)` -- добавить name

### Тесты

- `test_dependency.py` -- critical обязателен, labels валидация
- `test_metrics.py` -- instance_name, name/critical, custom labels
- `test_api.py` -- DependencyHealth("test-app", ...), critical обязателен,
  WithLabel, env vars
- `test_fastapi.py` -- обновить с новым API

### Проверка

```bash
cd sdk-python && make test && make lint
```

---

## Фаза 5: C# SDK v0.2

**Цель**: реализация изменений в C# SDK.

**Статус**: [x] Завершена

### `sdk-csharp/DepHealth.Core/Endpoint.cs`

- Добавить `Labels` property (алиас для Metadata), валидация

### `sdk-csharp/DepHealth.Core/Dependency.cs`

- `Builder.CriticalValue` -> `bool?` (nullable),
  валидация null -> ValidationException

### `sdk-csharp/DepHealth.Core/PrometheusExporter.cs`

- Конструктор: `PrometheusExporter(string instanceName, string[]? customLabelNames,
  CollectorRegistry?)`
- LabelNames: `["name", "dependency", "type", "host", "port", "critical"] + custom`
- `BuildLabelValues()` -- добавить instanceName, critical (yes/no), custom labels

### `sdk-csharp/DepHealth.Core/DepHealth.cs`

- `CreateBuilder()` -> `CreateBuilder(string name)`
- Валидация name, `DEPHEALTH_NAME` env var
- `AddHttp/Postgres/...` -- `critical` обязательный (убрать `= false`),
  `labels` параметр
- Env vars: `DEPHEALTH_<DEP>_CRITICAL`, `DEPHEALTH_<DEP>_LABEL_<KEY>`

### `sdk-csharp/DepHealth.AspNetCore/ServiceCollectionExtensions.cs`

- `AddDepHealth(services, name, configure)` -- добавить name

### Тесты

- `PrometheusExporterTests` -- instanceName, name/critical, custom labels
- `DependencyTests` -- critical обязателен, labels валидация
- `DepHealthMonitorTests` -- CreateBuilder(name), critical обязателен
- `ServiceCollectionExtensionsTests` -- AddDepHealth с name

### Проверка

```bash
cd sdk-csharp && make test && make lint
```

---

## Фаза 6: Conformance-тесты и тестовые приложения

**Цель**: обновить conformance runner, сценарии и все тестовые/conformance-сервисы.

**Статус**: [x] Завершена

### Conformance runner

- `conformance/runner/verify.py` -- `REQUIRED_LABELS = {"name", "dependency", "type",
  "host", "port", "critical"}`, проверки: name формат, critical values (yes/no),
  custom labels
- `conformance/runner/cross_verify.py` -- обновить cross-verification

### Conformance сценарии (8 файлов)

Обновить все YAML-файлы в `conformance/scenarios/`:

- `basic-health.yml`, `labels.yml`, `latency.yml`, `partial-failure.yml`,
  `full-failure.yml`, `recovery.yml`, `timeout.yml`, `initial-state.yml`
- Добавить `name` и `critical` в expected_dependencies
- В `labels.yml` -- добавить проверку custom labels

### Conformance-сервисы (4 шт.)

- `conformance/test-service/main.go` -- `dephealth.New("conformance-service", ...)`,
  Critical обязателен
- `conformance/test-service-java/.../application.yml` --
  `dephealth.name: conformance-service`, critical для всех
- `conformance/test-service-python/main.py` --
  `DependencyHealth("conformance-service", ...)`, critical
- `conformance/test-service-csharp/Program.cs` --
  `CreateBuilder("conformance-service")`, critical

### Тестовые приложения (4 шт.)

- `test-services/go-service/main.go` -- `New("dephealth-test-go", ...)`,
  Critical для всех
- `test-services/java-service/.../application.yml` -- name и critical
- `test-services/python-service/main.py` --
  `DependencyHealth("dephealth-test-python", ...)`
- test-services/csharp-service/ -- если существует, аналогично

### Проверка

```bash
cd sdk-go && make test
cd sdk-java && make test
cd sdk-python && make test
cd sdk-csharp && make test
```

---

## Фаза 7: Документация, CHANGELOG, версии

**Цель**: обновить всю документацию, CHANGELOG, номера версий.

**Статус**: [x] Завершена

### Quickstart-гайды (4 файла)

- `docs/quickstart/go.md` -- `New("my-service", ...)`, Critical, WithLabel,
  примеры метрик
- `docs/quickstart/java.md` -- `builder("my-service", registry)`, critical, label()
- `docs/quickstart/python.md` -- `DependencyHealth("my-service", ...)`, critical, labels
- `docs/quickstart/csharp.md` -- `CreateBuilder("my-service")`, critical, labels

### Миграционные гайды (4 файла)

- `docs/migration/go.md` -- раздел "v0.1 -> v0.2": New() signature,
  Critical обязателен, Metadata -> Labels, WithLabel
- `docs/migration/java.md` -- аналогично
- `docs/migration/python.md` -- аналогично
- `docs/migration/csharp.md` -- аналогично

### Прочая документация

- `docs/specification.md` -- обновить обзор меток
- `docs/grafana-dashboards.md` -- обновить PromQL с name/critical

### CHANGELOG.md

- Добавить секцию `[0.2.0]` с breaking changes и новыми фичами

### Версии в манифестах

- `sdk-java/pom.xml` -- version: 0.1.0 -> 0.2.0
- `sdk-python/pyproject.toml` -- version: 0.1.0 -> 0.2.0
- `sdk-csharp/Directory.Build.props` -- version -> 0.2.0

### Проверка

```bash
npx markdownlint-cli2 "**/*.md"
```
