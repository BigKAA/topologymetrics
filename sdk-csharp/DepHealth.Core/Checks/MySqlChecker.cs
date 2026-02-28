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
            var conn = new MySqlConnection(connStr);
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
        catch (MySqlException me) when (me.ErrorCode == MySqlErrorCode.AccessDenied)
        {
            throw new Exceptions.CheckAuthException("MySQL auth error: " + me.Message, me);
        }
    }
}
