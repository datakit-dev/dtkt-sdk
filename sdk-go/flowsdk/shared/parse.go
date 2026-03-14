package shared

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
)

type (
	ParseNodeFunc func(string, *cel.Ast)
	ParseExprFunc func(*cel.Ast)
)

func IsInvalidExprError(err error) bool {
	return errors.Is(err, invalidExprErr)
}

func IsValidExpr(expr string) (string, bool) {
	matches := validExpr.FindStringSubmatch(expr)

	if len(matches) != 1 {
		return "", false
	}

	celExpr := strings.TrimSpace(expr[len(matches[0]):])

	if len(celExpr) == 0 {
		return "", false
	}

	return celExpr, true
}

func ParseExprWithEnv(env *cel.Env, expr string, visitor ParseExprFunc) (*cel.Ast, error) {
	expr, valid := IsValidExpr(expr)
	if !valid {
		return nil, NewExprError(fmt.Sprintf("%s: %s", InvalidExprErrPrefix, expr))
	}

	ast, iss := env.Parse(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	ast, iss = env.Check(ast)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	if visitor != nil {
		visitor(ast)
	}

	return ast, nil
}

func ParseExpr(run Runtime, expr string, visitor ParseExprFunc) (*cel.Ast, error) {
	env, err := run.Env()
	if err != nil {
		return nil, err
	}

	return ParseExprWithEnv(env, expr, visitor)
}
