using System.IO;
using Prometheus;

namespace DepHealth.Core.Tests;

public class PrometheusExporterTests
{
    private const string InstanceName = "test-app";

    [Fact]
    public async Task SetHealth_ExportsGaugeWithAllLabels()
    {
        var registry = Metrics.NewCustomRegistry();
        var exporter = new PrometheusExporter(InstanceName, registry: registry);
        var dep = CreateDep("test-db", DependencyType.Postgres, true, "db.local", "5432");
        var ep = dep.Endpoints[0];

        exporter.SetHealth(dep, ep, 1.0);

        var output = await ScrapeAsync(registry);
        Assert.Contains("app_dependency_health", output);
        Assert.Contains("name=\"test-app\"", output);
        Assert.Contains("dependency=\"test-db\"", output);
        Assert.Contains("type=\"postgres\"", output);
        Assert.Contains("host=\"db.local\"", output);
        Assert.Contains("port=\"5432\"", output);
        Assert.Contains("critical=\"yes\"", output);
    }

    [Fact]
    public async Task SetHealth_CriticalNo()
    {
        var registry = Metrics.NewCustomRegistry();
        var exporter = new PrometheusExporter(InstanceName, registry: registry);
        var dep = CreateDep("test-cache", DependencyType.Redis, false, "cache", "6379");
        var ep = dep.Endpoints[0];

        exporter.SetHealth(dep, ep, 1.0);

        var output = await ScrapeAsync(registry);
        Assert.Contains("critical=\"no\"", output);
    }

    [Fact]
    public async Task ObserveLatency_ExportsHistogram()
    {
        var registry = Metrics.NewCustomRegistry();
        var exporter = new PrometheusExporter(InstanceName, registry: registry);
        var dep = CreateDep("test-cache", DependencyType.Redis, false, "cache", "6379");
        var ep = dep.Endpoints[0];

        exporter.ObserveLatency(dep, ep, TimeSpan.FromMilliseconds(50));

        var output = await ScrapeAsync(registry);
        Assert.Contains("app_dependency_latency_seconds", output);
        Assert.Contains("name=\"test-app\"", output);
    }

    [Fact]
    public async Task HealthDescription_MatchesSpec()
    {
        var registry = Metrics.NewCustomRegistry();
        var exporter = new PrometheusExporter(InstanceName, registry: registry);
        var dep = CreateDep("test-db", DependencyType.Postgres, true, "db", "5432");
        var ep = dep.Endpoints[0];

        exporter.SetHealth(dep, ep, 1.0);

        var output = await ScrapeAsync(registry);
        Assert.Contains("Health status of a dependency (1 = healthy, 0 = unhealthy)", output);
    }

    [Fact]
    public async Task CustomLabels_ExportedInOrder()
    {
        var registry = Metrics.NewCustomRegistry();
        var customLabels = new[] { "region", "shard" };
        var exporter = new PrometheusExporter(InstanceName, customLabels, registry);

        var labels = new Dictionary<string, string>
        {
            ["region"] = "eu",
            ["shard"] = "1"
        };
        var dep = Dependency.CreateBuilder("test-db", DependencyType.Postgres)
            .WithEndpoint(new Endpoint("db.local", "5432", labels))
            .WithCritical(true)
            .Build();
        var ep = dep.Endpoints[0];

        exporter.SetHealth(dep, ep, 1.0);

        var output = await ScrapeAsync(registry);
        Assert.Contains("region=\"eu\"", output);
        Assert.Contains("shard=\"1\"", output);
    }

    [Fact]
    public async Task CustomLabels_MissingValue_EmptyString()
    {
        var registry = Metrics.NewCustomRegistry();
        var customLabels = new[] { "region" };
        var exporter = new PrometheusExporter(InstanceName, customLabels, registry);

        var dep = CreateDep("test-db", DependencyType.Postgres, true, "db.local", "5432");
        var ep = dep.Endpoints[0];

        exporter.SetHealth(dep, ep, 1.0);

        var output = await ScrapeAsync(registry);
        Assert.Contains("region=\"\"", output);
    }

    [Fact]
    public async Task LabelOrder_NameDependencyTypeHostPortCriticalCustom()
    {
        var registry = Metrics.NewCustomRegistry();
        var customLabels = new[] { "region" };
        var exporter = new PrometheusExporter(InstanceName, customLabels, registry);

        var labels = new Dictionary<string, string> { ["region"] = "eu" };
        var dep = Dependency.CreateBuilder("test-db", DependencyType.Postgres)
            .WithEndpoint(new Endpoint("db.local", "5432", labels))
            .WithCritical(true)
            .Build();
        var ep = dep.Endpoints[0];

        exporter.SetHealth(dep, ep, 1.0);

        var output = await ScrapeAsync(registry);
        // Verify that label order is correct
        var nameIdx = output.IndexOf("name=\"test-app\"");
        var depIdx = output.IndexOf("dependency=\"test-db\"");
        var typeIdx = output.IndexOf("type=\"postgres\"");
        var hostIdx = output.IndexOf("host=\"db.local\"");
        var portIdx = output.IndexOf("port=\"5432\"");
        var critIdx = output.IndexOf("critical=\"yes\"");
        var regionIdx = output.IndexOf("region=\"eu\"");

        Assert.True(nameIdx < depIdx, "name before dependency");
        Assert.True(depIdx < typeIdx, "dependency before type");
        Assert.True(typeIdx < hostIdx, "type before host");
        Assert.True(hostIdx < portIdx, "host before port");
        Assert.True(portIdx < critIdx, "port before critical");
        Assert.True(critIdx < regionIdx, "critical before custom");
    }

    private static Dependency CreateDep(string name, DependencyType type,
        bool critical, string host, string port)
    {
        return Dependency.CreateBuilder(name, type)
            .WithEndpoint(new Endpoint(host, port))
            .WithCritical(critical)
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
