namespace DepHealth;

/// <summary>
/// Parser for URLs, JDBC URLs, connection strings, and explicit parameters (host/port).
/// </summary>
public static class ConfigParser
{
    private static readonly Dictionary<string, DependencyType> SchemeToType =
        new(StringComparer.OrdinalIgnoreCase)
        {
            ["postgres"] = DependencyType.Postgres,
            ["postgresql"] = DependencyType.Postgres,
            ["mysql"] = DependencyType.MySql,
            ["redis"] = DependencyType.Redis,
            ["rediss"] = DependencyType.Redis,
            ["amqp"] = DependencyType.Amqp,
            ["amqps"] = DependencyType.Amqp,
            ["http"] = DependencyType.Http,
            ["https"] = DependencyType.Http,
            ["grpc"] = DependencyType.Grpc,
            ["kafka"] = DependencyType.Kafka,
            ["ldap"] = DependencyType.Ldap,
            ["ldaps"] = DependencyType.Ldap
        };

    private static readonly Dictionary<string, DependencyType> JdbcSubprotocolToType =
        new(StringComparer.OrdinalIgnoreCase)
        {
            ["postgresql"] = DependencyType.Postgres,
            ["mysql"] = DependencyType.MySql
        };

    private static readonly HashSet<string> HostKeys =
        new(StringComparer.OrdinalIgnoreCase)
        {
            "host", "server", "data source", "address", "addr", "network address"
        };

    private static readonly HashSet<string> PortKeys =
        new(StringComparer.OrdinalIgnoreCase) { "port" };

    /// <summary>
    /// Parses a URL (supports multi-host for Kafka, IPv6).
    /// </summary>
    public static List<ParsedConnection> ParseUrl(string rawUrl)
    {
        if (string.IsNullOrWhiteSpace(rawUrl))
        {
            throw new ConfigurationException("URL must not be empty");
        }

        var colonSlashSlash = rawUrl.IndexOf("://", StringComparison.Ordinal);
        if (colonSlashSlash < 0)
        {
            throw new ConfigurationException($"URL must have a scheme (e.g. http://): {rawUrl}");
        }

        var scheme = rawUrl[..colonSlashSlash].ToLowerInvariant();
        if (!SchemeToType.TryGetValue(scheme, out var type))
        {
            throw new ConfigurationException($"Unsupported URL scheme: {scheme}");
        }

        var rest = rawUrl[(colonSlashSlash + 3)..];

        // Strip userinfo (user:pass@)
        var atSign = rest.IndexOf('@');
        if (atSign >= 0)
        {
            rest = rest[(atSign + 1)..];
        }

        // Strip path/query/fragment
        var pathStart = rest.IndexOf('/');
        var hostPortPart = pathStart >= 0 ? rest[..pathStart] : rest;
        var queryStart = hostPortPart.IndexOf('?');
        if (queryStart >= 0)
        {
            hostPortPart = hostPortPart[..queryStart];
        }

        // Multi-host (Kafka): host1:port1,host2:port2
        if (type == DependencyType.Kafka || hostPortPart.Contains(','))
        {
            return ParseMultiHost(hostPortPart, scheme, type);
        }

        var (host, port) = ExtractHostPort(hostPortPart, scheme);
        ValidatePort(port, rawUrl);
        return [new ParsedConnection(host, port, type)];
    }

    /// <summary>
    /// Parses a JDBC URL: jdbc:subprotocol://host:port/db
    /// </summary>
    public static List<ParsedConnection> ParseJdbc(string jdbcUrl)
    {
        if (string.IsNullOrWhiteSpace(jdbcUrl))
        {
            throw new ConfigurationException("JDBC URL must not be empty");
        }

        if (!jdbcUrl.StartsWith("jdbc:", StringComparison.OrdinalIgnoreCase))
        {
            throw new ConfigurationException($"JDBC URL must start with 'jdbc:': {jdbcUrl}");
        }

        var withoutJdbc = jdbcUrl[5..];
        var colonSlashSlash = withoutJdbc.IndexOf("://", StringComparison.Ordinal);
        if (colonSlashSlash < 0)
        {
            throw new ConfigurationException($"JDBC URL must contain ://: {jdbcUrl}");
        }

        var subprotocol = withoutJdbc[..colonSlashSlash].ToLowerInvariant();
        if (!JdbcSubprotocolToType.ContainsKey(subprotocol))
        {
            throw new ConfigurationException($"Unsupported JDBC subprotocol: {subprotocol}");
        }

        var normalizedUrl = subprotocol + withoutJdbc[colonSlashSlash..];
        return ParseUrl(normalizedUrl);
    }

