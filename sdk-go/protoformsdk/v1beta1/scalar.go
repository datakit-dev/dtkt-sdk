package v1beta1

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Binding[bool] = (*Scalar[bool])(nil)
var _ FieldType = (*ScalarField)(nil)

type (
	ScalarField struct {
		ScalarType
		element *Element
	}
	Scalar[T ScalarValue] struct {
		Getter[T]
		Setter[T]
		Parser[T]
		Stringer[T]
		parent     *Message
		descriptor protoreflect.FieldDescriptor
	}
	ScalarType interface {
		Parent() *Message
		Descriptor() protoreflect.FieldDescriptor
		GetAny() any
		SetAny(any) error
		String() string
	}
	ScalarValue interface {
		bool | []byte | int32 | int64 | uint32 | uint64 | float32 | float64 | string | protoreflect.EnumNumber
	}
)

func NewScalarField(parent *Message, field protoreflect.FieldDescriptor) (*ScalarField, bool) {
	scalar, ok := NewScalar(parent, field)
	if !ok {
		return nil, false
	}

	element := NewElement(field)
	if !element.IsValid() {
		switch field.Kind() {
		case protoreflect.BoolKind:
			element.AsConfirm()
		case protoreflect.EnumKind:
			element.AsSelect(func(selec *SelectElement) {
				rules := element.rules.GetEnum()
				for idx := range field.Enum().Values().Len() {
					desc := field.Enum().Values().Get(idx)
					if rules != nil && len(rules.In) > 0 && !slices.Contains(rules.In, int32(desc.Number())) {
						continue
					}

					if rules != nil && len(rules.NotIn) > 0 && slices.Contains(rules.NotIn, int32(desc.Number())) {
						continue
					}

					selec.options = append(selec.options, util.NewMapPair(string(desc.Name()), any(protoreflect.ValueOfEnum(desc.Number()))))
				}
			})
		default:
			element.AsInput(func(input *InputElement) {
				input.AsMultiline(field.Kind() == protoreflect.BytesKind)
			})
		}
	}

	return &ScalarField{
		ScalarType: scalar,
		element:    element,
	}, true
}

