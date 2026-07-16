package runtime

import (
	"errors"
	"fmt"

	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"google.golang.org/protobuf/proto"
)

type Done struct {
	proto *flowv1beta1.Runtime_Done
}

func NewDone(proto *flowv1beta1.Runtime_Done) *Done {
	if proto == nil {
		proto = new(flowv1beta1.Runtime_Done)
	}
	return &Done{
		proto: proto,
	}
}

func IsDoneError(err error) (*Done, bool) {
	if err != nil {
		done := new(Done)
		if errors.As(err, &done) {
			return done, true
		}
	}
	return nil, false
}

func (e *Done) Proto() *flowv1beta1.Runtime_Done {
	return proto.CloneOf(e.proto)
}

func (e *Done) Error() string {
	if e.proto.Reason == "" {
		e.proto.Reason = "reason unknown"
	}

	if e.proto.Id != "" {
		return fmt.Sprintf("%s: %s", e.proto.GetId(), e.proto.GetReason())
	}

	return e.proto.Reason
}
