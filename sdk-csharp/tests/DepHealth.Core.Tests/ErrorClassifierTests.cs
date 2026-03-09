using System.Net.Sockets;
using System.Security.Authentication;
using DepHealth.Exceptions;

namespace DepHealth.Core.Tests;

public class ErrorClassifierTests
{
    [Fact]
    public void Classify_Null_ReturnsOk()
    {
        var result = ErrorClassifier.Classify(null);
        Assert.Equal(StatusCategory.Ok, result.Category);
        Assert.Equal("ok", result.Detail);
    }

    [Fact]
    public void Classify_DepHealthException_ReturnsItsClassification()
    {
        var ex = new CheckAuthException("auth failed");
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.AuthError, result.Category);
        Assert.Equal("auth_error", result.Detail);
    }

    [Fact]
    public void Classify_TimeoutException_ReturnsTimeout()
    {
        var result = ErrorClassifier.Classify(new TimeoutException("timed out"));
        Assert.Equal(StatusCategory.Timeout, result.Category);
        Assert.Equal("timeout", result.Detail);
    }

    [Fact]
    public void Classify_OperationCanceledException_ReturnsTimeout()
    {
        var result = ErrorClassifier.Classify(new OperationCanceledException());
        Assert.Equal(StatusCategory.Timeout, result.Category);
    }

    [Fact]
    public void Classify_SocketException_ConnectionRefused()
    {
        var ex = new SocketException((int)SocketError.ConnectionRefused);
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.ConnectionError, result.Category);
        Assert.Equal("connection_refused", result.Detail);
    }

    [Fact]
    public void Classify_SocketException_HostNotFound()
    {
        var ex = new SocketException((int)SocketError.HostNotFound);
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.DnsError, result.Category);
    }

    [Fact]
    public void Classify_SocketException_TimedOut()
    {
        var ex = new SocketException((int)SocketError.TimedOut);
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.Timeout, result.Category);
    }

    [Fact]
    public void Classify_SocketException_HostUnreachable()
    {
        var ex = new SocketException((int)SocketError.HostUnreachable);
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.ConnectionError, result.Category);
    }

    [Fact]
    public void Classify_AuthenticationException_ReturnsTlsError()
    {
        var result = ErrorClassifier.Classify(new AuthenticationException("tls failed"));
        Assert.Equal(StatusCategory.TlsError, result.Category);
    }

    [Fact]
    public void Classify_InnerException_Propagates()
    {
        var inner = new SocketException((int)SocketError.ConnectionRefused);
        var outer = new InvalidOperationException("wrapper", inner);
        var result = ErrorClassifier.Classify(outer);
        Assert.Equal(StatusCategory.ConnectionError, result.Category);
        Assert.Equal("connection_refused", result.Detail);
    }

    [Fact]
    public void Classify_UnknownException_ReturnsError()
    {
        var result = ErrorClassifier.Classify(new InvalidOperationException("unknown"));
        Assert.Equal(StatusCategory.Error, result.Category);
        Assert.Equal("error", result.Detail);
    }

    [Fact]
    public void Classify_DnsErrorByMessage_ReturnsDnsError()
    {
        var ex = new InvalidOperationException("No such host is known");
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.DnsError, result.Category);
    }

    [Fact]
    public void HasConnectionRefusedSocket_WithConnectionRefused_ReturnsTrue()
    {
        var ex = new SocketException((int)SocketError.ConnectionRefused);
        Assert.True(ErrorClassifier.HasConnectionRefusedSocket(ex));
    }

    [Fact]
    public void HasConnectionRefusedSocket_WithNested_ReturnsTrue()
    {
        var inner = new SocketException((int)SocketError.ConnectionRefused);
        var outer = new InvalidOperationException("wrapper", inner);
        Assert.True(ErrorClassifier.HasConnectionRefusedSocket(outer));
    }

    [Fact]
    public void HasConnectionRefusedSocket_WithOtherSocket_ReturnsFalse()
    {
        var ex = new SocketException((int)SocketError.HostNotFound);
        Assert.False(ErrorClassifier.HasConnectionRefusedSocket(ex));
    }

    [Fact]
    public void HasConnectionRefusedSocket_Null_ReturnsFalse()
    {
        Assert.False(ErrorClassifier.HasConnectionRefusedSocket(null));
    }

    [Fact]
    public void Classify_CheckTimeoutException_ReturnsTimeout()
    {
        var ex = new CheckTimeoutException("check timed out");
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.Timeout, result.Category);
    }

    [Fact]
    public void Classify_CheckDnsException_ReturnsDnsError()
    {
        var ex = new CheckDnsException("dns failed");
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.DnsError, result.Category);
    }

    [Fact]
    public void Classify_CheckTlsException_ReturnsTlsError()
    {
        var ex = new CheckTlsException("tls failed");
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.TlsError, result.Category);
    }

    [Fact]
    public void Classify_ConnectionRefusedException_ReturnsConnectionError()
    {
        var ex = new ConnectionRefusedException("refused");
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.ConnectionError, result.Category);
    }

    [Fact]
    public void Classify_UnhealthyException_ReturnsUnhealthy()
    {
        var ex = new UnhealthyException("not healthy");
        var result = ErrorClassifier.Classify(ex);
        Assert.Equal(StatusCategory.Unhealthy, result.Category);
    }
}
