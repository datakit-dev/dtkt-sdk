package fivetran

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	fivetranv1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/lib/fivetran/v1"
	"github.com/fivetran/go-fivetran"
)

type (
	Client struct {
		*fivetran.Client
		creds *fivetranv1.Credentials
	}
	InstanceWithCredentials interface {
		v1beta1.InstanceType
		GetFivetranCredentials(context.Context) (*fivetranv1.Credentials, error)
	}
)

func NewClient(creds *fivetranv1.Credentials) (*Client, error) {
	if creds == nil {
		return nil, fmt.Errorf("config cannot be nil")
	} else if creds.ApiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	} else if creds.ApiSecret == "" {
		return nil, fmt.Errorf("api_secret is required")
	}

	return &Client{
		Client: fivetran.New(creds.ApiKey, creds.ApiSecret),
		creds:  creds,
	}, nil
}

func GetClientFromInstance[I InstanceWithCredentials](ctx context.Context, mux v1beta1.InstanceMux[I]) (*Client, error) {
	inst, err := mux.GetInstance(ctx)
	if err != nil {
		return nil, err
	}

	config, err := inst.GetFivetranCredentials(ctx)
	if err != nil {
		return nil, err
	}

	return NewClient(config)
}
