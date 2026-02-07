using DepHealth;
using DepHealth.AspNetCore;
using Prometheus;

var builder = WebApplication.CreateBuilder(args);

// --- Конфигурация из переменных окружения ---
var primaryDbUrl = Environment.GetEnvironmentVariable("PRIMARY_DATABASE_URL")
    ?? "postgres://dephealth:dephealth-test-pass@postgres-primary.dephealth-conformance.svc:5432/dephealth?sslmode=disable";
var replicaDbUrl = Environment.GetEnvironmentVariable("REPLICA_DATABASE_URL")
    ?? "postgres://dephealth:dephealth-test-pass@postgres-replica.dephealth-conformance.svc:5432/dephealth?sslmode=disable";
var redisUrl = Environment.GetEnvironmentVariable("REDIS_URL")
    ?? "redis://redis.dephealth-conformance.svc:6379/0";
var rabbitmqUrl = Environment.GetEnvironmentVariable("RABBITMQ_URL")
    ?? "amqp://dephealth:dephealth-test-pass@rabbitmq.dephealth-conformance.svc:5672/";
var kafkaHost = Environment.GetEnvironmentVariable("KAFKA_HOST")
    ?? "kafka.dephealth-conformance.svc";
var kafkaPort = Environment.GetEnvironmentVariable("KAFKA_PORT")
    ?? "9092";
var httpStubUrl = Environment.GetEnvironmentVariable("HTTP_STUB_URL")
    ?? "http://http-stub.dephealth-conformance.svc:8080";
var grpcStubHost = Environment.GetEnvironmentVariable("GRPC_STUB_HOST")
    ?? "grpc-stub.dephealth-conformance.svc";
var grpcStubPort = Environment.GetEnvironmentVariable("GRPC_STUB_PORT")
    ?? "9090";
var intervalStr = Environment.GetEnvironmentVariable("CHECK_INTERVAL") ?? "10";

var checkInterval = TimeSpan.FromSeconds(int.Parse(intervalStr));

// --- Регистрация DepHealth с 7 зависимостями ---
builder.Services.AddDepHealth(dh =>
{
    dh.AddPostgres("postgres-primary", primaryDbUrl, critical: true);
    dh.AddPostgres("postgres-replica", replicaDbUrl);
    dh.AddRedis("redis-cache", redisUrl, critical: true);
    dh.AddAmqp("rabbitmq", rabbitmqUrl);
    dh.AddKafka("kafka-main", $"kafka://{kafkaHost}:{kafkaPort}");
    dh.AddHttp("http-service", httpStubUrl, healthPath: "/health");
    dh.AddGrpc("grpc-service", grpcStubHost, grpcStubPort);
    dh.WithCheckInterval(checkInterval);
});

var app = builder.Build();

// --- Endpoints ---

app.MapGet("/", () => Results.Json(new
{
    service = "dephealth-conformance-csharp",
    version = "0.1.0",
    language = "csharp"
}));

app.MapGet("/health", () => Results.Text("OK"));

app.MapDepHealthEndpoints();

// Prometheus /metrics
app.MapMetrics();

app.Run();
