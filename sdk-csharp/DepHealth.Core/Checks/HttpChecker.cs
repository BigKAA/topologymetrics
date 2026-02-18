using System.Net.Http.Headers;

namespace DepHealth.Checks;

/// <summary>
/// HTTP health checker — performs a GET request to healthPath and expects a 2xx response.
/// </summary>
public sealed class HttpChecker : IHealthChecker
{
    private const string DefaultHealthPath = "/health";

    private readonly string _healthPath;
    private readonly bool _tlsEnabled;
    private readonly bool _tlsSkipVerify;
    private readonly IReadOnlyDictionary<string, string> _headers;

    public DependencyType Type => DependencyType.Http;

    public HttpChecker(
        string healthPath = DefaultHealthPath,
        bool tlsEnabled = false,
        bool tlsSkipVerify = false,
        IDictionary<string, string>? headers = null,
        string? bearerToken = null,
        string? basicAuthUsername = null,
        string? basicAuthPassword = null)
    {
        ValidateAuthConflicts(headers, bearerToken, basicAuthUsername, basicAuthPassword);
        _healthPath = healthPath;
        _tlsEnabled = tlsEnabled;
        _tlsSkipVerify = tlsSkipVerify;
        _headers = BuildResolvedHeaders(headers, bearerToken, basicAuthUsername, basicAuthPassword);
    }

    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        var scheme = _tlsEnabled ? "https" : "http";
        var host = endpoint.Host.Contains(':') ? $"[{endpoint.Host}]" : endpoint.Host;
        var uri = new Uri($"{scheme}://{host}:{endpoint.Port}{_healthPath}");

        var handler = new HttpClientHandler();
        if (_tlsEnabled && _tlsSkipVerify)
        {
            handler.ServerCertificateCustomValidationCallback =
                (_, _, _, _) => true;
        }

        using var client = new HttpClient(handler);
        client.Timeout = Timeout.InfiniteTimeSpan;
        client.DefaultRequestHeaders.UserAgent.Add(
            new ProductInfoHeaderValue("dephealth", "0.5.0"));

        foreach (var (key, value) in _headers)
        {
            client.DefaultRequestHeaders.TryAddWithoutValidation(key, value);
        }

        using var response = await client.GetAsync(uri, ct).ConfigureAwait(false);

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

    internal static void ValidateAuthConflicts(
        IDictionary<string, string>? headers,
        string? bearerToken,
        string? basicAuthUsername,
        string? basicAuthPassword)
    {
        var methods = 0;

        if (!string.IsNullOrEmpty(bearerToken))
        {
            methods++;
        }

        if (!string.IsNullOrEmpty(basicAuthUsername) || !string.IsNullOrEmpty(basicAuthPassword))
        {
            methods++;
        }

        if (headers is not null)
        {
            foreach (var key in headers.Keys)
            {
                if (key.Equals("Authorization", StringComparison.OrdinalIgnoreCase))
                {
                    methods++;
                    break;
                }
            }
        }

        if (methods > 1)
        {
            throw new ValidationException(
                "conflicting auth methods: specify only one of bearerToken, basicAuth, or Authorization header");
        }
    }

    private static IReadOnlyDictionary<string, string> BuildResolvedHeaders(
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
