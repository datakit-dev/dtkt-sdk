package resource

const (
	Automation   NameType = "automations"
	Connection   NameType = "connections"
	Deployment   NameType = "deployments"
	File         NameType = "files"
	Flow         NameType = "flows"
	FlowRun      NameType = "flowruns"
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
		FlowRun:      segmentPattern,
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
		FlowRun:      coreRoots,
		Integration:  coreRoots,
		Method:       coreReflect,
		Operation:    {Automation, Deployment, FlowRun, Integration},
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
	CoreTypes = []string{
		Integration.String(),
		Deployment.String(),
		Connection.String(),
		Flow.String(),
		FlowRun.String(),
		Service.String(),
		Method.String(),
		Type.String(),
		Operation.String(),
		Organization.String(),
		User.String(),
	}
	coreAliases = map[string]NameType{
		"intgr":        Integration,
		"integration":  Integration,
		"integrations": Integration,

		"depl":        Deployment,
		"deploy":      Deployment,
		"deploys":     Deployment,
		"deployment":  Deployment,
		"deployments": Deployment,

		"conn":        Connection,
		"conns":       Connection,
		"connection":  Connection,
		"connections": Connection,

		"flow":  Flow,
		"flows": Flow,
		"fr":    Flow,

		"flowrun":  FlowRun,
		"flowruns": FlowRun,

		"svc":      Service,
		"service":  Service,
		"services": Service,

		"method":  Method,
		"methods": Method,

		"type":  Type,
		"types": Type,

		"op":         Operation,
		"operation":  Operation,
		"operations": Operation,

		"org":           Organization,
		"organization":  Organization,
		"organizations": Organization,

		"user":  User,
		"users": User,
	}
)

func AddressableTypes() []NameType {
	return coreAddrs
}
