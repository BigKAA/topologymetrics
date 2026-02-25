namespace DepHealth.Core.Tests;

public class ConfigParserTests
{
    // --- ParseUrl ---

    [Theory]
    [InlineData("postgres://localhost:5432/mydb", "localhost", "5432", DependencyType.Postgres)]
    [InlineData("postgresql://db.host:5433/mydb", "db.host", "5433", DependencyType.Postgres)]
    [InlineData("mysql://db:3306/test", "db", "3306", DependencyType.MySql)]
    [InlineData("redis://cache:6379", "cache", "6379", DependencyType.Redis)]
    [InlineData("rediss://cache:6380", "cache", "6380", DependencyType.Redis)]
    [InlineData("amqp://broker:5672/", "broker", "5672", DependencyType.Amqp)]
    [InlineData("amqps://broker:5671/", "broker", "5671", DependencyType.Amqp)]
    [InlineData("http://api.svc:8080/health", "api.svc", "8080", DependencyType.Http)]
    [InlineData("https://secure.api:443/health", "secure.api", "443", DependencyType.Http)]
    [InlineData("grpc://grpc-svc:50051", "grpc-svc", "50051", DependencyType.Grpc)]
    [InlineData("ldap://ldap.host:389", "ldap.host", "389", DependencyType.Ldap)]
    [InlineData("ldaps://ldap.host:636", "ldap.host", "636", DependencyType.Ldap)]
    public void ParseUrl_ValidUrls(string url, string expectedHost, string expectedPort, DependencyType expectedType)
    {
        var result = ConfigParser.ParseUrl(url);
        Assert.Single(result);
        Assert.Equal(expectedHost, result[0].Host);
        Assert.Equal(expectedPort, result[0].Port);
        Assert.Equal(expectedType, result[0].Type);
    }

    [Fact]
    public void ParseUrl_WithUserInfo()
    {
        var result = ConfigParser.ParseUrl("postgres://user:pass@db.host:5432/mydb");
        Assert.Single(result);
        Assert.Equal("db.host", result[0].Host);
        Assert.Equal("5432", result[0].Port);
    }

    [Fact]
    public void ParseUrl_DefaultPort()
    {
        var result = ConfigParser.ParseUrl("http://api.svc/health");
        Assert.Single(result);
        Assert.Equal("80", result[0].Port);
    }

    [Fact]
    public void ParseUrl_LdapDefaultPort()
    {
        var result = ConfigParser.ParseUrl("ldap://ldap.host");
        Assert.Single(result);
        Assert.Equal("389", result[0].Port);
    }

    [Fact]
    public void ParseUrl_LdapsDefaultPort()
    {
        var result = ConfigParser.ParseUrl("ldaps://ldap.host");
        Assert.Single(result);
        Assert.Equal("636", result[0].Port);
    }

    [Fact]
    public void ParseUrl_IPv6()
    {
        var result = ConfigParser.ParseUrl("postgres://[::1]:5432/mydb");
        Assert.Single(result);
        Assert.Equal("::1", result[0].Host);
        Assert.Equal("5432", result[0].Port);
    }

    [Fact]
    public void ParseUrl_KafkaMultiHost()
    {
        var result = ConfigParser.ParseUrl("kafka://broker1:9092,broker2:9093,broker3:9094");
        Assert.Equal(3, result.Count);
        Assert.Equal("broker1", result[0].Host);
        Assert.Equal("9092", result[0].Port);
        Assert.Equal("broker2", result[1].Host);
        Assert.Equal("9093", result[1].Port);
        Assert.Equal("broker3", result[2].Host);
        Assert.Equal("9094", result[2].Port);
    }

    [Fact]
    public void ParseUrl_Empty_Throws()
    {
        Assert.Throws<ConfigurationException>(() => ConfigParser.ParseUrl(""));
    }

    [Fact]
    public void ParseUrl_NoScheme_Throws()
    {
        Assert.Throws<ConfigurationException>(() => ConfigParser.ParseUrl("localhost:5432"));
    }

