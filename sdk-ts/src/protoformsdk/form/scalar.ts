import { type DescField, ScalarType } from "@bufbuild/protobuf";
import { type ScalarValue } from "@bufbuild/protobuf/reflect";

import { Element } from "./element";
import { BaseField } from "./field";
import { type Message } from "./message";

const byteEncoder = new TextEncoder();

export class ScalarField extends BaseField {
  parent: Message;
  desc: DescField & {
    fieldKind: "scalar";
  };

  elem: Element;

  constructor(parent: Message, name: string) {
    super();

    const desc = parent.value.desc.field[name];
    if (!desc) throw new Error(`field not found in message: ${name}`);
    if (desc.fieldKind !== "scalar") throw new Error(`expected scalar field: ${name}, got: ${desc.fieldKind}`);

    this.parent = parent;
    this.desc = desc;
    this.elem = new Element(desc);
  }

  getValue(): ScalarValue {
    return this.parent.value.get(this.desc);
  }

  setValue(value: ScalarValue) {
    this.parent.value.set(this.desc, value);
  }

  parseValue(str: string) {
    this.setValue(parseScalarValue(this.desc.scalar, str, this.desc.longAsString));
  }

  stringValue(): string {
    return String(this.getValue());
  }
}

export function parseScalarValue(scalar: ScalarType, str: string, longAsString = true): ScalarValue {
  switch (scalar) {
    case ScalarType.BOOL:
      return Boolean(str);
    case ScalarType.BYTES:
      return byteEncoder.encode(str);
    case ScalarType.STRING:
      return str;
    case ScalarType.DOUBLE:
    case ScalarType.FLOAT:
      return parseFloat(str);
    case ScalarType.INT32:
    case ScalarType.FIXED32:
    case ScalarType.UINT32:
    case ScalarType.SFIXED32:
    case ScalarType.SINT32:
      return parseInt(str);
    case ScalarType.INT64:
    case ScalarType.UINT64:
    case ScalarType.SINT64:
    case ScalarType.FIXED64:
    case ScalarType.SFIXED64:
      if (longAsString) {
        return str;
      }
      return BigInt(str);
  }
}
