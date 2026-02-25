*[Русская версия](api-reference.ru.md)*

# API Reference

Complete reference of all public symbols in the dephealth Go SDK.

**Module path:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth`

## Package `dephealth`

The core package providing dependency health monitoring with Prometheus metrics.

### Constants

#### Version

```go
const Version = "0.8.0"
```

SDK version used in User-Agent headers.

#### Default Values

| Constant | Value | Description |
| --- | --- | --- |
| `DefaultCheckInterval` | `15s` | Default interval between health checks |
| `DefaultTimeout` | `5s` | Default timeout for a single check |
| `DefaultInitialDelay` | `5s` | Default initial delay before first check |
| `DefaultFailureThreshold` | `1` | Failures before marking unhealthy |
| `DefaultSuccessThreshold` | `1` | Successes before marking healthy |

#### Range Bounds

| Constant | Value |
| --- | --- |
| `MinCheckInterval` | `1s` |
| `MaxCheckInterval` | `10m` |
| `MinTimeout` | `100ms` |
| `MaxTimeout` | `30s` |
| `MinInitialDelay` | `0` |
| `MaxInitialDelay` | `5m` |
| `MinThreshold` | `1` |
| `MaxThreshold` | `10` |

#### DependencyType

```go
type DependencyType string
```

| Constant | Value |
| --- | --- |
| `TypeHTTP` | `"http"` |
| `TypeGRPC` | `"grpc"` |
| `TypeTCP` | `"tcp"` |
| `TypePostgres` | `"postgres"` |
| `TypeMySQL` | `"mysql"` |
| `TypeRedis` | `"redis"` |
| `TypeAMQP` | `"amqp"` |
| `TypeKafka` | `"kafka"` |
| `TypeLDAP` | `"ldap"` |

#### StatusCategory

```go
type StatusCategory string
```

| Constant | Value | Description |
| --- | --- | --- |
| `StatusOK` | `"ok"` | Healthy |
| `StatusTimeout` | `"timeout"` | Check timed out |
| `StatusConnectionError` | `"connection_error"` | Connection refused or reset |
| `StatusDNSError` | `"dns_error"` | DNS resolution failed |
| `StatusAuthError` | `"auth_error"` | Authentication/authorization failure |
| `StatusTLSError` | `"tls_error"` | TLS handshake failure |
| `StatusUnhealthy` | `"unhealthy"` | Reachable but unhealthy |
| `StatusError` | `"error"` | Other error |
| `StatusUnknown` | `"unknown"` | Not yet checked |

#### Sentinel Errors

```go
var (
    ErrTimeout           = errors.New("health check timeout")
    ErrConnectionRefused = errors.New("connection refused")
    ErrUnhealthy         = errors.New("dependency unhealthy")
    ErrAlreadyStarted    = errors.New("scheduler already started")
    ErrNotStarted        = errors.New("scheduler not started")
    ErrEndpointNotFound  = errors.New("endpoint not found")
)
```

#### Other Variables

```go
var ValidTypes map[DependencyType]bool          // Map of all valid dependency types
var AllStatusCategories []StatusCategory         // All 8 status categories (excludes StatusUnknown)
var DefaultPorts map[string]string               // Default ports by URL scheme
```

### Interfaces

#### HealthChecker

```go
type HealthChecker interface {
    Check(ctx context.Context, endpoint Endpoint) error
    Type() string
}
```

Interface for dependency health checks. `Check()` returns `nil` if healthy
or an error describing the failure. `Type()` returns the dependency type
string (e.g., `"http"`).

#### ClassifiedError

```go
type ClassifiedError interface {
    error
    StatusCategory() StatusCategory
    StatusDetail() string
}
```

Error with status classification. Health checkers return errors implementing
this interface to provide precise `status` and `detail` values for metrics.

### Types

#### DepHealth

```go
type DepHealth struct{ /* private */ }
```

Main SDK entry point. Combines metrics export and check scheduling.

| Method | Signature | Description |
| --- | --- | --- |
| `Start` | `(ctx context.Context) error` | Start periodic health checks |
| `Stop` | `()` | Stop all checks and clean up |
| `Health` | `() map[string]bool` | Quick health map (key: `dep/host:port`) |
| `HealthDetails` | `() map[string]EndpointStatus` | Detailed status per endpoint |
| `AddEndpoint` | `(depName string, depType DependencyType, critical bool, ep Endpoint, checker HealthChecker) error` | Add endpoint at runtime |
| `RemoveEndpoint` | `(depName, host, port string) error` | Remove endpoint at runtime |
| `UpdateEndpoint` | `(depName, oldHost, oldPort string, newEp Endpoint, checker HealthChecker) error` | Replace endpoint atomically |

#### Endpoint

```go
type Endpoint struct {
    Host   string
    Port   string
    Labels map[string]string
}
```

Network endpoint for a dependency.

#### Dependency

```go
type Dependency struct {
    Name      string
    Type      DependencyType
    Critical  *bool
    Endpoints []Endpoint
    Config    CheckConfig
}
```

| Method | Signature | Description |
| --- | --- | --- |
| `Validate` | `() error` | Validate the dependency configuration |

#### EndpointStatus

```go
type EndpointStatus struct {
    Healthy       *bool              // nil = unknown (before first check)
    Status        StatusCategory
    Detail        string             // e.g. "http_503", "grpc_not_serving"
    Latency       time.Duration
    Type          DependencyType
    Name          string
    Host          string
    Port          string
    Critical      bool
    LastCheckedAt time.Time          // zero before first check
    Labels        map[string]string
}
```

Detailed health check state for a single endpoint.

| Method | Signature | Description |
| --- | --- | --- |
| `LatencyMillis` | `() float64` | Latency in milliseconds |
| `MarshalJSON` | `() ([]byte, error)` | Custom JSON (latency as `latency_ms`) |
| `UnmarshalJSON` | `(data []byte) error` | Custom JSON unmarshaling |

JSON serialization: `Latency` serialized as `latency_ms` (float, milliseconds).
`LastCheckedAt` serialized as `null` when zero.

#### CheckConfig

```go
type CheckConfig struct {
    Interval         time.Duration
    Timeout          time.Duration
    InitialDelay     time.Duration
    FailureThreshold int
    SuccessThreshold int
}
```

| Method/Function | Signature | Description |
| --- | --- | --- |
| `DefaultCheckConfig` | `() CheckConfig` | Returns config with default values |
| `Validate` | `() error` | Validate ranges |

#### CheckResult

```go
type CheckResult struct {
    Category StatusCategory
    Detail   string
}
```

Classification of a health check outcome.

#### DependencyConfig

```go
type DependencyConfig struct {
    URL               string
    Host              string
    Port              string
    Critical          *bool
    Interval          time.Duration
    Timeout           time.Duration
    Labels            map[string]string

    // HTTP options
    HTTPHealthPath    string
    HTTPTLS           *bool
    HTTPTLSSkipVerify *bool
    HTTPHeaders       map[string]string
    HTTPBearerToken   string
    HTTPBasicUser     string
    HTTPBasicPass     string

    // gRPC options
    GRPCServiceName   string
    GRPCTLS           *bool
    GRPCTLSSkipVerify *bool
    GRPCMetadata      map[string]string
    GRPCBearerToken   string
    GRPCBasicUser     string
    GRPCBasicPass     string

    // Database options
    PostgresQuery     string
    MySQLQuery        string
    RedisPassword     string
    RedisDB           *int
    AMQPURL           string

    // LDAP options
    LDAPCheckMethod   string
    LDAPBindDN        string
    LDAPBindPassword  string
    LDAPBaseDN        string
    LDAPSearchFilter  string
    LDAPSearchScope   string
    LDAPStartTLS      *bool
    LDAPTLSSkipVerify *bool
    LDAPUseTLS        bool
}
```

Configuration for a single dependency. Populated by `DependencyOption`
functions and passed to checker factories.

#### ParsedConnection

```go
type ParsedConnection struct {
    Host     string
    Port     string
    ConnType DependencyType
}
```

Result of parsing a URL or connection string.

#### ClassifiedCheckError

```go
type ClassifiedCheckError struct {
    Category StatusCategory
    Detail   string
    Cause    error
}
```

Ready-to-use `ClassifiedError` implementation.

| Method | Signature | Description |
| --- | --- | --- |
| `Error` | `() string` | Returns the cause error message |
| `Unwrap` | `() error` | Returns the cause for `errors.Is`/`errors.As` |
| `StatusCategory` | `() StatusCategory` | Returns the status category |
| `StatusDetail` | `() string` | Returns the detail string |

#### InvalidLabelError

```go
type InvalidLabelError struct {
    Label string
}
```

| Method | Signature | Description |
| --- | --- | --- |
| `Error` | `() string` | Returns error message with the invalid label name |

### Functions

#### Constructor

```go
func New(name string, group string, opts ...Option) (*DepHealth, error)
```

Creates a `DepHealth` instance. `name` and `group` are required (via API
argument or env vars `DEPHEALTH_NAME`, `DEPHEALTH_GROUP`). Returns an error
if validation fails.

#### Dependency Factories

Each factory registers a dependency and returns an `Option` for `New()`.

```go
func HTTP(name string, opts ...DependencyOption) Option
func GRPC(name string, opts ...DependencyOption) Option
func TCP(name string, opts ...DependencyOption) Option
func Postgres(name string, opts ...DependencyOption) Option
func MySQL(name string, opts ...DependencyOption) Option
func Redis(name string, opts ...DependencyOption) Option
func AMQP(name string, opts ...DependencyOption) Option
func Kafka(name string, opts ...DependencyOption) Option
func LDAP(name string, opts ...DependencyOption) Option
```

#### AddDependency

```go
func AddDependency(name string, depType DependencyType, checker HealthChecker, opts ...DependencyOption) Option
```

Registers an arbitrary dependency with a custom `HealthChecker`. Used by
contrib modules and custom checkers.

#### URL Parsers

```go
func ParseURL(rawURL string) ([]ParsedConnection, error)
```

Parses a URL into host/port/type. Supported schemes: `http`, `https`,
`grpc`, `tcp`, `postgresql`, `postgres`, `mysql`, `redis`, `rediss`,
`amqp`, `amqps`, `kafka`, `ldap`, `ldaps`. Kafka multi-host URLs
(`kafka://host1:9092,host2:9092`) return multiple connections.

