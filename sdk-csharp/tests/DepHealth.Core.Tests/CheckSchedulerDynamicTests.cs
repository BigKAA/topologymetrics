using System.IO;
using Prometheus;

namespace DepHealth.Core.Tests;

/// <summary>
/// Tests for dynamic endpoint management on CheckScheduler:
/// AddEndpoint, RemoveEndpoint, UpdateEndpoint.
/// </summary>
public class CheckSchedulerDynamicTests
{
    private static CheckScheduler CreateScheduler(CollectorRegistry? registry = null)
    {
        registry ??= Metrics.NewCustomRegistry();
        var metrics = new PrometheusExporter("test-app", "test-group", registry: registry);
        var config = CheckConfig.CreateBuilder()
            .WithInterval(TimeSpan.FromSeconds(2))
            .WithTimeout(TimeSpan.FromSeconds(1))
            .WithInitialDelay(TimeSpan.Zero)
            .Build();
        return new CheckScheduler(metrics, config);
    }

    private static Endpoint CreateEp(string host = "127.0.0.1", string port = "1234",
        Dictionary<string, string>? labels = null)
    {
        return labels is not null
            ? new Endpoint(host, port, labels)
            : new Endpoint(host, port);
    }

    private static async Task<string> ScrapeAsync(CollectorRegistry registry)
    {
        using var stream = new MemoryStream();
        await registry.CollectAndExportAsTextAsync(stream);
        stream.Position = 0;
        using var reader = new StreamReader(stream);
        return await reader.ReadToEndAsync();
    }

    // --- AddEndpoint ---

    [Fact]
    public void AddEndpoint_AfterStart_AppearsInHealth()
    {
        var scheduler = CreateScheduler();
        scheduler.Start();

        var ep = CreateEp("10.0.0.1", "8080");
        scheduler.AddEndpoint("dynamic-svc", DependencyType.Http, true, ep, new SuccessChecker());

        Thread.Sleep(1500);

        var health = scheduler.Health();
        Assert.True(health.ContainsKey("dynamic-svc:10.0.0.1:8080"));
        Assert.True(health["dynamic-svc:10.0.0.1:8080"]);

        var details = scheduler.HealthDetails();
        var es = details["dynamic-svc:10.0.0.1:8080"];
        Assert.True(es.Healthy);
        Assert.Equal("http", es.Type);
        Assert.Equal("dynamic-svc", es.Name);
        Assert.Equal("10.0.0.1", es.Host);
        Assert.Equal("8080", es.Port);
        Assert.True(es.Critical);

        scheduler.Stop();
    }

    [Fact]
    public void AddEndpoint_Idempotent()
    {
        var scheduler = CreateScheduler();
        scheduler.Start();

        var ep = CreateEp("10.0.0.1", "8080");
        scheduler.AddEndpoint("svc", DependencyType.Tcp, false, ep, new SuccessChecker());
        scheduler.AddEndpoint("svc", DependencyType.Tcp, false, ep, new SuccessChecker());

        var details = scheduler.HealthDetails();
        var count = details.Keys.Count(k => k == "svc:10.0.0.1:8080");
        Assert.Equal(1, count);

        scheduler.Stop();
    }

    [Fact]
    public void AddEndpoint_BeforeStart_Throws()
    {
        var scheduler = CreateScheduler();
        var ep = CreateEp();

        Assert.Throws<InvalidOperationException>(() =>
            scheduler.AddEndpoint("svc", DependencyType.Tcp, false, ep, new SuccessChecker()));
    }

    [Fact]
    public void AddEndpoint_AfterStop_Throws()
    {
        var scheduler = CreateScheduler();
        scheduler.Start();
        scheduler.Stop();

        var ep = CreateEp();
        Assert.Throws<InvalidOperationException>(() =>
            scheduler.AddEndpoint("svc", DependencyType.Tcp, false, ep, new SuccessChecker()));
    }

