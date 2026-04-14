package runtime

import (
	"fmt"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
)

// compiledRequestKind identifies the type of a compiled request tree node.
type compiledRequestKind int

const (
	compiledRequestLiteral compiledRequestKind = iota
	compiledRequestCEL
	compiledRequestList
	compiledRequestStruct
)

// compiledRequest is the compiled (executable) representation of a
// structpb.Value tree from MethodCall.request. CEL leaves hold a compiled
// cel.Program; literals hold a static ref.Val.
type compiledRequest struct {
	kind    compiledRequestKind
	literal ref.Val                     // compiledRequestLiteral
	program cel.Program                 // compiledRequestCEL
	list    []*compiledRequest          // compiledRequestList
	fields  map[string]*compiledRequest // compiledRequestStruct
}

// lintRequestTree recursively walks a structpb.Value and validates all CEL
// expression leaves via parseCEL (AST-only). Returns structured diagnostics.
// The path parameter tracks location within the request tree.
func lintRequestTree(v *structpb.Value, path string) []LintDiagnostic {
	if v == nil {
		return nil
	}
	switch v.GetKind().(type) {
	case *structpb.Value_StringValue:
		s := v.GetStringValue()
		if _, ok := shared.IsValidExpr(s); ok {
			if _, err := parseCEL(s); err != nil {
				return []LintDiagnostic{{
					Severity: SeverityError,
					Path:     path,
					Message:  err.Error(),
					Code:     CodeInvalidCEL,
				}}
			}
		}
	case *structpb.Value_StructValue:
		var diags []LintDiagnostic
		for key, field := range v.GetStructValue().GetFields() {
			diags = append(diags, lintRequestTree(field, path+"."+key)...)
		}
		return diags
	case *structpb.Value_ListValue:
		var diags []LintDiagnostic
		for i, elem := range v.GetListValue().GetValues() {
			diags = append(diags, lintRequestTree(elem, fmt.Sprintf("%s[%d]", path, i))...)
		}
		return diags
	}
	return nil
}

// compileRequestTree recursively walks a structpb.Value, compiling CEL
// expression leaves into executable programs. Used by the compile phase at
// executor creation time.
func compileRequestTree(env shared.Env, v *structpb.Value) (*compiledRequest, error) {
	if v == nil {
		return &compiledRequest{kind: compiledRequestLiteral, literal: types.NullValue}, nil
	}
	switch v.GetKind().(type) {
	case *structpb.Value_NullValue:
		return &compiledRequest{kind: compiledRequestLiteral, literal: types.NullValue}, nil
	case *structpb.Value_BoolValue:
		return &compiledRequest{kind: compiledRequestLiteral, literal: types.Bool(v.GetBoolValue())}, nil
	case *structpb.Value_NumberValue:
		return &compiledRequest{kind: compiledRequestLiteral, literal: types.Double(v.GetNumberValue())}, nil
	case *structpb.Value_StringValue:
		s := v.GetStringValue()
		if _, ok := shared.IsValidExpr(s); ok {
			prog, err := compileCEL(env, s)
			if err != nil {
				return nil, err
			}
			return &compiledRequest{kind: compiledRequestCEL, program: prog}, nil
		}
		return &compiledRequest{kind: compiledRequestLiteral, literal: types.String(s)}, nil
	case *structpb.Value_ListValue:
		items := v.GetListValue().GetValues()
		list := make([]*compiledRequest, len(items))
		for i, elem := range items {
			c, err := compileRequestTree(env, elem)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", i, err)
			}
			list[i] = c
		}
		return &compiledRequest{kind: compiledRequestList, list: list}, nil
	case *structpb.Value_StructValue:
		fields := v.GetStructValue().GetFields()
		compiled := make(map[string]*compiledRequest, len(fields))
		for key, field := range fields {
			c, err := compileRequestTree(env, field)
			if err != nil {
				return nil, fmt.Errorf(".%s: %w", key, err)
			}
			compiled[key] = c
		}
		return &compiledRequest{kind: compiledRequestStruct, fields: compiled}, nil
	default:
		return &compiledRequest{kind: compiledRequestLiteral, literal: types.NullValue}, nil
	}
}

// evalRequest evaluates a compiled request tree against a CEL activation,
// producing a ref.Val suitable for conversion to an RPC request.
func evalRequest(cr *compiledRequest, vars map[string]any) (ref.Val, error) {
	switch cr.kind {
	case compiledRequestLiteral:
		return cr.literal, nil
	case compiledRequestCEL:
		return evalCEL(cr.program, vars)
	case compiledRequestList:
		vals := make([]ref.Val, len(cr.list))
		for i, item := range cr.list {
			v, err := evalRequest(item, vars)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", i, err)
			}
			vals[i] = v
		}
		return types.DefaultTypeAdapter.NativeToValue(vals), nil
	case compiledRequestStruct:
		m := make(map[ref.Val]ref.Val, len(cr.fields))
		for key, field := range cr.fields {
			v, err := evalRequest(field, vars)
			if err != nil {
				return nil, fmt.Errorf(".%s: %w", key, err)
			}
			m[types.String(key)] = v
		}
		return types.DefaultTypeAdapter.NativeToValue(m), nil
	default:
		return types.NullValue, nil
	}
}

// resolveRequestValue resolves the RPC request value. If a compiled request
// tree is provided, it evaluates it against the activation. Otherwise it
// falls back to the activation's first input value.
func resolveRequestValue(req *compiledRequest, act *activation, vars map[string]any) (*expr.Value, error) {
	if req == nil {
		return act.FirstInputValue(), nil
	}
	result, err := evalRequest(req, vars)
	if err != nil {
		return nil, err
	}
	return refValToExpr(result)
}

// transformResponse evaluates the response CEL expression with this.response
// bound to the raw RPC result. If responseProg is nil, falls back to
// responseToExpr (current behavior).
func transformResponse(responseProg cel.Program, resp proto.Message, adapter types.Adapter) (*expr.Value, error) {
	if responseProg == nil {
		return responseToExpr(resp)
	}
	respVal, err := responseToExpr(resp)
	if err != nil {
		return nil, err
	}
	result, err := evalCEL(responseProg, map[string]any{
		"this": map[string]any{
			"response": exprToRefVal(adapter, respVal),
		},
	})
	if err != nil {
		return nil, err
	}
	return refValToExpr(result)
}
