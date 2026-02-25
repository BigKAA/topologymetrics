namespace DepHealth.Core.Tests;

/// <summary>
/// Integration tests for dynamic endpoint management on DepHealthMonitor facade:
/// AddEndpoint, RemoveEndpoint, UpdateEndpoint — input validation and delegation.
/// </summary>
public class DepHealthMonitorDynamicTests
{
    private static DepHealthMonitor CreateMonitor(
        TimeSpan? interval = null, TimeSpan? timeout = null)
    {
        var builder = DepHealthMonitor.CreateBuilder("test-app", "test-group")
            .WithCheckInterval(interval ?? TimeSpan.FromSeconds(2))
            .WithCheckTimeout(timeout ?? TimeSpan.FromSeconds(1))
            .WithInitialDelay(TimeSpan.Zero);

        return builder.Build();
    }

    // --- AddEndpoint ---

    [Fact]
    public void AddEndpoint_AfterStart_AppearsInHealth()
    {
        using var dh = CreateMonitor();
        dh.Start();

        var ep = new Endpoint("10.0.0.1", "8080");
        dh.AddEndpoint("dynamic-svc", DependencyType.Http, true, ep, new FakeChecker());

        Thread.Sleep(1500);

        var health = dh.Health();
        Assert.True(health.ContainsKey("dynamic-svc:10.0.0.1:8080"));
        Assert.True(health["dynamic-svc:10.0.0.1:8080"]);

        var details = dh.HealthDetails();
        var es = details["dynamic-svc:10.0.0.1:8080"];
        Assert.True(es.Healthy);
        Assert.Equal("http", es.Type);
        Assert.Equal("dynamic-svc", es.Name);

        dh.Stop();
    }

    [Fact]
    public void AddEndpoint_InvalidName_Throws()
    {
        using var dh = CreateMonitor();
        dh.Start();

        var ep = new Endpoint("10.0.0.1", "8080");
        Assert.Throws<ValidationException>(() =>
            dh.AddEndpoint("Invalid-Name", DependencyType.Http, true, ep, new FakeChecker()));

        dh.Stop();
    }

    [Fact]
    public void AddEndpoint_InvalidType_Throws()
    {
        using var dh = CreateMonitor();
        dh.Start();

        var ep = new Endpoint("10.0.0.1", "8080");
        Assert.Throws<ValidationException>(() =>
            dh.AddEndpoint("svc", (DependencyType)999, true, ep, new FakeChecker()));

        dh.Stop();
    }

    [Fact]
    public void AddEndpoint_MissingHost_Throws()
    {
        using var dh = CreateMonitor();
        dh.Start();

        var ep = new Endpoint("", "8080");
        Assert.Throws<ValidationException>(() =>
            dh.AddEndpoint("svc", DependencyType.Http, true, ep, new FakeChecker()));

        dh.Stop();
    }

    [Fact]
    public void AddEndpoint_ReservedLabel_Throws()
    {
        using var dh = CreateMonitor();
        dh.Start();

        var labels = new Dictionary<string, string> { ["host"] = "bad" };
        var ep = new Endpoint("10.0.0.1", "8080", labels);
        Assert.Throws<ValidationException>(() =>
            dh.AddEndpoint("svc", DependencyType.Http, true, ep, new FakeChecker()));

        dh.Stop();
    }

    // --- RemoveEndpoint ---

    [Fact]
    public void RemoveEndpoint_DisappearsFromHealth()
    {
        using var dh = CreateMonitor();
        dh.Start();

        var ep = new Endpoint("10.0.0.1", "8080");
        dh.AddEndpoint("svc", DependencyType.Tcp, false, ep, new FakeChecker());

        Thread.Sleep(1500);
        Assert.True(dh.Health().ContainsKey("svc:10.0.0.1:8080"));

        dh.RemoveEndpoint("svc", "10.0.0.1", "8080");

        Assert.False(dh.Health().ContainsKey("svc:10.0.0.1:8080"));
        Assert.False(dh.HealthDetails().ContainsKey("svc:10.0.0.1:8080"));

        dh.Stop();
    }

    // --- UpdateEndpoint ---

    [Fact]
    public void UpdateEndpoint_SwapsEndpoint()
    {
        using var dh = CreateMonitor();
        dh.Start();

        var oldEp = new Endpoint("10.0.0.1", "8080");
        dh.AddEndpoint("svc", DependencyType.Tcp, true, oldEp, new FakeChecker());

        Thread.Sleep(1500);
        Assert.True(dh.Health().ContainsKey("svc:10.0.0.1:8080"));

        var newEp = new Endpoint("10.0.0.2", "9090");
        dh.UpdateEndpoint("svc", "10.0.0.1", "8080", newEp, new FakeChecker());

        Assert.False(dh.HealthDetails().ContainsKey("svc:10.0.0.1:8080"));

        Thread.Sleep(1500);

        Assert.True(dh.Health().ContainsKey("svc:10.0.0.2:9090"));
        var es = dh.HealthDetails()["svc:10.0.0.2:9090"];
        Assert.True(es.Healthy);
        Assert.True(es.Critical);

        dh.Stop();
    }

    [Fact]
    public void UpdateEndpoint_MissingNewHost_Throws()
    {
        using var dh = CreateMonitor();
        dh.Start();

        var oldEp = new Endpoint("10.0.0.1", "8080");
        dh.AddEndpoint("svc", DependencyType.Tcp, false, oldEp, new FakeChecker());

        var newEp = new Endpoint("", "9090");
        Assert.Throws<ValidationException>(() =>
            dh.UpdateEndpoint("svc", "10.0.0.1", "8080", newEp, new FakeChecker()));

        dh.Stop();
    }

    [Fact]
    public void UpdateEndpoint_NotFound_Throws()
    {
        using var dh = CreateMonitor();
        dh.Start();

        var newEp = new Endpoint("10.0.0.2", "9090");
        Assert.Throws<DepHealth.Exceptions.EndpointNotFoundException>(() =>
            dh.UpdateEndpoint("no-such", "1.2.3.4", "9999", newEp, new FakeChecker()));

        dh.Stop();
    }

    // --- Config inheritance ---

    [Fact]
    public void AddEndpoint_InheritsGlobalConfig()
    {
        // Create monitor with custom interval/timeout
        using var dh = CreateMonitor(
            interval: TimeSpan.FromSeconds(2),
            timeout: TimeSpan.FromSeconds(1));
        dh.Start();

        var ep = new Endpoint("10.0.0.1", "8080");
        dh.AddEndpoint("svc", DependencyType.Tcp, false, ep, new FakeChecker());

        Thread.Sleep(1500);

        // If the endpoint uses global config, it should have completed a check by now
        var details = dh.HealthDetails();
        var key = "svc:10.0.0.1:8080";
        Assert.True(details.ContainsKey(key));
        var es = details[key];
        Assert.True(es.Healthy);
        Assert.NotNull(es.LastCheckedAt);
        Assert.True(es.Latency > TimeSpan.Zero);

        dh.Stop();
    }

    // --- Test checker ---

    private sealed class FakeChecker : IHealthChecker
    {
        public DependencyType Type => DependencyType.Http;

        public Task CheckAsync(Endpoint endpoint, CancellationToken ct = default) =>
            Task.CompletedTask;
    }
}
