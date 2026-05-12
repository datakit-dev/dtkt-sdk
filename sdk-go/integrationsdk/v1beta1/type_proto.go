package v1beta1

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewTypeSchemaForProto[T proto.Message](registry *TypeRegistry, typeProto T) (*TypeSchema[T], error) {
	schemaBytes, err := generateProtoSchemaBytes(registry, typeProto.ProtoReflect().Descriptor())
	if err != nil {
		return nil, err
	}

	name := string(typeProto.ProtoReflect().Descriptor().FullName())
	opts := []common.JSONSchemaOpt{
		common.WithSchemaCompiler(registry.compiler),
		common.WithJSONSchemaBytes(schemaBytes),
		common.WithSchemaID(registry.baseUri.JoinPath(name + JSONSchemaFileExt).String()),
	}

	jsonSchema, err := common.JSONSchemaFor[T](opts...)
	if err != nil {
		return nil, err
	}

	typeSchema := &sharedv1beta1.TypeSchema{
		Uri:        registry.baseUri.JoinPath(name).String(),
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
		isProto:    true,
		isEmpty:    typeProto.ProtoReflect().Descriptor().Fields().Len() == 0,
	}, nil
}

func generateProtoSchemaBytes(registry *TypeRegistry, desc protoreflect.MessageDescriptor) (rawSchema []byte, err error) {
	schemaGen := protoschema.NewGenerator(protoschema.WithJSONNames())
	err = schemaGen.Add(desc)
	if err != nil {
		return nil, err
	}

	for typeName, jsonSchema := range schemaGen.Generate() {
		b, err := encoding.ToJSONV2(jsonSchema)
		if err != nil {
			return nil, err
		}

		if typeName == desc.FullName() {
			rawSchema = b
			continue
		}

		structSchema := new(structpb.Struct)
		err = encoding.FromJSONV2(b, structSchema)
		if err != nil {
			return nil, err
		}

		structSchema.Fields["$id"] = structpb.NewStringValue(registry.baseUri.JoinPath(string(typeName) + JSONSchemaFileExt).String())
		err = registry.syncer.StoreType(
			&sharedv1beta1.TypeSchema{
				Uri:        registry.baseUri.JoinPath(string(typeName)).String(),
				JsonSchema: structSchema,
				ProtoName:  string(typeName),
				ModTime:    timestamppb.Now(),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("store type: %s: %w", typeName, err)
		}
	}

	return
}
