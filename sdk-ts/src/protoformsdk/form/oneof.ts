import { type DescField, type DescOneof } from "@bufbuild/protobuf";
import { type ReflectMessage } from "@bufbuild/protobuf/reflect";

import { Element } from "./element";
import { BaseField, type Field, FieldGroup } from "./field";
import { getField } from "./helper";
import { Message } from "./message";

type OneOfFieldDesc = DescField & {
  oneof: DescOneof;
};
type OneOfValue = string | number | bigint | boolean | ReflectMessage | Uint8Array;

export class OneOfField extends BaseField {
  parent: Message;
  oneof: DescOneof;
  desc: OneOfFieldDesc;
  private group: FieldGroup;

  elem: Element;

  constructor(parent: Message, oneof: DescOneof) {
    super();

    this.parent = parent;
    this.oneof = oneof;

    const desc = parent.value.oneofCase(oneof) ?? oneof.fields[0];
    if (!desc) throw new Error(`invalid one of: ${oneof.name}`);

    this.desc = desc as OneOfFieldDesc;
    this.elem = new Element(this.desc);

    const fields: Field[] = [];
    for (const desc of this.oneof.fields) {
      fields.push(getField(this.parent, desc));
    }
    this.group = new FieldGroup(fields);
  }

  isSet() {
    return !!this.parent.value.oneofCase(this.oneof);
  }

  fieldGroup(): FieldGroup {
    const field = this.getField();
    if (field instanceof Message) {
      return field.fieldGroup();
    }
    return new FieldGroup([field]);
  }

  getField(): Field {
    const field = this.group.fields.find((field) => field.desc === this.desc) ?? this.group.fields[0];
    if (!field) throw new Error(`invalid one of "${this.oneof.name}": field not found`);
    return field;
  }

  setField(name: string) {
    const field = this.group.fields.find((field) => field.desc.name === name);
    if (field && field.desc != this.desc) {
      this.desc = field.desc as OneOfFieldDesc;
      this.elem = new Element(this.desc);
    }
  }

  getValue(): OneOfValue {
    return this.parent.value.get(this.desc);
  }

  setValue(value: OneOfValue) {
    this.parent.value.set(this.desc, value);
  }
}
