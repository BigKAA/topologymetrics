using RabbitMQ.Client;

namespace DepHealth.Checks;

/// <summary>
/// AMQP (RabbitMQ) health checker — проверяет соединение connect → close.
/// </summary>
public sealed class AmqpChecker : IHealthChecker
{
    private readonly string? _username;
    private readonly string? _password;
    private readonly string _vhost;

    public DependencyType Type => DependencyType.Amqp;

    public AmqpChecker(string? username = null, string? password = null, string vhost = "/")
    {
        _username = username;
        _password = password;
        _vhost = vhost;
    }

    public Task CheckAsync(Endpoint endpoint, CancellationToken ct)
    {
        var factory = new ConnectionFactory
        {
            HostName = endpoint.Host,
            Port = endpoint.PortAsInt(),
            VirtualHost = _vhost,
            RequestedConnectionTimeout = TimeSpan.FromSeconds(5)
        };

        if (_username is not null)
        {
            factory.UserName = _username;
        }

        if (_password is not null)
        {
            factory.Password = _password;
        }

        using var connection = factory.CreateConnection();
        if (!connection.IsOpen)
        {
            throw new Exceptions.UnhealthyException("AMQP connection is not open");
        }

        return Task.CompletedTask;
    }
}
