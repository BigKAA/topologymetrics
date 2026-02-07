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
Каждый Makefile загружает `../.env` через `-include ../.env` (опционально).

| Переменная | Описание | По умолчанию |
| --- | --- | --- |
| `<LANG>_VERSION` | Версия языка/рантайма | `GO_VERSION ?= 1.25` |
| `IMAGE_REGISTRY` | Registry для базовых образов (pull) | `docker.io` |
| `MCR_REGISTRY` | Registry для MCR-образов (только C#) | `mcr.microsoft.com` |
| `PUSH_REGISTRY` | Registry для push собранных образов | (пусто — локальный тег) |
| `IMAGE_NAME` | Имя Docker-образа | `dephealth-test-go` |
| `IMAGE_TAG` | Тег Docker-образа | `latest` |
| `LINT_VERSION` | Версия линтера (если отдельный образ) | `v2.8.0` |

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
| Go | `$(IMAGE_REGISTRY)/golang:$(GO_VERSION)` | `$(IMAGE_REGISTRY)/golangci/golangci-lint:$(LINT_VERSION)` |
| Python | `$(IMAGE_REGISTRY)/python:$(PYTHON_VERSION)-slim` | Встроенный (`ruff`, `mypy`) |
| Java | `$(IMAGE_REGISTRY)/maven:3.9-eclipse-temurin-$(JAVA_VERSION)` | Встроенный (SpotBugs, Checkstyle) |
| C# | `$(MCR_REGISTRY)/dotnet/sdk:$(DOTNET_VERSION)` | Встроенный (dotnet format) |

## Docker-образы тестовых сервисов

Формат имени: `dephealth-test-{lang}`

При заданном `PUSH_REGISTRY` образы получают полный путь:

- `$(PUSH_REGISTRY)/dephealth-test-go:latest`
- `$(PUSH_REGISTRY)/dephealth-test-python:latest`
- `$(PUSH_REGISTRY)/dephealth-test-java:latest`
- `$(PUSH_REGISTRY)/dephealth-test-csharp:latest`

Без `PUSH_REGISTRY` — локальный тег без registry-префикса.

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
-include ../.env

GO_VERSION     ?= 1.25
LINT_VERSION   ?= v2.8.0
IMAGE_REGISTRY ?= docker.io
PUSH_REGISTRY  ?=
IMAGE_NAME     ?= dephealth-test-go
IMAGE_TAG      ?= latest

CACHE_VOLUME = dephealth-go-cache
GO_IMAGE     = $(IMAGE_REGISTRY)/golang:$(GO_VERSION)
LINT_IMAGE   = $(IMAGE_REGISTRY)/golangci/golangci-lint:$(LINT_VERSION)

ifdef PUSH_REGISTRY
  FULL_IMAGE = $(PUSH_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
else
  FULL_IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)
endif

image:
    docker build \
        --build-arg REGISTRY=$(IMAGE_REGISTRY) \
        -t $(FULL_IMAGE) \
        -f $(PROJECT_ROOT)/test-services/go-service/Dockerfile \
        $(PROJECT_ROOT)
```

### Python (`sdk-python/Makefile`)

```makefile
-include ../.env

PYTHON_VERSION ?= 3.12
IMAGE_REGISTRY ?= docker.io
PUSH_REGISTRY  ?=

PY_IMAGE = $(IMAGE_REGISTRY)/python:$(PYTHON_VERSION)-slim

image:
    docker build \
        --build-arg REGISTRY=$(IMAGE_REGISTRY) \
        -t $(FULL_IMAGE) \
        -f $(DOCKERFILE) \
        $(PROJECT_ROOT)
```

### Java (`sdk-java/Makefile`)

```makefile
-include ../.env

JAVA_VERSION   ?= 21
IMAGE_REGISTRY ?= docker.io
PUSH_REGISTRY  ?=

MAVEN_IMAGE   = $(IMAGE_REGISTRY)/maven:3.9-eclipse-temurin-$(JAVA_VERSION)
RUNTIME_IMAGE = $(IMAGE_REGISTRY)/eclipse-temurin:$(JAVA_VERSION)-jre-alpine

image:
    docker build \
        --build-arg REGISTRY=$(IMAGE_REGISTRY) \
        -t $(FULL_IMAGE) \
        -f $(PROJECT_ROOT)/test-services/java-service/Dockerfile \
        $(PROJECT_ROOT)
```

### C# (`sdk-csharp/Makefile`)

```makefile
-include ../.env

DOTNET_VERSION ?= 8.0
MCR_REGISTRY   ?= mcr.microsoft.com
PUSH_REGISTRY  ?=

SDK_IMAGE     = $(MCR_REGISTRY)/dotnet/sdk:$(DOTNET_VERSION)
RUNTIME_IMAGE = $(MCR_REGISTRY)/dotnet/aspnet:$(DOTNET_VERSION)-alpine

image:
    docker build \
        --build-arg MCR_REGISTRY=$(MCR_REGISTRY) \
        -t $(FULL_IMAGE) \
        -f $(PROJECT_ROOT)/test-services/csharp-service/Dockerfile \
        $(PROJECT_ROOT)
```
