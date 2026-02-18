namespace DepHealth.Core.Tests;

public class DepHealthMonitorTests
{
    [Fact]
    public void Builder_CreateAndDispose()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddHttp("api", "http://localhost:8080", critical: true)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_MultipleTypes()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddHttp("api", "http://localhost:8080", critical: true)
            .AddTcp("service", "localhost", "9090", critical: false)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithPostgresUrl()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddPostgres("db", "postgres://user:pass@db.local:5432/mydb", critical: true)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithRedisUrl()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddRedis("cache", "redis://cache.local:6379", critical: false)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithAmqpUrl()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddAmqp("mq", "amqp://user:pass@broker.local:5672/%2F", critical: false)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithKafkaUrl()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddKafka("events", "kafka://broker1:9092,broker2:9092", critical: false)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithGrpc()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddGrpc("grpc-svc", "localhost", "50051", critical: false)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithCritical()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddHttp("api", "http://localhost:8080", critical: true)
            .AddPostgres("db", "postgres://localhost:5432/db", critical: true)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Health_BeforeStart_ReturnsEmpty()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddHttp("api", "http://localhost:8080", critical: false)
            .Build();

        Assert.Empty(dh.Health());
    }

    [Fact]
    public void Builder_MissingName_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            DepHealthMonitor.CreateBuilder("", "test-group"));
    }

    [Fact]
    public void Builder_InvalidName_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            DepHealthMonitor.CreateBuilder("My-App", "test-group"));
    }

    [Fact]
    public void Builder_MissingGroup_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            DepHealthMonitor.CreateBuilder("test-app", ""));
    }

    [Fact]
    public void Builder_InvalidGroup_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            DepHealthMonitor.CreateBuilder("test-app", "My-Group"));
    }

    [Fact]
    public void Builder_MissingCritical_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            DepHealthMonitor.CreateBuilder("test-app", "test-group")
                .AddHttp("api", "http://localhost:8080")
                .Build());
    }

    [Fact]
    public void Builder_WithLabels()
    {
        var labels = new Dictionary<string, string> { ["region"] = "eu" };
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddHttp("api", "http://localhost:8080", critical: true, labels: labels)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_WithReservedLabel_Throws()
    {
        var labels = new Dictionary<string, string> { ["host"] = "bad" };
        Assert.Throws<ValidationException>(() =>
            DepHealthMonitor.CreateBuilder("test-app", "test-group")
                .AddHttp("api", "http://localhost:8080", critical: true, labels: labels));
    }

    [Fact]
    public void Builder_WithGroupReservedLabel_Throws()
    {
        var labels = new Dictionary<string, string> { ["group"] = "bad" };
        Assert.Throws<ValidationException>(() =>
            DepHealthMonitor.CreateBuilder("test-app", "test-group")
                .AddHttp("api", "http://localhost:8080", critical: true, labels: labels));
    }

    [Fact]
    public void Builder_NameFromEnv()
    {
        Environment.SetEnvironmentVariable("DEPHEALTH_NAME", "env-app");
        try
        {
            using var dh = DepHealthMonitor.CreateBuilder("", "test-group")
                .AddHttp("api", "http://localhost:8080", critical: true)
                .Build();
            Assert.NotNull(dh);
        }
        finally
        {
            Environment.SetEnvironmentVariable("DEPHEALTH_NAME", null);
        }
    }

    [Fact]
    public void Builder_GroupFromEnv()
    {
        Environment.SetEnvironmentVariable("DEPHEALTH_GROUP", "env-group");
        try
        {
            using var dh = DepHealthMonitor.CreateBuilder("test-app", "")
                .AddHttp("api", "http://localhost:8080", critical: true)
                .Build();
            Assert.NotNull(dh);
        }
        finally
        {
            Environment.SetEnvironmentVariable("DEPHEALTH_GROUP", null);
        }
    }

    [Fact]
    public void Builder_GroupApiOverridesEnv()
    {
        Environment.SetEnvironmentVariable("DEPHEALTH_GROUP", "env-group");
        try
        {
            // API value takes precedence over env var
            using var dh = DepHealthMonitor.CreateBuilder("test-app", "api-group")
                .AddHttp("api", "http://localhost:8080", critical: true)
                .Build();
            Assert.NotNull(dh);
        }
        finally
        {
            Environment.SetEnvironmentVariable("DEPHEALTH_GROUP", null);
        }
    }

    [Fact]
    public void Builder_MissingGroupNoEnv_Throws()
    {
        Environment.SetEnvironmentVariable("DEPHEALTH_GROUP", null);
        var ex = Assert.Throws<ValidationException>(() =>
            DepHealthMonitor.CreateBuilder("test-app", ""));
        Assert.Contains("group is required", ex.Message);
        Assert.Contains("DEPHEALTH_GROUP", ex.Message);
    }

    [Fact]
    public void Builder_CriticalFromEnv()
    {
        Environment.SetEnvironmentVariable("DEPHEALTH_API_CRITICAL", "yes");
        try
        {
            using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
                .AddHttp("api", "http://localhost:8080")
                .Build();
            Assert.NotNull(dh);
        }
        finally
        {
            Environment.SetEnvironmentVariable("DEPHEALTH_API_CRITICAL", null);
        }
    }

    [Fact]
    public void Builder_LabelFromEnv()
    {
        Environment.SetEnvironmentVariable("DEPHEALTH_API_LABEL_REGION", "us");
        try
        {
            using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
                .AddHttp("api", "http://localhost:8080", critical: true)
                .Build();
            Assert.NotNull(dh);
        }
        finally
        {
            Environment.SetEnvironmentVariable("DEPHEALTH_API_LABEL_REGION", null);
        }
    }

    [Fact]
    public void Builder_WithCustomChecker()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddCustom("svc", DependencyType.Http, "localhost", "8080",
                new FakeChecker(), critical: true)
            .Build();

        Assert.NotNull(dh);
    }

    [Fact]
    public void Builder_NoDependencies_Succeeds()
    {
        using var dh = DepHealthMonitor.CreateBuilder("leaf-app", "test-group").Build();
        Assert.NotNull(dh);
    }

    [Fact]
    public void Health_NoDependencies_ReturnsEmpty()
    {
        using var dh = DepHealthMonitor.CreateBuilder("leaf-app", "test-group").Build();
        Assert.Empty(dh.Health());
    }

    [Fact]
    public void StartStop_NoDependencies_Succeeds()
    {
        using var dh = DepHealthMonitor.CreateBuilder("leaf-app", "test-group").Build();
        dh.Start();
        dh.Stop();
    }

    private sealed class FakeChecker : IHealthChecker
    {
        public DependencyType Type => DependencyType.Http;

        public Task CheckAsync(Endpoint endpoint, CancellationToken ct = default) =>
            Task.CompletedTask;
    }
}
