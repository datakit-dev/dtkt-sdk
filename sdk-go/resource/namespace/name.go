package namespace

import (
	"path"
	"strings"
)

type (
	Name    []Segment
	Segment [2]string
)

func EmptyName() Name {
	return Name{}
}

func NewName(input string) Name {
	if input == "" {
		return EmptyName()
	}

	parts := strings.Split(strings.Trim(input, "/"), "/")
	name := make(Name, 0, (len(parts)+1)/2)
	for i := 0; i < len(parts); i++ {
		if i+1 < len(parts) {
			name = append(name, Segment{parts[i], parts[i+1]})
			i++
		}
	}

	return name
}

func (n Name) IsValid(ns *NS) bool {
	_, ok := n.Type(ns)
	return ok
}

func (n Name) Type(ns *NS) (typ *NameType, _ bool) {
	for typ = range ns.hierarchy {
		if typ.IsName(n.String()) {
			return typ, true
		}
	}
	return
}

func (n Name) IsOneOf(types ...*NameType) bool {
	for _, typ := range types {
		if typ.IsName(n.String()) {
			return true
		}
	}
	return false
}

func (n Name) Equal(other Name) bool {
	return n.String() == other.String()
}

func (n Name) Short() string {
	name := path.Base(n.String())
	if name == "." || name == "/" {
		return ""
	}
	return name
}

func (n Name) Append(typ *NameType, val string) Name {
	return append(n, typ.New(val)...)
}

func (n Name) First() Name {
	if len(n) == 0 {
		return Name{}
	}
	return n[:1]
}

func (n Name) Last() Name {
	if len(n) == 0 {
		return Name{}
	}
	return n[len(n)-1:]
}

func (n Name) Parent() Name {
	if len(n) <= 1 {
		return Name{}
	}
	return n[:len(n)-1]
}

func (n Name) HasParent() bool {
	return len(n) > 1
}

func (n Name) String() string {
	var parts []string
	for _, p := range n {
		k, v := p[0], p[1]
		if k == "" || v == "" {
			parts = append(parts, k)
			continue
		}
		parts = append(parts, k, v)
	}
	return strings.Join(parts, "/")
}

func (n Name) Split() []string {
	var parts []string
	for _, p := range n {
		k, v := p[0], p[1]
		if k == "" || v == "" {
			parts = append(parts, k)
			continue
		}
		parts = append(parts, k, v)
	}
	return parts
}

func (n Segment) String() string {
	return strings.Join(n[:], "/")
}
