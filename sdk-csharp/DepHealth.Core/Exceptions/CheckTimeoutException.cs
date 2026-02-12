namespace DepHealth.Exceptions;

/// <summary>
/// Timeout during dependency health check.
/// </summary>
public class CheckTimeoutException : DepHealthException
{
    public CheckTimeoutException(string message) : base(message) { }
    public CheckTimeoutException(string message, Exception innerException) : base(message, innerException) { }
}
