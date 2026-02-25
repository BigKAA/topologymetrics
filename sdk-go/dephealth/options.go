package dephealth

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Option is a functional option for New().
type Option func(*config) error

// DependencyOption is an option for a specific dependency.
type DependencyOption func(*DependencyConfig)

// config is the internal configuration for DepHealth.
type config struct {
	interval   time.Duration
	timeout    time.Duration
	registerer prometheus.Registerer
	logger     *slog.Logger
	entries    []dependencyEntry
}

// DependencyConfig is the configuration for a single dependency.
// Exported for use in checkerFactory (checks package).
type DependencyConfig struct {
	URL      string
	Host     string
	Port     string
	Critical *bool // nil = not set (validation error)
	Interval time.Duration
	Timeout  time.Duration
	Labels   map[string]string // Custom labels via WithLabel.

	// Checker-specific options.
	HTTPHealthPath    string
	HTTPTLS           *bool
	HTTPTLSSkipVerify *bool
	HTTPHeaders       map[string]string
	HTTPBearerToken   string
	HTTPBasicUser     string
	HTTPBasicPass     string

	GRPCServiceName   string
	GRPCTLS           *bool
	GRPCTLSSkipVerify *bool
	GRPCMetadata      map[string]string
	GRPCBearerToken   string
	GRPCBasicUser     string
	GRPCBasicPass     string

	PostgresQuery string
	MySQLQuery    string

	RedisPassword string
	RedisDB       *int

	AMQPURL string

	LDAPCheckMethod   string // "anonymous_bind", "simple_bind", "root_dse", "search"
	LDAPBindDN        string
	LDAPBindPassword  string
	LDAPBaseDN        string
	LDAPSearchFilter  string
	LDAPSearchScope   string // "base", "one", "sub"
	LDAPStartTLS      *bool
	LDAPTLSSkipVerify *bool
}

// dependencyEntry is a dependency with its checker, ready for registration.
type dependencyEntry struct {
	dep     Dependency
	checker HealthChecker
}

// CheckerFactory is a function that creates a checker from DependencyConfig.
type CheckerFactory func(dc *DependencyConfig) HealthChecker

// checkerFactories is the registry of checker factories by dependency type.
var checkerFactories = map[DependencyType]CheckerFactory{}

// RegisterCheckerFactory registers a checker factory for the specified type.
// Called from the checks package init().
func RegisterCheckerFactory(depType DependencyType, factory CheckerFactory) {
	checkerFactories[depType] = factory
}

// --- Global options (Option) ---

// WithCheckInterval sets the global check interval.
func WithCheckInterval(d time.Duration) Option {
	return func(c *config) error {
		c.interval = d
		return nil
	}
}

// WithTimeout sets the global check timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) error {
		c.timeout = d
		return nil
	}
}

// WithRegisterer sets a custom prometheus.Registerer for the public API.
func WithRegisterer(r prometheus.Registerer) Option {
	return func(c *config) error {
		c.registerer = r
		return nil
	}
}

// WithLogger sets the logger for the public API.
func WithLogger(l *slog.Logger) Option {
	return func(c *config) error {
		c.logger = l
		return nil
	}
}

// --- Dependency options (DependencyOption) ---

// FromURL sets the URL for parsing the dependency host/port.
func FromURL(rawURL string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.URL = rawURL
	}
}

// FromParams sets the dependency host and port explicitly.
func FromParams(host, port string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.Host = host
		dc.Port = port
	}
}

// Critical sets the criticality of a dependency.
// Required for every dependency; if not specified, a configuration error is returned.
func Critical(v bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.Critical = &v
	}
}

// WithLabel adds a custom label to the dependency.
// The label name is validated according to Prometheus naming conventions.
// Overriding required labels (name, dependency, type, host, port, critical) is not allowed.
func WithLabel(key, value string) DependencyOption {
	return func(dc *DependencyConfig) {
		if dc.Labels == nil {
			dc.Labels = make(map[string]string)
		}
		dc.Labels[key] = value
	}
}

// CheckInterval sets the check interval for a specific dependency.
func CheckInterval(d time.Duration) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.Interval = d
	}
}

