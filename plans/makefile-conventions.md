# Конвенции Makefile для SDK

Единые правила оформления Makefile для всех SDK проекта dephealth.
Каждый SDK (`sdk-go/`, `sdk-python/`, `sdk-java/`, `sdk-csharp/`) содержит
собственный `Makefile`, работающий через Docker без локальных компиляторов.

## Обязательные цели

| Цель | Описание |
| --- | --- |
| `build` | Компиляция / проверка сборки пакета |
| `test` | Запуск юнит-тестов |
| `test-coverage` | Тесты с отчётом о покрытии |
| `lint` | Статический анализ / линтер |
| `fmt` | Автоформатирование кода |
| `image` | Сборка Docker-образа тестового сервиса |
| `push` | Загрузка образа в registry |
| `pull` | Скачать все необходимые Docker-образы |
| `clean` | Очистка кэшей (Docker volumes) |
| `help` | Список целей с описаниями |

## Переменные

Все переменные задаются через `?=` (перезаписываемые извне).

| Переменная | Описание | Пример |
| --- | --- | --- |
| `<LANG>_VERSION` | Версия языка/рантайма | `GO_VERSION ?= 1.25` |
| `REGISTRY` | Container registry | `harbor.kryukov.lan/library` |
| `IMAGE_NAME` | Имя Docker-образа | `dephealth-test-go` |
| `IMAGE_TAG` | Тег Docker-образа | `latest` |
| `LINT_VERSION` | Версия линтера (если отдельный образ) | `v2.1.6` |

## Docker volumes

Формат имени: `dephealth-{lang}-cache`

| Язык | Volume | Содержимое |
| --- | --- | --- |
| Go | `dephealth-go-cache` | `/go` (модули + build cache) |
| Python | `dephealth-python-cache` | pip cache (`/root/.cache/pip`) |
| Java | `dephealth-java-cache` | Maven cache (`/root/.m2`) |
| C# | `dephealth-csharp-cache` | NuGet cache (`/root/.nuget`) |

## Docker-образы для сборки

| Язык | Образ сборки | Образ линтера |
| --- | --- | --- |
| Go | `harbor.kryukov.lan/homelab/golang:$(GO_VERSION)` | `harbor.kryukov.lan/homelab/golangci/golangci-lint:$(LINT_VERSION)` |
| Python | `harbor.kryukov.lan/homelab/python:$(PYTHON_VERSION)-slim` | Встроенный (`ruff`, `mypy`) |
| Java | `harbor.kryukov.lan/homelab/maven:$(JAVA_VERSION)` | Встроенный (SpotBugs, Checkstyle) |
| C# | `mcr.microsoft.com/dotnet/sdk:$(DOTNET_VERSION)` | Встроенный (dotnet format) |

## Docker-образы тестовых сервисов

Формат имени: `dephealth-test-{lang}`

Примеры:

- `harbor.kryukov.lan/library/dephealth-test-go:latest`
- `harbor.kryukov.lan/library/dephealth-test-python:latest`
- `harbor.kryukov.lan/library/dephealth-test-java:latest`
- `harbor.kryukov.lan/library/dephealth-test-csharp:latest`

## Предварительное скачивание образов

Перед первым использованием необходимо скачать Docker-образы:

```bash
cd sdk-{lang}
make pull
```

Цель `pull` скачивает все образы, необходимые для работы остальных целей.
Это разовая операция; повторно запускать нужно только при смене версий
в переменных `<LANG>_VERSION` / `LINT_VERSION`.

## Общие правила

1. **Без локальных зависимостей** — все цели работают только через Docker.
2. **Монтирование проекта** — корень монтируется в `/workspace`, рабочая
   директория — `/workspace/sdk-{lang}`.
3. **`.PHONY`** — все цели объявлены как `.PHONY`.
4. **Флаг `-buildvcs=false`** — для Go (избегает ошибок git внутри контейнера).
5. **Контекст image** — корень проекта (`$(PROJECT_ROOT)`),
   `Dockerfile` из `test-services/{lang}-service/Dockerfile`.
6. **Комментарии** — на русском, формат `## цель: описание` (для `help`).

## Примеры

### Go (`sdk-go/Makefile`)

