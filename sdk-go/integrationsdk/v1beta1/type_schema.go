package v1beta1

import (
	"fmt"
	"net/url"
	"path"
	"reflect"
	"strings"
	"time"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	JSONSchemaFileExt  = ".jsonschema.json"
	ProtoSchemaFileExt = ".proto"
)

type (
	// TypeSchema is a generic wrapper for any serializable Go type (T) providing a
	// URI, resolved JSON Schema, and a fully qualified proto type name of either:
	// 1) Protobuf message or enum type name if T is a native protobuf type;
	// 2) Protobuf message wrapper type capable of encoding/decoding T.
	TypeSchema[T any] struct {
		typeSchema       *sharedv1beta1.TypeSchema
		typeProto        proto.Message
		jsonSchema       *common.JSONSchema[T]
		isProto, isEmpty bool
	}
	// A generic empty message that you can re-use to avoid defining duplicated
	// empty types in your APIs.
	Empty struct{}
)

func NewTypeSchemaFor[T any](registry *TypeRegistry, typeName string) (_ *TypeSchema[T], err error) {
	var (
		value      T
		jsonSchema *common.JSONSchema[T]
		jsonOpts   = []common.JSONSchemaOpt{
			common.WithSchemaCompiler(registry.compiler),
		}
		isEmpty bool
	)

	typeProto, isProto := any(value).(proto.Message)
	if isProto {
		rawSchema, err := NewProtoSchema(registry, typeProto.ProtoReflect().Descriptor())
		if err != nil {
			return nil, err
		}

		jsonOpts = append(jsonOpts,
			common.WithRawSchema(rawSchema),
		)

		jsonSchema, err = common.JSONSchemaFor[T](jsonOpts...)
		if err != nil {
			return nil, err
		}

		isEmpty = typeProto.ProtoReflect().Descriptor().Fields().Len() == 0
	} else {
		if registry.protoGen {
			typeName = protoschema.SanitizeMessageName(typeName)
			jsonOpts = append(jsonOpts,
				common.WithSchemaID(registry.baseUri.JoinPath(
					string(protoreflect.FullName(registry.protoPkg).Append(protoreflect.Name(typeName)))+JSONSchemaFileExt,
				).String()),
			)
		} else {
			jsonOpts = append(jsonOpts,
				common.WithSchemaID(registry.baseUri.JoinPath(typeName+JSONSchemaFileExt).String()),
			)
		}

		jsonSchema, err = common.JSONSchemaFor[T](jsonOpts...)
		if err != nil {
			return nil, err
		}

		if registry.protoGen {
			file, err := protoschema.NewParser(protoschema.ParserOptions{
				PackageName: registry.protoPkg,
				MessageName: typeName,
			}).Parse(jsonSchema.Bytes())
			if err != nil {
				return nil, err
			}

			// Build the file descriptor
			fd, err := protodesc.NewFile(file, protoregistry.GlobalFiles)
			if err != nil {
				return nil, fmt.Errorf("failed to create file descriptor: %w", err)
			}

			// Return the first (main) message descriptor
			if fd.Messages().Len() == 0 {
				return nil, fmt.Errorf("no message descriptor found")
			}

			err = protoregistry.GlobalFiles.RegisterFile(fd)
			if err != nil {
				return nil, fmt.Errorf("failed to register file descriptor: %w", err)
			}

			desc := fd.Messages().Get(0)
			typeProto = dynamicpb.NewMessageType(desc).New().Interface()
			typeName = string(typeProto.ProtoReflect().Descriptor().FullName())
			isEmpty = desc.Fields().Len() == 0
		} else {
			switch any(value).(type) {
			case time.Duration:
				typeProto = &durationpb.Duration{}
			case time.Time:
				typeProto = &timestamppb.Timestamp{}
			case Empty, struct{}:
				typeProto = &emptypb.Empty{}
				isEmpty = true
			default:
				isEmpty = jsonSchema.JSONSchema().Properties == nil || jsonSchema.JSONSchema().Properties.Len() == 0

				refType := reflect.TypeOf(value)
				if refType != nil {
					if refType.Kind() == reflect.Pointer {
						refType = refType.Elem()
					}

					switch refType.Kind() {
					case reflect.Struct:
						if refType.NumField() == 0 {
							typeProto = &emptypb.Empty{}
							isEmpty = true
						} else {
							typeProto = &structpb.Struct{}
						}
					default:
						typeProto, err = common.WrapProto(value)
						if err != nil {
							return nil, err
						}
					}
				} else {
					typeProto, err = common.WrapProto(value)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	typeSchema := &sharedv1beta1.TypeSchema{
		Uri:        registry.baseUri.JoinPath(typeName).String(),
		JsonSchema: jsonSchema.ToProto(),
		ProtoName:  string(typeProto.ProtoReflect().Descriptor().FullName()),
		ModTime:    timestamppb.Now(),
	}

	err = registry.syncer.StoreType(typeSchema)
	if err != nil {
		return nil, err
	}

	return &TypeSchema[T]{
		typeSchema: typeSchema,
		typeProto:  typeProto,
		jsonSchema: jsonSchema,
		isProto:    isProto,
		isEmpty:    isEmpty,
	}, nil
}

func TypePathFromUri(uriStr string) (string, error) {
	uri, err := url.Parse(uriStr)
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(strings.Trim(uri.Path, "/"), JSONSchemaFileExt), nil
}

func TypeNameFromUri(uriStr string) (string, error) {
	typPath, err := TypePathFromUri(uriStr)
	if err != nil {
		return "", err
	}

	return path.Base(typPath), nil
}

func (s *TypeSchema[T]) JSONSchema() *common.JSONSchema[T] {
	return s.jsonSchema
}

func (s *TypeSchema[T]) ValidateAny(value any) (v T, err error) {
	v, ok := value.(T)
	if ok {
		err = s.Validate(v)
		if err != nil {
			return
		}
	}

	b, err := encoding.ToJSONV2(value)
	if err != nil {
		return
	}

	if s.isProto {
		v = s.typeProto.ProtoReflect().New().Interface().(T)
		err = encoding.FromJSONV2(b, v)
	} else {
		err = encoding.FromJSONV2(b, &v)
	}
	if err != nil {
		return
	}
	err = s.Validate(v)
	return
}

func (s *TypeSchema[T]) Validate(value T) error {
	if s.isProto {
		if msg, ok := any(value).(proto.Message); ok {
			return protovalidate.Validate(msg)
		}
	}
	return s.jsonSchema.Validate(value)
}

func (s *TypeSchema[T]) IsEmpty() bool {
	return s.isEmpty
}

func (s *TypeSchema[T]) ToProto() *sharedv1beta1.TypeSchema {
	return s.typeSchema
}
