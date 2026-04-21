package entadapter

import (
	"reflect"

	entfield "entgo.io/ent/schema/field"
)

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

// fixDescriptorPkgPath sets Info.PkgPath on a descriptor when the GoType is
// a composite type (slice/map) whose element lives in an external package.
// Ent's Bytes.GoType uses indirect(t).PkgPath() which returns "" for slices
// and maps, causing the import template to skip required imports.
func fixDescriptorPkgPath[T any](desc *entfield.Descriptor) {
	if desc.Info == nil || desc.Info.PkgPath != "" {
		return
	}
	if pkg := deepPkgPath(reflect.TypeFor[T]()); pkg != "" {
		desc.Info.PkgPath = pkg
	}
}
