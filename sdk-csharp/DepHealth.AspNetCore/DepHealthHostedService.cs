using Microsoft.Extensions.Hosting;

namespace DepHealth.AspNetCore;

/// <summary>
/// IHostedService for starting/stopping the CheckScheduler with the application lifecycle.
/// </summary>
public sealed class DepHealthHostedService : IHostedService
{
    private readonly DepHealthMonitor _monitor;

    /// <summary>
    /// Initializes a new instance of the <see cref="DepHealthHostedService"/> class.
    /// </summary>
    /// <param name="monitor">The dephealth monitor instance to manage.</param>
    public DepHealthHostedService(DepHealthMonitor monitor)
    {
        _monitor = monitor;
    }

    /// <summary>
    /// Starts the dephealth check scheduler when the application starts.
    /// </summary>
    public Task StartAsync(CancellationToken cancellationToken)
    {
        _monitor.Start();
        return Task.CompletedTask;
    }

    /// <summary>
    /// Stops the dephealth check scheduler when the application shuts down.
    /// </summary>
    public Task StopAsync(CancellationToken cancellationToken)
    {
        _monitor.Stop();
        return Task.CompletedTask;
    }
}
