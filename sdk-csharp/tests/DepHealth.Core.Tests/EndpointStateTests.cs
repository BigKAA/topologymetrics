namespace DepHealth.Core.Tests;

public class EndpointStateTests
{
    [Fact]
    public void InitialState_IsNull()
    {
        var state = new EndpointState();
        Assert.Null(state.Healthy);
    }

    [Fact]
    public void FirstSuccess_SetsHealthy()
    {
        var state = new EndpointState();
        state.RecordSuccess(1);
        Assert.True(state.Healthy);
    }

    [Fact]
    public void FirstFailure_SetsUnhealthy()
    {
        var state = new EndpointState();
        state.RecordFailure(1);
        Assert.False(state.Healthy);
    }

    [Fact]
    public void FailureThreshold_RequiresConsecutiveFailures()
    {
        var state = new EndpointState();
        state.RecordSuccess(1);  // healthy = true
        Assert.True(state.Healthy);

        state.RecordFailure(3);  // 1/3
        Assert.True(state.Healthy);  // still healthy

        state.RecordFailure(3);  // 2/3
        Assert.True(state.Healthy);

        state.RecordFailure(3);  // 3/3 → unhealthy
        Assert.False(state.Healthy);
    }

    [Fact]
    public void SuccessThreshold_RequiresConsecutiveSuccesses()
    {
        var state = new EndpointState();
        state.RecordFailure(1);  // unhealthy
        Assert.False(state.Healthy);

        state.RecordSuccess(2);  // 1/2
        Assert.False(state.Healthy);

        state.RecordSuccess(2);  // 2/2 → healthy
        Assert.True(state.Healthy);
    }

    [Fact]
    public void IntermittentSuccess_ResetsFailureCounter()
    {
        var state = new EndpointState();
        state.RecordSuccess(1);  // healthy
        state.RecordFailure(3);  // 1/3
        state.RecordFailure(3);  // 2/3
        state.RecordSuccess(1);  // resets failures
        state.RecordFailure(3);  // 1/3 again
        Assert.True(state.Healthy);  // still healthy
    }
}
