using System.Net;
using DepHealth.Checks;
using DepHealth.Exceptions;

namespace DepHealth.Core.Tests.Checks;

// ---------------------------------------------------------------------------
// HTTP auth validation
// ---------------------------------------------------------------------------

public class HttpAuthValidationTests
{
    [Fact]
    public void NoAuth_NoError()
    {
        HttpChecker.ValidateAuthConflicts(null, null, null, null);
    }

    [Fact]
    public void SingleBearer_Ok()
    {
        HttpChecker.ValidateAuthConflicts(null, "token", null, null);
    }

    [Fact]
    public void SingleBasicAuth_Ok()
    {
        HttpChecker.ValidateAuthConflicts(null, null, "user", "pass");
    }

    [Fact]
    public void SingleAuthorizationHeader_Ok()
    {
        var headers = new Dictionary<string, string> { ["Authorization"] = "Custom" };
        HttpChecker.ValidateAuthConflicts(headers, null, null, null);
    }

    [Fact]
    public void BearerWithNonAuthHeader_Ok()
    {
        var headers = new Dictionary<string, string> { ["X-Custom"] = "value" };
        HttpChecker.ValidateAuthConflicts(headers, "token", null, null);
    }

    [Fact]
    public void ConflictBearerAndBasicAuth()
    {
        var ex = Assert.Throws<ValidationException>(() =>
            HttpChecker.ValidateAuthConflicts(null, "token", "user", "pass"));
        Assert.Contains("conflicting auth methods", ex.Message);
    }

    [Fact]
    public void ConflictBearerAndAuthorizationHeader()
    {
        var headers = new Dictionary<string, string> { ["Authorization"] = "Custom" };
        var ex = Assert.Throws<ValidationException>(() =>
            HttpChecker.ValidateAuthConflicts(headers, "token", null, null));
        Assert.Contains("conflicting auth methods", ex.Message);
    }

    [Fact]
    public void ConflictBasicAuthAndAuthorizationHeader()
    {
        var headers = new Dictionary<string, string> { ["Authorization"] = "Custom" };
        var ex = Assert.Throws<ValidationException>(() =>
            HttpChecker.ValidateAuthConflicts(headers, null, "user", "pass"));
        Assert.Contains("conflicting auth methods", ex.Message);
    }

    [Fact]
    public void AuthorizationCaseInsensitive()
    {
        var headers = new Dictionary<string, string> { ["authorization"] = "Custom" };
        var ex = Assert.Throws<ValidationException>(() =>
            HttpChecker.ValidateAuthConflicts(headers, "token", null, null));
        Assert.Contains("conflicting auth methods", ex.Message);
    }
}

public class HttpCheckerConstructorTests
{
    [Fact]
    public void BearerToken_CreatesChecker()
    {
        var checker = new HttpChecker(bearerToken: "my-token");
        Assert.Equal(DependencyType.Http, checker.Type);
    }

    [Fact]
    public void BasicAuth_CreatesChecker()
    {
        var checker = new HttpChecker(basicAuthUsername: "admin", basicAuthPassword: "password");
        Assert.Equal(DependencyType.Http, checker.Type);
    }

    [Fact]
    public void CustomHeaders_CreatesChecker()
    {
        var checker = new HttpChecker(headers: new Dictionary<string, string> { ["X-API-Key"] = "key" });
        Assert.Equal(DependencyType.Http, checker.Type);
    }

    [Fact]
    public void NoAuth_CreatesChecker()
    {
        var checker = new HttpChecker();
        Assert.Equal(DependencyType.Http, checker.Type);
    }

    [Fact]
    public void ConflictRaisesValidationException()
    {
        Assert.Throws<ValidationException>(() =>
            new HttpChecker(bearerToken: "token", basicAuthUsername: "u", basicAuthPassword: "p"));
    }
}

// ---------------------------------------------------------------------------
// gRPC auth validation
// ---------------------------------------------------------------------------

public class GrpcAuthValidationTests
{
    [Fact]
    public void NoAuth_NoError()
    {
        GrpcChecker.ValidateAuthConflicts(null, null, null, null);
    }