    [Fact]
    public void ParseUrl_UnsupportedScheme_Throws()
    {
        Assert.Throws<ConfigurationException>(() => ConfigParser.ParseUrl("ftp://host:21"));
    }

    // --- ParseJdbc ---

    [Theory]
    [InlineData("jdbc:postgresql://db:5432/mydb", "db", "5432", DependencyType.Postgres)]
    [InlineData("jdbc:mysql://db:3306/mydb", "db", "3306", DependencyType.MySql)]
    public void ParseJdbc_ValidUrls(string url, string expectedHost, string expectedPort, DependencyType expectedType)
    {
        var result = ConfigParser.ParseJdbc(url);
        Assert.Single(result);
        Assert.Equal(expectedHost, result[0].Host);
        Assert.Equal(expectedPort, result[0].Port);
        Assert.Equal(expectedType, result[0].Type);
    }

    [Fact]
    public void ParseJdbc_Empty_Throws()
    {
        Assert.Throws<ConfigurationException>(() => ConfigParser.ParseJdbc(""));
    }

    [Fact]
    public void ParseJdbc_NoJdbcPrefix_Throws()
    {
        Assert.Throws<ConfigurationException>(() => ConfigParser.ParseJdbc("postgresql://db:5432"));
    }

    // --- ParseConnectionString ---

    [Fact]
    public void ParseConnectionString_StandardFormat()
    {
        var ep = ConfigParser.ParseConnectionString("Host=db.local;Port=5432;Database=mydb");
        Assert.Equal("db.local", ep.Host);
        Assert.Equal("5432", ep.Port);
    }

    [Fact]
    public void ParseConnectionString_ServerKeyword()
    {
        var ep = ConfigParser.ParseConnectionString("Server=db.local;Port=3306;Database=mydb");
        Assert.Equal("db.local", ep.Host);
        Assert.Equal("3306", ep.Port);
    }

    [Fact]
    public void ParseConnectionString_HostWithPort()
    {
        var ep = ConfigParser.ParseConnectionString("Host=db.local:5432;Database=mydb");
        Assert.Equal("db.local", ep.Host);
        Assert.Equal("5432", ep.Port);
    }

    [Fact]
    public void ParseConnectionString_SqlServerFormat()
    {
        var ep = ConfigParser.ParseConnectionString("Server=db.local,1433;Database=mydb");
        Assert.Equal("db.local", ep.Host);
        Assert.Equal("1433", ep.Port);
    }

    [Fact]
    public void ParseConnectionString_Empty_Throws()
    {
        Assert.Throws<ConfigurationException>(() => ConfigParser.ParseConnectionString(""));
    }

    [Fact]
    public void ParseConnectionString_NoHost_Throws()
    {
        Assert.Throws<ConfigurationException>(() =>
            ConfigParser.ParseConnectionString("Port=5432;Database=mydb"));
    }

    // --- ParseParams ---

    [Fact]
    public void ParseParams_ValidHostPort()
    {
        var ep = ConfigParser.ParseParams("db.local", "5432");
        Assert.Equal("db.local", ep.Host);
        Assert.Equal("5432", ep.Port);
    }

    [Fact]
    public void ParseParams_IPv6WithBrackets()
    {
        var ep = ConfigParser.ParseParams("[::1]", "5432");
        Assert.Equal("::1", ep.Host);
        Assert.Equal("5432", ep.Port);
    }

    [Fact]
    public void ParseParams_EmptyHost_Throws()
    {
        Assert.Throws<ConfigurationException>(() => ConfigParser.ParseParams("", "5432"));
    }

    [Fact]
    public void ParseParams_InvalidPort_Throws()
    {
        Assert.Throws<ConfigurationException>(() => ConfigParser.ParseParams("host", "abc"));
    }

    [Fact]
    public void ParseParams_PortOutOfRange_Throws()
    {
        Assert.Throws<ConfigurationException>(() => ConfigParser.ParseParams("host", "70000"));
    }
}
