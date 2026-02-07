namespace DepHealth.Exceptions;

/// <summary>
/// Базовое исключение SDK dephealth.
/// </summary>
public class DepHealthException : Exception
{
    public DepHealthException(string message) : base(message) { }
    public DepHealthException(string message, Exception innerException) : base(message, innerException) { }
}
