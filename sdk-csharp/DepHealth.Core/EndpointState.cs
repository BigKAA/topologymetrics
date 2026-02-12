namespace DepHealth;

/// <summary>
/// Thread-safe endpoint state: healthy/unhealthy, counters for consecutive successes/failures.
/// </summary>
internal sealed class EndpointState
{
    private readonly object _lock = new();
    private bool? _healthy;       // null = UNKNOWN
    private int _consecutiveFailures;
    private int _consecutiveSuccesses;

    public bool? Healthy
    {
        get { lock (_lock) { return _healthy; } }
    }

    public void RecordSuccess(int successThreshold)
    {
        lock (_lock)
        {
            _consecutiveFailures = 0;
            _consecutiveSuccesses++;

            if (_healthy is null)
            {
                // First check — immediate transition
                _healthy = true;
                return;
            }

            if (!_healthy.Value && _consecutiveSuccesses >= successThreshold)
            {
                _healthy = true;
            }
        }
    }

    public void RecordFailure(int failureThreshold)
    {
        lock (_lock)
        {
            _consecutiveSuccesses = 0;
            _consecutiveFailures++;

            if (_healthy is null)
            {
                // First check — immediate transition
                _healthy = false;
                return;
            }

            if (_healthy.Value && _consecutiveFailures >= failureThreshold)
            {
                _healthy = false;
            }
        }
    }
}
