using System.Text.Json;
using Prometheus;

namespace DepHealth.Core.Tests;

public class HealthDetailsTests
{
    private static CheckScheduler CreateScheduler()
    {
        var registry = Metrics.NewCustomRegistry();
        var metrics = new PrometheusExporter("test-app", "test-group", registry: registry);
        var config = CheckConfig.Defaults();
        return new CheckScheduler(metrics, config);
    }

    private static Dependency CreateDep(
        string name, bool critical,
        string host = "127.0.0.1", string port = "1234",
        DependencyType type = DependencyType.Tcp,
        Dictionary<string, string>? labels = null)
    {
        var ep = labels is not null
            ? new Endpoint(host, port, labels)
            : new Endpoint(host, port);

        return Dependency.CreateBuilder(name, type)
            .WithCritical(critical)
            .WithEndpoint(ep)
            .WithConfig(CheckConfig.CreateBuilder()
                .WithInterval(TimeSpan.FromSeconds(2))
                .WithTimeout(TimeSpan.FromSeconds(1))
                .WithInitialDelay(TimeSpan.Zero)
                .Build())
            .Build();
    }

    [Fact]
    public void EmptyBeforeAddingDependencies()
    {
        var scheduler = CreateScheduler();
        var details = scheduler.HealthDetails();
        Assert.Empty(details);
    }

    [Fact]
    public void UnknownStateBeforeFirstCheck()
    {
        var scheduler = CreateScheduler();
        var labels = new Dictionary<string, string> { ["role"] = "primary" };

        var ep = new Endpoint("pg.svc", "5432", labels);
        var dep = Dependency.CreateBuilder("test-dep", DependencyType.Postgres)
            .WithCritical(true)
            .WithEndpoint(ep)
            .WithConfig(CheckConfig.CreateBuilder()
                .WithInterval(TimeSpan.FromSeconds(60))
                .WithTimeout(TimeSpan.FromSeconds(5))
                .WithInitialDelay(TimeSpan.FromMinutes(5))
                .Build())
            .Build();

        scheduler.AddDependency(dep, new BlockingChecker());

        // Start with large initial delay — check won't run yet
        scheduler.Start();
        Thread.Sleep(100);

        var details = scheduler.HealthDetails();
        Assert.Single(details);

        var key = "test-dep:pg.svc:5432";
        Assert.True(details.ContainsKey(key));
        var es = details[key];

        // UNKNOWN state
        Assert.Null(es.Healthy);
        Assert.Equal(StatusCategory.Unknown, es.Status);
        Assert.Equal("unknown", es.Detail);
        Assert.Equal(TimeSpan.Zero, es.Latency);
        Assert.Null(es.LastCheckedAt);

        // Static fields
        Assert.Equal("postgres", es.Type);
        Assert.Equal("test-dep", es.Name);
        Assert.Equal("pg.svc", es.Host);
        Assert.Equal("5432", es.Port);
        Assert.True(es.Critical);
        Assert.Equal("primary", es.Labels["role"]);

        scheduler.Stop();
    }

    [Fact]
    public void HealthyEndpoint()
    {
        var scheduler = CreateScheduler();
        var dep = CreateDep("test-dep", critical: false);
        scheduler.AddDependency(dep, new SuccessChecker());
        scheduler.Start();

        Thread.Sleep(1500);

        var details = scheduler.HealthDetails();
        var key = "test-dep:127.0.0.1:1234";
        Assert.True(details.ContainsKey(key));
        var es = details[key];

        Assert.True(es.Healthy);
        Assert.Equal(StatusCategory.Ok, es.Status);
        Assert.Equal("ok", es.Detail);
        Assert.True(es.Latency > TimeSpan.Zero);
        Assert.NotNull(es.LastCheckedAt);
        Assert.Equal("tcp", es.Type);

        scheduler.Stop();
    }

    [Fact]
    public void UnhealthyEndpoint()
    {
        var scheduler = CreateScheduler();
        var dep = CreateDep("test-dep", critical: true);
        scheduler.AddDependency(dep, new FailChecker());
        scheduler.Start();

        Thread.Sleep(1500);

        var details = scheduler.HealthDetails();
        var key = "test-dep:127.0.0.1:1234";
        var es = details[key];

        Assert.False(es.Healthy);
        Assert.Equal(StatusCategory.Error, es.Status);
        Assert.Equal("error", es.Detail);
        Assert.True(es.Latency > TimeSpan.Zero);
        Assert.NotNull(es.LastCheckedAt);

        scheduler.Stop();
    }

