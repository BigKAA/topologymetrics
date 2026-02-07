namespace DepHealth.Exceptions;

/// <summary>
/// Зависимость нездорова (проверка не прошла).
/// </summary>
public class UnhealthyException : DepHealthException
{
    public UnhealthyException(string message) : base(message) { }
    public UnhealthyException(string message, Exception innerException) : base(message, innerException) { }
}
