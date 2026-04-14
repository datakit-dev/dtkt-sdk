package executor

import (
	expr "cel.dev/expr"
	"google.golang.org/protobuf/proto"
)

// StateNode is the interface satisfied by all flat typed node messages
// (RunSnapshot_InputNode, RunSnapshot_VarNode, etc.). It enables generic runtime
// code (publishing, outbox serialization) to work with any node kind.
type StateNode interface {
	GetId() string
	GetValue() *expr.Value
	proto.Message
}
