package v1beta1

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	aiv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/ai/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Env = (*testEnv)(nil)
var _ Resolver = (*testResolver)(nil)

type (
	testEnv struct {
		t        *testing.T
		resolver *testResolver
	}
	testResolver struct {
		api.Resolver
		t     *testing.T
		resps []proto.Message
	}
)

func TestResolveOptions(t *testing.T) {
	testResps := []*aiv1beta1.ListModelsResponse{
		{
			Models: []*aiv1beta1.Model{
				{
					Id:   "model1",
					Name: "Model 1",
					Capabilities: []aiv1beta1.ModelCapability{
						aiv1beta1.ModelCapability_MODEL_CAPABILITY_EMBEDDING,
					},
				},
			},
			NextPageToken: "page2",
		},
		{
			Models: []*aiv1beta1.Model{
				{
					Id:   "model2",
					Name: "Model 2",
					Capabilities: []aiv1beta1.ModelCapability{
						aiv1beta1.ModelCapability_MODEL_CAPABILITY_EMBEDDING,
					},
				},
			},
		},
	}

	env := &testEnv{
		t: t,
		resolver: &testResolver{
			Resolver: api.GlobalResolver(),
			t:        t,
			resps: util.SliceMap(testResps, func(r *aiv1beta1.ListModelsResponse) proto.Message {
				return r
			}),
		},
	}

	msg := NewMessage((&aiv1beta1.GenerateEmbeddingsRequest{}).ProtoReflect())
	var modelField Field
	for _, field := range msg.FieldGroup().GetFields() {
		if field.Descriptor().Name() == "model_id" {
			modelField = field
			break
		}
	}

	if modelField.FieldType == nil {
		t.Fatal("expected valid model_id field")
	} else if selectElem, ok := modelField.Element().Type.(*SelectElement); !ok {
		t.Fatal("expected model_id field select element")
	} else {
		selectElem.element.UseFilterField = new(true)

		err := selectElem.Load(env)
		if err != nil {
			t.Fatalf("failed to load model_id field: %s", err)
		}

		opts := selectElem.GetOptions()
		merged := &aiv1beta1.ListModelsResponse{}
		for _, res := range testResps {
			proto.Merge(merged, res)
		}

		if opts.Len() != len(merged.Models) {
			t.Fatalf("expected %d options, got: %d", len(merged.Models), opts.Len())
		}

		var idx int
		opts.Range(func(key string, val any) bool {
			if key != merged.Models[idx].Name {
				t.Fatalf("expected option %d key: %q, got: %q", idx, key, merged.Models[idx].Name)
				return false
			}

			if v, ok := val.(protoreflect.Value); ok {
				val = v.Interface()
			}

			if val != merged.Models[idx].Id {
				t.Fatalf("expected option %d value: %q, got: %q", idx, val, merged.Models[idx].Id)
				return false
			}

			t.Logf("Option %d label=%q value=%v", idx, key, val)

			idx++
			return true
		})
	}
}

func (e *testEnv) Context() context.Context {
	return e.t.Context()
}

func (e *testEnv) Logger() *slog.Logger {
	return slog.Default()
}

func (e *testEnv) OnGroupCompleted(g *FieldGroup, f GroupCallbackFunc) error {
	return f(g)
}

func (e *testEnv) Resolver() Resolver {
	return e.resolver
}

func (e *testEnv) Encoder() (encoding.Encoder, bool) { return nil, false }

func (e *testEnv) Decoder() (encoding.Decoder, bool) { return nil, false }

func (e *testEnv) Validator() (protovalidate.Validator, error) {
	return e.resolver.GetValidator()
}

func (r *testResolver) InvokeMethod(_ context.Context, name protoreflect.FullName, req proto.Message) (proto.Message, error) {
	if len(r.resps) > 0 {
		res := r.resps[0]
		if len(r.resps) > 1 {
			r.resps = r.resps[1:]
		} else {
			r.resps = r.resps[:0]
		}
		return res, nil
	}
	return nil, fmt.Errorf("test responses empty")
}
