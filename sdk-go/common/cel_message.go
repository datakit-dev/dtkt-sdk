package common

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

type celMessage struct {
	msg         proto.Message
	refVal      ref.Val
	indexer     traits.Indexer
	fieldTester traits.FieldTester
	adapter     *CELTypes
	aliases     map[string]string
}

func (p *celMessage) ConvertToNative(t reflect.Type) (any, error) {
	return p.refVal.ConvertToNative(t)
}

func (p *celMessage) ConvertToType(typeVal ref.Type) ref.Val {
	return p.refVal.ConvertToType(typeVal)
}

func (p *celMessage) Equal(other ref.Val) ref.Val {
	return p.refVal.Equal(other)
}

func (p *celMessage) Type() ref.Type {
	return p.refVal.Type()
}

func (p *celMessage) Value() any {
	return p.refVal.Value()
}

func (p *celMessage) Get(index ref.Val) ref.Val {
	name := p.normalize(index)
	refVal := p.indexer.Get(name)

	field := p.msg.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(fmt.Sprint(name.Value())))
	if field != nil {
		switch {
		case field.Enum() != nil:
			if field.IsList() {
				list, ok := refVal.(traits.Lister)
				if !ok {
					return refVal
				}

				// Wrap each element in the list as an enum.
				length := list.Size().Value().(int64)
				enumList := make([]ref.Val, 0, length)
				for i := range length {
					elem := list.Get(types.Int(i))
					enumList = append(enumList, &celEnum{
						desc:   field.Enum(),
						refVal: elem,
					})
				}

				return types.NewRefValList(p.adapter, enumList)
			}

			return &celEnum{
				desc:   field.Enum(),
				refVal: refVal,
			}
		case field.Message() != nil && !field.IsMap():
			if field.Message().FullName() == "google.protobuf.Any" {
				if field.IsList() {
					list, ok := refVal.(traits.Lister)
					if !ok {
						return refVal
					}

					length := list.Size().Value().(int64)
					msgList := make([]ref.Val, 0, length)
					for i := range length {
						elem := list.Get(types.Int(i))
						anyMsg, ok := elem.Value().(*anypb.Any)
						if !ok {
							msgList = append(msgList, elem)
							continue
						}

						msg, err := anypb.UnmarshalNew(anyMsg, proto.UnmarshalOptions{
							Resolver: p.adapter.resolver,
						})
						if err != nil {
							msgList = append(msgList, types.WrapErr(err))
							continue
						}

						msgList = append(msgList, p.adapter.NativeToValue(msg))
					}

					return types.NewRefValList(p.adapter, msgList)
				}

				msg := p.msg.ProtoReflect().Get(field).Message().Interface()
				if anyMsg, ok := msg.(*anypb.Any); ok {
					msg, err := anypb.UnmarshalNew(anyMsg, proto.UnmarshalOptions{
						Resolver: p.adapter.resolver,
					})
					if err != nil {
						return types.WrapErr(err)
					}

					return p.adapter.NativeToValue(msg)
				}
			}
		}
	}

	return p.adapter.wrapRefVal(refVal)
}

func (p *celMessage) IsSet(field ref.Val) ref.Val {
	if p.fieldTester == nil {
		return types.NewErr("presence testing is not supported for %v", p.refVal.Type())
	}
	return p.fieldTester.IsSet(p.normalize(field))
}

func (p *celMessage) normalize(val ref.Val) ref.Val {
	str, ok := val.(types.String)
	if !ok {
		return val
	}

	name := string(str)
	if canonical, ok := p.aliases[name]; ok {
		if canonical == name {
			return val
		}
		return types.String(canonical)
	}

	if canonical, ok := p.aliases[normalizeProtoFieldName(name)]; ok {
		return types.String(canonical)
	}

	return val
}

// Implement traits.Lister interface by delegating to inner refVal
// This is required for cel.ValueAsProto to work correctly with list types

func (p *celMessage) Add(other ref.Val) ref.Val {
	if lister, ok := p.refVal.(traits.Lister); ok {
		return lister.Add(other)
	}
	return types.NewErr("add not supported on %v", p.refVal.Type())
}

func (p *celMessage) Contains(value ref.Val) ref.Val {
	if container, ok := p.refVal.(traits.Container); ok {
		return container.Contains(value)
	}
	return types.NewErr("contains not supported on %v", p.refVal.Type())
}

func (p *celMessage) Iterator() traits.Iterator {
	if iterable, ok := p.refVal.(traits.Iterable); ok {
		return iterable.Iterator()
	}
	return nil
}

func (p *celMessage) Size() ref.Val {
	if sizer, ok := p.refVal.(traits.Sizer); ok {
		return sizer.Size()
	}
	return types.NewErr("size not supported on %v", p.refVal.Type())
}

// Implement traits.Mapper interface by delegating to inner refVal
// This is required for cel.ValueAsProto to work correctly with map types

func (p *celMessage) Find(key ref.Val) (ref.Val, bool) {
	// For proto messages, delegate to Get() which uses indexer
	// For actual maps, try the inner refVal's Find method
	if mapper, ok := p.refVal.(traits.Mapper); ok {
		return mapper.Find(key)
	}

	// Fall back to Get() for proto message field access
	val := p.Get(key)
	if types.IsError(val) {
		return val, false
	}
	return val, true
}
