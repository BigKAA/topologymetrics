*[English version](api-reference.md)*

# Справочник API

Полный справочник всех публичных символов Go SDK dephealth.

**Путь модуля:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth`

## Пакет `dephealth`

Основной пакет для мониторинга состояния зависимостей с экспортом метрик Prometheus.

### Константы

#### Version

```go
const Version = "0.8.0"
```

Версия SDK, используется в заголовках User-Agent.

#### Значения по умолчанию

| Константа | Значение | Описание |
| --- | --- | --- |
| `DefaultCheckInterval` | `15s` | Интервал между проверками |
| `DefaultTimeout` | `5s` | Тайм-аут одной проверки |
| `DefaultInitialDelay` | `5s` | Начальная задержка перед первой проверкой |
| `DefaultFailureThreshold` | `1` | Количество сбоев для пометки unhealthy |
| `DefaultSuccessThreshold` | `1` | Количество успехов для пометки healthy |

#### Допустимые диапазоны

| Константа | Значение |
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

| Константа | Значение |
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

| Константа | Значение | Описание |
| --- | --- | --- |
| `StatusOK` | `"ok"` | Зависимость доступна |
| `StatusTimeout` | `"timeout"` | Тайм-аут проверки |
| `StatusConnectionError` | `"connection_error"` | Соединение отклонено или сброшено |
| `StatusDNSError` | `"dns_error"` | Ошибка DNS-разрешения |
| `StatusAuthError` | `"auth_error"` | Ошибка аутентификации/авторизации |
| `StatusTLSError` | `"tls_error"` | Ошибка TLS-рукопожатия |
| `StatusUnhealthy` | `"unhealthy"` | Доступна, но нездорова |
| `StatusError` | `"error"` | Прочие ошибки |
| `StatusUnknown` | `"unknown"` | Ещё не проверялась |

#### Сигнальные ошибки

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

#### Прочие переменные

```go
var ValidTypes map[DependencyType]bool          // Карта допустимых типов зависимостей
var AllStatusCategories []StatusCategory         // Все 8 категорий статуса (без StatusUnknown)
var DefaultPorts map[string]string               // Порты по умолчанию для URL-схем
```

### Интерфейсы

#### HealthChecker

```go
type HealthChecker interface {
    Check(ctx context.Context, endpoint Endpoint) error
    Type() string
}
```

Интерфейс для проверки состояния зависимости. `Check()` возвращает `nil`,
если зависимость здорова, или ошибку с описанием проблемы. `Type()`
возвращает строку типа зависимости (например, `"http"`).

#### ClassifiedError

```go
type ClassifiedError interface {
    error
    StatusCategory() StatusCategory
    StatusDetail() string
}
```

Ошибка с классификацией статуса. Чекеры возвращают ошибки, реализующие
этот интерфейс, для точного указания значений `status` и `detail` в метриках.

### Типы

#### DepHealth

```go
type DepHealth struct{ /* приватные поля */ }
```

Главная точка входа SDK. Объединяет экспорт метрик и планирование проверок.

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `Start` | `(ctx context.Context) error` | Запустить периодические проверки |
| `Stop` | `()` | Остановить проверки и освободить ресурсы |
| `Health` | `() map[string]bool` | Быстрая карта здоровья (ключ: `dep/host:port`) |
| `HealthDetails` | `() map[string]EndpointStatus` | Детальный статус по каждому эндпоинту |
| `AddEndpoint` | `(depName string, depType DependencyType, critical bool, ep Endpoint, checker HealthChecker) error` | Добавить эндпоинт в рантайме |
| `RemoveEndpoint` | `(depName, host, port string) error` | Удалить эндпоинт в рантайме |
| `UpdateEndpoint` | `(depName, oldHost, oldPort string, newEp Endpoint, checker HealthChecker) error` | Заменить эндпоинт атомарно |

#### Endpoint

```go
type Endpoint struct {
    Host   string
    Port   string
    Labels map[string]string
}
```

Сетевой эндпоинт зависимости.

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

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `Validate` | `() error` | Валидация конфигурации зависимости |

#### EndpointStatus

```go
type EndpointStatus struct {
    Healthy       *bool              // nil = неизвестно (до первой проверки)
    Status        StatusCategory
    Detail        string             // например "http_503", "grpc_not_serving"
    Latency       time.Duration
    Type          DependencyType
    Name          string
    Host          string
    Port          string
    Critical      bool
    LastCheckedAt time.Time          // нулевое значение до первой проверки
    Labels        map[string]string
}
```

Детальное состояние проверки для одного эндпоинта.

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `LatencyMillis` | `() float64` | Задержка в миллисекундах |
| `MarshalJSON` | `() ([]byte, error)` | JSON-сериализация (latency как `latency_ms`) |
| `UnmarshalJSON` | `(data []byte) error` | JSON-десериализация |

JSON-сериализация: `Latency` сериализуется как `latency_ms` (float, миллисекунды).
`LastCheckedAt` сериализуется как `null`, если значение нулевое.

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

| Метод/Функция | Сигнатура | Описание |
| --- | --- | --- |
| `DefaultCheckConfig` | `() CheckConfig` | Конфигурация со значениями по умолчанию |
| `Validate` | `() error` | Валидация диапазонов |

#### CheckResult

```go
type CheckResult struct {
    Category StatusCategory
    Detail   string
}
```

Классификация результата проверки.

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

    // HTTP-опции
    HTTPHealthPath    string
    HTTPTLS           *bool
    HTTPTLSSkipVerify *bool
    HTTPHeaders       map[string]string
    HTTPBearerToken   string
    HTTPBasicUser     string
    HTTPBasicPass     string

    // gRPC-опции
    GRPCServiceName   string
    GRPCTLS           *bool
    GRPCTLSSkipVerify *bool
    GRPCMetadata      map[string]string
    GRPCBearerToken   string
    GRPCBasicUser     string
    GRPCBasicPass     string

    // Опции баз данных
    PostgresQuery     string
    MySQLQuery        string
    RedisPassword     string
    RedisDB           *int
    AMQPURL           string

    // LDAP-опции
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

Конфигурация для одной зависимости. Заполняется функциями `DependencyOption`
и передаётся в фабрики чекеров.

#### ParsedConnection

```go
type ParsedConnection struct {
    Host     string
    Port     string
    ConnType DependencyType
}
```

Результат парсинга URL или строки подключения.

#### ClassifiedCheckError

```go
type ClassifiedCheckError struct {
    Category StatusCategory
    Detail   string
    Cause    error
}
```

Готовая реализация `ClassifiedError`.

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `Error` | `() string` | Сообщение ошибки-причины |
| `Unwrap` | `() error` | Причина для `errors.Is`/`errors.As` |
| `StatusCategory` | `() StatusCategory` | Категория статуса |
| `StatusDetail` | `() string` | Строка детализации |

#### InvalidLabelError

```go
type InvalidLabelError struct {
    Label string
}
```

| Метод | Сигнатура | Описание |
| --- | --- | --- |
| `Error` | `() string` | Сообщение с именем невалидной метки |

### Функции

#### Конструктор

```go
func New(name string, group string, opts ...Option) (*DepHealth, error)
```

Создаёт экземпляр `DepHealth`. Параметры `name` и `group` обязательны
(через аргументы API или переменные окружения `DEPHEALTH_NAME`,
`DEPHEALTH_GROUP`). Возвращает ошибку при невалидной конфигурации.

#### Фабрики зависимостей

Каждая фабрика регистрирует зависимость и возвращает `Option` для `New()`.

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

Регистрирует произвольную зависимость с пользовательским `HealthChecker`.
Используется contrib-модулями и пользовательскими чекерами.

#### Парсеры URL

```go
func ParseURL(rawURL string) ([]ParsedConnection, error)
```

Парсит URL в host/port/type. Поддерживаемые схемы: `http`, `https`,
`grpc`, `tcp`, `postgresql`, `postgres`, `mysql`, `redis`, `rediss`,
`amqp`, `amqps`, `kafka`. Kafka multi-host URL
(`kafka://host1:9092,host2:9092`) возвращает несколько соединений.

