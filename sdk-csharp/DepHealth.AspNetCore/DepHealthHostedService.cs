using Microsoft.Extensions.Hosting;

namespace DepHealth.AspNetCore;

/// <summary>
/// IHostedService for starting/stopping the CheckScheduler with the application lifecycle.
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
