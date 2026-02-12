package biz.kryukov.dev.dephealth;

import biz.kryukov.dev.dephealth.checks.HttpHealthChecker;
import biz.kryukov.dev.dephealth.checks.TcpHealthChecker;
import biz.kryukov.dev.dephealth.checks.GrpcHealthChecker;
import biz.kryukov.dev.dephealth.checks.PostgresHealthChecker;
import biz.kryukov.dev.dephealth.checks.MysqlHealthChecker;
import biz.kryukov.dev.dephealth.checks.RedisHealthChecker;
import biz.kryukov.dev.dephealth.checks.AmqpHealthChecker;
import biz.kryukov.dev.dephealth.checks.KafkaHealthChecker;
import biz.kryukov.dev.dephealth.metrics.MetricsExporter;
import biz.kryukov.dev.dephealth.parser.ConfigParser;

import biz.kryukov.dev.dephealth.scheduler.CheckScheduler;

import io.micrometer.core.instrument.MeterRegistry;

import java.net.URI;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.TreeSet;
import java.util.regex.Pattern;

/**
 * Entry point for the dephealth SDK.
 *
 * <p>Usage:
 * <pre>{@code
 * DepHealth depHealth = DepHealth.builder("order-api", meterRegistry)
 *     .checkInterval(Duration.ofSeconds(15))
 *     .dependency("postgres-main", DependencyType.POSTGRES, d -> d
 *         .url("postgres://localhost:5432/db")
 *         .critical(true))
 *     .dependency("redis-cache", DependencyType.REDIS, d -> d
 *         .url("redis://localhost:6379")
 *         .critical(false))
 *     .build();
 *
 * depHealth.start();
 * // ...
 * depHealth.stop();
 * }</pre>
 */
public final class DepHealth {

    private static final Pattern NAME_PATTERN = Pattern.compile("^[a-z][a-z0-9-]*$");
    private static final int MAX_NAME_LENGTH = 63;
    private static final String ENV_NAME = "DEPHEALTH_NAME";

    private final CheckScheduler scheduler;

    private DepHealth(CheckScheduler scheduler) {
        this.scheduler = scheduler;
    }

    /** Starts periodic health checks. */
    public void start() {
        scheduler.start();
    }

    /** Stops all health checks. */
    public void stop() {
        scheduler.stop();
    }

    /** Returns current health status. Key: "name:host:port", value: healthy. */
    public Map<String, Boolean> health() {
        return scheduler.health();
    }

    /**
     * Creates a builder with a required application name.
     *
     * @param name          unique application name ({@code name} label)
     * @param meterRegistry Micrometer meter registry
     * @return builder
     */
    public static Builder builder(String name, MeterRegistry meterRegistry) {
        return new Builder(name, meterRegistry);
    }

    /**
     * Dependency configuration using the builder pattern.
     */
    public static final class DependencyBuilder {
        private String url;
        private String jdbcUrl;
        private String host;
        private String port;
        private Boolean criticalValue;
        private boolean criticalSet;
        private Duration interval;
        private Duration timeout;
        private final Map<String, String> labels = new LinkedHashMap<>();

        // HTTP
        private String httpHealthPath;
        private Boolean httpTls;
        private Boolean httpTlsSkipVerify;

        // gRPC
        private String grpcServiceName;
        private Boolean grpcTls;

        // DB
        private String dbUsername;
        private String dbPassword;
        private String dbDatabase;
        private String dbQuery;
        private javax.sql.DataSource dataSource;

        // Redis
        private String redisPassword;
        private Integer redisDb;
        private redis.clients.jedis.JedisPool jedisPool;

        // AMQP
        private String amqpUrl;
        private String amqpUsername;
        private String amqpPassword;
        private String amqpVirtualHost;

        private DependencyBuilder() {}

        public DependencyBuilder url(String url) {
            this.url = url;
            return this;
        }

        public DependencyBuilder jdbcUrl(String jdbcUrl) {
            this.jdbcUrl = jdbcUrl;
            return this;
        }

        public DependencyBuilder host(String host) {
            this.host = host;
            return this;
        }

        public DependencyBuilder port(String port) {
            this.port = port;
            return this;
        }

        /**
         * Sets the criticality of the dependency. Required parameter.
         */
        public DependencyBuilder critical(boolean critical) {
            this.criticalValue = critical;
            this.criticalSet = true;
            return this;
        }