```go
func ParseConnectionString(connStr string) (string, string, error)
```

Парсит строки подключения `Key=Value;Key=Value`. Возвращает `(host, port, error)`.

```go
func ParseJDBC(jdbcURL string) ([]ParsedConnection, error)
```

Парсит JDBC URL: `jdbc:postgresql://host:port/db`,
`jdbc:mysql://host:port/db`.

```go
func ParseParams(host, port string) (Endpoint, error)
```

Создаёт `Endpoint` из явных host и port. Валидирует диапазон порта
(1-65535), поддерживает IPv6-адреса в скобках.

#### Валидаторы

```go
func ValidateName(name string) error
```

Валидация name/group: `[a-z][a-z0-9-]*`, 1-63 символа.

```go
func ValidateLabelName(name string) error
```

Валидация имени метки: `[a-zA-Z_][a-zA-Z0-9_]*`. Отклоняет зарезервированные
имена: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`.

```go
func ValidateLabels(labels map[string]string) error
```

Валидация всех пользовательских меток.

#### Утилиты

```go
func BoolToYesNo(v bool) string
```

Конвертирует `bool` в `"yes"` / `"no"` для метки `critical`.

#### Реестр

```go
func RegisterCheckerFactory(depType DependencyType, factory CheckerFactory)
```

Регистрирует фабрику чекера для указанного типа. Вызывается из функций
`init()` пакетов чекеров.

### Типы опций

#### Option

```go
type Option func(*config) error
```

Функциональная опция для `New()`.

#### DependencyOption

```go
type DependencyOption func(*DependencyConfig)
```

Опция для конкретной зависимости.

#### CheckerFactory

```go
type CheckerFactory func(dc *DependencyConfig) HealthChecker
```

Функция, создающая чекер из `DependencyConfig`.

### Глобальные опции

Передаются в `New()`, применяются ко всем зависимостям, если не переопределены
на уровне зависимости.

| Функция | Сигнатура | Описание |
| --- | --- | --- |
| `WithCheckInterval` | `(d time.Duration) Option` | Глобальный интервал проверок (по умолчанию 15s) |
| `WithTimeout` | `(d time.Duration) Option` | Глобальный тайм-аут проверок (по умолчанию 5s) |
| `WithRegisterer` | `(r prometheus.Registerer) Option` | Пользовательский регистратор Prometheus |
| `WithLogger` | `(l *slog.Logger) Option` | Логгер для операций SDK |

### Опции зависимостей

#### Общие

| Функция | Сигнатура | Описание |
| --- | --- | --- |
| `FromURL` | `(rawURL string) DependencyOption` | Парсить host/port из URL |
| `FromParams` | `(host, port string) DependencyOption` | Указать host/port явно |
| `Critical` | `(v bool) DependencyOption` | Отметить как критическую (обязательно) |
| `WithLabel` | `(key, value string) DependencyOption` | Добавить метку Prometheus |
| `CheckInterval` | `(d time.Duration) DependencyOption` | Интервал для конкретной зависимости |
| `Timeout` | `(d time.Duration) DependencyOption` | Тайм-аут для конкретной зависимости |

#### HTTP

| Функция | Сигнатура | Описание |
| --- | --- | --- |
| `WithHTTPHealthPath` | `(path string) DependencyOption` | Путь проверки (по умолчанию `/health`) |
| `WithHTTPTLS` | `(enabled bool) DependencyOption` | Включить HTTPS (авто для `https://`) |
| `WithHTTPTLSSkipVerify` | `(skip bool) DependencyOption` | Пропустить проверку TLS-сертификата |
| `WithHTTPHeaders` | `(headers map[string]string) DependencyOption` | Пользовательские HTTP-заголовки |
| `WithHTTPBearerToken` | `(token string) DependencyOption` | Bearer-токен |
| `WithHTTPBasicAuth` | `(username, password string) DependencyOption` | Basic-аутентификация |

