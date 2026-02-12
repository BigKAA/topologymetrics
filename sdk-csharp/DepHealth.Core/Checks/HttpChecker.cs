using System.Net.Http.Headers;
using System.Net.Security;

namespace DepHealth.Checks;

/// <summary>
/// HTTP health checker — performs a GET request to healthPath and expects a 2xx response.
/// </summary>
public sealed class HttpChecker : IHealthChecker
{
    private const string DefaultHealthPath = "/health";
    private const string UserAgentValue = "dephealth/0.2.1";

    private readonly string _healthPath;
    private readonly bool _tlsEnabled;
    private readonly bool _tlsSkipVerify;

    public DependencyType Type => DependencyType.Http;

    public HttpChecker(string healthPath = DefaultHealthPath, bool tlsEnabled = false, bool tlsSkipVerify = false)
    {
        _healthPath = healthPath;
        _tlsEnabled = tlsEnabled;
        _tlsSkipVerify = tlsSkipVerify;
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
        client.DefaultRequestHeaders.UserAgent.Add(
            new ProductInfoHeaderValue("dephealth", "0.2.1"));

        using var response = await client.GetAsync(uri, ct).ConfigureAwait(false);

        if (!response.IsSuccessStatusCode)
        {
            throw new Exceptions.UnhealthyException(
                $"HTTP health check failed: status {(int)response.StatusCode}");
        }
    }
}
