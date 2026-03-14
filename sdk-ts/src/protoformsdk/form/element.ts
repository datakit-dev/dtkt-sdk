import { create, type DescField, getOption, hasOption } from "@bufbuild/protobuf";

import { type ConfirmElement, field as fieldExt, type FieldElement, FieldElementSchema, type FileElement, type InputElement, type MultiSelectElement, type SelectElement } from "../../proto/dtkt/protoform/v1beta1/protoform_pb";

import { getFieldDescription, getFieldTitle } from "./helper";

export type ElementType = ConfirmElement | InputElement | FileElement | SelectElement | MultiSelectElement;

export class Element {
  proto: FieldElement;
  field: DescField;

  constructor(desc: DescField) {
    if (hasOption(desc, fieldExt)) {
      this.proto = getOption(desc, fieldExt);
    } else {
      this.proto = create(FieldElementSchema);
    }

    this.field = desc;
  }

  isValid(): boolean {
    return this.proto.type.value !== undefined;
  }

  isHidden(): boolean {
    return this.proto.hidden ?? false;
  }

  getType(): ElementType | undefined {
    return this.proto.type.value;
  }

  getTitle(): string {
    return this.proto.title ?? getFieldTitle(this.field);
  }

  getDescription(): string {
    return this.proto.description ?? getFieldDescription(this.field);
  }
}
