package integrationsdk

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
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
		*middleware.Request
		v1beta1.InstanceType

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

func NewInstance[C any, I v1beta1.InstanceType](intgr *Integration[C, I], check *basev1beta1.CheckConfigRequest) (*instance, error) {
	config, err := common.UnwrapProtoAnyAs[C](check.GetConfigData())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to unmarshal config: %w", err))
	}

	err = intgr.schema.Validate(config)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("config validation failed: %w", err))
	}

	req := middleware.NewRequest(check.Connection, check.ConfigHash, check.ConfigGen)
	root := filepath.Join(intgr.root, util.HashShort(req.ResourceName()))
	err = os.MkdirAll(root, 0o700)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("create data root failed: %w", err))
	}

	var logOpts []any
	if resource.Connection.IsName(req.ResourceName()) {
		logOpts = append(logOpts, slog.String("connection", req.ResourceName()))
	}

	logOpts = append(logOpts, slog.String("instance", req.HashShort()))

	ctx, cancel := context.WithCancel(middleware.AddRequestToContext(context.Background(), req))
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
		Request:      req,
		InstanceType: inst,
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
	return c.intgr.schema
}

func (c *instanceConfig[C, I]) Types() *v1beta1.TypeRegistry {
	return c.intgr.registry
}

func (i *instance) close() error {
	i.cancel()
	return i.InstanceType.Close()
}