```go
func ParseConnectionString(connStr string) (string, string, error)
```

Parses `Key=Value;Key=Value` connection strings. Returns `(host, port, error)`.

```go
func ParseJDBC(jdbcURL string) ([]ParsedConnection, error)
```

Parses JDBC URLs: `jdbc:postgresql://host:port/db`,
`jdbc:mysql://host:port/db`.

```go
func ParseParams(host, port string) (Endpoint, error)
```

Creates an `Endpoint` from explicit host and port. Validates port range
(1-65535), handles IPv6 addresses in brackets.

#### Validators

```go
func ValidateName(name string) error
```

Validates name/group: `[a-z][a-z0-9-]*`, 1-63 chars.

```go
func ValidateLabelName(name string) error
```

Validates custom label name: `[a-zA-Z_][a-zA-Z0-9_]*`. Rejects reserved
labels: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`.

```go
func ValidateLabels(labels map[string]string) error
```

Validates all custom labels.

#### Utilities

```go
func BoolToYesNo(v bool) string
```

Converts `bool` to `"yes"` / `"no"` for the `critical` label.

#### Registry

```go
func RegisterCheckerFactory(depType DependencyType, factory CheckerFactory)
```

Registers a checker factory for the specified type. Called from checker
package `init()` functions.

### Option Types

#### Option

```go
type Option func(*config) error
```

Functional option for `New()`.

#### DependencyOption

```go
type DependencyOption func(*DependencyConfig)
```

Option for a specific dependency.

#### CheckerFactory

```go
type CheckerFactory func(dc *DependencyConfig) HealthChecker
```

Function that creates a checker from `DependencyConfig`.

### Global Options

Passed to `New()`, apply to all dependencies unless overridden per-dependency.

| Function | Signature | Description |
| --- | --- | --- |
| `WithCheckInterval` | `(d time.Duration) Option` | Global check interval (default 15s) |
| `WithTimeout` | `(d time.Duration) Option` | Global check timeout (default 5s) |
| `WithRegisterer` | `(r prometheus.Registerer) Option` | Custom Prometheus registerer |
| `WithLogger` | `(l *slog.Logger) Option` | Logger for SDK operations |

### Dependency Options

#### Common

| Function | Signature | Description |
| --- | --- | --- |
| `FromURL` | `(rawURL string) DependencyOption` | Parse host/port from URL |
| `FromParams` | `(host, port string) DependencyOption` | Set host/port explicitly |
| `Critical` | `(v bool) DependencyOption` | Mark as critical (required) |
| `WithLabel` | `(key, value string) DependencyOption` | Add custom Prometheus label |
| `CheckInterval` | `(d time.Duration) DependencyOption` | Per-dependency check interval |
| `Timeout` | `(d time.Duration) DependencyOption` | Per-dependency timeout |

#### HTTP

| Function | Signature | Description |
| --- | --- | --- |
| `WithHTTPHealthPath` | `(path string) DependencyOption` | Health check path (default `/health`) |
| `WithHTTPTLS` | `(enabled bool) DependencyOption` | Enable HTTPS (auto for `https://`) |
| `WithHTTPTLSSkipVerify` | `(skip bool) DependencyOption` | Skip TLS cert verification |
| `WithHTTPHeaders` | `(headers map[string]string) DependencyOption` | Custom HTTP headers |
| `WithHTTPBearerToken` | `(token string) DependencyOption` | Bearer token auth |
| `WithHTTPBasicAuth` | `(username, password string) DependencyOption` | Basic auth |

