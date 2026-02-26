namespace DepHealth.Exceptions;

/// <summary>
/// Authentication or authorization failure during dependency health check.
/// </summary>
public class CheckAuthException : DepHealthException
{
    /// <summary>
    /// Initializes a new instance of the <see cref="CheckAuthException"/> class with the specified message.
    /// </summary>
    /// <param name="message">The error message.</param>
    public CheckAuthException(string message) : base(message) { }

    /// <summary>
    /// Initializes a new instance of the <see cref="CheckAuthException"/> class with a message and inner exception.
    /// </summary>
    /// <param name="message">The error message.</param>
    /// <param name="innerException">The exception that caused the authentication failure.</param>
    public CheckAuthException(string message, Exception innerException) : base(message, innerException) { }

    /// <inheritdoc />
    public override string ExceptionStatusCategory => StatusCategory.AuthError;

    /// <inheritdoc />
    public override string ExceptionStatusDetail => "auth_error";
}
