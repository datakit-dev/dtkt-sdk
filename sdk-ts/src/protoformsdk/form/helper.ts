import { type DescField, ScalarType } from "@bufbuild/protobuf";

import { getDescription } from "./description";
import { EnumField } from "./enum";
import { ListField } from "./list";
import { MapField } from "./map";
import { type Message, MessageField } from "./message";
import { ScalarField } from "./scalar";

export function getField(parent: Message, desc: DescField) {
  switch (desc.fieldKind) {
    case "enum":
      return new EnumField(parent, desc.localName);
    case "list":
      return new ListField(parent, desc.localName);
    case "map":
      return new MapField(parent, desc.localName);
    case "message":
      return new MessageField(parent, desc.localName);
    case "scalar":
      return new ScalarField(parent, desc.localName);
  }
}

export function getFieldTitle(desc: DescField): string {
  return desc.jsonName;
}

export function getFieldDescription(desc: DescField): string {
  let description = getDescription(desc);
  if (description) {
    return description;
  }

  if (desc.fieldKind == "map") {
    let mapVal: string;
    if (desc.enum) {
      mapVal = desc.enum.typeName;
    } else if (desc.message) {
      mapVal = desc.message.typeName;
    } else {
      mapVal = getScalarTypeName(desc.scalar);
    }
    description = `map<${getScalarTypeName(desc.mapKey)}, ${mapVal}>`;
  } else if (desc.message) {
    description = desc.message.typeName;
  } else if (desc.enum) {
    description = desc.enum.typeName;
  } else {
    description = getScalarTypeName(desc.scalar);
  }

  if (desc.fieldKind == "list") {
    description = `list<${description}>`;
  }

  return description;
}

export function getScalarTypeName(scalarType: ScalarType): string {
  switch (scalarType) {
    case ScalarType.DOUBLE:
      return "double";
    case ScalarType.FLOAT:
      return "float";
    case ScalarType.INT64:
      return "int64";
    case ScalarType.UINT64:
      return "uint64";
    case ScalarType.INT32:
      return "int32";
    case ScalarType.FIXED64:
      return "fixed64";
    case ScalarType.FIXED32:
      return "fixed32";
    case ScalarType.BOOL:
      return "bool";
    case ScalarType.STRING:
      return "string";
    case ScalarType.BYTES:
      return "bytes";
    case ScalarType.UINT32:
      return "uint32";
    case ScalarType.SFIXED32:
      return "sfixed32";
    case ScalarType.SFIXED64:
      return "sfixed64";
    case ScalarType.SINT32:
      return "sint32";
    case ScalarType.SINT64:
      return "sint64";
  }
}
