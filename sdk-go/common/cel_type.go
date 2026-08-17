package common

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"buf.build/go/protovalidate"
	"cel.dev/expr"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/jhump/protoreflect/v2/protoresolve"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ types.Adapter = (*CELTypes)(nil)
var _ types.Provider = (*CELTypes)(nil)

type (
	CELTypes struct {
		resolver CELResolver
		registry *types.Registry
	}
	CELResolver interface {
		protoresolve.SerializationResolver
		RangeServices(func(protoreflect.ServiceDescriptor) bool)
		RangeMethods(func(protoreflect.MethodDescriptor) bool)
		FindMethodByName(protoreflect.FullName) (protoreflect.MethodDescriptor, error)
		RangeFiles(func(protoreflect.FileDescriptor) bool)
		GetValidator() (protovalidate.Validator, error)
	}
)

func NewCELTypes(resolver CELResolver) (*CELTypes, error) {
	if resolver == nil {
		return nil, fmt.Errorf("resolver required")
	}

	registry, err := types.NewProtoRegistry(
		types.JSONFieldNames(true),
	)
	if err != nil {
		return nil, err
	}

	resolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd == nil {
			return true
		}

		err = registry.RegisterDescriptor(fd)
		if err != nil {
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return &CELTypes{
		resolver: resolver,
		registry: registry,
	}, nil
}

func (t *CELTypes) Copy() *CELTypes {
	return &CELTypes{
		resolver: t.resolver,
		registry: t.registry.Copy(),
	}
}

func (t *CELTypes) Resolver() CELResolver {
	return t.resolver
}

func (t *CELTypes) RegisterDescriptor(fd protoreflect.FileDescriptor) error {
	return t.registry.RegisterDescriptor(fd)
}

func (t *CELTypes) RegisterType(types ...ref.Type) error {
	return t.registry.RegisterType(types...)
}

func (t *CELTypes) ExprValueToNative(exprVal *expr.ExprValue) (any, error) {
	if exprErr := exprVal.GetError(); exprErr != nil {
		err := errors.Join(util.SliceMap(exprErr.GetErrors(), func(e *expr.Status) error {
			return status.FromProto(&spb.Status{
				Code:    e.Code,
				Message: e.Message,
				Details: e.Details,
			}).Err()
		})...)
		if err != nil {
			return nil, err
		}
	} else if exprVal.GetUnknown() != nil {
		return nil, status.Error(codes.Unknown, "unknown error")
	}

	value := exprVal.GetValue()
	if value == nil {
		return nil, nil
	}

	switch value.GetKind().(type) {
	case *expr.Value_NullValue:
		return structpb.NewNullValue(), nil
	case *expr.Value_ObjectValue:
		return anypb.UnmarshalNew(value.GetObjectValue(), proto.UnmarshalOptions{
			DiscardUnknown: true,
			Resolver:       t.resolver,
		})
	case *expr.Value_MapValue:
		entries := value.GetMapValue().GetEntries()
		if len(entries) == 0 {
			return nil, fmt.Errorf("unknown map type")
		}

		keys, err := util.SliceMapError(entries, func(entry *expr.MapValue_Entry) (any, error) {
			return t.ExprValueToNative(&expr.ExprValue{
				Kind: &expr.ExprValue_Value{
					Value: entry.GetKey(),
				},
			})
		})
		if err != nil {
			return nil, err
		}

		values, err := util.SliceMapError(entries, func(entry *expr.MapValue_Entry) (any, error) {
			return t.ExprValueToNative(&expr.ExprValue{
				Kind: &expr.ExprValue_Value{
					Value: entry.GetValue(),
				},
			})
		})
		if err != nil {
			return nil, err
		}

		keyType, valType := reflect.TypeOf(keys[0]), reflect.TypeFor[any]()
		native := reflect.MakeMapWithSize(reflect.MapOf(keyType, valType), len(keys))
		for idx := range len(keys) {
			native.SetMapIndex(reflect.ValueOf(keys[idx]), reflect.ValueOf(values[idx]))
		}
		return native.Interface(), nil
	case *expr.Value_ListValue:
		values := value.GetListValue().GetValues()
		native := make([]any, len(values))
		for idx, value := range values {
			val, err := t.ExprValueToNative(&expr.ExprValue{
				Kind: &expr.ExprValue_Value{
					Value: value,
				},
			})
			if err != nil {
				return nil, err
			}
			native[idx] = val
		}
		return native, nil
	}

	refVal, err := cel.ProtoAsValue(t, value)
	if err != nil {
		return nil, err
	}

	return refVal.Value(), nil
}

func (t *CELTypes) ValueToExprValue(refVal ref.Val) *expr.ExprValue {
	switch refVal.Type() {
	case types.NullType:
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_NullValue{
						NullValue: structpb.NullValue_NULL_VALUE,
					},
				},
			},
		}
	case types.BoolType:
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_BoolValue{BoolValue: refVal.Value().(bool)},
				},
			},
		}
	case types.BytesType:
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_BytesValue{BytesValue: refVal.Value().([]byte)},
				},
			},
		}
	case types.DoubleType:
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_DoubleValue{DoubleValue: refVal.Value().(float64)},
				},
			},
		}
	case types.IntType:
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_Int64Value{Int64Value: refVal.Value().(int64)},
				},
			},
		}
	case types.ListType:
		l := refVal.(traits.Lister)
		sz := l.Size().(types.Int)
		elts := make([]*expr.Value, 0, int64(sz))
		for i := types.Int(0); i < sz; i++ {
			elts = append(elts, t.ValueToExprValue(l.Get(i)).GetValue())
		}
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_ListValue{
						ListValue: &expr.ListValue{Values: elts},
					},
				},
			},
		}
	case types.MapType:
		mapper := refVal.(traits.Mapper)
		sz := mapper.Size().(types.Int)
		entries := make([]*expr.MapValue_Entry, 0, int64(sz))
		for it := mapper.Iterator(); it.HasNext().(types.Bool); {
			k := it.Next()
			v := mapper.Get(k)
			entries = append(entries, &expr.MapValue_Entry{Key: t.ValueToExprValue(k).GetValue(), Value: t.ValueToExprValue(v).GetValue()})
		}
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_MapValue{
						MapValue: &expr.MapValue{Entries: entries},
					},
				},
			},
		}
	case types.StringType:
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_StringValue{StringValue: refVal.Value().(string)},
				},
			},
		}
	case types.TypeType:
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_TypeValue{TypeValue: refVal.(ref.Type).TypeName()},
				},
			},
		}
	case types.UintType:
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_Uint64Value{Uint64Value: refVal.Value().(uint64)},
				},
			},
		}
	default:
		obj, err := ProtoOptions{Resolver: t.resolver}.WrapAny(refVal.Value())
		if err != nil {
			return &expr.ExprValue{
				Kind: &expr.ExprValue_Error{
					Error: &expr.ErrorSet{
						Errors: []*expr.Status{
							{Code: int32(codes.Unknown), Message: err.Error()},
						},
					},
				},
			}
		}
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Value{
				Value: &expr.Value{
					Kind: &expr.Value_ObjectValue{ObjectValue: obj},
				},
			},
		}
	}
}

