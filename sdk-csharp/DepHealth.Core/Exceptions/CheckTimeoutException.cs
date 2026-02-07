namespace DepHealth.Exceptions;

/// <summary>
/// Таймаут при проверке здоровья зависимости.
/// </summary>
public class CheckTimeoutException : DepHealthException
{
    public CheckTimeoutException(string message) : base(message) { }
    public CheckTimeoutException(string message, Exception innerException) : base(message, innerException) { }
}
