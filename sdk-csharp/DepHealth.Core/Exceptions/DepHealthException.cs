namespace DepHealth.Exceptions;

/// <summary>
/// Base exception for the dephealth SDK.
/// </summary>
public class DepHealthException : Exception
{
    public DepHealthException(string message) : base(message) { }
    public DepHealthException(string message, Exception innerException) : base(message, innerException) { }
}
