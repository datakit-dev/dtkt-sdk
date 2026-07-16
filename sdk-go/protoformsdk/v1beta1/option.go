package v1beta1

import (
	"context"
	"fmt"
	"reflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	OptionsFilterField        = "filter"
	OptionsPageSizeField      = "page_size"
	OptionsPageTokenField     = "page_token"
	OptionsNextPageTokenField = "next_page_token"
	OptionsPageSize           = 100
)

type OptionsElement interface {
	GetFilterExpr() string
	GetLabelExpr() string
	GetMethodName() string
	GetResponseField() string
	GetUseFilterField() bool
	GetValueExpr() string
	SetOption(string, any)
	isOptionsElement()
}

func ResolveOptions(env Env, elem OptionsElement) error {
	if elem == nil || elem.GetMethodName() == "" || elem.GetResponseField() == "" || elem.GetLabelExpr() == "" {
		return nil
	}

	method, err := env.Resolver().FindMethodByName(protoreflect.FullName(elem.GetMethodName()))
	if err != nil {
		return fmt.Errorf("method not found: %s", elem.GetMethodName())
	}

	reqType, err := env.Resolver().FindMessageByName(method.Input().FullName())
	if err != nil {
		return fmt.Errorf("method %s request type %s not found", elem.GetMethodName(), method.Input().FullName())
	}

	resType, err := env.Resolver().FindMessageByName(method.Output().FullName())
	if err != nil {
		return fmt.Errorf("method %s response type %s not found", elem.GetMethodName(), method.Output().FullName())
	}

	var resField protoreflect.FieldDescriptor
	for idx := range resType.Descriptor().Fields().Len() {
		field := resType.Descriptor().Fields().Get(idx)

		if elem.GetResponseField() == string(field.Name()) {
			resField = field
			break
		}
	}

	if resField == nil {
		return fmt.Errorf("method %s response field %s not found", elem.GetMethodName(), elem.GetResponseField())
	} else if resField.Cardinality() != protoreflect.Repeated {
		return fmt.Errorf("method %s response field %s is not repeated", elem.GetMethodName(), elem.GetResponseField())
	}

	opts, err := loadOptions(env.Context(), env.Resolver(), elem, method, reqType.New())
	if err != nil {
		return err
	}

	return setOptions(env.Context(), env.Resolver(), elem, opts)
}

type optionsPage struct {
	options       []any
	nextPageToken string
}

func loadOptions(ctx context.Context, resolver Resolver, elem OptionsElement, method protoreflect.MethodDescriptor, req protoreflect.Message) (opts []any, err error) {
	loadPage := func(pageToken string) (page optionsPage, err error) {
		for idx := range req.Descriptor().Fields().Len() {
			field := req.Descriptor().Fields().Get(idx)

			switch field.Name() {
			case OptionsFilterField:
				if !elem.GetUseFilterField() || elem.GetFilterExpr() == "" {
					continue
				} else if field.Kind() == protoreflect.StringKind && field.Cardinality() != protoreflect.Repeated {
					req.Set(field, protoreflect.ValueOfString(elem.GetFilterExpr()))
				}
			case OptionsPageSizeField:
				if field.Cardinality() != protoreflect.Repeated {
					switch field.Kind() {
					case protoreflect.Int32Kind:
						req.Set(field, protoreflect.ValueOfInt32(OptionsPageSize))
					case protoreflect.Int64Kind:
						req.Set(field, protoreflect.ValueOfInt64(OptionsPageSize))
					case protoreflect.Uint32Kind:
						req.Set(field, protoreflect.ValueOfUint32(OptionsPageSize))
					case protoreflect.Uint64Kind:
						req.Set(field, protoreflect.ValueOfUint64(OptionsPageSize))
					}
				}
			case OptionsPageTokenField:
				if pageToken == "" {
					continue
				} else if field.Kind() == protoreflect.StringKind && field.Cardinality() != protoreflect.Repeated {
					req.Set(field, protoreflect.ValueOfString(pageToken))
				}
			}
		}

		res, err := resolver.InvokeMethod(ctx, method.FullName(), req.Interface())
		if err != nil {
			return
		}

		res.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, val protoreflect.Value) bool {
			switch string(field.Name()) {
			case elem.GetResponseField():
				if val.IsValid() {
					list := val.List()
					page.options = make([]any, list.Len())

					for idx := range list.Len() {
						opt := list.Get(idx)

						if field.Message() != nil {
							page.options[idx] = opt.Message().Interface()
						} else {
							page.options[idx] = opt.Interface()
						}
					}
				}
			case OptionsNextPageTokenField:
				if field.Kind() == protoreflect.StringKind && field.Cardinality() != protoreflect.Repeated {
					page.nextPageToken = val.String()
				}
			}

			return true
		})

		return
	}

	var page optionsPage
	for {
		page, err = loadPage(page.nextPageToken)
		if err != nil {
			return
		}

		if len(page.options) > 0 {
			opts = append(opts, page.options...)
		}

		if page.nextPageToken == "" {
			return
		}
	}
}

func setOptions(ctx context.Context, resolver common.CELResolver, elem OptionsElement, opts []any) error {
	types, err := common.NewCELTypes(resolver)
	if err != nil {
		return err
	}

	labelProg, err := common.CompileCELExpr(elem.GetLabelExpr(),
		cel.CustomTypeAdapter(types),
		cel.CustomTypeProvider(types),
		cel.Variable("this", cel.DynType),
	)
	if err != nil {
		return err
	}

	var valueProg cel.Program
	if elem.GetValueExpr() != "" {
		valueProg, err = common.CompileCELExpr(elem.GetValueExpr(),
			cel.CustomTypeAdapter(types),
			cel.CustomTypeProvider(types),
			cel.Variable("this", cel.DynType),
		)
		if err != nil {
			return err
		}
	}

	var filterProg cel.Program
	if elem.GetFilterExpr() != "" {
		filterProg, err = common.CompileCELExpr(elem.GetFilterExpr(),
			cel.CustomTypeAdapter(types),
			cel.CustomTypeProvider(types),
			cel.Variable("this", cel.DynType),
		)
		if err != nil {
			return err
		}
	}

	for _, opt := range opts {
		if filterProg != nil {
			filter_, _, err := filterProg.ContextEval(ctx, map[string]any{
				"this": opt,
			})
			if err != nil {
				return err
			}

			filter, err := filter_.ConvertToNative(reflect.TypeFor[bool]())
			if err != nil {
				return err
			} else if !filter.(bool) {
				continue
			}
		}

		labelRef, _, err := labelProg.ContextEval(ctx, map[string]any{
			"this": opt,
		})
		if err != nil {
			return err
		}

		label, err := labelRef.ConvertToNative(reflect.TypeFor[any]())
		if err != nil {
			return err
		}

		var value any
		if valueProg != nil {
			valueRef, _, err := valueProg.ContextEval(ctx, map[string]any{
				"this": opt,
			})
			if err != nil {
				return err
			}

			value, err = valueRef.ConvertToNative(reflect.TypeFor[any]())
			if err != nil {
				return err
			}
		} else {
			value = opt
		}

		if msg, ok := value.(proto.Message); ok {
			value = msg.ProtoReflect()
		}

		elem.SetOption(fmt.Sprint(label), protoreflect.ValueOf(value))
	}

	return nil
}
