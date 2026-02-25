using System.Net.Sockets;
using Novell.Directory.Ldap;

namespace DepHealth.Checks;

/// <summary>
/// LDAP check method.
/// </summary>
public enum LdapCheckMethod
{
    AnonymousBind,
    SimpleBind,
    RootDse,
    Search
}

/// <summary>
/// LDAP search scope.
/// </summary>
public enum LdapSearchScope
{
    Base,
    One,
    Sub
}

/// <summary>
/// LDAP health checker — supports anonymous bind, simple bind, RootDSE query, search.
/// Supports LDAP (plain), LDAPS (TLS), and StartTLS connections.
/// Two modes: standalone (creates a new connection) or pool (uses existing connection).
/// </summary>
public sealed class LdapChecker : IHealthChecker
{
    private readonly LdapCheckMethod _checkMethod;
    private readonly string _bindDN;
    private readonly string _bindPassword;
    private readonly string _baseDN;
    private readonly string _searchFilter;
    private readonly LdapSearchScope _searchScope;
    private readonly bool _useTls;
    private readonly bool _startTls;
    private readonly bool _tlsSkipVerify;
    private readonly ILdapConnection? _connection;

    public DependencyType Type => DependencyType.Ldap;

    /// <summary>
    /// Standalone mode constructor.
    /// </summary>
    public LdapChecker(
        LdapCheckMethod checkMethod = LdapCheckMethod.RootDse,
        string bindDN = "",
        string bindPassword = "",
        string baseDN = "",
        string searchFilter = "(objectClass=*)",
        LdapSearchScope searchScope = LdapSearchScope.Base,
        bool useTls = false,
        bool startTls = false,
        bool tlsSkipVerify = false)
    {
        Validate(checkMethod, bindDN, bindPassword, baseDN, useTls, startTls);
        _checkMethod = checkMethod;
        _bindDN = bindDN;
        _bindPassword = bindPassword;
        _baseDN = baseDN;
        _searchFilter = searchFilter;
        _searchScope = searchScope;
        _useTls = useTls;
        _startTls = startTls;
        _tlsSkipVerify = tlsSkipVerify;
    }

