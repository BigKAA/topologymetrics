namespace DepHealth.Exceptions;

/// <summary>
/// Authentication or authorization failure during dependency health check.
/// </summary>
public class CheckAuthException : DepHealthException
{
    public CheckAuthException(string message) : base(message) { }
    public CheckAuthException(string message, Exception innerException) : base(message, innerException) { }

    public override string ExceptionStatusCategory => StatusCategory.AuthError;
    public override string ExceptionStatusDetail => "auth_error";
}