#### gRPC

| Function | Signature | Description |
| --- | --- | --- |
| `WithGRPCServiceName` | `(name string) DependencyOption` | Service name (empty = server health) |
| `WithGRPCTLS` | `(enabled bool) DependencyOption` | Enable TLS |
| `WithGRPCTLSSkipVerify` | `(skip bool) DependencyOption` | Skip TLS cert verification |
| `WithGRPCMetadata` | `(metadata map[string]string) DependencyOption` | Custom gRPC metadata |
| `WithGRPCBearerToken` | `(token string) DependencyOption` | Bearer token auth |
| `WithGRPCBasicAuth` | `(username, password string) DependencyOption` | Basic auth |

#### PostgreSQL

| Function | Signature | Description |
| --- | --- | --- |
| `WithPostgresQuery` | `(query string) DependencyOption` | Health check query (default `SELECT 1`) |

#### MySQL

| Function | Signature | Description |
| --- | --- | --- |
| `WithMySQLQuery` | `(query string) DependencyOption` | Health check query (default `SELECT 1`) |

#### Redis

| Function | Signature | Description |
| --- | --- | --- |
| `WithRedisPassword` | `(password string) DependencyOption` | Password (standalone mode) |
| `WithRedisDB` | `(db int) DependencyOption` | Database number (standalone mode) |

