package entadapter

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	entfield "entgo.io/ent/schema/field"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/protostore"
	"google.golang.org/protobuf/encoding/protojson"
)

func ListValueScanner[S []T, T any](id ...string) entfield.TypeValueScanner[S] {
	return entfield.ValueScannerFunc[S, *sql.Null[[]byte]]{
		V: func(list S) (driver.Value, error) {
			b, err := encoding.ToJSONV2(list, encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
				Resolver: protostore.GetResolver(),
			}))
			if err != nil {
				return nil, err
			}

			return json.RawMessage(b), nil
		},
		S: func(ns *sql.Null[[]byte]) (list S, err error) {
			if ns.Valid {
				err = encoding.FromJSONV2(ns.V, &list, encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: protostore.GetResolver(),
				}))
			}
			return
		},
	}
}

func MapValueScanner[M map[K]V, K protostore.MapKeyType, V any](id ...string) entfield.TypeValueScanner[M] {
	return entfield.ValueScannerFunc[M, *sql.Null[[]byte]]{
		V: func(m M) (driver.Value, error) {
			b, err := encoding.ToJSONV2(m, encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
				Resolver: protostore.GetResolver(),
			}))
			if err != nil {
				return nil, err
			}

			return json.RawMessage(b), nil
		},
		S: func(ns *sql.Null[[]byte]) (m M, err error) {
			if ns.Valid {
				err = encoding.FromJSONV2(ns.V, &m, encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: protostore.GetResolver(),
				}))
			}
			return
		},
	}
}
