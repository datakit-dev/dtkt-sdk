import { type DescField } from "@bufbuild/protobuf";
import { type ReflectMap } from "@bufbuild/protobuf/reflect";

import { Element } from "./element";
import { BaseField } from "./field";
import { type Message } from "./message";

export class MapField extends BaseField {
  parent: Message;
  desc: DescField & {
    fieldKind: "map";
  };

  value: ReflectMap;
  elem: Element;

  constructor(parent: Message, name: string) {
    super();

    const desc = parent.value.desc.field[name];
    if (!desc) throw new Error(`field not found in message: ${name}`);
    if (desc.fieldKind !== "map") throw new Error(`expected map field: ${name}, got: ${desc.fieldKind}`);

    this.parent = parent;
    this.desc = desc;
    this.value = parent.value.get(desc);
    this.elem = new Element(desc);
  }

  getValue(): ReflectMap {
    return this.parent.value.get(this.desc);
  }

  setValue(value: ReflectMap) {
    this.parent.value.set(this.desc, value);
  }
}
