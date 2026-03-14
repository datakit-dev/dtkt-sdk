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
		Address() network.Address
		Logger() *slog.Logger
		Package() *sharedv1beta1.Package
		ConfigSchema() *sharedv1beta1.TypeSchema
		GetInstance(context.Context) (I, error)
		GetDataRoot(context.Context) (string, error)
		Types() *TypeRegistry
		Actions() *ActionRegistry
		Events() *EventRegistry
		EventSources() *EventSourceRegistry
	}
	InstanceType interface {
		CheckAuth(context.Context, *basev1beta1.CheckAuthRequest) (*basev1beta1.CheckAuthResponse, error)
		Close() error
	}
)
