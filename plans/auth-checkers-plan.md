# Authentication for HTTP and gRPC Health Checkers — v0.5.0

## Summary

Add authentication support to HTTP and gRPC health checkers across all 4 SDKs.
Currently HTTP and gRPC checkers have no authentication — only TLS configuration.
This plan adds custom headers/metadata, Bearer token, and Basic Auth helpers.

## Requirements (agreed)

| ID | Requirement | Details |
|----|-------------|---------|
| FR-1 | Custom Headers (HTTP) | `map[string]string` added to every health-check request |
| FR-2 | Custom Metadata (gRPC) | `map[string]string` added to every Health/Check call |
| FR-3 | WithBearerToken helper | Adds `Authorization: Bearer <token>` to headers/metadata |
| FR-4 | WithBasicAuth helper | Adds `Authorization: Basic <base64(user:pass)>` to headers/metadata |
| FR-5 | Conflict validation | BearerToken + BasicAuth + custom `Authorization` header = error at creation time |
| FR-6 | Per-dependency config | Each dependency has its own auth settings (already in architecture) |
| FR-7 | Framework Integration | Spring Boot Starter: YAML config for bearer-token, basic-auth, headers |
| NFR-1 | No credential logging | Credentials MUST NOT appear in String()/toString()/__repr__ |
| NFR-2 | Backward compatibility | Existing code without auth continues to work unchanged |
| NFR-3 | Consistent across SDKs | All 4 SDKs implement the same auth options |
| NFR-4 | Spec-first | spec/check-behavior.md updated before implementation |
| FR-8 | auth_error classification | HTTP 401/403 → status_category="auth_error", status_detail="auth_error" |

## Version

**v0.5.0** — new feature, backward compatible. All SDKs bump to 0.5.0.

---

## Phase 1: Specification Update

**Goal**: Update spec/check-behavior.md and spec/config-contract.md with authentication parameters.

### spec/check-behavior.md changes

#### Section 4.1 HTTP Checker — add parameters:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `headers` | `map[string, string]` | `{}` | Custom HTTP headers added to every health-check request |
| `bearerToken` | `string` | `""` | Convenience: adds `Authorization: Bearer <token>` header |
| `basicAuth.username` | `string` | `""` | Convenience: adds `Authorization: Basic <base64>` header |
| `basicAuth.password` | `string` | `""` | Used together with `basicAuth.username` |

Validation rules:
- If `bearerToken` is set AND `headers` contains `Authorization` key → **validation error**
- If `basicAuth` is set AND `headers` contains `Authorization` key → **validation error**
- If `bearerToken` is set AND `basicAuth` is set → **validation error**
- Only one auth method allowed at a time

Behavior:
- Headers from `headers` map are added to every request
- `bearerToken` adds `Authorization: Bearer <token>` header
- `basicAuth` adds `Authorization: Basic <base64(username:password)>` header
- `User-Agent: dephealth/<version>` is always set (custom `User-Agent` in headers overrides it)

Error classification:
- HTTP 401 → `status_category="auth_error"`, `status_detail="auth_error"`
- HTTP 403 → `status_category="auth_error"`, `status_detail="auth_error"`
- Other non-2xx → unchanged (`status_category="unhealthy"`, `status_detail="http_NNN"`)

#### Section 4.2 gRPC Checker — add parameters:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `metadata` | `map[string, string]` | `{}` | Custom gRPC metadata added to every Health/Check call |
| `bearerToken` | `string` | `""` | Convenience: adds `authorization: Bearer <token>` metadata |
| `basicAuth.username` | `string` | `""` | Convenience: adds `authorization: Basic <base64>` metadata |
| `basicAuth.password` | `string` | `""` | Used together with `basicAuth.username` |

Same validation rules as HTTP.

Error classification:
- gRPC status UNAUTHENTICATED → `status_category="auth_error"`, `status_detail="auth_error"`
- gRPC status PERMISSION_DENIED → `status_category="auth_error"`, `status_detail="auth_error"`

### spec/config-contract.md changes

Add HTTP/gRPC auth parameters to configuration tables:

