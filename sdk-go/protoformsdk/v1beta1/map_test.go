package v1beta1_test

import (
	"strconv"
	"testing"

	form "github.com/datakit-dev/dtkt-sdk/sdk-go/protoformsdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestMapParser(t *testing.T) {
	boolMap, err := form.MapMessageParser(
		func(s string) (b bool) {
			b, _ = strconv.ParseBool(s)
			return
		},
		func() protoreflect.Message {
			return (&emptypb.Empty{}).ProtoReflect()
		}).Parse(`{"false": {}}`)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%#v", form.MapMessageStringer(func(b bool) string { return strconv.FormatBool(b) }).StringOf(boolMap))
	}

	strMap, err := form.MapMessageParser(
		func(s string) string { return s },
		func() protoreflect.Message {
			return (&emptypb.Empty{}).ProtoReflect()
		}).Parse(`{"foo": {}}`)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%#v", form.MapMessageStringer(func(s string) string { return s }).StringOf(strMap))
	}

	intMap, err := form.MapMessageParser(
		func(s string) int32 {
			i, _ := strconv.Atoi(s)
			return int32(i)
		},
		func() protoreflect.Message {
			return (&emptypb.Empty{}).ProtoReflect()
		}).Parse(`{"-123": {}}`)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%#v", form.MapMessageStringer(func(i int32) string { return util.FormatInt32(i) }).StringOf(intMap))
	}

	uintMap, err := form.MapMessageParser(
		func(s string) uint32 {
			i, _ := strconv.ParseUint(s, 10, 32)
			return uint32(i)
		},
		func() protoreflect.Message {
			return (&emptypb.Empty{}).ProtoReflect()
		}).Parse(`{"123": {}}`)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%#v", form.MapMessageStringer(func(i uint32) string { return util.FormatUInt32(i) }).StringOf(uintMap))
	}
}