#### gRPC

| Функция | Сигнатура | Описание |
| --- | --- | --- |
| `WithGRPCServiceName` | `(name string) DependencyOption` | Имя сервиса (пустая строка = здоровье сервера) |
| `WithGRPCTLS` | `(enabled bool) DependencyOption` | Включить TLS |
| `WithGRPCTLSSkipVerify` | `(skip bool) DependencyOption` | Пропустить проверку TLS-сертификата |
| `WithGRPCMetadata` | `(metadata map[string]string) DependencyOption` | Пользовательские метаданные gRPC |
| `WithGRPCBearerToken` | `(token string) DependencyOption` | Bearer-токен |
| `WithGRPCBasicAuth` | `(username, password string) DependencyOption` | Basic-аутентификация |

#### PostgreSQL

| Функция | Сигнатура | Описание |
| --- | --- | --- |
| `WithPostgresQuery` | `(query string) DependencyOption` | SQL-запрос проверки (по умолчанию `SELECT 1`) |

#### MySQL

| Функция | Сигнатура | Описание |
| --- | --- | --- |
| `WithMySQLQuery` | `(query string) DependencyOption` | SQL-запрос проверки (по умолчанию `SELECT 1`) |

#### Redis

| Функция | Сигнатура | Описание |
| --- | --- | --- |
| `WithRedisPassword` | `(password string) DependencyOption` | Пароль (автономный режим) |
| `WithRedisDB` | `(db int) DependencyOption` | Номер базы данных (автономный режим) |

