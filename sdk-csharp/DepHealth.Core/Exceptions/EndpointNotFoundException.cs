namespace DepHealth.Exceptions;

/// <summary>
/// Thrown when an endpoint lookup fails during dynamic update/remove operations.
/// </summary>
public class EndpointNotFoundException : InvalidOperationException
{
    public EndpointNotFoundException(string depName, string host, string port)
        : base($"Endpoint not found: {depName}:{host}:{port}")
    {
    }
}
