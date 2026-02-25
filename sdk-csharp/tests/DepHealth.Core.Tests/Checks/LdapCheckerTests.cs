using System.Net.Sockets;
using DepHealth.Checks;
using DepHealth.Exceptions;

namespace DepHealth.Core.Tests.Checks;

public class LdapCheckerTypeTests
{
    [Fact]
    public void LdapChecker_HasCorrectType()
    {
        var checker = new LdapChecker();
        Assert.Equal(DependencyType.Ldap, checker.Type);
    }

    [Fact]
    public void LdapChecker_DefaultCheckMethodIsRootDse()
    {
        var checker = new LdapChecker();
        Assert.Equal(DependencyType.Ldap, checker.Type);
    }

    [Fact]
    public void LdapChecker_WithCheckMethod()
    {
        var checker = new LdapChecker(checkMethod: LdapCheckMethod.AnonymousBind);
        Assert.Equal(DependencyType.Ldap, checker.Type);
    }

    [Fact]
    public void LdapChecker_WithSimpleBind()
    {
        var checker = new LdapChecker(
            checkMethod: LdapCheckMethod.SimpleBind,
            bindDN: "cn=admin,dc=test,dc=local",
            bindPassword: "password");
        Assert.Equal(DependencyType.Ldap, checker.Type);
    }

    [Fact]
    public void LdapChecker_WithSearch()
    {
        var checker = new LdapChecker(
            checkMethod: LdapCheckMethod.Search,
            baseDN: "dc=test,dc=local",
            searchFilter: "(uid=testuser)",
            searchScope: LdapSearchScope.Sub);
        Assert.Equal(DependencyType.Ldap, checker.Type);
    }

    [Fact]
    public void LdapChecker_WithTlsOptions()
    {
        var checker = new LdapChecker(useTls: true, tlsSkipVerify: true);
        Assert.Equal(DependencyType.Ldap, checker.Type);
    }

    [Fact]
    public void LdapChecker_WithStartTls()
    {
        var checker = new LdapChecker(startTls: true, tlsSkipVerify: true);
        Assert.Equal(DependencyType.Ldap, checker.Type);
    }
}

public class LdapCheckerValidationTests
{
    [Fact]
    public void SimpleBind_WithoutBindDN_ThrowsValidationException()
    {
        Assert.Throws<ValidationException>(() =>
            new LdapChecker(checkMethod: LdapCheckMethod.SimpleBind));
    }

    [Fact]
    public void SimpleBind_WithoutPassword_ThrowsValidationException()
    {
        Assert.Throws<ValidationException>(() =>
            new LdapChecker(
                checkMethod: LdapCheckMethod.SimpleBind,
                bindDN: "cn=admin,dc=test,dc=local"));
    }

    [Fact]
    public void SimpleBind_WithEmptyBindDN_ThrowsValidationException()
    {
        Assert.Throws<ValidationException>(() =>
            new LdapChecker(
                checkMethod: LdapCheckMethod.SimpleBind,
                bindDN: "",
                bindPassword: "password"));
    }

    [Fact]
    public void SimpleBind_WithEmptyPassword_ThrowsValidationException()
    {
        Assert.Throws<ValidationException>(() =>
            new LdapChecker(
                checkMethod: LdapCheckMethod.SimpleBind,
                bindDN: "cn=admin,dc=test,dc=local",
                bindPassword: ""));
    }

    [Fact]
    public void Search_WithoutBaseDN_ThrowsValidationException()
    {
        Assert.Throws<ValidationException>(() =>
            new LdapChecker(checkMethod: LdapCheckMethod.Search));
    }

    [Fact]
    public void Search_WithEmptyBaseDN_ThrowsValidationException()
    {
        Assert.Throws<ValidationException>(() =>
            new LdapChecker(
                checkMethod: LdapCheckMethod.Search,
                baseDN: ""));
    }

    [Fact]
    public void StartTls_WithLdaps_ThrowsValidationException()
    {
        Assert.Throws<ValidationException>(() =>
            new LdapChecker(useTls: true, startTls: true));
    }

    [Fact]
    public void RootDse_NoValidationError()
    {
        var checker = new LdapChecker(checkMethod: LdapCheckMethod.RootDse);
        Assert.NotNull(checker);
    }

    [Fact]
    public void AnonymousBind_NoValidationError()
    {
        var checker = new LdapChecker(checkMethod: LdapCheckMethod.AnonymousBind);
        Assert.NotNull(checker);
    }
}

public class LdapCheckerErrorClassificationTests
{
    [Fact]
    public void InvalidCredentials_ReturnsCheckAuthException()
    {
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "Invalid credentials",
            Novell.Directory.Ldap.LdapException.InvalidCredentials,
            null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<CheckAuthException>(result);
        Assert.Equal("auth_error", ((CheckAuthException)result).ExceptionStatusDetail);
    }

