using System.Text.RegularExpressions;
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
/// var dh = DepHealthMonitor.CreateBuilder("my-service")
///     .AddPostgres("db", "postgres://user:pass@host:5432/mydb", critical: true)
///     .AddRedis("cache", "redis://cache:6379", critical: false)
///     .AddHttp("payment", "http://payment:8080", critical: true)
///     .Build();
/// dh.Start();
/// // ...
/// dh.Stop();
/// </code>
/// </para>
/// </summary>
public sealed partial class DepHealthMonitor : IDisposable
{
    private const int MaxNameLength = 63;

    [GeneratedRegex("^[a-z][a-z0-9-]*$")]
    private static partial Regex NamePattern();

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

    /// <summary>Создаёт новый builder с обязательным именем приложения.</summary>
    public static Builder CreateBuilder(string name) => new(name);

    /// <summary>
    /// Fluent builder для конфигурации dephealth.
    /// </summary>
    public sealed class Builder
    {
        private readonly string _name;
        private CollectorRegistry? _registry;
        private ILogger? _logger;
        private TimeSpan _defaultInterval = CheckConfig.DefaultInterval;
        private TimeSpan _defaultTimeout = CheckConfig.DefaultTimeout;
        private TimeSpan _defaultInitialDelay = TimeSpan.Zero;
        private readonly List<DependencyEntry> _entries = [];

        internal Builder(string name)
        {
            var resolvedName = name;
            var envName = Environment.GetEnvironmentVariable("DEPHEALTH_NAME");
            if (!string.IsNullOrEmpty(envName) && string.IsNullOrEmpty(resolvedName))
            {
                resolvedName = envName;
            }

            if (string.IsNullOrEmpty(resolvedName))
            {
                resolvedName = name;
            }

            ValidateInstanceName(resolvedName);
            _name = resolvedName;
        }

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
            string healthPath = "/health", bool? critical = null,
            Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var checker = new HttpChecker(
                healthPath: healthPath,
                tlsEnabled: url.StartsWith("https://", StringComparison.OrdinalIgnoreCase));

            return AddDependency(name, DependencyType.Http, parsed, checker, critical, labels);
        }

        public Builder AddGrpc(string name, string host, string port,
            bool tlsEnabled = false, bool? critical = null,
            Dictionary<string, string>? labels = null)
        {
            var ep = ConfigParser.ParseParams(host, port);
            var checker = new GrpcChecker(tlsEnabled: tlsEnabled);

            return AddDependency(name, DependencyType.Grpc,
                [new ParsedConnection(ep.Host, ep.Port, DependencyType.Grpc)],
                checker, critical, labels);
        }

        public Builder AddTcp(string name, string host, string port,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var ep = ConfigParser.ParseParams(host, port);
            var checker = new TcpChecker();

            return AddDependency(name, DependencyType.Tcp,
                [new ParsedConnection(ep.Host, ep.Port, DependencyType.Tcp)],
                checker, critical, labels);
        }

        public Builder AddPostgres(string name, string url,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (username, password) = ExtractUrlCredentials(url);
            var connStr = BuildPostgresConnectionString(parsed[0], username, password, url);
            var checker = new PostgresChecker(connStr);

            return AddDependency(name, DependencyType.Postgres, parsed, checker, critical, labels);
        }

        public Builder AddMySql(string name, string url,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (username, password) = ExtractUrlCredentials(url);
            var connStr = BuildMySqlConnectionString(parsed[0], username, password, url);
            var checker = new MySqlChecker(connStr);

            return AddDependency(name, DependencyType.MySql, parsed, checker, critical, labels);
        }

        public Builder AddRedis(string name, string url,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (_, password) = ExtractUrlCredentials(url);
            var connStr = $"{parsed[0].Host}:{parsed[0].Port},connectTimeout=5000,abortConnect=true";
            if (!string.IsNullOrEmpty(password))
            {
                connStr += $",password={password}";
            }

            var checker = new RedisChecker(connStr);
            return AddDependency(name, DependencyType.Redis, parsed, checker, critical, labels);
        }

        public Builder AddAmqp(string name, string url,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (username, password) = ExtractUrlCredentials(url);
            var vhost = ExtractAmqpVhost(url);
            var checker = new AmqpChecker(username: username, password: password, vhost: vhost);

            return AddDependency(name, DependencyType.Amqp, parsed, checker, critical, labels);
        }

        public Builder AddKafka(string name, string url,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var checker = new KafkaChecker();

            return AddDependency(name, DependencyType.Kafka, parsed, checker, critical, labels);
        }

