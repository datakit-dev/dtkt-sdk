package entadapter

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"entgo.io/ent"
	entfield "entgo.io/ent/schema/field"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostore"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func MapField[K protostore.MapKeyType, V any, M map[K]V](opts ...FieldOption) *Field[M] {
	var (
		keyType K
		valType V
		mapType M
		name    string
	)

	switch v := any(valType).(type) {
	case proto.Message:
		name = fmt.Sprintf("map_%T_%s", keyType, util.ToSnakeCase(v.ProtoReflect().Descriptor().Name()))
	case protoreflect.Enum:
		name = fmt.Sprintf("map_%T_%s", keyType, util.ToSnakeCase(v.Descriptor().Name()))
	default:
		name = fmt.Sprintf("map_%T_%T", keyType, v)
	}

	field := entfield.Bytes(name).GoType(mapType)

	return NewFieldOf[M](
		setDescriptorPkgPathFor[V](
			field.ValueScanner(MapScanner[K, V](field)),
		), opts...,
	)
}

func MapScanner[K protostore.MapKeyType, V any, M map[K]V](field ent.Field) entfield.TypeValueScanner[M] {
	return entfield.ValueScannerFunc[M, *sql.Null[[]byte]]{
		V: func(m M) (driver.Value, error) {
			b, err := encoding.ToJSON(m, encoding.WithProtoEncodeOptions(protojson.MarshalOptions{
				Resolver: protostore.Resolver,
			}))
			if err != nil {
				return nil, fmt.Errorf("field: %s: %w", field.Descriptor().Name, err)
			}

			return json.RawMessage(b), nil
		},
		S: func(ns *sql.Null[[]byte]) (m M, err error) {
			if ns.Valid {
				err = encoding.FromJSON(ns.V, &m, encoding.WithProtoDecodeOptions(protojson.UnmarshalOptions{
					Resolver: protostore.Resolver,
				}))
				if err != nil {
					return nil, fmt.Errorf("field: %s: %w", field.Descriptor().Name, err)
				}
			}
			return
		},
	}
}
