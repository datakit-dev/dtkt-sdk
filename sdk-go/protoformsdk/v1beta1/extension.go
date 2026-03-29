package v1beta1

import (
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	"github.com/jhump/protoreflect/v2/sourceloc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type ProtoSourceInfoOptions struct {
	Multiline bool
}

func GetProtoDescription(d protoreflect.Descriptor) string {
	return ProtoSourceInfoOptions{Multiline: true}.GetDescription(d)
}

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

func (o ProtoSourceInfoOptions) GetSourceLocation(d protoreflect.Descriptor) (_ protoreflect.SourceLocation, ok bool) {
	sp := sourceloc.PathFor(d)
	if sp == nil {
		return
	}
	return d.ParentFile().SourceLocations().ByPath(sp), true
}

func (o ProtoSourceInfoOptions) GetDescription(d protoreflect.Descriptor) (_ string) {
	sl, ok := o.GetSourceLocation(d)
	if !ok {
		return
	}

	comments := sl.LeadingDetachedComments
	if sl.LeadingComments != "" {
		comments = append(comments, strings.TrimSpace(sl.LeadingComments))
	}

	if sl.TrailingComments != "" {
		comments = append(comments, strings.TrimSpace(sl.TrailingComments))
	}

	if o.Multiline {
		return strings.Join(comments, "\n")
	}

	return strings.Join(comments, " ")
}
