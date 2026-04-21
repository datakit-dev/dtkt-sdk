package entadapter

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	entfield "entgo.io/ent/schema/field"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/protostore"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func MapField[M map[K]V, K protostore.MapKeyType, V any](opts ...FieldOption) *fieldType[M] {
	var (
		map_  M
		key   K
		value V
		name  string
	)

	switch v := any(value).(type) {
	case proto.Message:
		name = fmt.Sprintf("map_%T_%s", key, util.ToSnakeCase(v.ProtoReflect().Descriptor().Name()))
	case protoreflect.Enum:
		name = fmt.Sprintf("map_%T_%s", key, util.ToSnakeCase(v.Descriptor().Name()))
	default:
		name = fmt.Sprintf("map_%T_%T", key, v)
	}

	field := &fieldType[M]{
		field: v1beta1.NewField(nil),
	}

	field.desc = entfield.
		Bytes(name).
		GoType(map_).
		ValueScanner(MapScanner(field)).
		Descriptor()

	// Fix PkgPath for map value types: Bytes.GoType uses indirect(t).PkgPath()
	// which returns "" for maps, causing the import template to skip the import.
	fixDescriptorPkgPath[V](field.desc)

	field.applyOptions(opts...)

	return field
}

func MapScanner[M map[K]V, K protostore.MapKeyType, V any](field *fieldType[M]) entfield.TypeValueScanner[M] {
	return entfield.ValueScannerFunc[M, *sql.Null[[]byte]]{
		V: func(m M) (driver.Value, error) {
			b, err := encoding.ToJSONV2(m, encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
				Resolver: protostore.GetResolver(),
			}))
			if err != nil {
				return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
			}

			return json.RawMessage(b), nil
		},
		S: func(ns *sql.Null[[]byte]) (m M, err error) {
			if ns.Valid {
				err = encoding.FromJSONV2(ns.V, &m, encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: protostore.GetResolver(),
				}))
				if err != nil {
					return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
				}
			}
			return
		},
	}
}
