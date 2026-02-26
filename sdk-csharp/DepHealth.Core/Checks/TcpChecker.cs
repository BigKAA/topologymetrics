using System.Net.Sockets;

namespace DepHealth.Checks;

/// <summary>
/// TCP health checker — establishes a TCP connection to the endpoint.
/// </summary>
public sealed class TcpChecker : IHealthChecker
{
    /// <summary>Gets the dependency type for this checker.</summary>
    public DependencyType Type => DependencyType.Tcp;

    /// <inheritdoc />
    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        using var client = new TcpClient();
        await client.ConnectAsync(endpoint.Host, endpoint.PortAsInt(), ct).ConfigureAwait(false);
    }
}
