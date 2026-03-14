package common

import (
	"reflect"

	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

type celIterator struct {
	refVal   ref.Val
	iterator traits.Iterator
	adapter  *CELTypes
}

func (i *celIterator) ConvertToNative(t reflect.Type) (any, error) {
	return i.refVal.ConvertToNative(t)
}

func (i *celIterator) ConvertToType(typeVal ref.Type) ref.Val {
	return i.refVal.ConvertToType(typeVal)
}

func (i *celIterator) Equal(other ref.Val) ref.Val {
	return i.refVal.Equal(other)
}

func (i *celIterator) Type() ref.Type {
	return i.refVal.Type()
}

func (i *celIterator) Value() any {
	return i.refVal.Value()
}

func (i *celIterator) HasNext() ref.Val {
	return i.iterator.HasNext()
}

func (i *celIterator) Next() ref.Val {
	return i.adapter.wrapRefVal(i.iterator.Next())
}
