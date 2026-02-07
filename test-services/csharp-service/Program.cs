using DepHealth;
using DepHealth.AspNetCore;
using Prometheus;

var builder = WebApplication.CreateBuilder(args);

// --- Конфигурация из переменных окружения ---
var postgresUrl = Environment.GetEnvironmentVariable("POSTGRES_URL")
    ?? "postgres://dephealth:dephealth-test-pass@postgres:5432/dephealth?sslmode=disable";
var redisUrl = Environment.GetEnvironmentVariable("REDIS_URL")
    ?? "redis://redis:6379/0";
var httpServiceUrl = Environment.GetEnvironmentVariable("HTTP_SERVICE_URL")
    ?? "http://http-stub:8080";
var httpHealthPath = Environment.GetEnvironmentVariable("HTTP_HEALTH_PATH")
    ?? "/health";
var grpcHost = Environment.GetEnvironmentVariable("GRPC_SERVICE_HOST")
    ?? "grpc-stub";
var grpcPort = Environment.GetEnvironmentVariable("GRPC_SERVICE_PORT")
    ?? "9090";
var intervalStr = Environment.GetEnvironmentVariable("DEPHEALTH_INTERVAL")
    ?? "10";

var checkInterval = TimeSpan.FromSeconds(int.Parse(intervalStr));

// --- Регистрация DepHealth ---
builder.Services.AddDepHealth(dh =>
{
    dh.AddPostgres("postgres-main", postgresUrl, critical: true);
    dh.AddRedis("redis-cache", redisUrl);
    dh.AddHttp("http-service", httpServiceUrl, healthPath: httpHealthPath);
    dh.AddGrpc("grpc-service", grpcHost, grpcPort);
    dh.WithCheckInterval(checkInterval);
});

var app = builder.Build();

// --- Endpoints ---

app.MapGet("/", () => Results.Json(new
{
    service = "dephealth-test-csharp",
    version = "0.1.0",
    language = "csharp"
}));

app.MapGet("/health", () => Results.Text("OK"));

app.MapDepHealthEndpoints();

// Prometheus /metrics
app.MapMetrics();

app.Run();
