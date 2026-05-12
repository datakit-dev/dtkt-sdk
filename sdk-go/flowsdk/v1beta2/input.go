package v1beta2

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	form "github.com/datakit-dev/dtkt-sdk/sdk-go/protoformsdk/v1beta1"
	"github.com/jhump/protoreflect/v2/protoresolve"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// GetInputBinding returns the protoform FieldElement and binding proto
// for a flow Input. Tuiform / protoformsdk consumes the binding's
// protoform-annotated fields and renders the matching widgets; the
// binding has its value pre-populated from the input's typed default
// when set.
//
// For scalar / list / map inputs the binding is one of Input_*Binding
// (a wrapper whose `value` field carries the FieldElement annotation).
// For Input_Message inputs the resolver looks up the declared FQN and
// the returned binding is a fresh instance of *that* message -- its
// own protoform-annotated fields drive the form. The Message default
// (a Struct) populates the typed message via the same protojson
// round-trip the runtime uses (see common.ProtoOptions.AsMessage).
//
// resolver is required only for Input_Message; other types ignore it.
//
// Returns (nil, nil, false) when the input's type is unrecognised
// (oneof unset), the message FQN can't be resolved by the resolver,
// or the default can't be coerced into the resolved type.
func GetInputBinding(input *flowv1beta2.Input, resolver protoresolve.SerializationResolver) (*protoformv1beta1.FieldElement, proto.Message, bool) {
	var binding proto.Message
	switch t := input.GetType().(type) {
	case *flowv1beta2.Input_Bool:
		b := &flowv1beta2.Input_BoolBinding{}
		if t.Bool.HasDefault() {
			b.SetValue(t.Bool.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_Bytes:
		b := &flowv1beta2.Input_BytesBinding{}
		if t.Bytes.HasDefault() {
			b.SetValue(t.Bytes.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_Double:
		b := &flowv1beta2.Input_DoubleBinding{}
		if t.Double.HasDefault() {
			b.SetValue(t.Double.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_Float:
		b := &flowv1beta2.Input_FloatBinding{}
		if t.Float.HasDefault() {
			b.SetValue(t.Float.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_Int64:
		b := &flowv1beta2.Input_Int64Binding{}
		if t.Int64.HasDefault() {
			b.SetValue(t.Int64.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_Uint64:
		b := &flowv1beta2.Input_Uint64Binding{}
		if t.Uint64.HasDefault() {
			b.SetValue(t.Uint64.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_Int32:
		b := &flowv1beta2.Input_Int32Binding{}
		if t.Int32.HasDefault() {
			b.SetValue(t.Int32.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_Uint32:
		b := &flowv1beta2.Input_Uint32Binding{}
		if t.Uint32.HasDefault() {
			b.SetValue(t.Uint32.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_String_:
		b := &flowv1beta2.Input_StringBinding{}
		if t.String_.HasDefault() {
			b.SetValue(t.String_.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_List:
		b := &flowv1beta2.Input_ListBinding{}
		if t.List.HasDefault() {
			b.SetValue(t.List.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_Map:
		b := &flowv1beta2.Input_MapBinding{}
		if t.Map.HasDefault() {
			b.SetValue(t.Map.GetDefault())
		}
		binding = b
	case *flowv1beta2.Input_Message:
		msgType, err := resolver.FindMessageByName(protoreflect.FullName(t.Message.GetType()))
		if err != nil {
			return nil, nil, false
		}
		if t.Message.HasDefault() {
			msg, err := (common.ProtoOptions{Resolver: resolver}).AsMessage(t.Message.GetDefault(), msgType)
			if err != nil {
				return nil, nil, false
			}
			binding = msg
		} else {
			binding = msgType.New().Interface()
		}
	}

	if binding == nil {
		return nil, nil, false
	}

	// For wrapper bindings (scalar / list / map), the FieldElement
	// lives on the binding's `value` field and is the single widget
	// hint. Surface the input id as its title since flow.Input has
	// no title field.
	//
	// For typed-message bindings there is no `value` field -- the
	// returned proto IS the form, and each of its fields carries its
	// own FieldElement annotation. We return a synthetic top-level
	// element holding the id; the caller iterates the field group
	// and the per-field annotations drive each widget.
	if desc := binding.ProtoReflect().Descriptor().Fields().ByName("value"); desc != nil {
		if elem, ok := form.GetFieldElement(desc); ok {
			title := input.GetId()
			elem.Title = &title
			return elem, binding, true
		}
	}

	title := input.GetId()
	return &protoformv1beta1.FieldElement{Title: &title}, binding, true
}