        /**
         * Adds a custom label.
         *
         * @param key   label name (format {@code [a-zA-Z_][a-zA-Z0-9_]*})
         * @param value label value
         * @return this
         * @throws ValidationException if the name is invalid or reserved
         */
        public DependencyBuilder label(String key, String value) {
            Endpoint.validateLabelName(key);
            this.labels.put(key, value);
            return this;
        }

        public DependencyBuilder interval(Duration interval) {
            this.interval = interval;
            return this;
        }

        public DependencyBuilder timeout(Duration timeout) {
            this.timeout = timeout;
            return this;
        }

        public DependencyBuilder httpHealthPath(String healthPath) {
            this.httpHealthPath = healthPath;
            return this;
        }

        public DependencyBuilder httpTls(boolean tls) {
            this.httpTls = tls;
            return this;
        }

        public DependencyBuilder httpTlsSkipVerify(boolean skip) {
            this.httpTlsSkipVerify = skip;
            return this;
        }

        public DependencyBuilder grpcServiceName(String serviceName) {
            this.grpcServiceName = serviceName;
            return this;
        }

        public DependencyBuilder grpcTls(boolean tls) {
            this.grpcTls = tls;
            return this;
        }

        public DependencyBuilder dbUsername(String username) {
            this.dbUsername = username;
            return this;
        }

        public DependencyBuilder dbPassword(String password) {
            this.dbPassword = password;
            return this;
        }

        public DependencyBuilder dbDatabase(String database) {
            this.dbDatabase = database;
            return this;
        }

        public DependencyBuilder dbQuery(String query) {
            this.dbQuery = query;
            return this;
        }

        public DependencyBuilder dataSource(javax.sql.DataSource dataSource) {
            this.dataSource = dataSource;
            return this;
        }

        public DependencyBuilder redisPassword(String password) {
            this.redisPassword = password;
            return this;
        }

        public DependencyBuilder redisDb(int db) {
            this.redisDb = db;
            return this;
        }

        public DependencyBuilder jedisPool(redis.clients.jedis.JedisPool jedisPool) {
            this.jedisPool = jedisPool;
            return this;
        }

        public DependencyBuilder amqpUrl(String amqpUrl) {
            this.amqpUrl = amqpUrl;
            return this;
        }

        public DependencyBuilder amqpUsername(String username) {
            this.amqpUsername = username;
            return this;
        }

        public DependencyBuilder amqpPassword(String password) {
            this.amqpPassword = password;
            return this;
        }

        public DependencyBuilder amqpVirtualHost(String virtualHost) {
            this.amqpVirtualHost = virtualHost;
            return this;
        }
    }

    public static final class Builder {
        private final String instanceName;
        private final MeterRegistry meterRegistry;
        private Duration globalInterval;
        private Duration globalTimeout;
        private final List<DependencyEntry> entries = new ArrayList<>();

        private Builder(String name, MeterRegistry meterRegistry) {
            this.meterRegistry = meterRegistry;
            // API parameter takes precedence over env var
            String resolvedName = name;
            if (resolvedName == null || resolvedName.isEmpty()) {
                resolvedName = System.getenv(ENV_NAME);
            }
            if (resolvedName == null || resolvedName.isEmpty()) {
                throw new ConfigurationException(
                        "instance name is required: pass it to builder() or set " + ENV_NAME);
            }
            validateInstanceName(resolvedName);
            this.instanceName = resolvedName;
        }

        public Builder checkInterval(Duration interval) {
            this.globalInterval = interval;
            return this;
        }

        public Builder timeout(Duration timeout) {
            this.globalTimeout = timeout;
            return this;
        }

        /**
         * Adds a dependency configured via a lambda.
         */
        public Builder dependency(String name, DependencyType type,
                                  java.util.function.Consumer<DependencyBuilder> configurer) {
            DependencyBuilder db = new DependencyBuilder();
            configurer.accept(db);
            applyEnvVars(name, db);
            entries.add(new DependencyEntry(name, type, db));
            return this;
        }

        /**
         * Adds a dependency with a pre-built checker (for connection pool integration).
         */
        public Builder dependency(String name, DependencyType type, HealthChecker checker,
                                  java.util.function.Consumer<DependencyBuilder> configurer) {
            DependencyBuilder db = new DependencyBuilder();
            configurer.accept(db);
            applyEnvVars(name, db);
            entries.add(new DependencyEntry(name, type, db, checker));
            return this;
        }

