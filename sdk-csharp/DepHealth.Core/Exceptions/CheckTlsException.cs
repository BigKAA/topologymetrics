namespace DepHealth.Exceptions;

/// <summary>
/// TLS/SSL handshake or certificate validation failure during dependency health check.
/// </summary>
public class CheckTlsException : DepHealthException
{
    public CheckTlsException(string message) : base(message) { }
    public CheckTlsException(string message, Exception innerException) : base(message, innerException) { }

    public override string ExceptionStatusCategory => StatusCategory.TlsError;
    public override string ExceptionStatusDetail => "tls_error";
}
