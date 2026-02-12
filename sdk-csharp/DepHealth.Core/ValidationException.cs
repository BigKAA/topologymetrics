namespace DepHealth;

/// <summary>
/// Configuration validation error.
/// </summary>
public class ValidationException : Exception
{
    public ValidationException(string message) : base(message) { }
}
