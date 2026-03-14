package form

import (
	"slices"
	"strings"

	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func (*SelectElement) isElementType()    {}
func (*SelectElement) isLoadElement()    {}
func (*SelectElement) isOptionsElement() {}

func (s *SelectElement) GetOptions() *util.OrderedMap[string, any] {
	slices.SortFunc(s.options, func(a, b util.MapPair[string, any]) int {
		return strings.Compare(a.Key, b.Key)
	})
	return util.NewOrderedMap[string, any](s.options...)
}

func (s *SelectElement) GetMethodName() string {
	return s.element.GetMethodName()
}

func (s *SelectElement) GetKeyExpr() string {
	return s.element.GetKeyExpr()
}

func (s *SelectElement) GetValExpr() string {
	return s.element.GetValExpr()
}

func (s *SelectElement) SetOption(env Env, key string, val protoreflect.Value) {
	s.options = append(s.options, util.NewMapPair(key, any(val)))
}

func (s *SelectElement) Load(env Env) error {
	return LoadOptions(env, s)
}