    [Fact]
    public void SingleBearer_Ok()
    {
        GrpcChecker.ValidateAuthConflicts(null, "token", null, null);
    }

    [Fact]
    public void SingleBasicAuth_Ok()
    {
        GrpcChecker.ValidateAuthConflicts(null, null, "user", "pass");
    }

    [Fact]
    public void SingleAuthorizationMetadata_Ok()
    {
        var metadata = new Dictionary<string, string> { ["authorization"] = "Custom" };
        GrpcChecker.ValidateAuthConflicts(metadata, null, null, null);
    }

    [Fact]
    public void BearerWithNonAuthMetadata_Ok()
    {
        var metadata = new Dictionary<string, string> { ["x-custom"] = "value" };
        GrpcChecker.ValidateAuthConflicts(metadata, "token", null, null);
    }

    [Fact]
    public void ConflictBearerAndBasicAuth()
    {
        var ex = Assert.Throws<ValidationException>(() =>
            GrpcChecker.ValidateAuthConflicts(null, "token", "user", "pass"));
        Assert.Contains("conflicting auth methods", ex.Message);
    }

    [Fact]
    public void ConflictBearerAndAuthorizationMetadata()
    {
        var metadata = new Dictionary<string, string> { ["authorization"] = "Custom" };
        var ex = Assert.Throws<ValidationException>(() =>
            GrpcChecker.ValidateAuthConflicts(metadata, "token", null, null));
        Assert.Contains("conflicting auth methods", ex.Message);
    }

    [Fact]
    public void ConflictBasicAuthAndAuthorizationMetadata()
    {
        var metadata = new Dictionary<string, string> { ["authorization"] = "Custom" };
        var ex = Assert.Throws<ValidationException>(() =>
            GrpcChecker.ValidateAuthConflicts(metadata, null, "user", "pass"));
        Assert.Contains("conflicting auth methods", ex.Message);
    }

    [Fact]
    public void AuthorizationCaseInsensitive()
    {
        var metadata = new Dictionary<string, string> { ["Authorization"] = "Custom" };
        var ex = Assert.Throws<ValidationException>(() =>
            GrpcChecker.ValidateAuthConflicts(metadata, "token", null, null));
        Assert.Contains("conflicting auth methods", ex.Message);
    }
}

public class GrpcCheckerConstructorTests
{
    [Fact]
    public void BearerToken_CreatesChecker()
    {
        var checker = new GrpcChecker(bearerToken: "my-token");
        Assert.Equal(DependencyType.Grpc, checker.Type);
    }

    [Fact]
    public void BasicAuth_CreatesChecker()
    {
        var checker = new GrpcChecker(basicAuthUsername: "admin", basicAuthPassword: "password");
        Assert.Equal(DependencyType.Grpc, checker.Type);
    }

    [Fact]
    public void CustomMetadata_CreatesChecker()
    {
        var checker = new GrpcChecker(metadata: new Dictionary<string, string> { ["x-api-key"] = "key" });
        Assert.Equal(DependencyType.Grpc, checker.Type);
    }

    [Fact]
    public void NoAuth_CreatesChecker()
    {
        var checker = new GrpcChecker();
        Assert.Equal(DependencyType.Grpc, checker.Type);
    }

    [Fact]
    public void ConflictRaisesValidationException()
    {
        Assert.Throws<ValidationException>(() =>
            new GrpcChecker(bearerToken: "token", basicAuthUsername: "u", basicAuthPassword: "p"));
    }
}

// ---------------------------------------------------------------------------
// HTTP auth integration tests (with real HTTP server)
// ---------------------------------------------------------------------------

public class HttpAuthIntegrationTests : IDisposable
{
    private HttpListener? _listener;

    public void Dispose()
    {
        _listener?.Close();
        GC.SuppressFinalize(this);
    }

