package encoding_test

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	commandv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/command/v1beta1"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestJSONDecodeV2_Duration(t *testing.T) {
	durStr := `"2h45m"`
	var d time.Duration
	err := encoding.FromJSONV2([]byte(durStr), &d)
	if err != nil {
		t.Fatal(err)
	}

	expected := 2*time.Hour + 45*time.Minute
	if d != expected {
		t.Fatalf("expected %v, got %v", expected, d)
	}

	// Test stream decode with multiple durations
	data := `"30m"
"1h15m"
"45s"`
	decode := encoding.NewJSONDecoderV2().StreamDecode(bytes.NewBuffer([]byte(data)))
	var durations []time.Duration
	for {
		var dur time.Duration
		err := decode(&dur)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("stream decode error: %v", err)
		}
		durations = append(durations, dur)
	}

	expectedDurations := []time.Duration{30 * time.Minute, 1*time.Hour + 15*time.Minute, 45 * time.Second}
	if len(durations) != len(expectedDurations) {
		t.Fatalf("expected %d durations, got %d", len(expectedDurations), len(durations))
	}
	for i, dur := range durations {
		if dur != expectedDurations[i] {
			t.Fatalf("at index %d, expected %v, got %v", i, expectedDurations[i], dur)
		}
	}
}

func TestJSONEncoderV2_Duration(t *testing.T) {
	d := 90 * time.Minute
	b, err := encoding.ToJSONV2(d)
	if err != nil {
		t.Fatal(err)
	}

	expected := `"1h30m0s"`
	if string(b) != expected {
		t.Fatalf("expected %s, got %s", expected, string(b))
	}

	d = 91 * time.Minute
	// Test raw mode
	b, err = encoding.ToJSONV2(d, encoding.WithEncodeRaw(true))
	if err != nil {
		t.Fatal(err)
	}

	expected = `1h31m0s`
	if string(b) != expected {
		t.Fatalf("expected %s, got %s", expected, string(b))
	}

	// Test stream encode with multiple durations

	var buf bytes.Buffer
	encode := encoding.NewJSONEncoderV2().StreamEncode(&buf)
	durations := []time.Duration{15 * time.Minute, 2*time.Hour + 30*time.Minute, 10 * time.Second}
	for _, dur := range durations {
		err := encode(dur)
		if err != nil {
			t.Fatalf("stream encode error: %v", err)
		}
	}

	expectedBuf := `"15m0s"
"2h30m0s"
"10s"
`

	if buf.String() != expectedBuf {
		t.Fatalf("expected %s, got %s", expectedBuf, buf.String())
	}
}

func TestJSONEncodeV2_Defaults(t *testing.T) {
	str := "foobar"
	b, err := encoding.ToJSONV2(str)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != strconv.Quote(str) {
		t.Fatalf("expected to be equal: %s != %s", str, string(b))
	}

	msg := &corev1.RunOperationMetadata{
		State: corev1.RunOperationMetadata_STATE_RUNNING,
	}

	b, err = encoding.ToJSONV2(msg)
	if err != nil {
		t.Fatal(err)
	}

	msg2 := new(corev1.RunOperationMetadata)
	err = encoding.FromJSONV2(b, msg2)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(msg, msg2) {
		t.Fatalf("expected to be equal: %v != %v", msg, msg2)
	}

	d := time.Duration(time.Minute)
	b, err = encoding.ToJSONV2(d)
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != `"1m0s"` {
		t.Fatalf(`expected "1m0s", got: %s`, string(b))
	}

	var d2 time.Duration
	err = encoding.FromJSONV2(b, &d2)
	if err != nil {
		t.Fatal(err)
	}

	if d != d2 {
		t.Fatalf("expected to be equal, %v != %v", d, d2)
	}
}

func TestJSONEncoderV2_ProtoJSONRedaction(t *testing.T) {
	nativeProto := &commandv1beta1.SSHConfig{
		Address: "localhost:9090",
		Auth: &commandv1beta1.SSHConfig_Password{
			Password: "supersecret",
		},
	}

	_, err := encoding.ToJSONV2(
		nativeProto,
		encoding.WithEncodeProtoJSONRedact(),
		encoding.WithEncodeMultiline(true),
		encoding.WithEncodeIndent("", "  "),
	)
	if err != nil {
		t.Fatal(err)
	}

	msgType, err := api.V1Beta1.FindMessageByName(nativeProto.ProtoReflect().Descriptor().FullName())
	if err != nil {
		t.Fatal(err)
	}

	dynProto := msgType.New()
	proto.Merge(dynProto.Interface(), nativeProto)

	_, err = encoding.ToJSONV2(
		dynProto.Interface(),
		encoding.WithEncodeProtoJSONRedact(),
		encoding.WithEncodeMultiline(true),
		encoding.WithEncodeIndent("", "  "),
	)
	if err != nil {
		t.Fatal(err)
	}

	anyProto, err := anypb.New(nativeProto)
	if err != nil {
		t.Fatal(err)
	}

	_, err = encoding.ToJSONV2(
		anyProto,
		encoding.WithEncodeProtoJSONRedact(),
		encoding.WithEncodeMultiline(true),
		encoding.WithEncodeIndent("", "  "),
	)
	if err != nil {
		t.Fatal(err)
	}
}
