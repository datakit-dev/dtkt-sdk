package common

import (
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
	"k8s.io/apiserver/pkg/cel/library"
)

func NewCELEnv(opts ...cel.EnvOption) (*cel.Env, error) {
	return cel.NewEnv(append([]cel.EnvOption{
		library.URLs(),
		ext.Encoders(),
		ext.Lists(),
		ext.Protos(),
		ext.Strings(),
		ext.TwoVarComprehensions(),
		CELLocaleLib(),
		CELEnumLib(),
		CELJSONLib(),
		CELUUIDLib(),
		CELDescriptorLib(),
		cel.Function("now",
			cel.SingletonFunctionBinding(
				func(...ref.Val) ref.Val {
					return types.Timestamp{
						Time: time.Now(),
					}
				},
			),
			cel.Overload("now", nil, cel.TimestampType),
		),
	}, opts...)...)
}

func CompileCELExpr(expr string, opts ...cel.EnvOption) (cel.Program, error) {
	env, err := NewCELEnv(opts...)
	if err != nil {
		return nil, err
	}

	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	return env.Program(ast)
}
