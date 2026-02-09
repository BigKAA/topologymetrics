using System.Reflection;
using DepHealth.Checks;

namespace DepHealth.Core.Tests;

/// <summary>
/// Тесты извлечения credentials из URL при построении DepHealthMonitor.
/// </summary>
public class CredentialsTests
{
    [Fact]
    public void AddPostgres_UrlWithCredentials()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app")
            .AddPostgres("db", "postgres://user:pass@host:5432/mydb", critical: true)
            .Build();

        var checker = GetChecker<PostgresChecker>(dh);
        var connStr = GetField<string>(checker, "_connectionString")!;

        Assert.Contains("Username=user", connStr);
        Assert.Contains("Password=pass", connStr);
        Assert.Contains("Database=mydb", connStr);
    }

    [Fact]
    public void AddMySql_UrlWithCredentials()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app")
            .AddMySql("mysql-db", "mysql://user:pass@host:3306/mydb", critical: true)
            .Build();

        var checker = GetChecker<MySqlChecker>(dh);
        var connStr = GetField<string>(checker, "_connectionString")!;

        Assert.Contains("User=user", connStr);
        Assert.Contains("Password=pass", connStr);
        Assert.Contains("Database=mydb", connStr);
    }

    [Fact]
    public void AddRedis_UrlWithPassword()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app")
            .AddRedis("cache", "redis://:secret@host:6379", critical: false)
            .Build();

        var checker = GetChecker<RedisChecker>(dh);
        var connStr = GetField<string>(checker, "_connectionString")!;

        Assert.Contains("password=secret", connStr);
    }

    [Fact]
    public void AddAmqp_UrlWithCredentials()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app")
            .AddAmqp("mq", "amqp://user:pass@host:5672/vhost", critical: false)
            .Build();

        var checker = GetChecker<AmqpChecker>(dh);

        Assert.Equal("user", GetField<string>(checker, "_username"));
        Assert.Equal("pass", GetField<string>(checker, "_password"));
        Assert.Equal("vhost", GetField<string>(checker, "_vhost"));
    }

    [Fact]
    public void AddPostgres_UrlEncoded()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app")
            .AddPostgres("db", "postgres://user%40dom:p%40ss@host:5432/db",
                critical: true)
            .Build();

        var checker = GetChecker<PostgresChecker>(dh);
        var connStr = GetField<string>(checker, "_connectionString")!;

        Assert.Contains("Username=user@dom", connStr);
        Assert.Contains("Password=p@ss", connStr);
    }

    [Fact]
    public void AddAmqp_DefaultVhost()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app")
            .AddAmqp("mq", "amqp://user:pass@host:5672/", critical: false)
            .Build();

        var checker = GetChecker<AmqpChecker>(dh);

        Assert.Equal("/", GetField<string>(checker, "_vhost"));
    }

    [Fact]
    public void AddRedis_NoPassword()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app")
            .AddRedis("cache", "redis://host:6379", critical: false)
            .Build();

        var checker = GetChecker<RedisChecker>(dh);
        var connStr = GetField<string>(checker, "_connectionString")!;

        Assert.DoesNotContain("password", connStr);
    }

    // --- Вспомогательные методы ---

    private static T GetChecker<T>(DepHealthMonitor dh) where T : class
    {
        // DepHealthMonitor -> _scheduler
        var schedulerField = typeof(DepHealthMonitor)
            .GetField("_scheduler", BindingFlags.NonPublic | BindingFlags.Instance)!;
        var scheduler = schedulerField.GetValue(dh)!;

        // CheckScheduler -> _deps (List<ScheduledDep>)
        var depsField = scheduler.GetType()
            .GetField("_deps", BindingFlags.NonPublic | BindingFlags.Instance)!;
        var deps = (System.Collections.IList)depsField.GetValue(scheduler)!;

        foreach (var dep in deps)
        {
            // ScheduledDep.Checker (record property)
            var checkerProp = dep!.GetType().GetProperty("Checker")!;
            var checker = checkerProp.GetValue(dep);
            if (checker is T typed)
            {
                return typed;
            }
        }

        throw new InvalidOperationException(
            $"Checker {typeof(T).Name} not found");
    }

    private static TField? GetField<TField>(object obj, string fieldName)
    {
        var field = obj.GetType()
            .GetField(fieldName, BindingFlags.NonPublic | BindingFlags.Instance)!;
        return (TField?)field.GetValue(obj);
    }
}
