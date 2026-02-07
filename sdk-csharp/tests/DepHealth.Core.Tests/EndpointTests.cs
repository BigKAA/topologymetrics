namespace DepHealth.Core.Tests;

public class EndpointTests
{
    [Fact]
    public void Constructor_SetsProperties()
    {
        var ep = new Endpoint("localhost", "5432");
        Assert.Equal("localhost", ep.Host);
        Assert.Equal("5432", ep.Port);
        Assert.Empty(ep.Metadata);
    }

    [Fact]
    public void PortAsInt_ReturnsInt()
    {
        var ep = new Endpoint("localhost", "5432");
        Assert.Equal(5432, ep.PortAsInt());
    }

    [Fact]
    public void Equals_SameHostPort()
    {
        var ep1 = new Endpoint("localhost", "5432");
        var ep2 = new Endpoint("localhost", "5432");
        Assert.Equal(ep1, ep2);
        Assert.Equal(ep1.GetHashCode(), ep2.GetHashCode());
    }

    [Fact]
    public void Equals_DifferentPort()
    {
        var ep1 = new Endpoint("localhost", "5432");
        var ep2 = new Endpoint("localhost", "3306");
        Assert.NotEqual(ep1, ep2);
    }

    [Fact]
    public void ToString_Format()
    {
        var ep = new Endpoint("db.example.com", "5432");
        Assert.Equal("db.example.com:5432", ep.ToString());
    }

    [Fact]
    public void Metadata_IsReadOnly()
    {
        var meta = new Dictionary<string, string> { ["vhost"] = "/" };
        var ep = new Endpoint("localhost", "5672", meta);
        Assert.Equal("/", ep.Metadata["vhost"]);
    }

    [Fact]
    public void Constructor_NullHost_Throws()
    {
        Assert.Throws<ArgumentNullException>(() => new Endpoint(null!, "5432"));
    }
}