// Timeout sets the check timeout for a specific dependency.
func Timeout(d time.Duration) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.Timeout = d
	}
}

// --- Checker wrappers (DependencyOption) ---

// WithHTTPHealthPath sets the path for HTTP health checks.
func WithHTTPHealthPath(path string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.HTTPHealthPath = path
	}
}

// WithHTTPTLS enables TLS for the HTTP checker.
func WithHTTPTLS(enabled bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.HTTPTLS = &enabled
	}
}

// WithHTTPTLSSkipVerify disables TLS certificate verification for HTTP.
func WithHTTPTLSSkipVerify(skip bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.HTTPTLSSkipVerify = &skip
	}
}

// WithGRPCServiceName sets the gRPC service name for health checks.
func WithGRPCServiceName(name string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.GRPCServiceName = name
	}
}

// WithGRPCTLS enables TLS for the gRPC checker.
func WithGRPCTLS(enabled bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.GRPCTLS = &enabled
	}
}

// WithGRPCTLSSkipVerify disables TLS certificate verification for gRPC.
func WithGRPCTLSSkipVerify(skip bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.GRPCTLSSkipVerify = &skip
	}
}

// WithHTTPHeaders sets custom HTTP headers for health check requests.
func WithHTTPHeaders(headers map[string]string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.HTTPHeaders = headers
	}
}

// WithHTTPBearerToken sets a Bearer token for HTTP health check requests.
// Adds Authorization: Bearer <token> header.
func WithHTTPBearerToken(token string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.HTTPBearerToken = token
	}
}

// WithHTTPBasicAuth sets Basic Auth credentials for HTTP health check requests.
// Adds Authorization: Basic <base64(username:password)> header.
func WithHTTPBasicAuth(username, password string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.HTTPBasicUser = username
		dc.HTTPBasicPass = password
	}
}

// WithGRPCMetadata sets custom gRPC metadata for health check calls.
func WithGRPCMetadata(metadata map[string]string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.GRPCMetadata = metadata
	}
}

// WithGRPCBearerToken sets a Bearer token for gRPC health check calls.
// Adds authorization: Bearer <token> metadata.
func WithGRPCBearerToken(token string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.GRPCBearerToken = token
	}
}

// WithGRPCBasicAuth sets Basic Auth credentials for gRPC health check calls.
// Adds authorization: Basic <base64(username:password)> metadata.
func WithGRPCBasicAuth(username, password string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.GRPCBasicUser = username
		dc.GRPCBasicPass = password
	}
}

// WithPostgresQuery sets the SQL query for PostgreSQL health checks.
func WithPostgresQuery(query string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.PostgresQuery = query
	}
}

// WithMySQLQuery sets the SQL query for MySQL health checks.
func WithMySQLQuery(query string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.MySQLQuery = query
	}
}

// WithRedisPassword sets the password for Redis (standalone mode).
func WithRedisPassword(password string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.RedisPassword = password
	}
}

// WithRedisDB sets the Redis database number (standalone mode).
func WithRedisDB(db int) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.RedisDB = &db
	}
}

// WithAMQPURL sets the full AMQP URL for connections.
func WithAMQPURL(url string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.AMQPURL = url
	}
}

// WithLDAPCheckMethod sets the LDAP check method.
// Valid values: "anonymous_bind", "simple_bind", "root_dse", "search".
func WithLDAPCheckMethod(method string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.LDAPCheckMethod = method
	}
}

// WithLDAPBindDN sets the DN for LDAP Simple Bind.
func WithLDAPBindDN(dn string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.LDAPBindDN = dn
	}
}

// WithLDAPBindPassword sets the password for LDAP Simple Bind.
func WithLDAPBindPassword(password string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.LDAPBindPassword = password
	}
}

// WithLDAPBaseDN sets the base DN for LDAP search method.
func WithLDAPBaseDN(baseDN string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.LDAPBaseDN = baseDN
	}
}

// WithLDAPSearchFilter sets the LDAP search filter.
func WithLDAPSearchFilter(filter string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.LDAPSearchFilter = filter
	}
}