```yaml
# HTTP dependency with auth
payment-api:
  type: http
  url: http://payment.svc:8080
  critical: true
  http:
    health-path: /ready
    bearer-token: "eyJhbG..."
    # OR
    basic-auth:
      username: "admin"
      password: "secret"
    # OR
    headers:
      X-API-Key: "my-key"
      X-Custom: "value"

# gRPC dependency with auth
grpc-backend:
  type: grpc
  url: grpc://backend.svc:9090
  critical: true
  grpc:
    metadata:
      authorization: "Bearer eyJhbG..."
      x-custom-header: "value"
```

### Deliverables

- [x] Updated `spec/check-behavior.md` sections 4.1 and 4.2
- [x] Updated `spec/config-contract.md` with auth config parameters
- [x] Updated error classification table in spec

---

## Phase 2: Go SDK Implementation

**Goal**: Add authentication to Go HTTP and gRPC checkers.

### 2.1 DependencyConfig (options.go)

Add new fields:

```go
type DependencyConfig struct {
    // ... existing fields ...

    // HTTP auth
    HTTPHeaders      map[string]string  // Custom headers
    HTTPBearerToken  string             // Bearer token
    HTTPBasicUser    string             // Basic Auth username
    HTTPBasicPass    string             // Basic Auth password

    // gRPC auth
    GRPCMetadata     map[string]string  // Custom metadata
    GRPCBearerToken  string             // Bearer token
    GRPCBasicUser    string             // Basic Auth username
    GRPCBasicPass    string             // Basic Auth password
}
```

### 2.2 DependencyOption functions (options.go)

```go
// HTTP auth options
func WithHTTPHeaders(headers map[string]string) DependencyOption
func WithHTTPBearerToken(token string) DependencyOption
func WithHTTPBasicAuth(username, password string) DependencyOption

// gRPC auth options
func WithGRPCMetadata(metadata map[string]string) DependencyOption
func WithGRPCBearerToken(token string) DependencyOption
func WithGRPCBasicAuth(username, password string) DependencyOption
```

### 2.3 HTTP Checker (checks/http.go)

Add `headers map[string]string` to HTTPChecker:

```go
type HTTPChecker struct {
    healthPath    string
    tlsEnabled    bool
    tlsSkipVerify bool
    headers       map[string]string  // NEW: custom headers
}

// New options:
func WithHeaders(headers map[string]string) HTTPOption
func WithBearerToken(token string) HTTPOption      // → headers["Authorization"] = "Bearer " + token
func WithBasicAuth(user, pass string) HTTPOption   // → headers["Authorization"] = "Basic " + base64

// NewHTTPChecker validates no Authorization conflict
```

In `Check()` method:
- Add all headers from `h.headers` to request
- HTTP 401/403 → return `NewClassifiedCheckError(msg, "auth_error", "auth_error")`

### 2.4 gRPC Checker (checks/grpc.go)

Add `metadata map[string]string` to GRPCChecker:

```go
type GRPCChecker struct {
    serviceName   string
    tlsEnabled    bool
    tlsSkipVerify bool
    metadata      map[string]string  // NEW: custom metadata
}

// New options:
func WithMetadata(metadata map[string]string) GRPCOption
func WithGRPCBearerToken(token string) GRPCOption     // → metadata["authorization"] = "Bearer " + token
func WithGRPCBasicAuth(user, pass string) GRPCOption   // → metadata["authorization"] = "Basic " + base64

// NewGRPCChecker validates no authorization conflict
```

In `Check()` method:
- Add metadata via `metadata.AppendToOutgoingContext(ctx, key, value)`
- gRPC UNAUTHENTICATED/PERMISSION_DENIED → return `NewClassifiedCheckError(msg, "auth_error", "auth_error")`

### 2.5 Factories (checks/factories.go)

Update `newHTTPFromConfig` and `newGRPCFromConfig` to pass auth params.

### 2.6 Validation

In checker constructors, validate:
- No more than one of: bearerToken, basicAuth, custom Authorization header
- If both bearerToken and custom headers with "Authorization" → error

### Deliverables

- [x] Updated `options.go` with auth DependencyConfig fields and option functions
- [x] Updated `checks/http.go` with headers support, bearer/basic helpers, 401/403 classification
- [x] Updated `checks/grpc.go` with metadata support, bearer/basic helpers, UNAUTHENTICATED classification
- [x] Updated `checks/factories.go` to wire auth params
- [x] Unit tests for auth options, validation, conflict detection
- [x] `make test` and `make lint` pass

