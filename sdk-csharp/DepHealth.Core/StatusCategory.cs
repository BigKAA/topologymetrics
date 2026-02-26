namespace DepHealth;

/// <summary>
/// Status category constants for health check classification.
/// </summary>
public static class StatusCategory
{
    /// <summary>Dependency is healthy and responding normally.</summary>
    public const string Ok = "ok";

    /// <summary>Health check timed out waiting for a response.</summary>
    public const string Timeout = "timeout";

    /// <summary>Connection to the dependency was refused or unreachable.</summary>
    public const string ConnectionError = "connection_error";

    /// <summary>DNS resolution failed for the dependency host.</summary>
    public const string DnsError = "dns_error";

    /// <summary>Authentication or authorization failed.</summary>
    public const string AuthError = "auth_error";

    /// <summary>TLS/SSL handshake or certificate validation failed.</summary>
    public const string TlsError = "tls_error";

    /// <summary>Dependency is reachable but reported an unhealthy state.</summary>
    public const string Unhealthy = "unhealthy";

    /// <summary>An unclassified error occurred during the health check.</summary>
    public const string Error = "error";

    /// <summary>Check status is not yet known (initial state).</summary>
    public const string Unknown = "unknown";

    /// <summary>All status category values in specification order (excludes Unknown).</summary>
    public static readonly string[] All =
        [Ok, Timeout, ConnectionError, DnsError, AuthError, TlsError, Unhealthy, Error];
}
