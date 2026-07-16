package v1beta1

import (
	"slices"
	"strings"

	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

var _ ElementType = (*SelectElement)(nil)
var _ LoadElement = (*SelectElement)(nil)
var _ OptionsElement = (*SelectElement)(nil)

type SelectElement struct {
	element *protoformv1beta1.SelectElement
	options []util.MapPair[string, any]
}

func NewSelectElement(elem *Element) *SelectElement {
	sel := elem.proto.GetSelect()
	if sel == nil {
		sel = &protoformv1beta1.SelectElement{}
	}

	elem.proto.Type = &protoformv1beta1.FieldElement_Select{
		Select: sel,
	}

	elem.Type = &SelectElement{
		element: sel,
	}

	return elem.Type.(*SelectElement)
}

func (s *SelectElement) GetOptions() *util.OrderedMap[string, any] {
	slices.SortFunc(s.options, func(a, b util.MapPair[string, any]) int {
		return strings.Compare(a.Key, b.Key)
	})
	return util.NewOrderedMap(s.options...)
}

func (s *SelectElement) GetMethodName() string {
	return s.element.GetMethodName()
}

func (s *SelectElement) GetLabelExpr() string {
	return s.element.GetLabelExpr()
}

func (s *SelectElement) GetValueExpr() string {
	return s.element.GetValueExpr()
}

func (s *SelectElement) GetFilterExpr() string {
	return s.element.GetFilterExpr()
}

func (s *SelectElement) GetResponseField() string {
	return s.element.GetResponseField()
}

func (s *SelectElement) GetUseFilterField() bool {
	return s.element.GetUseFilterField()
}

func (s *SelectElement) ClearOptions() {
	s.options = nil
}

func (s *SelectElement) SetOption(key string, val any) {
	s.options = append(s.options, util.NewMapPair(key, val))
}

func (s *SelectElement) Load(env Env) error {
	return ResolveOptions(env, s)
}

func (*SelectElement) isElementType()    {}
func (*SelectElement) isLoadElement()    {}
func (*SelectElement) isOptionsElement() {}
