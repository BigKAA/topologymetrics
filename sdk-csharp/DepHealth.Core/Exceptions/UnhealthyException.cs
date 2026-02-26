namespace DepHealth.Exceptions;

/// <summary>
/// Dependency is unhealthy (check failed).
/// </summary>
public class UnhealthyException : DepHealthException
{
    private readonly string _detail;

    /// <summary>
    /// Initializes a new instance of the <see cref="UnhealthyException"/> class with the specified message.
    /// </summary>
    /// <param name="message">The error message.</param>
    public UnhealthyException(string message) : base(message)
    {
        _detail = "unhealthy";
    }

    /// <summary>
    /// Initializes a new instance of the <see cref="UnhealthyException"/> class with a message and custom detail.
    /// </summary>
    /// <param name="message">The error message.</param>
    /// <param name="detail">A specific detail string describing the unhealthy reason.</param>
    public UnhealthyException(string message, string detail) : base(message)
    {
        _detail = detail;
    }

    /// <summary>
    /// Initializes a new instance of the <see cref="UnhealthyException"/> class with a message and inner exception.
    /// </summary>
    /// <param name="message">The error message.</param>
    /// <param name="innerException">The exception that caused the unhealthy state.</param>
    public UnhealthyException(string message, Exception innerException) : base(message, innerException)
    {
        _detail = "unhealthy";
    }

    /// <inheritdoc />
    public override string ExceptionStatusCategory => StatusCategory.Unhealthy;

    /// <inheritdoc />
    public override string ExceptionStatusDetail => _detail;
}
