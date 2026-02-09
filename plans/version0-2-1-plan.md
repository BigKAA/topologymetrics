# План релиза v0.2.1 — HTTP redirect fix

## Описание

Исправление поведения HTTP-чекеров: Go и Java SDK теперь следуют HTTP-редиректам (3xx),
как Python и C# SDK. Обновлена спецификация и документация. Обновлены скриншоты Grafana дашбордов.

## Предусловия

- Изменения уже внесены и протестированы (Go: 4/4 OK, Java: 167 OK, markdownlint: 0 ошибок)
- 4 файла изменены, но НЕ закоммичены
- Доступ к k8s-кластеру (`kubectl`, `helm`)
- Harbor registry: `harbor.kryukov.lan`
- Grafana: `http://grafana.kryukov.lan` (admin / dephealth)

---

## Фаза 0: Запуск тестовой среды и мониторинга ⬜

> **Цель**: запустить инфраструктуру, тестовые сервисы и мониторинг как можно раньше,
> чтобы к моменту создания скриншотов (фаза 6) метрики накопились за 15+ минут.

### Задачи

1. **Проверить текущее состояние кластера**:

   ```bash
   kubectl get ns
   kubectl get pods -n dephealth-test
   kubectl get pods -n dephealth-monitoring
   ```

2. **Развернуть инфраструктуру зависимостей** (если не запущена):

   ```bash
   helm upgrade --install dephealth-infra deploy/helm/dephealth-infra/ \
     -f deploy/helm/dephealth-infra/values-homelab.yaml \
     -n dephealth-test --create-namespace
   ```

   Компоненты: PostgreSQL (primary + replica), Redis, Kafka, RabbitMQ, HTTP stub, gRPC stub.

3. **Развернуть тестовые сервисы** (если не запущены):

   ```bash
   helm upgrade --install dephealth-services deploy/helm/dephealth-services/ \
     -f deploy/helm/dephealth-services/values-homelab.yaml \
     -n dephealth-test
   ```

   Сервисы: go-service, python-service, java-service, csharp-service
   (каждый экспортирует метрики `app_dependency_health` и `app_dependency_latency_seconds`).

4. **Развернуть мониторинг** (если не запущен):

   ```bash
   helm upgrade --install dephealth-monitoring deploy/helm/dephealth-monitoring/ \
     -f deploy/helm/dephealth-monitoring/values-homelab.yaml \
     -n dephealth-monitoring --create-namespace
   ```

   Компоненты: VictoriaMetrics, Grafana, VMAlert, Alertmanager.

5. **Дождаться готовности всех подов**:

   ```bash
   kubectl wait --for=condition=Ready pod --all -n dephealth-test --timeout=120s
   kubectl wait --for=condition=Ready pod --all -n dephealth-monitoring --timeout=120s
   ```

6. **Проверить доступность Grafana**: открыть `http://grafana.kryukov.lan`

### Зависимости

- Нет (запускается первой)

### Валидация

- Все поды в `dephealth-test` и `dephealth-monitoring` в статусе Running
- Grafana доступна по URL, дашборды загружены
- VictoriaMetrics получает метрики от 4 тестовых сервисов

---

## Фаза 1: Коммит и пуш HTTP redirect fix ⬜

### Задачи

1. Проверить текущий `git diff` — убедиться, что изменения корректны
2. Добавить в staging 4 изменённых файла:
   - `sdk-go/dephealth/checks/http.go`
   - `sdk-java/dephealth-core/src/main/java/biz/kryukov/dev/dephealth/checks/HttpHealthChecker.java`
   - `spec/check-behavior.md`
   - `README.md`
3. Коммит: `fix(http): следовать HTTP-редиректам в Go и Java SDK`
4. Пуш в `master`

### Зависимости

- Нет (параллельно с фазой 0)

### Валидация

- `git log -1` показывает коммит
- `git status` — чистое рабочее дерево

---

## Фаза 2: Пересборка контейнеров ⬜

### 2.1 Conformance test-services

Пересобрать образы conformance-сервисов с обновлённым кодом SDK:

| Сервис | Dockerfile | Образ |
|--------|-----------|-------|
| Go | `conformance/test-service/Dockerfile` | `harbor.kryukov.lan/library/dephealth-conformance-go` |
| Java | `conformance/test-service-java/Dockerfile` | `harbor.kryukov.lan/library/dephealth-conformance-java` |
| Python | `conformance/test-service-python/Dockerfile` | `harbor.kryukov.lan/library/dephealth-conformance-python` |
| C# | `conformance/test-service-csharp/Dockerfile` | `harbor.kryukov.lan/library/dephealth-conformance-csharp` |

### 2.2 Test-services (Kubernetes)

Пересобрать образы тестовых сервисов:

| Сервис | Dockerfile | Образ |
|--------|-----------|-------|
| Go | `test-services/go-service/Dockerfile` | `harbor.kryukov.lan/library/dephealth-go-service` |
| Java | `test-services/java-service/Dockerfile` | `harbor.kryukov.lan/library/dephealth-java-service` |
| Python | `test-services/python-service/Dockerfile` | `harbor.kryukov.lan/library/dephealth-python-service` |
| C# | `test-services/csharp-service/Dockerfile` | `harbor.kryukov.lan/library/dephealth-csharp-service` |

