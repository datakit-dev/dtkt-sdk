package resource

import "sync"

type (
	Hierarchy map[NameType][]NameType
	Patterns  map[NameType]Pattern
)

var (
	hierarchy    Hierarchy
	hierarchyMut sync.Mutex
)

func SetHierarchy(h Hierarchy) {
	hierarchyMut.Lock()
	defer hierarchyMut.Unlock()

	patternCache.Clear()
	for typ := range h {
		typ.Pattern().loadOrStore()
	}
	hierarchy = h
}
