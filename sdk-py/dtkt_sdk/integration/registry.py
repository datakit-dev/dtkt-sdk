from collections.abc import Mapping, Sequence, MutableSequence, MutableMapping
from typing import get_origin, get_args
from pydantic import TypeAdapter
import base64
from dataclasses import is_dataclass

from protobuf.wkt import Struct, Timestamp, Value
from protobuf import (
    Registry,
    Message,
    DescMessage,
)

from dtkt_sdk.common.network import type_uri

from dtkt_sdk.gen.dtkt.shared.v1beta1.messages_pb import TypeSchema, PageToken


class TypeRegistry:
    def __init__(self, types: list[type] = []):
        self.registry = Registry()
        self._cache = dict[str, TypeSchema]()

        for typ in types:
            self.register_type(typ)

    def register_type(self, typ: type):
        if issubclass(typ, Message):
            desc = typ.desc()
            type_name = desc.type_name
            proto_name = desc.type_name

            if not self.registry.message(type_name):
                self.registry.add(typ)
        else:
            value = Value.from_python(self.normalize_type(typ))
            desc = value.desc()
            type_name = typ.__name__
            proto_name = desc.type_name

        uri = type_uri(type_name)
        schema = TypeAdapter(typ).json_schema(mode="validation")
        schema.update({"$id": uri})

        type_schema = TypeSchema(
            uri=uri,
            json_schema=Struct.from_python(schema),
            proto_name=proto_name,
            mod_time=Timestamp.now(),
        )

        self._cache[type_name] = type_schema
        return type_schema

    def get_type(self, type_name: str) -> TypeSchema | None:
        if type_name in self._cache:
            return self._cache[type_name]

        desc = self.registry.message(type_name)
        if desc:
            return self.register_type(desc.type)

    def list_types(self, page_size: int = 0, page_token: str = ""):
        token = (
            PageToken.from_binary(base64.b64decode(page_token)) if page_token else None
        )
        types = []
        last_index = 0
        for desc in self.registry:
            if isinstance(desc, DescMessage):
                type_schema = self.get_type(desc.type_name)
                if type_schema:
                    last_index += 1
                    if token and (
                        last_index <= token.id
                        or (
                            type_schema.mod_time
                            and token.time
                            and type_schema.mod_time.to_datetime()
                            < token.time.to_datetime()
                        )
                    ):
                        continue
                    if page_size > 0 and len(types) == page_size:
                        next_token = PageToken(
                            id=last_index - 1, time=types[-1].mod_time
                        )
                        return types, base64.b64encode(next_token.to_binary()).decode()
                    types.append(type_schema)
        return types, ""

    @classmethod
    def normalize_type(cls, typ: type):
        """Normalize a type into a scalar, list, or dict representation."""

        if is_dataclass(typ):
            return {}

        if typ is bool or typ is int or typ is float or typ is str:
            return typ()

        origin = get_origin(typ)

        if origin in (list, tuple, set, frozenset, Sequence, MutableSequence):
            args = get_args(typ)
            if args:
                return [cls.normalize_type(args[0])]
            return []

        if origin in (dict, Mapping, MutableMapping):
            args = get_args(typ)
            if len(args) == 2:
                return {}
            return {}

        if isinstance(typ, type) and issubclass(typ, (bool, int, float, str)):
            return typ()

        if isinstance(typ, type) and issubclass(typ, Sequence):
            return []

        if isinstance(typ, type) and issubclass(typ, Mapping):
            return {}

        raise TypeError(f"Cannot normalize type: {typ!r}")