        public DepHealth build() {
            // Collect all unique custom label keys from all dependencies
            List<String> customLabelKeys = collectCustomLabelKeys();

            MetricsExporter metricsExporter = new MetricsExporter(
                    meterRegistry, instanceName, customLabelKeys);
            CheckScheduler scheduler = new CheckScheduler(metricsExporter);

            for (DependencyEntry entry : entries) {
                buildAndRegister(entry, scheduler);
            }

            return new DepHealth(scheduler);
        }

        /**
         * Collects all unique custom label keys, sorted alphabetically.
         */
        private List<String> collectCustomLabelKeys() {
            TreeSet<String> keys = new TreeSet<>();
            for (DependencyEntry entry : entries) {
                keys.addAll(entry.config.labels.keySet());
            }
            return List.copyOf(keys);
        }

        /**
         * Applies env vars for critical and labels to DependencyBuilder.
         * Format: DEPHEALTH_&lt;DEP&gt;_CRITICAL=yes|no, DEPHEALTH_&lt;DEP&gt;_LABEL_&lt;KEY&gt;=value.
         * API parameters take precedence over env vars.
         */
        private void applyEnvVars(String depName, DependencyBuilder db) {
            String envPrefix = "DEPHEALTH_" + depName.toUpperCase().replace('-', '_');

            // DEPHEALTH_<DEP>_CRITICAL
            if (!db.criticalSet) {
                String criticalEnv = System.getenv(envPrefix + "_CRITICAL");
                if (criticalEnv != null) {
                    if ("yes".equalsIgnoreCase(criticalEnv)) {
                        db.criticalValue = true;
                        db.criticalSet = true;
                    } else if ("no".equalsIgnoreCase(criticalEnv)) {
                        db.criticalValue = false;
                        db.criticalSet = true;
                    }
                }
            }

            // DEPHEALTH_<DEP>_LABEL_<KEY>=value
            String labelPrefix = envPrefix + "_LABEL_";
            for (Map.Entry<String, String> envEntry : System.getenv().entrySet()) {
                if (envEntry.getKey().startsWith(labelPrefix)) {
                    String labelKey = envEntry.getKey().substring(labelPrefix.length())
                            .toLowerCase();
                    if (!db.labels.containsKey(labelKey)) {
                        // Validate and add only if not already set via API
                        Endpoint.validateLabelName(labelKey);
                        db.labels.put(labelKey, envEntry.getValue());
                    }
                }
            }
        }

        private void buildAndRegister(DependencyEntry entry, CheckScheduler scheduler) {
            DependencyBuilder db = entry.config;

            // Extract credentials and parameters from URL if not explicitly set
            extractUrlCredentials(db, entry.type);

            // Resolve interval and timeout
            Duration interval = resolveInterval(db);
            Duration timeout = resolveTimeout(db, interval);

            // Resolve endpoints with labels
            List<Endpoint> endpoints = resolveEndpoints(db, entry.type);

            CheckConfig config = CheckConfig.builder()
                    .interval(interval)
                    .timeout(timeout)
                    .initialDelay(Duration.ZERO) // PublicAPI: initialDelay=0
                    .build();

            Dependency.Builder depBuilder = Dependency.builder(entry.name, entry.type)
                    .endpoints(endpoints)
                    .config(config);
            if (db.criticalSet) {
                depBuilder.critical(db.criticalValue);
            }
            Dependency dependency = depBuilder.build();

            // Create or use the pre-built checker
            HealthChecker checker = entry.checker != null
                    ? entry.checker
                    : createChecker(entry.type, db);

            scheduler.addDependency(dependency, checker);
        }

