using System.Net.Http.Headers;
using System.Net.Security;
using System.Reflection;

namespace DepHealth.Checks;

/// <summary>
/// HTTP health checker — performs a GET request to healthPath and expects a 2xx response.
/// </summary>
public sealed class HttpChecker : IHealthChecker, IDisposable
{
    private const string DefaultHealthPath = "/health";

    private static readonly string SdkVersion =
        typeof(HttpChecker).Assembly.GetCustomAttribute<AssemblyInformationalVersionAttribute>()
            ?.InformationalVersion ?? typeof(HttpChecker).Assembly.GetName().Version?.ToString(3) ?? "0.0.0";

    private readonly string _healthPath;
    private readonly bool _tlsEnabled;
    private readonly string? _hostHeader;
    private readonly HttpClient _client;

    /// <summary>Gets the dependency type for this checker.</summary>
    public DependencyType Type => DependencyType.Http;

    /// <summary>Creates a new instance of <see cref="HttpChecker"/>.</summary>
    /// <param name="healthPath">HTTP path to query for health status (default: <c>/health</c>).</param>
    /// <param name="tlsEnabled">Whether to use HTTPS instead of HTTP.</param>
    /// <param name="tlsSkipVerify">Whether to skip TLS certificate verification.</param>
    /// <param name="headers">Optional custom HTTP headers to include in the request.</param>
    /// <param name="bearerToken">Optional Bearer token for authentication.</param>
    /// <param name="basicAuthUsername">Optional username for HTTP Basic authentication.</param>
    /// <param name="basicAuthPassword">Optional password for HTTP Basic authentication.</param>
    /// <param name="hostHeader">Optional Host header override for routing through ingress/gateway.</param>
    public HttpChecker(
        string healthPath = DefaultHealthPath,
        bool tlsEnabled = false,
        bool tlsSkipVerify = false,
        IDictionary<string, string>? headers = null,
        string? bearerToken = null,
        string? basicAuthUsername = null,
        string? basicAuthPassword = null,
        string? hostHeader = null)
    {
        AuthValidation.ValidateNoConflicts(headers, "Authorization",
            bearerToken, basicAuthUsername, basicAuthPassword);
        AuthValidation.ValidateHostHeaderConflict(headers, hostHeader);
        _healthPath = healthPath;
        _tlsEnabled = tlsEnabled;
        _hostHeader = hostHeader;

        var handler = new SocketsHttpHandler
        {
            PooledConnectionLifetime = TimeSpan.FromMinutes(2)
        };

        if (tlsEnabled && (tlsSkipVerify || !string.IsNullOrEmpty(hostHeader)))
        {
            var sslOptions = new SslClientAuthenticationOptions();

            if (tlsSkipVerify)
            {
#pragma warning disable CA5359 // Intentional: user explicitly opted into skipping TLS verification
                sslOptions.RemoteCertificateValidationCallback = (_, _, _, _) => true;
#pragma warning restore CA5359
            }

            if (!string.IsNullOrEmpty(hostHeader))
            {
                sslOptions.TargetHost = hostHeader;
            }

            handler.SslOptions = sslOptions;
        }

        _client = new HttpClient(handler)
        {
            Timeout = Timeout.InfiniteTimeSpan
        };
        _client.DefaultRequestHeaders.UserAgent.Add(
            new ProductInfoHeaderValue("dephealth", SdkVersion));

        var resolvedHeaders = BuildResolvedHeaders(headers, bearerToken, basicAuthUsername, basicAuthPassword);
        foreach (var (key, value) in resolvedHeaders)
        {
            _client.DefaultRequestHeaders.TryAddWithoutValidation(key, value);
        }
    }

    /// <inheritdoc />
    public void Dispose() => _client.Dispose();

    /// <inheritdoc />
    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        var scheme = _tlsEnabled ? "https" : "http";
        var host = endpoint.Host.Contains(':', StringComparison.Ordinal) ? $"[{endpoint.Host}]" : endpoint.Host;
        var uri = new Uri($"{scheme}://{host}:{endpoint.Port}{_healthPath}");

        using var request = new HttpRequestMessage(HttpMethod.Get, uri);
        if (_hostHeader is not null)
        {
            request.Headers.Host = _hostHeader;
        }

        using var response = await _client.SendAsync(request, ct).ConfigureAwait(false);

        if (!response.IsSuccessStatusCode)
        {
            var statusCode = (int)response.StatusCode;

            // HTTP 401/403 → auth_error.
            if (statusCode is 401 or 403)
            {
                throw new Exceptions.CheckAuthException(
                    $"HTTP health check failed: status {statusCode}");
            }

            throw new Exceptions.UnhealthyException(
                $"HTTP health check failed: status {statusCode}",
                $"http_{statusCode}");
        }
    }

    private static Dictionary<string, string> BuildResolvedHeaders(
        IDictionary<string, string>? headers,
        string? bearerToken,
        string? basicAuthUsername,
        string? basicAuthPassword)
    {
        var resolved = new Dictionary<string, string>(StringComparer.Ordinal);

        if (headers is not null)
        {
            foreach (var (key, value) in headers)
            {
                resolved[key] = value;
            }
        }

        if (!string.IsNullOrEmpty(bearerToken))
        {
            resolved["Authorization"] = $"Bearer {bearerToken}";
        }

        if (!string.IsNullOrEmpty(basicAuthUsername) || !string.IsNullOrEmpty(basicAuthPassword))
        {
            var credentials = Convert.ToBase64String(
                System.Text.Encoding.UTF8.GetBytes($"{basicAuthUsername}:{basicAuthPassword}"));
            resolved["Authorization"] = $"Basic {credentials}";
        }

        return resolved;
    }
}
