package form

import (
	"slices"
	"strings"

	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ ElementType = (*MultiSelectElement)(nil)
var _ LoadElement = (*MultiSelectElement)(nil)
var _ OptionsElement = (*MultiSelectElement)(nil)

type (
	MultiSelectElement struct {
		element *protoformv1beta1.MultiSelectElement
		options []util.MapPair[string, any]
	}
)

func NewMultiSelectElement(elem *Element) *MultiSelectElement {
	sel := elem.proto.GetMultiSelect()
	if sel == nil {
		sel = &protoformv1beta1.MultiSelectElement{}
	}

	elem.proto.Type = &protoformv1beta1.FieldElement_MultiSelect{
		MultiSelect: sel,
	}

	elem.Type = &MultiSelectElement{
		element: sel,
	}

	return elem.Type.(*MultiSelectElement)
}

func (*MultiSelectElement) isElementType()    {}
func (*MultiSelectElement) isLoadElement()    {}
func (*MultiSelectElement) isOptionsElement() {}

func (s *MultiSelectElement) GetOptions() *util.OrderedMap[string, any] {
	slices.SortFunc(s.options, func(a, b util.MapPair[string, any]) int {
		return strings.Compare(a.Key, b.Key)
	})
	return util.NewOrderedMap[string, any](s.options...)
}

func (s *MultiSelectElement) GetMethodName() string {
	return s.element.GetMethodName()
}

func (s *MultiSelectElement) GetKeyExpr() string {
	return s.element.GetKeyExpr()
}

func (s *MultiSelectElement) GetValExpr() string {
	return s.element.GetValExpr()
}

func (s *MultiSelectElement) SetOption(env Env, key string, val protoreflect.Value) {
	s.options = append(s.options, util.NewMapPair(key, any(val)))
}

func (s *MultiSelectElement) Load(env Env) error {
	return LoadOptions(env, s)
}
