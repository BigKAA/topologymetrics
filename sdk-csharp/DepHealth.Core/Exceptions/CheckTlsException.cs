namespace DepHealth.Exceptions;

/// <summary>
/// TLS/SSL handshake or certificate validation failure during dependency health check.
/// </summary>
public class CheckTlsException : DepHealthException
{
    /// <summary>
    /// Initializes a new instance of the <see cref="CheckTlsException"/> class.
    /// </summary>
    public CheckTlsException() { }

    /// <summary>
    /// Initializes a new instance of the <see cref="CheckTlsException"/> class with the specified message.
    /// </summary>
    /// <param name="message">The error message.</param>
    public CheckTlsException(string message) : base(message) { }

    /// <summary>
    /// Initializes a new instance of the <see cref="CheckTlsException"/> class with a message and inner exception.
    /// </summary>
    /// <param name="message">The error message.</param>
    /// <param name="innerException">The exception that caused the TLS failure.</param>
    public CheckTlsException(string message, Exception innerException) : base(message, innerException) { }

    /// <inheritdoc />
    public override string ExceptionStatusCategory => StatusCategory.TlsError;

    /// <inheritdoc />
    public override string ExceptionStatusDetail => "tls_error";
}
