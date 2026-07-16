package util

import (
	"fmt"
	"strings"

	"github.com/jhump/protoreflect/v2/sourceloc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type ProtoSourceInfoOptions struct {
	Multiline bool
}

func GetProtoDescription(d protoreflect.Descriptor) string {
	return ProtoSourceInfoOptions{Multiline: true}.GetDescription(d)
}

func ProtoDiff(msgA proto.Message, msgB proto.Message) (fieldMask *fieldmaskpb.FieldMask) {
	fieldMask = new(fieldmaskpb.FieldMask)
	if protoreflect.ValueOfMessage(msgA.ProtoReflect()).Equal(protoreflect.ValueOfMessage(msgB.ProtoReflect())) {
		return
	}

	diffFunc := func(other proto.Message, desc protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if !value.Equal(other.ProtoReflect().Get(desc)) {
			if other.ProtoReflect().Has(desc) && desc.Message() != nil && (!desc.IsList() && !desc.IsMap()) {
				for _, path := range ProtoDiff(value.Message().Interface(), other.ProtoReflect().Get(desc).Message().Interface()).GetPaths() {
					fieldMask.Paths = append(fieldMask.Paths, fmt.Sprintf("%s.%s", desc.Name(), path))
				}
			} else {
				fieldMask.Paths = append(fieldMask.Paths, string(desc.Name()))
			}
		}
		return true
	}

	msgA.ProtoReflect().Range(func(desc protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		return diffFunc(msgB, desc, value)
	})

	msgB.ProtoReflect().Range(func(desc protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		return diffFunc(msgA, desc, value)
	})

	fieldMask.Normalize()

	return
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
