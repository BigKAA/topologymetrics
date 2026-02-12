using MySqlConnector;

namespace DepHealth.Checks;

/// <summary>
/// MySQL health checker — executes SELECT 1.
/// </summary>
public sealed class MySqlChecker : IHealthChecker
{
    private readonly string? _connectionString;

    public DependencyType Type => DependencyType.MySql;

    public MySqlChecker(string? connectionString = null)
    {
        _connectionString = connectionString;
    }

    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        var connStr = _connectionString ??
            $"Server={endpoint.Host};Port={endpoint.Port};ConnectionTimeout=5";

        await using var conn = new MySqlConnection(connStr);
        await conn.OpenAsync(ct).ConfigureAwait(false);
        await using var cmd = conn.CreateCommand();
        cmd.CommandText = "SELECT 1";
        await cmd.ExecuteScalarAsync(ct).ConfigureAwait(false);
    }
}
