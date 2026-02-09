namespace DepHealth.Core.Tests;

public class EndpointTests
{
    [Fact]
    public void Constructor_SetsProperties()
    {
        var ep = new Endpoint("localhost", "5432");
        Assert.Equal("localhost", ep.Host);
        Assert.Equal("5432", ep.Port);
        Assert.Empty(ep.Labels);
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
    public void Labels_IsReadOnly()
    {
        var labels = new Dictionary<string, string> { ["region"] = "eu" };
        var ep = new Endpoint("localhost", "5672", labels);
        Assert.Equal("eu", ep.Labels["region"]);
    }

    [Fact]
    public void Constructor_NullHost_Throws()
    {
        Assert.Throws<ArgumentNullException>(() => new Endpoint(null!, "5432"));
    }

    [Fact]
    public void ValidateLabelName_ValidName()
    {
        Endpoint.ValidateLabelName("region");
        Endpoint.ValidateLabelName("_private");
        Endpoint.ValidateLabelName("my_label_123");
    }

    [Fact]
    public void ValidateLabelName_InvalidPattern_Throws()
    {
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabelName("123abc"));
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabelName("my-label"));
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabelName(""));
    }

    [Fact]
    public void ValidateLabelName_Reserved_Throws()
    {
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabelName("name"));
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabelName("dependency"));
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabelName("type"));
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabelName("host"));
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabelName("port"));
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabelName("critical"));
    }

    [Fact]
    public void ValidateLabels_ValidLabels()
    {
        var labels = new Dictionary<string, string>
        {
            ["region"] = "eu",
            ["shard"] = "1"
        };
        Endpoint.ValidateLabels(labels);
    }

    [Fact]
    public void ValidateLabels_ReservedKey_Throws()
    {
        var labels = new Dictionary<string, string> { ["host"] = "bad" };
        Assert.Throws<ValidationException>(() => Endpoint.ValidateLabels(labels));
    }
}
