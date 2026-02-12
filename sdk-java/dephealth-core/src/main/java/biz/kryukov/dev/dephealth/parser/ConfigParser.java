package biz.kryukov.dev.dephealth.parser;

import biz.kryukov.dev.dephealth.ConfigurationException;
import biz.kryukov.dev.dephealth.DefaultPorts;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Parser for URLs, JDBC URLs, connection strings, and explicit host/port parameters.
 */
public final class ConfigParser {

    private ConfigParser() {}

    private static final Map<String, DependencyType> SCHEME_TO_TYPE = Map.ofEntries(
            Map.entry("postgres", DependencyType.POSTGRES),
            Map.entry("postgresql", DependencyType.POSTGRES),
            Map.entry("mysql", DependencyType.MYSQL),
            Map.entry("redis", DependencyType.REDIS),
            Map.entry("rediss", DependencyType.REDIS),
            Map.entry("amqp", DependencyType.AMQP),
            Map.entry("amqps", DependencyType.AMQP),
            Map.entry("http", DependencyType.HTTP),
            Map.entry("https", DependencyType.HTTP),
            Map.entry("grpc", DependencyType.GRPC),
            Map.entry("kafka", DependencyType.KAFKA)
    );

    private static final Map<String, DependencyType> JDBC_SUBPROTOCOL_TO_TYPE = Map.of(
            "postgresql", DependencyType.POSTGRES,
            "mysql", DependencyType.MYSQL
    );

    private static final Set<String> HOST_KEYS = Set.of(
            "host", "server", "data source", "address", "addr", "network address"
    );

    private static final Set<String> PORT_KEYS = Set.of("port");

    /**
     * Parses a URL (supports multi-host for Kafka, IPv6).
     *
     * @param rawUrl URL to parse
     * @return list of parsed connections (one or more for multi-host)
     */
    public static List<ParsedConnection> parseUrl(String rawUrl) {
        if (rawUrl == null || rawUrl.isBlank()) {
            throw new ConfigurationException("URL must not be empty");
        }

        // Determine the scheme
        int colonSlashSlash = rawUrl.indexOf("://");
        if (colonSlashSlash < 0) {
            throw new ConfigurationException("URL must have a scheme (e.g. http://): " + rawUrl);
        }
        String scheme = rawUrl.substring(0, colonSlashSlash).toLowerCase();
        DependencyType type = SCHEME_TO_TYPE.get(scheme);
        if (type == null) {
            throw new ConfigurationException("Unsupported URL scheme: " + scheme);
        }

        String rest = rawUrl.substring(colonSlashSlash + 3);

        // Strip userinfo (user:pass@)
        int atSign = rest.indexOf('@');
        if (atSign >= 0) {
            rest = rest.substring(atSign + 1);
        }

        // Strip path/query/fragment
        int pathStart = rest.indexOf('/');
        String hostPortPart = pathStart >= 0 ? rest.substring(0, pathStart) : rest;
        // Also strip query without path
        int queryStart = hostPortPart.indexOf('?');
        if (queryStart >= 0) {
            hostPortPart = hostPortPart.substring(0, queryStart);
        }

        // Multi-host (Kafka): host1:port1,host2:port2
        if (type == DependencyType.KAFKA || hostPortPart.contains(",")) {
            return parseMultiHost(hostPortPart, scheme, type);
        }

        String[] hp = extractHostPort(hostPortPart, scheme);
        validatePort(hp[1], rawUrl);
        return List.of(new ParsedConnection(hp[0], hp[1], type));
    }

    /**
     * Parses a JDBC URL: jdbc:subprotocol://host:port/db
     */
    public static List<ParsedConnection> parseJdbc(String jdbcUrl) {
        if (jdbcUrl == null || jdbcUrl.isBlank()) {
            throw new ConfigurationException("JDBC URL must not be empty");
        }
        if (!jdbcUrl.toLowerCase().startsWith("jdbc:")) {
            throw new ConfigurationException("JDBC URL must start with 'jdbc:': " + jdbcUrl);
        }

        String withoutJdbc = jdbcUrl.substring(5); // strip "jdbc:"

        // Determine the subprotocol
        int colonSlashSlash = withoutJdbc.indexOf("://");
        if (colonSlashSlash < 0) {
            throw new ConfigurationException("JDBC URL must contain ://: " + jdbcUrl);
        }
        String subprotocol = withoutJdbc.substring(0, colonSlashSlash).toLowerCase();
        DependencyType type = JDBC_SUBPROTOCOL_TO_TYPE.get(subprotocol);
        if (type == null) {
            throw new ConfigurationException("Unsupported JDBC subprotocol: " + subprotocol);
        }

        // Parse as a regular URL
        String normalizedUrl = subprotocol + withoutJdbc.substring(colonSlashSlash);
        return parseUrl(normalizedUrl);
    }

