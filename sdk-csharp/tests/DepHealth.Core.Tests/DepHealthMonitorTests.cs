namespace DepHealth.Core.Tests;

public class DepHealthMonitorTests
{
    [Fact]
    public void Builder_CreateAndDispose()
    {
        using var dh = DepHealthMonitor.CreateBuilder()
            .AddHttp("api", "http://localhost:8080")
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_MultipleTypes()
    {
        using var dh = DepHealthMonitor.CreateBuilder()
            .AddHttp("api", "http://localhost:8080")
            .AddTcp("service", "localhost", "9090")
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithPostgresUrl()
    {
        using var dh = DepHealthMonitor.CreateBuilder()
            .AddPostgres("db", "postgres://user:pass@db.local:5432/mydb")
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithRedisUrl()
    {
        using var dh = DepHealthMonitor.CreateBuilder()
            .AddRedis("cache", "redis://cache.local:6379")
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithAmqpUrl()
    {
        using var dh = DepHealthMonitor.CreateBuilder()
            .AddAmqp("mq", "amqp://user:pass@broker.local:5672/%2F")
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithKafkaUrl()
    {
        using var dh = DepHealthMonitor.CreateBuilder()
            .AddKafka("events", "kafka://broker1:9092,broker2:9092")
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithGrpc()
    {
        using var dh = DepHealthMonitor.CreateBuilder()
            .AddGrpc("grpc-svc", "localhost", "50051")
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithCritical()
    {
        using var dh = DepHealthMonitor.CreateBuilder()
            .AddHttp("api", "http://localhost:8080", critical: true)
            .AddPostgres("db", "postgres://localhost:5432/db", critical: true)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Health_BeforeStart_ReturnsEmpty()
    {
        using var dh = DepHealthMonitor.CreateBuilder()
            .AddHttp("api", "http://localhost:8080")
            .Build();

        Assert.Empty(dh.Health());
    }
}
