package protostore

import (
	"reflect"
	"testing"

	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/proto"
)

func checkMessageFields(t *testing.T, msg proto.Message) {
	for idx := range msg.ProtoReflect().Descriptor().Fields().Len() {
		desc := msg.ProtoReflect().Descriptor().Fields().Get(idx)
		typ := ReflectFieldType(desc)
		if typ == nil {
			t.Errorf("field: %s, reflect type nil", desc.FullName())
		}

		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}

		t.Log(typ.String())
	}
}

func TestReflection(t *testing.T) {
	for _, msg := range []proto.Message{
		&corev1.Automation{},
		&sharedv1beta1.Package{},
		&sharedv1beta1.Package_BuildConfig{},
		&sharedv1beta1.Package_DeployConfig{},
	} {
		checkMessageFields(t, msg)
	}
}
