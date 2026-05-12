package resource

const (
	Automation   NameType = "automations"
	Connection   NameType = "connections"
	Deployment   NameType = "deployments"
	File         NameType = "files"
	Flow         NameType = "flows"
	Integration  NameType = "integrations"
	Method       NameType = "methods"
	Operation    NameType = "operations"
	Organization NameType = "organizations"
	Service      NameType = "services"
	Type         NameType = "types"
	User         NameType = "users"
	root         NameType = ""
)

var (
	corePatterns = map[NameType]Pattern{
		Automation:   segmentPattern,
		Connection:   segmentPattern,
		Deployment:   segmentPattern,
		File:         segmentPattern,
		Flow:         segmentPattern,
		Integration:  segmentPattern,
		Method:       protoPattern,
		Operation:    segmentPattern,
		Organization: segmentPattern,
		Service:      protoPattern,
		Type:         protoPattern,
		User:         segmentPattern,
	}
	coreHierarchy = map[NameType][]NameType{
		Automation:   coreRoots,
		Connection:   coreRoots,
		Deployment:   coreRoots,
		File:         coreAddrs,
		Flow:         coreRoots,
		Integration:  coreRoots,
		Method:       coreReflect,
		Operation:    {Automation, Deployment, Integration},
		Organization: {},
		Service:      coreReflect,
		Type:         coreReflect,
		User:         {},
	}
	coreRoots = []NameType{
		User,
		Organization,
	}
	coreAddrs = []NameType{
		Connection,
		Deployment,
	}
	coreReflect = []NameType{
		Connection,
		Deployment,
		Organization,
		User,
		root,
	}
)

func AddressableTypes() []NameType {
	return coreAddrs
}
