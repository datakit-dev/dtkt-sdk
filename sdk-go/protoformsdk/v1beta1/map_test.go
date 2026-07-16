package v1beta1

import (
	"strconv"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestMapParser(t *testing.T) {
	boolMap, err := messageMapParser(
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
		t.Logf("%#v", messageMapStringer(func(b bool) string { return strconv.FormatBool(b) }).StringOf(boolMap))
	}

	strMap, err := messageMapParser(
		func(s string) string { return s },
		func() protoreflect.Message {
			return (&emptypb.Empty{}).ProtoReflect()
		}).Parse(`{"foo": {}}`)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%#v", messageMapStringer(func(s string) string { return s }).StringOf(strMap))
	}

	intMap, err := messageMapParser(
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
		t.Logf("%#v", messageMapStringer(func(i int32) string { return util.FormatInt32(i) }).StringOf(intMap))
	}

	uintMap, err := messageMapParser(
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
		t.Logf("%#v", messageMapStringer(func(i uint32) string { return util.FormatUInt32(i) }).StringOf(uintMap))
	}
}