---

## Phase 3: Java SDK Implementation

**Goal**: Add authentication to Java HTTP and gRPC checkers.

### 3.1 HttpHealthChecker.java

Add to Builder:

```java
public static final class Builder {
    // ... existing ...
    public Builder headers(Map<String, String> headers)
    public Builder bearerToken(String token)
    public Builder basicAuth(String username, String password)
    public HttpHealthChecker build()  // validates conflicts
}
```

In `check()` method:
- Add headers to HttpRequest
- HTTP 401/403 → throw `UnhealthyException(msg, "auth_error")`

### 3.2 GrpcHealthChecker.java

Add to Builder:

```java
public static final class Builder {
    // ... existing ...
    public Builder metadata(Map<String, String> metadata)
    public Builder bearerToken(String token)
    public Builder basicAuth(String username, String password)
    public GrpcHealthChecker build()  // validates conflicts
}
```

In `check()` method:
- Add metadata to gRPC call via `Metadata` object
- UNAUTHENTICATED/PERMISSION_DENIED → throw `UnhealthyException(msg, "auth_error")`

### 3.3 Spring Boot Starter

#### DependencyProperties.java — add fields:

```java
// HTTP auth
private Map<String, String> httpHeaders;
private String httpBearerToken;
private String httpBasicUsername;
private String httpBasicPassword;

// gRPC auth
private Map<String, String> grpcMetadata;
private String grpcBearerToken;
private String grpcBasicUsername;
private String grpcBasicPassword;
```

#### DepHealthAutoConfiguration.java — wire new properties

#### application.yml example:

```yaml
dephealth:
  dependencies:
    payment-api:
      type: http
      url: http://payment.svc:8080
      critical: true
      http-bearer-token: ${PAYMENT_TOKEN}
      # OR
      http-basic-username: ${PAYMENT_USER}
      http-basic-password: ${PAYMENT_PASS}
      # OR
      http-headers:
        X-API-Key: ${API_KEY}
```

### Deliverables

- [x] Updated `HttpHealthChecker.java` with auth support
- [x] Updated `GrpcHealthChecker.java` with auth support
- [x] Updated `DependencyProperties.java` with auth config fields
- [x] Updated `DepHealthAutoConfiguration.java` to wire auth
- [x] Unit tests for auth, validation, conflicts
- [x] `make test` and `make lint` pass

---

## Phase 4: Python SDK Implementation

**Goal**: Add authentication to Python HTTP and gRPC checkers.

### 4.1 HTTPChecker (checks/http.py)

```python
class HTTPChecker:
    def __init__(
        self,
        health_path: str = "/health",
        timeout: float = 5.0,
        tls: bool = False,
        tls_skip_verify: bool = False,
        headers: dict[str, str] | None = None,       # NEW
        bearer_token: str | None = None,              # NEW
        basic_auth: tuple[str, str] | None = None,    # NEW: (username, password)
    ):
        # Validate: no more than one auth method
        # Merge headers + auth helper → self._headers
```

In `check()` method:
- Pass headers to `session.get(url, headers=self._headers)`
- HTTP 401/403 → raise `CheckAuthError("auth_error")`

### 4.2 GRPCChecker (checks/grpc.py)

```python
class GRPCChecker:
    def __init__(
        self,
        service_name: str = "",
        timeout: float = 5.0,
        tls: bool = False,
        tls_skip_verify: bool = False,
        metadata: dict[str, str] | None = None,       # NEW
        bearer_token: str | None = None,               # NEW
        basic_auth: tuple[str, str] | None = None,     # NEW: (username, password)
    ):
        # Validate: no more than one auth method
        # Merge metadata + auth helper → self._metadata
```

In `check()` method:
- Pass metadata to gRPC call: `stub.Check(request, timeout=..., metadata=metadata_tuples)`
- UNAUTHENTICATED/PERMISSION_DENIED → raise `CheckAuthError("auth_error")`

### 4.3 CheckAuthError

Already exists in `checker.py` with `status_category="auth_error"`. Verify it has correct `status_detail`.

### Deliverables

- [x] Updated `checks/http.py` with auth support
- [x] Updated `checks/grpc.py` with auth support
- [x] Verify/update `CheckAuthError` classification
- [x] Unit tests for auth, validation, conflicts
- [x] `make test` and `make lint` pass