#### AMQP

| Функция | Сигнатура | Описание |
| --- | --- | --- |
| `WithAMQPURL` | `(url string) DependencyOption` | Полный AMQP URL |

#### LDAP

| Функция | Сигнатура | Описание |
| --- | --- | --- |
| `WithLDAPCheckMethod` | `(method string) DependencyOption` | Метод проверки: `anonymous_bind`, `simple_bind`, `root_dse`, `search` |
| `WithLDAPBindDN` | `(dn string) DependencyOption` | DN для простой привязки |
| `WithLDAPBindPassword` | `(password string) DependencyOption` | Пароль для простой привязки |
| `WithLDAPBaseDN` | `(baseDN string) DependencyOption` | Базовый DN для поиска |
| `WithLDAPSearchFilter` | `(filter string) DependencyOption` | LDAP-фильтр поиска (по умолчанию `(objectClass=*)`) |
| `WithLDAPSearchScope` | `(scope string) DependencyOption` | Область поиска: `base`, `one`, `sub` |
| `WithLDAPStartTLS` | `(enabled bool) DependencyOption` | Использовать StartTLS (только с `ldap://`) |
| `WithLDAPTLSSkipVerify` | `(skip bool) DependencyOption` | Пропустить проверку TLS-сертификата |

---

## Пакет `checks`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks`

Импорт этого пакета регистрирует фабрики для **всех 9 типов чекеров**
через blank-импорты под-пакетов. Также предоставляет обратно совместимые
псевдонимы типов и обёртки конструкторов.

### Обратно совместимые псевдонимы (устаревшие)

Все псевдонимы ниже **устаревшие**. Используйте под-пакеты напрямую.

#### Псевдонимы типов

| Псевдоним | Целевой тип | Под-пакет |
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

#### Обёртки конструкторов

| Обёртка | Целевая функция |
| --- | --- |
| `NewTCPChecker` | `tcpcheck.New` |
| `NewHTTPChecker` | `httpcheck.New` |
| `NewGRPCChecker` | `grpccheck.New` |
| `NewPostgresChecker` | `pgcheck.New` |
| `NewMySQLChecker` | `mysqlcheck.New` |
| `NewRedisChecker` | `redischeck.New` |
| `NewAMQPChecker` | `amqpcheck.New` |
| `NewKafkaChecker` | `kafkacheck.New` |

#### Обёртки опций

| Обёртка | Целевая функция |
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

## Под-пакеты (`checks/*`)

Каждый под-пакет предоставляет одну реализацию чекера. Импорт под-пакета
регистрирует его фабрику через `init()`.

### `checks/httpcheck`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck`

HTTP-чекер. Отправляет GET-запросы, успех при ответе 2xx.
Автоматически следует редиректам.

```go
type Checker struct{ /* приватные поля */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // возвращает "http"
```

| Опция | Сигнатура | Описание |
| --- | --- | --- |
| `WithHealthPath` | `(path string) Option` | Путь проверки (по умолчанию `/health`) |
| `WithTLSEnabled` | `(enabled bool) Option` | Включить HTTPS |
| `WithTLSSkipVerify` | `(skip bool) Option` | Пропустить проверку TLS-сертификата |
| `WithHeaders` | `(headers map[string]string) Option` | Пользовательские HTTP-заголовки |
| `WithBearerToken` | `(token string) Option` | Bearer-токен |
| `WithBasicAuth` | `(username, password string) Option` | Basic-аутентификация |

**Классификация ошибок:**

| Условие | Категория | Детализация |
| --- | --- | --- |
| Статус 401/403 | `auth_error` | `auth_error` |
| Статус не 2xx | `unhealthy` | `http_<код>` (например `http_503`) |

