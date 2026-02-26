using System.Net.Sockets;
using StackExchange.Redis;

namespace DepHealth.Checks;

/// <summary>
/// Redis health checker — sends PING, expects PONG.
/// Supports standalone mode and integration with IConnectionMultiplexer.
/// </summary>
public sealed class RedisChecker : IHealthChecker
{
    private readonly IConnectionMultiplexer? _multiplexer;
    private readonly string? _connectionString;

    /// <summary>Gets the dependency type for this checker.</summary>
    public DependencyType Type => DependencyType.Redis;

    /// <summary>
    /// Standalone mode: creates a new connection.
    /// </summary>
    public RedisChecker(string? connectionString = null)
    {
        _connectionString = connectionString;
    }

    /// <summary>
    /// Pool mode: uses an existing IConnectionMultiplexer.
    /// </summary>
    public RedisChecker(IConnectionMultiplexer multiplexer)
    {
        _multiplexer = multiplexer ?? throw new ArgumentNullException(nameof(multiplexer));
    }

    /// <inheritdoc />
    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        if (_multiplexer is not null)
        {
            await CheckWithMultiplexerAsync().ConfigureAwait(false);
        }
        else
        {
            await CheckWithNewConnectionAsync(endpoint).ConfigureAwait(false);
        }
    }

    private async Task CheckWithMultiplexerAsync()
    {
        try
        {
            var db = _multiplexer!.GetDatabase();
            var result = await db.PingAsync().ConfigureAwait(false);
            if (result == TimeSpan.Zero)
            {
                throw new Exceptions.UnhealthyException("Redis PING returned zero latency (possible error)");
            }
        }
        catch (Exceptions.DepHealthException)
        {
            throw;
        }
        catch (Exception e)
        {
            throw ClassifyRedisError(e);
        }
    }

    private async Task CheckWithNewConnectionAsync(Endpoint endpoint)
    {
        var connStr = _connectionString ??
            $"{endpoint.Host}:{endpoint.Port},connectTimeout=5000,abortConnect=true";

        IConnectionMultiplexer mux;
        try
        {
            mux = await ConnectionMultiplexer.ConnectAsync(connStr).ConfigureAwait(false);
        }
        catch (Exceptions.DepHealthException)
        {
            throw;
        }
        catch (Exception e)
        {
            throw ClassifyRedisError(e);
        }

        try
        {
            var db = mux.GetDatabase();
            await db.PingAsync().ConfigureAwait(false);
        }
        catch (Exceptions.DepHealthException)
        {
            throw;
        }
        catch (Exception e)
        {
            throw ClassifyRedisError(e);
        }
        finally
        {
            await mux.DisposeAsync().ConfigureAwait(false);
        }
    }

    private static Exception ClassifyRedisError(Exception e)
    {
        var msg = e.Message ?? "";

        // Auth errors
        if (msg.Contains("NOAUTH", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("WRONGPASS", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("AUTH", StringComparison.OrdinalIgnoreCase))
        {
            return new Exceptions.CheckAuthException("Redis auth error: " + msg, e);
        }

        // StackExchange.Redis wraps SocketException in RedisConnectionException;
        // check InnerException chain for SocketException with ConnectionRefused
        if (HasConnectionRefusedSocket(e))
        {
            return new Exceptions.ConnectionRefusedException("Redis connection refused: " + msg, e);
        }

        // Message-based fallback for cases where SocketException is not in the chain
        if (msg.Contains("connection refused", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("No connection", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("It was not possible to connect", StringComparison.OrdinalIgnoreCase))
        {
            return new Exceptions.ConnectionRefusedException("Redis connection refused: " + msg, e);
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
}
