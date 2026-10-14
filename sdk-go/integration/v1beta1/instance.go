package v1beta1

import (
	context "context"
	"log/slog"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
)

type (
	InstanceMux[I InstanceType] interface {
		Name() string
		Address() network.Address
		Logger() *slog.Logger
		GetInstance(context.Context) (I, error)

		Actions() *ActionRegistry
		Events() *EventRegistry
		EventSources() *SourceRegistry
		Package() *sharedv1beta1.Package
		Types() *TypeRegistry
	}
	Instance[C any] interface {
		Name() string
		Address() network.Address
		Logger() *slog.Logger

		Config() C
		DataRoot() string
		KeyVal() *KeyVal

		Actions() *ActionRegistry
		Events() *EventRegistry
		EventSources() *SourceRegistry
		Package() *sharedv1beta1.Package
		Schema() *TypeSchema[C]
		Types() *TypeRegistry
	}
	InstanceType interface {
		CheckAuth(context.Context, *basev1beta1.CheckAuthRequest) (*basev1beta1.CheckAuthResponse, error)
		Close() error
	}
)
