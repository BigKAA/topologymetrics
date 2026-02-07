.PHONY: build test test-coverage lint fmt image push image-conformance push-conformance pull clean help

## Конфигурация
PROJECT_ROOT := $(shell cd .. && pwd)
DOTNET_VERSION ?= 8.0
SDK_IMAGE ?= harbor.kryukov.lan/mcr/dotnet/sdk:$(DOTNET_VERSION)
RUNTIME_IMAGE ?= harbor.kryukov.lan/mcr/dotnet/aspnet:$(DOTNET_VERSION)-alpine
REGISTRY ?= harbor.kryukov.lan/library
IMAGE_NAME ?= dephealth-test-csharp
IMAGE_TAG ?= latest
CONFORMANCE_IMAGE_NAME ?= dephealth-conformance-csharp
CACHE_VOLUME ?= dephealth-csharp-cache

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
		-t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
		-f $(PROJECT_ROOT)/test-services/csharp-service/Dockerfile \
		$(PROJECT_ROOT)

## push: Загрузка образа тестового сервиса в registry
push:
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

## image-conformance: Сборка Docker-образа conformance-сервиса
image-conformance:
	docker build \
		-t $(REGISTRY)/$(CONFORMANCE_IMAGE_NAME):$(IMAGE_TAG) \
		-f $(PROJECT_ROOT)/conformance/test-service-csharp/Dockerfile \
		$(PROJECT_ROOT)

## push-conformance: Загрузка образа conformance-сервиса в registry
push-conformance:
	docker push $(REGISTRY)/$(CONFORMANCE_IMAGE_NAME):$(IMAGE_TAG)

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
