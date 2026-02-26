namespace DepHealth;

/// <summary>
/// Configuration validation error.
/// </summary>
public class ValidationException : Exception
{
    /// <summary>
    /// Initializes a new instance of the <see cref="ValidationException"/> class with the specified message.
    /// </summary>
    /// <param name="message">The validation error message.</param>
    public ValidationException(string message) : base(message) { }
}
