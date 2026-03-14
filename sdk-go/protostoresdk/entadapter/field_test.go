package entadapter

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/v1beta1"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestField(t *testing.T) {
	msg := &durationpb.Duration{}
	desc := msg.ProtoReflect().Descriptor().Fields().ByName("seconds")

	field := NewField(v1beta1.NewField(desc))
	if field.desc == nil {
		t.Errorf("field %s descriptor is nil", desc.FullName())
	} else if field.desc.Err != nil {
		t.Error(field.desc.Err)
	}

	msgField := MessageField[timestamppb.Timestamp]()
	if msgField == nil {
		t.Fatal("message is nil")
	}
}
