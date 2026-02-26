namespace DepHealth.Exceptions;

/// <summary>
/// DNS resolution failure during dependency health check.
/// </summary>
public class CheckDnsException : DepHealthException
{
    /// <summary>
    /// Initializes a new instance of the <see cref="CheckDnsException"/> class with the specified message.
    /// </summary>
    /// <param name="message">The error message.</param>
    public CheckDnsException(string message) : base(message) { }

    /// <summary>
    /// Initializes a new instance of the <see cref="CheckDnsException"/> class with a message and inner exception.
    /// </summary>
    /// <param name="message">The error message.</param>
    /// <param name="innerException">The exception that caused the DNS resolution failure.</param>
    public CheckDnsException(string message, Exception innerException) : base(message, innerException) { }

    /// <inheritdoc />
    public override string ExceptionStatusCategory => StatusCategory.DnsError;

    /// <inheritdoc />
    public override string ExceptionStatusDetail => "dns_error";
}
