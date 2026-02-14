using Confluent.Kafka;

namespace DepHealth.Checks;

/// <summary>
/// Kafka health checker — requests metadata from the broker.
/// </summary>
public sealed class KafkaChecker : IHealthChecker
{
    public DependencyType Type => DependencyType.Kafka;

    public Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        var config = new AdminClientConfig
        {
            BootstrapServers = $"{endpoint.Host}:{endpoint.Port}",
            SocketTimeoutMs = 5000
        };

        using var adminClient = new AdminClientBuilder(config).Build();
        var metadata = adminClient.GetMetadata(TimeSpan.FromSeconds(5));

        if (metadata.Brokers.Count == 0)
        {
            throw new Exceptions.UnhealthyException("Kafka: no brokers in metadata response", "no_brokers");
        }

        return Task.CompletedTask;
    }
}