    [Fact]
    public async Task AddEndpoint_Metrics()
    {
        var registry = Metrics.NewCustomRegistry();
        var scheduler = CreateScheduler(registry);
        scheduler.Start();

        var ep = CreateEp("db.local", "5432");
        scheduler.AddEndpoint("test-db", DependencyType.Postgres, true, ep, new SuccessChecker());

        Thread.Sleep(1500);

        var output = await ScrapeAsync(registry);
        Assert.Contains("app_dependency_health", output);
        Assert.Contains("dependency=\"test-db\"", output);
        Assert.Contains("type=\"postgres\"", output);
        Assert.Contains("host=\"db.local\"", output);
        Assert.Contains("port=\"5432\"", output);
        Assert.Contains("critical=\"yes\"", output);

        scheduler.Stop();
    }

    // --- RemoveEndpoint ---

    [Fact]
    public void RemoveEndpoint_AfterStart_DisappearsFromHealth()
    {
        var scheduler = CreateScheduler();
        scheduler.Start();

        var ep = CreateEp("10.0.0.1", "8080");
        scheduler.AddEndpoint("svc", DependencyType.Tcp, false, ep, new SuccessChecker());

        Thread.Sleep(1500);

        // Verify it's there
        Assert.True(scheduler.Health().ContainsKey("svc:10.0.0.1:8080"));

        scheduler.RemoveEndpoint("svc", "10.0.0.1", "8080");

        Assert.False(scheduler.Health().ContainsKey("svc:10.0.0.1:8080"));
        Assert.False(scheduler.HealthDetails().ContainsKey("svc:10.0.0.1:8080"));

        scheduler.Stop();
    }

    [Fact]
    public void RemoveEndpoint_Idempotent()
    {
        var scheduler = CreateScheduler();
        scheduler.Start();

        // Remove non-existent — no exception
        scheduler.RemoveEndpoint("no-such", "1.2.3.4", "9999");

        scheduler.Stop();
    }

    [Fact]
    public async Task RemoveEndpoint_MetricsDeleted()
    {
        var registry = Metrics.NewCustomRegistry();
        var scheduler = CreateScheduler(registry);
        scheduler.Start();

        var ep = CreateEp("db.local", "5432");
        scheduler.AddEndpoint("test-db", DependencyType.Postgres, true, ep, new SuccessChecker());

        Thread.Sleep(1500);

        // Verify metrics present
        var before = await ScrapeAsync(registry);
        Assert.Contains("host=\"db.local\"", before);

        scheduler.RemoveEndpoint("test-db", "db.local", "5432");

        // After removal, health gauge for this endpoint should be gone
        var after = await ScrapeAsync(registry);
        Assert.DoesNotContain("host=\"db.local\"", after);

        scheduler.Stop();
    }

    [Fact]
    public void RemoveEndpoint_BeforeStart_Throws()
    {
        var scheduler = CreateScheduler();
        Assert.Throws<InvalidOperationException>(() =>
            scheduler.RemoveEndpoint("svc", "127.0.0.1", "8080"));
    }

    // --- UpdateEndpoint ---

    [Fact]
    public void UpdateEndpoint_SwapsEndpoint()
    {
        var scheduler = CreateScheduler();
        scheduler.Start();

        var oldEp = CreateEp("10.0.0.1", "8080");
        scheduler.AddEndpoint("svc", DependencyType.Tcp, true, oldEp, new SuccessChecker());

        Thread.Sleep(1500);
        Assert.True(scheduler.Health().ContainsKey("svc:10.0.0.1:8080"));

        var newEp = CreateEp("10.0.0.2", "9090");
        scheduler.UpdateEndpoint("svc", "10.0.0.1", "8080", newEp, new SuccessChecker());

        // Old gone immediately
        Assert.False(scheduler.HealthDetails().ContainsKey("svc:10.0.0.1:8080"));

        Thread.Sleep(1500);

        // New appeared
        Assert.True(scheduler.Health().ContainsKey("svc:10.0.0.2:9090"));
        var es = scheduler.HealthDetails()["svc:10.0.0.2:9090"];
        Assert.True(es.Healthy);
        Assert.True(es.Critical); // critical flag preserved from old

        scheduler.Stop();
    }