func (t *CELTypes) ValueToNative(refVal ref.Val) (any, error) {
	return t.ExprValueToNative(t.ValueToExprValue(refVal))
}

func (t *CELTypes) NativeToValue(val any) ref.Val {
	if val == nil {
		return types.NullValue
	}

	switch val := val.(type) {
	case error:
		return types.WrapErr(val)
	case *anypb.Any:
		v, err := ProtoOptions{
			Resolver: t.resolver,
		}.UnwrapAny(val)
		if err != nil {
			return types.NewErr("failed to unmarshal dynamic type %q: %v", val.GetTypeUrl(), err)
		}

		return t.NativeToValue(v)
	case proto.Message:
		if err := t.RegisterMessage(val); err != nil {
			return types.WrapErr(err)
		}
	case protoreflect.Message:
		return t.NativeToValue(val.Interface())
	case protoreflect.List:
		return t.wrapRefVal(t.registry.NativeToValue(val))
	case protoreflect.Enum:
		return &celEnum{
			desc:   val.Descriptor(),
			refVal: types.Int(val.Number()),
		}
	}

	return t.wrapRefVal(t.registry.NativeToValue(val))
}

func (t *CELTypes) FindIdent(identName string) (ref.Val, bool) {
	return t.registry.FindIdent(identName)
}

