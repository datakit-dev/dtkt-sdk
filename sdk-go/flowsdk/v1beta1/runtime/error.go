package runtime

import (
	"errors"
	"fmt"

	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"google.golang.org/protobuf/proto"
)

type (
	DoneError struct {
		proto *flowv1beta1.Runtime_Done
	}
)

func NewDoneError(proto *flowv1beta1.Runtime_Done) *DoneError {
	if proto == nil {
		proto = new(flowv1beta1.Runtime_Done)
	}
	return &DoneError{
		proto: proto,
	}
}

func IsDoneError(err error) (*DoneError, bool) {
	if err != nil {
		doneErr := new(DoneError)
		if errors.As(err, &doneErr) {
			return doneErr, true
		}
	}
	return nil, false
}

func IsRuntimeDone(run *Runtime) (doneErr *DoneError, ok bool) {
	run.nodes.Range(func(_ string, node *Node) bool {
		switch value := node.value.(type) {
		case *flowv1beta1.Runtime_Done:
			// Prefer error Dones: keep searching if we already have a non-error.
			if !ok || value.GetIsError() {
				doneErr = NewDoneError(value)
				ok = true
			}
			if value.GetIsError() {
				return false // error found — stop immediately
			}
		}
		return true
	})
	return
}

func (e *DoneError) Proto() *flowv1beta1.Runtime_Done {
	return proto.CloneOf(e.proto)
}

func (e *DoneError) Error() string {
	if e.proto.Reason == "" {
		e.proto.Reason = "reason unknown"
	}

	if e.proto.Id != "" {
		return fmt.Sprintf("%s: %s", e.proto.Id, e.proto.Reason)
	}

	return e.proto.Reason
}
