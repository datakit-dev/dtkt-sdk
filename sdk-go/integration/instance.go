package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integration/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/middleware"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/resource"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

var _ v1beta1.InstanceConfig[any] = (*instanceConfig[any, v1beta1.InstanceType])(nil)

type (
	instance struct {
		v1beta1.InstanceType

		resourceName,
		configHash string
		createTime time.Time
		cancel     context.CancelFunc
	}
	instanceConfig[C any, I v1beta1.InstanceType] struct {
		intgr  *Integration[C, I]
		log    *slog.Logger
		root   string
		config C
	}
	NewInstanceFunc[C any, I v1beta1.InstanceType] func(context.Context, v1beta1.InstanceConfig[C]) (I, error)
)

func newInstance[C any, I v1beta1.InstanceType](intgr *Integration[C, I], check *basev1beta1.CheckConfigRequest) (*instance, error) {
	config, err := common.UnwrapProtoAnyAs[C](check.GetConfigData())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to unmarshal config: %w", err))
	}

	err = intgr.config.Validate(config)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("config validation failed: %w", err))
	}

	root := filepath.Join(intgr.root, util.HashShort(check.Connection))

	err = os.MkdirAll(root, 0o700)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("create data root failed: %w", err))
	}

	var logOpts []any
	if resource.Connection.IsName(check.Connection) {
		logOpts = append(logOpts,
			slog.String("connection", check.Connection),
		)
	}

	ctx, cancel := context.WithCancel(
		middleware.NewRequestContext(intgr.ctx,
			middleware.NewRequest(check.Connection, check.ConfigHash, check.ConfigGen),
		),
	)
	inst, err := intgr.newInstance(ctx, &instanceConfig[C, I]{
		intgr:  intgr,
		log:    intgr.log.With(logOpts...),
		root:   root,
		config: config,
	})
	if err != nil {
		cancel()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return &instance{
		InstanceType: inst,
		resourceName: check.Connection,
		configHash:   check.ConfigHash,
		createTime:   time.Now(),
		cancel:       cancel,
	}, nil
}

func (c *instanceConfig[C, I]) Actions() *v1beta1.ActionRegistry {
	return c.intgr.actions
}

func (c *instanceConfig[C, I]) Address() network.Address {
	return c.intgr.addr
}

func (c *instanceConfig[C, I]) Get() C {
	return c.config
}

func (c *instanceConfig[C, I]) Events() *v1beta1.EventRegistry {
	return c.intgr.events
}

func (c *instanceConfig[C, I]) EventSources() *v1beta1.EventSourceRegistry {
	return c.intgr.sources
}

func (c *instanceConfig[C, I]) DataRoot() string {
	return c.root
}

func (c *instanceConfig[C, I]) Logger() *slog.Logger {
	return c.log
}

func (c *instanceConfig[C, I]) Package() *sharedv1beta1.Package {
	return c.intgr.spec
}

func (c *instanceConfig[C, I]) Schema() *v1beta1.TypeSchema[C] {
	return c.intgr.config
}

func (c *instanceConfig[C, I]) Types() *v1beta1.TypeRegistry {
	return c.intgr.registry
}

func (i *instance) close() error {
	i.cancel()
	return i.InstanceType.Close()
}