#### AMQP

| Function | Signature | Description |
| --- | --- | --- |
| `WithAMQPURL` | `(url string) DependencyOption` | Full AMQP URL |

#### LDAP

| Function | Signature | Description |
| --- | --- | --- |
| `WithLDAPCheckMethod` | `(method string) DependencyOption` | Check method: `anonymous_bind`, `simple_bind`, `root_dse`, `search` |
| `WithLDAPBindDN` | `(dn string) DependencyOption` | DN for simple bind |
| `WithLDAPBindPassword` | `(password string) DependencyOption` | Password for simple bind |
| `WithLDAPBaseDN` | `(baseDN string) DependencyOption` | Base DN for search method |
| `WithLDAPSearchFilter` | `(filter string) DependencyOption` | LDAP search filter (default `(objectClass=*)`) |
| `WithLDAPSearchScope` | `(scope string) DependencyOption` | Search scope: `base`, `one`, `sub` |
| `WithLDAPStartTLS` | `(enabled bool) DependencyOption` | Use StartTLS (only with `ldap://`) |
| `WithLDAPTLSSkipVerify` | `(skip bool) DependencyOption` | Skip TLS certificate verification |

---

## Package `checks`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks`

Importing this package registers factories for **all 9 checker types**
via blank imports of sub-packages. Also provides backward-compatible
type aliases and constructor wrappers.

