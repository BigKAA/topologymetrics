package dephealth

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Option — функциональная опция для New().
type Option func(*config) error

// DependencyOption — опция для конкретной зависимости.
type DependencyOption func(*DependencyConfig)

// config — внутренняя конфигурация DepHealth.
type config struct {
	interval   time.Duration
	timeout    time.Duration
	registerer prometheus.Registerer
	logger     *slog.Logger
	entries    []dependencyEntry
}

// DependencyConfig — конфигурация одной зависимости.
// Экспортируется для использования в checkerFactory (пакет checks).
type DependencyConfig struct {
	URL      string
	Host     string
	Port     string
	Critical bool
	Interval time.Duration
	Timeout  time.Duration

	// Checker-специфичные опции.
	HTTPHealthPath    string
	HTTPTLS           *bool
	HTTPTLSSkipVerify *bool

	GRPCServiceName   string
	GRPCTLS           *bool
	GRPCTLSSkipVerify *bool

	PostgresQuery string
	MySQLQuery    string

	RedisPassword string
	RedisDB       *int

	AMQPURL string
}

// dependencyEntry — зависимость с чекером, готовая к регистрации.
type dependencyEntry struct {
	dep     Dependency
	checker HealthChecker
}

// CheckerFactory — функция для создания чекера из DependencyConfig.
type CheckerFactory func(dc *DependencyConfig) HealthChecker

// checkerFactories — реестр фабрик чекеров по типу зависимости.
var checkerFactories = map[DependencyType]CheckerFactory{}

// RegisterCheckerFactory регистрирует фабрику чекера для указанного типа.
// Вызывается из init() пакета checks.
func RegisterCheckerFactory(depType DependencyType, factory CheckerFactory) {
	checkerFactories[depType] = factory
}

// --- Глобальные опции (Option) ---

// WithCheckInterval задаёт глобальный интервал проверок.
func WithCheckInterval(d time.Duration) Option {
	return func(c *config) error {
		c.interval = d
		return nil
	}
}

// WithTimeout задаёт глобальный таймаут проверки.
func WithTimeout(d time.Duration) Option {
	return func(c *config) error {
		c.timeout = d
		return nil
	}
}

// WithRegisterer задаёт кастомный prometheus.Registerer для публичного API.
func WithRegisterer(r prometheus.Registerer) Option {
	return func(c *config) error {
		c.registerer = r
		return nil
	}
}

// WithLogger задаёт логгер для публичного API.
func WithLogger(l *slog.Logger) Option {
	return func(c *config) error {
		c.logger = l
		return nil
	}
}

// --- Опции зависимостей (DependencyOption) ---

// FromURL задаёт URL для парсинга host/port зависимости.
func FromURL(rawURL string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.URL = rawURL
	}
}

// FromParams задаёт host и port зависимости явно.
func FromParams(host, port string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.Host = host
		dc.Port = port
	}
}

// Critical помечает зависимость как критическую.
func Critical(v bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.Critical = v
	}
}

// CheckInterval задаёт интервал проверки для конкретной зависимости.
func CheckInterval(d time.Duration) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.Interval = d
	}
}

// Timeout задаёт таймаут проверки для конкретной зависимости.
func Timeout(d time.Duration) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.Timeout = d
	}
}

// --- Checker-обёртки (DependencyOption) ---

// WithHTTPHealthPath задаёт путь для HTTP health check.
func WithHTTPHealthPath(path string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.HTTPHealthPath = path
	}
}

// WithHTTPTLS включает TLS для HTTP-чекера.
func WithHTTPTLS(enabled bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.HTTPTLS = &enabled
	}
}

// WithHTTPTLSSkipVerify отключает проверку TLS-сертификатов для HTTP.
func WithHTTPTLSSkipVerify(skip bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.HTTPTLSSkipVerify = &skip
	}
}

// WithGRPCServiceName задаёт имя gRPC-сервиса для проверки.
func WithGRPCServiceName(name string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.GRPCServiceName = name
	}
}

// WithGRPCTLS включает TLS для gRPC-чекера.
func WithGRPCTLS(enabled bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.GRPCTLS = &enabled
	}
}

// WithGRPCTLSSkipVerify отключает проверку TLS-сертификатов для gRPC.
func WithGRPCTLSSkipVerify(skip bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.GRPCTLSSkipVerify = &skip
	}
}

// WithPostgresQuery задаёт SQL-запрос для проверки PostgreSQL.
func WithPostgresQuery(query string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.PostgresQuery = query
	}
}

// WithMySQLQuery задаёт SQL-запрос для проверки MySQL.
func WithMySQLQuery(query string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.MySQLQuery = query
	}
}

// WithRedisPassword задаёт пароль для Redis (standalone-режим).
func WithRedisPassword(password string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.RedisPassword = password
	}
}

