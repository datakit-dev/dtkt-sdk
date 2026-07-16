package v1beta1

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func GetFieldElement(desc protoreflect.FieldDescriptor) (*protoformv1beta1.FieldElement, bool) {
	if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
		if proto.HasExtension(opts, protoformv1beta1.E_Field) {
			if elem, ok := proto.GetExtension(opts, protoformv1beta1.E_Field).(*protoformv1beta1.FieldElement); ok && elem != nil {
				return elem, ok
			}
		}
	}
	return nil, false
}

func GetFieldRules(desc protoreflect.FieldDescriptor) (*validate.FieldRules, bool) {
	if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
		if proto.HasExtension(opts, validate.E_Field) {
			if rules, ok := proto.GetExtension(opts, validate.E_Field).(*validate.FieldRules); ok && rules != nil {
				return rules, ok
			}
		}
	}
	return nil, false
}
