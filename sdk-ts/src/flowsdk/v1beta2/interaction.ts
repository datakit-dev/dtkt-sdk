import { create, type DescField, type DescMessage, getOption, hasOption, type Message } from "@bufbuild/protobuf";

import { Interaction_ConfirmBindingSchema, Interaction_FileBindingSchema, type Interaction_Input, Interaction_InputBindingSchema, Interaction_MultiSelectBindingSchema, Interaction_SelectBindingSchema } from "../../proto/dtkt/flow/v1beta2/spec_pb";
import { field as fieldExt, type FieldElement, FieldElementSchema } from "../../proto/dtkt/protoform/v1beta1/protoform_pb";

export type InteractionBindingResult = {
  /**
   * The protoform FieldElement driving the widget. Lifted from the
   * binding's `value` field's annotation, then overlayed with the
   * input's title / description and any element-specific options
   * the spec carried (ConfirmElement labels, InputElement multiline
   * flag, Select/MultiSelect choices, etc.).
   */
  element: FieldElement;
  /**
   * The empty binding proto (ConfirmBinding / InputBinding / FileBinding
   * / SelectBinding / MultiSelectBinding) whose `value` field is what
   * the user fills in.
   */
  binding: Message;
  /**
   * Descriptor for the binding -- consumers `reflect(desc, binding)`
   * for protoform rendering or `toJson(desc, binding)` when packing
   * the response struct.
   */
  desc: DescMessage;
};

/**
 * Returns the protoform FieldElement and binding proto for an
 * interaction input. Mirrors `GetInteractionBinding` in
 * `sdk-go/flowsdk/v1beta2/interaction.go`.
 *
 * The binding's `value` field carries a `(dtkt.protoform.v1beta1.field)`
 * extension declaring the widget kind (confirm / input / file / select /
 * multi_select). We lift that annotation, overlay the input's title /
 * description, and merge any element-specific options the spec set
 * (button labels for Confirm, multiline for Input, choices for Select /
 * MultiSelect). The resulting FieldElement drives the widget; the
 * empty binding is the message the user populates and we send back.
 *
 * Returns `undefined` if the element oneof is unset (which can't happen
 * for a well-formed event but defends against forward-compat changes).
 */
export function getInteractionBinding(input: Interaction_Input): InteractionBindingResult | undefined {
  let binding: Message;
  let desc: DescMessage;
  switch (input.element.case) {
    case "confirm":
      binding = create(Interaction_ConfirmBindingSchema);
      desc = Interaction_ConfirmBindingSchema;
      break;
    case "input":
      binding = create(Interaction_InputBindingSchema);
      desc = Interaction_InputBindingSchema;
      break;
    case "file":
      binding = create(Interaction_FileBindingSchema);
      desc = Interaction_FileBindingSchema;
      break;
    case "select":
      binding = create(Interaction_SelectBindingSchema);
      desc = Interaction_SelectBindingSchema;
      break;
    case "multiSelect":
      binding = create(Interaction_MultiSelectBindingSchema);
      desc = Interaction_MultiSelectBindingSchema;
      break;
    default:
      return undefined;
  }

  const valueField: DescField | undefined = desc.field.value;
  let element: FieldElement;
  if (valueField && hasOption(valueField, fieldExt)) {
    element = getOption(valueField, fieldExt);
  } else {
    element = create(FieldElementSchema);
  }

  element.title = input.title;
  if (input.description !== undefined) {
    element.description = input.description;
  }

  // Merge element-specific options from the spec into the FieldElement
  // so the widget receives the Confirm/Input/Select/MultiSelect
  // overrides authored on the interaction.
  switch (input.element.case) {
    case "confirm":
      if (element.type.case === "confirm") {
        element.type = { case: "confirm", value: { ...element.type.value, ...input.element.value } };
      } else {
        element.type = { case: "confirm", value: input.element.value };
      }
      break;
    case "input":
      element.type = { case: "input", value: input.element.value };
      break;
    case "select":
      element.type = { case: "select", value: input.element.value };
      break;
    case "multiSelect":
      element.type = { case: "multiSelect", value: input.element.value };
      break;
    case "file":
      element.type = { case: "file", value: input.element.value };
      break;
  }

  return { element, binding, desc };
}
