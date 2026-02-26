namespace DepHealth.Exceptions;

/// <summary>
/// Base exception for the dephealth SDK with optional status classification.
/// </summary>
public class DepHealthException : Exception
{
    /// <summary>
    /// Initializes a new instance of the <see cref="DepHealthException"/> class with the specified message.
    /// </summary>
    /// <param name="message">The error message.</param>
    public DepHealthException(string message) : base(message) { }

    /// <summary>
    /// Initializes a new instance of the <see cref="DepHealthException"/> class with a message and inner exception.
    /// </summary>
    /// <param name="message">The error message.</param>
    /// <param name="innerException">The exception that caused this error.</param>
    public DepHealthException(string message, Exception innerException) : base(message, innerException) { }

    /// <summary>Returns the status category for this error.</summary>
    public virtual string ExceptionStatusCategory => StatusCategory.Error;

    /// <summary>Returns the detail value for this error.</summary>
    public virtual string ExceptionStatusDetail => "error";
}