// WithRedisDB задаёт номер базы данных Redis (standalone-режим).
func WithRedisDB(db int) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.RedisDB = &db
	}
}

// WithAMQPURL задаёт полный AMQP URL для подключения.
func WithAMQPURL(url string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.AMQPURL = url
	}
}

// --- Фабрики зависимостей (Option) ---

// makeDepOption создаёт общую фабрику зависимости для заданного типа.
func makeDepOption(name string, depType DependencyType, opts []DependencyOption) Option {
	return func(c *config) error {
		dc := applyDepOpts(opts)

		// Автоматически включаем TLS для https:// URL.
		if depType == TypeHTTP && dc.URL != "" && strings.HasPrefix(strings.ToLower(dc.URL), "https://") {
			if dc.HTTPTLS == nil {
				enabled := true
				dc.HTTPTLS = &enabled
			}
		}

		dep, err := buildDependency(name, depType, dc, c)
		if err != nil {
			return err
		}

		factory, ok := checkerFactories[depType]
		if !ok {
			return fmt.Errorf("dependency %q: no checker factory registered for type %q; import github.com/BigKAA/topologymetrics/dephealth/checks", name, depType)
		}

		c.entries = append(c.entries, dependencyEntry{
			dep:     dep,
			checker: factory(dc),
		})
		return nil
	}
}

// HTTP регистрирует HTTP-зависимость.
func HTTP(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeHTTP, opts)
}

// GRPC регистрирует gRPC-зависимость.
func GRPC(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeGRPC, opts)
}

// TCP регистрирует TCP-зависимость.
func TCP(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeTCP, opts)
}

// Postgres регистрирует PostgreSQL-зависимость.
func Postgres(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypePostgres, opts)
}

// MySQL регистрирует MySQL-зависимость.
func MySQL(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeMySQL, opts)
}

// Redis регистрирует Redis-зависимость.
func Redis(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeRedis, opts)
}

// AMQP регистрирует AMQP-зависимость.
func AMQP(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeAMQP, opts)
}

// Kafka регистрирует Kafka-зависимость.
func Kafka(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeKafka, opts)
}

// --- Contrib-хелпер ---

// AddDependency создаёт Option для регистрации произвольной зависимости.
// Используется contrib-модулями для интеграции с пулами соединений.
func AddDependency(name string, depType DependencyType, checker HealthChecker, opts ...DependencyOption) Option {
	return func(c *config) error {
		dc := applyDepOpts(opts)
		dep, err := buildDependency(name, depType, dc, c)
		if err != nil {
			return err
		}

		c.entries = append(c.entries, dependencyEntry{
			dep:     dep,
			checker: checker,
		})
		return nil
	}
}

// --- Вспомогательные функции ---

// applyDepOpts применяет опции зависимости и возвращает конфигурацию.
func applyDepOpts(opts []DependencyOption) *DependencyConfig {
	dc := &DependencyConfig{}
	for _, o := range opts {
		o(dc)
	}
	return dc
}

// buildDependency собирает Dependency из DependencyConfig и глобальной config.
func buildDependency(name string, depType DependencyType, dc *DependencyConfig, c *config) (Dependency, error) {
	var endpoints []Endpoint

	if dc.URL != "" {
		parsed, err := ParseURL(dc.URL)
		if err != nil {
			return Dependency{}, fmt.Errorf("dependency %q: %w", name, err)
		}
		for _, p := range parsed {
			endpoints = append(endpoints, Endpoint{Host: p.Host, Port: p.Port})
		}
	} else if dc.Host != "" {
		ep, err := ParseParams(dc.Host, dc.Port)
		if err != nil {
			return Dependency{}, fmt.Errorf("dependency %q: %w", name, err)
		}
		endpoints = []Endpoint{ep}
	} else {
		return Dependency{}, fmt.Errorf("dependency %q: missing URL or host/port parameters", name)
	}

	// Определяем интервал: per-dependency → global → default.
	interval := DefaultCheckInterval
	if c.interval > 0 {
		interval = c.interval
	}
	if dc.Interval > 0 {
		interval = dc.Interval
	}

	// Определяем таймаут: per-dependency → global → default.
	timeout := DefaultTimeout
	if c.timeout > 0 {
		timeout = c.timeout
	}
	if dc.Timeout > 0 {
		timeout = dc.Timeout
	}

	dep := Dependency{
		Name:      name,
		Type:      depType,
		Critical:  dc.Critical,
		Endpoints: endpoints,
		Config: CheckConfig{
			Interval:         interval,
			Timeout:          timeout,
			InitialDelay:     0,
			FailureThreshold: DefaultFailureThreshold,
			SuccessThreshold: DefaultSuccessThreshold,
		},
	}

	return dep, nil
}
