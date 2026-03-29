package spectest

import (
"context"

"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
"github.com/google/cel-go/common/types/ref"
"google.golang.org/protobuf/proto"
"google.golang.org/protobuf/reflect/protoreflect"
)

var _ shared.Runtime = (*MockRuntime)(nil)

// MockRuntime is a minimal shared.Runtime for use in unit tests.
// Call NewMockRuntime to get one backed by the global TestEnv, or set EnvFn
// manually for custom behaviour.
type MockRuntime struct {
	Ctx   context.Context
	EnvFn func() (shared.Env, error)
	Conns shared.ConnectorProvider
}

// NewMockRuntime returns a MockRuntime backed by the shared global TestEnv.
func NewMockRuntime(ctx context.Context) *MockRuntime {
	return &MockRuntime{
		Ctx: ctx,
		EnvFn: func() (shared.Env, error) {
			return NewTestEnv()
		},
	}
}

func (m *MockRuntime) Context() context.Context { return m.Ctx }

func (m *MockRuntime) Env() (shared.Env, error) {
	if m.EnvFn != nil {
		return m.EnvFn()
	}
	return nil, nil
}

func (m *MockRuntime) Connectors() shared.ConnectorProvider {
	if m.Conns != nil {
		return m.Conns
	}
	return &noopConnectorProvider{}
}

func (m *MockRuntime) GetNode(string) (shared.SpecNode, bool)   { return nil, false }
func (m *MockRuntime) GetValue(string) (any, error)             { return nil, nil }
func (m *MockRuntime) GetSendCh(string) (chan<- ref.Val, error) { return nil, nil }
func (m *MockRuntime) GetRecvCh(string) (<-chan any, error)     { return nil, nil }

// ---- noopConnectorProvider -------------------------------------------------

type noopConnectorProvider struct{}

func (n *noopConnectorProvider) GetConnector(_ context.Context, _ string, _ shared.Package, _ []string) (shared.Connector, error) {
	return &MockConnector{Client: &MockDynamicClient{}}, nil
}

// ---- ServerStream factory helper ------------------------------------------

// ServerStreamConnector returns a MockConnector whose DynamicClient creates the
// given pre-built DynamicServerStream on every CallServerStream call.
func ServerStreamConnector(stream common.DynamicServerStream) *MockConnector {
	return &MockConnector{
		Client: &MockDynamicClient{
			ServerStreamFactory: func(_ context.Context, _ protoreflect.FullName, _ proto.Message) (common.DynamicServerStream, error) {
				return stream, nil
			},
		},
	}
}
