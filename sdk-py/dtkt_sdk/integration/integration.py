import os
import logging
import tempfile
from pathlib import Path
import hashlib
import nats
from nats.aio.client import Client as Broker

from protobuf import (
    Message,
)

from connectrpc.request import RequestContext
from connectrpc.errors import ConnectError
from connectrpc.code import Code

from dtkt_sdk.env import var
from dtkt_sdk.gen.dtkt.core.v1.spec_pb import PackageSpec
from dtkt_sdk.gen.dtkt.base.v1beta1.services_connect import (
    BaseService,
)
from dtkt_sdk.common.network import type_uri

from dtkt_sdk.gen.dtkt.base.v1beta1.messages_pb import (
    GetPackageRequest,
    GetPackageResponse,
    CheckConfigRequest,
    CheckConfigResponse,
    ListTypesRequest,
    ListTypesResponse,
    GetTypeRequest,
    GetTypeResponse,
    CloseRequest,
    CloseResponse,
    CheckAuthRequest,
    CheckAuthResponse,
)
from dtkt_sdk.integration.instance import (
    BaseInstance,
    InstanceMux,
    build_instance_key,
)

from dtkt_sdk.integration.util import (
    resolve_package_spec,
)

from dtkt_sdk.integration.registry import TypeRegistry

logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")

logger = logging.getLogger(__name__)


class Integration[T: Message, I: BaseInstance](BaseService):
    def __init__(
        self,
        package_spec: bytes | str | PackageSpec,
        config_type: type[T],
        instance_type: type[I],
        custom_types: list[type] = [],
    ):
        self.package = resolve_package_spec(package_spec)
        self.config_schema = instance_type.config_schema(
            config_type, type_uri("Config")
        )
        self.types = TypeRegistry(list[type]([config_type]) + custom_types)
        self.mux = InstanceMux(instance_type)

        self.config_type = config_type
        self.instance_type = instance_type
        self.root = os.getenv(var.DATA_ROOT, "")
        self.broker: Broker | None = None

        if not self.root:
            self.root = tempfile.gettempdir()
            if self.package.identity and self.package.identity.name:
                self.root = str(
                    Path(self.root)
                    / f"{self.package.identity.name.lower()}@{self.package.identity.version.lower()}"
                )

        Path(self.root).mkdir(mode=0o777, parents=True, exist_ok=True)

        logger.info(f"Integration initialized with data root: {self.root}")

    async def get_broker(self):
        if self.broker:
            return self.broker

        nats_addr = os.getenv(var.NATS_ADDRESS, "")
        if not nats_addr:
            return None

        logger.info(f"Connecting to nats server: {nats_addr}")

        self.broker = await nats.connect(nats_addr)
        return self.broker

    async def check_config(
        self,
        req: CheckConfigRequest,
        ctx: RequestContext[CheckConfigRequest, CheckConfigResponse],
        /,
    ) -> CheckConfigResponse:
        if not req.config_data:
            raise ConnectError(Code.INVALID_ARGUMENT, "Config data is required")

        key = build_instance_key(req.connection, req.config_hash, req.config_gen)
        try:
            inst = self.mux.get_instance(key)
            if inst:
                return CheckConfigResponse()
        except ConnectError:
            pass

        try:
            # Keyed on the connection name alone (not the mux key, which also bakes in
            # config_hash/config_gen) so the data root stays stable across config/reconcile
            # changes, matching the Go SDK's util.HashShort(check.Connection) behavior.
            hash = hashlib.sha256(req.connection.encode()).hexdigest()[:12]
            root = str(Path(self.root) / hash)

            broker = await self.get_broker()
            if broker:
                keyval = await broker.jetstream().create_key_value(bucket=hash)
            else:
                keyval = None

            inst = await self.instance_type.from_config(
                config_type=self.config_type,
                any_msg=req.config_data,
                name=req.connection,
                root=root,
                keyval=keyval,
            )
        except ValueError as e:
            raise ConnectError(Code.INVALID_ARGUMENT, str(e)) from e

        await self.mux.set_instance(
            req.connection,
            req.config_hash,
            req.config_gen,
            inst,
        )

        return CheckConfigResponse()

    async def check_auth(
        self,
        request: CheckAuthRequest,
        ctx: RequestContext[CheckAuthRequest, CheckAuthResponse],
        /,
    ) -> CheckAuthResponse:
        inst = self.mux.get_instance(ctx)
        if not inst:
            raise ConnectError(Code.INVALID_ARGUMENT, "Instance not found")

        resp = await inst.check_auth(request, ctx)
        if resp:
            return resp

        return CheckAuthResponse()

    async def get_package(
        self,
        req: GetPackageRequest,
        ctx: RequestContext[GetPackageRequest, GetPackageResponse],
        /,
    ) -> GetPackageResponse:
        return GetPackageResponse(
            package=self.package,
            config_schema=self.config_schema,
        )

    async def get_type(
        self,
        request: GetTypeRequest,
        ctx: RequestContext[GetTypeRequest, GetTypeResponse],
        /,
    ) -> GetTypeResponse:
        if (
            request.name == self.config_type.desc().name
            or request.name == self.config_type.desc().type_name
        ):
            return GetTypeResponse(
                type=self.config_schema,
            )

        _type = self.types.get_type(request.name)
        if _type:
            return GetTypeResponse(
                type=_type,
            )

        raise ConnectError(
            Code.NOT_FOUND,
            f"Type not found for request: {request.name}",
        )

    async def list_types(
        self,
        request: ListTypesRequest,
        ctx: RequestContext[ListTypesRequest, ListTypesResponse],
        /,
    ) -> ListTypesResponse:
        _types, next_page_token = self.types.list_types(
            request.page_size, request.page_token
        )
        return ListTypesResponse(types=_types, next_page_token=next_page_token)

    async def close(
        self, request: CloseRequest, ctx: RequestContext[CloseRequest, CloseResponse], /
    ) -> CloseResponse:
        await self.mux.del_instance(ctx)
        return CloseResponse()
