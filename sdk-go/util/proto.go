package util

import (
	"strings"

	"github.com/jhump/protoreflect/v2/sourceloc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoSourceInfoOptions struct {
	Multiline bool
}

func GetProtoDescription(d protoreflect.Descriptor) string {
	return ProtoSourceInfoOptions{Multiline: true}.GetDescription(d)
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
