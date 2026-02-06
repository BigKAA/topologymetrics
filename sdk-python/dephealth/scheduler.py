"""Check scheduler: периодический запуск проверок зависимостей."""

from __future__ import annotations

import asyncio
import logging
import threading
import time
from dataclasses import dataclass, field

from dephealth.checker import CheckError, HealthChecker
from dephealth.dependency import Dependency, Endpoint
from dephealth.metrics import MetricsExporter

logger = logging.getLogger("dephealth.scheduler")


@dataclass
class _EndpointState:
    """Состояние проверки одного endpoint."""

    healthy: bool = False
    consecutive_failures: int = 0
    consecutive_successes: int = 0


@dataclass
class _SchedulerEntry:
    """Запись планировщика: зависимость + чекер + состояния endpoint-ов."""

    dep: Dependency
    checker: HealthChecker
    states: dict[str, _EndpointState] = field(default_factory=dict)


class CheckScheduler:
    """Планировщик проверок зависимостей.

    Поддерживает два режима:
    - asyncio (основной): через asyncio.create_task
    - threading (fallback): через threading.Timer
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
        """Добавляет зависимость для мониторинга."""
        entry = _SchedulerEntry(dep=dep, checker=checker)
        for ep in dep.endpoints:
            key = f"{ep.host}:{ep.port}"
            entry.states[key] = _EndpointState()
        self._entries.append(entry)

    async def start(self) -> None:
        """Запуск в asyncio-режиме."""
        self._running = True
        for entry in self._entries:
            for ep in entry.dep.endpoints:
                task = asyncio.create_task(self._run_loop(entry, ep))
                self._tasks.append(task)
        self._log.info("Scheduler started (%d entries)", len(self._entries))

    async def stop(self) -> None:
        """Остановка asyncio-режима."""
        self._running = False
        for task in self._tasks:
            task.cancel()
        await asyncio.gather(*self._tasks, return_exceptions=True)
        self._tasks.clear()
        self._log.info("Scheduler stopped")

    def start_sync(self) -> None:
        """Запуск в threading-режиме (fallback)."""
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
        """Остановка threading-режима."""
        self._running = False
        for event in self._threads:
            event.set()
        self._threads.clear()
        self._log.info("Scheduler stopped (sync mode)")

    def health(self) -> dict[str, bool]:
        """Текущее состояние всех зависимостей."""
        result: dict[str, bool] = {}
        for entry in self._entries:
            # Зависимость здорова, если хотя бы один endpoint здоров.
            healthy = any(s.healthy for s in entry.states.values())
            result[entry.dep.name] = healthy
        return result

    async def _run_loop(self, entry: _SchedulerEntry, ep: Endpoint) -> None:
        """Цикл проверки одного endpoint (asyncio)."""
        if entry.dep.config.initial_delay > 0:
            await asyncio.sleep(entry.dep.config.initial_delay)

        while self._running:
            await self._check_once(entry, ep)
            await asyncio.sleep(entry.dep.config.interval)

    def _run_thread(
        self, entry: _SchedulerEntry, ep: Endpoint, stop_event: threading.Event
    ) -> None:
        """Цикл проверки одного endpoint (threading)."""
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
        """Одна проверка endpoint с обновлением метрик и порогов."""
        key = f"{ep.host}:{ep.port}"
        state = entry.states[key]

        start = time.monotonic()
        try:
            await asyncio.wait_for(
                entry.checker.check(ep),
                timeout=entry.dep.config.timeout,
            )
            success = True
        except (CheckError, TimeoutError, Exception) as e:
            success = False
            self._log.debug("Check failed for %s (%s): %s", entry.dep.name, key, e)

        duration = time.monotonic() - start
        self._metrics.observe_latency(entry.dep, ep, duration)

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
