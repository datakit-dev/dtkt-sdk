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
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

var _ v1beta1.Instance[any] = (*instance[any, v1beta1.InstanceType])(nil)

type (
	instance[C any, I v1beta1.InstanceType] struct {
		intgr  *Integration[C, I]
		log    *slog.Logger
		keyval *v1beta1.KeyVal

		name,
		configHash,
		dataRoot string
		created time.Time
		config  C

		impl   I
		cancel context.CancelFunc
	}
)

func newInstance[C any, I v1beta1.InstanceType](intgr *Integration[C, I], check *basev1beta1.CheckConfigRequest) (*instance[C, I], error) {
	config, err := common.UnwrapProtoAnyAs[C](check.GetConfigData())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to unmarshal config: %w", err))
	}

	err = intgr.config.Validate(config)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("config validation failed: %w", err))
	}

	hash := util.HashShort(check.Connection)
	root := filepath.Join(intgr.root, hash)

	err = os.MkdirAll(root, 0o700)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("create data root failed: %w", err))
	}

	ctx, cancel := context.WithCancel(
		middleware.NewRequestContext(intgr.ctx,
			middleware.NewRequest(check.Connection, check.ConfigHash, check.ConfigGen),
		),
	)

	inst := &instance[C, I]{
		intgr:  intgr,
		log:    intgr.log.With(slog.String("connection", check.Connection)),
		keyval: v1beta1.NewKeyVal(newJetStreamKeyVal(ctx, intgr, check.Connection, hash)),

		name:       check.Connection,
		dataRoot:   root,
		configHash: check.ConfigHash,
		created:    time.Now(),
		config:     config,
	}

	impl, err := intgr.newInstance(ctx, inst)
	if err != nil {
		cancel()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	inst.impl = impl
	inst.cancel = cancel

	return inst, nil
}

func (c *instance[C, I]) Name() string {
	return c.name
}

func (c *instance[C, I]) Actions() *v1beta1.ActionRegistry {
	return c.intgr.actions
}

func (c *instance[C, I]) Address() network.Address {
	return c.intgr.addr
}

func (c *instance[C, I]) Config() C {
	return c.config
}

func (c *instance[C, I]) KeyVal() *v1beta1.KeyVal {
	return c.keyval
}

func (c *instance[C, I]) Events() *v1beta1.EventRegistry {
	return c.intgr.events
}

func (c *instance[C, I]) EventSources() *v1beta1.SourceRegistry {
	return c.intgr.sources
}

func (c *instance[C, I]) DataRoot() string {
	return c.dataRoot
}

func (c *instance[C, I]) Logger() *slog.Logger {
	return c.log
}

func (c *instance[C, I]) Package() *sharedv1beta1.Package {
	return c.intgr.spec
}

func (c *instance[C, I]) Schema() *v1beta1.TypeSchema[C] {
	return c.intgr.config
}

func (c *instance[C, I]) Types() *v1beta1.TypeRegistry {
	return c.intgr.registry
}

func (i *instance[C, I]) close() error {
	i.cancel()
	return i.impl.Close()
}
