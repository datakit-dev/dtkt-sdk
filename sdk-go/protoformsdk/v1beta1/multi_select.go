package v1beta1

import (
	"slices"
	"strings"

	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
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

func (s *MultiSelectElement) GetOptions() *util.OrderedMap[string, any] {
	slices.SortFunc(s.options, func(a, b util.MapPair[string, any]) int {
		return strings.Compare(a.Key, b.Key)
	})
	return util.NewOrderedMap(s.options...)
}

func (s *MultiSelectElement) GetMethodName() string {
	return s.element.GetMethodName()
}

func (s *MultiSelectElement) GetLabelExpr() string {
	return s.element.GetLabelExpr()
}

func (s *MultiSelectElement) GetValueExpr() string {
	return s.element.GetValueExpr()
}

func (s *MultiSelectElement) GetFilterExpr() string {
	return s.element.GetFilterExpr()
}

func (s *MultiSelectElement) GetResponseField() string {
	return s.element.GetResponseField()
}

func (s *MultiSelectElement) GetUseFilterField() bool {
	return s.element.GetUseFilterField()
}

func (s *MultiSelectElement) SetOption(key string, val any) {
	s.options = append(s.options, util.NewMapPair(key, val))
}

func (s *MultiSelectElement) ClearOptions() {
	s.options = nil
}

func (s *MultiSelectElement) Load(env Env) error {
	return ResolveOptions(env, s)
}

func (*MultiSelectElement) isElementType()    {}
func (*MultiSelectElement) isLoadElement()    {}
func (*MultiSelectElement) isOptionsElement() {}
