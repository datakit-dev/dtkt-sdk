package sqladapter

import (
	"math"
	"reflect"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	sqlv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/sql/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostore/v1beta1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ColumnFromProto(desc protoreflect.FieldDescriptor, opts ...v1beta1.FieldOption) (*sqlv1beta1.Column, error) {
	field := v1beta1.NewField(desc, opts...)
	colType := typeFromRefType(field.Type(), field.Rules())
	builder := sqlv1beta1.Column_builder{
		Name: new(field.Proto().GetName()),
		Type: colType,
	}

	if field.Proto().GetNullable() {
		builder.Nullable = new(true)
	} else if desc.ContainingOneof() != nil && !desc.ContainingOneof().IsSynthetic() {
		builder.Nullable = new(true)
	} else if desc.IsList() || desc.IsMap() || desc.Message() != nil {
		builder.Nullable = new(true)
	}

	if field.Rules().GetRequired() {
		builder.Nullable = nil
	}

	return builder.Build(), nil
}

func typeFromRefType(refType reflect.Type, rules *validate.FieldRules) *sqlv1beta1.Type {
	isPointer := refType.Kind() == reflect.Pointer
	if isPointer {
		refType = refType.Elem()
	}

	typ := sqlv1beta1.Type_builder{}
	switch refType.Kind() {
	case reflect.Bool:
		typ.BoolType = sqlv1beta1.BoolType_builder{}.Build()
	case reflect.String:
		b := sqlv1beta1.StringType_builder{}
		if rules != nil && rules.HasString() && rules.GetString().HasMaxLen() && rules.GetString().GetMaxLen() <= math.MaxUint32 {
			b.Size = new(uint32(rules.GetString().GetMaxLen()))
		}
		typ.StringType = b.Build()
	case reflect.Int32, reflect.Int64:
		typ.IntegerType = sqlv1beta1.IntegerType_builder{}.Build()
	case reflect.Uint32, reflect.Uint64:
		typ.IntegerType = sqlv1beta1.IntegerType_builder{
			Unsigned: new(true),
		}.Build()
	case reflect.Float32, reflect.Float64:
		switch refType.Bits() {
		case 32:
			typ.FloatType = sqlv1beta1.FloatType_builder{
				Precision: new(uint32(7)),
			}.Build()
		case 64:
			typ.FloatType = sqlv1beta1.FloatType_builder{
				Precision: new(uint32(15)),
			}.Build()
		}
	case reflect.Map:
		typ.MapType = sqlv1beta1.MapType_builder{
			KeyType: typeFromRefType(refType.Key(), nil),
			ValType: typeFromRefType(refType.Elem(), nil),
		}.Build()
	case reflect.Slice:
		if refType.Elem().Kind() == reflect.Uint8 {
			b := sqlv1beta1.BinaryType_builder{}
			if rules != nil && rules.HasBytes() && rules.GetBytes().HasMaxLen() && rules.GetBytes().GetMaxLen() <= math.MaxUint32 {
				b.Size = new(uint32(rules.GetBytes().GetMaxLen()))
			}
			typ.BinaryType = b.Build()
		} else {
			typ.ListType = sqlv1beta1.ListType_builder{
				ItemType: typeFromRefType(refType.Elem(), nil),
			}.Build()
		}
	case reflect.Struct:
		if msg, ok := reflect.New(refType).Interface().(proto.Message); ok {
			switch msg := msg.(type) {
			case *timestamppb.Timestamp:
				typ.TimeType = sqlv1beta1.TimeType_builder{
					Precision: new(uint32(9)),
					// Scale:     new(uint32(9)),
				}.Build()
			default:
				typ.MessageType = sqlv1beta1.MessageType_builder{
					Name: new(string(msg.ProtoReflect().Descriptor().FullName())),
				}.Build()
			}
		}
	}

	return typ.Build()
}
