namespace DepHealth;

/// <summary>
/// Configuration validation error.
/// </summary>
public class ValidationException : Exception
{
    /// <summary>
    /// Initializes a new instance of the <see cref="ValidationException"/> class.
    /// </summary>
    public ValidationException() { }

    /// <summary>
    /// Initializes a new instance of the <see cref="ValidationException"/> class with the specified message.
    /// </summary>
    /// <param name="message">The validation error message.</param>
    public ValidationException(string message) : base(message) { }

    /// <summary>
    /// Initializes a new instance of the <see cref="ValidationException"/> class with a message and inner exception.
    /// </summary>
    /// <param name="message">The validation error message.</param>
    /// <param name="innerException">The exception that caused the validation error.</param>
    public ValidationException(string message, Exception innerException) : base(message, innerException) { }
}
