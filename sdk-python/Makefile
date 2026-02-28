# sdk-python/Makefile — Docker-обёртки для сборки, тестирования и линтинга Python SDK.
# Не требует локальной установки Python, ruff или mypy.
#
# Использование:
#   make pull           — скачать необходимые Docker-образы (первый запуск)
#   make test           — запуск тестов
#   make test-coverage  — тесты с покрытием
#   make lint           — ruff + mypy
#   make fmt            — автоформатирование (ruff format + ruff check --fix)
#   make docs           — генерация API-документации (pdoc → docs/_build/)
#   make build          — сборка пакета (sdist + wheel)
#   make publish        — публикация на PyPI (через ~/.pypirc)
#   make image          — сборка Docker-образа тестового сервиса
#   make push           — загрузка образа в registry
#   make clean          — очистка кэшей

# --- Переменные ---
-include ../.env

PYTHON_VERSION ?= 3.12
IMAGE_REGISTRY ?= docker.io
PUSH_REGISTRY  ?=
IMAGE_NAME     ?= dephealth-test-python
IMAGE_TAG      ?= latest

# Docker volumes
CACHE_VOLUME = dephealth-python-cache
PY_IMAGE     = $(IMAGE_REGISTRY)/python:$(PYTHON_VERSION)-slim

# Полное имя образа (с registry-префиксом если задан)
ifdef PUSH_REGISTRY
  FULL_IMAGE = $(PUSH_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
  FULL_CONFORMANCE_IMAGE = $(PUSH_REGISTRY)/dephealth-conformance-python:$(IMAGE_TAG)
else
  FULL_IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)
  FULL_CONFORMANCE_IMAGE = dephealth-conformance-python:$(IMAGE_TAG)
endif

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

.PHONY: build test test-coverage lint fmt security audit deadcode check-all docs publish image push image-conformance push-conformance pull clean help

## build: сборка пакета (sdist + wheel)
build:
	$(call remove_old,build)
	$(DOCKER_RUN) --name $(CN)-build $(PY_IMAGE) \
		sh -c 'pip install -q build && python -m build'

## publish: публикация на PyPI (используется ~/.pypirc)
publish:
	$(call remove_old,publish)
	$(DOCKER_RUN) \
		-v $(HOME)/.pypirc:/root/.pypirc:ro \
		--name $(CN)-publish $(PY_IMAGE) \
		sh -c 'pip install -q build twine && python -m build && twine upload dist/*'

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

## security: проверка безопасности кода (Bandit rules через Ruff)
security:
	$(call remove_old,security)
	$(DOCKER_RUN) --name $(CN)-security $(PY_IMAGE) \
		sh -c '$(PIP_INSTALL) && ruff check --select S dephealth/ dephealth_fastapi/'

## audit: проверка зависимостей на уязвимости (pip-audit)
audit:
	$(call remove_old,audit)
	$(DOCKER_RUN) --name $(CN)-audit $(PY_IMAGE) \
		sh -c 'pip install -q --upgrade pip 2>/dev/null && $(PIP_INSTALL) && pip-audit'

## deadcode: поиск мёртвого кода (vulture)
deadcode:
	$(call remove_old,deadcode)
	$(DOCKER_RUN) --name $(CN)-deadcode $(PY_IMAGE) \
		sh -c '$(PIP_INSTALL) && vulture dephealth/ dephealth_fastapi/ --min-confidence 80'

## check-all: все проверки (lint + security + audit + deadcode)
check-all: lint audit deadcode

## fmt: автоформатирование кода
fmt:
	$(call remove_old,fmt)
	$(DOCKER_RUN) --name $(CN)-fmt $(PY_IMAGE) \
		sh -c '$(PIP_INSTALL) && ruff format . && ruff check --fix .'

## docs: генерация API-документации (pdoc)
docs:
	$(call remove_old,docs)
	$(DOCKER_RUN) --name $(CN)-docs $(PY_IMAGE) \
		sh -c 'pip install -q -e ".[all,docs]" 2>/dev/null && \
		python -c "import pkgutil,dephealth,dephealth_fastapi; print(\" \".join(m.name for pkg in [dephealth,dephealth_fastapi] for m in pkgutil.walk_packages(pkg.__path__,pkg.__name__+\".\")))" \
		| xargs pdoc -o docs/_build dephealth dephealth_fastapi'

## image: сборка Docker-образа тестового сервиса
image:
	docker build \
		--build-arg REGISTRY=$(IMAGE_REGISTRY) \
		-t $(FULL_IMAGE) \
		-f $(DOCKERFILE) \
		$(PROJECT_ROOT)

## push: загрузка образа тестового сервиса в registry
push:
ifndef PUSH_REGISTRY
	$(error PUSH_REGISTRY не задан. Укажите: make push PUSH_REGISTRY=registry.example.com/project)
endif
	docker push $(FULL_IMAGE)

## image-conformance: сборка Docker-образа conformance-сервиса
image-conformance:
	docker build \
		--build-arg REGISTRY=$(IMAGE_REGISTRY) \
		-t $(FULL_CONFORMANCE_IMAGE) \
		-f $(PROJECT_ROOT)/conformance/test-service-python/Dockerfile \
		$(PROJECT_ROOT)

## push-conformance: загрузка conformance-образа в registry
push-conformance:
ifndef PUSH_REGISTRY
	$(error PUSH_REGISTRY не задан. Укажите: make push-conformance PUSH_REGISTRY=registry.example.com/project)
endif
	docker push $(FULL_CONFORMANCE_IMAGE)

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
