namespace DepHealth;

/// <summary>
/// Interface for dependency health checking.
/// Implementations must be thread-safe.
/// </summary>
public interface IHealthChecker
{
    /// <summary>
    /// Performs a health check on the endpoint.
    /// </summary>
    /// <param name="endpoint">The endpoint to check.</param>
    /// <param name="ct">Cancellation token (used as timeout).</param>
    /// <exception cref="Exception">If the dependency is unhealthy or the check failed.</exception>
    Task CheckAsync(Endpoint endpoint, CancellationToken ct);

    /// <summary>
    /// The dependency type.
    /// </summary>
    DependencyType Type { get; }
}
