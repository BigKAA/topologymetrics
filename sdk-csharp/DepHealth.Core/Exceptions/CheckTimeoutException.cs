namespace DepHealth.Exceptions;

/// <summary>
/// Timeout during dependency health check.
/// </summary>
public class CheckTimeoutException : DepHealthException
{
    /// <summary>
    /// Initializes a new instance of the <see cref="CheckTimeoutException"/> class.
    /// </summary>
    public CheckTimeoutException() { }

    /// <summary>
    /// Initializes a new instance of the <see cref="CheckTimeoutException"/> class with the specified message.
    /// </summary>
    /// <param name="message">The error message.</param>
    public CheckTimeoutException(string message) : base(message) { }

    /// <summary>
    /// Initializes a new instance of the <see cref="CheckTimeoutException"/> class with a message and inner exception.
    /// </summary>
    /// <param name="message">The error message.</param>
    /// <param name="innerException">The exception that caused the timeout.</param>
    public CheckTimeoutException(string message, Exception innerException) : base(message, innerException) { }

    /// <inheritdoc />
    public override string ExceptionStatusCategory => StatusCategory.Timeout;

    /// <inheritdoc />
    public override string ExceptionStatusDetail => "timeout";
}
