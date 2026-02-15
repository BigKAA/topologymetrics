"""Check scheduler: periodic dependency health checks."""

from __future__ import annotations

import asyncio
import logging
import threading
import time
from dataclasses import dataclass, field
from datetime import UTC, datetime

from dephealth.check_result import STATUS_UNKNOWN, classify_error
from dephealth.checker import CheckError, HealthChecker
from dephealth.dependency import Dependency, Endpoint
from dephealth.endpoint_status import EndpointStatus
from dephealth.metrics import MetricsExporter

logger = logging.getLogger("dephealth.scheduler")


@dataclass
class _EndpointState:
    """Health check state of a single endpoint."""

    healthy: bool | None = None
    consecutive_failures: int = 0
    consecutive_successes: int = 0

    # Dynamic fields updated after each check (for HealthDetails API).
    last_status: str = STATUS_UNKNOWN
    last_detail: str = "unknown"
    last_latency: float = 0.0
    last_checked_at: datetime | None = None

    # Static fields set at creation time.
    dep_name: str = ""
    dep_type: str = ""
    host: str = ""
    port: str = ""
    critical: bool = False
    labels: dict[str, str] = field(default_factory=dict)


@dataclass
class _SchedulerEntry:
    """Scheduler entry: dependency + checker + endpoint states."""

    dep: Dependency
    checker: HealthChecker
    states: dict[str, _EndpointState] = field(default_factory=dict)


