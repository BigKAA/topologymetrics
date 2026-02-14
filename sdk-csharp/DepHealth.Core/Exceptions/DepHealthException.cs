namespace DepHealth.Exceptions;

/// <summary>
/// Base exception for the dephealth SDK with optional status classification.
/// </summary>
public class DepHealthException : Exception
{
    public DepHealthException(string message) : base(message) { }
    public DepHealthException(string message, Exception innerException) : base(message, innerException) { }

    /// <summary>Returns the status category for this error.</summary>
    public virtual string ExceptionStatusCategory => StatusCategory.Error;

    /// <summary>Returns the detail value for this error.</summary>
    public virtual string ExceptionStatusDetail => "error";
}
