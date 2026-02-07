using DepHealth.Checks;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;
using Prometheus;

namespace DepHealth;

/// <summary>
/// Точка входа SDK dephealth.
/// <para>
/// Использование:
/// <code>
/// var dh = DepHealth.DepHealthBuilder.Create()
///     .AddPostgres("db", "postgres://user:pass@host:5432/mydb")
///     .AddRedis("cache", "redis://cache:6379")
///     .AddHttp("payment", "http://payment:8080")
///     .Build();
/// dh.Start();
/// // ...
/// dh.Stop();
/// </code>
/// </para>
/// </summary>
public sealed class DepHealthMonitor : IDisposable
{
    private readonly CheckScheduler _scheduler;

    private DepHealthMonitor(CheckScheduler scheduler)
    {
        _scheduler = scheduler;
    }

    /// <summary>Запускает периодические проверки.</summary>
    public void Start() => _scheduler.Start();

    /// <summary>Останавливает все проверки.</summary>
    public void Stop() => _scheduler.Stop();

    /// <summary>Возвращает текущее состояние здоровья. Key: "name:host:port", value: healthy.</summary>
    public Dictionary<string, bool> Health() => _scheduler.Health();

    public void Dispose() => _scheduler.Dispose();

    /// <summary>Создаёт новый builder.</summary>
    public static Builder CreateBuilder() => new();

    /// <summary>
    /// Fluent builder для конфигурации dephealth.
    /// </summary>
    public sealed class Builder
    {
        private CollectorRegistry? _registry;
        private ILogger? _logger;
        private TimeSpan _defaultInterval = CheckConfig.DefaultInterval;
        private TimeSpan _defaultTimeout = CheckConfig.DefaultTimeout;
        private TimeSpan _defaultInitialDelay = TimeSpan.Zero;
        private readonly List<DependencyEntry> _entries = [];

        internal Builder() { }

        public Builder WithRegistry(CollectorRegistry registry)
        {
            _registry = registry;
            return this;
        }

        public Builder WithLogger(ILogger logger)
        {
            _logger = logger;
            return this;
        }

        public Builder WithCheckInterval(TimeSpan interval)
        {
            _defaultInterval = interval;
            return this;
        }

        public Builder WithCheckTimeout(TimeSpan timeout)
        {
            _defaultTimeout = timeout;
            return this;
        }

        public Builder WithInitialDelay(TimeSpan initialDelay)
        {
            _defaultInitialDelay = initialDelay;
            return this;
        }

        // --- Convenience methods ---

        public Builder AddHttp(string name, string url,
            string healthPath = "/health", bool critical = false)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var checker = new HttpChecker(
                healthPath: healthPath,
                tlsEnabled: url.StartsWith("https://", StringComparison.OrdinalIgnoreCase));

            return AddDependency(name, DependencyType.Http, parsed, checker, critical);
        }

        public Builder AddGrpc(string name, string host, string port,
            bool tlsEnabled = false, bool critical = false)
        {
            var ep = ConfigParser.ParseParams(host, port);
            var checker = new GrpcChecker(tlsEnabled: tlsEnabled);

            return AddDependency(name, DependencyType.Grpc,
                [new ParsedConnection(ep.Host, ep.Port, DependencyType.Grpc)],
                checker, critical);
        }

        public Builder AddTcp(string name, string host, string port, bool critical = false)
        {
            var ep = ConfigParser.ParseParams(host, port);
            var checker = new TcpChecker();

            return AddDependency(name, DependencyType.Tcp,
                [new ParsedConnection(ep.Host, ep.Port, DependencyType.Tcp)],
                checker, critical);
        }

        public Builder AddPostgres(string name, string url, bool critical = false)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (username, password) = ExtractUrlCredentials(url);
            var connStr = BuildPostgresConnectionString(parsed[0], username, password, url);
            var checker = new PostgresChecker(connStr);

