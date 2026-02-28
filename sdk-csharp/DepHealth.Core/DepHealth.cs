using System.Text.RegularExpressions;
using DepHealth.Checks;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;
using Prometheus;

namespace DepHealth;

/// <summary>
/// Entry point for the dephealth SDK.
/// <para>
/// Usage:
/// <code>
/// var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
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

    /// <summary>Starts periodic health checks.</summary>
    public void Start() => _scheduler.Start();

    /// <summary>Stops all health checks.</summary>
    public void Stop() => _scheduler.Stop();

    /// <summary>Returns current health status. Key: "name:host:port", value: healthy.</summary>
    public Dictionary<string, bool> Health() => _scheduler.Health();

    /// <summary>Returns detailed health status for all endpoints. Key: "name:host:port".</summary>
    public Dictionary<string, EndpointStatus> HealthDetails() => _scheduler.HealthDetails();

    /// <summary>
    /// Dynamically adds a new endpoint at runtime.
    /// Validates inputs and delegates to the scheduler.
    /// </summary>
    public void AddEndpoint(string depName, DependencyType depType,
        bool critical, Endpoint ep, IHealthChecker checker)
    {
        ValidateDynamicEndpointArgs(depName, depType, ep);
        _scheduler.AddEndpoint(depName, depType, critical, ep, checker);
    }

    /// <summary>
    /// Dynamically removes an endpoint at runtime.
    /// </summary>
    public void RemoveEndpoint(string depName, string host, string port)
    {
        _scheduler.RemoveEndpoint(depName, host, port);
    }

    /// <summary>
    /// Dynamically replaces an endpoint with a new one at runtime.
    /// Validates new endpoint inputs and delegates to the scheduler.
    /// </summary>
    public void UpdateEndpoint(string depName, string oldHost, string oldPort,
        Endpoint newEp, IHealthChecker checker)
    {
        ValidateEndpointFields(newEp);
        _scheduler.UpdateEndpoint(depName, oldHost, oldPort, newEp, checker);
    }

    /// <inheritdoc />
    public void Dispose() => _scheduler.Dispose();

    private static void ValidateDynamicEndpointArgs(string depName, DependencyType depType, Endpoint ep)
    {
        // Validate dependency name using same rules as Dependency.Builder
        if (string.IsNullOrEmpty(depName) || depName.Length > MaxNameLength)
        {
            throw new ValidationException(
                $"dependency name must be 1-{MaxNameLength} characters, got '{depName}' ({depName?.Length ?? 0} chars)");
        }

        if (!NamePattern().IsMatch(depName))
        {
            throw new ValidationException(
                $"dependency name must match ^[a-z][a-z0-9-]*$, got '{depName}'");
        }

        if (!Enum.IsDefined(depType))
        {
            throw new ValidationException(
                $"invalid dependency type: {depType}");
        }

        ValidateEndpointFields(ep);
    }

    private static void ValidateEndpointFields(Endpoint ep)
    {
        if (string.IsNullOrEmpty(ep.Host))
        {
            throw new ValidationException("endpoint host must not be empty");
        }

        if (string.IsNullOrEmpty(ep.Port))
        {
            throw new ValidationException("endpoint port must not be empty");
        }

        Endpoint.ValidateLabels(ep.Labels);
    }

    /// <summary>Creates a new builder with a required application name and group.</summary>
    public static Builder CreateBuilder(string name, string group) => new(name, group);

    /// <summary>
    /// Fluent builder for configuring dephealth.
    /// </summary>
    public sealed class Builder
    {
        private readonly string _name;
        private readonly string _group;
        private CollectorRegistry? _registry;
        private ILogger? _logger;
        private TimeSpan _defaultInterval = CheckConfig.DefaultInterval;
        private TimeSpan _defaultTimeout = CheckConfig.DefaultTimeout;
        private TimeSpan _defaultInitialDelay = TimeSpan.Zero;
        private readonly List<DependencyEntry> _entries = [];

        internal Builder(string name, string group)
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

            // group: API > env var > error
            var resolvedGroup = group;
            if (string.IsNullOrEmpty(resolvedGroup))
            {
                resolvedGroup = Environment.GetEnvironmentVariable("DEPHEALTH_GROUP");
            }

            if (string.IsNullOrEmpty(resolvedGroup))
            {
                throw new ValidationException(
                    "group is required: pass to CreateBuilder() or set DEPHEALTH_GROUP");
            }

            ValidateInstanceName(resolvedGroup);
            _group = resolvedGroup;
        }

        /// <summary>Sets a custom Prometheus <see cref="CollectorRegistry"/>.</summary>
        public Builder WithRegistry(CollectorRegistry registry)
        {
            _registry = registry;
            return this;
        }

        /// <summary>Sets a logger for diagnostic messages.</summary>
        public Builder WithLogger(ILogger logger)
        {
            _logger = logger;
            return this;
        }

        /// <summary>Sets the default check interval for all dependencies.</summary>
        public Builder WithCheckInterval(TimeSpan interval)
        {
            _defaultInterval = interval;
            return this;
        }

        /// <summary>Sets the default check timeout for all dependencies.</summary>
        public Builder WithCheckTimeout(TimeSpan timeout)
        {
            _defaultTimeout = timeout;
            return this;
        }

        /// <summary>Sets the default initial delay before the first check.</summary>
        public Builder WithInitialDelay(TimeSpan initialDelay)
        {
            _defaultInitialDelay = initialDelay;
            return this;
        }

        // --- Convenience methods ---

        /// <summary>Adds an HTTP dependency to monitor.</summary>
        /// <param name="name">Dependency name.</param>
        /// <param name="url">HTTP(S) URL of the service.</param>
        /// <param name="healthPath">Health check path (default: "/health").</param>
        /// <param name="critical">Whether the dependency is critical.</param>
        /// <param name="labels">Custom Prometheus labels.</param>
        /// <param name="headers">Custom HTTP headers.</param>
        /// <param name="bearerToken">Bearer token for authentication.</param>
        /// <param name="basicAuthUsername">Basic auth username.</param>
        /// <param name="basicAuthPassword">Basic auth password.</param>
        public Builder AddHttp(string name, string url,
            string healthPath = "/health", bool? critical = null,
            Dictionary<string, string>? labels = null,
            Dictionary<string, string>? headers = null,
            string? bearerToken = null,
            string? basicAuthUsername = null,
            string? basicAuthPassword = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var checker = new HttpChecker(
                healthPath: healthPath,
                tlsEnabled: url.StartsWith("https://", StringComparison.OrdinalIgnoreCase),
                headers: headers,
                bearerToken: bearerToken,
                basicAuthUsername: basicAuthUsername,
                basicAuthPassword: basicAuthPassword);

            return AddDependency(name, DependencyType.Http, parsed, checker, critical, labels);
        }

        /// <summary>Adds a gRPC dependency to monitor.</summary>
        /// <param name="name">Dependency name.</param>
        /// <param name="host">gRPC server host.</param>
        /// <param name="port">gRPC server port.</param>
        /// <param name="tlsEnabled">Whether TLS is enabled.</param>
        /// <param name="critical">Whether the dependency is critical.</param>
        /// <param name="labels">Custom Prometheus labels.</param>
        /// <param name="metadata">gRPC metadata headers.</param>
        /// <param name="bearerToken">Bearer token for authentication.</param>
        /// <param name="basicAuthUsername">Basic auth username.</param>
        /// <param name="basicAuthPassword">Basic auth password.</param>
        public Builder AddGrpc(string name, string host, string port,
            bool tlsEnabled = false, bool? critical = null,
            Dictionary<string, string>? labels = null,
            Dictionary<string, string>? metadata = null,
            string? bearerToken = null,
            string? basicAuthUsername = null,
            string? basicAuthPassword = null)
        {
            var ep = ConfigParser.ParseParams(host, port);
            var checker = new GrpcChecker(
                tlsEnabled: tlsEnabled,
                metadata: metadata,
                bearerToken: bearerToken,
                basicAuthUsername: basicAuthUsername,
                basicAuthPassword: basicAuthPassword);

            return AddDependency(name, DependencyType.Grpc,
                [new ParsedConnection(ep.Host, ep.Port, DependencyType.Grpc)],
                checker, critical, labels);
        }

        /// <summary>Adds a raw TCP dependency to monitor.</summary>
        /// <param name="name">Dependency name.</param>
        /// <param name="host">TCP server host.</param>
        /// <param name="port">TCP server port.</param>
        /// <param name="critical">Whether the dependency is critical.</param>
        /// <param name="labels">Custom Prometheus labels.</param>
        public Builder AddTcp(string name, string host, string port,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var ep = ConfigParser.ParseParams(host, port);
            var checker = new TcpChecker();

            return AddDependency(name, DependencyType.Tcp,
                [new ParsedConnection(ep.Host, ep.Port, DependencyType.Tcp)],
                checker, critical, labels);
        }

        /// <summary>Adds a PostgreSQL dependency to monitor.</summary>
        /// <param name="name">Dependency name.</param>
        /// <param name="url">PostgreSQL connection URL.</param>
        /// <param name="critical">Whether the dependency is critical.</param>
        /// <param name="labels">Custom Prometheus labels.</param>
        public Builder AddPostgres(string name, string url,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (username, password) = ExtractUrlCredentials(url);
            var connStr = BuildPostgresConnectionString(parsed[0], username, password, url);
            var checker = new PostgresChecker(connStr);

            return AddDependency(name, DependencyType.Postgres, parsed, checker, critical, labels);
        }

        /// <summary>Adds a MySQL dependency to monitor.</summary>
        /// <param name="name">Dependency name.</param>
        /// <param name="url">MySQL connection URL.</param>
        /// <param name="critical">Whether the dependency is critical.</param>
        /// <param name="labels">Custom Prometheus labels.</param>
        public Builder AddMySql(string name, string url,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (username, password) = ExtractUrlCredentials(url);
            var connStr = BuildMySqlConnectionString(parsed[0], username, password, url);
            var checker = new MySqlChecker(connStr);

            return AddDependency(name, DependencyType.MySql, parsed, checker, critical, labels);
        }

        /// <summary>Adds a Redis dependency to monitor.</summary>
        /// <param name="name">Dependency name.</param>
        /// <param name="url">Redis connection URL.</param>
        /// <param name="critical">Whether the dependency is critical.</param>
        /// <param name="labels">Custom Prometheus labels.</param>
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

        /// <summary>Adds an AMQP (RabbitMQ) dependency to monitor.</summary>
        /// <param name="name">Dependency name.</param>
        /// <param name="url">AMQP connection URL.</param>
        /// <param name="critical">Whether the dependency is critical.</param>
        /// <param name="labels">Custom Prometheus labels.</param>
        public Builder AddAmqp(string name, string url,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var (username, password) = ExtractUrlCredentials(url);
            var vhost = ExtractAmqpVhost(url);
            var checker = new AmqpChecker(username: username, password: password, vhost: vhost);

            return AddDependency(name, DependencyType.Amqp, parsed, checker, critical, labels);
        }

        /// <summary>Adds a Kafka dependency to monitor.</summary>
        /// <param name="name">Dependency name.</param>
        /// <param name="url">Kafka bootstrap server URL.</param>
        /// <param name="critical">Whether the dependency is critical.</param>
        /// <param name="labels">Custom Prometheus labels.</param>
        public Builder AddKafka(string name, string url,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var parsed = ConfigParser.ParseUrl(url);
            var checker = new KafkaChecker();

            return AddDependency(name, DependencyType.Kafka, parsed, checker, critical, labels);
        }

        /// <summary>Adds an LDAP dependency to monitor.</summary>
        /// <param name="name">Dependency name.</param>
        /// <param name="host">LDAP server host.</param>
        /// <param name="port">LDAP server port.</param>
        /// <param name="checkMethod">LDAP check method (default: RootDse).</param>
        /// <param name="bindDN">Bind DN for authentication.</param>
        /// <param name="bindPassword">Bind password for authentication.</param>
        /// <param name="baseDN">Base DN for search operations.</param>
        /// <param name="searchFilter">LDAP search filter.</param>
        /// <param name="searchScope">LDAP search scope.</param>
        /// <param name="useTls">Whether to use LDAPS (TLS).</param>
        /// <param name="startTls">Whether to use StartTLS.</param>
        /// <param name="tlsSkipVerify">Whether to skip TLS certificate verification.</param>
        /// <param name="critical">Whether the dependency is critical.</param>
        /// <param name="labels">Custom Prometheus labels.</param>
        public Builder AddLdap(string name, string host, string port,
            LdapCheckMethod checkMethod = LdapCheckMethod.RootDse,
            string bindDN = "", string bindPassword = "",
            string baseDN = "", string searchFilter = "(objectClass=*)",
            LdapSearchScope searchScope = LdapSearchScope.Base,
            bool useTls = false, bool startTls = false,
            bool tlsSkipVerify = false,
            bool? critical = null, Dictionary<string, string>? labels = null)
        {
            var ep = ConfigParser.ParseParams(host, port);
            var checker = new LdapChecker(
                checkMethod: checkMethod,
                bindDN: bindDN,
                bindPassword: bindPassword,
                baseDN: baseDN,
                searchFilter: searchFilter,
                searchScope: searchScope,
                useTls: useTls,
                startTls: startTls,
                tlsSkipVerify: tlsSkipVerify);

            return AddDependency(name, DependencyType.Ldap,
                [new ParsedConnection(ep.Host, ep.Port, DependencyType.Ldap)],
                checker, critical, labels);
        }

        /// <summary>
        /// Adds a dependency with a custom health checker.
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

        /// <summary>Validates configuration and builds the <see cref="DepHealthMonitor"/> instance.</summary>
        /// <exception cref="ValidationException">Thrown when configuration validation fails.</exception>
        public DepHealthMonitor Build()
        {
            ApplyEnvVars();

            var customLabelKeys = CollectCustomLabelKeys();
            var metrics = new PrometheusExporter(_name, _group, customLabelKeys, _registry);

            var globalConfig = CheckConfig.CreateBuilder()
                .WithInterval(_defaultInterval)
                .WithTimeout(_defaultTimeout)
                .WithInitialDelay(_defaultInitialDelay)
                .Build();

            var scheduler = new CheckScheduler(metrics, globalConfig, _logger);

            foreach (var entry in _entries)
            {
                var config = globalConfig;

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
                        var labelKey = envKey[prefix.Length..].ToUpperInvariant();
                        var labelValue = Environment.GetEnvironmentVariable(envKey) ?? "";
                        if (!entry.Labels.ContainsKey(labelKey))
                        {
                            entry.Labels[labelKey] = labelValue;
                            // Update labels in endpoints
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
            var atSign = rest.IndexOf('@', StringComparison.Ordinal);
            if (atSign < 0)
            {
                return (null, null);
            }

            var userInfo = rest[..atSign];
            var colonIndex = userInfo.IndexOf(':', StringComparison.Ordinal);

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

            // Extract database from URL path
            var pathStart = url.IndexOf('/', url.IndexOf("://", StringComparison.Ordinal) + 3);
            if (pathStart >= 0)
            {
                // Find @ if present, look at path after it
                var afterAt = url.IndexOf('@', StringComparison.Ordinal);
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

            // Extract database from URL path
            var afterAt = url.IndexOf('@', StringComparison.Ordinal);
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
            var atSign = rest.IndexOf('@', StringComparison.Ordinal);
            if (atSign >= 0)
            {
                rest = rest[(atSign + 1)..];
            }

            var pathStart = rest.IndexOf('/', StringComparison.Ordinal);
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