    /// <summary>
    /// Pool mode constructor: uses an existing ILdapConnection.
    /// </summary>
    public LdapChecker(
        ILdapConnection connection,
        LdapCheckMethod checkMethod = LdapCheckMethod.RootDse,
        string bindDN = "",
        string bindPassword = "",
        string baseDN = "",
        string searchFilter = "(objectClass=*)",
        LdapSearchScope searchScope = LdapSearchScope.Base)
    {
        _connection = connection ?? throw new ArgumentNullException(nameof(connection));
        Validate(checkMethod, bindDN, bindPassword, baseDN, false, false);
        _checkMethod = checkMethod;
        _bindDN = bindDN;
        _bindPassword = bindPassword;
        _baseDN = baseDN;
        _searchFilter = searchFilter;
        _searchScope = searchScope;
    }

    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        if (_connection is not null)
        {
            await CheckWithConnectionAsync(_connection, ct).ConfigureAwait(false);
        }
        else
        {
            await CheckStandaloneAsync(endpoint, ct).ConfigureAwait(false);
        }
    }

    private async Task CheckStandaloneAsync(Endpoint endpoint, CancellationToken ct)
    {
        var options = new LdapConnectionOptions();
        if (_useTls)
        {
            options.UseSsl();
        }

        if ((_useTls || _startTls) && _tlsSkipVerify)
        {
            options.ConfigureRemoteCertificateValidationCallback(
                (_, _, _, _) => true);
        }

        var conn = new LdapConnection(options);
        try
        {
            conn.ConnectionTimeout = 5000;
            await conn.ConnectAsync(endpoint.Host, endpoint.PortAsInt(), ct)
                .ConfigureAwait(false);

            if (_startTls)
            {
                await conn.StartTlsAsync(ct).ConfigureAwait(false);
            }

            await CheckWithConnectionAsync(conn, ct).ConfigureAwait(false);
        }
        catch (Exceptions.DepHealthException)
        {
            throw;
        }
        catch (LdapException e)
        {
            throw ClassifyLdapError(e);
        }
        catch (Exception e)
        {
            throw ClassifyGenericError(e);
        }
        finally
        {
            try { conn.Disconnect(); }
            catch
            {
                // Ignore disconnect errors.
            }

            conn.Dispose();
        }
    }

    private async Task CheckWithConnectionAsync(ILdapConnection conn, CancellationToken ct)
    {
        try
        {
            switch (_checkMethod)
            {
                case LdapCheckMethod.AnonymousBind:
                    await conn.BindAsync("", "", ct).ConfigureAwait(false);
                    break;
                case LdapCheckMethod.SimpleBind:
                    await conn.BindAsync(_bindDN, _bindPassword, ct).ConfigureAwait(false);
                    break;
                case LdapCheckMethod.RootDse:
                    await SearchRootDseAsync(conn, ct).ConfigureAwait(false);
                    break;
                case LdapCheckMethod.Search:
                    await SearchWithConfigAsync(conn, ct).ConfigureAwait(false);
                    break;
                default:
                    await SearchRootDseAsync(conn, ct).ConfigureAwait(false);
                    break;
            }
        }
        catch (Exceptions.DepHealthException)
        {
            throw;
        }
        catch (LdapException e)
        {
            throw ClassifyLdapError(e);
        }
    }

    private static async Task SearchRootDseAsync(ILdapConnection conn, CancellationToken ct)
    {
        var constraints = new LdapSearchConstraints { MaxResults = 1 };
        var results = await conn.SearchAsync(
            "", LdapConnection.ScopeBase, "(objectClass=*)",
            new[] { "namingContexts", "subschemaSubentry" }, false, constraints, ct)
            .ConfigureAwait(false);

        // Consume at least one entry to verify the search succeeded.
        await results.HasMoreAsync(ct).ConfigureAwait(false);
    }

    private async Task SearchWithConfigAsync(ILdapConnection conn, CancellationToken ct)
    {
        // Bind before search if credentials are provided.
        if (!string.IsNullOrEmpty(_bindDN))
        {
            await conn.BindAsync(_bindDN, _bindPassword, ct).ConfigureAwait(false);
        }

        var filter = string.IsNullOrEmpty(_searchFilter) ? "(objectClass=*)" : _searchFilter;
        var scope = ToLdapScope(_searchScope);
        var constraints = new LdapSearchConstraints { MaxResults = 1 };

        var results = await conn.SearchAsync(
            _baseDN, scope, filter,
            new[] { "dn" }, false, constraints, ct)
            .ConfigureAwait(false);

        await results.HasMoreAsync(ct).ConfigureAwait(false);
    }

    private static int ToLdapScope(LdapSearchScope scope) => scope switch
    {
        LdapSearchScope.One => LdapConnection.ScopeOne,
        LdapSearchScope.Sub => LdapConnection.ScopeSub,
        _ => LdapConnection.ScopeBase
    };

    internal static Exception ClassifyLdapError(LdapException e)
    {
        var rc = e.ResultCode;

        // Auth errors: Invalid Credentials (49), Insufficient Access Rights (50).
        if (rc == LdapException.InvalidCredentials
            || rc == LdapException.InsufficientAccessRights)
        {
            return new Exceptions.CheckAuthException("LDAP auth error: " + e.LdapErrorMessage, e);
        }

        // Server unavailable: Busy (51), Unavailable (52), Unwilling To Perform (53).
        if (rc == LdapException.Busy
            || rc == LdapException.Unavailable
            || rc == LdapException.UnwillingToPerform)
        {
            return new Exceptions.UnhealthyException(
                "LDAP server unhealthy: " + e.LdapErrorMessage, e);
        }

        // Connection errors: Server Down (81), Connect Error (91).
        if (rc == LdapException.ServerDown || rc == LdapException.ConnectError)
        {
            return new Exceptions.ConnectionRefusedException(
                "LDAP connection error: " + e.LdapErrorMessage, e);
        }

        // TLS errors: TLS not supported (112), SSL handshake failed (113).
        if (rc == LdapException.TlsNotSupported || rc == LdapException.SslHandshakeFailed)
        {
            return new Exceptions.CheckTlsException(
                "LDAP TLS error: " + e.LdapErrorMessage, e);
        }

        // Timeout: Ldap Timeout (85).
        if (rc == LdapException.LdapTimeout)
        {
            return new Exceptions.CheckTimeoutException(
                "LDAP timeout: " + e.LdapErrorMessage, e);
        }

        return ClassifyGenericError(e);
    }

    internal static Exception ClassifyGenericError(Exception e)
    {
        if (HasConnectionRefusedSocket(e))
        {
            return new Exceptions.ConnectionRefusedException(
                "LDAP connection refused: " + e.Message, e);
        }

        var msg = e.Message ?? "";

        if (msg.Contains("tls:", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("x509:", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("certificate", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("SSL", StringComparison.Ordinal)
            || msg.Contains("handshake", StringComparison.OrdinalIgnoreCase))
        {
            return new Exceptions.CheckTlsException("LDAP TLS error: " + msg, e);
        }

        if (msg.Contains("connection refused", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("connect timed out", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("Failed to connect", StringComparison.OrdinalIgnoreCase))
        {
            return new Exceptions.ConnectionRefusedException(
                "LDAP connection error: " + msg, e);
        }

        return e;
    }

    private static bool HasConnectionRefusedSocket(Exception? e)
    {
        while (e is not null)
        {
            if (e is SocketException { SocketErrorCode: SocketError.ConnectionRefused })
            {
                return true;
            }

            e = e.InnerException;
        }

        return false;
    }

    internal static void Validate(
        LdapCheckMethod checkMethod, string bindDN, string bindPassword,
        string baseDN, bool useTls, bool startTls)
    {
        if (checkMethod == LdapCheckMethod.SimpleBind)
        {
            if (string.IsNullOrEmpty(bindDN) || string.IsNullOrEmpty(bindPassword))
            {
                throw new ValidationException(
                    "LDAP simple_bind requires bindDN and bindPassword");
            }
        }

        if (checkMethod == LdapCheckMethod.Search)
        {
            if (string.IsNullOrEmpty(baseDN))
            {
                throw new ValidationException("LDAP search requires baseDN");
            }
        }

        if (startTls && useTls)
        {
            throw new ValidationException("startTLS and ldaps:// are incompatible");
        }
    }
}
