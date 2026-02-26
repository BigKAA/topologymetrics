using RabbitMQ.Client;

namespace DepHealth.Checks;

/// <summary>
/// AMQP (RabbitMQ) health checker — verifies connectivity via connect and close.
/// </summary>
public sealed class AmqpChecker : IHealthChecker
{
    private readonly string? _username;
    private readonly string? _password;
    private readonly string _vhost;

    /// <summary>Gets the dependency type for this checker.</summary>
    public DependencyType Type => DependencyType.Amqp;

    /// <summary>Creates a new instance of <see cref="AmqpChecker"/>.</summary>
    /// <param name="username">Optional AMQP username for authentication.</param>
    /// <param name="password">Optional AMQP password for authentication.</param>
    /// <param name="vhost">Virtual host to connect to (default: <c>/</c>).</param>
    public AmqpChecker(string? username = null, string? password = null, string vhost = "/")
    {
        _username = username;
        _password = password;
        _vhost = vhost;
    }

    /// <inheritdoc />
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

        try
        {
            using var connection = factory.CreateConnection();
            if (!connection.IsOpen)
            {
                throw new Exceptions.UnhealthyException("AMQP connection is not open");
            }
        }
        catch (Exceptions.DepHealthException)
        {
            throw;
        }
        catch (Exception e)
        {
            throw ClassifyAmqpError(e);
        }

        return Task.CompletedTask;
    }

    private static Exception ClassifyAmqpError(Exception e)
    {
        var msg = e.Message ?? "";
        if (msg.Contains("403", StringComparison.Ordinal)
            || msg.Contains("ACCESS_REFUSED", StringComparison.OrdinalIgnoreCase))
        {
            return new Exceptions.CheckAuthException("AMQP auth error: " + msg, e);
        }

        return e;
    }
}