    [Fact]
    public void UpdateEndpoint_NotFound_Throws()
    {
        var scheduler = CreateScheduler();
        scheduler.Start();

        var newEp = CreateEp("10.0.0.2", "9090");
        Assert.Throws<DepHealth.Exceptions.EndpointNotFoundException>(() =>
            scheduler.UpdateEndpoint("no-such", "1.2.3.4", "9999", newEp, new SuccessChecker()));

        scheduler.Stop();
    }

    [Fact]
    public async Task UpdateEndpoint_MetricsSwap()
    {
        var registry = Metrics.NewCustomRegistry();
        var scheduler = CreateScheduler(registry);
        scheduler.Start();

        var oldEp = CreateEp("old-host", "1111");
        scheduler.AddEndpoint("svc", DependencyType.Tcp, false, oldEp, new SuccessChecker());

        Thread.Sleep(1500);

        var before = await ScrapeAsync(registry);
        Assert.Contains("host=\"old-host\"", before);

        var newEp = CreateEp("new-host", "2222");
        scheduler.UpdateEndpoint("svc", "old-host", "1111", newEp, new SuccessChecker());

        Thread.Sleep(1500);

        var after = await ScrapeAsync(registry);
        Assert.DoesNotContain("host=\"old-host\"", after);
        Assert.Contains("host=\"new-host\"", after);

        scheduler.Stop();
    }

    // --- Stop after dynamic add ---

    [Fact]
    public void StopAfterDynamicAdd_CleansUp()
    {
        var scheduler = CreateScheduler();
        scheduler.Start();

        var ep = CreateEp("10.0.0.1", "8080");
        scheduler.AddEndpoint("svc", DependencyType.Tcp, false, ep, new SuccessChecker());

        Thread.Sleep(500);

        // Stop should not throw and should complete cleanly
        scheduler.Stop();

        // After stop, data still accessible (last known state)
        var details = scheduler.HealthDetails();
        Assert.True(details.ContainsKey("svc:10.0.0.1:8080"));
    }

    // --- Concurrent Add/Remove/Health ---

    [Fact]
    public async Task ConcurrentAddRemoveHealth_NoExceptions()
    {
        var scheduler = CreateScheduler();
        scheduler.Start();

        var tasks = new List<Task>();
        // Concurrent adds
        for (var i = 0; i < 10; i++)
        {
            var idx = i;
            tasks.Add(Task.Run(() =>
            {
                var ep = CreateEp($"host-{idx}", $"{3000 + idx}");
                scheduler.AddEndpoint($"svc-{idx}", DependencyType.Tcp, false, ep, new SuccessChecker());
            }));
        }

        await Task.WhenAll(tasks);
        tasks.Clear();

        Thread.Sleep(1500);

        // Concurrent reads + removes
        for (var i = 0; i < 10; i++)
        {
            var idx = i;
            tasks.Add(Task.Run(() =>
            {
                // Read health (should not throw)
                for (var j = 0; j < 50; j++)
                {
                    var h = scheduler.Health();
                    var d = scheduler.HealthDetails();
                    Assert.NotNull(h);
                    Assert.NotNull(d);
                }
            }));

            // Remove half the endpoints concurrently
            if (idx % 2 == 0)
            {
                tasks.Add(Task.Run(() =>
                {
                    scheduler.RemoveEndpoint($"svc-{idx}", $"host-{idx}", $"{3000 + idx}");
                }));
            }
        }

        await Task.WhenAll(tasks);

        scheduler.Stop();
    }

    // --- Test checkers ---

    private sealed class SuccessChecker : IHealthChecker
    {
        public DependencyType Type => DependencyType.Tcp;
        public Task CheckAsync(Endpoint endpoint, CancellationToken ct) => Task.CompletedTask;
    }

}
