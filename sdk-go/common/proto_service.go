package common

import (
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	ProtoService  string
	ProtoServices []ProtoService
)

func (st *ProtoServices) Add(names ...string) {
	copy := slices.Clone(*st)
	for _, name := range names {
		if !slices.Contains(copy, ProtoService(name)) {
			*st = append(*st, ProtoService(name))
		}
	}
}

func (st ProtoServices) Names() []string {
	return st.Strings()
}

func (st ProtoServices) Strings() []string {
	return util.SliceSet(util.SliceMap(st, func(s ProtoService) string {
		return s.String()
	}))
}

func (st ProtoService) ProtoPkg() ProtoPackage {
	return ProtoPackage(protoreflect.FullName(st).Parent())
}

func (st ProtoService) String() string {
	return string(st)
}
