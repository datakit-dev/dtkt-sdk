import { type DescField } from "@bufbuild/protobuf";
import { type ReflectList } from "@bufbuild/protobuf/reflect";

import { Element } from "./element";
import { BaseField } from "./field";
import { type Message } from "./message";

export class ListField extends BaseField {
  parent: Message;
  desc: DescField & {
    fieldKind: "list";
  };

  value: ReflectList;
  elem: Element;

  constructor(parent: Message, name: string) {
    super();

    const desc = parent.value.desc.field[name];
    if (!desc) throw new Error(`field not found in message: ${name}`);
    if (desc.fieldKind !== "list") throw new Error(`expected list field: ${name}, got: ${desc.fieldKind}`);

    this.parent = parent;
    this.desc = desc;
    this.value = parent.value.get(desc);
    this.elem = new Element(desc);
  }

  getValue(): ReflectList {
    return this.parent.value.get(this.desc);
  }

  setValue(value: ReflectList) {
    this.parent.value.set(this.desc, value);
  }
}
