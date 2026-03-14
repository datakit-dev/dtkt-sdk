import { type DescField } from "@bufbuild/protobuf";
import { type Violation } from "@bufbuild/protovalidate";

import { type Element } from "./element";
import { type Message } from "./message";

export type Field = {
  parent: Message;
  desc: DescField;
  elem: Element;
  error?: Violation;
};

export class BaseField {
  error: Violation | undefined;
}

export class FieldGroup {
  fields: Field[];

  constructor(fields: Field[]) {
    this.fields = fields;
  }

  * [Symbol.iterator]() {
    for (const field of this.fields) {
      yield field;
    }
  }
}
