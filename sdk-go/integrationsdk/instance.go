package integrationsdk

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/middleware"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
)

type (
	Instance[I v1beta1.InstanceType] struct {
		*middleware.Request
		v1beta1.InstanceType
	}
	NewInstanceFunc[C any, I v1beta1.InstanceType] func(context.Context, C) (I, error)
)

func NewInstance[C any, I v1beta1.InstanceType](ctx context.Context, intgr *Integration[C, I], req *basev1beta1.CheckConfigRequest) (*Instance[I], error) {
	value, err := common.UnwrapProtoAnyAs[C](req.GetConfigData())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to unmarshal config: %w", err))
	}

	err = intgr.config.Validate(value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("config validation failed: %w", err))
	}

	mreq := middleware.NewRequest(req.Connection, req.ConfigHash, req.ConfigGen)
	inst, err := intgr.newInstance(middleware.AddRequestToContext(ctx, mreq), value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return &Instance[I]{
		Request:      mreq,
		InstanceType: inst,
	}, nil
}
