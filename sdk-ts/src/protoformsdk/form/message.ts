import { type DescField, type DescOneof, type Message as ProtoMessage } from "@bufbuild/protobuf";
import { reflect, type ReflectMessage } from "@bufbuild/protobuf/reflect";
import { type Validator } from "@bufbuild/protovalidate";

import { Element } from "./element";
import { BaseField, type Field, FieldGroup } from "./field";
import { getField } from "./helper";
import { OneOfField } from "./oneof";

export class Message extends BaseField {
  value: ReflectMessage;
  private group: FieldGroup | undefined;

  constructor(value: ReflectMessage) {
    super();
    this.value = value;
  }

  fieldGroup(filter?: (field: Field) => boolean): FieldGroup {
    if (this.group) {
      return this.group;
    }

    const fields: Field[] = [];
    const oneOfs: DescOneof[] = [];
    for (const desc of this.value.fields) {
      const oneOf = desc.oneof;
      if (oneOf) {
        if (!oneOfs.find((o) => o.name === oneOf.name)) {
          oneOfs.push(oneOf);
          fields.push(new OneOfField(this, oneOf));
        }
        break;
      }
      fields.push(getField(this, desc));
    }

    this.group = new FieldGroup(fields.filter((field: Field) => {
      return filter?.(field) ?? true;
    }));

    return this.group;
  }

  validate(validator: Validator) {
    for (const field of this.fieldGroup()) {
      field.error = undefined;
    }

    const result = validator.validate(this.value.desc, this.value.message);
    if (result.kind !== "valid") {
      for (const violation of result.violations ?? []) {
        for (const field of this.fieldGroup()) {
          const fieldViolation = violation.field[0];
          if (fieldViolation?.kind === "field" && fieldViolation.name == field.desc.name) {
            field.error = violation;
          }
        }
      }
    }
    return result;
  }

  get(): ProtoMessage {
    return this.value.message;
  }

  set(value: ProtoMessage) {
    this.value = reflect(this.value.desc, value);
  }
};

export class MessageField extends Message {
  parent: Message;
  desc: DescField & {
    fieldKind: "message";
  };

  elem: Element;

  constructor(parent: Message, name: string) {
    const desc = parent.value.desc.field[name];
    if (!desc) throw new Error(`field not found in message: ${name}`);
    if (desc.fieldKind !== "message") throw new Error(`expected message field: ${name}, got: ${desc.fieldKind}`);

    super(parent.value.get(desc));

    this.parent = parent;
    this.desc = desc;
    this.elem = new Element(desc);
  }

  isSet(): boolean {
    return this.parent.value.isSet(this.desc);
  }

  get(): ProtoMessage {
    return this.parent.value.get(this.desc).message;
  }

  set(value: ProtoMessage) {
    super.set(value);
    this.parent.value.set(this.desc, this.value);
  }
}
