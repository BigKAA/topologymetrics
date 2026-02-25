using DepHealth.Checks;

namespace DepHealth.Core.Tests.Checks;

public class CheckerTypeTests
{
    [Fact]
    public void TcpChecker_HasCorrectType()
    {
        var checker = new TcpChecker();
        Assert.Equal(DependencyType.Tcp, checker.Type);
    }

    [Fact]
    public void HttpChecker_HasCorrectType()
    {
        var checker = new HttpChecker();
        Assert.Equal(DependencyType.Http, checker.Type);
    }

    [Fact]
    public void HttpChecker_DefaultHealthPath()
    {
        var checker = new HttpChecker();
        Assert.Equal(DependencyType.Http, checker.Type);
    }

    [Fact]
    public void HttpChecker_CustomHealthPath()
    {
        var checker = new HttpChecker(healthPath: "/ready", tlsEnabled: true, tlsSkipVerify: true);
        Assert.Equal(DependencyType.Http, checker.Type);
    }

    [Fact]
    public void GrpcChecker_HasCorrectType()
    {
        var checker = new GrpcChecker();
        Assert.Equal(DependencyType.Grpc, checker.Type);
    }

    [Fact]
    public void PostgresChecker_HasCorrectType()
    {
        var checker = new PostgresChecker();
        Assert.Equal(DependencyType.Postgres, checker.Type);
    }

    [Fact]
    public void PostgresChecker_WithConnectionString()
    {
        var checker = new PostgresChecker("Host=db;Port=5432");
        Assert.Equal(DependencyType.Postgres, checker.Type);
    }

    [Fact]
    public void MySqlChecker_HasCorrectType()
    {
        var checker = new MySqlChecker();
        Assert.Equal(DependencyType.MySql, checker.Type);
    }

    [Fact]
    public void RedisChecker_HasCorrectType()
    {
        var checker = new RedisChecker();
        Assert.Equal(DependencyType.Redis, checker.Type);
    }

    [Fact]
    public void AmqpChecker_HasCorrectType()
    {
        var checker = new AmqpChecker();
        Assert.Equal(DependencyType.Amqp, checker.Type);
    }

    [Fact]
    public void AmqpChecker_WithCredentials()
    {
        var checker = new AmqpChecker(username: "guest", password: "guest", vhost: "/");
        Assert.Equal(DependencyType.Amqp, checker.Type);
    }

    [Fact]
    public void KafkaChecker_HasCorrectType()
    {
        var checker = new KafkaChecker();
        Assert.Equal(DependencyType.Kafka, checker.Type);
    }

    [Fact]
    public void LdapChecker_HasCorrectType()
    {
        var checker = new LdapChecker();
        Assert.Equal(DependencyType.Ldap, checker.Type);
    }
}
