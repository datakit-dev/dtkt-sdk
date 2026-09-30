import yaml
import json
from dtkt_sdk.gen.dtkt.core.v1.spec_pb import PackageSpec
from dtkt_sdk.gen.dtkt.shared.v1beta1.messages_pb import Package
from protovalidate import validate, ValidationError
from protobuf import Oneof
from dtkt_sdk.common.validate import violation_messages
import functools
import time
from typing import Callable, Coroutine


def resolve_package_spec(spec: bytes | str | PackageSpec):
    if isinstance(spec, (bytes, str)):
        try:
            spec = PackageSpec.from_json(json.dumps(yaml.safe_load(spec)))
        except Exception:
            spec = PackageSpec.from_json(spec)

    if spec.spec:
        match spec.api_version:
            case "v1beta1":
                spec.version = Oneof("v1beta1", Package.from_json(spec.spec.to_json()))
            case _:
                raise ValueError(f"Unsupported api_version: {spec.api_version}")

    try:
        validate(spec)
    except ValidationError as e:
        raise ValueError(
            f"Invalid config: {list(violation_messages(e.violations))}"
        ) from e

    if not spec.version:
        raise ValueError("Spec version is not set")

    return spec.version.value


def measure_time[R, **P](func: Callable[P, R]) -> Callable[P, tuple[R, float]]:
    @functools.wraps(func)
    def wrapper(*args: P.args, **kwargs: P.kwargs) -> tuple[R, float]:
        start_time = time.perf_counter()
        result = func(*args, **kwargs)
        elapsed_time = time.perf_counter() - start_time
        return result, elapsed_time

    return wrapper


def measure_time_async[R, **P](
    func: Callable[P, Coroutine[None, None, R]],
) -> Callable[P, Coroutine[None, None, tuple[R, float]]]:

    @functools.wraps(func)
    async def wrapper(*args: P.args, **kwargs: P.kwargs) -> tuple[R, float]:
        start_time = time.perf_counter()
        result = await func(*args, **kwargs)
        elapsed_time = time.perf_counter() - start_time
        return result, elapsed_time

    return wrapper
