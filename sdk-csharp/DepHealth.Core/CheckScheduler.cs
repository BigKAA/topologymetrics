using System.Collections.Concurrent;
using System.Diagnostics;
using DepHealth.Exceptions;
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
    private readonly CheckConfig _globalConfig;
    private readonly List<ScheduledDep> _deps = [];
    private readonly ConcurrentDictionary<string, EndpointState> _states = new();
    private readonly ConcurrentDictionary<string, CancellationTokenSource> _cancellations = new();
    private readonly object _mutationLock = new();

    private CancellationTokenSource? _globalCts;
    private volatile bool _started;
    private volatile bool _stopped;

    /// <summary>Creates a new CheckScheduler.</summary>
    /// <param name="metrics">Prometheus metrics exporter.</param>
    /// <param name="globalConfig">Global check configuration.</param>
    /// <param name="logger">Optional logger for diagnostic messages.</param>
    public CheckScheduler(PrometheusExporter metrics, CheckConfig globalConfig, ILogger? logger = null)
    {
        _metrics = metrics;
        _globalConfig = globalConfig;
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
            var state = new EndpointState();
            state.SetStaticFields(
                dependency.Name,
                dependency.Type.Label(),
                ep.Host,
                ep.Port,
                dependency.Critical,
                ep.Labels);
            _states[key] = state;
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

        _globalCts = new CancellationTokenSource();

        foreach (var dep in _deps)
        {
            foreach (var ep in dep.Dependency.Endpoints)
            {
                var config = dep.Dependency.Config;
                var key = StateKey(dep.Dependency.Name, ep);
                var cts = CancellationTokenSource.CreateLinkedTokenSource(_globalCts.Token);
                _cancellations[key] = cts;

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

        // Cancel all via global token — linked CTS will propagate
        _globalCts?.Cancel();
        _globalCts?.Dispose();

        foreach (var (_, cts) in _cancellations)
        {
            cts.Dispose();
        }

        _cancellations.Clear();
        _logger.LogInformation("dephealth: scheduler stopped");
    }

    /// <summary>
    /// Returns detailed health status for all endpoints, including UNKNOWN ones.
    /// </summary>
    public Dictionary<string, EndpointStatus> HealthDetails()
    {
        var result = new Dictionary<string, EndpointStatus>();
        foreach (var (key, state) in _states)
        {
            result[key] = state.ToEndpointStatus();
        }

        return result;
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

    /// <summary>
    /// Dynamically adds a new endpoint after the scheduler has started.
    /// Idempotent: if the endpoint already exists, does nothing.
    /// </summary>
    public void AddEndpoint(string depName, DependencyType depType,
        bool critical, Endpoint ep, IHealthChecker checker)
    {
        lock (_mutationLock)
        {
            if (!_started || _stopped)
            {
                throw new InvalidOperationException(
                    "Cannot add endpoint: scheduler must be started and not stopped");
            }

            var key = StateKey(depName, ep);
            if (_states.ContainsKey(key))
            {
                return; // idempotent
            }

            var dep = Dependency.CreateBuilder(depName, depType)
                .WithCritical(critical)
                .WithEndpoint(ep)
                .WithConfig(_globalConfig)
                .Build();

            var state = new EndpointState();
            state.SetStaticFields(depName, depType.Label(), ep.Host, ep.Port, critical, ep.Labels);
            _states[key] = state;

            var cts = CancellationTokenSource.CreateLinkedTokenSource(_globalCts!.Token);
            _cancellations[key] = cts;

            _ = RunCheckLoopAsync(dep, checker, ep, _globalConfig, cts.Token);

            _logger.LogInformation(
                "dephealth: dynamically added endpoint {Key}", key);
        }
    }

    /// <summary>
    /// Dynamically removes an endpoint after the scheduler has started.
    /// Idempotent: if the endpoint does not exist, does nothing.
    /// </summary>
    public void RemoveEndpoint(string depName, string host, string port)
    {
        lock (_mutationLock)
        {
            if (!_started)
            {
                throw new InvalidOperationException(
                    "Cannot remove endpoint: scheduler must be started");
            }

            var key = $"{depName}:{host}:{port}";
            if (!_states.TryRemove(key, out var state))
            {
                return; // idempotent
            }

            if (_cancellations.TryRemove(key, out var cts))
            {
                cts.Cancel();
                cts.Dispose();
            }

            // Build a minimal Dependency+Endpoint for metric deletion
            var status = state.ToEndpointStatus();
            var ep = new Endpoint(host, port, new Dictionary<string, string>(status.Labels ?? new Dictionary<string, string>()));
            var dep = Dependency.CreateBuilder(depName, state.ToDependencyType())
                .WithCritical(status.Critical)
                .WithEndpoint(ep)
                .Build();

            _metrics.DeleteMetrics(dep, ep);

            _logger.LogInformation(
                "dephealth: dynamically removed endpoint {Key}", key);
        }
    }

    /// <summary>
    /// Dynamically replaces an endpoint with a new one.
    /// Throws <see cref="EndpointNotFoundException"/> if the old endpoint does not exist.
    /// </summary>
    public void UpdateEndpoint(string depName, string oldHost, string oldPort,
        Endpoint newEp, IHealthChecker checker)
    {
        lock (_mutationLock)
        {
            if (!_started || _stopped)
            {
                throw new InvalidOperationException(
                    "Cannot update endpoint: scheduler must be started and not stopped");
            }

            var oldKey = $"{depName}:{oldHost}:{oldPort}";
            if (!_states.TryRemove(oldKey, out var oldState))
            {
                throw new EndpointNotFoundException(depName, oldHost, oldPort);
            }

            // Cancel old check loop
            if (_cancellations.TryRemove(oldKey, out var oldCts))
            {
                oldCts.Cancel();
                oldCts.Dispose();
            }

            // Delete old metrics
            var oldStatus = oldState.ToEndpointStatus();
            var oldEp = new Endpoint(oldHost, oldPort, new Dictionary<string, string>(oldStatus.Labels ?? new Dictionary<string, string>()));
            var oldDep = Dependency.CreateBuilder(depName, oldState.ToDependencyType())
                .WithCritical(oldStatus.Critical)
                .WithEndpoint(oldEp)
                .Build();
            _metrics.DeleteMetrics(oldDep, oldEp);

            // Create new endpoint
            var newKey = StateKey(depName, newEp);
            var newDep = Dependency.CreateBuilder(depName, checker.Type)
                .WithCritical(oldStatus.Critical)
                .WithEndpoint(newEp)
                .WithConfig(_globalConfig)
                .Build();

            var newState = new EndpointState();
            newState.SetStaticFields(depName, checker.Type.Label(),
                newEp.Host, newEp.Port, oldStatus.Critical, newEp.Labels);
            _states[newKey] = newState;

            var newCts = CancellationTokenSource.CreateLinkedTokenSource(_globalCts!.Token);
            _cancellations[newKey] = newCts;

            _ = RunCheckLoopAsync(newDep, checker, newEp, _globalConfig, newCts.Token);

            _logger.LogInformation(
                "dephealth: dynamically updated endpoint {OldKey} -> {NewKey}", oldKey, newKey);
        }
    }

    /// <inheritdoc />
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
        if (!_states.TryGetValue(key, out var state))
        {
            return; // endpoint was removed
        }
        var sw = Stopwatch.StartNew();

        try
        {
            using var cts = new CancellationTokenSource(config.Timeout);
            checker.CheckAsync(ep, cts.Token).GetAwaiter().GetResult();

            sw.Stop();
            var duration = sw.Elapsed;

            var wasBefore = state.Healthy;
            state.RecordSuccess(config.SuccessThreshold);

            var result = ErrorClassifier.Classify(null);
            state.StoreCheckResult(result.Category, result.Detail, duration);
            _metrics.SetHealth(dep, ep, 1.0);
            _metrics.ObserveLatency(dep, ep, duration);
            _metrics.SetStatus(dep, ep, result.Category);
            _metrics.SetStatusDetail(dep, ep, result.Detail);

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

            var result = ErrorClassifier.Classify(e);
            state.StoreCheckResult(result.Category, result.Detail, duration);
            _metrics.SetHealth(dep, ep, 0.0);
            _metrics.ObserveLatency(dep, ep, duration);
            _metrics.SetStatus(dep, ep, result.Category);
            _metrics.SetStatusDetail(dep, ep, result.Detail);

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
