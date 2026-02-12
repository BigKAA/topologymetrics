namespace DepHealth.Exceptions;

/// <summary>
/// Dependency is unhealthy (check failed).
/// </summary>
public class UnhealthyException : DepHealthException
{
    public UnhealthyException(string message) : base(message) { }
    public UnhealthyException(string message, Exception innerException) : base(message, innerException) { }
}
