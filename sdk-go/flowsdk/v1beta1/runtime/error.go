package runtime

import (
	"errors"
	"fmt"

	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"google.golang.org/protobuf/proto"
)

type Done struct {
	proto *flowv1beta1.Done
}

func NewDone(proto *flowv1beta1.Done) *Done {
	if proto == nil {
		proto = new(flowv1beta1.Done)
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

func (e *Done) Proto() *flowv1beta1.Done {
	if e.proto == nil {
		e.proto = new(flowv1beta1.Done)
	}
	return proto.CloneOf(e.proto)
}

func (e *Done) IsError() bool {
	return e.Proto().GetIsError()
}

func (e *Done) Error() string {
	return e.String()
}

func (e *Done) String() string {
	reason := e.Proto().GetReason()
	if reason == "" {
		reason = "unknown"
	}

	if e.Proto().GetId() != "" {
		return fmt.Sprintf("%s: %s", e.Proto().GetId(), reason)
	}

	return reason
}
