.PHONY: build test test-coverage lint fmt publish image push image-conformance push-conformance pull clean help

## Конфигурация
-include ../.env

PROJECT_ROOT := $(shell cd .. && pwd)
JAVA_VERSION ?= 21
IMAGE_REGISTRY ?= docker.io
PUSH_REGISTRY  ?=
IMAGE_NAME ?= dephealth-test-java
IMAGE_TAG ?= latest
CONFORMANCE_IMAGE_NAME ?= dephealth-conformance-java
CACHE_VOLUME ?= dephealth-java-cache

MAVEN_IMAGE   ?= $(IMAGE_REGISTRY)/maven:3.9-eclipse-temurin-$(JAVA_VERSION)
RUNTIME_IMAGE ?= $(IMAGE_REGISTRY)/eclipse-temurin:$(JAVA_VERSION)-jre-alpine

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

## publish: Публикация на Maven Central (release-профиль)
publish:
	docker run --rm --network=host \
		-v $(PROJECT_ROOT):/workspace \
		-v $(CACHE_VOLUME):/root/.m2/repository \
		-v $(HOME)/.m2/settings.xml:/root/.m2/settings.xml:ro \
		-v $(HOME)/.gnupg:/host-gnupg:ro \
		-w /workspace/sdk-java \
		$(MAVEN_IMAGE) \
		sh -c 'cp -r /host-gnupg /root/.gnupg && chmod 700 /root/.gnupg && mvn clean deploy -Prelease -DskipTests'

## image: Сборка Docker-образа тестового сервиса
image:
	docker build \
		--build-arg REGISTRY=$(IMAGE_REGISTRY) \
		-t $(FULL_IMAGE) \
		-f $(PROJECT_ROOT)/test-services/java-service/Dockerfile \
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
		--build-arg REGISTRY=$(IMAGE_REGISTRY) \
		-t $(FULL_CONFORMANCE_IMAGE) \
		-f $(PROJECT_ROOT)/conformance/test-service-java/Dockerfile \
		$(PROJECT_ROOT)

## push-conformance: Загрузка образа conformance-сервиса в registry
push-conformance:
ifndef PUSH_REGISTRY
	$(error PUSH_REGISTRY не задан. Укажите: make push-conformance PUSH_REGISTRY=registry.example.com/project)
endif
	docker push $(FULL_CONFORMANCE_IMAGE)

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