// WithLDAPSearchScope sets the LDAP search scope.
// Valid values: "base", "one", "sub".
func WithLDAPSearchScope(scope string) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.LDAPSearchScope = scope
	}
}

// WithLDAPStartTLS enables StartTLS for LDAP connections (only with ldap://).
func WithLDAPStartTLS(enabled bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.LDAPStartTLS = &enabled
	}
}

// WithLDAPTLSSkipVerify disables TLS certificate verification for LDAP.
func WithLDAPTLSSkipVerify(skip bool) DependencyOption {
	return func(dc *DependencyConfig) {
		dc.LDAPTLSSkipVerify = &skip
	}
}

// --- Dependency factories (Option) ---

// makeDepOption creates a common dependency factory for the given type.
func makeDepOption(name string, depType DependencyType, opts []DependencyOption) Option {
	return func(c *config) error {
		dc := applyDepOpts(opts)

		// Automatically enable TLS for https:// URLs.
		if depType == TypeHTTP && dc.URL != "" && strings.HasPrefix(strings.ToLower(dc.URL), "https://") {
			if dc.HTTPTLS == nil {
				enabled := true
				dc.HTTPTLS = &enabled
			}
		}

		// Validate auth configuration: at most one auth method allowed.
		if depType == TypeHTTP {
			if err := validateHTTPAuthConfig(dc); err != nil {
				return fmt.Errorf("dependency %q: %w", name, err)
			}
		}
		if depType == TypeGRPC {
			if err := validateGRPCAuthConfig(dc); err != nil {
				return fmt.Errorf("dependency %q: %w", name, err)
			}
		}
		if depType == TypeLDAP {
			if err := validateLDAPConfig(dc); err != nil {
				return fmt.Errorf("dependency %q: %w", name, err)
			}
		}

		dep, err := buildDependency(name, depType, dc, c)
		if err != nil {
			return err
		}

		factory, ok := checkerFactories[depType]
		if !ok {
			return fmt.Errorf("dependency %q: no checker factory registered for type %q; import .../dephealth/checks (all) or a specific sub-package like .../dephealth/checks/httpcheck", name, depType)
		}

		c.entries = append(c.entries, dependencyEntry{
			dep:     dep,
			checker: factory(dc),
		})
		return nil
	}
}

// HTTP registers an HTTP dependency.
func HTTP(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeHTTP, opts)
}

// GRPC registers a gRPC dependency.
func GRPC(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeGRPC, opts)
}

// TCP registers a TCP dependency.
func TCP(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeTCP, opts)
}

// Postgres registers a PostgreSQL dependency.
func Postgres(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypePostgres, opts)
}

// MySQL registers a MySQL dependency.
func MySQL(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeMySQL, opts)
}

// Redis registers a Redis dependency.
func Redis(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeRedis, opts)
}

// AMQP registers an AMQP dependency.
func AMQP(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeAMQP, opts)
}

// Kafka registers a Kafka dependency.
func Kafka(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeKafka, opts)
}

// LDAP registers an LDAP dependency.
func LDAP(name string, opts ...DependencyOption) Option {
	return makeDepOption(name, TypeLDAP, opts)
}

// --- Contrib helper ---

// AddDependency creates an Option for registering an arbitrary dependency.
// Used by contrib modules for connection pool integration.
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

// --- Helper functions ---

// applyDepOpts applies dependency options and returns the configuration.
func applyDepOpts(opts []DependencyOption) *DependencyConfig {
	dc := &DependencyConfig{}
	for _, o := range opts {
		o(dc)
	}
	return dc
}

