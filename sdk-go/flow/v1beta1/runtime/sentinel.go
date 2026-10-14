package runtime

import (
	"errors"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc/codes"
	statusgrpc "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	doneProto *flowv1beta1.Done
	skipProto *flowv1beta1.Skip
)

type Done struct {
	proto *flowv1beta1.Done
}

func NewDone(status *statusgrpc.Status) *Done {
	if status == nil {
		status = statusgrpc.New(codes.Unknown, "unknown")
	}

	return &Done{
		proto: &flowv1beta1.Done{
			Code:    code.Code(status.Code()),
			Message: status.Message(),
		},
	}
}

func IsDone(value any) (*Done, bool) {
	if value == nil {
		return nil, false
	}

	switch value := value.(type) {
	case *Done:
		return value, true
	case error:
		done := new(Done)
		if errors.As(value, &done) {
			return done, true
		}
	case *flowv1beta1.Done:
		return &Done{proto: value}, true
	case proto.Message:
		if value.ProtoReflect().Descriptor().FullName() == doneProto.ProtoReflect().Descriptor().FullName() {
			proto := doneProto.ProtoReflect().New().Interface().(*flowv1beta1.Done)
			err := mergeMessage(api.GlobalResolver(), proto, value)
			if err == nil {
				return &Done{proto: proto}, true
			}
		}
	}

	return nil, false
}

func IsSkip(value any) (*flowv1beta1.Skip, bool) {
	if value == nil {
		return nil, false
	}

	msg, ok := value.(proto.Message)
	if !ok || msg.ProtoReflect().Descriptor().FullName() != skipProto.ProtoReflect().Descriptor().FullName() {
		return nil, false
	}

	return skipProto.ProtoReflect().New().Interface().(*flowv1beta1.Skip), true
}

func (e *Done) Proto() *flowv1beta1.Done {
	if e == nil || e.proto == nil {
		return nil
	}

	return proto.CloneOf(e.proto)
}

func (e *Done) IsEOF() bool {
	if e == nil || e.proto == nil {
		return false
	}

	return e.proto.GetCode() == code.Code_OK
}

func (e *Done) IsError() bool {
	if e == nil || e.proto == nil {
		return false
	}

	return e.proto.GetCode() > code.Code_OK
}

func (e *Done) Status() *statusgrpc.Status {
	code := codes.Unknown
	if e != nil && e.proto != nil {
		code = codes.Code(e.proto.Code)
	}
	return statusgrpc.New(code, e.String())
}

func (e *Done) Error() string {
	return e.String()
}

func (e *Done) String() string {
	if e == nil || e.proto == nil {
		return "Unknown"
	}

	if e.proto.GetMessage() != "" && e.proto.GetIdent() != "" {
		return fmt.Sprintf("%s: %s: %s", e.proto.GetIdent(), common.ProtoEnumToString(e.proto.GetCode(), true), e.proto.GetMessage())
	} else if e.proto.GetMessage() != "" {
		return fmt.Sprintf("%s: %s", common.ProtoEnumToString(e.proto.GetCode(), true), e.proto.GetMessage())
	}

	return common.ProtoEnumToString(e.proto.GetCode(), true)
}
