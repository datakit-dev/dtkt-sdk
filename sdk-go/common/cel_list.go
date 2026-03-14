package common

import (
	"reflect"

	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

type celList struct {
	refVal  ref.Val
	lister  traits.Lister
	adapter *CELTypes
}

func (l *celList) ConvertToNative(t reflect.Type) (any, error) {
	return l.refVal.ConvertToNative(t)
}

func (l *celList) ConvertToType(typeVal ref.Type) ref.Val {
	return l.refVal.ConvertToType(typeVal)
}

func (l *celList) Equal(other ref.Val) ref.Val {
	return l.refVal.Equal(other)
}

func (l *celList) Type() ref.Type {
	return l.refVal.Type()
}

func (l *celList) Value() any {
	return l.refVal.Value()
}

func (l *celList) Add(other ref.Val) ref.Val {
	return l.adapter.wrapRefVal(l.lister.Add(other))
}

func (l *celList) Contains(elem ref.Val) ref.Val {
	return l.lister.Contains(elem)
}

func (l *celList) Get(index ref.Val) ref.Val {
	return l.adapter.wrapRefVal(l.lister.Get(index))
}

func (l *celList) Iterator() traits.Iterator {
	return l.adapter.wrapIterator(l.lister.Iterator())
}

func (l *celList) Size() ref.Val {
	return l.lister.Size()
}
