package shared

import (
	"context"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type (
	ConnectorProvider interface {
		// GetConnector returns a Connector for a connection node with the given id.
		// It must match the given package (name & version) or implement the given
		// list of services.
		GetConnector(ctx context.Context, id string, pkg Package, services []string) (Connector, error)
	}
	Connector interface {
		SpecNode() SpecNode
		GetResolver(context.Context) (Resolver, error)
		GetClient(context.Context) (context.Context, common.DynamicClient, error)
	}
	Package interface {
		GetName() string
		GetVersion() string
	}
	Resolver interface {
		protoregistry.ExtensionTypeResolver
		protoregistry.MessageTypeResolver
		RangeServices(func(protoreflect.ServiceDescriptor) bool)
		RangeMethods(func(protoreflect.MethodDescriptor) bool)
		FindMethodByName(protoreflect.FullName) (protoreflect.MethodDescriptor, error)
		RangeFiles(func(protoreflect.FileDescriptor) bool)
		GetValidator() (protovalidate.Validator, error)
	}
)