    /**
     * Parses a connection string in key=value;key=value format.
     *
     * @return Endpoint with host and port
     */
    public static Endpoint parseConnectionString(String connStr) {
        if (connStr == null || connStr.isBlank()) {
            throw new ConfigurationException("Connection string must not be empty");
        }

        Map<String, String> pairs = parseKeyValuePairs(connStr);

        String host = null;
        String port = null;

        for (Map.Entry<String, String> entry : pairs.entrySet()) {
            String key = entry.getKey().toLowerCase();
            String value = entry.getValue();

            if (HOST_KEYS.contains(key)) {
                // host may contain port via comma (SQL Server) or colon
                if (value.contains(",")) {
                    // SQL Server: Server=host,port
                    String[] parts = value.split(",", 2);
                    host = parts[0].trim();
                    if (port == null) {
                        port = parts[1].trim();
                    }
                } else if (value.contains(":") && !value.startsWith("[")) {
                    // host:port (but not IPv6)
                    String[] parts = value.split(":", 2);
                    host = parts[0].trim();
                    if (port == null) {
                        port = parts[1].trim();
                    }
                } else {
                    host = value.trim();
                }
            } else if (PORT_KEYS.contains(key)) {
                port = value.trim();
            }
        }

        if (host == null || host.isEmpty()) {
            throw new ConfigurationException("Connection string must contain a host key: " + connStr);
        }
        if (port == null || port.isEmpty()) {
            throw new ConfigurationException("Connection string must contain a port: " + connStr);
        }

        validatePort(port, connStr);
        return new Endpoint(host, port);
    }

    /**
     * Creates an Endpoint from explicit host and port parameters.
     */
    public static Endpoint parseParams(String host, String port) {
        if (host == null || host.isBlank()) {
            throw new ConfigurationException("host must not be empty");
        }
        if (port == null || port.isBlank()) {
            throw new ConfigurationException("port must not be empty");
        }

        // Strip IPv6 brackets
        String cleanHost = host;
        if (cleanHost.startsWith("[") && cleanHost.endsWith("]")) {
            cleanHost = cleanHost.substring(1, cleanHost.length() - 1);
        }

        validatePort(port, host + ":" + port);
        return new Endpoint(cleanHost, port);
    }

    private static List<ParsedConnection> parseMultiHost(String hostPortPart, String scheme,
                                                         DependencyType type) {
        String[] segments = hostPortPart.split(",");
        List<ParsedConnection> result = new ArrayList<>(segments.length);
        for (String segment : segments) {
            String trimmed = segment.trim();
            if (trimmed.isEmpty()) {
                continue;
            }
            String[] hp = extractHostPort(trimmed, scheme);
            validatePort(hp[1], trimmed);
            result.add(new ParsedConnection(hp[0], hp[1], type));
        }
        if (result.isEmpty()) {
            throw new ConfigurationException("No hosts found in multi-host URL: " + hostPortPart);
        }
        return result;
    }

    private static String[] extractHostPort(String hostPort, String scheme) {
        String host;
        String port;

        // IPv6: [::1]:port
        if (hostPort.startsWith("[")) {
            int closeBracket = hostPort.indexOf(']');
            if (closeBracket < 0) {
                throw new ConfigurationException("Invalid IPv6 address: " + hostPort);
            }
            host = hostPort.substring(1, closeBracket);
            String remainder = hostPort.substring(closeBracket + 1);
            if (remainder.startsWith(":")) {
                port = remainder.substring(1);
            } else {
                port = DefaultPorts.forScheme(scheme);
            }
        } else {
            int lastColon = hostPort.lastIndexOf(':');
            if (lastColon > 0 && lastColon < hostPort.length() - 1) {
                host = hostPort.substring(0, lastColon);
                port = hostPort.substring(lastColon + 1);
            } else {
                host = lastColon == hostPort.length() - 1
                        ? hostPort.substring(0, lastColon) : hostPort;
                port = DefaultPorts.forScheme(scheme);
            }
        }

        if (host == null || host.isEmpty()) {
            throw new ConfigurationException("Empty host in: " + hostPort);
        }
        if (port == null) {
            throw new ConfigurationException("Cannot determine port for: " + hostPort);
        }

        return new String[]{host, port};
    }

    private static void validatePort(String portStr, String source) {
        try {
            int port = Integer.parseInt(portStr);
            if (port < 1 || port > 65535) {
                throw new ConfigurationException(
                        "Port out of range (1-65535): " + port + " in " + source);
            }
        } catch (NumberFormatException e) {
            throw new ConfigurationException("Invalid port: '" + portStr + "' in " + source);
        }
    }

    private static Map<String, String> parseKeyValuePairs(String connStr) {
        Map<String, String> result = new LinkedHashMap<>();
        String[] pairs = connStr.split(";");
        for (String pair : pairs) {
            String trimmed = pair.trim();
            if (trimmed.isEmpty()) {
                continue;
            }
            int eq = trimmed.indexOf('=');
            if (eq < 0) {
                continue;
            }
            String key = trimmed.substring(0, eq).trim();
            String value = trimmed.substring(eq + 1).trim();
            if (!key.isEmpty()) {
                result.put(key, value);
            }
        }
        return result;
    }
}
