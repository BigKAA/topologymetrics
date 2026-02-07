using Npgsql;

namespace DepHealth.Checks;

/// <summary>
/// PostgreSQL health checker — выполняет SELECT 1.
/// Поддерживает автономный режим и интеграцию с NpgsqlDataSource.
/// </summary>
public sealed class PostgresChecker : IHealthChecker
{
    private readonly NpgsqlDataSource? _dataSource;
    private readonly string? _connectionString;

    public DependencyType Type => DependencyType.Postgres;

    /// <summary>
    /// Автономный режим: создаёт новое соединение.
    /// </summary>
    public PostgresChecker(string? connectionString = null)
    {
        _connectionString = connectionString;
    }

    /// <summary>
    /// Pool-режим: использует существующий NpgsqlDataSource.
    /// </summary>
    public PostgresChecker(NpgsqlDataSource dataSource)
    {
        _dataSource = dataSource ?? throw new ArgumentNullException(nameof(dataSource));
    }

    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        if (_dataSource is not null)
        {
            await CheckWithDataSourceAsync(ct).ConfigureAwait(false);
        }
        else
        {
            await CheckWithNewConnectionAsync(endpoint, ct).ConfigureAwait(false);
        }
    }

    private async Task CheckWithDataSourceAsync(CancellationToken ct)
    {
        await using var conn = await _dataSource!.OpenConnectionAsync(ct).ConfigureAwait(false);
        await using var cmd = conn.CreateCommand();
        cmd.CommandText = "SELECT 1";
        await cmd.ExecuteScalarAsync(ct).ConfigureAwait(false);
    }

    private async Task CheckWithNewConnectionAsync(Endpoint endpoint, CancellationToken ct)
    {
        var connStr = _connectionString ??
            $"Host={endpoint.Host};Port={endpoint.Port};Timeout=5";

        await using var conn = new NpgsqlConnection(connStr);
        await conn.OpenAsync(ct).ConfigureAwait(false);
        await using var cmd = conn.CreateCommand();
        cmd.CommandText = "SELECT 1";
        await cmd.ExecuteScalarAsync(ct).ConfigureAwait(false);
    }
}
