package common

import (
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ cel.Library = (*celEnumExt)(nil)
var _ CELEnumType = (*celEnum)(nil)

type (
	celEnumExt struct{}
	celEnum    struct {
		desc   protoreflect.EnumDescriptor
		refVal ref.Val
	}
	CELEnumType interface {
		ref.Val
		EnumDescriptor() protoreflect.EnumDescriptor
		EnumNumber() protoreflect.EnumNumber
		EnumName(trimPrefix bool) string
	}
)

func CELEnumExt() cel.EnvOption {
	return cel.Lib(&celEnumExt{})
}

func (*celEnumExt) LibraryName() string {
	return "dtkt.ext.enum"
}

func (*celEnumExt) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("enumName",
			cel.Overload("enum_to_name",
				[]*cel.Type{cel.IntType},
				cel.StringType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					if len(args) > 0 {
						if enum, ok := args[0].(*celEnum); ok {
							return types.String(enum.EnumName(false))
						}
					}
					return types.NoSuchOverloadErr()
				}),
			),
			cel.Overload("enum_to_name_trimmed",
				[]*cel.Type{cel.IntType, cel.BoolType},
				cel.StringType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					if len(args) > 0 {
						if enum, ok := args[0].(*celEnum); ok {
							if len(args) == 2 {
								trimPrefix, _ := args[1].ConvertToType(cel.BoolType).Value().(bool)
								return types.String(enum.EnumName(trimPrefix))
							}
							return types.String(enum.EnumName(false))
						}
					}
					return types.NoSuchOverloadErr()
				}),
			),
		),
	}
}

func (*celEnumExt) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{}
}

func (e *celEnum) ConvertToNative(t reflect.Type) (any, error) {
	return e.refVal.ConvertToNative(t)
}

func (e *celEnum) ConvertToType(typeVal ref.Type) ref.Val {
	return e.refVal.ConvertToType(typeVal)
}

func (e *celEnum) Equal(other ref.Val) ref.Val {
	return e.refVal.Equal(other)
}

func (e *celEnum) Type() ref.Type {
	return e.refVal.Type()
}

func (e *celEnum) Value() any {
	return e.refVal.Value()
}

func (e *celEnum) EnumDescriptor() protoreflect.EnumDescriptor {
	return e.desc
}

func (e *celEnum) EnumNumber() protoreflect.EnumNumber {
	num, ok := e.refVal.Value().(int64)
	if !ok {
		return 0
	}
	return protoreflect.EnumNumber(num)
}

func (e *celEnum) EnumName(trimPrefix bool) string {
	// For enum fields, retrieve the raw numeric value and wrap it so that
	// enum methods are accessible.
	desc := e.EnumDescriptor().Values().ByNumber(e.EnumNumber())
	if desc == nil {
		return ""
	}

	if trimPrefix {
		return TrimProtoEnumPrefix(e.desc, string(desc.Name()))
	}

	return string(desc.Name())
}
