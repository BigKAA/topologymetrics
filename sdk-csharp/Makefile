.PHONY: build test test-coverage lint fmt image push image-conformance push-conformance pull clean help

## Конфигурация
-include ../.env

PROJECT_ROOT := $(shell cd .. && pwd)
DOTNET_VERSION ?= 8.0
MCR_REGISTRY   ?= mcr.microsoft.com
PUSH_REGISTRY  ?=
IMAGE_NAME ?= dephealth-test-csharp
IMAGE_TAG ?= latest
CONFORMANCE_IMAGE_NAME ?= dephealth-conformance-csharp
CACHE_VOLUME ?= dephealth-csharp-cache

SDK_IMAGE     ?= $(MCR_REGISTRY)/dotnet/sdk:$(DOTNET_VERSION)
RUNTIME_IMAGE ?= $(MCR_REGISTRY)/dotnet/aspnet:$(DOTNET_VERSION)-alpine

# Полное имя образа (с registry-префиксом если задан)
ifdef PUSH_REGISTRY
  FULL_IMAGE = $(PUSH_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
  FULL_CONFORMANCE_IMAGE = $(PUSH_REGISTRY)/$(CONFORMANCE_IMAGE_NAME):$(IMAGE_TAG)
else
  FULL_IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)
  FULL_CONFORMANCE_IMAGE = $(CONFORMANCE_IMAGE_NAME):$(IMAGE_TAG)
endif

DOCKER_RUN = docker run --rm \
	-v $(PROJECT_ROOT):/workspace \
	-v $(CACHE_VOLUME):/root/.nuget \
	-w /workspace/sdk-csharp \
	$(SDK_IMAGE)

## build: Компиляция проекта
build:
	$(DOCKER_RUN) dotnet build --verbosity minimal

## test: Запуск юнит-тестов
test:
	$(DOCKER_RUN) dotnet test --verbosity minimal

## test-coverage: Тесты с отчётом покрытия
test-coverage:
	$(DOCKER_RUN) dotnet test --collect:"XPlat Code Coverage" --verbosity minimal

## lint: Проверка форматирования (dotnet format)
lint:
	$(DOCKER_RUN) dotnet format --verify-no-changes --verbosity minimal

## fmt: Автоформатирование кода
fmt:
	$(DOCKER_RUN) dotnet format --verbosity minimal

## image: Сборка Docker-образа тестового сервиса
image:
	docker build \
		--build-arg MCR_REGISTRY=$(MCR_REGISTRY) \
		-t $(FULL_IMAGE) \
		-f $(PROJECT_ROOT)/test-services/csharp-service/Dockerfile \
		$(PROJECT_ROOT)

## push: Загрузка образа тестового сервиса в registry
push:
ifndef PUSH_REGISTRY
	$(error PUSH_REGISTRY не задан. Укажите: make push PUSH_REGISTRY=registry.example.com/project)
endif
	docker push $(FULL_IMAGE)

## image-conformance: Сборка Docker-образа conformance-сервиса
image-conformance:
	docker build \
		--build-arg MCR_REGISTRY=$(MCR_REGISTRY) \
		-t $(FULL_CONFORMANCE_IMAGE) \
		-f $(PROJECT_ROOT)/conformance/test-service-csharp/Dockerfile \
		$(PROJECT_ROOT)

## push-conformance: Загрузка образа conformance-сервиса в registry
push-conformance:
ifndef PUSH_REGISTRY
	$(error PUSH_REGISTRY не задан. Укажите: make push-conformance PUSH_REGISTRY=registry.example.com/project)
endif
	docker push $(FULL_CONFORMANCE_IMAGE)

## pull: Загрузка всех необходимых Docker-образов
pull:
	docker pull $(SDK_IMAGE)
	docker pull $(RUNTIME_IMAGE)

## clean: Очистка кешей (Docker volume)
clean:
	docker volume rm -f $(CACHE_VOLUME)

## help: Список доступных целей
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | column -t -s ':'
