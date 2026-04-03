package runtime

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/funcs"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

var _ shared.Env = (*Env)(nil)

type (
	Env struct {
		*cel.Env
		vars     cel.Activation
		types    *common.CELTypes
		resolver shared.Resolver
	}
	EnvOption func(*Env)
)

func NewEnv(proto *flowv1beta1.Runtime, opts ...Option) (*Env, error) {
	env := &Env{}
	env.applyOptions(opts...)

	if env.resolver == nil {
		env.resolver = api.GlobalResolver()
	}

	types, err := common.NewCELTypes(env.resolver)
	if err != nil {
		return nil, err
	}

	env.types = types

	vars, err := cel.ContextProtoVars(proto)
	if err != nil {
		return nil, err
	}

	env.vars = vars

	celOpts := append([]cel.EnvOption{
		cel.CustomTypeProvider(env.types),
		cel.CustomTypeAdapter(env.types),
		cel.Container("dtkt"),
		cel.Abbrevs(
			string(new(flowv1beta1.Runtime_Done).ProtoReflect().Descriptor().FullName()),
			string(new(flowv1beta1.Runtime_EOF).ProtoReflect().Descriptor().FullName()),
		),
		cel.DeclareContextProto(proto.ProtoReflect().Descriptor()),
	}, funcs.EnvOptions(env)...)

	env.Env, err = common.NewCELEnv(celOpts...)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func (e *Env) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
}

func (e *Env) Vars() cel.Activation {
	return e.vars
}

func (e *Env) TypeAdapter() types.Adapter {
	return e.types
}

func (e *Env) TypeProvider() types.Provider {
	return e.types
}

func (e *Env) Resolver() shared.Resolver {
	return e.resolver
}
