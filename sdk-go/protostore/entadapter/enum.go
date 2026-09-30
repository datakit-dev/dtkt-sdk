package entadapter

import (
	"database/sql/driver"
	"fmt"
	"slices"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	entfield "entgo.io/ent/schema/field"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostore"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

func EnumField[E protostore.EnumType](opts ...FieldOption) *Field[E] {
	var (
		enum  E
		field = entfield.
			String(util.ToSnakeCase(enum.Descriptor().Name())).
			GoType(enum)
	)

	return NewFieldOf[E](
		field.ValueScanner(EnumValueScanner[E](field)),
		opts...,
	)
}

func EnumValues[E protostore.EnumType]() (values []string) {
	var enum E
	for idx := range enum.Descriptor().Values().Len() {
		values = append(values, string(enum.Descriptor().Values().Get(idx).Name()))
	}
	return
}

func EnumValueScanner[E protostore.EnumType](field ent.Field) entfield.TypeValueScanner[E] {
	values := EnumValues[E]()
	return entfield.ValueScannerFunc[E, *sql.NullString]{
		V: func(value E) (driver.Value, error) {
			if value < 0 || int(value) >= len(values) {
				return nil, fmt.Errorf("field: %s: invalid enum value: %d", field.Descriptor().Name, value)
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
