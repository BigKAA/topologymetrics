namespace DepHealth.Exceptions;

/// <summary>
/// Connection refused during dependency health check.
/// </summary>
public class ConnectionRefusedException : DepHealthException
{
    public ConnectionRefusedException(string message) : base(message) { }
    public ConnectionRefusedException(string message, Exception innerException) : base(message, innerException) { }
}