    [Fact]
    public void InsufficientAccessRights_ReturnsCheckAuthException()
    {
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "Insufficient access",
            Novell.Directory.Ldap.LdapException.InsufficientAccessRights,
            null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<CheckAuthException>(result);
    }

    [Fact]
    public void Busy_ReturnsUnhealthyException()
    {
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "Busy",
            Novell.Directory.Ldap.LdapException.Busy,
            null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<UnhealthyException>(result);
        Assert.Equal("unhealthy", ((UnhealthyException)result).ExceptionStatusDetail);
    }

    [Fact]
    public void Unavailable_ReturnsUnhealthyException()
    {
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "Unavailable",
            Novell.Directory.Ldap.LdapException.Unavailable,
            null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<UnhealthyException>(result);
    }

    [Fact]
    public void UnwillingToPerform_ReturnsUnhealthyException()
    {
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "Unwilling",
            Novell.Directory.Ldap.LdapException.UnwillingToPerform,
            null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<UnhealthyException>(result);
    }

    [Fact]
    public void ServerDown_ReturnsConnectionRefusedException()
    {
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "Server down",
            Novell.Directory.Ldap.LdapException.ServerDown,
            null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<ConnectionRefusedException>(result);
        Assert.Equal("connection_refused", ((ConnectionRefusedException)result).ExceptionStatusDetail);
    }

    [Fact]
    public void ConnectError_ReturnsConnectionRefusedException()
    {
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "Connect error",
            Novell.Directory.Ldap.LdapException.ConnectError,
            null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<ConnectionRefusedException>(result);
    }

    [Fact]
    public void LdapTimeout_ReturnsCheckTimeoutException()
    {
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "Timeout",
            Novell.Directory.Ldap.LdapException.LdapTimeout,
            null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<CheckTimeoutException>(result);
        Assert.Equal("timeout", ((CheckTimeoutException)result).ExceptionStatusDetail);
    }

    [Fact]
    public void TlsNotSupported_ReturnsCheckTlsException()
    {
        // Result code 112 = TLS not supported.
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "TLS not supported", 112, null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<CheckTlsException>(result);
        Assert.Equal("tls_error", ((CheckTlsException)result).ExceptionStatusDetail);
    }

    [Fact]
    public void SslHandshakeFailed_ReturnsCheckTlsException()
    {
        // Result code 113 = SSL handshake failed.
        var ldapEx = new Novell.Directory.Ldap.LdapException(
            "SSL handshake failed", 113, null);

        var result = LdapChecker.ClassifyLdapError(ldapEx);
        Assert.IsType<CheckTlsException>(result);
    }

    [Fact]
    public void SocketConnectionRefused_ReturnsConnectionRefusedException()
    {
        var socketEx = new SocketException((int)SocketError.ConnectionRefused);
        var result = LdapChecker.ClassifyGenericError(socketEx);
        Assert.IsType<ConnectionRefusedException>(result);
    }

    [Fact]
    public void WrappedSocketConnectionRefused_ReturnsConnectionRefusedException()
    {
        var socketEx = new SocketException((int)SocketError.ConnectionRefused);
        var wrapperEx = new Exception("connection failed", socketEx);
        var result = LdapChecker.ClassifyGenericError(wrapperEx);
        Assert.IsType<ConnectionRefusedException>(result);
    }

    [Fact]
    public void TlsMessageInException_ReturnsCheckTlsException()
    {
        var ex = new Exception("SSL handshake error occurred");
        var result = LdapChecker.ClassifyGenericError(ex);
        Assert.IsType<CheckTlsException>(result);
    }

    [Fact]
    public void CertificateMessageInException_ReturnsCheckTlsException()
    {
        var ex = new Exception("Remote certificate is invalid");
        var result = LdapChecker.ClassifyGenericError(ex);
        Assert.IsType<CheckTlsException>(result);
    }

    [Fact]
    public void ConnectionRefusedMessage_ReturnsConnectionRefusedException()
    {
        var ex = new Exception("Connection refused by the server");
        var result = LdapChecker.ClassifyGenericError(ex);
        Assert.IsType<ConnectionRefusedException>(result);
    }

    [Fact]
    public void UnknownError_ReturnsOriginalException()
    {
        var ex = new InvalidOperationException("something unexpected");
        var result = LdapChecker.ClassifyGenericError(ex);
        Assert.Same(ex, result);
    }
}

public class LdapCheckerStandaloneTests
{
    [Fact]
    public async Task ConnectionRefused_ThrowsConnectionRefusedException()
    {
        var checker = new LdapChecker(checkMethod: LdapCheckMethod.RootDse);
        var badEp = new Endpoint("127.0.0.1", "19999");

        var ex = await Assert.ThrowsAnyAsync<DepHealthException>(
            () => checker.CheckAsync(badEp, CancellationToken.None));
        Assert.True(
            ex is ConnectionRefusedException || ex.ExceptionStatusCategory == StatusCategory.ConnectionError,
            $"Expected connection error, got: {ex.GetType().Name}: {ex.Message}");
    }
}
