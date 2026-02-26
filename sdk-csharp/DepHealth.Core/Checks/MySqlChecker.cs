using MySqlConnector;

namespace DepHealth.Checks;

/// <summary>
/// MySQL health checker — executes SELECT 1.
/// </summary>
public sealed class MySqlChecker : IHealthChecker
{
    private readonly string? _connectionString;

    /// <summary>Gets the dependency type for this checker.</summary>
    public DependencyType Type => DependencyType.MySql;

    /// <summary>Creates a new instance of <see cref="MySqlChecker"/>.</summary>
    /// <param name="connectionString">Optional MySQL connection string. If <c>null</c>, a default string is built from the endpoint.</param>
    public MySqlChecker(string? connectionString = null)
    {
        _connectionString = connectionString;
    }

    /// <inheritdoc />
    public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        var connStr = _connectionString ??
            $"Server={endpoint.Host};Port={endpoint.Port};ConnectionTimeout=5";

        try
        {
            await using var conn = new MySqlConnection(connStr);
            await conn.OpenAsync(ct).ConfigureAwait(false);
            await using var cmd = conn.CreateCommand();
            cmd.CommandText = "SELECT 1";
            await cmd.ExecuteScalarAsync(ct).ConfigureAwait(false);
        }
        catch (Exceptions.DepHealthException)
        {
            throw;
        }
        catch (MySqlException me) when (me.ErrorCode == MySqlErrorCode.AccessDenied)
        {
            throw new Exceptions.CheckAuthException("MySQL auth error: " + me.Message, me);
        }
    }
}
