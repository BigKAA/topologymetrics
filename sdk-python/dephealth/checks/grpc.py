"""gRPC health checker."""

from __future__ import annotations

from dephealth.checker import CheckConnectionRefusedError, CheckTimeoutError, UnhealthyError
from dephealth.dependency import Endpoint


class GRPCChecker:
    """Проверка доступности через gRPC Health/Check."""

    def __init__(
        self,
        service_name: str = "",
        timeout: float = 5.0,
        tls: bool = False,
        tls_skip_verify: bool = False,
    ) -> None:
        self._service_name = service_name
        self._timeout = timeout
        self._tls = tls
        self._tls_skip_verify = tls_skip_verify

    async def check(self, endpoint: Endpoint) -> None:
        """Вызывает grpc.health.v1.Health/Check."""
        try:
            import grpc
            from grpc_health.v1 import health_pb2, health_pb2_grpc
        except ImportError:
            msg = "grpcio and grpcio-health-checking are required for gRPC checker"
            raise ImportError(msg) from None

        target = f"{endpoint.host}:{endpoint.port}"

        if self._tls:
            if self._tls_skip_verify:
                # Небезопасный канал для тестирования.
                channel_creds = grpc.ssl_channel_credentials(
                    root_certificates=None,
                )
                channel = grpc.aio.secure_channel(target, channel_creds)
            else:
                channel_creds = grpc.ssl_channel_credentials()
                channel = grpc.aio.secure_channel(target, channel_creds)
        else:
            channel = grpc.aio.insecure_channel(target)

        try:
            stub = health_pb2_grpc.HealthStub(channel)
            request = health_pb2.HealthCheckRequest(service=self._service_name)
            try:
                response = await stub.Check(
                    request,
                    timeout=self._timeout,
                )
            except grpc.aio.AioRpcError as e:
                if e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    msg = f"gRPC health check to {target} timed out"
                    raise CheckTimeoutError(msg) from e
                if e.code() == grpc.StatusCode.UNAVAILABLE:
                    msg = f"gRPC connection to {target} unavailable: {e.details()}"
                    raise CheckConnectionRefusedError(msg) from e
                msg = f"gRPC health check to {target} failed: {e.details()}"
                raise CheckConnectionRefusedError(msg) from e

            serving = health_pb2.HealthCheckResponse.SERVING
            if response.status != serving:
                msg = f"gRPC service {self._service_name!r} at {target} is not SERVING"
                raise UnhealthyError(msg)
        finally:
            await channel.close()

    def checker_type(self) -> str:
        return "grpc"
