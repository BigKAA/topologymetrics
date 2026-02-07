namespace DepHealth.Exceptions;

/// <summary>
/// Соединение отклонено при проверке здоровья зависимости.
/// </summary>
public class ConnectionRefusedException : DepHealthException
{
    public ConnectionRefusedException(string message) : base(message) { }
    public ConnectionRefusedException(string message, Exception innerException) : base(message, innerException) { }
}