### Backward-Compatible Aliases (Deprecated)

All aliases below are **deprecated**. Use the sub-packages directly.

#### Type Aliases

| Alias | Target | Sub-package |
| --- | --- | --- |
| `TCPChecker` | `tcpcheck.Checker` | `checks/tcpcheck` |
| `HTTPChecker` | `httpcheck.Checker` | `checks/httpcheck` |
| `HTTPOption` | `httpcheck.Option` | `checks/httpcheck` |
| `GRPCChecker` | `grpccheck.Checker` | `checks/grpccheck` |
| `GRPCOption` | `grpccheck.Option` | `checks/grpccheck` |
| `PostgresChecker` | `pgcheck.Checker` | `checks/pgcheck` |
| `PostgresOption` | `pgcheck.Option` | `checks/pgcheck` |
| `MySQLChecker` | `mysqlcheck.Checker` | `checks/mysqlcheck` |
| `MySQLOption` | `mysqlcheck.Option` | `checks/mysqlcheck` |
| `RedisChecker` | `redischeck.Checker` | `checks/redischeck` |
| `RedisOption` | `redischeck.Option` | `checks/redischeck` |
| `AMQPChecker` | `amqpcheck.Checker` | `checks/amqpcheck` |
| `AMQPOption` | `amqpcheck.Option` | `checks/amqpcheck` |
| `KafkaChecker` | `kafkacheck.Checker` | `checks/kafkacheck` |

#### Constructor Wrappers

| Wrapper | Target |
| --- | --- |
| `NewTCPChecker` | `tcpcheck.New` |
| `NewHTTPChecker` | `httpcheck.New` |
| `NewGRPCChecker` | `grpccheck.New` |
| `NewPostgresChecker` | `pgcheck.New` |
| `NewMySQLChecker` | `mysqlcheck.New` |
| `NewRedisChecker` | `redischeck.New` |
| `NewAMQPChecker` | `amqpcheck.New` |
| `NewKafkaChecker` | `kafkacheck.New` |

#### Option Wrappers

| Wrapper | Target |
| --- | --- |
| `WithHealthPath` | `httpcheck.WithHealthPath` |
| `WithTLSEnabled` | `httpcheck.WithTLSEnabled` |
| `WithHTTPTLSSkipVerify` | `httpcheck.WithTLSSkipVerify` |
| `WithHeaders` | `httpcheck.WithHeaders` |
| `WithBearerToken` | `httpcheck.WithBearerToken` |
| `WithBasicAuth` | `httpcheck.WithBasicAuth` |
| `WithServiceName` | `grpccheck.WithServiceName` |
| `WithGRPCTLS` | `grpccheck.WithTLS` |
| `WithGRPCTLSSkipVerify` | `grpccheck.WithTLSSkipVerify` |
| `WithMetadata` | `grpccheck.WithMetadata` |
| `WithGRPCBearerToken` | `grpccheck.WithBearerToken` |
| `WithGRPCBasicAuth` | `grpccheck.WithBasicAuth` |
| `WithPostgresDB` | `pgcheck.WithDB` |
| `WithPostgresDSN` | `pgcheck.WithDSN` |
| `WithPostgresQuery` | `pgcheck.WithQuery` |
| `WithMySQLDB` | `mysqlcheck.WithDB` |
| `WithMySQLDSN` | `mysqlcheck.WithDSN` |
| `WithMySQLQuery` | `mysqlcheck.WithQuery` |
| `WithRedisClient` | `redischeck.WithClient` |
| `WithRedisPassword` | `redischeck.WithPassword` |
| `WithRedisDB` | `redischeck.WithDB` |
| `WithAMQPURL` | `amqpcheck.WithURL` |

---

## Sub-packages (`checks/*`)

Each sub-package provides one checker implementation. Importing a
sub-package registers its factory via `init()`.

