package runtime

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc/mock"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
)

// Test infrastructure shared by lint, action, stream, and CEL typed-proto
// tests in this package. Three primitives:
//
//   - flowResolver: a shared.Resolver wrapping a *mock.Client (for method
//     dispatch) plus a list of FileDescriptors (for CEL type registration
//     and typed message resolution). Subsumes earlier per-test resolver
//     types.
//   - buildSyntheticFile: builds a protoreflect.FileDescriptor from a
//     descriptive spec. Used wherever a test needs a custom proto schema
//     without round-tripping through buf.
//   - withMockConnection: wires a single connection backed by a mock client
//     and resolver into the executor.
//
// Higher-level helpers (mockRPCOptions, packageMockOptions, etc.) are thin
// wrappers over these primitives so the same composition works everywhere.

// flowResolver wraps a *mock.Client with extra FileDescriptors. RangeFiles
// exposes the files for CEL type registration; FindMethodByName and
// FindMessageByName resolve from the embedded files first, then fall back
// to the *mock.Client / protoregistry.GlobalTypes.
type flowResolver struct {
	*mock.Client
	files []protoreflect.FileDescriptor
}

// newFlowResolver wraps client with extra files. client must not be nil;
// for tests that don't dispatch RPCs (e.g. lint), pass mock.NewClient().
func newFlowResolver(client *mock.Client, files ...protoreflect.FileDescriptor) *flowResolver {
	return &flowResolver{Client: client, files: files}
}

func (r *flowResolver) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	for _, fd := range r.files {
		if !f(fd) {
			return
		}
	}
}

func (r *flowResolver) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	for _, fd := range r.files {
		for i := 0; i < fd.Services().Len(); i++ {
			svc := fd.Services().Get(i)
			for j := 0; j < svc.Methods().Len(); j++ {
				md := svc.Methods().Get(j)
				if md.FullName() == name {
					return md, nil
				}
			}
		}
	}
	return r.Client.FindMethodByName(name)
}

func (r *flowResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	for _, fd := range r.files {
		for i := 0; i < fd.Messages().Len(); i++ {
			md := fd.Messages().Get(i)
			if md.FullName() == name {
				return dynamicpb.NewMessageType(md), nil
			}
		}
	}
	return protoregistry.GlobalTypes.FindMessageByName(name)
}

// syntheticFileSpec describes a proto file to build at test time. The
// resulting FileDescriptor can be embedded in a flowResolver to expose
// custom message and method shapes without generating a real proto.
type syntheticFileSpec struct {
	fileName    string
	packageName string
	messages    []syntheticMessage
	services    []syntheticService
}

type syntheticMessage struct {
	name   string
	fields []syntheticField
}

type syntheticField struct {
	name      string
	number    int32
	fieldType descriptorpb.FieldDescriptorProto_Type
	repeated  bool
	typeName  string // ".pkg.MsgName" for message/enum fields
}

type syntheticService struct {
	name    string
	methods []syntheticMethod
}

type syntheticMethod struct {
	name            string
	inputType       string // fully-qualified, leading dot
	outputType      string
	clientStreaming bool
	serverStreaming bool
}

func buildSyntheticFile(t *testing.T, spec syntheticFileSpec) protoreflect.FileDescriptor {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    proto.String(spec.fileName),
		Syntax:  proto.String("proto3"),
		Package: proto.String(spec.packageName),
	}
	for _, m := range spec.messages {
		desc := &descriptorpb.DescriptorProto{Name: proto.String(m.name)}
		for _, f := range m.fields {
			label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
			if f.repeated {
				label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
			}
			fld := &descriptorpb.FieldDescriptorProto{
				Name:   proto.String(f.name),
				Number: proto.Int32(f.number),
				Type:   f.fieldType.Enum(),
				Label:  label.Enum(),
			}
			if f.typeName != "" {
				fld.TypeName = proto.String(f.typeName)
			}
			desc.Field = append(desc.Field, fld)
		}
		fdp.MessageType = append(fdp.MessageType, desc)
	}
	for _, s := range spec.services {
		svc := &descriptorpb.ServiceDescriptorProto{Name: proto.String(s.name)}
		for _, m := range s.methods {
			svc.Method = append(svc.Method, &descriptorpb.MethodDescriptorProto{
				Name:            proto.String(m.name),
				InputType:       proto.String(m.inputType),
				OutputType:      proto.String(m.outputType),
				ClientStreaming: proto.Bool(m.clientStreaming),
				ServerStreaming: proto.Bool(m.serverStreaming),
			})
		}
		fdp.Service = append(fdp.Service, svc)
	}
	file, err := protodesc.NewFile(fdp, nil)
	if err != nil {
		t.Fatalf("building synthetic file %q: %v", spec.fileName, err)
	}
	return file
}

