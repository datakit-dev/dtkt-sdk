package common

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/jhump/protoreflect/v2/protoresolve"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	CELTypes struct {
		resolver     CELResolver
		registry     *types.Registry
		fieldAliases map[string]map[string]string
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

	registry, err := types.NewRegistry()
	if err != nil {
		return nil, err
	}

	resolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
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
		resolver:     resolver,
		registry:     registry,
		fieldAliases: map[string]map[string]string{},
	}, nil
}

func (t *CELTypes) RegisterDescriptor(fd protoreflect.FileDescriptor) error {
	return t.registry.RegisterDescriptor(fd)
}

func (t *CELTypes) RegisterType(types ...ref.Type) error {
	return t.registry.RegisterType(types...)
}

func (t *CELTypes) NativeToValue(val any) ref.Val {
	switch val := val.(type) {
	case proto.Message:
		if err := t.registerMessage(val); err != nil {
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
	case ref.Val:
		// Already a CEL value, pass through to registry.
	default:
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				return types.NullValue
			}
			return t.NativeToValue(rv.Elem().Interface())
		}
		if rv.Kind() == reflect.Struct {
			sv, err := structToValue(val)
			if err != nil {
				return types.WrapErr(err)
			}
			return t.NativeToValue(sv)
		}
	}
	return t.wrapRefVal(t.registry.NativeToValue(val))
}

// structToValue converts a plain Go struct to *structpb.Value via JSON
// round-trip so CEL can evaluate expressions against it.
func structToValue(val any) (*structpb.Value, error) {
	b, err := encoding.ToJSONV2(val)
	if err != nil {
		return nil, err
	}
	sv := &structpb.Value{}
	if err = encoding.FromJSONV2(b, sv); err != nil {
		return nil, err
	}
	return sv, nil
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

func (t *CELTypes) registerMessage(msg proto.Message) error {
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

	aliases := t.aliasMap(msg.ProtoReflect().Descriptor())
	if len(aliases) == 0 {
		return val
	}

	fieldTester, _ := val.(traits.FieldTester)

	return &celMessage{
		msg:         msg,
		refVal:      val,
		indexer:     indexer,
		fieldTester: fieldTester,
		adapter:     t,
		aliases:     aliases,
	}
}

func (t *CELTypes) aliasMap(desc protoreflect.MessageDescriptor) map[string]string {
	fullName := string(desc.FullName())
	if cached, ok := t.fieldAliases[fullName]; ok {
		return cached
	}

	fields := desc.Fields()
	aliases := make(map[string]string, fields.Len()*2)
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		protoName := string(fd.Name())
		jsonName := fd.JSONName()
		aliases[protoName] = protoName
		aliases[jsonName] = protoName
	}

	t.fieldAliases[fullName] = aliases
	return aliases
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
