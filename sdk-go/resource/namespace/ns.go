package namespace

import (
	"regexp"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

type (
	NS struct {
		hierarchy Hierarchy
		cache     util.SyncMap[string, *regexp.Regexp]
	}
	Hierarchy map[*NameType][]*NameType
)

func New(hierarchy Hierarchy) *NS {
	ns := &NS{
		hierarchy: map[*NameType][]*NameType{},
	}

	for typ, sub := range hierarchy {
		typ.ns = ns
		ns.hierarchy[typ] = sub
		for _, typ := range sub {
			typ.ns = ns
			typ.pattern.getRegex(ns)
		}
		typ.pattern.getRegex(ns)
	}

	return ns
}
