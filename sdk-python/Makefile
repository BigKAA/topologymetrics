# sdk-python/Makefile — Docker-обёртки для сборки, тестирования и линтинга Python SDK.
# Не требует локальной установки Python, ruff или mypy.
#
# Использование:
#   make pull           — скачать необходимые Docker-образы (первый запуск)
#   make test           — запуск тестов
#   make test-coverage  — тесты с покрытием
#   make lint           — ruff + mypy
#   make fmt            — автоформатирование (ruff format + ruff check --fix)
#   make build          — проверка сборки пакета
#   make image          — сборка Docker-образа тестового сервиса
#   make push           — загрузка образа в registry
#   make clean          — очистка кэшей

# --- Переменные ---
PYTHON_VERSION ?= 3.12
REGISTRY       ?= harbor.kryukov.lan/library
IMAGE_NAME     ?= dephealth-test-python
IMAGE_TAG      ?= latest

# Docker volumes
CACHE_VOLUME = dephealth-python-cache
PY_IMAGE     = harbor.kryukov.lan/homelab/python:$(PYTHON_VERSION)-slim

# Пути
PROJECT_ROOT = $(shell cd .. && pwd)
DOCKERFILE   = $(PROJECT_ROOT)/test-services/python-service/Dockerfile

# Префикс имён контейнеров
CN = dephealth-py

# Удаление предыдущего контейнера
define remove_old
	@docker rm -f $(CN)-$(1) 2>/dev/null || true
endef

# Общие флаги docker run
DOCKER_RUN = docker run --rm \
	-v $(PROJECT_ROOT):/workspace \
	-v $(CACHE_VOLUME):/root/.cache/pip \
	-w /workspace/sdk-python

# Установка пакета с dev-зависимостями
PIP_INSTALL = pip install -q -e ".[dev]" 2>/dev/null

.PHONY: build test test-coverage lint fmt image push image-conformance push-conformance pull clean help

## build: проверка сборки пакета
build:
	$(call remove_old,build)
	$(DOCKER_RUN) --name $(CN)-build $(PY_IMAGE) \
		sh -c '$(PIP_INSTALL) && python -c "import dephealth; print(\"OK\")"'

## test: запуск юнит-тестов
test:
	$(call remove_old,test)
	$(DOCKER_RUN) --name $(CN)-test $(PY_IMAGE) \
		sh -c '$(PIP_INSTALL) && pytest -v --tb=short'

## test-coverage: тесты с отчётом о покрытии
test-coverage:
	$(call remove_old,coverage)
	$(DOCKER_RUN) --name $(CN)-coverage $(PY_IMAGE) \
		sh -c '$(PIP_INSTALL) && pytest --cov=dephealth --cov-report=term-missing --tb=short'

## lint: статический анализ (ruff + mypy)
lint:
	$(call remove_old,lint)
	$(DOCKER_RUN) --name $(CN)-lint $(PY_IMAGE) \
		sh -c '$(PIP_INSTALL) && ruff check . && ruff format --check . && mypy dephealth/ dephealth_fastapi/ --strict'

## fmt: автоформатирование кода
fmt:
	$(call remove_old,fmt)
	$(DOCKER_RUN) --name $(CN)-fmt $(PY_IMAGE) \
		sh -c '$(PIP_INSTALL) && ruff format . && ruff check --fix .'

## image: сборка Docker-образа тестового сервиса
image:
	docker build \
		-t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
		-f $(DOCKERFILE) \
		$(PROJECT_ROOT)

## push: загрузка образа тестового сервиса в registry
push:
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

## image-conformance: сборка Docker-образа conformance-сервиса
image-conformance:
	docker build \
		-t $(REGISTRY)/dephealth-conformance-python:$(IMAGE_TAG) \
		-f $(PROJECT_ROOT)/conformance/test-service-python/Dockerfile \
		$(PROJECT_ROOT)

## push-conformance: загрузка conformance-образа в registry
push-conformance:
	docker push $(REGISTRY)/dephealth-conformance-python:$(IMAGE_TAG)

## pull: скачать необходимые Docker-образы
pull:
	docker pull $(PY_IMAGE)
	@echo ""
	@echo "Образ скачан:"
	@echo "  $(PY_IMAGE) — build, test, lint, fmt"

## clean: удаление Docker volume с кэшем
clean:
	docker volume rm $(CACHE_VOLUME) 2>/dev/null || true

## help: список целей
help:
	@echo "Цели:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
