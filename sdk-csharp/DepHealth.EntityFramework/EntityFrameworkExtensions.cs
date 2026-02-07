using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Infrastructure;

namespace DepHealth.EntityFramework;

/// <summary>
/// Extension-методы для автоматического добавления PostgreSQL checker из DbContext.
/// </summary>
public static class EntityFrameworkExtensions
{
    /// <summary>
    /// Добавляет PostgreSQL checker из connection string, извлечённого из DbContext.
    /// </summary>
    /// <typeparam name="TContext">Тип DbContext.</typeparam>
    /// <param name="builder">Builder dephealth.</param>
    /// <param name="name">Имя зависимости.</param>
    /// <param name="context">Экземпляр DbContext.</param>
    /// <param name="critical">Критичная ли зависимость.</param>
    public static DepHealthMonitor.Builder AddNpgsqlFromContext<TContext>(
        this DepHealthMonitor.Builder builder,
        string name,
        TContext context,
        bool critical = false)
        where TContext : DbContext
    {
        var connStr = context.Database.GetConnectionString()
            ?? throw new ConfigurationException(
                $"Cannot extract connection string from DbContext {typeof(TContext).Name}");

        // Извлечь host и port из connection string
        var endpoint = ConfigParser.ParseConnectionString(connStr);

        var checker = new Checks.PostgresChecker(connStr);

        builder.AddCustom(name, DependencyType.Postgres,
            endpoint.Host, endpoint.Port, checker, critical);

        return builder;
    }
}
