using Grpc.Core;
using Grpc.Health.V1;
using Grpc.Net.Client;

namespace DepHealth.Checks;

/// <summary>
/// gRPC health checker — uses the gRPC Health Check Protocol.
/// </summary>
public sealed class GrpcChecker : IHealthChecker
{
    private readonly bool _tlsEnabled;
    private readonly Metadata? _metadata;

    /// <summary>Gets the dependency type for this checker.</summary>
    public DependencyType Type => DependencyType.Grpc;

    /// <summary>Creates a new instance of <see cref="GrpcChecker"/>.</summary>
    /// <param name="tlsEnabled">Whether to use HTTPS (TLS) for the gRPC channel.</param>
    /// <param name="metadata">Optional gRPC metadata entries to include in the health check call.</param>
    /// <param name="bearerToken">Optional Bearer token for authentication.</param>
    /// <param name="basicAuthUsername">Optional username for Basic authentication.</param>
    /// <param name="basicAuthPassword">Optional password for Basic authentication.</param>
    public GrpcChecker(
        bool tlsEnabled = false,
        IDictionary<string, string>? metadata = null,
        string? bearerToken = null,
        string? basicAuthUsername = null,
        string? basicAuthPassword = null)
    {
        ValidateAuthConflicts(metadata, bearerToken, basicAuthUsername, basicAuthPassword);
        _tlsEnabled = tlsEnabled;
        _metadata = BuildResolvedMetadata(metadata, bearerToken, basicAuthUsername, basicAuthPassword);
    }

    /// <inheritdoc />
    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        var scheme = _tlsEnabled ? "https" : "http";
        var host = endpoint.Host.Contains(':', StringComparison.Ordinal) ? $"[{endpoint.Host}]" : endpoint.Host;
        var address = $"{scheme}://{host}:{endpoint.Port}";

        using var channel = GrpcChannel.ForAddress(address, new GrpcChannelOptions
        {
            HttpHandler = new SocketsHttpHandler
            {
                EnableMultipleHttp2Connections = true,
                ConnectTimeout = Timeout.InfiniteTimeSpan
            }
        });

        var client = new Health.HealthClient(channel);

        var callOptions = new CallOptions(cancellationToken: ct);
        if (_metadata is not null)
        {
            callOptions = callOptions.WithHeaders(_metadata);
        }

        HealthCheckResponse response;
        try
        {
            response = await client.CheckAsync(
                new HealthCheckRequest(), callOptions).ConfigureAwait(false);
        }
        catch (RpcException ex) when (ex.StatusCode is StatusCode.Unauthenticated or StatusCode.PermissionDenied)
        {
            throw new Exceptions.CheckAuthException(
                $"gRPC health check to {address}: {ex.Status.Detail}", ex);
        }

        if (response.Status != HealthCheckResponse.Types.ServingStatus.Serving)
        {
            var detail = response.Status == HealthCheckResponse.Types.ServingStatus.NotServing
                ? "grpc_not_serving"
                : "grpc_unknown";
            throw new Exceptions.UnhealthyException(
                $"gRPC health check returned: {response.Status}", detail);
        }
    }

    internal static void ValidateAuthConflicts(
        IDictionary<string, string>? metadata,
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

        if (metadata is not null
            && metadata.Keys.Any(key => key.Equals("authorization", StringComparison.OrdinalIgnoreCase)))
        {
            methods++;
        }

        if (methods > 1)
        {
            throw new ValidationException(
                "conflicting auth methods: specify only one of bearerToken, basicAuth, or authorization metadata");
        }
    }

    private static Metadata? BuildResolvedMetadata(
        IDictionary<string, string>? metadata,
        string? bearerToken,
        string? basicAuthUsername,
        string? basicAuthPassword)
    {
        var resolved = new Dictionary<string, string>(StringComparer.Ordinal);

        if (metadata is not null)
        {
            foreach (var (key, value) in metadata)
            {
                resolved[key] = value;
            }
        }

        if (!string.IsNullOrEmpty(bearerToken))
        {
            resolved["authorization"] = $"Bearer {bearerToken}";
        }

        if (!string.IsNullOrEmpty(basicAuthUsername) || !string.IsNullOrEmpty(basicAuthPassword))
        {
            var credentials = Convert.ToBase64String(
                System.Text.Encoding.UTF8.GetBytes($"{basicAuthUsername}:{basicAuthPassword}"));
            resolved["authorization"] = $"Basic {credentials}";
        }

        if (resolved.Count == 0)
        {
            return null;
        }

        var grpcMetadata = new Metadata();
        foreach (var (key, value) in resolved)
        {
            grpcMetadata.Add(key, value);
        }

        return grpcMetadata;
    }
}
