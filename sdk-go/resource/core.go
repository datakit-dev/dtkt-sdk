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

// func init() {
// coreTypes = []NameType{
// 	Automation,
// 	Connection,
// 	Deployment,
// 	File,
// 	Flow,
// 	Integration,
// 	Method,
// 	Operation,
// 	Organization,
// 	Service,
// 	Type,
// 	User,
// }

// Automation   NameType = "dtkt.core.v1.Automation"
// Connection   NameType = "dtkt.core.v1.Connection"
// Deployment   NameType = "dtkt.core.v1.Deployment"
// File         NameType = "dtkt.core.v1.File"
// Flow         NameType = "dtkt.core.v1.Flow"
// Integration  NameType = "dtkt.core.v1.Integration"
// Method       NameType = "dtkt.core.v1.Method"
// Service      NameType = "dtkt.core.v1.Service"
// Type         NameType = "dtkt.core.v1.Type"

// 	for _, coreType := range coreTypes {
// 		desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(coreType.String()))
// 		if err != nil {
// 			log.Fatalf("%s: %s", coreType, err)
// 		}

// 		if opts, ok := desc.Options().(*descriptorpb.MessageOptions); ok && opts != nil {
// 			if proto.HasExtension(opts, annotations.E_Resource) {
// 				if ext, ok := proto.GetExtension(opts, annotations.E_Resource).(*annotations.ResourceDescriptor); ok {
// 					for _, pattern := range ext.GetPattern() {
// 						fmt.Println(coreType, pattern)
// 					}
// 				}
// 			}
// 		}
// 	}
// }
