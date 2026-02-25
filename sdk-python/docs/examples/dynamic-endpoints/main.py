# Example: dynamic endpoint management via a REST API.
# Endpoints can be added, removed, and updated at runtime.
#
# Install:
#   pip install dephealth[fastapi]
#
# Run:
#   uvicorn main:app --host 0.0.0.0 --port 8080

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from dephealth import DependencyType, Endpoint
from dephealth.api import DependencyHealth, http_check
from dephealth.checks.http import HTTPChecker
from dephealth_fastapi import DepHealthMiddleware, dephealth_lifespan

# Start with one static HTTP dependency.
app = FastAPI(
    title="Dynamic endpoints example",
    lifespan=dephealth_lifespan(
        "gateway",
        "platform",
        http_check(
            "users-api",
            url="http://users.internal:8080",
            critical=True,
        ),
    ),
)

app.add_middleware(DepHealthMiddleware)


# --- Request models ---


class AddEndpointRequest(BaseModel):
    name: str
    host: str
    port: str
    critical: bool = True


class UpdateEndpointRequest(BaseModel):
    name: str
    old_host: str
    old_port: str
    new_host: str
    new_port: str


class RemoveEndpointRequest(BaseModel):
    name: str
    host: str
    port: str


# --- Endpoints ---


@app.get("/health")
async def health(request: Request) -> JSONResponse:
    """Current health status with endpoint details."""
    dh: DependencyHealth = request.app.state.dephealth
    details = dh.health_details()
    return JSONResponse(content={k: v.to_dict() for k, v in details.items()})


@app.post("/endpoints", status_code=201)
async def add_endpoint(req: AddEndpointRequest, request: Request) -> dict[str, str]:
    """Add a new monitored HTTP endpoint at runtime."""
    dh: DependencyHealth = request.app.state.dephealth

    await dh.add_endpoint(
        req.name,
        DependencyType.HTTP,
        req.critical,
        Endpoint(host=req.host, port=req.port),
        HTTPChecker(),
    )
    return {"status": "added"}


@app.delete("/endpoints")
async def remove_endpoint(req: RemoveEndpointRequest, request: Request) -> dict[str, str]:
    """Remove a monitored endpoint at runtime."""
    dh: DependencyHealth = request.app.state.dephealth

    await dh.remove_endpoint(req.name, req.host, req.port)
    return {"status": "removed"}


@app.put("/endpoints")
async def update_endpoint(req: UpdateEndpointRequest, request: Request) -> dict[str, str]:
    """Replace an endpoint's target at runtime."""
    dh: DependencyHealth = request.app.state.dephealth

    await dh.update_endpoint(
        req.name,
        req.old_host,
        req.old_port,
        Endpoint(host=req.new_host, port=req.new_port),
        HTTPChecker(),
    )
    return {"status": "updated"}
