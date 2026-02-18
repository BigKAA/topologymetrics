namespace DepHealth.AspNetCore.Tests;

public class DepHealthHostedServiceTests
{
    [Fact]
    public async Task StartAsync_StartsMonitor()
    {
        using var monitor = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddHttp("test", "http://localhost:8080", critical: true)
            .Build();

        var service = new DepHealthHostedService(monitor);

        await service.StartAsync(CancellationToken.None);
        // If Start() throws an exception — the test will fail

        await service.StopAsync(CancellationToken.None);
    }
}
