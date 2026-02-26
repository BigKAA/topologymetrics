namespace DepHealth.Exceptions;

/// <summary>
/// Connection refused during dependency health check.
/// </summary>
public class ConnectionRefusedException : DepHealthException
{
    /// <summary>
    /// Initializes a new instance of the <see cref="ConnectionRefusedException"/> class with the specified message.
    /// </summary>
    /// <param name="message">The error message.</param>
    public ConnectionRefusedException(string message) : base(message) { }

    /// <summary>
    /// Initializes a new instance of the <see cref="ConnectionRefusedException"/> class with a message and inner exception.
    /// </summary>
    /// <param name="message">The error message.</param>
    /// <param name="innerException">The exception that caused the connection refusal.</param>
    public ConnectionRefusedException(string message, Exception innerException) : base(message, innerException) { }

    /// <inheritdoc />
    public override string ExceptionStatusCategory => StatusCategory.ConnectionError;

    /// <inheritdoc />
    public override string ExceptionStatusDetail => "connection_refused";
}
