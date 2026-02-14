using System.Net.Sockets;
using System.Security.Authentication;
using DepHealth.Exceptions;

namespace DepHealth;

/// <summary>
/// Classifies exceptions into status category and detail for metrics.
/// </summary>
public static class ErrorClassifier
{
    /// <summary>
    /// Classifies an exception (or null for success) into a <see cref="CheckResult"/>.
    /// </summary>
    public static CheckResult Classify(Exception? err)
    {
        if (err is null)
        {
            return CheckResult.Ok;
        }

        // 1. DepHealthException carries its own classification
        if (err is DepHealthException dhe)
        {
            return new CheckResult(dhe.ExceptionStatusCategory, dhe.ExceptionStatusDetail);
        }

        // 2. Platform timeout types
        if (err is TimeoutException or OperationCanceledException)
        {
            return new CheckResult(StatusCategory.Timeout, "timeout");
        }

        // 3. Socket errors
        if (err is SocketException se)
        {
            return ClassifySocketException(se);
        }

        // 4. DNS errors (System.Net.Sockets.SocketException with specific error code
        // is handled above; direct check for HttpRequestException wrapping DNS)
        if (IsDnsError(err))
        {
            return new CheckResult(StatusCategory.DnsError, "dns_error");
        }

        // 5. TLS/SSL errors
        if (err is AuthenticationException)
        {
            return new CheckResult(StatusCategory.TlsError, "tls_error");
        }

        // 6. Check inner exception
        if (err.InnerException is not null)
        {
            var inner = Classify(err.InnerException);
            if (inner.Category != StatusCategory.Error)
            {
                return inner;
            }
        }

        // 7. Fallback
        return new CheckResult(StatusCategory.Error, "error");
    }

    private static CheckResult ClassifySocketException(SocketException se)
    {
        return se.SocketErrorCode switch
        {
            SocketError.ConnectionRefused => new CheckResult(StatusCategory.ConnectionError, "connection_refused"),
            SocketError.HostNotFound or SocketError.NoData =>
                new CheckResult(StatusCategory.DnsError, "dns_error"),
            SocketError.TimedOut => new CheckResult(StatusCategory.Timeout, "timeout"),
            SocketError.HostUnreachable or SocketError.NetworkUnreachable =>
                new CheckResult(StatusCategory.ConnectionError, "connection_refused"),
            _ => new CheckResult(StatusCategory.Error, "error")
        };
    }

    private static bool IsDnsError(Exception err)
    {
        var msg = err.Message;
        return msg.Contains("No such host is known", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("Name or service not known", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("nodename nor servname provided", StringComparison.OrdinalIgnoreCase);
    }
}
