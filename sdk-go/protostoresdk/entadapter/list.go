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

func ListField[S []V, V any](opts ...FieldOption) *fieldType[S] {
	var (
		list  S
		value V
		name  string
	)

	switch v := any(value).(type) {
	case proto.Message:
		name = fmt.Sprintf("list_%s", util.ToSnakeCase(v.ProtoReflect().Descriptor().Name()))
	case protoreflect.Enum:
		name = fmt.Sprintf("list_%s", util.ToSnakeCase(v.Descriptor().Name()))
	default:
		name = fmt.Sprintf("list_%T", v)
	}

	field := &fieldType[S]{
		field: v1beta1.NewField(nil),
	}

	field.desc = entfield.
		Bytes(name).
		GoType(list).
		ValueScanner(ListScanner(field)).
		Descriptor()

	// Fix PkgPath for slice element types: Bytes.GoType uses indirect(t).PkgPath()
	// which returns "" for slices, causing the import template to skip the import.
	fixDescriptorPkgPath[V](field.desc)

	field.applyOptions(opts...)

	return field
}

func ListScanner[S []V, V any](field *fieldType[S]) entfield.TypeValueScanner[S] {
	return entfield.ValueScannerFunc[S, *sql.Null[[]byte]]{
		V: func(list S) (driver.Value, error) {
			b, err := encoding.ToJSONV2(list, encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
				Resolver: protostore.GetResolver(),
			}))
			if err != nil {
				return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
			}

			return json.RawMessage(b), nil
		},
		S: func(ns *sql.Null[[]byte]) (list S, err error) {
			if ns.Valid {
				err = encoding.FromJSONV2(ns.V, &list, encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
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
