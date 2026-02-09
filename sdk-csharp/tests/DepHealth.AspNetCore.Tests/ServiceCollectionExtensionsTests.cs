using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;

namespace DepHealth.AspNetCore.Tests;

public class ServiceCollectionExtensionsTests
{
    [Fact]
    public void AddDepHealth_RegistersServices()
    {
        var services = new ServiceCollection();
        services.AddLogging();

        services.AddDepHealth("test-app", dh =>
        {
            dh.AddHttp("test-api", "http://localhost:8080", critical: true);
        });

        var sp = services.BuildServiceProvider();

        var monitor = sp.GetService<DepHealthMonitor>();
        Assert.NotNull(monitor);

        var hostedServices = sp.GetServices<IHostedService>();
        Assert.Contains(hostedServices, s => s is DepHealthHostedService);
    }
}
