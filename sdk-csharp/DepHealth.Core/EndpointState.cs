namespace DepHealth;

/// <summary>
/// Thread-safe endpoint state: healthy/unhealthy, counters for consecutive successes/failures,
/// and detailed status for the HealthDetails() API.
/// </summary>
internal sealed class EndpointState
{
    private readonly object _lock = new();
    private bool? _healthy;       // null = UNKNOWN
    private int _consecutiveFailures;
    private int _consecutiveSuccesses;

    // Dynamic fields (updated under lock by StoreCheckResult)
    private string _lastStatus = StatusCategory.Unknown;
    private string _lastDetail = "unknown";
    private TimeSpan _lastLatency;
    private DateTimeOffset? _lastCheckedAt;

    // Static fields (set once before Start via SetStaticFields)
    private string _depName = "";
    private string _depType = "";
    private string _host = "";
    private string _port = "";
    private bool _critical;
    private Dictionary<string, string> _labels = new();

    public bool? Healthy
    {
        get { lock (_lock) { return _healthy; } }
    }

    public void SetStaticFields(
        string depName, string depType, string host, string port,
        bool critical, IReadOnlyDictionary<string, string> labels)
    {
        _depName = depName;
        _depType = depType;
        _host = host;
        _port = port;
        _critical = critical;
        _labels = new Dictionary<string, string>(labels);
    }

    public void StoreCheckResult(string category, string detail, TimeSpan latency)
    {
        lock (_lock)
        {
            _lastStatus = category;
            _lastDetail = detail;
            _lastLatency = latency;
            _lastCheckedAt = DateTimeOffset.UtcNow;
        }
    }

    public EndpointStatus ToEndpointStatus()
    {
        lock (_lock)
        {
            return new EndpointStatus(
                healthy: _healthy,
                status: _lastStatus,
                detail: _lastDetail,
                latency: _lastLatency,
                type: _depType,
                name: _depName,
                host: _host,
                port: _port,
                critical: _critical,
                lastCheckedAt: _lastCheckedAt,
                labels: new Dictionary<string, string>(_labels));
        }
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