```makefile
GO_VERSION   ?= 1.25
LINT_VERSION ?= v2.1.6
REGISTRY     ?= harbor.kryukov.lan/library
IMAGE_NAME   ?= dephealth-test-go
IMAGE_TAG    ?= latest

CACHE_VOLUME = dephealth-go-cache
GO_IMAGE     = harbor.kryukov.lan/homelab/golang:$(GO_VERSION)
LINT_IMAGE   = harbor.kryukov.lan/homelab/golangci/golangci-lint:$(LINT_VERSION)
PROJECT_ROOT = $(shell cd .. && pwd)

DOCKER_RUN = docker run --rm \
    -v $(PROJECT_ROOT):/workspace \
    -v $(CACHE_VOLUME):/go \
    -w /workspace/sdk-go \
    -e GOFLAGS=-buildvcs=false

test:
    $(DOCKER_RUN) $(GO_IMAGE) go test -race -count=1 ./...

lint:
    docker run --rm \
        -v $(PROJECT_ROOT):/workspace \
        -v $(CACHE_VOLUME):/go \
        -w /workspace/sdk-go \
        -e GOFLAGS=-buildvcs=false \
        $(LINT_IMAGE) golangci-lint run ./...

image:
    docker build \
        -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
        -f $(PROJECT_ROOT)/test-services/go-service/Dockerfile \
        $(PROJECT_ROOT)
```

### Python (`sdk-python/Makefile`)

```makefile
PYTHON_VERSION ?= 3.12
REGISTRY       ?= harbor.kryukov.lan/library
IMAGE_NAME     ?= dephealth-test-python
IMAGE_TAG      ?= latest

CACHE_VOLUME = dephealth-python-cache
PY_IMAGE     = harbor.kryukov.lan/homelab/python:$(PYTHON_VERSION)-slim
PROJECT_ROOT = $(shell cd .. && pwd)

DOCKER_RUN = docker run --rm \
    -v $(PROJECT_ROOT):/workspace \
    -v $(CACHE_VOLUME):/root/.cache/pip \
    -w /workspace/sdk-python

test:
    $(DOCKER_RUN) $(PY_IMAGE) \
        sh -c 'pip install -q -e ".[dev]" && pytest -v --tb=short'

lint:
    $(DOCKER_RUN) $(PY_IMAGE) \
        sh -c 'pip install -q ruff mypy && ruff check . && mypy dephealth/ --strict'

fmt:
    $(DOCKER_RUN) $(PY_IMAGE) \
        sh -c 'pip install -q ruff && ruff format . && ruff check --fix .'

image:
    docker build \
        -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
        -f $(PROJECT_ROOT)/test-services/python-service/Dockerfile \
        $(PROJECT_ROOT)
```

### Java (`sdk-java/Makefile`)

```makefile
JAVA_VERSION ?= 3.9-eclipse-temurin-21
REGISTRY     ?= harbor.kryukov.lan/library
IMAGE_NAME   ?= dephealth-test-java
IMAGE_TAG    ?= latest

CACHE_VOLUME = dephealth-java-cache
MVN_IMAGE    = harbor.kryukov.lan/homelab/maven:$(JAVA_VERSION)
PROJECT_ROOT = $(shell cd .. && pwd)

DOCKER_RUN = docker run --rm \
    -v $(PROJECT_ROOT):/workspace \
    -v $(CACHE_VOLUME):/root/.m2 \
    -w /workspace/sdk-java

test:
    $(DOCKER_RUN) $(MVN_IMAGE) mvn test -q

lint:
    $(DOCKER_RUN) $(MVN_IMAGE) mvn spotbugs:check checkstyle:check -q

build:
    $(DOCKER_RUN) $(MVN_IMAGE) mvn package -DskipTests -q

image:
    docker build \
        -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
        -f $(PROJECT_ROOT)/test-services/java-service/Dockerfile \
        $(PROJECT_ROOT)
```

### C# (`sdk-csharp/Makefile`)

```makefile
DOTNET_VERSION ?= 9.0
REGISTRY       ?= harbor.kryukov.lan/library
IMAGE_NAME     ?= dephealth-test-csharp
IMAGE_TAG      ?= latest

CACHE_VOLUME = dephealth-csharp-cache
DOTNET_IMAGE = mcr.microsoft.com/dotnet/sdk:$(DOTNET_VERSION)
PROJECT_ROOT = $(shell cd .. && pwd)

DOCKER_RUN = docker run --rm \
    -v $(PROJECT_ROOT):/workspace \
    -v $(CACHE_VOLUME):/root/.nuget \
    -w /workspace/sdk-csharp

test:
    $(DOCKER_RUN) $(DOTNET_IMAGE) dotnet test --verbosity minimal

lint:
    $(DOCKER_RUN) $(DOTNET_IMAGE) dotnet format --verify-no-changes

build:
    $(DOCKER_RUN) $(DOTNET_IMAGE) dotnet build --no-restore

image:
    docker build \
        -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
        -f $(PROJECT_ROOT)/test-services/csharp-service/Dockerfile \
        $(PROJECT_ROOT)
```
