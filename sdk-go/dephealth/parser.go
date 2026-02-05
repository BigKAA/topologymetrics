package dephealth

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Default ports per dependency type (from spec/config-contract.md).
var DefaultPorts = map[string]string{
	"postgres":   "5432",
	"postgresql": "5432",
	"mysql":      "3306",
	"redis":      "6379",
	"rediss":     "6379",
	"amqp":       "5672",
	"amqps":      "5671",
	"http":       "80",
	"https":      "443",
	"grpc":       "443",
	"kafka":      "9092",
}

// schemeToType maps URL schemes to DependencyType.
var schemeToType = map[string]DependencyType{
	"postgres":   TypePostgres,
	"postgresql": TypePostgres,
	"mysql":      TypeMySQL,
	"redis":      TypeRedis,
	"rediss":     TypeRedis,
	"amqp":       TypeAMQP,
	"amqps":      TypeAMQP,
	"http":       TypeHTTP,
	"https":      TypeHTTP,
	"grpc":       TypeGRPC,
	"kafka":      TypeKafka,
}

// jdbcSubprotocolToType maps JDBC subprotocols to DependencyType.
var jdbcSubprotocolToType = map[string]DependencyType{
	"postgresql": TypePostgres,
	"mysql":      TypeMySQL,
}

// ParsedConnection holds the result of parsing a connection string/URL.
type ParsedConnection struct {
	Host     string
	Port     string
	ConnType DependencyType
}

// ParseURL parses a full URL and extracts host, port, and connection type.
// Supports schemes: postgres://, postgresql://, mysql://, redis://, rediss://,
// amqp://, amqps://, http://, https://, grpc://, kafka://.
//
// For URLs with multiple hosts (e.g. kafka://broker-0:9092,broker-1:9092),
// returns multiple ParsedConnection entries.
func ParseURL(rawURL string) ([]ParsedConnection, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("empty URL")
	}

	// Handle kafka:// multi-host URLs: kafka://host1:port1,host2:port2
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme == "" {
		return nil, fmt.Errorf("missing scheme in URL %q", rawURL)
	}

	connType, ok := schemeToType[scheme]
	if !ok {
		return nil, fmt.Errorf("unsupported URL scheme %q", scheme)
	}

	defaultPort := DefaultPorts[scheme]

	// Check for multi-host (comma-separated in host part)
	hostPart := u.Host
	if strings.Contains(hostPart, ",") {
		return parseMultiHost(hostPart, defaultPort, connType)
	}

	host, port, err := extractHostPort(hostPart, defaultPort)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	return []ParsedConnection{{Host: host, Port: port, ConnType: connType}}, nil
}

// parseMultiHost handles comma-separated host:port pairs.
func parseMultiHost(hostPart, defaultPort string, connType DependencyType) ([]ParsedConnection, error) {
	parts := strings.Split(hostPart, ",")
	results := make([]ParsedConnection, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		host, port, err := extractHostPort(part, defaultPort)
		if err != nil {
			return nil, fmt.Errorf("invalid host %q: %w", part, err)
		}

		results = append(results, ParsedConnection{Host: host, Port: port, ConnType: connType})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no hosts found in %q", hostPart)
	}

	return results, nil
}

// extractHostPort splits a host:port string, applying default port if missing.
// Handles IPv6 addresses in brackets: [::1]:5432 → host=::1, port=5432.
func extractHostPort(hostPort, defaultPort string) (string, string, error) {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		// No port specified — use default
		host = hostPort
		port = defaultPort

		// Remove brackets from IPv6 if present
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
			host = host[1 : len(host)-1]
		}
	}

	if host == "" {
		return "", "", fmt.Errorf("empty host")
	}

	if port == "" {
		port = defaultPort
	}

	if err := validatePort(port); err != nil {
		return "", "", err
	}

	return host, port, nil
}

// ParseConnectionString parses a Key=Value;Key=Value connection string.
// Supported host keys (case-insensitive): Host, Server, Data Source, Address, Addr, Network Address.
// Supported port keys: Port.
// Also handles Server=host,port and Host=host:port formats.
func ParseConnectionString(connStr string) (string, string, error) {
	if connStr == "" {
		return "", "", fmt.Errorf("empty connection string")
	}

	pairs := parseKeyValuePairs(connStr)

	host := findValue(pairs, "host", "server", "data source", "address", "addr", "network address")
	port := findValue(pairs, "port")

	if host == "" {
		return "", "", fmt.Errorf("missing host in connection string")
	}

	// Handle Server=host,port (SQL Server convention)
	if port == "" && strings.Contains(host, ",") {
		parts := strings.SplitN(host, ",", 2)
		host = strings.TrimSpace(parts[0])
		port = strings.TrimSpace(parts[1])
	}

	// Handle Host=host:port
	if port == "" && strings.Contains(host, ":") {
		h, p, err := net.SplitHostPort(host)
		if err == nil {
			host = h
			port = p
		}
	}

	// Remove brackets from IPv6
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	if port != "" {
		if err := validatePort(port); err != nil {
			return "", "", fmt.Errorf("invalid port in connection string: %w", err)
		}
	}

	return host, port, nil
}

// parseKeyValuePairs splits a connection string into key-value pairs.
func parseKeyValuePairs(connStr string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(connStr, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])
		result[strings.ToLower(key)] = value
	}
	return result
}

// findValue looks up first matching key (case-insensitive) from pairs.
func findValue(pairs map[string]string, keys ...string) string {
	for _, key := range keys {
		if v, ok := pairs[key]; ok && v != "" {
			return v
		}
	}
	return ""
}

// ParseJDBC parses a JDBC URL: jdbc:subprotocol://host:port/db.
func ParseJDBC(jdbcURL string) ([]ParsedConnection, error) {
	if jdbcURL == "" {
		return nil, fmt.Errorf("empty JDBC URL")
	}

	if !strings.HasPrefix(strings.ToLower(jdbcURL), "jdbc:") {
		return nil, fmt.Errorf("invalid JDBC URL %q: missing jdbc: prefix", jdbcURL)
	}

	// Remove "jdbc:" prefix and parse as regular URL
	innerURL := jdbcURL[5:]

	u, err := url.Parse(innerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid JDBC URL %q: %w", jdbcURL, err)
	}

	subprotocol := strings.ToLower(u.Scheme)
	connType, ok := jdbcSubprotocolToType[subprotocol]
	if !ok {
		return nil, fmt.Errorf("unsupported JDBC subprotocol %q", subprotocol)
	}

	defaultPort := DefaultPorts[subprotocol]
	host, port, err := extractHostPort(u.Host, defaultPort)
	if err != nil {
		return nil, fmt.Errorf("invalid JDBC URL %q: %w", jdbcURL, err)
	}

	return []ParsedConnection{{Host: host, Port: port, ConnType: connType}}, nil
}

// ParseParams creates an Endpoint from explicit host and port parameters.
func ParseParams(host, port string) (Endpoint, error) {
	if host == "" {
		return Endpoint{}, fmt.Errorf("empty host")
	}
	if port == "" {
		return Endpoint{}, fmt.Errorf("empty port")
	}

	// Remove brackets from IPv6
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	if err := validatePort(port); err != nil {
		return Endpoint{}, err
	}

	return Endpoint{Host: host, Port: port}, nil
}

// validatePort checks that port is a valid number in 1-65535.
func validatePort(port string) error {
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port %q: %w", port, err)
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("port %d out of range 1-65535", p)
	}
	return nil
}
