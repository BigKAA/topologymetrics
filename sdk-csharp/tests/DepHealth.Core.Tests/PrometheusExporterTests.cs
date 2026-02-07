using System.IO;
using Prometheus;

namespace DepHealth.Core.Tests;

public class PrometheusExporterTests
{
    [Fact]
    public async Task SetHealth_ExportsGauge()
    {
        var registry = Metrics.NewCustomRegistry();
        var exporter = new PrometheusExporter(registry);
        var dep = CreateDep("test-db", DependencyType.Postgres, "db.local", "5432");
        var ep = dep.Endpoints[0];

        exporter.SetHealth(dep, ep, 1.0);

        var output = await ScrapeAsync(registry);
        Assert.Contains("app_dependency_health", output);
        Assert.Contains("dependency=\"test-db\"", output);
        Assert.Contains("type=\"postgres\"", output);
        Assert.Contains("host=\"db.local\"", output);
        Assert.Contains("port=\"5432\"", output);
    }

    [Fact]
    public async Task ObserveLatency_ExportsHistogram()
    {
        var registry = Metrics.NewCustomRegistry();
        var exporter = new PrometheusExporter(registry);
        var dep = CreateDep("test-cache", DependencyType.Redis, "cache", "6379");
        var ep = dep.Endpoints[0];

        exporter.ObserveLatency(dep, ep, TimeSpan.FromMilliseconds(50));

        var output = await ScrapeAsync(registry);
        Assert.Contains("app_dependency_latency_seconds", output);
    }

    [Fact]
    public async Task HealthDescription_MatchesSpec()
    {
        var registry = Metrics.NewCustomRegistry();
        var exporter = new PrometheusExporter(registry);
        var dep = CreateDep("test-db", DependencyType.Postgres, "db", "5432");
        var ep = dep.Endpoints[0];

        exporter.SetHealth(dep, ep, 1.0);

        var output = await ScrapeAsync(registry);
        Assert.Contains("Health status of a dependency (1 = healthy, 0 = unhealthy)", output);
    }

    private static Dependency CreateDep(string name, DependencyType type, string host, string port)
    {
        return Dependency.CreateBuilder(name, type)
            .WithEndpoint(new Endpoint(host, port))
            .Build();
    }

    private static async Task<string> ScrapeAsync(CollectorRegistry registry)
    {
        using var stream = new MemoryStream();
        await registry.CollectAndExportAsTextAsync(stream);
        stream.Position = 0;
        using var reader = new StreamReader(stream);
        return await reader.ReadToEndAsync();
    }
}
