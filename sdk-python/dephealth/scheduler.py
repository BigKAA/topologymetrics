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
from dephealth.dependency import CheckConfig, Dependency, DependencyType, Endpoint
from dephealth.endpoint_status import EndpointStatus
from dephealth.metrics import MetricsExporter

logger = logging.getLogger("dephealth.scheduler")


class EndpointNotFoundError(Exception):
    """Raised when a dynamic endpoint operation targets a non-existent endpoint."""

    def __init__(self, dep_name: str, host: str, port: str) -> None:
        self.dep_name = dep_name
        self.host = host
        self.port = port
        super().__init__(f"Endpoint not found: {dep_name}:{host}:{port}")


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
        global_config: CheckConfig | None = None,
        log: logging.Logger | None = None,
    ) -> None:
        self._metrics = metrics
        self._global_config = global_config or CheckConfig(initial_delay=0)
        self._log = log or logger

        self._entries: list[_SchedulerEntry] = []
        self._lock = threading.Lock()

        # Per-endpoint state, keyed by "dep_name:host:port".
        self._states: dict[str, _EndpointState] = {}

        # Per-endpoint tracking for asyncio mode.
        self._ep_tasks: dict[str, asyncio.Task[None]] = {}

        # Per-endpoint tracking for threading mode.
        self._ep_threads: dict[str, threading.Thread] = {}
        self._ep_stop_events: dict[str, threading.Event] = {}

        self._started = False
        self._stopped = False
        self._mode: str = ""  # "async" or "sync"

    def add(self, dep: Dependency, checker: HealthChecker) -> None:
        """Add a dependency for monitoring (called before start)."""
        entry = _SchedulerEntry(dep=dep, checker=checker)
        for ep in dep.endpoints:
            hp_key = f"{ep.host}:{ep.port}"
            state = _EndpointState(
                dep_name=dep.name,
                dep_type=str(dep.type),
                host=ep.host,
                port=ep.port,
                critical=dep.critical,
                labels=dict(ep.labels),
            )
            entry.states[hp_key] = state
            state_key = f"{dep.name}:{ep.host}:{ep.port}"
            self._states[state_key] = state
        self._entries.append(entry)

    async def start(self) -> None:
        """Start in asyncio mode."""
        self._started = True
        self._stopped = False
        self._mode = "async"
        for entry in self._entries:
            for ep in entry.dep.endpoints:
                state_key = f"{entry.dep.name}:{ep.host}:{ep.port}"
                task = asyncio.create_task(self._run_loop(entry, ep))
                self._ep_tasks[state_key] = task
        self._log.info("Scheduler started (%d entries)", len(self._entries))

    async def stop(self) -> None:
        """Stop asyncio mode."""
        self._stopped = True
        with self._lock:
            tasks = list(self._ep_tasks.values())
        for task in tasks:
            task.cancel()
        await asyncio.gather(*tasks, return_exceptions=True)
        with self._lock:
            self._ep_tasks.clear()
        self._log.info("Scheduler stopped")

    def start_sync(self) -> None:
        """Start in threading mode (fallback)."""
        self._started = True
        self._stopped = False
        self._mode = "sync"
        for entry in self._entries:
            for ep in entry.dep.endpoints:
                state_key = f"{entry.dep.name}:{ep.host}:{ep.port}"
                stop_event = threading.Event()
                self._ep_stop_events[state_key] = stop_event
                t = threading.Thread(
                    target=self._run_thread,
                    args=(entry, ep, stop_event),
                    daemon=True,
                )
                self._ep_threads[state_key] = t
                t.start()
        self._log.info("Scheduler started in sync mode (%d entries)", len(self._entries))

    def stop_sync(self) -> None:
        """Stop threading mode."""
        self._stopped = True
        with self._lock:
            events = list(self._ep_stop_events.values())
            threads = list(self._ep_threads.values())
        for event in events:
            event.set()
        for t in threads:
            t.join(timeout=5.0)
        with self._lock:
            self._ep_stop_events.clear()
            self._ep_threads.clear()
        self._log.info("Scheduler stopped (sync mode)")

    def health(self) -> dict[str, bool]:
        """Return current health status of all dependencies."""
        with self._lock:
            states_snapshot = list(self._states.items())

        # Group by dep_name — a dependency is healthy if at least one endpoint is healthy.
        by_dep: dict[str, list[bool | None]] = {}
        for _key, state in states_snapshot:
            by_dep.setdefault(state.dep_name, []).append(state.healthy)

        return {name: any(h for h in healths) for name, healths in by_dep.items()}

    def health_details(self) -> dict[str, EndpointStatus]:
        """Return detailed health status of all endpoints.

        Key format: ``"dependency:host:port"``.
        Includes UNKNOWN endpoints (before first check) with status="unknown"
        and healthy=None.
        """
        with self._lock:
            states_snapshot = list(self._states.items())

        result: dict[str, EndpointStatus] = {}
        for key, state in states_snapshot:
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

    # --- Dynamic endpoint management ---

    async def add_endpoint(
        self,
        dep_name: str,
        dep_type: DependencyType,
        critical: bool,
        ep: Endpoint,
        checker: HealthChecker,
    ) -> None:
        """Add a new endpoint at runtime (asyncio mode)."""
        self._ensure_running()
        state_key = f"{dep_name}:{ep.host}:{ep.port}"

        with self._lock:
            if state_key in self._states:
                return  # Idempotent: already exists.

            dep = Dependency(
                name=dep_name,
                type=dep_type,
                critical=critical,
                endpoints=[ep],
                config=self._global_config,
            )
            entry = _SchedulerEntry(dep=dep, checker=checker)

            state = _EndpointState(
                dep_name=dep_name,
                dep_type=str(dep_type),
                host=ep.host,
                port=ep.port,
                critical=critical,
                labels=dict(ep.labels),
            )
            hp_key = f"{ep.host}:{ep.port}"
            entry.states[hp_key] = state
            self._states[state_key] = state
            self._entries.append(entry)

            if self._mode == "async":
                task = asyncio.create_task(self._run_loop(entry, ep))
                self._ep_tasks[state_key] = task
            elif self._mode == "sync":
                stop_event = threading.Event()
                self._ep_stop_events[state_key] = stop_event
                t = threading.Thread(
                    target=self._run_thread,
                    args=(entry, ep, stop_event),
                    daemon=True,
                )
                self._ep_threads[state_key] = t
                t.start()

        self._log.info("Endpoint added: %s", state_key)

    async def remove_endpoint(self, dep_name: str, host: str, port: str) -> None:
        """Remove an endpoint at runtime (asyncio mode)."""
        self._ensure_running()
        state_key = f"{dep_name}:{host}:{port}"
        task: asyncio.Task[None] | None = None

        with self._lock:
            state = self._states.get(state_key)
            if state is None:
                return  # Idempotent: already gone.

            # Find and remove the entry.
            entry = self._find_entry(state_key)
            dep = entry.dep if entry else None
            ep = Endpoint(host=host, port=port, labels=dict(state.labels))

            if self._mode == "async":
                task = self._ep_tasks.pop(state_key, None)
                if task is not None:
                    task.cancel()
            elif self._mode == "sync":
                stop_event = self._ep_stop_events.pop(state_key, None)
                if stop_event is not None:
                    stop_event.set()
                self._ep_threads.pop(state_key, None)

            del self._states[state_key]
            if entry is not None:
                self._entries.remove(entry)

        # Await cancelled task outside lock to avoid deadlock.
        if task is not None:
            await asyncio.gather(task, return_exceptions=True)

        # Clean up metrics.
        if dep is not None:
            self._metrics.delete_metrics(dep, ep)

        self._log.info("Endpoint removed: %s", state_key)

    async def update_endpoint(
        self,
        dep_name: str,
        old_host: str,
        old_port: str,
        new_ep: Endpoint,
        checker: HealthChecker,
    ) -> None:
        """Replace an endpoint at runtime (asyncio mode)."""
        old_key = f"{dep_name}:{old_host}:{old_port}"
        with self._lock:
            if old_key not in self._states:
                raise EndpointNotFoundError(dep_name, old_host, old_port)
            old_state = self._states[old_key]
            dep_type = DependencyType(old_state.dep_type)
            critical = old_state.critical

        await self.remove_endpoint(dep_name, old_host, old_port)
        await self.add_endpoint(dep_name, dep_type, critical, new_ep, checker)

    def add_endpoint_sync(
        self,
        dep_name: str,
        dep_type: DependencyType,
        critical: bool,
        ep: Endpoint,
        checker: HealthChecker,
    ) -> None:
        """Add a new endpoint at runtime (threading mode)."""
        self._ensure_running()
        state_key = f"{dep_name}:{ep.host}:{ep.port}"

        with self._lock:
            if state_key in self._states:
                return

            dep = Dependency(
                name=dep_name,
                type=dep_type,
                critical=critical,
                endpoints=[ep],
                config=self._global_config,
            )
            entry = _SchedulerEntry(dep=dep, checker=checker)

            state = _EndpointState(
                dep_name=dep_name,
                dep_type=str(dep_type),
                host=ep.host,
                port=ep.port,
                critical=critical,
                labels=dict(ep.labels),
            )
            hp_key = f"{ep.host}:{ep.port}"
            entry.states[hp_key] = state
            self._states[state_key] = state
            self._entries.append(entry)

            stop_event = threading.Event()
            self._ep_stop_events[state_key] = stop_event
            t = threading.Thread(
                target=self._run_thread,
                args=(entry, ep, stop_event),
                daemon=True,
            )
            self._ep_threads[state_key] = t
            t.start()

        self._log.info("Endpoint added (sync): %s", state_key)

    def remove_endpoint_sync(self, dep_name: str, host: str, port: str) -> None:
        """Remove an endpoint at runtime (threading mode)."""
        self._ensure_running()
        state_key = f"{dep_name}:{host}:{port}"

        with self._lock:
            state = self._states.get(state_key)
            if state is None:
                return

            entry = self._find_entry(state_key)
            dep = entry.dep if entry else None
            ep = Endpoint(host=host, port=port, labels=dict(state.labels))

            stop_event = self._ep_stop_events.pop(state_key, None)
            if stop_event is not None:
                stop_event.set()
            thread = self._ep_threads.pop(state_key, None)

            del self._states[state_key]
            if entry is not None:
                self._entries.remove(entry)

        if thread is not None:
            thread.join(timeout=5.0)

        if dep is not None:
            self._metrics.delete_metrics(dep, ep)

        self._log.info("Endpoint removed (sync): %s", state_key)

    def update_endpoint_sync(
        self,
        dep_name: str,
        old_host: str,
        old_port: str,
        new_ep: Endpoint,
        checker: HealthChecker,
    ) -> None:
        """Replace an endpoint at runtime (threading mode)."""
        old_key = f"{dep_name}:{old_host}:{old_port}"
        with self._lock:
            if old_key not in self._states:
                raise EndpointNotFoundError(dep_name, old_host, old_port)
            old_state = self._states[old_key]
            dep_type = DependencyType(old_state.dep_type)
            critical = old_state.critical

        self.remove_endpoint_sync(dep_name, old_host, old_port)
        self.add_endpoint_sync(dep_name, dep_type, critical, new_ep, checker)

    def _ensure_running(self) -> None:
        """Raise if the scheduler is not in a running state."""
        if not self._started:
            msg = "Scheduler not started"
            raise RuntimeError(msg)
        if self._stopped:
            msg = "Scheduler already stopped"
            raise RuntimeError(msg)

    def _find_entry(self, state_key: str) -> _SchedulerEntry | None:
        """Find the _SchedulerEntry owning the given state key (lock must be held)."""
        parts = state_key.split(":", 2)
        dep_name = parts[0]
        hp_key = f"{parts[1]}:{parts[2]}"
        for entry in self._entries:
            if entry.dep.name == dep_name and hp_key in entry.states:
                return entry
        return None

    async def _run_loop(self, entry: _SchedulerEntry, ep: Endpoint) -> None:
        """Check loop for a single endpoint (asyncio)."""
        if entry.dep.config.initial_delay > 0:
            await asyncio.sleep(entry.dep.config.initial_delay)

        while not self._stopped:
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
            while not self._stopped and not stop_event.is_set():
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
