import { create, type DescField, type DescMessage, fromJson, getOption, hasOption, type Message, type Registry } from "@bufbuild/protobuf";

import { type Input, Input_BoolBindingSchema, Input_BytesBindingSchema, Input_DoubleBindingSchema, Input_FloatBindingSchema, Input_Int32BindingSchema, Input_Int64BindingSchema, Input_ListBindingSchema, Input_MapBindingSchema, Input_StringBindingSchema, Input_Uint32BindingSchema, Input_Uint64BindingSchema } from "../../proto/dtkt/flow/v1beta2/spec_pb";
import { field as fieldExt, type FieldElement, FieldElementSchema } from "../../proto/dtkt/protoform/v1beta1/protoform_pb";

export type InputBindingResult = {
  /**
   * The protoform FieldElement driving the widget. For wrapper
   * bindings (scalar / list / map) it's the annotation lifted from
   * the binding's `value` field. For typed-message bindings it's a
   * synthetic element holding the input id as title -- per-field
   * annotations on the typed message drive the actual widgets.
   */
  element: FieldElement;
  /**
   * The binding proto, pre-populated from the input's default when
   * set. For scalar / list / map inputs it's an Input_*Binding
   * wrapper. For Message inputs it's a fresh instance of the
   * declared FQN type (its own fields drive the form).
   */
  binding: Message;
  /**
   * Descriptor for the returned binding -- consumers use this to
   * `reflect(desc, msg)` for protoform rendering or `anyPack(desc,
   * msg)` for wire encoding.
   */
  desc: DescMessage;
};

/**
 * Returns the protoform FieldElement and binding proto for a flow
 * Input. Tuiform / protoformsdk consumes the binding's
 * protoform-annotated fields and renders the matching widgets; the
 * binding is pre-populated from the input's typed default when set.
 *
 * For scalar / list / map inputs the binding is one of Input_*Binding
 * (a wrapper whose `value` field carries the FieldElement annotation).
 * For Message inputs the registry looks up the declared FQN and the
 * returned binding is a fresh instance of *that* message -- its own
 * protoform-annotated fields drive the form. The Message default
 * (a Struct) populates the typed message via a protojson round-trip.
 *
 * Mirrors `GetInputBinding` in `sdk-go/flowsdk/v1beta2/input.go`.
 *
 * Returns `undefined` when the input's type is unrecognised (oneof
 * unset), the message FQN can't be resolved by the registry, or the
 * default can't be coerced into the resolved type.
 */
export function getInputBinding(input: Input, registry: Registry): InputBindingResult | undefined {
  const id = input.id;

  switch (input.type.case) {
    case "bool": {
      const binding = create(Input_BoolBindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_BoolBindingSchema, id);
    }
    case "bytes": {
      const binding = create(Input_BytesBindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_BytesBindingSchema, id);
    }
    case "double": {
      const binding = create(Input_DoubleBindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_DoubleBindingSchema, id);
    }
    case "float": {
      const binding = create(Input_FloatBindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_FloatBindingSchema, id);
    }
    case "int64": {
      const binding = create(Input_Int64BindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_Int64BindingSchema, id);
    }
    case "uint64": {
      const binding = create(Input_Uint64BindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_Uint64BindingSchema, id);
    }
    case "int32": {
      const binding = create(Input_Int32BindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_Int32BindingSchema, id);
    }
    case "uint32": {
      const binding = create(Input_Uint32BindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_Uint32BindingSchema, id);
    }
    case "string": {
      const binding = create(Input_StringBindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_StringBindingSchema, id);
    }
    case "list": {
      const binding = create(Input_ListBindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_ListBindingSchema, id);
    }
    case "map": {
      const binding = create(Input_MapBindingSchema);
      if (input.type.value.default !== undefined) {
        binding.value = input.type.value.default;
      }
      return wrapperResult(binding, Input_MapBindingSchema, id);
    }
    case "message": {
      const desc = registry.getMessage(input.type.value.type);
      if (!desc) {
        return undefined;
      }
      let binding: Message;
      if (input.type.value.default !== undefined) {
        // Lift the Struct-shaped default (typed as JsonObject in TS
        // proto due to bufbuild's Struct -> JSON specialization) into
        // the declared FQN via protojson. Mirrors
        // common.ProtoOptions.AsMessage on the Go side.
        try {
          binding = fromJson(desc, input.type.value.default, { registry });
        } catch {
          return undefined;
        }
      } else {
        binding = create(desc);
      }
      return {
        element: create(FieldElementSchema, { title: id }),
        binding,
        desc,
      };
    }
  }

  return undefined;
}

/**
 * Builds a wrapper-binding result by lifting the FieldElement
 * annotation off the binding's `value` field. The annotation drives
 * widget choice (Confirm / Input single-line / Input multiline /
 * etc.); the input id overrides the element title since flow.Input
 * has no title field of its own.
 */
function wrapperResult(binding: Message, schema: DescMessage, id: string): InputBindingResult {
  const valueField: DescField | undefined = schema.field.value;
  let element: FieldElement;
  if (valueField && hasOption(valueField, fieldExt)) {
    element = getOption(valueField, fieldExt);
  } else {
    element = create(FieldElementSchema);
  }
  element.title = id;
  return { element, binding, desc: schema };
}
