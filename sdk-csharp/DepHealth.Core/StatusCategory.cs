namespace DepHealth;

/// <summary>
/// Status category constants for health check classification.
/// </summary>
public static class StatusCategory
{
    public const string Ok = "ok";
    public const string Timeout = "timeout";
    public const string ConnectionError = "connection_error";
    public const string DnsError = "dns_error";
    public const string AuthError = "auth_error";
    public const string TlsError = "tls_error";
    public const string Unhealthy = "unhealthy";
    public const string Error = "error";

    /// <summary>All status category values in specification order.</summary>
    public static readonly string[] All =
        [Ok, Timeout, ConnectionError, DnsError, AuthError, TlsError, Unhealthy, Error];
}
