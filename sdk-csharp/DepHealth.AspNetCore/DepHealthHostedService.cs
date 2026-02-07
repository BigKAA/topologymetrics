using Microsoft.Extensions.Hosting;

namespace DepHealth.AspNetCore;

/// <summary>
/// IHostedService для запуска/остановки CheckScheduler вместе с приложением.
/// </summary>
public sealed class DepHealthHostedService : IHostedService
{
    private readonly DepHealthMonitor _monitor;

    public DepHealthHostedService(DepHealthMonitor monitor)
    {
        _monitor = monitor;
    }

    public Task StartAsync(CancellationToken cancellationToken)
    {
        _monitor.Start();
        return Task.CompletedTask;
    }

    public Task StopAsync(CancellationToken cancellationToken)
    {
        _monitor.Stop();
        return Task.CompletedTask;
    }
}