### `checks/grpccheck`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck`

Чекер по протоколу gRPC Health Checking Protocol. Создаёт новое соединение
для каждой проверки, отправляет `Health/Check` и закрывает.
Использует `passthrough:///` resolver.

```go
type Checker struct{ /* приватные поля */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // возвращает "grpc"
```

| Опция | Сигнатура | Описание |
| --- | --- | --- |
| `WithServiceName` | `(name string) Option` | Имя сервиса (пустая строка = общее здоровье) |
| `WithTLS` | `(enabled bool) Option` | Включить TLS |
| `WithTLSSkipVerify` | `(skip bool) Option` | Пропустить проверку TLS-сертификата |
| `WithMetadata` | `(md map[string]string) Option` | Пользовательские метаданные gRPC |
| `WithBearerToken` | `(token string) Option` | Bearer-токен |
| `WithBasicAuth` | `(username, password string) Option` | Basic-аутентификация |

**Классификация ошибок:**

| Условие | Категория | Детализация |
| --- | --- | --- |
| UNAUTHENTICATED / PERMISSION_DENIED | `auth_error` | `auth_error` |
| NOT_SERVING | `unhealthy` | `grpc_not_serving` |
| Прочие не-SERVING | `unhealthy` | `grpc_unknown` |

### `checks/tcpcheck`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/tcpcheck`

TCP-чекер. Устанавливает TCP-соединение и немедленно закрывает его.
Данные не отправляются и не принимаются.

```go
type Checker struct{}

func New() *Checker
func NewFromConfig(_ *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // возвращает "tcp"
```

Специфичных опций нет.

### `checks/pgcheck`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck`

Чекер PostgreSQL. Поддерживает автономный режим (новое соединение на каждую
проверку) и режим пула (существующий `*sql.DB`). Использует драйвер `pgx`.

```go
type Checker struct{ /* приватные поля */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // возвращает "postgres"
```

| Опция | Сигнатура | Описание |
| --- | --- | --- |
| `WithDB` | `(db *sql.DB) Option` | Использовать существующий пул соединений |
| `WithDSN` | `(dsn string) Option` | Пользовательский DSN (автономный режим) |
| `WithQuery` | `(query string) Option` | SQL-запрос проверки (по умолчанию `SELECT 1`) |

**Классификация ошибок:**

| Условие | Категория | Детализация |
| --- | --- | --- |
| SQLSTATE 28000 / 28P01 | `auth_error` | `auth_error` |

### `checks/mysqlcheck`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck`

Чекер MySQL. Поддерживает автономный режим и режим пула.
Использует `go-sql-driver/mysql`.

```go
type Checker struct{ /* приватные поля */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // возвращает "mysql"
```

| Опция | Сигнатура | Описание |
| --- | --- | --- |
| `WithDB` | `(db *sql.DB) Option` | Использовать существующий пул соединений |
| `WithDSN` | `(dsn string) Option` | Пользовательский DSN (автономный режим) |
| `WithQuery` | `(query string) Option` | SQL-запрос проверки (по умолчанию `SELECT 1`) |

**Вспомогательная функция:**

```go
func URLToDSN(rawURL string) string
```

Конвертирует `mysql://user:pass@host:3306/db` в формат DSN
`go-sql-driver/mysql`: `user:pass@tcp(host:3306)/db`. Возвращает пустую
строку при ошибке парсинга.

**Классификация ошибок:**

| Условие | Категория | Детализация |
| --- | --- | --- |
| Ошибка 1045 / Access denied | `auth_error` | `auth_error` |

### `checks/redischeck`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck`

Чекер Redis с использованием команды `PING`. Поддерживает автономный режим
и режим пула. Использует `go-redis/v9`.

```go
type Checker struct{ /* приватные поля */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // возвращает "redis"
```

| Опция | Сигнатура | Описание |
| --- | --- | --- |
| `WithClient` | `(client redis.Cmdable) Option` | Использовать существующий Redis-клиент |
| `WithPassword` | `(password string) Option` | Пароль (автономный режим) |
| `WithDB` | `(db int) Option` | Номер базы данных (автономный режим) |

**Классификация ошибок:**

