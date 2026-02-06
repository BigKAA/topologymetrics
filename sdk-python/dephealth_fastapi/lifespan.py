"""Lifespan-менеджер: запуск и остановка DependencyHealth вместе с FastAPI."""

from __future__ import annotations

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from typing import Any

from fastapi import FastAPI

from dephealth.api import DependencyHealth, _DependencySpec


def dephealth_lifespan(
    *specs: _DependencySpec,
    **kwargs: Any,  # noqa: ANN401
) -> object:
    """Фабрика lifespan для FastAPI.

    Возвращает callable, совместимый с ``FastAPI(lifespan=...)``.

    Пример::

        app = FastAPI(lifespan=dephealth_lifespan(
            http_check("payment", url="http://payment:8080"),
            postgres_check("db", url="postgres://db:5432/mydb"),
        ))
    """

    @asynccontextmanager
    async def _lifespan(app: FastAPI) -> AsyncIterator[dict[str, Any]]:
        dh = DependencyHealth(*specs, **kwargs)
        app.state.dephealth = dh
        await dh.start()
        try:
            yield {"dephealth": dh}
        finally:
            await dh.stop()

    return _lifespan
