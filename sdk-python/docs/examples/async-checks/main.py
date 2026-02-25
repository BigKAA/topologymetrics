# Example: standalone async monitoring without a web framework.
# Uses asyncio directly to start monitoring, query health status,
# and shut down gracefully on SIGINT/SIGTERM.
#
# Install:
#   pip install dephealth[postgres,redis]
#
# Run:
#   python main.py

import asyncio
import contextlib
import json
import signal
from datetime import timedelta

from dephealth.api import DependencyHealth, http_check, postgres_check, redis_check

shutdown_event = asyncio.Event()


def _handle_signal() -> None:
    shutdown_event.set()


async def main() -> None:
    # Create DependencyHealth with multiple checks and custom intervals.
    dh = DependencyHealth(
        "worker-service",
        "backend",
        http_check(
            "auth-api",
            url="http://auth.internal:8080",
            critical=True,
            interval=timedelta(seconds=5),
            timeout=timedelta(seconds=2),
        ),
        postgres_check(
            "orders-db",
            url="postgres://app:secret@pg.db:5432/orders",
            critical=True,
        ),
        redis_check(
            "cache",
            url="redis://redis.cache:6379",
            critical=False,
            interval=timedelta(seconds=10),
        ),
        check_interval=timedelta(seconds=15),
        timeout=timedelta(seconds=5),
    )

    # Start async monitoring — creates an asyncio.Task per endpoint.
    await dh.start()

    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, _handle_signal)

    # Periodically print health status until shutdown.
    while not shutdown_event.is_set():
        health = dh.health()
        details = dh.health_details()

        print("--- Health Status ---")
        for name, ok in health.items():
            print(f"  {name}: {'healthy' if ok else 'UNHEALTHY'}")

        print("--- Endpoint Details ---")
        for key, status in details.items():
            info = status.to_dict()
            print(f"  {key}: {json.dumps(info, default=str)}")

        print()

        with contextlib.suppress(TimeoutError):
            await asyncio.wait_for(shutdown_event.wait(), timeout=10.0)

    await dh.stop()
    print("Shut down gracefully.")


if __name__ == "__main__":
    asyncio.run(main())
