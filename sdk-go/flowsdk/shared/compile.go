package shared

import (
	"fmt"

	"github.com/google/cel-go/cel"
)

func CompileExprWithEnv(env *cel.Env, expr string, opts ...cel.ProgramOption) (cel.Program, error) {
	expr, valid := IsValidExpr(expr)
	if !valid {
		return nil, NewExprError(fmt.Sprintf("%s: %s", InvalidExprErrPrefix, expr))
	}

	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	return env.Program(ast, opts...)
}

func CompileExpr(run Runtime, expr string, opts ...cel.ProgramOption) (cel.Program, error) {
	env, err := run.Env()
	if err != nil {
		return nil, err
	}
	opts = append(opts, cel.InterruptCheckFrequency(1))
	return CompileExprWithEnv(env, expr, opts...)
}
