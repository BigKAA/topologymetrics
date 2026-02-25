# Example: basic FastAPI application with dephealth monitoring.
# Monitors an HTTP dependency and exposes Prometheus metrics.
#
# Install:
#   pip install dephealth[fastapi]
#
# Run:
#   uvicorn main:app --host 0.0.0.0 --port 8080

from fastapi import FastAPI

from dephealth.api import http_check
from dephealth_fastapi import DepHealthMiddleware, dependencies_router, dephealth_lifespan

# Create FastAPI app with dephealth lifespan.
# DependencyHealth starts on startup and stops on shutdown automatically.
app = FastAPI(
    title="Basic dephealth example",
    lifespan=dephealth_lifespan(
        "my-service",
        "backend",
        http_check(
            "payment-api",
            url="https://payment.internal:8443/health",
            critical=True,
        ),
    ),
)

# Expose Prometheus metrics on GET /metrics.
app.add_middleware(DepHealthMiddleware)

# Expose JSON health status on GET /health/dependencies.
app.include_router(dependencies_router)


@app.get("/")
async def root() -> dict[str, str]:
    return {"message": "Hello, World!"}