func NewScalar(parent *Message, field protoreflect.FieldDescriptor) (ScalarType, bool) {
	if parent == nil || field == nil {
		return nil, false
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		return &Scalar[bool]{
			Getter:     GetterFunc[bool](func() bool { return parent.Get().Get(field).Bool() }),
			Setter:     SetterFunc[bool](func(v bool) { parent.Get().Set(field, protoreflect.ValueOfBool(v)) }),
			Parser:     ParserFunc[bool](strconv.ParseBool),
			parent:     parent,
			descriptor: field,
		}, true
	case protoreflect.BytesKind:
		return &Scalar[[]byte]{
			Getter:     GetterFunc[[]byte](func() []byte { return parent.Get().Get(field).Bytes() }),
			Setter:     SetterFunc[[]byte](func(v []byte) { parent.Get().Set(field, protoreflect.ValueOfBytes(v)) }),
			Parser:     ParserFunc[[]byte](func(s string) ([]byte, error) { return []byte(s), nil }),
			parent:     parent,
			descriptor: field,
		}, true
	case protoreflect.EnumKind:
		return &Scalar[protoreflect.EnumNumber]{
			Getter:     GetterFunc[protoreflect.EnumNumber](func() protoreflect.EnumNumber { return parent.Get().Get(field).Enum() }),
			Setter:     SetterFunc[protoreflect.EnumNumber](func(v protoreflect.EnumNumber) { parent.Get().Set(field, protoreflect.ValueOfEnum(v)) }),
			Parser:     ParserFunc[protoreflect.EnumNumber](func(s string) (protoreflect.EnumNumber, error) { return util.EnumNumberFromString(field, s) }),
			Stringer:   StringerFunc[protoreflect.EnumNumber](func(v protoreflect.EnumNumber) string { return util.EnumNumberToString(field, v) }),
			parent:     parent,
			descriptor: field,
		}, true
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return &Scalar[int32]{
			Getter:     GetterFunc[int32](func() int32 { return int32(parent.Get().Get(field).Int()) }),
			Setter:     SetterFunc[int32](func(v int32) { parent.Get().Set(field, protoreflect.ValueOfInt32(v)) }),
			Parser:     ParserFunc[int32](util.ParseInt32),
			Stringer:   StringerFunc[int32](func(i int32) string { return strconv.FormatInt(int64(i), 10) }),
			parent:     parent,
			descriptor: field,
		}, true
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return &Scalar[int64]{
			Getter:     GetterFunc[int64](func() int64 { return parent.Get().Get(field).Int() }),
			Setter:     SetterFunc[int64](func(v int64) { parent.Get().Set(field, protoreflect.ValueOfInt64(v)) }),
			Parser:     ParserFunc[int64](util.ParseInt64),
			Stringer:   StringerFunc[int64](func(i int64) string { return strconv.FormatInt(i, 10) }),
			parent:     parent,
			descriptor: field,
		}, true
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return &Scalar[uint32]{
			Getter:     GetterFunc[uint32](func() uint32 { return uint32(parent.Get().Get(field).Uint()) }),
			Setter:     SetterFunc[uint32](func(v uint32) { parent.Get().Set(field, protoreflect.ValueOfUint32(v)) }),
			Parser:     ParserFunc[uint32](util.ParseUInt32),
			Stringer:   StringerFunc[uint32](func(i uint32) string { return strconv.FormatUint(uint64(i), 10) }),
			parent:     parent,
			descriptor: field,
		}, true
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return &Scalar[uint64]{
			Getter:     GetterFunc[uint64](func() uint64 { return uint64(parent.Get().Get(field).Uint()) }),
			Setter:     SetterFunc[uint64](func(v uint64) { parent.Get().Set(field, protoreflect.ValueOfUint64(v)) }),
			Parser:     ParserFunc[uint64](util.ParseUInt64),
			Stringer:   StringerFunc[uint64](func(i uint64) string { return strconv.FormatUint(i, 10) }),
			parent:     parent,
			descriptor: field,
		}, true
	case protoreflect.FloatKind:
		return &Scalar[float32]{
			Getter:     GetterFunc[float32](func() float32 { return float32(parent.Get().Get(field).Float()) }),
			Setter:     SetterFunc[float32](func(v float32) { parent.Get().Set(field, protoreflect.ValueOfFloat32(v)) }),
			Parser:     ParserFunc[float32](util.ParseFloat32),
			Stringer:   StringerFunc[float32](func(i float32) string { return strconv.FormatInt(int64(i), 10) }),
			parent:     parent,
			descriptor: field,
		}, true
	case protoreflect.DoubleKind:
		return &Scalar[float64]{
			Getter:     GetterFunc[float64](func() float64 { return parent.Get().Get(field).Float() }),
			Setter:     SetterFunc[float64](func(v float64) { parent.Get().Set(field, protoreflect.ValueOfFloat64(v)) }),
			Parser:     ParserFunc[float64](util.ParseFloat64),
			Stringer:   StringerFunc[float64](func(i float64) string { return fmt.Sprintf("%f", i) }),
			parent:     parent,
			descriptor: field,
		}, true
	case protoreflect.StringKind:
		return &Scalar[string]{
			Getter:     GetterFunc[string](func() string { return parent.Get().Get(field).String() }),
			Setter:     SetterFunc[string](func(s string) { parent.Get().Set(field, protoreflect.ValueOfString(s)) }),
			Parser:     ParserFunc[string](func(s string) (string, error) { return s, nil }),
			Stringer:   StringerFunc[string](func(s string) string { return s }),
			parent:     parent,
			descriptor: field,
		}, true
	}
	return nil, false
}

func (s *ScalarField) isFieldType() {}

func (s *ScalarField) Element() *Element {
	return s.element
}

func (s *Scalar[T]) Parent() *Message {
	return s.parent
}

func (s *Scalar[T]) Descriptor() protoreflect.FieldDescriptor {
	return s.descriptor
}

func (s *Scalar[T]) Get() (v T) {
	if s.Getter != nil {
		return s.Getter.Get()
	}
	return
}

func (s *Scalar[T]) Set(v T) {
	if s.Setter != nil {
		s.Setter.Set(v)
	}
	s.parent.Set(s.parent.Get())
}

func (s *Scalar[T]) GetAny() any {
	return s.Get()
}

func (s *Scalar[T]) SetAny(v any) error {
	str := util.StringFormatAny(v)
	if len(str) > 0 {
		value, err := s.Parse(str)
		if err != nil {
			return err
		}
		s.Set(value)
	}
	return nil
}

func (m *Scalar[T]) Parse(str string) (T, error) {
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

func (s *Scalar[T]) StringOf(v T) string {
	if s.Stringer != nil {
		return s.Stringer.StringOf(v)
	}
	return util.StringFormatAny(v)
}

func (s *Scalar[T]) String() string {
	return s.StringOf(s.Get())
}
