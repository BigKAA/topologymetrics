namespace DepHealth.Exceptions;

/// <summary>
/// DNS resolution failure during dependency health check.
/// </summary>
public class CheckDnsException : DepHealthException
{
    public CheckDnsException(string message) : base(message) { }
    public CheckDnsException(string message, Exception innerException) : base(message, innerException) { }

    public override string ExceptionStatusCategory => StatusCategory.DnsError;
    public override string ExceptionStatusDetail => "dns_error";
}
