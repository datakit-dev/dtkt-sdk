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

func ListField[V any, S []V](opts ...FieldOption) *Field[S] {
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

	field := entfield.Bytes(name).GoType(list)

	return NewFieldOf[S](
		setDescriptorPkgPathFor[V](
			field.ValueScanner(ListScanner[V](field)),
		), opts...,
	)
}

func ListScanner[V any, S []V](field ent.Field) entfield.TypeValueScanner[S] {
	return entfield.ValueScannerFunc[S, *sql.Null[[]byte]]{
		V: func(list S) (driver.Value, error) {
			b, err := encoding.ToJSON(list, encoding.WithProtoEncodeOptions(protojson.MarshalOptions{
				Resolver: protostore.Resolver,
			}))
			if err != nil {
				return nil, fmt.Errorf("list field: %s: %w", field.Descriptor().Name, err)
			}

			return json.RawMessage(b), nil
		},
		S: func(ns *sql.Null[[]byte]) (list S, err error) {
			if ns.Valid {
				err = encoding.FromJSON(ns.V, &list, encoding.WithProtoDecodeOptions(protojson.UnmarshalOptions{
					Resolver: protostore.Resolver,
				}))
				if err != nil {
					return nil, fmt.Errorf("list field: %s: %w", field.Descriptor().Name, err)
				}
			}
			return
		},
	}
}