### Задачи

1. Уточнить у пользователя:
   - Точные имена образов и тегов
   - Нужно ли пересобирать ВСЕ образы или только Go и Java (затронутые изменениями)
   - Нужно ли пушить в Harbor
2. Собрать образы через `docker build`
3. Запушить в Harbor через `docker push`
4. **Перезапустить тестовые сервисы** в кластере (чтобы использовали новые образы):

   ```bash
   kubectl rollout restart deployment -n dephealth-test
   kubectl wait --for=condition=Ready pod --all -n dephealth-test --timeout=120s
   ```

### Зависимости

- Фаза 1 (код должен быть актуален)

### Валидация

- `docker images | grep dephealth` — новые образы присутствуют
- Тестовые сервисы в кластере перезапущены с новыми образами

---

## Фаза 3: Проверка в кластере ⬜

### Задачи

1. Уточнить у пользователя:
   - Namespace для деплоя (`dephealth-conformance` по умолчанию)
   - Нужно ли переустанавливать Helm chart
2. Обновить деплой в кластере:
   - `helm upgrade` для `dephealth-conformance` chart
   - Дождаться готовности подов
3. Запустить conformance-тесты:
   - `./conformance/run.sh` с нужными параметрами
   - Проверить результат: все 8 сценариев x 4 SDK = 32/32

### Зависимости

- Фаза 2 (образы должны быть пересобраны и запушены)

### Валидация

- Все conformance-тесты проходят (32/32)
- `kubectl get pods -n dephealth-conformance` — все поды Running

---

## Фаза 4: Поднять версии → 0.2.1 ⬜

### Файлы для обновления

| Файл | Строка | Текущее | Новое |
|------|--------|---------|-------|
| `sdk-java/pom.xml` | 9 | `<version>0.2.0</version>` | `<version>0.2.1</version>` |
| `sdk-java/dephealth-core/pom.xml` | 10 | `<version>0.2.0</version>` | `<version>0.2.1</version>` |
| `sdk-java/dephealth-spring-boot-starter/pom.xml` | 10 | `<version>0.2.0</version>` | `<version>0.2.1</version>` |
| `sdk-python/pyproject.toml` | 10 | `version = "0.2.0"` | `version = "0.2.1"` |
| `sdk-csharp/Directory.Build.props` | 10 | `<Version>0.2.0</Version>` | `<Version>0.2.1</Version>` |
| `sdk-go/dephealth/checks/doc.go` | 18 | `Version = "0.1.0"` | `Version = "0.2.1"` |

> **Внимание**: Go SDK содержит константу `Version = "0.1.0"` — она не была обновлена
> при v0.2.0 и должна быть исправлена на 0.2.1.

### Задачи

1. Обновить все файлы версий (таблица выше)
2. Запустить тесты для каждого SDK:
   - `cd sdk-java && make test`
   - `cd sdk-python && make test`
   - `cd sdk-go && make test`
   - `cd sdk-csharp && make test`
3. Запустить линтеры:
   - `cd sdk-java && make lint`
   - `cd sdk-go && make lint`
4. Коммит: `chore: поднять версии до 0.2.1`

### Зависимости

- Фаза 3 (conformance-тесты должны пройти)

### Валидация

- Все тесты проходят
- `grep -r "0.2.1"` показывает обновлённые версии
- `git diff --stat` показывает только файлы версий

---

## Фаза 5: Обновить CHANGELOG.md ⬜

### Содержимое секции [0.2.1]

```markdown
## [0.2.1] - YYYY-MM-DD

### Fixed

- **Go SDK**: HTTP-чекер теперь следует HTTP-редиректам (3xx) вместо ошибки
- **Java SDK**: HTTP-чекер теперь следует HTTP-редиректам (3xx) вместо ошибки
- **Java SDK**: обновлён User-Agent с 0.1.0 до 0.2.0
- **Спецификация**: обновлена таблица edge cases — редиректы следуются, ожидается финальный 2xx

### Changed

- **Документация**: обновлены скриншоты Grafana дашбордов
```

### Задачи

1. Добавить секцию `[0.2.1]` перед `[0.2.0]` в `CHANGELOG.md`
2. Добавить ссылку на релиз внизу файла:
   `[0.2.1]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.2.1`
3. Запустить markdownlint: `npx markdownlint-cli2 CHANGELOG.md`
4. Коммит: `docs: CHANGELOG v0.2.1`

### Зависимости

- Фаза 4 (версии должны быть обновлены)

### Валидация

- Markdownlint: 0 ошибок
- Секция [0.2.1] присутствует с корректной датой и содержимым

---

## Фаза 6: Обновить скриншоты Grafana дашбордов ⬜

> **Цель**: сделать актуальные скриншоты всех 5 дашбордов с живыми метриками.
> К этому моменту с фазы 0 должно пройти 15+ минут — метрики накоплены.

