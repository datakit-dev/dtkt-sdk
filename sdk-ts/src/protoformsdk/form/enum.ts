import { type DescField } from "@bufbuild/protobuf";

import { Element } from "./element";
import { BaseField } from "./field";
import { type Message } from "./message";

export class EnumField extends BaseField {
  parent: Message;
  desc: DescField & {
    fieldKind: "enum";
  };

  value: number;
  elem: Element;

  constructor(parent: Message, name: string) {
    super();

    const desc = parent.value.desc.field[name];
    if (!desc) throw new Error(`field not found in message: ${name}`);
    if (desc.fieldKind !== "enum") throw new Error(`expected enum field: ${name}, got: ${desc.fieldKind}`);

    this.parent = parent;
    this.desc = desc;
    this.value = parent.value.get(desc);
    this.elem = new Element(desc);
  }

  getValue(): number {
    return this.parent.value.get(this.desc);
  }

  setValue(value: number) {
    this.parent.value.set(this.desc, value);
  }

  parseValue(value: string) {
    for (const enumValue of this.desc.enum.values) {
      if (enumValue.name === value) {
        this.setValue(enumValue.number);
        return;
      }
    }
  }

  valueString(): string {
    return this.desc.enum.value[this.getValue()]?.name ?? "";
  }
}
