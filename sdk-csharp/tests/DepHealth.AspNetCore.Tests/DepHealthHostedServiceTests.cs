using Moq;

namespace DepHealth.AspNetCore.Tests;

public class DepHealthHostedServiceTests
{
    [Fact]
    public async Task StartAsync_StartsMonitor()
    {
        using var monitor = DepHealthMonitor.CreateBuilder()
            .AddHttp("test", "http://localhost:8080")
            .Build();

        var service = new DepHealthHostedService(monitor);

        await service.StartAsync(CancellationToken.None);
        // Если Start() выброшено исключение — тест упадёт

        await service.StopAsync(CancellationToken.None);
    }
}