    private (HttpListener Listener, int Port) StartServer(Action<HttpListenerContext> handler)
    {
        // Find an available port by binding to port 0.
        var tcpListener = new System.Net.Sockets.TcpListener(IPAddress.Loopback, 0);
        tcpListener.Start();
        var port = ((IPEndPoint)tcpListener.LocalEndpoint).Port;
        tcpListener.Stop();

        var listener = new HttpListener();
        listener.Prefixes.Add($"http://127.0.0.1:{port}/");
        listener.Start();
        _listener = listener;

        // Handle one request in background.
        _ = Task.Run(() =>
        {
            try
            {
                var ctx = listener.GetContext();
                handler(ctx);
                ctx.Response.Close();
            }
            catch (HttpListenerException)
            {
                // Listener closed.
            }
        });

        return (listener, port);
    }

    [Fact]
    public async Task BearerToken_Success()
    {
        var (_, port) = StartServer(ctx =>
        {
            var auth = ctx.Request.Headers["Authorization"];
            ctx.Response.StatusCode = auth == "Bearer test-token" ? 200 : 401;
        });

        var checker = new HttpChecker(bearerToken: "test-token");
        var ep = new Endpoint("127.0.0.1", port.ToString());
        await checker.CheckAsync(ep, CancellationToken.None);
    }

    [Fact]
    public async Task BasicAuth_Success()
    {
        var expectedCred = Convert.ToBase64String(
            System.Text.Encoding.UTF8.GetBytes("admin:password"));

        var (_, port) = StartServer(ctx =>
        {
            var auth = ctx.Request.Headers["Authorization"];
            ctx.Response.StatusCode = auth == $"Basic {expectedCred}" ? 200 : 401;
        });

        var checker = new HttpChecker(basicAuthUsername: "admin", basicAuthPassword: "password");
        var ep = new Endpoint("127.0.0.1", port.ToString());
        await checker.CheckAsync(ep, CancellationToken.None);
    }

    [Fact]
    public async Task CustomHeaders_Success()
    {
        var (_, port) = StartServer(ctx =>
        {
            var apiKey = ctx.Request.Headers["X-API-Key"];
            ctx.Response.StatusCode = apiKey == "my-key" ? 200 : 403;
        });

        var checker = new HttpChecker(headers: new Dictionary<string, string> { ["X-API-Key"] = "my-key" });
        var ep = new Endpoint("127.0.0.1", port.ToString());
        await checker.CheckAsync(ep, CancellationToken.None);
    }

    [Fact]
    public async Task Http401_RaisesCheckAuthException()
    {
        var (_, port) = StartServer(ctx =>
        {
            ctx.Response.StatusCode = 401;
        });

        var checker = new HttpChecker();
        var ep = new Endpoint("127.0.0.1", port.ToString());
        var ex = await Assert.ThrowsAsync<CheckAuthException>(
            () => checker.CheckAsync(ep, CancellationToken.None));
        Assert.Equal("auth_error", ex.ExceptionStatusCategory);
        Assert.Equal("auth_error", ex.ExceptionStatusDetail);
    }

    [Fact]
    public async Task Http403_RaisesCheckAuthException()
    {
        var (_, port) = StartServer(ctx =>
        {
            ctx.Response.StatusCode = 403;
        });

        var checker = new HttpChecker();
        var ep = new Endpoint("127.0.0.1", port.ToString());
        var ex = await Assert.ThrowsAsync<CheckAuthException>(
            () => checker.CheckAsync(ep, CancellationToken.None));
        Assert.Equal("auth_error", ex.ExceptionStatusCategory);
    }

    [Fact]
    public async Task Http500_RaisesUnhealthyNotAuth()
    {
        var (_, port) = StartServer(ctx =>
        {
            ctx.Response.StatusCode = 500;
        });

        var checker = new HttpChecker();
        var ep = new Endpoint("127.0.0.1", port.ToString());
        var ex = await Assert.ThrowsAsync<UnhealthyException>(
            () => checker.CheckAsync(ep, CancellationToken.None));
        Assert.Equal("unhealthy", ex.ExceptionStatusCategory);
        Assert.Equal("http_500", ex.ExceptionStatusDetail);
    }
}