    /// <summary>
    /// Parses a connection string in key=value;key=value format.
    /// </summary>
    public static Endpoint ParseConnectionString(string connStr)
    {
        if (string.IsNullOrWhiteSpace(connStr))
        {
            throw new ConfigurationException("Connection string must not be empty");
        }

        var pairs = ParseKeyValuePairs(connStr);

        string? host = null;
        string? port = null;

        foreach (var (key, value) in pairs)
        {
            if (HostKeys.Contains(key))
            {
                if (value.Contains(','))
                {
                    // SQL Server: Server=host,port
                    var parts = value.Split(',', 2);
                    host = parts[0].Trim();
                    port ??= parts[1].Trim();
                }
                else if (value.Contains(':') && !value.StartsWith('['))
                {
                    // host:port (but not IPv6)
                    var parts = value.Split(':', 2);
                    host = parts[0].Trim();
                    port ??= parts[1].Trim();
                }
                else
                {
                    host = value.Trim();
                }
            }
            else if (PortKeys.Contains(key))
            {
                port = value.Trim();
            }
        }

        if (string.IsNullOrEmpty(host))
        {
            throw new ConfigurationException($"Connection string must contain a host key: {connStr}");
        }

        if (string.IsNullOrEmpty(port))
        {
            throw new ConfigurationException($"Connection string must contain a port: {connStr}");
        }

        ValidatePort(port, connStr);
        return new Endpoint(host, port);
    }

    /// <summary>
    /// Creates an Endpoint from explicit host and port parameters.
    /// </summary>
    public static Endpoint ParseParams(string host, string port)
    {
        if (string.IsNullOrWhiteSpace(host))
        {
            throw new ConfigurationException("host must not be empty");
        }

        if (string.IsNullOrWhiteSpace(port))
        {
            throw new ConfigurationException("port must not be empty");
        }

        var cleanHost = host;
        if (cleanHost.StartsWith('[') && cleanHost.EndsWith(']'))
        {
            cleanHost = cleanHost[1..^1];
        }

        ValidatePort(port, $"{host}:{port}");
        return new Endpoint(cleanHost, port);
    }

    private static List<ParsedConnection> ParseMultiHost(
        string hostPortPart, string scheme, DependencyType type)
    {
        var segments = hostPortPart.Split(',');
        var result = new List<ParsedConnection>(segments.Length);

        foreach (var segment in segments)
        {
            var trimmed = segment.Trim();
            if (string.IsNullOrEmpty(trimmed))
            {
                continue;
            }

            var (host, port) = ExtractHostPort(trimmed, scheme);
            ValidatePort(port, trimmed);
            result.Add(new ParsedConnection(host, port, type));
        }

        if (result.Count == 0)
        {
            throw new ConfigurationException($"No hosts found in multi-host URL: {hostPortPart}");
        }

        return result;
    }

    private static (string Host, string Port) ExtractHostPort(string hostPort, string scheme)
    {
        string host;
        string? port;

        // IPv6: [::1]:port
        if (hostPort.StartsWith('['))
        {
            var closeBracket = hostPort.IndexOf(']');
            if (closeBracket < 0)
            {
                throw new ConfigurationException($"Invalid IPv6 address: {hostPort}");
            }

            host = hostPort[1..closeBracket];
            var remainder = hostPort[(closeBracket + 1)..];
            port = remainder.StartsWith(':') ? remainder[1..] : DefaultPorts.ForScheme(scheme);
        }
        else
        {
            var lastColon = hostPort.LastIndexOf(':');
            if (lastColon > 0 && lastColon < hostPort.Length - 1)
            {
                host = hostPort[..lastColon];
                port = hostPort[(lastColon + 1)..];
            }
            else
            {
                host = lastColon == hostPort.Length - 1
                    ? hostPort[..lastColon]
                    : hostPort;
                port = DefaultPorts.ForScheme(scheme);
            }
        }

        if (string.IsNullOrEmpty(host))
        {
            throw new ConfigurationException($"Empty host in: {hostPort}");
        }

        if (port is null)
        {
            throw new ConfigurationException($"Cannot determine port for: {hostPort}");
        }

        return (host, port);
    }

    private static void ValidatePort(string portStr, string source)
    {
        if (!int.TryParse(portStr, out var port))
        {
            throw new ConfigurationException($"Invalid port: '{portStr}' in {source}");
        }

        if (port < 1 || port > 65535)
        {
            throw new ConfigurationException($"Port out of range (1-65535): {port} in {source}");
        }
    }

    private static Dictionary<string, string> ParseKeyValuePairs(string connStr)
    {
        var result = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        var pairs = connStr.Split(';');

        foreach (var pair in pairs)
        {
            var trimmed = pair.Trim();
            if (string.IsNullOrEmpty(trimmed))
            {
                continue;
            }

            var eq = trimmed.IndexOf('=');
            if (eq < 0)
            {
                continue;
            }

            var key = trimmed[..eq].Trim();
            var value = trimmed[(eq + 1)..].Trim();
            if (!string.IsNullOrEmpty(key))
            {
                result[key] = value;
            }
        }

        return result;
    }
}
