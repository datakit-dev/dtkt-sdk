package common

import (
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	"k8s.io/apiserver/pkg/cel/library"
)

var defaultCELOpts = sync.OnceValue(func() []cel.EnvOption {
	return []cel.EnvOption{
		library.URLs(),
		ext.Encoders(),
		ext.Lists(),
		ext.Protos(),
		ext.Strings(
			ext.StringsVersion(4),
		),
		CELEnumLib(),
		CELJSONLib(),
		CELUUIDLib(),
	}
})

func NewCELEnv(opts ...cel.EnvOption) (*cel.Env, error) {
	opts = append(defaultCELOpts(), opts...)

	return cel.NewEnv(opts...)
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
