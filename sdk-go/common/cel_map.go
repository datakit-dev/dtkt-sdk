package common

import (
	"reflect"

	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

type celMap struct {
	refVal  ref.Val
	mapper  traits.Mapper
	adapter *CELTypes
}

func (m *celMap) ConvertToNative(t reflect.Type) (any, error) {
	return m.refVal.ConvertToNative(t)
}

func (m *celMap) ConvertToType(typeVal ref.Type) ref.Val {
	return m.refVal.ConvertToType(typeVal)
}

func (m *celMap) Equal(other ref.Val) ref.Val {
	return m.refVal.Equal(other)
}

func (m *celMap) Type() ref.Type {
	return m.refVal.Type()
}

func (m *celMap) Value() any {
	return m.refVal.Value()
}

func (m *celMap) Contains(elem ref.Val) ref.Val {
	return m.mapper.Contains(elem)
}

func (m *celMap) Get(index ref.Val) ref.Val {
	return m.adapter.wrapRefVal(m.mapper.Get(index))
}

func (m *celMap) Iterator() traits.Iterator {
	return m.adapter.wrapIterator(m.mapper.Iterator())
}

func (m *celMap) Size() ref.Val {
	return m.mapper.Size()
}

func (m *celMap) Find(key ref.Val) (ref.Val, bool) {
	val, found := m.mapper.Find(key)
	return m.adapter.wrapRefVal(val), found
}

func (m *celMap) Fold(folder traits.Folder) {
	if foldable, ok := m.mapper.(traits.Foldable); ok {
		foldable.Fold(folder)
	}
}
