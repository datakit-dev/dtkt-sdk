package entadapter

import (
	"database/sql/driver"
	"fmt"
	"slices"

	"entgo.io/ent/dialect/sql"
	entfield "entgo.io/ent/schema/field"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/protostore"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

func EnumField[E protostore.EnumType](opts ...FieldOption) *fieldType[E] {
	f := &fieldType[E]{
		field: v1beta1.NewField(nil),
	}

	var enum E
	f.desc = entfield.String(util.ToSnakeCase(enum.Descriptor().Name())).
		GoType(enum).
		ValueScanner(EnumValueScanner[E]()).
		Descriptor()

	f.applyOptions(opts...)

	return f
}

func EnumValues[E protostore.EnumType]() (values []string) {
	var enum E
	for idx := range enum.Descriptor().Values().Len() {
		if idx > 0 {
			values = append(values, string(enum.Descriptor().Values().Get(idx).Name()))
		}
	}
	return
}

func EnumValueScanner[E protostore.EnumType](id ...string) entfield.TypeValueScanner[E] {
	values := EnumValues[E]()
	return entfield.ValueScannerFunc[E, *sql.NullString]{
		V: func(value E) (driver.Value, error) {
			if value <= 0 || int(value) >= len(values) {
				return nil, fmt.Errorf("invalid enum value: %d", value)
			}
			return values[int(value)], nil
		},
		S: func(ns *sql.NullString) (_ E, err error) {
			if ns.Valid {
				if idx := slices.Index(values, ns.String); idx > -1 && idx < len(values) {
					return E(idx), nil
				}
			}
			return
		},
	}
}
