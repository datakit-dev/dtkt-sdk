import logging
from connectrpc.request import RequestContext
from connectrpc.errors import ConnectError
from connectrpc.code import Code

from connectrpc.interceptor import (
    UnaryInterceptor,
    ClientStreamInterceptor,
    ServerStreamInterceptor,
    BidiStreamInterceptor,
)
from grpcreflect._gen.grpc.reflection.v1.reflection_connect import (
    ServerReflection as ServerReflectionV1,
)
from grpcreflect._gen.grpc.reflection.v1alpha.reflection_connect import (
    ServerReflection as ServerReflectionV1Alpha,
)
from dtkt_sdk.gen.dtkt.base.v1beta1.services_connect import (
    BaseService,
)

from dtkt_sdk.integration.instance import (
    BaseInstance,
    InstanceContext,
    InstanceMux,
    get_instance_key,
)

from dtkt_sdk.integration.util import measure_time_async

from collections.abc import Callable, Awaitable, AsyncIterator
from typing import Any, Literal

from dtkt_sdk.common.middleware import (
    RESOURCE_NAME_HEADER,
)

logger = logging.getLogger(__name__)

skipServiceMethods: dict[str, set[str] | Literal[True]] = {
    BaseService.desc().type_name: {"GetPackage", "CheckConfig", "ListTypes", "GetType"},
    ServerReflectionV1.desc().type_name: True,
    ServerReflectionV1Alpha.desc().type_name: True,
}


def shouldSkipMethod(service_name: str, method_name: str) -> bool:
    skipService = skipServiceMethods.get(service_name, set())
    return skipService is True or method_name in skipService


class Interceptor[I: BaseInstance](
    UnaryInterceptor,
    ClientStreamInterceptor,
    ServerStreamInterceptor,
    BidiStreamInterceptor,
):
    def __init__(self, mux: InstanceMux):
        self.mux = mux

    def new_instance_context(self, ctx: RequestContext) -> InstanceContext[I]:
        inst = self.mux.get_instance(get_instance_key(ctx))
        if inst is None:
            resource_name = ctx.request_headers.get(RESOURCE_NAME_HEADER)
            if resource_name is None:
                resource_name = "<unknown>"

            raise ConnectError(
                Code.INVALID_ARGUMENT,
                f"Instance not found for resource: {resource_name}",
            )

        return InstanceContext(ctx, inst)

    async def intercept_unary(
        self,
        call_next: Callable[[Any, RequestContext], Awaitable[Any]],
        request: Any,
        ctx: RequestContext,
    ):
        skipService = skipServiceMethods.get(ctx.method.service_name, set())
        if skipService is True or ctx.method.name in skipService:
            return await call_next(request, ctx)

        return await call_next(request, self.new_instance_context(ctx))

    async def intercept_client_stream(
        self,
        call_next: Callable[[AsyncIterator[Any], RequestContext], Awaitable[Any]],
        request: AsyncIterator[Any],
        ctx: RequestContext,
    ):
        if shouldSkipMethod(ctx.method.service_name, ctx.method.name):
            return await call_next(request, ctx)

        return await call_next(request, self.new_instance_context(ctx))

    def intercept_server_stream(
        self,
        call_next: Callable[[Any, RequestContext], AsyncIterator[Any]],
        request: Any,
        ctx: RequestContext,
    ):
        if shouldSkipMethod(ctx.method.service_name, ctx.method.name):
            return call_next(request, ctx)

        return call_next(request, self.new_instance_context(ctx))

    def intercept_bidi_stream(
        self,
        call_next: Callable[[AsyncIterator[Any], RequestContext], AsyncIterator[Any]],
        request: AsyncIterator[Any],
        ctx: RequestContext,
    ):
        if shouldSkipMethod(ctx.method.service_name, ctx.method.name):
            return call_next(request, ctx)

        return call_next(request, self.new_instance_context(ctx))

    @measure_time_async
    async def _measure_unary_call(
        self,
        call_next: Callable[[Any, RequestContext], Awaitable[Any]],
        request: Any,
        ctx: RequestContext,
    ):
        return await call_next(request, ctx)
