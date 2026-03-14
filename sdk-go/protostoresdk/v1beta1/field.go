package v1beta1

import (
	"reflect"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/protovalidate"
	protostorev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protostore/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/protostore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type (
	Field struct {
		proto    *protostorev1beta1.Field
		rules    *validate.FieldRules
		desc     protoreflect.FieldDescriptor
		rType    reflect.Type
		validate []protovalidate.ValidationOption
	}
	FieldOption func(*Field)
)

func NewField(desc protoreflect.FieldDescriptor, opts ...FieldOption) *Field {
	var (
		field *protostorev1beta1.Field
		rules *validate.FieldRules
		rType reflect.Type
	)

	if desc != nil {
		rType = protostore.ReflectFieldType(desc)

		if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
			if proto.HasExtension(opts, protostorev1beta1.E_Field) {
				if f, ok := proto.GetExtension(opts, protostorev1beta1.E_Field).(*protostorev1beta1.Field); ok && f != nil {
					field = f
				}
			}
		}

		if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
			if proto.HasExtension(opts, validate.E_Field) {
				if r, ok := proto.GetExtension(opts, validate.E_Field).(*validate.FieldRules); ok && r != nil {
					rules = r
				}
			}
		}
	}

	if field == nil {
		field = &protostorev1beta1.Field{}
	}

	if rules == nil {
		rules = &validate.FieldRules{}
	}

	if field.GetName() == "" && desc != nil {
		field.Name = new(string(desc.Name()))
	}

	return (&Field{
		proto: field,
		rules: rules,
		desc:  desc,
		rType: rType,
	}).applyOptions(opts...)
}

func (f *Field) Proto() *protostorev1beta1.Field {
	return f.proto
}

func (f *Field) Descriptor() protoreflect.FieldDescriptor {
	return f.desc
}

func (f *Field) ReflectType() reflect.Type {
	return f.rType
}

func (f *Field) applyOptions(opts ...FieldOption) *Field {
	for _, o := range opts {
		if o != nil {
			o(f)
		}
	}
	return f
}