        /// <summary>
        /// Добавляет зависимость с кастомным чекером.
        /// </summary>
        public Builder AddCustom(string name, DependencyType type, string host, string port,
            IHealthChecker checker, bool? critical = null,
            Dictionary<string, string>? labels = null)
        {
            var ep = ConfigParser.ParseParams(host, port);
            return AddDependency(name, type,
                [new ParsedConnection(ep.Host, ep.Port, type)],
                checker, critical, labels);
        }

        public DepHealthMonitor Build()
        {
            ApplyEnvVars();

            var customLabelKeys = CollectCustomLabelKeys();
            var metrics = new PrometheusExporter(_name, customLabelKeys, _registry);
            var scheduler = new CheckScheduler(metrics, _logger);

            foreach (var entry in _entries)
            {
                var config = CheckConfig.CreateBuilder()
                    .WithInterval(_defaultInterval)
                    .WithTimeout(_defaultTimeout)
                    .WithInitialDelay(_defaultInitialDelay)
                    .Build();

                var depBuilder = Dependency.CreateBuilder(entry.Name, entry.Type)
                    .WithEndpoints(entry.Endpoints)
                    .WithConfig(config);

                if (entry.Critical is not null)
                {
                    depBuilder.WithCritical(entry.Critical.Value);
                }

                var dep = depBuilder.Build();

                scheduler.AddDependency(dep, entry.Checker);
            }

            return new DepHealthMonitor(scheduler);
        }

        private Builder AddDependency(string name, DependencyType type,
            List<ParsedConnection> parsed, IHealthChecker checker,
            bool? critical, Dictionary<string, string>? labels)
        {
            var mergedLabels = labels ?? new Dictionary<string, string>();
            Endpoint.ValidateLabels(mergedLabels);
            var endpoints = parsed.Select(p =>
                new Endpoint(p.Host, p.Port, mergedLabels)).ToList();
            _entries.Add(new DependencyEntry(name, type, endpoints, checker, critical, mergedLabels));
            return this;
        }

        private void ApplyEnvVars()
        {
            for (var i = 0; i < _entries.Count; i++)
            {
                var entry = _entries[i];
                var depKey = entry.Name.ToUpperInvariant().Replace('-', '_');

                // DEPHEALTH_<DEP>_CRITICAL
                if (entry.Critical is null)
                {
                    var criticalEnv = Environment.GetEnvironmentVariable(
                        $"DEPHEALTH_{depKey}_CRITICAL");
                    if (!string.IsNullOrEmpty(criticalEnv))
                    {
                        var criticalValue = criticalEnv.Equals("yes",
                            StringComparison.OrdinalIgnoreCase);
                        _entries[i] = entry with { Critical = criticalValue };
                        entry = _entries[i];
                    }
                }

                // DEPHEALTH_<DEP>_LABEL_<KEY>
                var prefix = $"DEPHEALTH_{depKey}_LABEL_";
                foreach (var envVar in Environment.GetEnvironmentVariables().Keys)
                {
                    var envKey = envVar.ToString()!;
                    if (envKey.StartsWith(prefix, StringComparison.OrdinalIgnoreCase))
                    {
                        var labelKey = envKey[prefix.Length..].ToLowerInvariant();
                        var labelValue = Environment.GetEnvironmentVariable(envKey) ?? "";
                        if (!entry.Labels.ContainsKey(labelKey))
                        {
                            entry.Labels[labelKey] = labelValue;
                            // Обновить labels в endpoints
                            var updatedEndpoints = entry.Endpoints
                                .Select(ep =>
                                {
                                    var newLabels = new Dictionary<string, string>(ep.Labels)
                                    {
                                        [labelKey] = labelValue
                                    };
                                    return new Endpoint(ep.Host, ep.Port, newLabels);
                                })
                                .ToList();
                            _entries[i] = entry with { Endpoints = updatedEndpoints };
                            entry = _entries[i];
                        }
                    }
                }
            }
        }

        private string[]? CollectCustomLabelKeys()
        {
            var keys = new SortedSet<string>(StringComparer.Ordinal);
            foreach (var entry in _entries)
            {
                foreach (var ep in entry.Endpoints)
                {
                    foreach (var key in ep.Labels.Keys)
                    {
                        keys.Add(key);
                    }
                }
            }

            return keys.Count > 0 ? keys.ToArray() : null;
        }

        private static void ValidateInstanceName(string name)
        {
            if (string.IsNullOrEmpty(name))
            {
                throw new ValidationException("instance name must not be empty");
            }

            if (name.Length > MaxNameLength)
            {
                throw new ValidationException(
                    $"instance name must be 1-{MaxNameLength} characters, got '{name}' ({name.Length} chars)");
            }

            if (!NamePattern().IsMatch(name))
            {
                throw new ValidationException(
                    $"instance name must match ^[a-z][a-z0-9-]*$, got '{name}'");
            }
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
            bool? Critical,
            Dictionary<string, string> Labels);
    }
}
