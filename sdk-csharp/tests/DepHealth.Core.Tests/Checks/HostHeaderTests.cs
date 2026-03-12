using System.Net;
using System.Net.Sockets;
using System.Text;
using DepHealth.Checks;

namespace DepHealth.Core.Tests.Checks;

/// <summary>
/// Tests for HTTP hostHeader and gRPC authority override features.
/// Uses raw TcpListener to accept requests with any Host header
/// (HttpListener rejects requests whose Host doesn't match its prefix).
/// </summary>
public class HttpHostHeaderTests
{
    /// <summary>
    /// Starts a minimal HTTP/1.1 server on a random port, returns parsed headers
    /// from the first incoming request and sends a 200 OK response.
    /// </summary>
    private static (Task<Dictionary<string, string>> Headers, int Port) StartRawHttpServer()
    {
        var listener = new TcpListener(IPAddress.Loopback, 0);
        listener.Start();
        var port = ((IPEndPoint)listener.LocalEndpoint).Port;

        var headersTask = Task.Run(async () =>
        {
            using var client = await listener.AcceptTcpClientAsync();
            listener.Stop();
            await using var stream = client.GetStream();
            using var reader = new StreamReader(stream, Encoding.ASCII, leaveOpen: true);

            // Read request line.
            await reader.ReadLineAsync();

            // Read headers until blank line.
            var headers = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
            while (await reader.ReadLineAsync() is { Length: > 0 } line)
            {
                var colon = line.IndexOf(':');
                if (colon > 0)
                {
                    headers[line[..colon].Trim()] = line[(colon + 1)..].Trim();
                }
            }

            // Send 200 OK response.
            var response = Encoding.ASCII.GetBytes("HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: close\r\n\r\n");
            await stream.WriteAsync(response);

            return headers;
        });

        return (headersTask, port);
    }

    [Fact]
    public async Task HostHeader_SentInRequest()
    {
        var (headersTask, port) = StartRawHttpServer();

        var checker = new HttpChecker(hostHeader: "payment.example.com");
        var ep = new Endpoint("127.0.0.1", port.ToString());
        await checker.CheckAsync(ep, CancellationToken.None);

        var headers = await headersTask;
        Assert.Equal("payment.example.com", headers["Host"]);
    }

    [Fact]
    public async Task HostHeader_WithCustomHeaders()
    {
        var (headersTask, port) = StartRawHttpServer();

        var checker = new HttpChecker(
            headers: new Dictionary<string, string> { ["X-API-Key"] = "key" },
            hostHeader: "payment.example.com");
        var ep = new Endpoint("127.0.0.1", port.ToString());
        await checker.CheckAsync(ep, CancellationToken.None);

        var headers = await headersTask;
        Assert.Equal("payment.example.com", headers["Host"]);
        Assert.Equal("key", headers["X-API-Key"]);
    }

    [Fact]
    public async Task HostHeader_WithBearerToken()
    {
        var (headersTask, port) = StartRawHttpServer();

        var checker = new HttpChecker(
            bearerToken: "my-token",
            hostHeader: "payment.example.com");
        var ep = new Endpoint("127.0.0.1", port.ToString());
        await checker.CheckAsync(ep, CancellationToken.None);

        var headers = await headersTask;
        Assert.Equal("payment.example.com", headers["Host"]);
        Assert.Equal("Bearer my-token", headers["Authorization"]);
    }

    [Fact]
    public async Task WithoutHostHeader_DefaultHostUsed()
    {
        var (headersTask, port) = StartRawHttpServer();

        var checker = new HttpChecker();
        var ep = new Endpoint("127.0.0.1", port.ToString());
        await checker.CheckAsync(ep, CancellationToken.None);

        var headers = await headersTask;
        Assert.Equal($"127.0.0.1:{port}", headers["Host"]);
    }
}

/// <summary>
/// Validation tests for hostHeader / grpcAuthority conflict detection.
/// </summary>
public class HostHeaderValidationTests
{
    [Fact]
    public void HostHeaderConflict_ThrowsValidationException()
    {
        var headers = new Dictionary<string, string> { ["Host"] = "other.com" };
        var ex = Assert.Throws<ValidationException>(() =>
            new HttpChecker(headers: headers, hostHeader: "payment.example.com"));
        Assert.Contains("conflicting Host header", ex.Message);
        Assert.Contains("hostHeader and headers both set Host", ex.Message);
    }

    [Fact]
    public void HostHeaderConflict_CaseInsensitive()
    {
        var headers = new Dictionary<string, string> { ["host"] = "other.com" };
        var ex = Assert.Throws<ValidationException>(() =>
            new HttpChecker(headers: headers, hostHeader: "payment.example.com"));
        Assert.Contains("conflicting Host header", ex.Message);
    }

    [Fact]
    public void HostHeader_NoConflictWhenAlone()
    {
        var checker = new HttpChecker(hostHeader: "payment.example.com");
        Assert.Equal(DependencyType.Http, checker.Type);
    }

    [Fact]
    public void HostHeader_NoConflictWithOtherHeaders()
    {
        var headers = new Dictionary<string, string> { ["X-Custom"] = "value" };
        var checker = new HttpChecker(headers: headers, hostHeader: "payment.example.com");
        Assert.Equal(DependencyType.Http, checker.Type);
    }
}

public class GrpcAuthorityValidationTests
{
    [Fact]
    public void AuthorityConflict_ThrowsValidationException()
    {
        var metadata = new Dictionary<string, string> { [":authority"] = "other.com" };
        var ex = Assert.Throws<ValidationException>(() =>
            new GrpcChecker(metadata: metadata, authority: "payment.example.com"));
        Assert.Contains("conflicting authority", ex.Message);
        Assert.Contains("grpcAuthority and metadata both set :authority", ex.Message);
    }

    [Fact]
    public void Authority_NoConflictWhenAlone()
    {
        var checker = new GrpcChecker(authority: "payment.example.com");
        Assert.Equal(DependencyType.Grpc, checker.Type);
    }

    [Fact]
    public void Authority_NoConflictWithOtherMetadata()
    {
        var metadata = new Dictionary<string, string> { ["x-custom"] = "value" };
        var checker = new GrpcChecker(metadata: metadata, authority: "payment.example.com");
        Assert.Equal(DependencyType.Grpc, checker.Type);
    }

    [Fact]
    public void Authority_WithBearerToken_NoConflict()
    {
        var checker = new GrpcChecker(bearerToken: "token", authority: "payment.example.com");
        Assert.Equal(DependencyType.Grpc, checker.Type);
    }
}
