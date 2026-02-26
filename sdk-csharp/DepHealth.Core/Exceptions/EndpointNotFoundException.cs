namespace DepHealth.Exceptions;

/// <summary>
/// Thrown when an endpoint lookup fails during dynamic update/remove operations.
/// </summary>
public class EndpointNotFoundException : InvalidOperationException
{
    /// <summary>
    /// Initializes a new instance of the <see cref="EndpointNotFoundException"/> class.
    /// </summary>
    /// <param name="depName">The dependency name that was searched.</param>
    /// <param name="host">The endpoint host that was not found.</param>
    /// <param name="port">The endpoint port that was not found.</param>
    public EndpointNotFoundException(string depName, string host, string port)
        : base($"Endpoint not found: {depName}:{host}:{port}")
    {
    }
}
