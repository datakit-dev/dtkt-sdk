package spectest

import (
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

var _ shared.Env = (*TestEnv)(nil)

// TestEnv implements shared.Env using the global proto resolver and an empty
// flow proto. The TypeAdapter wraps CELTypes so NativeToValue works for any
// proto message registered in the global registry.
type TestEnv struct {
	*cel.Env
	celTypes *common.CELTypes
	vars     cel.Activation
}

var globalTestEnv = sync.OnceValues(func() (*TestEnv, error) {
	emptyProto := &flowv1beta1.Runtime{}

	vars, err := cel.ContextProtoVars(emptyProto)
	if err != nil {
		return nil, err
	}

	celTypes, err := common.NewCELTypes(api.GlobalResolver())
	if err != nil {
		return nil, err
	}

	celEnv, err := common.NewCELEnv(
		cel.CustomTypeProvider(celTypes),
		cel.CustomTypeAdapter(celTypes),
		cel.Container("dtkt"),
		cel.Abbrevs(
			string(new(flowv1beta1.Runtime_Done).ProtoReflect().Descriptor().FullName()),
			string(new(flowv1beta1.Runtime_EOF).ProtoReflect().Descriptor().FullName()),
		),
		cel.DeclareContextProto(emptyProto.ProtoReflect().Descriptor()),
	)
	if err != nil {
		return nil, err
	}

	return &TestEnv{
		Env:      celEnv,
		celTypes: celTypes,
		vars:     vars,
	}, nil
})

// NewTestEnv returns a shared.Env built from the global proto registry.
// It is safe to call multiple times; the environment is constructed once.
func NewTestEnv() (*TestEnv, error) {
	return globalTestEnv()
}

func (e *TestEnv) TypeAdapter() types.Adapter   { return e.celTypes }
func (e *TestEnv) TypeProvider() types.Provider { return e.celTypes }
func (e *TestEnv) Vars() cel.Activation         { return e.vars }
func (e *TestEnv) Resolver() shared.Resolver    { return api.GlobalResolver() }

// Check, Compile, Parse, and Program delegate to the embedded *cel.Env.
