using Npgsql;

namespace DepHealth.Checks;

/// <summary>
/// PostgreSQL health checker — executes SELECT 1.
/// Supports standalone mode and integration with NpgsqlDataSource.
/// </summary>
public sealed class PostgresChecker : IHealthChecker
{
    private readonly NpgsqlDataSource? _dataSource;
    private readonly string? _connectionString;

    /// <summary>Gets the dependency type for this checker.</summary>
    public DependencyType Type => DependencyType.Postgres;

    /// <summary>
    /// Standalone mode: creates a new connection.
    /// </summary>
    public PostgresChecker(string? connectionString = null)
    {
        _connectionString = connectionString;
    }

    /// <summary>
    /// Pool mode: uses an existing NpgsqlDataSource.
    /// </summary>
    public PostgresChecker(NpgsqlDataSource dataSource)
    {
        _dataSource = dataSource ?? throw new ArgumentNullException(nameof(dataSource));
    }

    /// <inheritdoc />
    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        try
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
        catch (PostgresException pe) when (pe.SqlState is "28000" or "28P01")
        {
            throw new Exceptions.CheckAuthException("PostgreSQL auth error: " + pe.Message, pe);
        }
    }

    private async Task CheckWithDataSourceAsync(CancellationToken ct)
    {
        var conn = await _dataSource!.OpenConnectionAsync(ct).ConfigureAwait(false);
        await using (conn.ConfigureAwait(false))
        {
            var cmd = conn.CreateCommand();
            await using (cmd.ConfigureAwait(false))
            {
                cmd.CommandText = "SELECT 1";
                await cmd.ExecuteScalarAsync(ct).ConfigureAwait(false);
            }
        }
    }

    private async Task CheckWithNewConnectionAsync(Endpoint endpoint, CancellationToken ct)
    {
        var connStr = _connectionString ??
            $"Host={endpoint.Host};Port={endpoint.Port};Timeout=5";

        var conn = new NpgsqlConnection(connStr);
        await using (conn.ConfigureAwait(false))
        {
            await conn.OpenAsync(ct).ConfigureAwait(false);
            var cmd = conn.CreateCommand();
            await using (cmd.ConfigureAwait(false))
            {
                cmd.CommandText = "SELECT 1";
                await cmd.ExecuteScalarAsync(ct).ConfigureAwait(false);
            }
        }
    }
}
