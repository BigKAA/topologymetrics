# sdk-go/Makefile — Docker-обёртки для сборки, тестирования и линтинга Go SDK.
# Не требует локальной установки Go или golangci-lint.
#
# Использование:
#   make pull           — скачать необходимые Docker-образы (первый запуск)
#   make test           — запуск тестов
#   make test-coverage  — тесты с покрытием
#   make lint           — golangci-lint
#   make fmt            — goimports + gofmt
#   make build          — компиляция пакетов
#   make image          — сборка Docker-образа тестового сервиса
#   make push           — загрузка образа в registry
#   make clean          — очистка кэшей

# --- Переменные ---
-include ../.env

GO_VERSION     ?= 1.25
LINT_VERSION   ?= v2.8.0
IMAGE_REGISTRY ?= docker.io
PUSH_REGISTRY  ?=
IMAGE_NAME     ?= dephealth-test-go
IMAGE_TAG      ?= latest

# Docker volumes
CACHE_VOLUME  = dephealth-go-cache
GO_IMAGE      = $(IMAGE_REGISTRY)/golang:$(GO_VERSION)
GO_ALPINE     = $(IMAGE_REGISTRY)/golang:$(GO_VERSION)-alpine
ALPINE_IMAGE  = $(IMAGE_REGISTRY)/alpine:3.21
LINT_IMAGE    = $(IMAGE_REGISTRY)/golangci/golangci-lint:$(LINT_VERSION)

# Полное имя образа (с registry-префиксом если задан)
ifdef PUSH_REGISTRY
  FULL_IMAGE = $(PUSH_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
else
  FULL_IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)
endif

# Пути (относительно корня проекта)
PROJECT_ROOT  = $(shell cd .. && pwd)
DOCKERFILE    = $(PROJECT_ROOT)/test-services/go-service/Dockerfile

# Префикс имён контейнеров
CN = dephealth-go

# Удаление предыдущего контейнера (если остался после прерывания)
define remove_old
	@docker rm -f $(CN)-$(1) 2>/dev/null || true
endef

# Общие флаги docker run
DOCKER_RUN = docker run --rm \
	-v $(PROJECT_ROOT):/workspace \
	-v $(CACHE_VOLUME):/go \
	-w /workspace/sdk-go \
	-e GOFLAGS=-buildvcs=false

.PHONY: build test test-coverage lint fmt image push pull clean help

## build: компиляция всех пакетов SDK
build:
	$(call remove_old,build)
	$(DOCKER_RUN) --name $(CN)-build $(GO_IMAGE) go build ./...

## test: запуск юнит-тестов
test:
	$(call remove_old,test)
	$(DOCKER_RUN) --name $(CN)-test $(GO_IMAGE) go test -race -count=1 ./...

## test-coverage: тесты с отчётом о покрытии
test-coverage:
	$(call remove_old,coverage)
	$(DOCKER_RUN) --name $(CN)-coverage $(GO_IMAGE) \
		sh -c 'go test -race -count=1 -coverprofile=/tmp/coverage.out ./... && go tool cover -func=/tmp/coverage.out'

## lint: статический анализ (golangci-lint)
lint:
	$(call remove_old,lint)
	docker run --rm --name $(CN)-lint \
		-v $(PROJECT_ROOT):/workspace \
		-v $(CACHE_VOLUME)-lint:/root/.cache \
		-w /workspace/sdk-go \
		-e GOFLAGS=-buildvcs=false \
		$(LINT_IMAGE) \
		golangci-lint run -v ./...

## fmt: форматирование кода (goimports + gofmt)
fmt:
	$(call remove_old,fmt)
	$(DOCKER_RUN) --name $(CN)-fmt $(GO_IMAGE) \
		sh -c 'go install golang.org/x/tools/cmd/goimports@latest && \
			goimports -w -local github.com/BigKAA/topologymetrics/sdk-go . && \
			gofmt -w .'

## image: сборка Docker-образа тестового сервиса
image:
	docker build \
		--build-arg REGISTRY=$(IMAGE_REGISTRY) \
		-t $(FULL_IMAGE) \
		-f $(DOCKERFILE) \
		$(PROJECT_ROOT)

## push: загрузка образа в registry
push:
ifndef PUSH_REGISTRY
	$(error PUSH_REGISTRY не задан. Укажите: make push PUSH_REGISTRY=registry.example.com/project)
endif
	docker push $(FULL_IMAGE)

## pull: скачать все необходимые Docker-образы
pull:
	docker pull $(GO_IMAGE)
	docker pull $(GO_ALPINE)
	docker pull $(ALPINE_IMAGE)
	docker pull $(LINT_IMAGE)
	@echo ""
	@echo "Все образы скачаны:"
	@echo "  $(GO_IMAGE)          — build, test, fmt"
	@echo "  $(GO_ALPINE)  — image (builder stage)"
	@echo "  $(ALPINE_IMAGE)          — image (runtime stage)"
	@echo "  $(LINT_IMAGE)    — lint"

## clean: удаление Docker volumes с кэшем
clean:
	docker volume rm $(CACHE_VOLUME) 2>/dev/null || true
	docker volume rm $(CACHE_VOLUME)-lint 2>/dev/null || true

## help: список целей
help:
	@echo "Цели:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