// envDepName converts a dependency name to UPPER_SNAKE_CASE for env vars.
// e.g. "postgres-main" -> "POSTGRES_MAIN"
func envDepName(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

// buildDependency assembles a Dependency from DependencyConfig and global config.
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

	// Critical: API > env var. Env var values: "yes"/"no".
	if dc.Critical == nil {
		envKey := "DEPHEALTH_" + envDepName(name) + "_CRITICAL"
		if v := os.Getenv(envKey); v != "" {
			switch strings.ToLower(v) {
			case "yes", "true":
				t := true
				dc.Critical = &t
			case "no", "false":
				f := false
				dc.Critical = &f
			}
		}
	}

	// Critical is required.
	if dc.Critical == nil {
		return Dependency{}, fmt.Errorf("missing critical for dependency %q", name)
	}

	// Validate labels.
	if err := ValidateLabels(dc.Labels); err != nil {
		return Dependency{}, fmt.Errorf("dependency %q: %w", name, err)
	}

	// Read custom labels from env vars: DEPHEALTH_<DEP>_LABEL_<KEY>.
	envPrefix := "DEPHEALTH_" + envDepName(name) + "_LABEL_"
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, envPrefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.ToLower(parts[0][len(envPrefix):])
			val := parts[1]
			// API has priority: skip if already set.
			if dc.Labels != nil {
				if _, exists := dc.Labels[key]; exists {
					continue
				}
			}
			if dc.Labels == nil {
				dc.Labels = make(map[string]string)
			}
			dc.Labels[key] = val
		}
	}

	// Merge labels into endpoints.
	if len(dc.Labels) > 0 {
		for i := range endpoints {
			if endpoints[i].Labels == nil {
				endpoints[i].Labels = make(map[string]string)
			}
			for k, v := range dc.Labels {
				endpoints[i].Labels[k] = v
			}
		}
	}

	// Determine interval: per-dependency > global > default.
	interval := DefaultCheckInterval
	if c.interval > 0 {
		interval = c.interval
	}
	if dc.Interval > 0 {
		interval = dc.Interval
	}

	// Determine timeout: per-dependency > global > default.
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

// validateHTTPAuthConfig checks that at most one HTTP auth method is configured.
func validateHTTPAuthConfig(dc *DependencyConfig) error {
	methods := 0
	if dc.HTTPBearerToken != "" {
		methods++
	}
	if dc.HTTPBasicUser != "" {
		methods++
	}
	for k := range dc.HTTPHeaders {
		if strings.EqualFold(k, "Authorization") {
			methods++
			break
		}
	}
	if methods > 1 {
		return fmt.Errorf("conflicting auth methods: specify only one of bearerToken, basicAuth, or Authorization header")
	}
	return nil
}

// validateLDAPConfig checks LDAP-specific configuration rules.
func validateLDAPConfig(dc *DependencyConfig) error {
	method := dc.LDAPCheckMethod
	if method == "" {
		method = "root_dse"
	}

	switch method {
	case "anonymous_bind", "simple_bind", "root_dse", "search":
		// valid
	default:
		return fmt.Errorf("invalid LDAP check method %q: must be one of anonymous_bind, simple_bind, root_dse, search", method)
	}

	if method == "simple_bind" && (dc.LDAPBindDN == "" || dc.LDAPBindPassword == "") {
		return fmt.Errorf("simple_bind requires both bindDN and bindPassword")
	}

	if method == "search" && dc.LDAPBaseDN == "" {
		return fmt.Errorf("search method requires baseDN")
	}

	// startTLS with ldaps:// is incompatible.
	if dc.LDAPStartTLS != nil && *dc.LDAPStartTLS && dc.URL != "" &&
		strings.HasPrefix(strings.ToLower(dc.URL), "ldaps://") {
		return fmt.Errorf("startTLS is incompatible with ldaps:// scheme")
	}

	scope := dc.LDAPSearchScope
	if scope != "" {
		switch scope {
		case "base", "one", "sub":
			// valid
		default:
			return fmt.Errorf("invalid LDAP search scope %q: must be one of base, one, sub", scope)
		}
	}

	return nil
}

// validateGRPCAuthConfig checks that at most one gRPC auth method is configured.
func validateGRPCAuthConfig(dc *DependencyConfig) error {
	methods := 0
	if dc.GRPCBearerToken != "" {
		methods++
	}
	if dc.GRPCBasicUser != "" {
		methods++
	}
	for k := range dc.GRPCMetadata {
		if strings.EqualFold(k, "authorization") {
			methods++
			break
		}
	}
	if methods > 1 {
		return fmt.Errorf("conflicting auth methods: specify only one of bearerToken, basicAuth, or authorization metadata")
	}
	return nil
}
