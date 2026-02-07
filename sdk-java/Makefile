.PHONY: build test test-coverage lint fmt image push image-conformance push-conformance pull clean help

## Конфигурация
PROJECT_ROOT := $(shell cd .. && pwd)
JAVA_VERSION ?= 21
MAVEN_IMAGE ?= harbor.kryukov.lan/homelab/maven:3.9-eclipse-temurin-$(JAVA_VERSION)
RUNTIME_IMAGE ?= harbor.kryukov.lan/homelab/eclipse-temurin:$(JAVA_VERSION)-jre-alpine
REGISTRY ?= harbor.kryukov.lan/library
IMAGE_NAME ?= dephealth-test-java
IMAGE_TAG ?= latest
CONFORMANCE_IMAGE_NAME ?= dephealth-conformance-java
CACHE_VOLUME ?= dephealth-java-cache

DOCKER_RUN = docker run --rm \
	-v $(PROJECT_ROOT):/workspace \
	-v $(CACHE_VOLUME):/root/.m2 \
	-w /workspace/sdk-java \
	$(MAVEN_IMAGE)

## build: Компиляция проекта
build:
	$(DOCKER_RUN) mvn package -DskipTests -q

## test: Запуск юнит-тестов
test:
	$(DOCKER_RUN) mvn test

## test-coverage: Тесты с отчётом покрытия
test-coverage:
	$(DOCKER_RUN) mvn test -Pcoverage

## lint: Статический анализ (Checkstyle + SpotBugs)
lint:
	$(DOCKER_RUN) mvn compile checkstyle:check spotbugs:check

## fmt: Проверка форматирования (Checkstyle)
fmt:
	$(DOCKER_RUN) mvn checkstyle:check

## image: Сборка Docker-образа тестового сервиса
image:
	docker build \
		-t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
		-f $(PROJECT_ROOT)/test-services/java-service/Dockerfile \
		$(PROJECT_ROOT)

## push: Загрузка образа тестового сервиса в registry
push:
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

## image-conformance: Сборка Docker-образа conformance-сервиса
image-conformance:
	docker build \
		-t $(REGISTRY)/$(CONFORMANCE_IMAGE_NAME):$(IMAGE_TAG) \
		-f $(PROJECT_ROOT)/conformance/test-service-java/Dockerfile \
		$(PROJECT_ROOT)

## push-conformance: Загрузка образа conformance-сервиса в registry
push-conformance:
	docker push $(REGISTRY)/$(CONFORMANCE_IMAGE_NAME):$(IMAGE_TAG)

## pull: Загрузка всех необходимых Docker-образов
pull:
	docker pull $(MAVEN_IMAGE)
	docker pull $(RUNTIME_IMAGE)

## clean: Очистка кешей (Docker volume)
clean:
	docker volume rm -f $(CACHE_VOLUME)

## help: Список доступных целей
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | column -t -s ':'