| Условие | Категория | Детализация |
| --- | --- | --- |
| NOAUTH / WRONGPASS | `auth_error` | `auth_error` |
| Connection refused / timeout | `connection_error` | `connection_refused` |

### `checks/amqpcheck`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/amqpcheck`

AMQP-чекер. Устанавливает AMQP-соединение и немедленно закрывает его.
Только автономный режим. Использует `amqp091-go`.

```go
type Checker struct{ /* приватные поля */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // возвращает "amqp"
```

| Опция | Сигнатура | Описание |
| --- | --- | --- |
| `WithURL` | `(url string) Option` | AMQP URL (по умолчанию `amqp://guest:guest@host:port/`) |

**Классификация ошибок:**

| Условие | Категория | Детализация |
| --- | --- | --- |
| 403 / ACCESS_REFUSED | `auth_error` | `auth_error` |

### `checks/kafkacheck`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/kafkacheck`

Чекер Kafka. Подключается к брокеру, запрашивает метаданные и закрывает
соединение. Только автономный режим. Использует `segmentio/kafka-go`.

```go
type Checker struct{}

func New() *Checker
func NewFromConfig(_ *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // возвращает "kafka"
```

Специфичных опций нет.

**Классификация ошибок:**

| Условие | Категория | Детализация |
| --- | --- | --- |
| Нет брокеров в метаданных | `unhealthy` | `no_brokers` |

### `checks/ldapcheck`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck`

LDAP-чекер. Поддерживает четыре метода проверки: анонимная привязка, простая
привязка, запрос RootDSE и поиск. Поддерживает LDAP, LDAPS и StartTLS
соединения. Использует `go-ldap/ldap/v3`.

```go
type Checker struct{ /* приватные поля */ }
type Option func(*Checker)

func New(opts ...Option) *Checker
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error
func (c *Checker) Type() string  // возвращает "ldap"
```

| Опция | Сигнатура | Описание |
| --- | --- | --- |
| `WithConn` | `(conn *ldap.Conn) Option` | Использовать существующее LDAP-соединение (режим пула) |
| `WithCheckMethod` | `(method CheckMethod) Option` | Метод проверки (по умолчанию `MethodRootDSE`) |
| `WithBindDN` | `(dn string) Option` | DN для простой привязки |
| `WithBindPassword` | `(password string) Option` | Пароль для простой привязки |
| `WithBaseDN` | `(baseDN string) Option` | Базовый DN для поиска |
| `WithSearchFilter` | `(filter string) Option` | Фильтр поиска (по умолчанию `(objectClass=*)`) |
| `WithSearchScope` | `(scope SearchScope) Option` | Область поиска (по умолчанию `ScopeBase`) |
| `WithStartTLS` | `(enabled bool) Option` | Включить StartTLS |
| `WithUseTLS` | `(enabled bool) Option` | Использовать TLS (LDAPS) |
| `WithTLSSkipVerify` | `(skip bool) Option` | Пропустить проверку TLS-сертификата |

**Константы:**

| Константа | Тип | Значение |
| --- | --- | --- |
| `MethodAnonymousBind` | `CheckMethod` | `"anonymous_bind"` |
| `MethodSimpleBind` | `CheckMethod` | `"simple_bind"` |
| `MethodRootDSE` | `CheckMethod` | `"root_dse"` |
| `MethodSearch` | `CheckMethod` | `"search"` |
| `ScopeBase` | `SearchScope` | `"base"` |
| `ScopeOne` | `SearchScope` | `"one"` |
| `ScopeSub` | `SearchScope` | `"sub"` |

**Классификация ошибок:**

| Условие | Категория | Детализация |
| --- | --- | --- |
| LDAP код результата 49 (Invalid Credentials) | `auth_error` | `auth_error` |
| LDAP код результата 50 (Insufficient Access Rights) | `auth_error` | `auth_error` |
| Ошибка TLS/StartTLS рукопожатия | `tls_error` | `tls_error` |
| Сервер LDAP недоступен/занят | `unhealthy` | `unhealthy` |

**Ошибки валидации (возвращаются из `New` или `NewFromConfig`):**

| Условие | Ошибка |
| --- | --- |
| `simple_bind` без `bindDN` или `bindPassword` | `"simple_bind requires bindDN and bindPassword"` |
| `search` без `baseDN` | `"search requires baseDN"` |
| `startTLS` с `useTLS` (LDAPS) | `"startTLS and useTLS are mutually exclusive"` |