### `checks/httpcheck`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck`

HTTP health checker. Sends GET requests, succeeds on 2xx response.
Follows redirects automatically.

```go
type Checker struct{ /* private */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // returns "http"
```

| Option | Signature | Description |
| --- | --- | --- |
| `WithHealthPath` | `(path string) Option` | Health check path (default `/health`) |
| `WithTLSEnabled` | `(enabled bool) Option` | Enable HTTPS |
| `WithTLSSkipVerify` | `(skip bool) Option` | Skip TLS cert verification |
| `WithHeaders` | `(headers map[string]string) Option` | Custom HTTP headers |
| `WithBearerToken` | `(token string) Option` | Bearer token auth |
| `WithBasicAuth` | `(username, password string) Option` | Basic auth |

**Error classification:**

| Condition | Category | Detail |
| --- | --- | --- |
| Status 401/403 | `auth_error` | `auth_error` |
| Status non-2xx | `unhealthy` | `http_<code>` (e.g., `http_503`) |

### `checks/grpccheck`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck`

gRPC Health Checking Protocol checker. Creates a new connection per check,
sends `Health/Check`, and closes. Uses `passthrough:///` resolver.

```go
type Checker struct{ /* private */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // returns "grpc"
```

| Option | Signature | Description |
| --- | --- | --- |
| `WithServiceName` | `(name string) Option` | Service name (empty = overall health) |
| `WithTLS` | `(enabled bool) Option` | Enable TLS |
| `WithTLSSkipVerify` | `(skip bool) Option` | Skip TLS cert verification |
| `WithMetadata` | `(md map[string]string) Option` | Custom gRPC metadata |
| `WithBearerToken` | `(token string) Option` | Bearer token auth |
| `WithBasicAuth` | `(username, password string) Option` | Basic auth |

**Error classification:**

| Condition | Category | Detail |
| --- | --- | --- |
| UNAUTHENTICATED / PERMISSION_DENIED | `auth_error` | `auth_error` |
| NOT_SERVING | `unhealthy` | `grpc_not_serving` |
| Other non-SERVING | `unhealthy` | `grpc_unknown` |

### `checks/tcpcheck`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/tcpcheck`

TCP connection checker. Establishes a TCP connection and immediately
closes it. No data is sent or received.

```go
type Checker struct{}

func New() *Checker
func NewFromConfig(_ *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // returns "tcp"
```

No checker-specific options.

### `checks/pgcheck`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck`

PostgreSQL health checker. Supports standalone mode (new connection per
check) and pool mode (existing `*sql.DB`). Uses `pgx` driver.

```go
type Checker struct{ /* private */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // returns "postgres"
```

| Option | Signature | Description |
| --- | --- | --- |
| `WithDB` | `(db *sql.DB) Option` | Use existing connection pool |
| `WithDSN` | `(dsn string) Option` | Custom DSN (standalone mode) |
| `WithQuery` | `(query string) Option` | Health check query (default `SELECT 1`) |

**Error classification:**

| Condition | Category | Detail |
| --- | --- | --- |
| SQLSTATE 28000 / 28P01 | `auth_error` | `auth_error` |

### `checks/mysqlcheck`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck`

MySQL health checker. Supports standalone mode and pool mode.
Uses `go-sql-driver/mysql`.

```go
type Checker struct{ /* private */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // returns "mysql"
```

| Option | Signature | Description |
| --- | --- | --- |
| `WithDB` | `(db *sql.DB) Option` | Use existing connection pool |
| `WithDSN` | `(dsn string) Option` | Custom DSN (standalone mode) |
| `WithQuery` | `(query string) Option` | Health check query (default `SELECT 1`) |

**Helper function:**

```go
func URLToDSN(rawURL string) string
```

Converts `mysql://user:pass@host:3306/db` to `go-sql-driver/mysql` DSN
format: `user:pass@tcp(host:3306)/db`. Returns empty string on parse error.

**Error classification:**

| Condition | Category | Detail |
| --- | --- | --- |
| Error 1045 / Access denied | `auth_error` | `auth_error` |

### `checks/redischeck`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck`

