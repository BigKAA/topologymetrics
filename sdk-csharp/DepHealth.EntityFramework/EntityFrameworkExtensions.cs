using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Infrastructure;

namespace DepHealth.EntityFramework;

/// <summary>
/// Extension methods for automatically adding a PostgreSQL checker from DbContext.
/// </summary>
public static class EntityFrameworkExtensions
{
    /// <summary>
    /// Adds a PostgreSQL checker using the connection string extracted from DbContext.
    /// </summary>
    /// <typeparam name="TContext">The DbContext type.</typeparam>
    /// <param name="builder">The dephealth builder.</param>
    /// <param name="name">Dependency name.</param>
    /// <param name="context">The DbContext instance.</param>
    /// <param name="critical">Whether the dependency is critical.</param>
    /// <param name="labels">Custom labels.</param>
    public static DepHealthMonitor.Builder AddNpgsqlFromContext<TContext>(
        this DepHealthMonitor.Builder builder,
        string name,
        TContext context,
        bool? critical = null,
        Dictionary<string, string>? labels = null)
        where TContext : DbContext
    {
        var connStr = context.Database.GetConnectionString()
            ?? throw new ConfigurationException(
                $"Cannot extract connection string from DbContext {typeof(TContext).Name}");

        // Extract host and port from connection string
        var endpoint = ConfigParser.ParseConnectionString(connStr);

        var checker = new Checks.PostgresChecker(connStr);

        builder.AddCustom(name, DependencyType.Postgres,
            endpoint.Host, endpoint.Port, checker, critical, labels);

        return builder;
    }
}
