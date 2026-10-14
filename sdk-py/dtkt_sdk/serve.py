from __future__ import annotations

import argparse
import asyncio
import signal
import sys

from pyvoy import PyvoyServer
import yaml

from .common import network

# Envoy's default per_connection_buffer_limit_bytes is 1MiB when unspecified
# (https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/listener/v3/listener.proto),
# which is small enough that a single large streaming RPC message (e.g. an
# arrow_batch batch well under typical gRPC/connect message limits) can still
# overflow it, causing Envoy itself to reject the request with a generic
# "413 Payload Too Large" before it ever reaches the ASGI app. Raising it
# here gives every integration served through pyvoy realistic headroom for
# large streaming messages instead of requiring each one to individually
# chunk payloads to fit under 1MiB.
_PER_CONNECTION_BUFFER_LIMIT_BYTES = 64 * 1024 * 1024  # 64MiB


class _Server(PyvoyServer):
    def get_envoy_config(self) -> dict:
        config = super().get_envoy_config()
        for listener in config.get("static_resources", {}).get("listeners", []):
            # UDP/QUIC listeners (HTTP/3) have their own flow-control model;
            # only TCP listeners use this field.
            if "udp_listener_config" not in listener:
                listener.setdefault(
                    "per_connection_buffer_limit_bytes",
                    _PER_CONNECTION_BUFFER_LIMIT_BYTES,
                )
        return config


async def _serve(app: str) -> int:
    host, port = network.address_from_env()
    tls = network.tls_config_from_env()

    if tls is not None:
        ca_cert, cert, key = tls
        server_cm = _Server(
            app,
            address=host,
            port=port,
            stdout=None,
            stderr=None,
            tls_cert=cert,
            tls_key=key,
            tls_ca_cert=ca_cert,
        )
    else:
        server_cm = _Server(app, address=host, port=port, stdout=None, stderr=None)

    async with server_cm as server:
        print(  # noqa: T201
            f"pyvoy listening on {server.listener_address}:{server.listener_port}",
            file=sys.stderr,
        )

        loop = asyncio.get_running_loop()
        stopping = loop.create_future()

        def request_stop() -> None:
            if not stopping.done():
                stopping.set_result(None)

        if sys.platform != "win32":
            loop.add_signal_handler(signal.SIGTERM, request_stop)
            loop.add_signal_handler(signal.SIGINT, request_stop)

        wait_task = asyncio.ensure_future(server.wait())
        done, _ = await asyncio.wait(
            {wait_task, stopping}, return_when=asyncio.FIRST_COMPLETED
        )

        if wait_task in done:
            return wait_task.result() or 0

        await server.stop()
        return (await wait_task) or 0


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Run a DataKit integration app behind pyvoy, "
        "binding and configuring TLS from the deployment's environment."
    )
    parser.add_argument("app", help="the app to run as 'module:attr' or just 'module'")
    args = parser.parse_args()

    sys.exit(asyncio.run(_serve(args.app)))


if __name__ == "__main__":
    main()
