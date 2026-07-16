package common

import (
	"github.com/google/cel-go/cel"
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
		CELEnumLib(),
		CELJSONLib(),
		CELUUIDLib(),
		CELProtoDescLib(),
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