func (t *CELTypes) EnumValue(enumName string) ref.Val {
	return t.registry.EnumValue(enumName)
}

func (t *CELTypes) FindStructType(structType string) (*types.Type, bool) {
	return t.registry.FindStructType(structType)
}

func (t *CELTypes) FindStructFieldNames(structType string) ([]string, bool) {
	return t.registry.FindStructFieldNames(structType)
}

func (t *CELTypes) FindStructFieldType(structType, fieldName string) (*types.FieldType, bool) {
	if ft, ok := t.registry.FindStructFieldType(structType, fieldName); ok {
		return ft, true
	}

	normalized := normalizeProtoFieldName(fieldName)
	if normalized == fieldName {
		return nil, false
	}

	return t.registry.FindStructFieldType(structType, normalized)
}

func (t *CELTypes) NewValue(structType string, fields map[string]ref.Val) ref.Val {
	return t.registry.NewValue(structType, fields)
}

func (t *CELTypes) RegisterMessage(msg proto.Message) error {
	err := protorange.Options{Resolver: t.resolver}.Range(msg.ProtoReflect(), func(values protopath.Values) error {
		v := values.Index(-1)

		switch v.Step.Kind() {
		case protopath.AnyExpandStep:
			return t.registry.RegisterMessage(v.Value.Message().Interface())
		}

		return nil
	}, nil)
	if err != nil {
		return err
	}

	return t.registry.RegisterMessage(msg)
}

func (t *CELTypes) wrapRefVal(val ref.Val) ref.Val {
	if val == nil {
		return nil
	}

	switch val.(type) {
	case *celMessage, *celList, *celMap, *celIterator, *celEnum:
		return val
	}

	msg, ok := val.Value().(proto.Message)
	if ok {
		if IsWellKnownName(msg.ProtoReflect().Descriptor().FullName()) {
			return val
		}

		return t.wrapProtoMessage(val, msg)
	}

	if lister, ok := val.(traits.Lister); ok {
		return &celList{
			refVal:  val,
			lister:  lister,
			adapter: t,
		}
	}

	if mapper, ok := val.(traits.Mapper); ok {
		return &celMap{
			refVal:  val,
			mapper:  mapper,
			adapter: t,
		}
	}

	if iterator, ok := val.(traits.Iterator); ok {
		return &celIterator{
			refVal:   val,
			iterator: iterator,
			adapter:  t,
		}
	}

	return val
}

func (t *CELTypes) wrapProtoMessage(val ref.Val, msg proto.Message) ref.Val {
	indexer, ok := val.(traits.Indexer)
	if !ok {
		return val
	}

	fieldTester, _ := val.(traits.FieldTester)

	return &celMessage{
		adapter:     t,
		msg:         msg,
		refVal:      val,
		indexer:     indexer,
		fieldTester: fieldTester,
	}
}

func (t *CELTypes) wrapIterator(iter traits.Iterator) traits.Iterator {
	if iter == nil {
		return nil
	}
	if _, ok := iter.(*celIterator); ok {
		return iter
	}

	refVal, ok := iter.(ref.Val)
	if !ok {
		return iter
	}

	return &celIterator{
		refVal:   refVal,
		iterator: iter,
		adapter:  t,
	}
}

func normalizeProtoFieldName(name string) string {
	if name == "" {
		return name
	}

	if strings.ContainsRune(name, '_') {
		return name
	}

	var b strings.Builder
	for idx, r := range name {
		if unicode.IsUpper(r) && idx > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}

	return b.String()
}
