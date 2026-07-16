package shared

import (
	"context"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var _ ConnectorProvider[Package] = GetConnectorFunc[Package](nil)

type (
	ConnectorProvider[P Package] interface {
		// GetConnector returns a Connector for a connection node with the given id.
		// It must match the given package (name & version) or implement the given
		// list of services.
		GetConnector(node Connection[P]) (Connector, error)
	}
	GetConnectorFunc[P Package] func(Connection[P]) (Connector, error)
	Connection[P Package]       interface {
		SpecNode
		GetPackage() P
		GetServices() []string
	}
	Package interface {
		GetName() string
		GetVersion() string
	}
	Connector interface {
		GetResolver(context.Context) (Resolver, error)
		GetClient(context.Context) (context.Context, Client, error)
	}
	Client interface {
		common.DynamicClient
		Address() network.Address
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

func (f GetConnectorFunc[T]) GetConnector(node Connection[T]) (Connector, error) {
	return f(node)
}
