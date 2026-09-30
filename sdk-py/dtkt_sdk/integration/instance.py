import logging

from typing import Protocol, runtime_checkable, Awaitable

from singleflight.basic import SingleFlight
from connectrpc.errors import ConnectError
from connectrpc.code import Code
from connectrpc.request import RequestContext

from protobuf import Message
from protobuf.wkt import Any as AnyMessage, Timestamp
from protovalidate import validate, ValidationError
from dtkt_sdk.common.validate import violation_messages
from protobuf.wkt import Struct
from pydantic import TypeAdapter

from dtkt_sdk.gen.dtkt.shared.v1beta1.messages_pb import TypeSchema
from dtkt_sdk.gen.dtkt.base.v1beta1.messages_pb import (
    CheckAuthRequest,
    CheckAuthResponse,
)

from dtkt_sdk.common.middleware import (
    RESOURCE_NAME_HEADER,
    CONFIG_HASH_HEADER,
    CONFIG_GEN_HEADER,
)

logger = logging.getLogger(__name__)


def get_instance_key(ctx: RequestContext) -> str:
    headers = ctx.request_headers
    resource_name = headers.get(RESOURCE_NAME_HEADER)
    if resource_name is None:
        raise ConnectError(
            Code.INVALID_ARGUMENT, f"Header required: {RESOURCE_NAME_HEADER}"
        )

    config_hash = headers.get(CONFIG_HASH_HEADER)
    if config_hash is None:
        raise ConnectError(
            Code.INVALID_ARGUMENT, f"Header required: {CONFIG_HASH_HEADER}"
        )

    config_gen = int(headers.get(CONFIG_GEN_HEADER, 0))
    if config_gen < 1:
        raise ConnectError(
            Code.INVALID_ARGUMENT, f"Header invalid: {CONFIG_GEN_HEADER}"
        )

    return build_instance_key(
        resource_name=resource_name,
        config_hash=config_hash,
        config_gen=config_gen,
    )


def build_instance_key(resource_name: str, config_hash: str, config_gen: int) -> str:
    return f"{resource_name}/{config_gen}/{config_hash}".lower()


class InstanceContext[I: BaseInstance](RequestContext):
    def __init__(self, ctx: RequestContext, instance: I):
        self.instance = instance

        timeout_ms = None
        if ctx.timeout_ms:
            timeout_ms = int(ctx.timeout_ms)

        super().__init__(
            method=ctx.method,
            http_method=ctx.http_method,
            request_headers=ctx.request_headers,
            timeout_ms=timeout_ms,
            server_address=ctx.server_address,
            client_address=ctx.client_address,
        )


@runtime_checkable
class BaseInstance[T: Message](Protocol):
    config: T
    data_root: str

    def __init__(self, config: T):
        self.config = config

    async def check_auth(
        self,
        request: CheckAuthRequest,
        ctx: RequestContext[CheckAuthRequest, CheckAuthResponse],
        /,
    ) -> CheckAuthResponse: ...

    async def close(self) -> None: ...

    @classmethod
    def from_config(cls, config_type: type[T], any_msg: AnyMessage):
        config = any_msg.unpack(config_type().desc())
        if not config:
            raise ValueError("Failed to unpack into the expected config type")
        elif not isinstance(config, config_type):
            raise ValueError("Unpacked config is not of the expected type")

        try:
            validate(config)
        except ValidationError as e:
            raise ValueError(
                f"Invalid config: {list(violation_messages(e.violations))}"
            ) from e

        return cls(config)

    @classmethod
    def config_schema(cls, config_type: type[T], uri: str) -> TypeSchema:
        adapter = TypeAdapter(config_type)
        return TypeSchema(
            uri=uri,
            json_schema=Struct.from_python(adapter.json_schema(mode="validation")),
            proto_name=config_type.desc().type_name,
            mod_time=Timestamp.now(),
        )


class InstanceMux[I: BaseInstance]:
    def __init__(
        self,
        instance_type: type[I],
    ):
        super().__init__()
        self.sf = SingleFlight()
        self.instance_type = instance_type
        self.instances: dict[str, I] = {}

    def get_instance(self, ctx_or_key: RequestContext | str) -> I:
        def perform_get(mux: InstanceMux[I], key: str) -> I | None:
            return mux.instances.get(key)

        if isinstance(ctx_or_key, RequestContext):
            ctx_or_key = get_instance_key(ctx_or_key)

        inst = self.sf.call(
            fn=lambda mux: perform_get(mux, ctx_or_key),
            key=ctx_or_key,
            mux=self,
        )
        if inst is None:
            raise ConnectError(
                Code.INVALID_ARGUMENT,
                f"Instance not found for key: {ctx_or_key}",
            )
        return inst

    async def set_instance(
        self, resource_name: str, config_hash: str, config_gen: int, inst: I
    ) -> tuple[str, I]:
        async def perform_set(mux: InstanceMux[I], key: str, inst: I):
            if key not in mux.instances:
                for _key in list(mux.instances.keys()):
                    if _key.startswith(resource_name + "/"):
                        await self.del_instance(_key)
                mux.instances.update({key: inst})
            return key, inst

        key = build_instance_key(resource_name, config_hash, config_gen)
        return await self.sf.call(
            fn=lambda mux: perform_set(mux, key, inst),
            key=key,
            mux=self,
        )

    async def del_instance(self, ctx_or_key: RequestContext | str):
        async def perform_del(mux: InstanceMux[I], key: str):
            inst = mux.instances.get(key)
            if inst is not None:
                await inst.close()
                del mux.instances[key]

        if isinstance(ctx_or_key, RequestContext):
            ctx_or_key = get_instance_key(ctx_or_key)

        await self.sf.call(
            fn=lambda mux: perform_del(mux, ctx_or_key),
            key=ctx_or_key,
            mux=self,
        )
