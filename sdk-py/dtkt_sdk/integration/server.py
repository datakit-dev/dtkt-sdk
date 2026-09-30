from typing import Protocol
import logging
from starlette.applications import Starlette
from starlette.routing import Mount

from grpcreflect import mount_reflection_handlers
from grpchealth import mount_health_handler, Checker
from protobuf import Message, DescService

from dtkt_sdk.integration import Integration
from dtkt_sdk.integration.instance import BaseInstance
from dtkt_sdk.gen.dtkt.base.v1beta1.services_connect import (
    BaseServiceASGIApplication,
)
from connectrpc.server import ConnectASGIApplication

from typing import Any
from collections.abc import Callable

from .interceptor import Interceptor
from .instance import InstanceMux

logger = logging.getLogger(__name__)


class Server[T: Message, I: BaseInstance]:
    def __init__(
        self,
        integration: Integration[T, I],
        services: list[Callable[[InstanceMux[I]], ConnectASGIApplication[Any]]] = [],
        *args,
        **kwargs,
    ):
        intercept = Interceptor[I](integration.mux)
        apps: dict[str, ConnectASGIApplication[Any]] = {}

        for service in services:
            app = service(integration.mux)
            if app.path in apps:
                raise ValueError(f"Service already registered at path: {app.path}")

            logger.info("Registering service at path: %s", app.path)

            app._interceptors = [
                intercept,
                *app._interceptors,
            ]
            apps[app.path] = app

        app = BaseServiceASGIApplication(service=integration, interceptors=[intercept])
        apps[app.path] = app

        # service_names = set()

        self.app = Starlette(
            *args,
            routes=[
                Mount(app.path, app=app)
                for app in mount_reflection_handlers(
                    *apps.values(), registry=integration.types.registry
                )
            ],
            **kwargs,
        )
        self.intgr = integration

    async def __call__(self, scope: Any, receive: Any, send: Any) -> None:
        await self.app(scope, receive, send)
