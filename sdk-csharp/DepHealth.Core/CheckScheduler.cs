using System.Diagnostics;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;

namespace DepHealth;

/// <summary>
/// Scheduler for periodic dependency health checks.
/// </summary>
public sealed class CheckScheduler : IDisposable
{
    private readonly PrometheusExporter _metrics;
    private readonly ILogger _logger;
    private readonly List<ScheduledDep> _deps = [];
    private readonly Dictionary<string, EndpointState> _states = new();
    private readonly List<CancellationTokenSource> _cancellations = [];

    private volatile bool _started;
    private volatile bool _stopped;

    public CheckScheduler(PrometheusExporter metrics, ILogger? logger = null)
    {
        _metrics = metrics;
        _logger = logger ?? NullLogger.Instance;
    }

    /// <summary>
    /// Registers a dependency for periodic health checking.
    /// </summary>
    public void AddDependency(Dependency dependency, IHealthChecker checker)
    {
        if (_started)
        {
            throw new InvalidOperationException("Cannot add dependency after scheduler started");
        }

        _deps.Add(new ScheduledDep(dependency, checker));

        foreach (var ep in dependency.Endpoints)
        {
            var key = StateKey(dependency.Name, ep);
            _states[key] = new EndpointState();
        }
    }

    /// <summary>
    /// Starts periodic health checks.
    /// </summary>
    public void Start()
    {
        if (_started)
        {
            throw new InvalidOperationException("Scheduler already started");
        }

        if (_stopped)
        {
            throw new InvalidOperationException("Scheduler already stopped");
        }

        foreach (var dep in _deps)
        {
            foreach (var ep in dep.Dependency.Endpoints)
            {
                var config = dep.Dependency.Config;
                var cts = new CancellationTokenSource();
                _cancellations.Add(cts);

                _ = RunCheckLoopAsync(dep.Dependency, dep.Checker, ep, config, cts.Token);
            }
        }

        _started = true;
        _logger.LogInformation("dephealth: scheduler started, {DepsCount} dependencies, {EndpointsCount} endpoints",
            _deps.Count, _states.Count);
    }

    /// <summary>
    /// Stops all health checks.
    /// </summary>
    public void Stop()
    {
        if (!_started || _stopped)
        {
            return;
        }

        _stopped = true;

        foreach (var cts in _cancellations)
        {
            cts.Cancel();
            cts.Dispose();
        }

        _cancellations.Clear();
        _logger.LogInformation("dephealth: scheduler stopped");
    }

    /// <summary>
    /// Returns the current health status of all endpoints.
    /// </summary>
    public Dictionary<string, bool> Health()
    {
        var result = new Dictionary<string, bool>();
        foreach (var (key, state) in _states)
        {
            var healthy = state.Healthy;
            if (healthy is not null)
            {
                result[key] = healthy.Value;
            }
        }

        return result;
    }

    public void Dispose()
    {
        Stop();
    }

    private async Task RunCheckLoopAsync(
        Dependency dep, IHealthChecker checker, Endpoint ep,
        CheckConfig config, CancellationToken ct)
    {
        // Initial delay
        if (config.InitialDelay > TimeSpan.Zero)
        {
            try
            {
                await Task.Delay(config.InitialDelay, ct).ConfigureAwait(false);
            }
            catch (OperationCanceledException)
            {
                return;
            }
        }

        while (!ct.IsCancellationRequested)
        {
            RunCheck(dep, checker, ep, config);

            try
            {
                await Task.Delay(config.Interval, ct).ConfigureAwait(false);
            }
            catch (OperationCanceledException)
            {
                return;
            }
        }
    }

    private void RunCheck(Dependency dep, IHealthChecker checker, Endpoint ep, CheckConfig config)
    {
        var key = StateKey(dep.Name, ep);
        var state = _states[key];
        var sw = Stopwatch.StartNew();

        try
        {
            using var cts = new CancellationTokenSource(config.Timeout);
            checker.CheckAsync(ep, cts.Token).GetAwaiter().GetResult();

            sw.Stop();
            var duration = sw.Elapsed;

            var wasBefore = state.Healthy;
            state.RecordSuccess(config.SuccessThreshold);

            _metrics.SetHealth(dep, ep, 1.0);
            _metrics.ObserveLatency(dep, ep, duration);

            if (wasBefore is false && state.Healthy is true)
            {
                _logger.LogInformation("dephealth: {Name} [{Endpoint}] recovered", dep.Name, ep);
            }
        }
        catch (Exception e)
        {
            sw.Stop();
            var duration = sw.Elapsed;

            var wasBefore = state.Healthy;
            state.RecordFailure(config.FailureThreshold);

            _metrics.SetHealth(dep, ep, 0.0);
            _metrics.ObserveLatency(dep, ep, duration);

            if (wasBefore is null or true)
            {
                var msg = e.Message ?? e.GetType().Name;
                if (e.InnerException is not null)
                {
                    msg += $" (cause: {e.InnerException.Message ?? e.InnerException.GetType().Name})";
                }

                _logger.LogWarning("dephealth: {Name} [{Endpoint}] check failed: {Message}",
                    dep.Name, ep, msg);
            }

            if (wasBefore is true && state.Healthy is false)
            {
                _logger.LogError("dephealth: {Name} [{Endpoint}] became unhealthy", dep.Name, ep);
            }
        }
    }

    private static string StateKey(string name, Endpoint ep) => $"{name}:{ep.Host}:{ep.Port}";

    private sealed record ScheduledDep(Dependency Dependency, IHealthChecker Checker);
}
