package v1beta1

import (
	"slices"
	"strings"

	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protoform"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

var _ ElementType = (*SelectElement)(nil)
var _ LoadElement = (*SelectElement)(nil)
var _ OptionsElement = (*SelectElement)(nil)
var _ ElementType = (*MultiSelectElement)(nil)
var _ LoadElement = (*MultiSelectElement)(nil)
var _ OptionsElement = (*MultiSelectElement)(nil)

type (
	SelectElement struct {
		proto   *protoformv1beta1.SelectElement
		options []util.MapPair[string, any]
	}
	MultiSelectElement struct {
		proto   *protoformv1beta1.MultiSelectElement
		options []util.MapPair[string, any]
	}
)

func NewSelectElement(elem *Element, opts ...ElementTypeOpt[*SelectElement]) *SelectElement {
	proto := elem.proto.GetSelect()
	if proto == nil {
		proto = &protoformv1beta1.SelectElement{}
	}

	selec := &SelectElement{
		proto: proto,
	}

	elem.proto.Type = &protoformv1beta1.FieldElement_Select{
		Select: proto,
	}
	elem.Type = selec

	for _, opt := range opts {
		if opt != nil {
			opt(selec)
		}
	}

	return selec
}

func NewMultiSelectElement(elem *Element, opts ...ElementTypeOpt[*MultiSelectElement]) *MultiSelectElement {
	proto := elem.proto.GetMultiSelect()
	if proto == nil {
		proto = &protoformv1beta1.MultiSelectElement{}
	}

	multiSelect := &MultiSelectElement{
		proto: proto,
	}

	elem.proto.Type = &protoformv1beta1.FieldElement_MultiSelect{
		MultiSelect: proto,
	}
	elem.Type = multiSelect

	for _, opt := range opts {
		if opt != nil {
			opt(multiSelect)
		}
	}

	return multiSelect
}

func (s *SelectElement) GetOptions() *util.OrderedMap[string, any] {
	slices.SortFunc(s.options, func(a, b util.MapPair[string, any]) int {
		return strings.Compare(a.Key, b.Key)
	})
	return util.NewOrderedMap(s.options...)
}

func (s *SelectElement) GetMethodName() string {
	return s.proto.GetMethodName()
}

func (s *SelectElement) GetLabelExpr() string {
	return s.proto.GetLabelExpr()
}

func (s *SelectElement) GetValueExpr() string {
	return s.proto.GetValueExpr()
}

func (s *SelectElement) GetFilterExpr() string {
	return s.proto.GetFilterExpr()
}

func (s *SelectElement) GetResponseField() string {
	return s.proto.GetResponseField()
}

func (s *SelectElement) GetUseFilterField() bool {
	return s.proto.GetUseFilterField()
}

func (s *SelectElement) ClearOptions() {
	s.options = nil
}

func (s *SelectElement) SetOption(key string, val any) {
	s.options = append(s.options, util.NewMapPair(key, val))
}

func (s *SelectElement) Load(env protoform.Env) error {
	return ResolveOptions(env, s)
}

func (s *MultiSelectElement) GetOptions() *util.OrderedMap[string, any] {
	slices.SortFunc(s.options, func(a, b util.MapPair[string, any]) int {
		return strings.Compare(a.Key, b.Key)
	})
	return util.NewOrderedMap(s.options...)
}

func (s *MultiSelectElement) GetMethodName() string {
	return s.proto.GetMethodName()
}

func (s *MultiSelectElement) GetLabelExpr() string {
	return s.proto.GetLabelExpr()
}

func (s *MultiSelectElement) GetValueExpr() string {
	return s.proto.GetValueExpr()
}

func (s *MultiSelectElement) GetFilterExpr() string {
	return s.proto.GetFilterExpr()
}

func (s *MultiSelectElement) GetResponseField() string {
	return s.proto.GetResponseField()
}

func (s *MultiSelectElement) GetUseFilterField() bool {
	return s.proto.GetUseFilterField()
}

func (s *MultiSelectElement) SetOption(key string, val any) {
	s.options = append(s.options, util.NewMapPair(key, val))
}

func (s *MultiSelectElement) ClearOptions() {
	s.options = nil
}

func (s *MultiSelectElement) Load(env protoform.Env) error {
	return ResolveOptions(env, s)
}

func (*SelectElement) isElementType()    {}
func (*SelectElement) isLoadElement()    {}
func (*SelectElement) isOptionsElement() {}

func (*MultiSelectElement) isElementType()    {}
func (*MultiSelectElement) isLoadElement()    {}
func (*MultiSelectElement) isOptionsElement() {}
