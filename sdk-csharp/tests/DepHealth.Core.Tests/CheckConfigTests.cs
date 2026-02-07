namespace DepHealth.Core.Tests;

public class CheckConfigTests
{
    [Fact]
    public void Defaults_HasCorrectValues()
    {
        var config = CheckConfig.Defaults();
        Assert.Equal(TimeSpan.FromSeconds(15), config.Interval);
        Assert.Equal(TimeSpan.FromSeconds(5), config.Timeout);
        Assert.Equal(TimeSpan.FromSeconds(5), config.InitialDelay);
        Assert.Equal(1, config.FailureThreshold);
        Assert.Equal(1, config.SuccessThreshold);
    }

    [Fact]
    public void Builder_CustomValues()
    {
        var config = CheckConfig.CreateBuilder()
            .WithInterval(TimeSpan.FromSeconds(10))
            .WithTimeout(TimeSpan.FromSeconds(3))
            .WithInitialDelay(TimeSpan.FromSeconds(1))
            .WithFailureThreshold(3)
            .WithSuccessThreshold(2)
            .Build();

        Assert.Equal(TimeSpan.FromSeconds(10), config.Interval);
        Assert.Equal(TimeSpan.FromSeconds(3), config.Timeout);
        Assert.Equal(TimeSpan.FromSeconds(1), config.InitialDelay);
        Assert.Equal(3, config.FailureThreshold);
        Assert.Equal(2, config.SuccessThreshold);
    }

    [Fact]
    public void Builder_IntervalTooSmall_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            CheckConfig.CreateBuilder()
                .WithInterval(TimeSpan.FromMilliseconds(500))
                .Build());
    }

    [Fact]
    public void Builder_TimeoutGreaterThanInterval_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            CheckConfig.CreateBuilder()
                .WithInterval(TimeSpan.FromSeconds(5))
                .WithTimeout(TimeSpan.FromSeconds(5))
                .Build());
    }

    [Fact]
    public void Builder_FailureThresholdOutOfRange_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            CheckConfig.CreateBuilder()
                .WithFailureThreshold(0)
                .Build());
    }

    [Fact]
    public void Builder_SuccessThresholdOutOfRange_Throws()
    {
        Assert.Throws<ValidationException>(() =>
            CheckConfig.CreateBuilder()
                .WithSuccessThreshold(11)
                .Build());
    }
}
