namespace DepHealth.Core.Tests;

public class DependencyTypeTests
{
    [Theory]
    [InlineData(DependencyType.Http, "http")]
    [InlineData(DependencyType.Grpc, "grpc")]
    [InlineData(DependencyType.Tcp, "tcp")]
    [InlineData(DependencyType.Postgres, "postgres")]
    [InlineData(DependencyType.MySql, "mysql")]
    [InlineData(DependencyType.Redis, "redis")]
    [InlineData(DependencyType.Amqp, "amqp")]
    [InlineData(DependencyType.Kafka, "kafka")]
    [InlineData(DependencyType.Ldap, "ldap")]
    public void Label_ReturnsCorrectString(DependencyType type, string expected)
    {
        Assert.Equal(expected, type.Label());
    }

    [Theory]
    [InlineData("http", DependencyType.Http)]
    [InlineData("HTTP", DependencyType.Http)]
    [InlineData("postgres", DependencyType.Postgres)]
    [InlineData("KAFKA", DependencyType.Kafka)]
    [InlineData("ldap", DependencyType.Ldap)]
    [InlineData("LDAP", DependencyType.Ldap)]
    public void FromLabel_CaseInsensitive(string label, DependencyType expected)
    {
        Assert.Equal(expected, DependencyTypeExtensions.FromLabel(label));
    }

    [Fact]
    public void FromLabel_UnknownThrows()
    {
        Assert.Throws<ArgumentException>(() => DependencyTypeExtensions.FromLabel("unknown"));
    }
}