---

## Phase 5: C# SDK Implementation

**Goal**: Add authentication to C# HTTP and gRPC checkers.

### 5.1 HttpChecker.cs

```csharp
public sealed class HttpChecker : IHealthChecker
{
    // ... existing fields ...
    private readonly IReadOnlyDictionary<string, string> _headers;  // NEW

    public HttpChecker(
        string healthPath = DefaultHealthPath,
        bool tlsEnabled = false,
        bool tlsSkipVerify = false,
        IDictionary<string, string>? headers = null,         // NEW
        string? bearerToken = null,                          // NEW
        string? basicAuthUsername = null,                     // NEW
        string? basicAuthPassword = null)                    // NEW
    {
        // Validate conflicts, merge into _headers
    }
}
```

In `CheckAsync()` method:
- Add headers to HttpRequestMessage
- HTTP 401/403 → throw `UnhealthyException(msg, "auth_error")`

### 5.2 GrpcChecker.cs

```csharp
public sealed class GrpcChecker : IHealthChecker
{
    // ... existing fields ...
    private readonly IReadOnlyDictionary<string, string> _metadata;  // NEW

    public GrpcChecker(
        bool tlsEnabled = false,
        IDictionary<string, string>? metadata = null,         // NEW
        string? bearerToken = null,                           // NEW
        string? basicAuthUsername = null,                      // NEW
        string? basicAuthPassword = null)                     // NEW
    {
        // Validate conflicts, merge into _metadata
    }
}
```

In `CheckAsync()` method:
- Add metadata to gRPC call via `Metadata` object
- UNAUTHENTICATED/PERMISSION_DENIED → throw `UnhealthyException(msg, "auth_error")`

### Deliverables

- [x] Updated `HttpChecker.cs` with auth support
- [x] Updated `GrpcChecker.cs` with auth support
- [x] Unit tests for auth, validation, conflicts
- [x] `make test` and `make lint` pass

---

## Phase 6: Conformance Tests

**Goal**: Add cross-SDK conformance tests verifying authentication behavior.

### Test scenarios

#### Scenario 1: HTTP Bearer Token

- Start test HTTP server that requires `Authorization: Bearer test-token`
- Configure HTTP checker with `bearerToken: "test-token"`
- Verify: health = 1 (healthy)

#### Scenario 2: HTTP Basic Auth

- Start test HTTP server that requires Basic Auth (admin:password)
- Configure HTTP checker with `basicAuth: {username: "admin", password: "password"}`
- Verify: health = 1 (healthy)

#### Scenario 3: HTTP Custom Headers

- Start test HTTP server that requires `X-API-Key: my-key`
- Configure HTTP checker with `headers: {"X-API-Key": "my-key"}`
- Verify: health = 1 (healthy)

#### Scenario 4: HTTP Auth Error (401)

- Start test HTTP server that requires auth
- Configure HTTP checker WITHOUT auth
- Verify: health = 0, status_detail = "auth_error"

#### Scenario 5: HTTP Auth Error (403)

- Start test HTTP server that returns 403 for wrong token
- Configure HTTP checker with wrong token
- Verify: health = 0, status_detail = "auth_error"

#### Scenario 6: gRPC Bearer Token

- Start test gRPC server that checks `authorization` metadata
- Configure gRPC checker with `bearerToken: "test-token"`
- Verify: health = 1 (healthy)

#### Scenario 7: gRPC Auth Error (UNAUTHENTICATED)

- Start test gRPC server that requires auth
- Configure gRPC checker WITHOUT auth
- Verify: health = 0, status_detail = "auth_error"

### Deliverables

- [ ] Test HTTP server with auth support (in conformance test infra)
- [ ] Test gRPC server with auth support (in conformance test infra)
- [ ] YAML scenarios for all 7 test cases
- [ ] All 4 SDKs pass conformance tests
- [ ] `make test` in conformance/ passes

---

## Phase 7: Documentation and Release

**Goal**: Update documentation, bump versions, create releases.

### Documentation

- [ ] Update `spec/check-behavior.md` (done in Phase 1)
- [ ] Update `spec/config-contract.md` (done in Phase 1)
- [ ] Update SDK READMEs with auth examples
- [ ] Update `docs/` guides with auth configuration examples
- [ ] Add auth section to Spring Boot Starter docs
- [ ] EN + RU versions

