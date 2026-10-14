package namespace

var (
	Automation   = NewNameType("automation", "automations", SegmentPattern)
	Connection   = NewNameType("connection", "connections", SegmentPattern)
	Deployment   = NewNameType("deployment", "deployments", SegmentPattern)
	File         = NewNameType("file", "files", SegmentPattern)
	Flow         = NewNameType("flow", "flows", SegmentPattern)
	Integration  = NewNameType("integration", "integrations", SegmentPattern)
	Method       = NewNameType("method", "methods", ProtoPattern)
	Operation    = NewNameType("operation", "operations", SegmentPattern)
	Organization = NewNameType("organization", "organizations", SegmentPattern)
	Service      = NewNameType("service", "services", ProtoPattern)
	Type         = NewNameType("type", "types", ProtoPattern)
	User         = NewNameType("user", "users", SegmentPattern)
)

var (
	core = New(Hierarchy{
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
	})
	coreRoots = []*NameType{
		User,
		Organization,
	}
	coreAddrs = []*NameType{
		Connection,
		Deployment,
	}
	coreReflect = []*NameType{
		Connection,
		Deployment,
		Organization,
		User,
		root,
	}
)

func Core() *NS {
	return core
}
