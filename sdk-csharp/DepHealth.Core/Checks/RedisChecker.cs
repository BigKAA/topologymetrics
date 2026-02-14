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

        var mux = await ConnectionMultiplexer.ConnectAsync(connStr).ConfigureAwait(false);
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
        if (msg.Contains("NOAUTH", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("WRONGPASS", StringComparison.OrdinalIgnoreCase)
            || msg.Contains("AUTH", StringComparison.OrdinalIgnoreCase))
        {
            return new Exceptions.CheckAuthException("Redis auth error: " + msg, e);
        }

        return e;
    }
}
