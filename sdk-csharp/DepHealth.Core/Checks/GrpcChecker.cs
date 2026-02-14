using Grpc.Health.V1;
using Grpc.Net.Client;

namespace DepHealth.Checks;

/// <summary>
/// gRPC health checker — uses the gRPC Health Check Protocol.
/// </summary>
public sealed class GrpcChecker : IHealthChecker
{
    private readonly bool _tlsEnabled;

    public DependencyType Type => DependencyType.Grpc;

    public GrpcChecker(bool tlsEnabled = false)
    {
        _tlsEnabled = tlsEnabled;
    }

    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        var scheme = _tlsEnabled ? "https" : "http";
        var host = endpoint.Host.Contains(':') ? $"[{endpoint.Host}]" : endpoint.Host;
        var address = $"{scheme}://{host}:{endpoint.Port}";

        using var channel = GrpcChannel.ForAddress(address, new GrpcChannelOptions
        {
            HttpHandler = new SocketsHttpHandler
            {
                EnableMultipleHttp2Connections = true
            }
        });

        var client = new Health.HealthClient(channel);
        var response = await client.CheckAsync(
            new HealthCheckRequest(),
            cancellationToken: ct).ConfigureAwait(false);

        if (response.Status != HealthCheckResponse.Types.ServingStatus.Serving)
        {
            var detail = response.Status == HealthCheckResponse.Types.ServingStatus.NotServing
                ? "grpc_not_serving"
                : "grpc_unknown";
            throw new Exceptions.UnhealthyException(
                $"gRPC health check returned: {response.Status}", detail);
        }
    }
}
