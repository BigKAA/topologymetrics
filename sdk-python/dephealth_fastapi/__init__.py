"""dephealth-fastapi — интеграция dephealth SDK с FastAPI."""

from dephealth_fastapi.endpoints import dependencies_router
from dephealth_fastapi.lifespan import dephealth_lifespan
from dephealth_fastapi.middleware import DepHealthMiddleware

__all__ = [
    "DepHealthMiddleware",
    "dephealth_lifespan",
    "dependencies_router",
]
