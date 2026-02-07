using System.Net.Sockets;

namespace DepHealth.Checks;

/// <summary>
/// TCP health checker — устанавливает TCP-соединение к эндпоинту.
/// </summary>
public sealed class TcpChecker : IHealthChecker
{
    public DependencyType Type => DependencyType.Tcp;

    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        using var client = new TcpClient();
        await client.ConnectAsync(endpoint.Host, endpoint.PortAsInt(), ct).ConfigureAwait(false);
    }
}