### Version bump

- [ ] All SDKs → v0.5.0
- [ ] Update version constants in code
- [ ] Update CHANGELOG.md

### Release

- [ ] Merge to master
- [ ] Create per-SDK tags: sdk-go/v0.5.0, sdk-java/v0.5.0, sdk-python/v0.5.0, sdk-csharp/v0.5.0
- [ ] GitHub Releases for each SDK
- [ ] Publish: PyPI, Maven Central
- [ ] NuGet: TODO (no make publish target)

### Deliverables

- [ ] All documentation updated (EN + RU)
- [ ] Version bumped to 0.5.0
- [ ] All SDKs tagged and released

---

## API Summary (all SDKs)

### Go SDK

```go
// HTTP
dephealth.HTTP("payment-api",
    dephealth.FromURL("http://payment.svc:8080"),
    dephealth.Critical(true),
    dephealth.WithHTTPBearerToken("eyJhbG..."),
    // OR
    dephealth.WithHTTPBasicAuth("admin", "secret"),
    // OR
    dephealth.WithHTTPHeaders(map[string]string{"X-API-Key": "xxx"}),
)

// gRPC
dephealth.GRPC("backend",
    dephealth.FromURL("grpc://backend.svc:9090"),
    dephealth.Critical(true),
    dephealth.WithGRPCBearerToken("eyJhbG..."),
    // OR
    dephealth.WithGRPCMetadata(map[string]string{"authorization": "Bearer xxx"}),
)
```

### Java SDK

```java
// HTTP
HttpHealthChecker.builder()
    .healthPath("/ready")
    .bearerToken("eyJhbG...")
    // OR
    .basicAuth("admin", "secret")
    // OR
    .headers(Map.of("X-API-Key", "xxx"))
    .build()

// gRPC
GrpcHealthChecker.builder()
    .bearerToken("eyJhbG...")
    // OR
    .metadata(Map.of("authorization", "Bearer xxx"))
    .build()
```

### Python SDK

```python
# HTTP
HTTPChecker(
    health_path="/ready",
    bearer_token="eyJhbG...",
    # OR
    basic_auth=("admin", "secret"),
    # OR
    headers={"X-API-Key": "xxx"},
)

# gRPC
GRPCChecker(
    bearer_token="eyJhbG...",
    # OR
    metadata={"authorization": "Bearer xxx"},
)
```

### C# SDK

```csharp
// HTTP
new HttpChecker(
    healthPath: "/ready",
    bearerToken: "eyJhbG...",
    // OR
    basicAuthUsername: "admin", basicAuthPassword: "secret",
    // OR
    headers: new Dictionary<string, string> { ["X-API-Key"] = "xxx" }
)

// gRPC
new GrpcChecker(
    bearerToken: "eyJhbG...",
    // OR
    metadata: new Dictionary<string, string> { ["authorization"] = "Bearer xxx" }
)
```

### Spring Boot YAML

```yaml
dephealth:
  dependencies:
    payment-api:
      type: http
      url: http://payment.svc:8080
      critical: true
      http-bearer-token: ${PAYMENT_TOKEN}
      # OR
      http-basic-username: ${PAYMENT_USER}
      http-basic-password: ${PAYMENT_PASS}
      # OR
      http-headers:
        X-API-Key: ${API_KEY}
    grpc-backend:
      type: grpc
      url: grpc://backend.svc:9090
      critical: true
      grpc-bearer-token: ${GRPC_TOKEN}
      # OR
      grpc-metadata:
        authorization: "Bearer ${GRPC_TOKEN}"
```

---

## Estimated effort per phase

| Phase | Scope | Complexity |
|-------|-------|------------|
| Phase 1: Spec | 2 files | Low |
| Phase 2: Go SDK | 4 files + tests | Medium |
| Phase 3: Java SDK | 4 files + starter + tests | Medium-High |
| Phase 4: Python SDK | 2 files + tests | Medium |
| Phase 5: C# SDK | 2 files + tests | Medium |
| Phase 6: Conformance | test infra + 7 scenarios | High |
| Phase 7: Docs + Release | docs + version + tags | Medium |
