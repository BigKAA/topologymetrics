// Example: dynamic endpoint management via a REST API.
// Endpoints can be added, removed, and updated at runtime.
//
// Prerequisites:
//   dotnet add package DepHealth.Core
//   dotnet add package DepHealth.AspNetCore
//
// Run:
//   dotnet run
//
// Usage:
//   curl -X POST http://localhost:5000/endpoints \
//     -H "Content-Type: application/json" \
//     -d '{"name":"billing-api","host":"billing.internal","port":"8080","critical":true}'
//
//   curl -X DELETE http://localhost:5000/endpoints \
//     -H "Content-Type: application/json" \
//     -d '{"name":"billing-api","host":"billing.internal","port":"8080"}'
//
//   curl -X PUT http://localhost:5000/endpoints \
//     -H "Content-Type: application/json" \
//     -d '{"name":"billing-api","oldHost":"billing.internal","oldPort":"8080","newHost":"billing-v2.internal","newPort":"8080"}'

using System.Text.Json;
using DepHealth;
using DepHealth.AspNetCore;
using DepHealth.Checks;
using Prometheus;

var builder = WebApplication.CreateBuilder(args);

// Start with one static HTTP dependency.
builder.Services.AddDepHealth("gateway", "platform", dh => dh
    .AddHttp("users-api", "http://users.internal:8080", critical: true)
);

var app = builder.Build();

// Prometheus metrics.
app.MapMetrics();

// Current health status.
app.MapGet("/health", (DepHealthMonitor monitor) =>
{
    var details = monitor.HealthDetails();
    return Results.Json(details, new JsonSerializerOptions { WriteIndented = true });
});

// POST /endpoints — add a new monitored endpoint.
app.MapPost("/endpoints", (AddEndpointRequest req, DepHealthMonitor monitor) =>
{
    var ep = new Endpoint(req.Host, req.Port);
    var checker = new HttpChecker();
    monitor.AddEndpoint(req.Name, DependencyType.Http, req.Critical, ep, checker);
    return Results.Json(new { status = "added" }, statusCode: 201);
});

// DELETE /endpoints — remove a monitored endpoint.
app.MapDelete("/endpoints", (RemoveEndpointRequest req, DepHealthMonitor monitor) =>
{
    monitor.RemoveEndpoint(req.Name, req.Host, req.Port);
    return Results.Json(new { status = "removed" });
});

// PUT /endpoints — update an existing endpoint's target.
app.MapPut("/endpoints", (UpdateEndpointRequest req, DepHealthMonitor monitor) =>
{
    var newEp = new Endpoint(req.NewHost, req.NewPort);
    var checker = new HttpChecker();
    monitor.UpdateEndpoint(req.Name, req.OldHost, req.OldPort, newEp, checker);
    return Results.Json(new { status = "updated" });
});

app.Run();

// --- Request models ---

record AddEndpointRequest(string Name, string Host, string Port, bool Critical = true);

record RemoveEndpointRequest(string Name, string Host, string Port);

record UpdateEndpointRequest(
    string Name,
    string OldHost, string OldPort,
    string NewHost, string NewPort);