class CheckScheduler:
    """Dependency health check scheduler.

    Supports two modes:
    - asyncio (primary): via asyncio.create_task
    - threading (fallback): via threading.Timer
    """

    def __init__(
        self,
        metrics: MetricsExporter,
        log: logging.Logger | None = None,
    ) -> None:
        self._metrics = metrics
        self._log = log or logger
        self._entries: list[_SchedulerEntry] = []
        self._tasks: list[asyncio.Task[None]] = []
        self._threads: list[threading.Event] = []
        self._running = False

    def add(self, dep: Dependency, checker: HealthChecker) -> None:
        """Add a dependency for monitoring."""
        entry = _SchedulerEntry(dep=dep, checker=checker)
        for ep in dep.endpoints:
            key = f"{ep.host}:{ep.port}"
            entry.states[key] = _EndpointState(
                dep_name=dep.name,
                dep_type=str(dep.type),
                host=ep.host,
                port=ep.port,
                critical=dep.critical,
                labels=dict(ep.labels),
            )
        self._entries.append(entry)

    async def start(self) -> None:
        """Start in asyncio mode."""
        self._running = True
        for entry in self._entries:
            for ep in entry.dep.endpoints:
                task = asyncio.create_task(self._run_loop(entry, ep))
                self._tasks.append(task)
        self._log.info("Scheduler started (%d entries)", len(self._entries))

    async def stop(self) -> None:
        """Stop asyncio mode."""
        self._running = False
        for task in self._tasks:
            task.cancel()
        await asyncio.gather(*self._tasks, return_exceptions=True)
        self._tasks.clear()
        self._log.info("Scheduler stopped")

    def start_sync(self) -> None:
        """Start in threading mode (fallback)."""
        self._running = True
        for entry in self._entries:
            for ep in entry.dep.endpoints:
                stop_event = threading.Event()
                self._threads.append(stop_event)
                t = threading.Thread(
                    target=self._run_thread,
                    args=(entry, ep, stop_event),
                    daemon=True,
                )
                t.start()
        self._log.info("Scheduler started in sync mode (%d entries)", len(self._entries))

    def stop_sync(self) -> None:
        """Stop threading mode."""
        self._running = False
        for event in self._threads:
            event.set()
        self._threads.clear()
        self._log.info("Scheduler stopped (sync mode)")

    def health(self) -> dict[str, bool]:
        """Return current health status of all dependencies."""
        result: dict[str, bool] = {}
        for entry in self._entries:
            # A dependency is healthy if at least one endpoint is healthy.
            healthy = any(s.healthy for s in entry.states.values())
            result[entry.dep.name] = healthy
        return result

    def health_details(self) -> dict[str, EndpointStatus]:
        """Return detailed health status of all endpoints.

        Key format: ``"dependency:host:port"``.
        Includes UNKNOWN endpoints (before first check) with status="unknown"
        and healthy=None.
        """
        result: dict[str, EndpointStatus] = {}
        for entry in self._entries:
            for ep in entry.dep.endpoints:
                hp_key = f"{ep.host}:{ep.port}"
                state = entry.states[hp_key]
                key = f"{entry.dep.name}:{ep.host}:{ep.port}"
                result[key] = EndpointStatus(
                    healthy=state.healthy,
                    status=state.last_status,
                    detail=state.last_detail,
                    latency=state.last_latency,
                    type=state.dep_type,
                    name=state.dep_name,
                    host=state.host,
                    port=state.port,
                    critical=state.critical,
                    last_checked_at=state.last_checked_at,
                    labels=dict(state.labels),
                )
        return result

    async def _run_loop(self, entry: _SchedulerEntry, ep: Endpoint) -> None:
        """Check loop for a single endpoint (asyncio)."""
        if entry.dep.config.initial_delay > 0:
            await asyncio.sleep(entry.dep.config.initial_delay)

        while self._running:
            await self._check_once(entry, ep)
            await asyncio.sleep(entry.dep.config.interval)

    def _run_thread(
        self, entry: _SchedulerEntry, ep: Endpoint, stop_event: threading.Event
    ) -> None:
        """Check loop for a single endpoint (threading)."""
        if entry.dep.config.initial_delay > 0:
            stop_event.wait(entry.dep.config.initial_delay)
            if stop_event.is_set():
                return

        loop = asyncio.new_event_loop()
        try:
            while self._running and not stop_event.is_set():
                loop.run_until_complete(self._check_once(entry, ep))
                stop_event.wait(entry.dep.config.interval)
        finally:
            loop.close()

    async def _check_once(self, entry: _SchedulerEntry, ep: Endpoint) -> None:
        """Run a single endpoint check and update metrics and thresholds."""
        key = f"{ep.host}:{ep.port}"
        state = entry.states[key]

        start = time.monotonic()
        check_err: BaseException | None = None
        try:
            await asyncio.wait_for(
                entry.checker.check(ep),
                timeout=entry.dep.config.timeout,
            )
            success = True
        except (CheckError, TimeoutError, Exception) as e:
            success = False
            check_err = e
            self._log.debug("Check failed for %s (%s): %s", entry.dep.name, key, e)

        duration = time.monotonic() - start
        self._metrics.observe_latency(entry.dep, ep, duration)

        # Classify the error and set status/detail metrics.
        result = classify_error(check_err)
        self._metrics.set_status(entry.dep, ep, result.category)
        self._metrics.set_status_detail(entry.dep, ep, result.detail)

        # Store classification results for health_details() API.
        state.last_status = result.category
        state.last_detail = result.detail
        state.last_latency = duration
        state.last_checked_at = datetime.now(UTC)

        if success:
            state.consecutive_successes += 1
            state.consecutive_failures = 0
            if state.consecutive_successes >= entry.dep.config.success_threshold:
                if not state.healthy:
                    self._log.info("Dependency %s (%s) is now healthy", entry.dep.name, key)
                state.healthy = True
                self._metrics.set_health(entry.dep, ep, 1.0)
        else:
            state.consecutive_failures += 1
            state.consecutive_successes = 0
            if state.consecutive_failures >= entry.dep.config.failure_threshold:
                if state.healthy:
                    self._log.warning("Dependency %s (%s) is now unhealthy", entry.dep.name, key)
                state.healthy = False
                self._metrics.set_health(entry.dep, ep, 0.0)
