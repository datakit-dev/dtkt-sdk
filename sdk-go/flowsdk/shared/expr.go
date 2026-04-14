package shared

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
)

const InvalidExprErrPrefix = "invalid expression"

var validExpr = regexp.MustCompile(`^\s?=\s?`)

var _ error = (*ExprError)(nil)

type (
	ExprError struct {
		msg string
	}
	ExprVisitFunc func(expr *cel.Ast)
	NodeVisitFunc func(id string, expr *cel.Ast)
)

func (f NodeVisitFunc) ExprVisitor(id string) ExprVisitFunc {
	return func(expr *cel.Ast) {
		f(id, expr)
	}
}

func ParseExpr(env Env, expr string, visitor ExprVisitFunc) (*cel.Ast, error) {
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

func CompileExpr(run Runtime, expr string, opts ...cel.ProgramOption) (cel.Program, error) {
	expr, valid := IsValidExpr(expr)
	if !valid {
		return nil, NewExprError(fmt.Sprintf("%s: %s", InvalidExprErrPrefix, expr))
	}

	env, err := run.Env()
	if err != nil {
		return nil, err
	}

	opts = append(opts, cel.InterruptCheckFrequency(1))

	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	return env.Program(ast, opts...)
}

func NewExprError(err string) *ExprError {
	return &ExprError{
		msg: err,
	}
}

func IsInvalidExprError(err error) bool {
	exprErr := new(ExprError)
	return errors.As(err, &exprErr)
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

func (e *ExprError) Is(err error) bool {
	return IsInvalidExprError(err)
}

func (e *ExprError) Error() string {
	return e.msg
}