    [Fact]
    public void KeysMatchHealth()
    {
        var scheduler = CreateScheduler();

        var ep1 = new Endpoint("host-1", "1111");
        var ep2 = new Endpoint("host-2", "2222");
        var dep = Dependency.CreateBuilder("multi-ep", DependencyType.Tcp)
            .WithCritical(false)
            .WithEndpoints([ep1, ep2])
            .WithConfig(CheckConfig.CreateBuilder()
                .WithInterval(TimeSpan.FromSeconds(2))
                .WithTimeout(TimeSpan.FromSeconds(1))
                .WithInitialDelay(TimeSpan.Zero)
                .Build())
            .Build();

        scheduler.AddDependency(dep, new SuccessChecker());
        scheduler.Start();

        Thread.Sleep(1500);

        var health = scheduler.Health();
        var details = scheduler.HealthDetails();

        // All keys from Health() must be in HealthDetails()
        foreach (var key in health.Keys)
        {
            Assert.True(details.ContainsKey(key),
                $"Key {key} in Health() but not in HealthDetails()");
        }

        // HealthDetails() includes all endpoints (same or more)
        Assert.True(details.Count >= health.Count);

        scheduler.Stop();
    }

    [Fact]
    public async Task ConcurrentAccess()
    {
        var scheduler = CreateScheduler();
        var dep = CreateDep("test-dep", critical: false);
        scheduler.AddDependency(dep, new SuccessChecker());
        scheduler.Start();

        await Task.Delay(1500);

        // Launch concurrent readers
        var tasks = new List<Task>();
        for (var i = 0; i < 10; i++)
        {
            tasks.Add(Task.Run(() =>
            {
                for (var j = 0; j < 100; j++)
                {
                    var details = scheduler.HealthDetails();
                    Assert.NotNull(details);
                }
            }));
        }

        await Task.WhenAll(tasks);
        scheduler.Stop();
    }

    [Fact]
    public void AfterStop()
    {
        var scheduler = CreateScheduler();
        var dep = CreateDep("test-dep", critical: false);
        scheduler.AddDependency(dep, new SuccessChecker());
        scheduler.Start();

        Thread.Sleep(1500);
        scheduler.Stop();

        // After Stop, HealthDetails should return last known state
        var details = scheduler.HealthDetails();
        Assert.NotEmpty(details);

        var key = "test-dep:127.0.0.1:1234";
        var es = details[key];
        Assert.True(es.Healthy);
    }

    [Fact]
    public void LabelsEmpty()
    {
        var scheduler = CreateScheduler();
        var dep = CreateDep("test-dep", critical: false);
        scheduler.AddDependency(dep, new SuccessChecker());
        scheduler.Start();

        Thread.Sleep(1500);

        var details = scheduler.HealthDetails();
        var key = "test-dep:127.0.0.1:1234";
        var es = details[key];

        // Labels should be empty, not null
        Assert.NotNull(es.Labels);
        Assert.Empty(es.Labels);

        scheduler.Stop();
    }

    [Fact]
    public void ResultMapIndependent()
    {
        var scheduler = CreateScheduler();
        var dep = CreateDep("test-dep", critical: false);
        scheduler.AddDependency(dep, new SuccessChecker());
        scheduler.Start();

        Thread.Sleep(1500);

        var details = scheduler.HealthDetails();
        var key = "test-dep:127.0.0.1:1234";

        // Modify returned map — should not affect internal state
        details.Remove(key);

        var details2 = scheduler.HealthDetails();
        Assert.True(details2.ContainsKey(key),
            "Internal state corrupted by external modification");
        Assert.Equal("test-dep", details2[key].Name);

        scheduler.Stop();
    }

