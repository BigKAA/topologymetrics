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
GO_VERSION   ?= 1.25
LINT_VERSION ?= v2.8.0
REGISTRY     ?= harbor.kryukov.lan/library
IMAGE_NAME   ?= dephealth-test-go
IMAGE_TAG    ?= latest

# Docker volumes
CACHE_VOLUME  = dephealth-go-cache
GO_IMAGE      = harbor.kryukov.lan/docker/golang:$(GO_VERSION)
GO_ALPINE     = harbor.kryukov.lan/docker/golang:$(GO_VERSION)-alpine
ALPINE_IMAGE  = harbor.kryukov.lan/docker/alpine:3.21
LINT_IMAGE    = harbor.kryukov.lan/docker/golangci/golangci-lint:$(LINT_VERSION)

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
			goimports -w -local github.com/BigKAA/topologymetrics . && \
			gofmt -w .'

## image: сборка Docker-образа тестового сервиса
image:
	docker build \
		-t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
		-f $(DOCKERFILE) \
		$(PROJECT_ROOT)

## push: загрузка образа в registry
push:
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

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