### Текущие скриншоты

Расположение: `docs/images/`

| Файл | Дашборд | UID |
|------|---------|-----|
| `dashboard-service-list.png` | Service List | `dephealth-service-list` |
| `dashboard-services-status.png` | Services Status | `dephealth-services-status` |
| `dashboard-service-status.png` | Service Status | `dephealth-service-status` |
| `dashboard-links-status.png` | Links Status | `dephealth-links-status` |
| `dashboard-link-status.png` | Link Status | `dephealth-link-status` |

### URL дашбордов

Grafana: `http://grafana.kryukov.lan` (admin / dephealth)

| Дашборд | URL |
|---------|-----|
| Service List | `http://grafana.kryukov.lan/d/dephealth-service-list/` |
| Services Status | `http://grafana.kryukov.lan/d/dephealth-services-status/` |
| Service Status | `http://grafana.kryukov.lan/d/dephealth-service-status/` |
| Links Status | `http://grafana.kryukov.lan/d/dephealth-links-status/` |
| Link Status | `http://grafana.kryukov.lan/d/dephealth-link-status/` |

### Задачи

1. Открыть каждый дашборд в Grafana (Playwright browser)
2. **Установить временной диапазон «Last 15 minutes»** на каждом дашборде
3. Для дашбордов с фильтрами (Service Status, Link Status) — выбрать репрезентативный сервис/соединение
4. Сделать полностраничный скриншот (full page) каждого дашборда
5. Сохранить скриншоты с теми же именами в `docs/images/`:
   - `docs/images/dashboard-service-list.png`
   - `docs/images/dashboard-services-status.png`
   - `docs/images/dashboard-service-status.png`
   - `docs/images/dashboard-links-status.png`
   - `docs/images/dashboard-link-status.png`
6. Проверить, что `docs/grafana-dashboards.md` ссылается на скриншоты корректно (имена не менялись)

### Порядок снятия скриншотов

1. **Service List** — главный обзор, без фильтров
2. **Services Status** — timeline всех сервисов, без фильтров
3. **Service Status** — выбрать сервис (например, `go-service`), показать детали
4. **Links Status** — таблица всех соединений, без фильтров
5. **Link Status** — выбрать соединение (например, `postgres` на `go-service`), показать детали

### Зависимости

- Фаза 0 (мониторинг и тестовые сервисы запущены, метрики накоплены 15+ мин)
- Фаза 2 (тестовые сервисы пересобраны с актуальным кодом) — желательно, но не блокирует

### Валидация

- Все 5 скриншотов обновлены в `docs/images/`
- На скриншотах видны данные за последние 15 минут
- На скриншотах все 4 сервиса отображаются с метриками
- Документация `docs/grafana-dashboards.md` корректно ссылается на изображения

---

## Фаза 7: Коммит обновлённых скриншотов ⬜

### Задачи

1. Добавить в staging обновлённые скриншоты:

   ```bash
   git add docs/images/dashboard-*.png
   ```

2. Коммит: `docs: обновить скриншоты Grafana дашбордов`

### Зависимости

- Фаза 6 (скриншоты сделаны)

### Валидация

- `git status` — чистое рабочее дерево
- `git log -1` показывает коммит со скриншотами

---

## Последующие шаги (вне этого плана)

После завершения фаз 0–7:

8. **Создать git tag** `v0.2.1`
9. **Создать GitHub Release** v0.2.1 с описанием изменений
10. **Опубликовать на PyPI**: `cd sdk-python && make publish`
11. **Опубликовать на Maven Central**: `cd sdk-java && make publish`
12. **NuGet** — при необходимости

---

## Сводка по фазам

| Фаза | Описание | Статус | Зависит от |
|------|----------|--------|-----------|
| 0 | Запуск тестовой среды и мониторинга | ⬜ | — |
| 1 | Коммит и пуш HTTP redirect fix | ⬜ | — (параллельно с 0) |
| 2 | Пересборка контейнеров + перезапуск сервисов | ⬜ | Фаза 1 |
| 3 | Проверка в кластере (conformance) | ⬜ | Фаза 2 |
| 4 | Поднять версии → 0.2.1 | ⬜ | Фаза 3 |
| 5 | Обновить CHANGELOG.md | ⬜ | Фаза 4 |
| 6 | Скриншоты Grafana дашбордов (15 мин) | ⬜ | Фазы 0, 2 |
| 7 | Коммит скриншотов | ⬜ | Фаза 6 |

### Граф зависимостей

```text
Фаза 0 (запуск среды) ─────────────────────────────────────┐
     │                                                      │
Фаза 1 (коммит fix) ──> Фаза 2 (контейнеры) ──┬──> Фаза 3 (conformance)
                                                │       │
                                                │   Фаза 4 (версии)
                                                │       │
                                                │   Фаза 5 (CHANGELOG)
                                                │
                                                └──> Фаза 6 (скриншоты) ──> Фаза 7 (коммит)
                                                         │
                                                  Фаза 0 ┘ (15 мин ожидания)
```