    [Fact]
    public void LatencyMillisProperty()
    {
        var es = new EndpointStatus(
            healthy: true,
            status: StatusCategory.Ok,
            detail: "ok",
            latency: TimeSpan.FromMilliseconds(2.5),
            type: "tcp",
            name: "test",
            host: "localhost",
            port: "1234",
            critical: false,
            lastCheckedAt: DateTimeOffset.UtcNow,
            labels: new Dictionary<string, string>());

        Assert.Equal(2.5, es.LatencyMillis);
    }

    [Fact]
    public void FacadeDelegates()
    {
        using var dh = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .AddTcp("svc", "localhost", "9090", critical: false)
            .Build();

        // Before Start — HealthDetails returns empty (no checks started)
        var before = dh.HealthDetails();
        Assert.NotNull(before);

        // Endpoints are registered — should have UNKNOWN state
        Assert.Single(before);
        var key = "svc:localhost:9090";
        Assert.True(before.ContainsKey(key));
        Assert.Null(before[key].Healthy);
        Assert.Equal(StatusCategory.Unknown, before[key].Status);
    }

    [Fact]
    public void JsonSerializationHealthy()
    {
        var now = new DateTimeOffset(2026, 2, 14, 10, 30, 0, TimeSpan.Zero);
        var es = new EndpointStatus(
            healthy: true,
            status: StatusCategory.Ok,
            detail: "ok",
            latency: TimeSpan.FromMilliseconds(2.3),
            type: "postgres",
            name: "postgres-main",
            host: "pg.svc",
            port: "5432",
            critical: true,
            lastCheckedAt: now,
            labels: new Dictionary<string, string> { ["role"] = "primary" });

        var json = JsonSerializer.Serialize(es);
        using var doc = JsonDocument.Parse(json);
        var root = doc.RootElement;

        Assert.True(root.GetProperty("healthy").GetBoolean());
        Assert.Equal("ok", root.GetProperty("status").GetString());
        Assert.Equal("ok", root.GetProperty("detail").GetString());
        Assert.Equal(2.3, root.GetProperty("latency_ms").GetDouble(), 3);
        Assert.Equal("postgres", root.GetProperty("type").GetString());
        Assert.Equal("postgres-main", root.GetProperty("name").GetString());
        Assert.Equal("pg.svc", root.GetProperty("host").GetString());
        Assert.Equal("5432", root.GetProperty("port").GetString());
        Assert.True(root.GetProperty("critical").GetBoolean());
        Assert.Equal("primary", root.GetProperty("labels").GetProperty("role").GetString());

        // last_checked_at should be a string (ISO 8601)
        Assert.Equal(JsonValueKind.String, root.GetProperty("last_checked_at").ValueKind);
    }

    [Fact]
    public void JsonSerializationUnknown()
    {
        var es = new EndpointStatus(
            healthy: null,
            status: StatusCategory.Unknown,
            detail: "unknown",
            latency: TimeSpan.Zero,
            type: "redis",
            name: "redis-cache",
            host: "redis.svc",
            port: "6379",
            critical: false,
            lastCheckedAt: null,
            labels: new Dictionary<string, string>());

        var json = JsonSerializer.Serialize(es);
        using var doc = JsonDocument.Parse(json);
        var root = doc.RootElement;

        Assert.Equal(JsonValueKind.Null, root.GetProperty("healthy").ValueKind);
        Assert.Equal("unknown", root.GetProperty("status").GetString());
        Assert.Equal(0.0, root.GetProperty("latency_ms").GetDouble());
        Assert.Equal(JsonValueKind.Null, root.GetProperty("last_checked_at").ValueKind);
    }

    // --- Test checkers ---

    private sealed class SuccessChecker : IHealthChecker
    {
        public DependencyType Type => DependencyType.Tcp;
        public Task CheckAsync(Endpoint endpoint, CancellationToken ct) => Task.CompletedTask;
    }

    private sealed class FailChecker : IHealthChecker
    {
        public DependencyType Type => DependencyType.Tcp;

        public Task CheckAsync(Endpoint endpoint, CancellationToken ct) =>
            Task.FromException(new Exception("connection refused"));
    }

    private sealed class BlockingChecker : IHealthChecker
    {
        public DependencyType Type => DependencyType.Postgres;

        public async Task CheckAsync(Endpoint endpoint, CancellationToken ct)
        {
            await Task.Delay(Timeout.Infinite, ct).ConfigureAwait(false);
        }
    }
}
