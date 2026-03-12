# План разработки: HTTP Host Header & gRPC Authority Override

## 📋 Метаданные

- **Версия плана**: 1.0.0
- **Дата создания**: 2026-03-12
- **Последнее обновление**: 2026-03-12
- **Статус**: Pending

---

## 📚 История версий

- **v1.0.0** (2026-03-12): Начальная версия плана

---

## 📍 Текущий статус

- **Активная фаза**: Phase 1
- **Активный подпункт**: 1.1
- **Последнее обновление**: 2026-03-12
- **Примечание**: План создан, ожидает начала выполнения

---

## 📑 Оглавление

- [ ] [Phase 1: Specification Update](#phase-1-specification-update)
- [ ] [Phase 2: Conformance Tests](#phase-2-conformance-tests)
- [ ] [Phase 3: Go SDK](#phase-3-go-sdk)
- [ ] [Phase 4: Java SDK](#phase-4-java-sdk)
- [ ] [Phase 5: C# SDK](#phase-5-c-sdk)
- [ ] [Phase 6: Python SDK](#phase-6-python-sdk)
- [ ] [Phase 7: Documentation](#phase-7-documentation)

---

## Phase 1: Specification Update

**Dependencies**: None
**Status**: Pending

### Описание

Обновление спецификации для новых опций `hostHeader` (HTTP) и `grpcAuthority` (gRPC).
При подключении по IP через ingress/gateway API пользователям необходимо задавать
заголовок `Host` (HTTP) или pseudo-header `:authority` (gRPC) для корректной маршрутизации.
Обе опции также устанавливают TLS SNI (ServerName) при включённом TLS,
чтобы TLS handshake не падал при подключении по IP с доменным сертификатом.

### Подпункты

- [ ] **1.1 config-contract.md (EN)**
  - **Dependencies**: None
  - **Description**: Добавить опции `hostHeader` и `grpcAuthority` в спецификацию конфигурации:
    - Section 7.3 (Dependency Options): добавить `WithHTTPHostHeader(string)`,
      `WithGRPCAuthority(string)` в programmatic API
    - Section 7.5 (Validation): добавить правила конфликтов —
      `hostHeader` + `Host` в headers → error;
      `grpcAuthority` + `:authority` в metadata → error
    - Section 8.2 (Env vars): добавить `DEPHEALTH_<NAME>_HOST_HEADER`,
      `DEPHEALTH_<NAME>_GRPC_AUTHORITY`
  - **Creates**:
    - Changes in `spec/config-contract.md`
  - **Links**: N/A

- [ ] **1.2 config-contract.ru.md (RU)**
  - **Dependencies**: 1.1
  - **Description**: Те же изменения в русской версии спецификации конфигурации
  - **Creates**:
    - Changes in `spec/config-contract.ru.md`
  - **Links**: N/A

- [ ] **1.3 check-behavior.md (EN)**
  - **Dependencies**: None
  - **Description**: Обновить секции HTTP (4.1) и gRPC (4.2):
    - Добавить `hostHeader` / `grpcAuthority` в таблицы параметров
    - Описать поведение: override Host заголовка, override SNI при включённом TLS
    - Добавить примечание: `hostHeader` НЕ влияет на метку `host` в метрике
      (остаётся реальный адрес endpoint)
  - **Creates**:
    - Changes in `spec/check-behavior.md`
  - **Links**: N/A

- [ ] **1.4 check-behavior.ru.md (RU)**
  - **Dependencies**: 1.3
  - **Description**: Те же изменения в русской версии спецификации поведения
  - **Creates**:
    - Changes in `spec/check-behavior.ru.md`
  - **Links**: N/A

### ✅ Критерии завершения Phase 1

- [ ] Все подпункты завершены (1.1, 1.2, 1.3, 1.4)
- [ ] Обе опции (`hostHeader`, `grpcAuthority`) задокументированы в config-contract
- [ ] Поведение (Host override, SNI override) задокументировано в check-behavior
- [ ] Правила валидации конфликтов задокументированы
- [ ] Переменные окружения задокументированы
- [ ] markdownlint проходит без ошибок

---

## Phase 2: Conformance Tests

**Dependencies**: Phase 1
**Status**: Pending

### Описание

Добавление conformance-тестов для проверки корректной работы `hostHeader` и `grpcAuthority`.
Тесты проверяют маршрутизацию через stub-сервисы с Host-based routing.

### Подпункты

- [ ] **2.1 HTTP Host header conformance test**
  - **Dependencies**: None
  - **Description**: Создать сценарий тестирования HTTP Host header override:
    - Настроить HTTP stub для требования определённого Host header при маршрутизации
    - Dependency с `hostHeader` → healthy
    - Dependency без `hostHeader` (подключение по IP) → 404 → unhealthy
  - **Creates**:
    - `conformance/scenarios/http-host-header.yml`
  - **Links**:
    - [Existing scenario example](conformance/scenarios/auth-http-header.yml)

- [ ] **2.2 gRPC authority conformance test**
  - **Dependencies**: None
  - **Description**: Создать сценарий тестирования gRPC authority override:
    - Настроить gRPC stub для проверки `:authority` pseudo-header
    - Dependency с `grpcAuthority` → healthy
    - Dependency без `grpcAuthority` → failure
  - **Creates**:
    - `conformance/scenarios/grpc-authority.yml`
  - **Links**:
    - [Existing gRPC scenario](conformance/scenarios/auth-grpc.yml)

- [ ] **2.3 Update stub services**
  - **Dependencies**: None
  - **Description**: Обновить HTTP/gRPC stub-сервисы для поддержки Host/authority-based
    routing (если текущие stub-ы не поддерживают эту функциональность)
  - **Creates**:
    - Changes in stub service code (if needed)
  - **Links**: N/A

### ✅ Критерии завершения Phase 2

- [ ] Все подпункты завершены (2.1, 2.2, 2.3)
- [ ] Сценарии корректно описывают ожидаемое поведение
- [ ] Stub-сервисы поддерживают Host/authority-based routing
- [ ] YAML-сценарии проходят валидацию

---

## Phase 3: Go SDK

**Dependencies**: Phase 1
**Status**: Pending

### Описание

Реализация `WithHTTPHostHeader` и `WithGRPCAuthority` в Go SDK.
Go требует особой обработки: для Host header нужен `req.Host` (не `req.Header.Set`),
для gRPC — `grpc.WithAuthority()`. TLS SNI задаётся через `tls.Config.ServerName`.

### Подпункты

- [ ] **3.1 Options и конфигурация**
  - **Dependencies**: None
  - **Description**: Добавить option-функции `WithHTTPHostHeader(string)` и
    `WithGRPCAuthority(string)`. Пробросить значения до checker-ов
  - **Creates**:
    - Changes in `sdk-go/dephealth/options.go`
    - Changes in dependency/endpoint configuration structs
  - **Links**: N/A

- [ ] **3.2 HTTP checker — Host header и SNI**
  - **Dependencies**: 3.1
  - **Description**: В `HttpChecker`:
    - Добавить поле `hostHeader`
    - Применять `req.Host = hostHeader` при выполнении запроса
    - При TLS + `hostHeader`: установить `tls.Config.ServerName = hostHeader`
  - **Creates**:
    - Changes in `sdk-go/dephealth/checks/httpcheck/httpcheck.go`
  - **Links**:
    - [Go net/http Request.Host](https://pkg.go.dev/net/http#Request)

- [ ] **3.3 gRPC checker — authority и SNI**
  - **Dependencies**: 3.1
  - **Description**: В `GrpcChecker`:
    - Добавить поле `authority`
    - Применять `grpc.WithAuthority(authority)` при создании соединения
    - При TLS + `authority`: установить `tls.Config.ServerName = authority`
  - **Creates**:
    - Changes in `sdk-go/dephealth/checks/grpccheck/grpccheck.go`
  - **Links**:
    - [gRPC WithAuthority](https://pkg.go.dev/google.golang.org/grpc#WithAuthority)

- [ ] **3.4 Валидация конфликтов**
  - **Dependencies**: 3.1
  - **Description**: Добавить правила валидации:
    - `hostHeader` set AND `headers` содержит `Host` (case-insensitive) → error
    - `grpcAuthority` set AND `metadata` содержит `:authority` → error
  - **Creates**:
    - Changes in `sdk-go/dephealth/validation.go`
  - **Links**: N/A

- [ ] **3.5 Unit tests**
  - **Dependencies**: 3.2, 3.3, 3.4
  - **Description**: Тесты:
    - HTTP: Host header отправляется, SNI при TLS, ошибка конфликта
    - gRPC: authority устанавливается, SNI при TLS, ошибка конфликта
  - **Creates**:
    - Changes in `sdk-go/dephealth/checks/httpcheck/httpcheck_test.go`
    - Changes in `sdk-go/dephealth/checks/grpccheck/grpccheck_test.go`
    - Changes in validation tests
  - **Links**: N/A

### ✅ Критерии завершения Phase 3

- [ ] Все подпункты завершены (3.1, 3.2, 3.3, 3.4, 3.5)
- [ ] `go test ./...` проходит без ошибок
- [ ] `go vet ./...` без предупреждений
- [ ] Host header корректно устанавливается через `req.Host`
- [ ] gRPC authority корректно устанавливается через `grpc.WithAuthority`
- [ ] TLS SNI устанавливается при включённом TLS
- [ ] Конфликты валидируются с понятными сообщениями об ошибках

---

## Phase 4: Java SDK

**Dependencies**: Phase 1
**Status**: Pending

### Описание

Реализация `httpHostHeader` и `grpcAuthority` в Java SDK.
Java 11+ HttpClient имеет restricted headers — потребуется workaround для Host.
gRPC-Java поддерживает `overrideAuthority()` нативно.

### Подпункты

- [ ] **4.1 Builder methods и конфигурация**
  - **Dependencies**: None
  - **Description**: Добавить builder-методы `httpHostHeader(String)` и
    `grpcAuthority(String)` в конфигурацию dependency
  - **Creates**:
    - Changes in builder/config classes under
      `sdk-java/dephealth-core/src/main/java/biz/kryukov/dev/dephealth/`
  - **Links**: N/A

- [ ] **4.2 HTTP checker — Host header и SNI**
  - **Dependencies**: 4.1
  - **Description**: В `HttpHealthChecker`:
    - Сохранить поле `hostHeader`
    - Установить Host header (workaround для restricted headers в Java 11+:
      system property `jdk.httpclient.allowRestrictedHeaders=host`
      или модификация URI authority)
    - При TLS: настроить `SSLParameters` с SNI (`SNIHostName`)
  - **Creates**:
    - Changes in `sdk-java/dephealth-core/src/main/java/biz/kryukov/dev/dephealth/checks/HttpHealthChecker.java`
  - **Links**:
    - [Java HttpClient restricted headers](https://docs.oracle.com/en/java/javase/11/docs/api/java.net.http/java/net/http/HttpRequest.html)

- [ ] **4.3 gRPC checker — authority и SNI**
  - **Dependencies**: 4.1
  - **Description**: В `GrpcHealthChecker`:
    - Сохранить поле `authority`
    - Использовать `ManagedChannelBuilder.overrideAuthority(authority)`
    - TLS SNI обрабатывается автоматически gRPC при установленном authority
  - **Creates**:
    - Changes in `sdk-java/dephealth-core/src/main/java/biz/kryukov/dev/dephealth/checks/GrpcHealthChecker.java`
  - **Links**:
    - [gRPC-Java ManagedChannelBuilder](https://grpc.github.io/grpc-java/javadoc/io/grpc/ManagedChannelBuilder.html)

- [ ] **4.4 Валидация конфликтов**
  - **Dependencies**: 4.1
  - **Description**: Добавить валидацию конфликтов по аналогии с Go SDK
  - **Creates**:
    - Changes in validation classes
  - **Links**: N/A

- [ ] **4.5 Unit tests**
  - **Dependencies**: 4.2, 4.3, 4.4
  - **Description**: Тесты: Host header, authority, SNI, конфликты
  - **Creates**:
    - Test files under `sdk-java/dephealth-core/src/test/`
  - **Links**: N/A

### ✅ Критерии завершения Phase 4

- [ ] Все подпункты завершены (4.1, 4.2, 4.3, 4.4, 4.5)
- [ ] `mvn test` проходит без ошибок
- [ ] Host header корректно устанавливается (с учётом restricted headers)
- [ ] gRPC authority работает через `overrideAuthority`
- [ ] TLS SNI устанавливается при включённом TLS
- [ ] Конфликты валидируются

---

## Phase 5: C# SDK

**Dependencies**: Phase 1
**Status**: Pending

### Описание

Реализация `HttpHostHeader` и `GrpcAuthority` в C# SDK.
В `System.Net.Http` Host header нужно устанавливать per-request через
`request.Headers.Host`, а не через `DefaultRequestHeaders`.
TLS SNI задаётся через `SslClientAuthenticationOptions.TargetHost`.

### Подпункты

- [ ] **5.1 Configuration options**
  - **Dependencies**: None
  - **Description**: Добавить опции `HttpHostHeader` и `GrpcAuthority`
    в конфигурацию dependency
  - **Creates**:
    - Changes in configuration classes under `sdk-csharp/DepHealth.Core/`
  - **Links**: N/A

- [ ] **5.2 HTTP checker — Host header и SNI**
  - **Dependencies**: 5.1
  - **Description**: В `HttpChecker`:
    - Сохранить поле `hostHeader`
    - Установить `request.Headers.Host = hostHeader` (per-request)
    - При TLS: установить `SslClientAuthenticationOptions.TargetHost = hostHeader`
  - **Creates**:
    - Changes in `sdk-csharp/DepHealth.Core/Checks/HttpChecker.cs`
  - **Links**:
    - [HttpRequestHeaders.Host](https://learn.microsoft.com/en-us/dotnet/api/system.net.http.headers.httprequestheaders.host)

- [ ] **5.3 gRPC checker — authority и SNI**
  - **Dependencies**: 5.1
  - **Description**: В `GrpcChecker`:
    - Сохранить поле `authority`
    - Override authority через `GrpcChannelOptions` или request headers
    - TLS: установить `TargetHost` в SSL options
  - **Creates**:
    - Changes in `sdk-csharp/DepHealth.Core/Checks/GrpcChecker.cs`
  - **Links**:
    - [GrpcChannelOptions](https://learn.microsoft.com/en-us/dotnet/api/grpc.net.client.grpcchanneloptions)

- [ ] **5.4 Валидация конфликтов**
  - **Dependencies**: 5.1
  - **Description**: Добавить валидацию в `AuthValidation`:
    - `hostHeader` set AND `headers` содержит `Host` → error
    - `grpcAuthority` set AND `metadata` содержит `:authority` → error
  - **Creates**:
    - Changes in `sdk-csharp/DepHealth.Core/AuthValidation.cs`
  - **Links**: N/A

- [ ] **5.5 Unit tests**
  - **Dependencies**: 5.2, 5.3, 5.4
  - **Description**: Тесты: Host header per-request, authority, SNI, конфликты.
    Сборка и тестирование через Docker:
    `docker run --rm -v "$(pwd):/src" -w /src --platform linux/amd64 harbor.kryukov.lan/mcr/dotnet/sdk:8.0 dotnet test`
  - **Creates**:
    - Test files under `sdk-csharp/tests/DepHealth.Core.Tests/`
  - **Links**: N/A

### ✅ Критерии завершения Phase 5

- [ ] Все подпункты завершены (5.1, 5.2, 5.3, 5.4, 5.5)
- [ ] `dotnet test` проходит без ошибок (через Docker)
- [ ] Host header устанавливается per-request (не через DefaultRequestHeaders)
- [ ] gRPC authority корректно override-ится
- [ ] TLS SNI (`TargetHost`) устанавливается при включённом TLS
- [ ] Конфликты валидируются

---

## Phase 6: Python SDK

**Dependencies**: Phase 1
**Status**: Pending

### Описание

Реализация `http_host_header` и `grpc_authority` в Python SDK.
aiohttp поддерживает Host header нативно через dict headers.
grpcio поддерживает authority через channel option `grpc.default_authority`.

### Подпункты

- [ ] **6.1 Options и конфигурация**
  - **Dependencies**: None
  - **Description**: Добавить опции `http_host_header` и `grpc_authority`
    в конфигурацию dependency
  - **Creates**:
    - Changes in configuration modules under `sdk-python/dephealth/`
  - **Links**: N/A

- [ ] **6.2 HTTP checker — Host header и SNI**
  - **Dependencies**: 6.1
  - **Description**: В HTTP checker:
    - Сохранить поле `host_header`
    - Добавить в request headers: `headers["Host"] = host_header`
      (aiohttp поддерживает нативно)
    - При TLS: установить `server_hostname` в connector для SNI
  - **Creates**:
    - Changes in `sdk-python/dephealth/checks/http.py`
  - **Links**:
    - [aiohttp ClientSession](https://docs.aiohttp.org/en/stable/client_reference.html)

- [ ] **6.3 gRPC checker — authority и SNI**
  - **Dependencies**: 6.1
  - **Description**: В gRPC checker:
    - Сохранить поле `authority`
    - Передать `options=[("grpc.default_authority", authority)]` при создании channel
    - TLS: override target name через channel options
  - **Creates**:
    - Changes in `sdk-python/dephealth/checks/grpc.py`
  - **Links**:
    - [gRPC Python channel options](https://grpc.github.io/grpc/python/glossary.html#term-channel_arguments)

- [ ] **6.4 Валидация конфликтов**
  - **Dependencies**: 6.1
  - **Description**: Добавить валидацию конфликтов по аналогии с другими SDK
  - **Creates**:
    - Changes in validation module
  - **Links**: N/A

- [ ] **6.5 Unit tests**
  - **Dependencies**: 6.2, 6.3, 6.4
  - **Description**: Тесты: Host header, authority, SNI, конфликты
  - **Creates**:
    - Test files under `sdk-python/tests/`
  - **Links**: N/A

### ✅ Критерии завершения Phase 6

- [ ] Все подпункты завершены (6.1, 6.2, 6.3, 6.4, 6.5)
- [ ] `pytest` проходит без ошибок
- [ ] Host header корректно устанавливается через aiohttp
- [ ] gRPC authority работает через `grpc.default_authority`
- [ ] TLS SNI устанавливается при включённом TLS
- [ ] Конфликты валидируются

---

## Phase 7: Documentation

**Dependencies**: Phase 3, Phase 4, Phase 5, Phase 6
**Status**: Pending

### Описание

Обновление документации для всех SDK: добавление описания новых опций,
примеров использования через ingress/gateway API.

### Подпункты

- [ ] **7.1 Go SDK documentation**
  - **Dependencies**: None
  - **Description**: Обновить документацию Go SDK: добавить `WithHTTPHostHeader`,
    `WithGRPCAuthority` в checker docs, примеры использования
  - **Creates**:
    - Changes in `sdk-go/docs/`
  - **Links**: N/A

- [ ] **7.2 Java SDK documentation**
  - **Dependencies**: None
  - **Description**: Обновить документацию Java SDK
  - **Creates**:
    - Changes in `sdk-java/docs/`
  - **Links**: N/A

- [ ] **7.3 C# SDK documentation**
  - **Dependencies**: None
  - **Description**: Обновить документацию C# SDK
  - **Creates**:
    - Changes in `sdk-csharp/docs/`
  - **Links**: N/A

- [ ] **7.4 Python SDK documentation**
  - **Dependencies**: None
  - **Description**: Обновить документацию Python SDK
  - **Creates**:
    - Changes in `sdk-python/docs/`
  - **Links**: N/A

- [ ] **7.5 Use-case example: ingress/gateway**
  - **Dependencies**: None
  - **Description**: Добавить пример использования health check через
    ingress controller / gateway API по IP с подстановкой Host header.
    Примеры для всех 4 языков
  - **Creates**:
    - Changes in `docs/` (or SDK-specific docs)
  - **Links**: N/A

### ✅ Критерии завершения Phase 7

- [ ] Все подпункты завершены (7.1, 7.2, 7.3, 7.4, 7.5)
- [ ] Все новые опции задокументированы во всех SDK
- [ ] Пример ingress/gateway use-case добавлен
- [ ] markdownlint проходит без ошибок
- [ ] Документация на EN и RU (где применимо)

---

## 📝 Примечания

- **Метка `host` в метрике НЕ меняется** — всегда отражает реальный адрес endpoint
  (IP или hostname из конфигурации), а не значение `hostHeader`
- **`hostHeader` / `grpcAuthority` независимы от аутентификации** — могут комбинироваться
  с `bearerToken`, `basicAuth`, custom `headers`/`metadata`
- **SNI override только при включённом TLS** — если `tlsEnabled=false`, значение
  используется только для Host/authority header, без влияния на TLS
- **При `tlsSkipVerify=true` + `hostHeader`/`grpcAuthority`** — SNI всё равно
  отправляется (для маршрутизации), но валидация сертификата пропускается
- **Фазы 3–6 (SDK) могут выполняться параллельно** — у них общая зависимость
  только от Phase 1 (спецификация)
