package common

import (
	"testing"

	"github.com/google/cel-go/cel"
)

func Test_CELUUID(t *testing.T) {
	env, err := cel.NewEnv(CELUUIDLib())
	if err != nil {
		t.Fatal(err)
	}

	ast, iss := env.Compile("[uuid.new(), uuid.new_v7()]")
	if iss.Err() != nil {
		t.Fatal(iss.Err())
	}

	prog, err := env.Program(ast)
	if err != nil {
		t.Fatal(err)
	}

	val, _, err := prog.Eval(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(val.Value())
}
