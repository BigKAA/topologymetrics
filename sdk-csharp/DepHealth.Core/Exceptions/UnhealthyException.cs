namespace DepHealth.Exceptions;

/// <summary>
/// Dependency is unhealthy (check failed).
/// </summary>
public class UnhealthyException : DepHealthException
{
    private readonly string _detail;

    public UnhealthyException(string message) : base(message)
    {
        _detail = "unhealthy";
    }

    public UnhealthyException(string message, string detail) : base(message)
    {
        _detail = detail;
    }

    public UnhealthyException(string message, Exception innerException) : base(message, innerException)
    {
        _detail = "unhealthy";
    }

    public override string ExceptionStatusCategory => StatusCategory.Unhealthy;
    public override string ExceptionStatusDetail => _detail;
}
