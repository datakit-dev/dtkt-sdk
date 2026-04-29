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
		proto *protostorev1beta1.Field
		desc  protoreflect.FieldDescriptor
		typ   reflect.Type
		rules *validate.FieldRules
		opts  []protovalidate.ValidationOption
	}
	FieldOption func(*Field)
)

func NewField(desc protoreflect.FieldDescriptor, opts ...FieldOption) *Field {
	field := &Field{
		desc: desc,
	}

	if desc != nil {
		field.typ = protostore.ReflectFieldType(desc)

		if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
			if proto.HasExtension(opts, protostorev1beta1.E_Field) {
				if f, ok := proto.GetExtension(opts, protostorev1beta1.E_Field).(*protostorev1beta1.Field); ok && f != nil {
					field.proto = f
				}
			}
		}

		if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
			if proto.HasExtension(opts, validate.E_Field) {
				if r, ok := proto.GetExtension(opts, validate.E_Field).(*validate.FieldRules); ok && r != nil {
					field.rules = r
				}
			}
		}
	}

	if field.proto == nil {
		field.proto = &protostorev1beta1.Field{}
	}

	if field.rules == nil {
		field.rules = &validate.FieldRules{}
	}

	if field.proto.GetName() == "" && desc != nil {
		field.proto.Name = new(string(desc.Name()))
	}

	return field.applyOptions(opts...)
}

func (f *Field) Proto() *protostorev1beta1.Field {
	return f.proto
}

func (f *Field) Descriptor() protoreflect.FieldDescriptor {
	return f.desc
}

func (f *Field) Type() reflect.Type {
	return f.typ
}

func (f *Field) applyOptions(opts ...FieldOption) *Field {
	for _, o := range opts {
		if o != nil {
			o(f)
		}
	}
	return f
}
