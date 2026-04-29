package entadapter

import (
	"slices"

	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/protostore"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/v1beta1"
	"google.golang.org/protobuf/proto"
)

type (
	Schema struct {
		mixin.Schema
		schemaType SchemaType
		intercept  FieldInterceptor
		skipFields []string
	}
	SchemaType interface {
		messageType() proto.Message
	}
	SchemaOption                                   func(*Schema)
	FieldInterceptor                               func(*field.Descriptor)
	schemaType[T any, M protostore.MessageType[T]] struct {
		message M
	}
)

func NewSchema[T any, M protostore.MessageType[T]](opts ...SchemaOption) Schema {
	schema := Schema{
		schemaType: &schemaType[T, M]{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&schema)
		}
	}
	return schema
}

func (s Schema) Fields() []ent.Field {
	var fields []ent.Field
	for idx := range s.schemaType.messageType().ProtoReflect().Descriptor().Fields().Len() {
		desc := s.schemaType.messageType().ProtoReflect().Descriptor().Fields().Get(idx)
		field := NewField(v1beta1.NewField(desc))

		if s.intercept != nil {
			s.intercept(field.desc)
		}

		if !field.field.Proto().GetSkip() && !slices.Contains(s.skipFields, field.field.Proto().GetName()) {
			fields = append(fields, field)
		}
	}
	return fields
}

func (s Schema) Annotations() []schema.Annotation {
	return []schema.Annotation{
		Annotation{
			ProtoName: string(s.schemaType.messageType().ProtoReflect().Descriptor().FullName()),
			IsMessage: true,
		},
	}
}

func WithInterceptFields(intercept FieldInterceptor) SchemaOption {
	return func(schema *Schema) {
		schema.intercept = intercept
	}
}

func WithSkipFields(fields ...string) SchemaOption {
	return func(schema *Schema) {
		schema.skipFields = fields
	}
}

func (s *schemaType[T, M]) messageType() proto.Message {
	return s.message
}
