from __future__ import annotations

import base64
import os

from dtkt_sdk.gen.dtkt.core.v1.common_pb import TLSCertBundle

from dtkt_sdk.env import var


def tls_config_from_env() -> tuple[bytes, bytes, bytes] | None:
    """Returns (ca_cert, cert, key) PEM bytes from DTKT_TLS_CA_CERT/DTKT_TLS_BUNDLE.

    Returns None if TLS is not configured for this deployment.
    """
    ca_cert_pem = os.environ.get(var.TLS_CA_CERT, "")
    bundle_data = os.environ.get(var.TLS_BUNDLE, "")
    if not ca_cert_pem or not bundle_data:
        return None

    try:
        bundle = TLSCertBundle.from_binary(base64.b64decode(bundle_data))
    except Exception as e:
        raise ValueError("Invalid TLS bundle JSON") from e

    return (
        ca_cert_pem.encode(),
        bundle.cert_pem,
        bundle.key_pem,
    )


def address_from_env() -> tuple[str, int]:
    """Returns the (host, port) to bind to, derived from DTKT_ADDRESS.

    An empty host means bind to all interfaces, matching how the daemon
    assigns addresses for both native and Docker-hosted deployments (Docker
    port mappings target a specific container port, not a specific host
    within the container's network namespace).
    """
    addr = os.environ.get(var.ADDRESS, "")
    host, _, port_str = addr.rpartition(":")
    port = int(port_str) if port_str else 0
    return host or "0.0.0.0", port


def address_url(network_type: str, target: str) -> str:
    """Mirrors sdk-go's network.Address.URL() network-type-to-scheme mapping."""
    if network_type == "tcp":
        return f"http://{target}"
    if network_type == "unix":
        return f"unix://localhost{target if target.startswith('/') else '/' + target}"
    return f"{network_type}://{target}"


def type_base_uri() -> str:
    """Returns the base URI type schemas are rooted under.

    Mirrors sdk-go's integration.go: prefers the daemon's context address
    joined with the deployment name when running as a deployment, falling
    back to this integration's own listen address otherwise.
    """
    context_addr = os.environ.get(var.CONTEXT_ADDRESS, "")
    deploy_name = os.environ.get(var.DEPLOY_NAME, "")

    if context_addr and "/deployments/" in deploy_name:
        network_type, _, target = context_addr.partition("://")
        return f"{address_url(network_type, target)}/schemas/{deploy_name}/types"

    network_type = os.environ.get(var.NETWORK, "mem")
    target = os.environ.get(var.ADDRESS, "")
    return f"{address_url(network_type, target)}/types"


def type_uri(name: str) -> str:
    """Returns the full URI for a named type schema (e.g. "Config")."""
    return f"{type_base_uri()}/{name}"
