package runtime

import (
	"context"
	"fmt"
	"reflect"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// eofRefVal is a CEL ref.Val that wraps the EOF proto sentinel.
// Used by the EOF() CEL function. refValToExpr handles it via the proto.Message
// default case, producing an ObjectValue that isEOFValue recognizes.
type eofRefVal struct{}

var eofRefValInstance ref.Val = eofRefVal{}

func (eofRefVal) ConvertToNative(typeDesc reflect.Type) (any, error) {
	eof := &flowv1beta2.EOF{}
	if typeDesc == reflect.TypeOf((*anypb.Any)(nil)) {
		return anypb.New(eof)
	}
	return eof, nil
}
func (eofRefVal) ConvertToType(typeVal ref.Type) ref.Val { return types.NewErr("no conversion") }
func (eofRefVal) Equal(other ref.Val) ref.Val            { _, ok := other.(eofRefVal); return types.Bool(ok) }
func (eofRefVal) Type() ref.Type {
	return types.NewObjectType("dtkt.flow.v1beta2.EOF")
}
func (eofRefVal) Value() any { return &flowv1beta2.EOF{} }

// newEOFValue creates a cel.expr.Value representing end-of-stream.
func newEOFValue() *expr.Value {
	eofAny, _ := anypb.New(&flowv1beta2.EOF{})
	return &expr.Value{Kind: &expr.Value_ObjectValue{ObjectValue: eofAny}}
}

// NewEOFValue creates a cel.expr.Value representing end-of-stream (exported for main).
func NewEOFValue() *expr.Value { return newEOFValue() }

// isEOFValue checks if a cel.expr.Value represents end-of-stream.
func isEOFValue(v *expr.Value) bool {
	if v == nil {
		return false
	}
	obj := v.GetObjectValue()
	if obj == nil {
		return false
	}
	return obj.MessageIs((*flowv1beta2.EOF)(nil))
}

// exprToRefVal converts a cel.expr.Value to a CEL ref.Val for use in CEL evaluation.
// Delegates to cel.ProtoAsValue with the given adapter for proper custom type resolution.
func exprToRefVal(adapter types.Adapter, v *expr.Value) ref.Val {
	if v == nil {
		return types.NullValue
	}
	rv, err := cel.ProtoAsValue(adapter, v)
	if err != nil {
		return types.NewErr("converting expr to ref.Val: %v", err)
	}
	return rv
}

// refValToExpr converts a CEL ref.Val to a cel.expr.Value for wire serialization.
// Delegates to cel.ValueAsProto from cel-go.
func refValToExpr(v ref.Val) (*expr.Value, error) {
	if v == nil {
		return &expr.Value{Kind: &expr.Value_NullValue{}}, nil
	}
	return cel.ValueAsProto(v)
}

// protoToExpr wraps a proto.Message in a cel.expr.Value via Any.
func protoToExpr(msg proto.Message) (*expr.Value, error) {
	a, err := anypb.New(msg)
	if err != nil {
		return nil, err
	}
	return &expr.Value{Kind: &expr.Value_ObjectValue{ObjectValue: a}}, nil
}

// responseToExpr converts a proto.Message response from an RPC client to *expr.Value.
// If the response is already *expr.Value, it is returned directly.
// Otherwise it is wrapped via protoToExpr (Any-encoded ObjectValue).
func responseToExpr(msg proto.Message) (*expr.Value, error) {
	if v, ok := msg.(*expr.Value); ok {
		return v, nil
	}
	return protoToExpr(msg)
}

// flowEventFromNode wraps a StateNode in the appropriate FlowEvent oneof variant.
func flowEventFromNode(eventType flowv1beta2.RunSnapshot_FlowEvent_EventType, node executor.StateNode) *flowv1beta2.RunSnapshot_FlowEvent {
	evt := &flowv1beta2.RunSnapshot_FlowEvent{}
	evt.SetEventType(eventType)
	switch n := node.(type) {
	case *flowv1beta2.RunSnapshot_InputNode:
		evt.SetInput(n)
	case *flowv1beta2.RunSnapshot_GeneratorNode:
		evt.SetGenerator(n)
	case *flowv1beta2.RunSnapshot_VarNode:
		evt.SetVar(n)
	case *flowv1beta2.RunSnapshot_ActionNode:
		evt.SetAction(n)
	case *flowv1beta2.RunSnapshot_StreamNode:
		evt.SetStream(n)
	case *flowv1beta2.RunSnapshot_OutputNode:
		evt.SetOutput(n)
	case *flowv1beta2.RunSnapshot_InteractionNode:
		evt.SetInteraction(n)
	}
	return evt
}

// runtimeNodeFromEvent extracts the StateNode from a FlowEvent's oneof.
func runtimeNodeFromEvent(event *flowv1beta2.RunSnapshot_FlowEvent) executor.StateNode {
	switch event.WhichData() {
	case flowv1beta2.RunSnapshot_FlowEvent_Input_case:
		return event.GetInput()
	case flowv1beta2.RunSnapshot_FlowEvent_Generator_case:
		return event.GetGenerator()
	case flowv1beta2.RunSnapshot_FlowEvent_Var_case:
		return event.GetVar()
	case flowv1beta2.RunSnapshot_FlowEvent_Action_case:
		return event.GetAction()
	case flowv1beta2.RunSnapshot_FlowEvent_Stream_case:
		return event.GetStream()
	case flowv1beta2.RunSnapshot_FlowEvent_Output_case:
		return event.GetOutput()
	case flowv1beta2.RunSnapshot_FlowEvent_Interaction_case:
		return event.GetInteraction()
	default:
		return nil
	}
}

// publishNode publishes a StateNode wrapped in a FlowEvent{NODE_OUTPUT} to a topic.
func publishNode(pub pubsub.Publisher, topic string, node executor.StateNode) error {
	return pub.Publish(topic, pubsub.NewMessage(flowEventFromNode(flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT, node)))
}

// publishStateEvent publishes a StateNode wrapped in a FlowEvent{NODE_UPDATE} to a topic.
// NODE_UPDATE events carry updated accumulator state without triggering downstream CEL evaluation.
func publishStateEvent(pub pubsub.Publisher, topic string, node executor.StateNode) error {
	return pub.Publish(topic, pubsub.NewMessage(flowEventFromNode(flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_UPDATE, node)))
}

// publishFlowState publishes a FlowState wrapped in a FlowEvent{FLOW_UPDATE}
// to the flow topic. Used for flow-level phase transitions (RUNNING, SUCCEEDED, etc.).
func publishFlowState(pub pubsub.Publisher, topic string, state *flowv1beta2.RunSnapshot_FlowState) error {
	evt := &flowv1beta2.RunSnapshot_FlowEvent{}
	evt.SetEventType(flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_FLOW_UPDATE)
	evt.SetFlow(state)
	return pub.Publish(topic, pubsub.NewMessage(evt))
}

// stateNodeBuilder builds the StateNode for a STATE event given the transforms slice.
type stateNodeBuilder func(transforms []*flowv1beta2.RunSnapshot_Transform) executor.StateNode

// newStateCallback returns a stateCallback that publishes STATE NodeEvents.
// numSteps is the total number of transform steps (for sizing the transforms slice).
// build constructs the node-type-specific StateNode with the provided transforms.
func newStateCallback(pub executor.PubSub, topic string, numSteps int, build stateNodeBuilder) stateCallback {
	return func(ctx context.Context, stepIdx int, acc *expr.Value) error {
		transforms := make([]*flowv1beta2.RunSnapshot_Transform, numSteps)
		transforms[stepIdx] = flowv1beta2.RunSnapshot_Transform_builder{
			Accumulator: acc,
		}.Build()
		return publishStateEvent(pub, topic, build(transforms))
	}
}

// exprToMessage converts a *expr.Value into a typed proto.Message for the given
// RPC method input. It creates a fresh instance of the method's input type via
// the resolver, then round-trips through JSON to populate it from the expr value.
// Uses shared.ExprValueToNative for resolver-aware value conversion.
func exprToMessage(env shared.Env, method protoreflect.FullName, input *expr.Value) (proto.Message, error) {
	resolver := env.Resolver()
	methodDesc, err := resolver.FindMethodByName(method)
	if err != nil {
		return input, nil
	}
	msgType, err := resolver.FindMessageByName(methodDesc.Input().FullName())
	if err != nil {
		return input, nil
	}
	req := msgType.New().Interface()
	if input == nil {
		return req, nil
	}
	native, err := shared.ExprValueToNative(env, input)
	if err != nil {
		return nil, fmt.Errorf("converting expr to native: %w", err)
	}
	if native == nil {
		return req, nil
	}
	b, err := encoding.ToJSONV2(native,
		encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
			Resolver: resolver,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("encoding request to JSON: %w", err)
	}
	err = encoding.FromJSONV2(b, req,
		encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
			Resolver: resolver,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("decoding request JSON to %s: %w", methodDesc.Input().FullName(), err)
	}
	return req, nil
}