Redis health checker using `PING` command. Supports standalone mode
and pool mode. Uses `go-redis/v9`.

```go
type Checker struct{ /* private */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // returns "redis"
```

| Option | Signature | Description |
| --- | --- | --- |
| `WithClient` | `(client redis.Cmdable) Option` | Use existing Redis client |
| `WithPassword` | `(password string) Option` | Password (standalone mode) |
| `WithDB` | `(db int) Option` | Database number (standalone mode) |

**Error classification:**

| Condition | Category | Detail |
| --- | --- | --- |
| NOAUTH / WRONGPASS | `auth_error` | `auth_error` |
| Connection refused / timeout | `connection_error` | `connection_refused` |

### `checks/amqpcheck`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/amqpcheck`

AMQP health checker. Establishes an AMQP connection and immediately closes
it. Only standalone mode. Uses `amqp091-go`.

```go
type Checker struct{ /* private */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // returns "amqp"
```

| Option | Signature | Description |
| --- | --- | --- |
| `WithURL` | `(url string) Option` | Custom AMQP URL (default `amqp://guest:guest@host:port/`) |

**Error classification:**

| Condition | Category | Detail |
| --- | --- | --- |
| 403 / ACCESS_REFUSED | `auth_error` | `auth_error` |

### `checks/kafkacheck`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/kafkacheck`

Kafka health checker. Connects to the broker, requests metadata, and
closes. Only standalone mode. Uses `segmentio/kafka-go`.

```go
type Checker struct{}

func New() *Checker
func NewFromConfig(_ *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // returns "kafka"
```

No checker-specific options.

**Error classification:**

| Condition | Category | Detail |
| --- | --- | --- |
| No brokers in metadata | `unhealthy` | `no_brokers` |

### `checks/ldapcheck`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck`

LDAP health checker. Supports four check methods: anonymous bind, simple
bind, RootDSE query, and search. Handles LDAP, LDAPS, and StartTLS
connections. Uses `go-ldap/ldap/v3`.

```go
type Checker struct{ /* private */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // returns "ldap"
```

| Option | Signature | Description |
| --- | --- | --- |
| `WithConn` | `(conn *ldap.Conn) Option` | Use existing LDAP connection (pool mode) |
| `WithCheckMethod` | `(method CheckMethod) Option` | Check method (default `MethodRootDSE`) |
| `WithBindDN` | `(dn string) Option` | DN for simple bind |
| `WithBindPassword` | `(password string) Option` | Password for simple bind |
| `WithBaseDN` | `(baseDN string) Option` | Base DN for search |
| `WithSearchFilter` | `(filter string) Option` | Search filter (default `(objectClass=*)`) |
| `WithSearchScope` | `(scope SearchScope) Option` | Search scope (default `ScopeBase`) |
| `WithStartTLS` | `(enabled bool) Option` | Enable StartTLS |
| `WithUseTLS` | `(enabled bool) Option` | Use TLS (LDAPS) |
| `WithTLSSkipVerify` | `(skip bool) Option` | Skip TLS certificate verification |

**Constants:**

| Constant | Type | Value |
| --- | --- | --- |
| `MethodAnonymousBind` | `CheckMethod` | `"anonymous_bind"` |
| `MethodSimpleBind` | `CheckMethod` | `"simple_bind"` |
| `MethodRootDSE` | `CheckMethod` | `"root_dse"` |
| `MethodSearch` | `CheckMethod` | `"search"` |
| `ScopeBase` | `SearchScope` | `"base"` |
| `ScopeOne` | `SearchScope` | `"one"` |
| `ScopeSub` | `SearchScope` | `"sub"` |

**Error classification:**

| Condition | Category | Detail |
| --- | --- | --- |
| LDAP result code 49 (Invalid Credentials) | `auth_error` | `auth_error` |
| LDAP result code 50 (Insufficient Access Rights) | `auth_error` | `auth_error` |
| TLS/StartTLS handshake failure | `tls_error` | `tls_error` |
| LDAP server down/busy/unavailable | `unhealthy` | `unhealthy` |

**Validation errors (returned from `New` or `NewFromConfig`):**