// withMockConnection wires a single connection ID backed by client (with any
// extra FileDescriptors registered for CEL type resolution) into the
// executor. For multi-connection setups, build the rpc.Connectors map
// directly and call WithConnectors.
func withMockConnection(id string, client *mock.Client, files ...protoreflect.FileDescriptor) Option {
	return WithConnectors(rpc.Connectors{
		id: &rpc.Connector{Client: client, Resolver: newFlowResolver(client, files...)},
	})
}

// --- Higher-level helpers built on the primitives ---------------------------

// packageMockClient registers handlers for the dtkt.shared.v1beta1.Package
// fixture used by the typed-proto CEL tests: a server stream that yields a
// few distinct Package messages, and a unary that returns one fully-
// populated Package fixture.
func packageMockClient() *mock.Client {
	c := mock.NewClient()
	c.RegisterServerStream("pkg.Stream", func(_ context.Context, _ proto.Message, send func(proto.Message) error) error {
		batch := []*sharedv1beta1.Package{
			{
				Identity:    &sharedv1beta1.Package_Identity{Name: "alpha", Version: "0.1.0"},
				Description: "first",
				Type:        sharedv1beta1.PackageType_PACKAGE_TYPE_GO,
			},
			{
				Identity:    &sharedv1beta1.Package_Identity{Name: "beta", Version: "0.2.0"},
				Description: "second",
				Type:        sharedv1beta1.PackageType_PACKAGE_TYPE_GO,
			},
			{
				Identity:    &sharedv1beta1.Package_Identity{Name: "gamma", Version: "0.3.0"},
				Description: "third",
				Type:        sharedv1beta1.PackageType_PACKAGE_TYPE_GO,
			},
		}
		for _, p := range batch {
			if err := send(p); err != nil {
				return err
			}
		}
		return nil
	})
	c.RegisterUnary("pkg.Fixed", func(_ context.Context, _ proto.Message) (proto.Message, error) {
		return &sharedv1beta1.Package{
			Identity: &sharedv1beta1.Package_Identity{
				Name:    "samplepkg",
				Version: "1.2.3",
			},
			Description: "a sample package",
			Type:        sharedv1beta1.PackageType_PACKAGE_TYPE_GO,
			Runtimes: []sharedv1beta1.Runtime{
				sharedv1beta1.Runtime_RUNTIME_NATIVE,
				sharedv1beta1.Runtime_RUNTIME_DOCKER,
			},
			Platforms: []*sharedv1beta1.Platform{
				{Os: sharedv1beta1.OS_OS_LINUX, Arch: sharedv1beta1.Arch_ARCH_AMD64},
				{Os: sharedv1beta1.OS_OS_DARWIN, Arch: sharedv1beta1.Arch_ARCH_ARM64},
			},
			Build: &sharedv1beta1.Package_BuildConfig{
				Env: map[string]string{"FOO": "bar", "BAZ": "qux"},
			},
			Deploy: &sharedv1beta1.Package_DeployConfig{
				Ports: []*sharedv1beta1.Package_DeployConfig_Port{
					{Name: "grpc", Protocol: "tcp", Port: "50051"},
					{Name: "http", Protocol: "tcp", Port: "8080"},
				},
			},
		}, nil
	})
	return c
}

// packageMockOptions wires a "pkg" connection backed by packageMockClient
// and exposes Package's FileDescriptor so CEL can access typed proto fields.
func packageMockOptions() []Option {
	return []Option{withMockConnection(
		"pkg",
		packageMockClient(),
		(&sharedv1beta1.Package{}).ProtoReflect().Descriptor().ParentFile(),
	)}
}

// echoRequestCaptureOptions wires an "echo" connection whose
// echo.Service.Echo unary stores the captured request in *captured for
// later assertion. The synthetic EchoRequest schema (name, age, city) lets
// schema lint pass and exprToMessage materialize a typed *dynamicpb.Message.
// The handler returns a structpb.StringValue so the response Any has a
// type URL resolvable via the global registry.
func echoRequestCaptureOptions(t *testing.T, captured *proto.Message) []Option {
	file := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    "echo.proto",
		packageName: "echo",
		messages: []syntheticMessage{
			{
				name: "EchoRequest",
				fields: []syntheticField{
					{name: "name", number: 1, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
					{name: "age", number: 2, fieldType: descriptorpb.FieldDescriptorProto_TYPE_INT64},
					{name: "city", number: 3, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
				},
			},
			{name: "EchoResponse"},
		},
		services: []syntheticService{
			{
				name: "Service",
				methods: []syntheticMethod{
					{name: "Echo", inputType: ".echo.EchoRequest", outputType: ".echo.EchoResponse"},
				},
			},
		},
	})

	c := mock.NewClient()
	c.RegisterUnary("echo.Service.Echo", func(_ context.Context, req proto.Message) (proto.Message, error) {
		*captured = req
		// Return a WKT so the response Any's type URL is resolvable through
		// cel-go's default global registry.
		return structpb.NewStringValue("ok"), nil
	})

	return []Option{withMockConnection("echo", c, file)}
}
