from __future__ import annotations

import argparse
import asyncio
import signal
import sys

from pyvoy import PyvoyServer

from .common import network


async def _serve(app: str) -> int:
    host, port = network.address_from_env()
    tls = network.tls_config_from_env()

    if tls is not None:
        ca_cert, cert, key = tls
        server_cm = PyvoyServer(
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
        server_cm = PyvoyServer(app, address=host, port=port, stdout=None, stderr=None)

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