        /**
         * Extracts credentials (username, password) and additional parameters
         * (database, vhost) from the URL if they were not explicitly set.
         */
        @SuppressWarnings("checkstyle:CyclomaticComplexity")
        private void extractUrlCredentials(DependencyBuilder db, DependencyType type) {
            String rawUrl = db.url;
            if (rawUrl == null || rawUrl.isEmpty()) {
                return;
            }
            // JDBC URL — skip, the JDBC driver parses credentials itself
            if (rawUrl.toLowerCase().startsWith("jdbc:")) {
                return;
            }
            try {
                URI uri = URI.create(rawUrl);
                String userInfo = uri.getRawUserInfo();
                if (userInfo != null && !userInfo.isEmpty()) {
                    String[] parts = userInfo.split(":", 2);
                    String user = URLDecoder.decode(parts[0], StandardCharsets.UTF_8);
                    String pass = parts.length > 1
                            ? URLDecoder.decode(parts[1], StandardCharsets.UTF_8) : null;

                    switch (type) {
                        case POSTGRES, MYSQL -> {
                            if (isBlank(db.dbUsername)) {
                                db.dbUsername = user;
                            }
                            if (isBlank(db.dbPassword) && pass != null) {
                                db.dbPassword = pass;
                            }
                        }
                        case REDIS -> {
                            // Redis URL: redis://:password@host or redis://user:password@host
                            if (isBlank(db.redisPassword) && pass != null) {
                                db.redisPassword = pass;
                            }
                        }
                        case AMQP -> {
                            if (isBlank(db.amqpUsername)) {
                                db.amqpUsername = user;
                            }
                            if (isBlank(db.amqpPassword) && pass != null) {
                                db.amqpPassword = pass;
                            }
                        }
                        default -> { /* TCP, HTTP, gRPC, Kafka — credentials not used */ }
                    }
                }

                // Extract database/vhost from path
                String path = uri.getPath();
                if (path != null && !path.isEmpty()) {
                    // For AMQP: path "/" means vhost "/" (default)
                    // For others: path "/" contains no useful information
                    if (path.length() > 1) {
                        String value = path.substring(1); // strip leading /
                        switch (type) {
                            case POSTGRES, MYSQL -> {
                                if (isBlank(db.dbDatabase)) {
                                    db.dbDatabase = value;
                                }
                            }
                            case AMQP -> {
                                if (isBlank(db.amqpVirtualHost)) {
                                    db.amqpVirtualHost = URLDecoder.decode(
                                            value, StandardCharsets.UTF_8);
                                }
                            }
                            case REDIS -> {
                                if (db.redisDb == null) {
                                    try {
                                        db.redisDb = Integer.parseInt(value);
                                    } catch (NumberFormatException ignored) {
                                        // Not a number — ignore
                                    }
                                }
                            }
                            default -> { /* Other types do not use path */ }
                        }
                    } else if (type == DependencyType.AMQP && "/".equals(path)) {
                        // AMQP: path "/" means vhost "/"
                        if (isBlank(db.amqpVirtualHost)) {
                            db.amqpVirtualHost = "/";
                        }
                    }
                }
            } catch (IllegalArgumentException ignored) {
                // Invalid URI — ignore, resolveEndpoints will handle the error
            }
        }

        private static boolean isBlank(String s) {
            return s == null || s.isEmpty();
        }

        private Duration resolveInterval(DependencyBuilder db) {
            if (db.interval != null) {
                return db.interval;
            }
            if (globalInterval != null) {
                return globalInterval;
            }
            return CheckConfig.DEFAULT_INTERVAL;
        }

        private Duration resolveTimeout(DependencyBuilder db, Duration interval) {
            Duration t = db.timeout != null ? db.timeout
                    : (globalTimeout != null ? globalTimeout : CheckConfig.DEFAULT_TIMEOUT);
            // Ensure timeout < interval
            if (t.compareTo(interval) >= 0) {
                t = Duration.ofMillis(interval.toMillis() - 1);
            }
            return t;
        }

        private List<Endpoint> resolveEndpoints(DependencyBuilder db, DependencyType type) {
            // Labels from DependencyBuilder are applied to all endpoints
            Map<String, String> depLabels = db.labels.isEmpty()
                    ? Map.of() : Map.copyOf(db.labels);

            if (db.host != null && db.port != null) {
                var parsed = ConfigParser.parseParams(db.host, db.port);
                return List.of(new Endpoint(parsed.host(), parsed.port(), depLabels));
            }
            if (db.jdbcUrl != null) {
                return ConfigParser.parseJdbc(db.jdbcUrl).stream()
                        .map(pc -> new Endpoint(pc.host(), pc.port(), depLabels))
                        .toList();
            }
            if (db.url != null) {
                // If URL starts with jdbc: — parse as JDBC
                if (db.url.toLowerCase().startsWith("jdbc:")) {
                    return ConfigParser.parseJdbc(db.url).stream()
                            .map(pc -> new Endpoint(pc.host(), pc.port(), depLabels))
                            .toList();
                }
                return ConfigParser.parseUrl(db.url).stream()
                        .map(pc -> new Endpoint(pc.host(), pc.port(), depLabels))
                        .toList();
            }
            throw new ConfigurationException(
                    "Dependency must have url, jdbcUrl, or host+port configured");
        }

