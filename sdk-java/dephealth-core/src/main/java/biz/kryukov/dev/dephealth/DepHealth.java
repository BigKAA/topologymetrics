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
import java.util.List;
import java.util.Map;

/**
 * Точка входа SDK dephealth.
 *
 * <p>Использование:
 * <pre>{@code
 * DepHealth depHealth = DepHealth.builder(meterRegistry)
 *     .checkInterval(Duration.ofSeconds(15))
 *     .dependency("postgres-main", DependencyType.POSTGRES, d -> d
 *         .url("postgres://localhost:5432/db")
 *         .critical(true))
 *     .dependency("redis-cache", DependencyType.REDIS, d -> d
 *         .url("redis://localhost:6379"))
 *     .build();
 *
 * depHealth.start();
 * // ...
 * depHealth.stop();
 * }</pre>
 */
public final class DepHealth {

    private final CheckScheduler scheduler;

    private DepHealth(CheckScheduler scheduler) {
        this.scheduler = scheduler;
    }

    /** Запускает периодические проверки. */
    public void start() {
        scheduler.start();
    }

    /** Останавливает все проверки. */
    public void stop() {
        scheduler.stop();
    }

    /** Возвращает текущее состояние здоровья. Key: "name:host:port", value: healthy. */
    public Map<String, Boolean> health() {
        return scheduler.health();
    }

    public static Builder builder(MeterRegistry meterRegistry) {
        return new Builder(meterRegistry);
    }

    /**
     * Конфигурация зависимости в builder pattern.
     */
    public static final class DependencyBuilder {
        private String url;
        private String jdbcUrl;
        private String host;
        private String port;
        private boolean critical;
        private Duration interval;
        private Duration timeout;

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

        public DependencyBuilder critical(boolean critical) {
            this.critical = critical;
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
        private final MeterRegistry meterRegistry;
        private Duration globalInterval;
        private Duration globalTimeout;
        private final List<DependencyEntry> entries = new ArrayList<>();

        private Builder(MeterRegistry meterRegistry) {
            this.meterRegistry = meterRegistry;
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
         * Добавляет зависимость с конфигурацией через лямбду.
         */
        public Builder dependency(String name, DependencyType type,
                                  java.util.function.Consumer<DependencyBuilder> configurer) {
            DependencyBuilder db = new DependencyBuilder();
            configurer.accept(db);
            entries.add(new DependencyEntry(name, type, db));
            return this;
        }

        /**
         * Добавляет зависимость с готовым чекером (для pool-интеграции).
         */
        public Builder dependency(String name, DependencyType type, HealthChecker checker,
                                  java.util.function.Consumer<DependencyBuilder> configurer) {
            DependencyBuilder db = new DependencyBuilder();
            configurer.accept(db);
            entries.add(new DependencyEntry(name, type, db, checker));
            return this;
        }

        public DepHealth build() {
            if (entries.isEmpty()) {
                throw new ConfigurationException("At least one dependency must be configured");
            }

            MetricsExporter metricsExporter = new MetricsExporter(meterRegistry);
            CheckScheduler scheduler = new CheckScheduler(metricsExporter);

            for (DependencyEntry entry : entries) {
                buildAndRegister(entry, scheduler);
            }

            return new DepHealth(scheduler);
        }

        private void buildAndRegister(DependencyEntry entry, CheckScheduler scheduler) {
            DependencyBuilder db = entry.config;

            // Извлекаем credentials и параметры из URL, если они не заданы явно
            extractUrlCredentials(db, entry.type);

            // Определяем interval и timeout
            Duration interval = resolveInterval(db);
            Duration timeout = resolveTimeout(db, interval);

            // Определяем endpoints
            List<Endpoint> endpoints = resolveEndpoints(db, entry.type);

            CheckConfig config = CheckConfig.builder()
                    .interval(interval)
                    .timeout(timeout)
                    .initialDelay(Duration.ZERO) // PublicAPI: initialDelay=0
                    .build();

            Dependency dependency = Dependency.builder(entry.name, entry.type)
                    .endpoints(endpoints)
                    .critical(db.critical)
                    .config(config)
                    .build();

            // Создаём или используем готовый чекер
            HealthChecker checker = entry.checker != null
                    ? entry.checker
                    : createChecker(entry.type, db);

            scheduler.addDependency(dependency, checker);
        }

        /**
         * Извлекает credentials (username, password) и дополнительные параметры
         * (database, vhost) из URL, если они не были заданы явно.
         */
        @SuppressWarnings("checkstyle:CyclomaticComplexity")
        private void extractUrlCredentials(DependencyBuilder db, DependencyType type) {
            String rawUrl = db.url;
            if (rawUrl == null || rawUrl.isEmpty()) {
                return;
            }
            // JDBC URL — пропускаем, JDBC-драйвер сам парсит credentials
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
                            // Redis URL: redis://:password@host или redis://user:password@host
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
                        default -> { /* TCP, HTTP, gRPC, Kafka — credentials не используются */ }
                    }
                }

                // Извлекаем database/vhost из path
                String path = uri.getPath();
                if (path != null && !path.isEmpty()) {
                    // Для AMQP: path "/" означает vhost "/" (default)
                    // Для остальных: path "/" не содержит полезной информации
                    if (path.length() > 1) {
                        String value = path.substring(1); // убираем ведущий /
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
                                        // Не число — игнорируем
                                    }
                                }
                            }
                            default -> { /* Другие типы не используют path */ }
                        }
                    } else if (type == DependencyType.AMQP && "/".equals(path)) {
                        // AMQP: путь "/" означает vhost "/"
                        if (isBlank(db.amqpVirtualHost)) {
                            db.amqpVirtualHost = "/";
                        }
                    }
                }
            } catch (IllegalArgumentException ignored) {
                // Невалидный URI — игнорируем, resolveEndpoints обработает ошибку
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
            // Гарантируем timeout < interval
            if (t.compareTo(interval) >= 0) {
                t = Duration.ofMillis(interval.toMillis() - 1);
            }
            return t;
        }

        private List<Endpoint> resolveEndpoints(DependencyBuilder db, DependencyType type) {
            if (db.host != null && db.port != null) {
                return List.of(ConfigParser.parseParams(db.host, db.port));
            }
            if (db.jdbcUrl != null) {
                return ConfigParser.parseJdbc(db.jdbcUrl).stream()
                        .map(pc -> new Endpoint(pc.host(), pc.port()))
                        .toList();
            }
            if (db.url != null) {
                // Если URL начинается с jdbc: — парсим как JDBC
                if (db.url.toLowerCase().startsWith("jdbc:")) {
                    return ConfigParser.parseJdbc(db.url).stream()
                            .map(pc -> new Endpoint(pc.host(), pc.port()))
                            .toList();
                }
                return ConfigParser.parseUrl(db.url).stream()
                        .map(pc -> new Endpoint(pc.host(), pc.port()))
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
                    // Передаём amqpUrl только если он задан явно (не из общего url)
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
    }

    private record DependencyEntry(String name, DependencyType type, DependencyBuilder config,
                                   HealthChecker checker) {
        DependencyEntry(String name, DependencyType type, DependencyBuilder config) {
            this(name, type, config, null);
        }
    }
}
