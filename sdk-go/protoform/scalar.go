package protoform

import (
	"fmt"
	"strconv"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Binding[bool] = (*ScalarField[bool])(nil)
var _ ScalarFieldType = (*ScalarField[string])(nil)

type (
	ScalarField[T ScalarValueType] struct {
		Getter[T]
		Setter[T]
		Parser[T]
		Stringer[T]
		parent     *Message
		descriptor protoreflect.FieldDescriptor
	}
	ScalarFieldType interface {
		Field
		GetAny() any
		SetAny(any) error
		String() string
		isScalar()
	}
	ScalarValueType interface {
		bool | []byte | int32 | int64 | uint32 | uint64 | float32 | float64 | string | protoreflect.EnumNumber
	}
)

func NewScalarField(parent *Message, field protoreflect.FieldDescriptor) (ScalarFieldType, bool) {
	var scalar ScalarFieldType
	switch field.Kind() {
	case protoreflect.BoolKind:
		scalar = &ScalarField[bool]{
			Getter:     GetterFunc[bool](func() bool { return parent.GetField(field).Bool() }),
			Setter:     SetterFunc[bool](func(v bool) { parent.SetField(field, protoreflect.ValueOfBool(v)) }),
			Parser:     ParserFunc[bool](strconv.ParseBool),
			Stringer:   StringerFunc[bool](strconv.FormatBool),
			parent:     parent,
			descriptor: field,
		}
	case protoreflect.BytesKind:
		scalar = &ScalarField[[]byte]{
			Getter:     GetterFunc[[]byte](func() []byte { return parent.GetField(field).Bytes() }),
			Setter:     SetterFunc[[]byte](func(v []byte) { parent.SetField(field, protoreflect.ValueOfBytes(v)) }),
			Parser:     ParserFunc[[]byte](func(s string) ([]byte, error) { return []byte(s), nil }),
			Stringer:   StringerFunc[[]byte](func(v []byte) string { return string(v) }),
			parent:     parent,
			descriptor: field,
		}
	case protoreflect.EnumKind:
		scalar = &ScalarField[protoreflect.EnumNumber]{
			Getter: GetterFunc[protoreflect.EnumNumber](func() protoreflect.EnumNumber { return parent.GetField(field).Enum() }),
			Setter: SetterFunc[protoreflect.EnumNumber](func(v protoreflect.EnumNumber) { parent.SetField(field, protoreflect.ValueOfEnum(v)) }),
			Parser: ParserFunc[protoreflect.EnumNumber](func(s string) (protoreflect.EnumNumber, error) {
				return common.ProtoEnumNumberFromString(field.Enum(), s), nil
			}),
			Stringer:   StringerFunc[protoreflect.EnumNumber](func(v protoreflect.EnumNumber) string { return common.ProtoEnumNumberToString(field.Enum(), v) }),
			parent:     parent,
			descriptor: field,
		}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		scalar = &ScalarField[int32]{
			Getter:     GetterFunc[int32](func() int32 { return int32(parent.GetField(field).Int()) }),
			Setter:     SetterFunc[int32](func(v int32) { parent.SetField(field, protoreflect.ValueOfInt32(v)) }),
			Parser:     ParserFunc[int32](util.ParseInt32),
			Stringer:   StringerFunc[int32](func(i int32) string { return strconv.FormatInt(int64(i), 10) }),
			parent:     parent,
			descriptor: field,
		}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		scalar = &ScalarField[int64]{
			Getter:     GetterFunc[int64](func() int64 { return parent.GetField(field).Int() }),
			Setter:     SetterFunc[int64](func(v int64) { parent.SetField(field, protoreflect.ValueOfInt64(v)) }),
			Parser:     ParserFunc[int64](util.ParseInt64),
			Stringer:   StringerFunc[int64](func(i int64) string { return strconv.FormatInt(i, 10) }),
			parent:     parent,
			descriptor: field,
		}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		scalar = &ScalarField[uint32]{
			Getter:     GetterFunc[uint32](func() uint32 { return uint32(parent.GetField(field).Uint()) }),
			Setter:     SetterFunc[uint32](func(v uint32) { parent.SetField(field, protoreflect.ValueOfUint32(v)) }),
			Parser:     ParserFunc[uint32](util.ParseUInt32),
			Stringer:   StringerFunc[uint32](func(i uint32) string { return strconv.FormatUint(uint64(i), 10) }),
			parent:     parent,
			descriptor: field,
		}
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		scalar = &ScalarField[uint64]{
			Getter:     GetterFunc[uint64](func() uint64 { return uint64(parent.GetField(field).Uint()) }),
			Setter:     SetterFunc[uint64](func(v uint64) { parent.SetField(field, protoreflect.ValueOfUint64(v)) }),
			Parser:     ParserFunc[uint64](util.ParseUInt64),
			Stringer:   StringerFunc[uint64](func(i uint64) string { return strconv.FormatUint(i, 10) }),
			parent:     parent,
			descriptor: field,
		}
	case protoreflect.FloatKind:
		scalar = &ScalarField[float32]{
			Getter:     GetterFunc[float32](func() float32 { return float32(parent.GetField(field).Float()) }),
			Setter:     SetterFunc[float32](func(v float32) { parent.SetField(field, protoreflect.ValueOfFloat32(v)) }),
			Parser:     ParserFunc[float32](util.ParseFloat32),
			Stringer:   StringerFunc[float32](func(i float32) string { return strconv.FormatInt(int64(i), 10) }),
			parent:     parent,
			descriptor: field,
		}
	case protoreflect.DoubleKind:
		scalar = &ScalarField[float64]{
			Getter:     GetterFunc[float64](func() float64 { return parent.GetField(field).Float() }),
			Setter:     SetterFunc[float64](func(v float64) { parent.SetField(field, protoreflect.ValueOfFloat64(v)) }),
			Parser:     ParserFunc[float64](util.ParseFloat64),
			Stringer:   StringerFunc[float64](func(i float64) string { return fmt.Sprintf("%f", i) }),
			parent:     parent,
			descriptor: field,
		}
	case protoreflect.StringKind:
		scalar = &ScalarField[string]{
			Getter:     GetterFunc[string](func() string { return parent.GetField(field).String() }),
			Setter:     SetterFunc[string](func(s string) { parent.SetField(field, protoreflect.ValueOfString(s)) }),
			Parser:     ParserFunc[string](func(s string) (string, error) { return s, nil }),
			Stringer:   StringerFunc[string](func(s string) string { return s }),
			parent:     parent,
			descriptor: field,
		}
	default:
		return nil, false
	}

	return scalar, true
}

func (s *ScalarField[T]) Validate(env Env) error {
	return s.parent.ValidateField(env, s.descriptor, protoreflect.ValueOf(s.Get()))
}

func (s *ScalarField[T]) Parent() *Message {
	return s.parent
}

func (s *ScalarField[T]) Descriptor() protoreflect.FieldDescriptor {
	return s.descriptor
}

func (s *ScalarField[T]) Get() (v T) {
	if s.Getter != nil {
		return s.Getter.Get()
	}
	return
}

func (s *ScalarField[T]) Set(v T) {
	if s.descriptor.Default().Equal(protoreflect.ValueOf(v)) {
		s.parent.ClearField(s.descriptor)
	} else if s.Setter != nil {
		s.Setter.Set(v)
	}
}

func (s *ScalarField[T]) GetAny() any {
	return s.Get()
}

func (s *ScalarField[T]) SetAny(v any) error {
	if v, ok := v.(T); ok {
		s.Set(v)
		return nil
	}

	value, err := s.Parse(util.StringFormatAny(v))
	if err != nil {
		return err
	}

	s.Set(value)

	return nil
}

func (m *ScalarField[T]) Parse(str string) (T, error) {
	if m.Parser != nil {
		return m.Parser.Parse(str)
	}

	var v T
	switch any(v).(type) {
	case []byte:
		return any([]byte(str)).(T), nil
	case protoreflect.EnumNumber:
		num, err := util.ScanValueFor[int32](str)
		if err != nil {
			return v, err
		}
		return any(protoreflect.EnumNumber(num)).(T), nil
	}

	val, err := util.ScanValue(v, str)
	if err != nil {
		return v, err
	}

	return val.(T), nil
}

func (s *ScalarField[T]) StringOf(v T) string {
	if s.Stringer != nil {
		return s.Stringer.StringOf(v)
	}
	return util.StringFormatAny(v)
}

func (s *ScalarField[T]) String() string {
	return s.StringOf(s.Get())
}

func (f *ScalarField[T]) isField() {}

func (f *ScalarField[T]) isScalar() {}
