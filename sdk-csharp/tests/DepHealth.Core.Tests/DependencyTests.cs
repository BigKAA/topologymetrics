namespace DepHealth.Core.Tests;

public class DependencyTests
{
    [Fact]
    public void Builder_ValidDependency()
    {
        var dep = Dependency.CreateBuilder("my-db", DependencyType.Postgres)
            .WithEndpoint(new Endpoint("db.local", "5432"))
            .WithCritical(true)
            .Build();

        Assert.Equal("my-db", dep.Name);
        Assert.Equal(DependencyType.Postgres, dep.Type);
        Assert.True(dep.Critical);
        Assert.Single(dep.Endpoints);
        Assert.Equal("db.local", dep.Endpoints[0].Host);
    }

    [Fact]
    public void Builder_MultipleEndpoints()
    {
        var dep = Dependency.CreateBuilder("kafka", DependencyType.Kafka)
            .WithEndpoints(new[]
            {
                new Endpoint("broker1", "9092"),
                new Endpoint("broker2", "9092")
            })
            .Build();

        Assert.Equal(2, dep.Endpoints.Count);
    }

    [Fact]
    public void Builder_EmptyName_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            Dependency.CreateBuilder("", DependencyType.Http)
                .WithEndpoint(new Endpoint("host", "80"))
                .Build());
    }

    [Fact]
    public void Builder_InvalidNamePattern_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            Dependency.CreateBuilder("My-DB", DependencyType.Postgres)
                .WithEndpoint(new Endpoint("host", "5432"))
                .Build());
    }

    [Fact]
    public void Builder_NoEndpoints_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            Dependency.CreateBuilder("my-db", DependencyType.Postgres)
                .Build());
    }

    [Fact]
    public void Builder_CustomConfig()
    {
        var config = CheckConfig.CreateBuilder()
            .WithInterval(TimeSpan.FromSeconds(10))
            .WithTimeout(TimeSpan.FromSeconds(3))
            .Build();

        var dep = Dependency.CreateBuilder("cache", DependencyType.Redis)
            .WithEndpoint(new Endpoint("redis", "6379"))
            .WithConfig(config)
            .Build();

        Assert.Equal(TimeSpan.FromSeconds(10), dep.Config.Interval);
    }
}