| Condition | Error |
| --- | --- |
| `simple_bind` without `bindDN` or `bindPassword` | `"simple_bind requires bindDN and bindPassword"` |
| `search` without `baseDN` | `"search requires baseDN"` |
| `startTLS` with `useTLS` (LDAPS) | `"startTLS and useTLS are mutually exclusive"` |

---

## Contrib Packages

### `contrib/sqldb`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb`

Integration with `*sql.DB` connection pools for PostgreSQL and MySQL.

```go
func FromDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option
```

Creates an `Option` for monitoring PostgreSQL via an existing `*sql.DB`.
The caller must provide `FromURL` or `FromParams` to determine metric labels.

```go
func FromMySQLDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option
```

Creates an `Option` for monitoring MySQL via an existing `*sql.DB`.
The caller must provide `FromURL` or `FromParams` to determine metric labels.

### `contrib/redispool`

**Import:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool`

Integration with `*redis.Client` connection pool.

```go
func FromClient(name string, client *redis.Client, opts ...dephealth.DependencyOption) dephealth.Option
```

Creates an `Option` for monitoring Redis via an existing `*redis.Client`.
Host and port are automatically extracted from `client.Options().Addr`.
Additional `DependencyOption` values (`Critical`, `CheckInterval`, etc.)
can be provided.

---

## Dynamic Endpoint Management

Methods for adding, removing, and updating endpoints at runtime on a
running `DepHealth` instance. All methods are thread-safe.

### AddEndpoint

```go
func (dh *DepHealth) AddEndpoint(depName string, depType DependencyType,
    critical bool, ep Endpoint, checker HealthChecker) error
```

Adds a new endpoint to a running `DepHealth` instance. A health-check
goroutine starts immediately using the global check interval and timeout.

**Validation:** `depName` via `ValidateName()`, `depType` against `ValidTypes`,
`ep.Host` and `ep.Port` must be non-empty, `ep.Labels` via `ValidateLabels()`.

**Idempotent:** if an endpoint with the same `depName:host:port` key already
exists, returns `nil` without modification.

**Errors:**

| Condition | Error |
| --- | --- |
| Scheduler not started or already stopped | `ErrNotStarted` |
| Invalid dependency name | validation error |
| Unknown dependency type | `"unknown dependency type"` |
| Missing host or port | `"missing host/port for endpoint"` |
| Reserved label name | `InvalidLabelError` |

### RemoveEndpoint

```go
func (dh *DepHealth) RemoveEndpoint(depName, host, port string) error
```

Removes an endpoint from a running `DepHealth` instance. Cancels the
health-check goroutine and deletes all associated Prometheus metrics.

**Idempotent:** if no endpoint with the given key exists, returns `nil`.

**Errors:**

| Condition | Error |
| --- | --- |
| Scheduler not started | `ErrNotStarted` |

### UpdateEndpoint

```go
func (dh *DepHealth) UpdateEndpoint(depName, oldHost, oldPort string,
    newEp Endpoint, checker HealthChecker) error
```

Atomically replaces an existing endpoint with a new one. The old endpoint's
goroutine is cancelled and its metrics are deleted; a new goroutine is
started for the new endpoint.

**Validation:** `newEp.Host` and `newEp.Port` must be non-empty,
`newEp.Labels` via `ValidateLabels()`.

**Errors:**

| Condition | Error |
| --- | --- |
| Scheduler not started or already stopped | `ErrNotStarted` |
| Old endpoint not found | `ErrEndpointNotFound` |
| Missing new host or port | `"missing host/port for new endpoint"` |
| Reserved label name | `InvalidLabelError` |

---

## See Also

- [Getting Started](getting-started.md) — installation and first example
- [Checkers](checkers.md) — detailed checker guide with examples
- [Configuration](configuration.md) — all configuration options
- [Custom Checkers](custom-checkers.md) — implementing `HealthChecker`
- [Selective Imports](selective-imports.md) — reducing binary size
- [Metrics](metrics.md) — Prometheus metrics reference
- [Troubleshooting](troubleshooting.md) — common issues and solutions
