package form

import (
	"fmt"
	"reflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	OptionsElement interface {
		GetMethodName() string
		GetKeyExpr() string
		GetValExpr() string
		SetOption(Env, string, protoreflect.Value)
		isOptionsElement()
	}
	OptionsInterceptor func(string, any) bool
	StepValue          struct {
		Step  protopath.Step
		Value protoreflect.Value
	}
)

func LoadOptions(env Env, elem OptionsElement) error {
	if elem == nil || elem.GetMethodName() == "" || elem.GetKeyExpr() == "" || elem.GetValExpr() == "" {
		return nil
	}

	method, err := env.Resolver().FindMethodByName(protoreflect.FullName(elem.GetMethodName()))
	if err != nil {
		return fmt.Errorf("method not found: %s", elem.GetMethodName())
	}

	reqType, err := env.Resolver().FindMessageByName(method.Input().FullName())
	if err != nil {
		return fmt.Errorf("request type not found: %s", method.Input().FullName())
	}

	msg, _ := NewMessage(reqType.New())
	return env.OnGroupCompleted(msg.FieldGroup(), func(group *FieldGroup) error {
		res, err := env.Resolver().InvokeMethod(env.Context(), method.FullName(), group.Message().Get().Interface())
		if err != nil {
			return fmt.Errorf("invoke method %s: %w", elem.GetMethodName(), err)
		}

		return BuildOptions(env, elem, res.ProtoReflect())
	})
}

func BuildOptions(env Env, elem OptionsElement, msg protoreflect.Message) error {
	types, err := common.NewCELTypes(env.Resolver())
	if err != nil {
		return err
	}

	keyProg, err := common.CompileCELExpr(elem.GetKeyExpr(),
		cel.CustomTypeAdapter(types),
		cel.CustomTypeProvider(types),
		cel.Variable("this", cel.DynType),
	)
	if err != nil {
		return err
	}

	valProg, err := common.CompileCELExpr(elem.GetValExpr(),
		cel.CustomTypeAdapter(types),
		cel.CustomTypeProvider(types),
		cel.Variable("this", cel.DynType),
	)
	if err != nil {
		return err
	}

	keysRef, _, err := keyProg.ContextEval(env.Context(), map[string]any{
		"this": msg,
	})
	if err != nil {
		return err
	}

	valsRef, _, err := valProg.ContextEval(env.Context(), map[string]any{
		"this": msg,
	})
	if err != nil {
		return err
	}

	keys, err := keysRef.ConvertToNative(reflect.TypeFor[[]string]())
	if err != nil {
		return err
	}

	vals, err := valsRef.ConvertToNative(reflect.TypeFor[[]any]())
	if err != nil {
		return err
	}

	if keys, ok := keys.([]string); ok {
		if vals, ok := vals.([]any); ok {
			if len(keys) != len(vals) {
				return fmt.Errorf("keys of length %d != values of length %d", len(keys), len(vals))
			}

			for idx := range len(keys) {
				elem.SetOption(env, keys[idx], protoreflect.ValueOf(vals[idx]))
			}
		}
	}
	return nil
}
