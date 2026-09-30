package entadapter

import (
	"reflect"

	"entgo.io/ent"
)

// setDescriptorPkgPathFor sets Info.PkgPath on a descriptor when the GoType is
// a composite type (slice/map) whose element lives in an external package.
// Ent's Bytes.GoType uses indirect(t).PkgPath() which returns "" for slices
// and maps, causing the import template to skip required imports.
func setDescriptorPkgPathFor[T any](field ent.Field) ent.Field {
	if field.Descriptor().Info != nil && field.Descriptor().Info.PkgPath == "" {
		if pkg := deepPkgPath(reflect.TypeFor[T]()); pkg != "" {
			field.Descriptor().Info.PkgPath = pkg
		}
	}
	return field
}

// deepPkgPath recursively finds the package path of a type by traversing
// through pointers, slices, arrays, and maps to find the element's package.
// This mirrors the unexported pkgPath function in entgo.io/ent/schema/field
// that the JSON field builder uses but Bytes.GoType does not.
func deepPkgPath(t reflect.Type) string {
	if t == nil {
		return ""
	}
	if pkg := t.PkgPath(); pkg != "" {
		return pkg
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Pointer, reflect.Map:
		return deepPkgPath(t.Elem())
	}
	return ""
}
