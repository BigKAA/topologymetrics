namespace DepHealth;

/// <summary>
/// Ошибка валидации конфигурации.
/// </summary>
public class ValidationException : Exception
{
    public ValidationException(string message) : base(message) { }
}