        @SuppressWarnings("checkstyle:CyclomaticComplexity")
        private HealthChecker createChecker(DependencyType type, DependencyBuilder db) {
            return switch (type) {
                case HTTP -> {
                    HttpHealthChecker.Builder b = HttpHealthChecker.builder();
                    if (db.httpHealthPath != null) {
                        b.healthPath(db.httpHealthPath);
                    }
                    // Auto-detect TLS from URL scheme
                    boolean tls = db.httpTls != null ? db.httpTls
                            : (db.url != null && db.url.toLowerCase().startsWith("https://"));
                    b.tlsEnabled(tls);
                    if (db.httpTlsSkipVerify != null) {
                        b.tlsSkipVerify(db.httpTlsSkipVerify);
                    }
                    yield b.build();
                }
                case GRPC -> {
                    GrpcHealthChecker.Builder b = GrpcHealthChecker.builder();
                    if (db.grpcServiceName != null) {
                        b.serviceName(db.grpcServiceName);
                    }
                    if (db.grpcTls != null) {
                        b.tlsEnabled(db.grpcTls);
                    }
                    yield b.build();
                }
                case TCP -> new TcpHealthChecker();
                case POSTGRES -> {
                    PostgresHealthChecker.Builder b = PostgresHealthChecker.builder();
                    if (db.dataSource != null) {
                        b.dataSource(db.dataSource);
                    }
                    if (db.dbUsername != null) {
                        b.username(db.dbUsername);
                    }
                    if (db.dbPassword != null) {
                        b.password(db.dbPassword);
                    }
                    if (db.dbDatabase != null) {
                        b.database(db.dbDatabase);
                    }
                    if (db.dbQuery != null) {
                        b.query(db.dbQuery);
                    }
                    yield b.build();
                }
                case MYSQL -> {
                    MysqlHealthChecker.Builder b = MysqlHealthChecker.builder();
                    if (db.dataSource != null) {
                        b.dataSource(db.dataSource);
                    }
                    if (db.dbUsername != null) {
                        b.username(db.dbUsername);
                    }
                    if (db.dbPassword != null) {
                        b.password(db.dbPassword);
                    }
                    if (db.dbDatabase != null) {
                        b.database(db.dbDatabase);
                    }
                    if (db.dbQuery != null) {
                        b.query(db.dbQuery);
                    }
                    yield b.build();
                }
                case REDIS -> {
                    RedisHealthChecker.Builder b = RedisHealthChecker.builder();
                    if (db.jedisPool != null) {
                        b.jedisPool(db.jedisPool);
                    }
                    if (db.redisPassword != null) {
                        b.password(db.redisPassword);
                    }
                    if (db.redisDb != null) {
                        b.database(db.redisDb);
                    }
                    yield b.build();
                }
                case AMQP -> {
                    AmqpHealthChecker.Builder b = AmqpHealthChecker.builder();
                    // Pass amqpUrl only if it was explicitly set (not from the generic url)
                    if (db.amqpUrl != null) {
                        b.amqpUrl(db.amqpUrl);
                    }
                    if (db.amqpUsername != null) {
                        b.username(db.amqpUsername);
                    }
                    if (db.amqpPassword != null) {
                        b.password(db.amqpPassword);
                    }
                    if (db.amqpVirtualHost != null) {
                        b.virtualHost(db.amqpVirtualHost);
                    }
                    yield b.build();
                }
                case KAFKA -> new KafkaHealthChecker();
            };
        }

        private static void validateInstanceName(String name) {
            if (name.isEmpty() || name.length() > MAX_NAME_LENGTH) {
                throw new ConfigurationException(
                        "instance name must be 1-" + MAX_NAME_LENGTH + " characters, got '"
                                + name + "' (" + name.length() + " chars)");
            }
            if (!NAME_PATTERN.matcher(name).matches()) {
                throw new ConfigurationException(
                        "instance name must match " + NAME_PATTERN.pattern()
                                + ", got '" + name + "'");
            }
        }
    }

    private record DependencyEntry(String name, DependencyType type, DependencyBuilder config,
                                   HealthChecker checker) {
        DependencyEntry(String name, DependencyType type, DependencyBuilder config) {
            this(name, type, config, null);
        }
    }
}