---

## Contrib-пакеты

### `contrib/sqldb`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb`

Интеграция с пулами соединений `*sql.DB` для PostgreSQL и MySQL.

```go
func FromDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option
```

Создаёт `Option` для мониторинга PostgreSQL через существующий `*sql.DB`.
Вызывающий должен указать `FromURL` или `FromParams` для определения
меток метрик.

```go
func FromMySQLDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option
```

Создаёт `Option` для мониторинга MySQL через существующий `*sql.DB`.
Вызывающий должен указать `FromURL` или `FromParams` для определения
меток метрик.

### `contrib/redispool`

**Импорт:** `github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool`

Интеграция с пулом соединений `*redis.Client`.

```go
func FromClient(name string, client *redis.Client, opts ...dephealth.DependencyOption) dephealth.Option
```

Создаёт `Option` для мониторинга Redis через существующий `*redis.Client`.
Host и port автоматически извлекаются из `client.Options().Addr`.
Дополнительные `DependencyOption` (`Critical`, `CheckInterval` и т.д.)
можно передать через `opts`.

---

## Динамическое управление эндпоинтами

Методы для добавления, удаления и обновления эндпоинтов в рантайме
на работающем экземпляре `DepHealth`. Все методы потокобезопасны.

### AddEndpoint

```go
func (dh *DepHealth) AddEndpoint(depName string, depType DependencyType,
    critical bool, ep Endpoint, checker HealthChecker) error
```

Добавляет новый эндпоинт к работающему экземпляру `DepHealth`. Горутина
проверки здоровья запускается немедленно с глобальным интервалом и тайм-аутом.

**Валидация:** `depName` через `ValidateName()`, `depType` по `ValidTypes`,
`ep.Host` и `ep.Port` не должны быть пустыми, `ep.Labels` через `ValidateLabels()`.

**Идемпотентность:** если эндпоинт с таким же ключом `depName:host:port` уже
существует, возвращает `nil` без изменений.

**Ошибки:**

| Условие | Ошибка |
| --- | --- |
| Планировщик не запущен или уже остановлен | `ErrNotStarted` |
| Невалидное имя зависимости | ошибка валидации |
| Неизвестный тип зависимости | `"unknown dependency type"` |
| Отсутствует host или port | `"missing host/port for endpoint"` |
| Зарезервированное имя метки | `InvalidLabelError` |

### RemoveEndpoint

```go
func (dh *DepHealth) RemoveEndpoint(depName, host, port string) error
```

Удаляет эндпоинт из работающего экземпляра `DepHealth`. Отменяет горутину
проверки и удаляет все связанные метрики Prometheus.

**Идемпотентность:** если эндпоинт с указанным ключом не существует,
возвращает `nil`.

**Ошибки:**

| Условие | Ошибка |
| --- | --- |
| Планировщик не запущен | `ErrNotStarted` |

### UpdateEndpoint

```go
func (dh *DepHealth) UpdateEndpoint(depName, oldHost, oldPort string,
    newEp Endpoint, checker HealthChecker) error
```

Атомарно заменяет существующий эндпоинт новым. Горутина старого эндпоинта
отменяется, его метрики удаляются; для нового эндпоинта запускается новая
горутина.

**Валидация:** `newEp.Host` и `newEp.Port` не должны быть пустыми,
`newEp.Labels` через `ValidateLabels()`.

**Ошибки:**

| Условие | Ошибка |
| --- | --- |
| Планировщик не запущен или уже остановлен | `ErrNotStarted` |
| Старый эндпоинт не найден | `ErrEndpointNotFound` |
| Отсутствует host или port нового эндпоинта | `"missing host/port for new endpoint"` |
| Зарезервированное имя метки | `InvalidLabelError` |

---

## Смотрите также

- [Начало работы](getting-started.ru.md) — установка и первый пример
- [Чекеры](checkers.ru.md) — подробное руководство по чекерам с примерами
- [Конфигурация](configuration.ru.md) — все параметры конфигурации
- [Пользовательские чекеры](custom-checkers.ru.md) — реализация `HealthChecker`
- [Выборочный импорт](selective-imports.ru.md) — уменьшение размера бинарника
- [Метрики](metrics.ru.md) — справочник метрик Prometheus
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
