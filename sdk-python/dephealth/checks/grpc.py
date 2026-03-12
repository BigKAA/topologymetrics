"""gRPC health checker."""

from __future__ import annotations

import base64

from dephealth.checker import (
    CheckAuthError,
    CheckConnectionRefusedError,
    CheckError,
    CheckTimeoutError,
    UnhealthyError,
)
from dephealth.dependency import Endpoint


def _validate_grpc_auth(
    metadata: dict[str, str] | None,
    bearer_token: str | None,
    basic_auth: tuple[str, str] | None,
) -> None:
    """Validate that at most one auth method is configured."""
    methods = 0
    if bearer_token:
        methods += 1
    if basic_auth:
        methods += 1
    if metadata:
        for key in metadata:
            if key.lower() == "authorization":
                methods += 1
                break
    if methods > 1:
        msg = (
            "conflicting auth methods: specify only one of "
            "bearer_token, basic_auth, or authorization metadata"
        )
        raise ValueError(msg)


def _validate_grpc_authority(
    metadata: dict[str, str] | None,
    authority: str | None,
) -> None:
    """Validate that authority does not conflict with :authority in metadata."""
    if not authority or not metadata:
        return
    if ":authority" in metadata:
        msg = (
            "conflicting :authority: specify only one of "
            "authority or :authority in metadata"
        )
        raise ValueError(msg)


def _build_metadata(
    metadata: dict[str, str] | None,
    bearer_token: str | None,
    basic_auth: tuple[str, str] | None,
) -> list[tuple[str, str]]:
    """Build the resolved metadata list from auth parameters."""
    resolved: dict[str, str] = {}
    if metadata:
        resolved.update(metadata)
    if bearer_token:
        resolved["authorization"] = f"Bearer {bearer_token}"
    if basic_auth:
        username, password = basic_auth
        credentials = base64.b64encode(
            f"{username}:{password}".encode(),
        ).decode("ascii")
        resolved["authorization"] = f"Basic {credentials}"
    return list(resolved.items())


class GRPCChecker:
    """Health check via gRPC Health/Check."""

    def __init__(
        self,
        service_name: str = "",
        timeout: float = 5.0,
        tls: bool = False,
        tls_skip_verify: bool = False,
        metadata: dict[str, str] | None = None,
        bearer_token: str | None = None,
        basic_auth: tuple[str, str] | None = None,
        authority: str | None = None,
    ) -> None:
        _validate_grpc_auth(metadata, bearer_token, basic_auth)
        _validate_grpc_authority(metadata, authority)
        self._service_name = service_name
        self._timeout = timeout
        self._tls = tls
        self._tls_skip_verify = tls_skip_verify
        self._metadata = _build_metadata(metadata, bearer_token, basic_auth)
        self._authority = authority

    async def check(self, endpoint: Endpoint) -> None:
        """Call grpc.health.v1.Health/Check."""
        try:
            import grpc
            from grpc_health.v1 import health_pb2, health_pb2_grpc
        except ImportError:
            msg = "grpcio and grpcio-health-checking are required for gRPC checker"
            raise ImportError(msg) from None

        target = f"{endpoint.host}:{endpoint.port}"

        options: list[tuple[str, str]] = []
        if self._authority:
            options.append(("grpc.default_authority", self._authority))

        if self._tls:
            if self._tls_skip_verify:
                channel_creds = grpc.ssl_channel_credentials(
                    root_certificates=None,
                )
            else:
                channel_creds = grpc.ssl_channel_credentials()
            if self._authority:
                options.append(
                    ("grpc.ssl_target_name_override", self._authority),
                )
            channel = grpc.aio.secure_channel(
                target, channel_creds, options=options or None,
            )
        else:
            channel = grpc.aio.insecure_channel(
                target, options=options or None,
            )

        try:
            stub = health_pb2_grpc.HealthStub(channel)
            request = health_pb2.HealthCheckRequest(service=self._service_name)
            call_kwargs: dict[str, object] = {"timeout": self._timeout}
            if self._metadata:
                call_kwargs["metadata"] = self._metadata
            try:
                response = await stub.Check(request, **call_kwargs)
            except grpc.aio.AioRpcError as e:
                if e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    msg = f"gRPC health check to {target} timed out"
                    raise CheckTimeoutError(msg) from e
                # UNAUTHENTICATED / PERMISSION_DENIED → auth_error.
                if e.code() in (
                    grpc.StatusCode.UNAUTHENTICATED,
                    grpc.StatusCode.PERMISSION_DENIED,
                ):
                    msg = f"gRPC health check to {target}: {e.details()}"
                    raise CheckAuthError(msg) from e
                if e.code() == grpc.StatusCode.UNAVAILABLE:
                    msg = f"gRPC connection to {target} unavailable: {e.details()}"
                    raise CheckConnectionRefusedError(msg) from e
                msg = f"gRPC health check to {target} failed: {e.details()}"
                raise CheckError(msg) from e

            serving = health_pb2.HealthCheckResponse.SERVING
            unknown = health_pb2.HealthCheckResponse.UNKNOWN
            if response.status != serving:
                detail = "grpc_unknown" if response.status == unknown else "grpc_not_serving"
                msg = f"gRPC service {self._service_name!r} at {target} is not SERVING"
                raise UnhealthyError(msg, detail=detail)
        finally:
            await channel.close()

    def checker_type(self) -> str:
        """Return the checker type."""
        return "grpc"
