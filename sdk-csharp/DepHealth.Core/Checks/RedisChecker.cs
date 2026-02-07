using StackExchange.Redis;

namespace DepHealth.Checks;

/// <summary>
/// Redis health checker — выполняет PING, ожидает PONG.
/// Поддерживает автономный режим и интеграцию с IConnectionMultiplexer.
/// </summary>
public sealed class RedisChecker : IHealthChecker
{
    private readonly IConnectionMultiplexer? _multiplexer;
    private readonly string? _connectionString;

    public DependencyType Type => DependencyType.Redis;

    /// <summary>
    /// Автономный режим: создаёт новое соединение.
    /// </summary>
    public RedisChecker(string? connectionString = null)
    {
        _connectionString = connectionString;
    }

    /// <summary>
    /// Pool-режим: использует существующий IConnectionMultiplexer.
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
        var db = _multiplexer!.GetDatabase();
        var result = await db.PingAsync().ConfigureAwait(false);
        if (result == TimeSpan.Zero)
        {
            throw new Exceptions.UnhealthyException("Redis PING returned zero latency (possible error)");
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
        finally
        {
            await mux.DisposeAsync().ConfigureAwait(false);
        }
    }
}