            return AddDependency(name, DependencyType.Postgres, parsed, checker, critical);
        }

        public Builder AddMySql(string name, string url, bool critical = false)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (username, password) = ExtractUrlCredentials(url);
            var connStr = BuildMySqlConnectionString(parsed[0], username, password, url);
            var checker = new MySqlChecker(connStr);

            return AddDependency(name, DependencyType.MySql, parsed, checker, critical);
        }

        public Builder AddRedis(string name, string url, bool critical = false)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (_, password) = ExtractUrlCredentials(url);
            var connStr = $"{parsed[0].Host}:{parsed[0].Port},connectTimeout=5000,abortConnect=true";
            if (!string.IsNullOrEmpty(password))
            {
                connStr += $",password={password}";
            }

            var checker = new RedisChecker(connStr);
            return AddDependency(name, DependencyType.Redis, parsed, checker, critical);
        }

        public Builder AddAmqp(string name, string url, bool critical = false)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (username, password) = ExtractUrlCredentials(url);
            var vhost = ExtractAmqpVhost(url);
            var checker = new AmqpChecker(username: username, password: password, vhost: vhost);

            return AddDependency(name, DependencyType.Amqp, parsed, checker, critical);
        }

        public Builder AddKafka(string name, string url, bool critical = false)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var checker = new KafkaChecker();

            return AddDependency(name, DependencyType.Kafka, parsed, checker, critical);
        }

        /// <summary>
        /// Добавляет зависимость с кастомным чекером.
        /// </summary>
        public Builder AddCustom(string name, DependencyType type, string host, string port,
            IHealthChecker checker, bool critical = false)
        {
            var ep = ConfigParser.ParseParams(host, port);
            return AddDependency(name, type,
                [new ParsedConnection(ep.Host, ep.Port, type)],
                checker, critical);
        }

        public DepHealthMonitor Build()
        {
            var metrics = new PrometheusExporter(_registry);
            var scheduler = new CheckScheduler(metrics, _logger);

            foreach (var entry in _entries)
            {
                var config = CheckConfig.CreateBuilder()
                    .WithInterval(_defaultInterval)
                    .WithTimeout(_defaultTimeout)
                    .WithInitialDelay(_defaultInitialDelay)
                    .Build();

                var dep = Dependency.CreateBuilder(entry.Name, entry.Type)
                    .WithEndpoints(entry.Endpoints)
                    .WithCritical(entry.Critical)
                    .WithConfig(config)
                    .Build();

                scheduler.AddDependency(dep, entry.Checker);
            }

            return new DepHealthMonitor(scheduler);
        }

        private Builder AddDependency(string name, DependencyType type,
            List<ParsedConnection> parsed, IHealthChecker checker, bool critical)
        {
            var endpoints = parsed.Select(p => new Endpoint(p.Host, p.Port)).ToList();
            _entries.Add(new DependencyEntry(name, type, endpoints, checker, critical));
            return this;
        }

        private static (string? Username, string? Password) ExtractUrlCredentials(string url)
        {
            var colonSlashSlash = url.IndexOf("://", StringComparison.Ordinal);
            if (colonSlashSlash < 0)
            {
                return (null, null);
            }

            var rest = url[(colonSlashSlash + 3)..];
            var atSign = rest.IndexOf('@');
            if (atSign < 0)
            {
                return (null, null);
            }

            var userInfo = rest[..atSign];
            var colonIndex = userInfo.IndexOf(':');

            if (colonIndex < 0)
            {
                return (Uri.UnescapeDataString(userInfo), null);
            }

            var username = Uri.UnescapeDataString(userInfo[..colonIndex]);
            var password = Uri.UnescapeDataString(userInfo[(colonIndex + 1)..]);
            return (username, password);
        }

        private static string BuildPostgresConnectionString(
            ParsedConnection parsed, string? username, string? password, string url)
        {
            var connStr = $"Host={parsed.Host};Port={parsed.Port};Timeout=5";
            if (!string.IsNullOrEmpty(username))
            {
                connStr += $";Username={username}";
            }

            if (!string.IsNullOrEmpty(password))
            {
                connStr += $";Password={password}";
            }

            // Извлечь database из URL path
            var pathStart = url.IndexOf('/', url.IndexOf("://", StringComparison.Ordinal) + 3);
            if (pathStart >= 0)
            {
                // Найти @ если есть, смотреть path после него
                var afterAt = url.IndexOf('@');
                if (afterAt >= 0)
                {
                    var afterAtPath = url.IndexOf('/', afterAt);
                    if (afterAtPath >= 0)
                    {
                        var db = url[(afterAtPath + 1)..].Split('?')[0];
                        if (!string.IsNullOrEmpty(db))
                        {
                            connStr += $";Database={db}";
                        }
                    }
                }
            }

            return connStr;
        }

        private static string BuildMySqlConnectionString(
            ParsedConnection parsed, string? username, string? password, string url)
        {
            var connStr = $"Server={parsed.Host};Port={parsed.Port};ConnectionTimeout=5";
            if (!string.IsNullOrEmpty(username))
            {
                connStr += $";User={username}";
            }

            if (!string.IsNullOrEmpty(password))
            {
                connStr += $";Password={password}";
            }

            // Извлечь database из URL path
            var afterAt = url.IndexOf('@');
            if (afterAt >= 0)
            {
                var pathStart = url.IndexOf('/', afterAt);
                if (pathStart >= 0)
                {
                    var db = url[(pathStart + 1)..].Split('?')[0];
                    if (!string.IsNullOrEmpty(db))
                    {
                        connStr += $";Database={db}";
                    }
                }
            }

            return connStr;
        }

        private static string ExtractAmqpVhost(string url)
        {
            var colonSlashSlash = url.IndexOf("://", StringComparison.Ordinal);
            if (colonSlashSlash < 0)
            {
                return "/";
            }

            var rest = url[(colonSlashSlash + 3)..];
            var atSign = rest.IndexOf('@');
            if (atSign >= 0)
            {
                rest = rest[(atSign + 1)..];
            }

            var pathStart = rest.IndexOf('/');
            if (pathStart < 0)
            {
                return "/";
            }

            var path = rest[(pathStart + 1)..].Split('?')[0];
            if (string.IsNullOrEmpty(path))
            {
                return "/";
            }

            return Uri.UnescapeDataString(path);
        }

        private sealed record DependencyEntry(
            string Name,
            DependencyType Type,
            List<Endpoint> Endpoints,
            IHealthChecker Checker,
            bool Critical);
    }
}
