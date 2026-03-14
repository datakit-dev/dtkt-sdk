package integrationsdk

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/middleware"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	instance struct {
		t      *testing.T
		config *config
	}
	instanceProto struct {
		t      *testing.T
		config *corev1.Flow
	}
	config struct {
		Foo string `json:"foo"`
	}
)

func newInstance(t *testing.T, config *config) (*instance, error) {
	t.Logf("newInstance called with config: %v", config)
	return &instance{
		t:      t,
		config: config,
	}, nil
}

func newInstanceProto(t *testing.T, config *corev1.Flow) (*instanceProto, error) {
	t.Logf("newInstance called with config: %v", config)
	return &instanceProto{
		t:      t,
		config: config,
	}, nil
}

func (i *instance) Close() error {
	i.t.Log("Close called")
	return nil
}

func (i *instance) CheckAuth(context.Context, *basev1beta1.CheckAuthRequest) (*basev1beta1.CheckAuthResponse, error) {
	return &basev1beta1.CheckAuthResponse{}, nil
}

func (i *instanceProto) Close() error {
	i.t.Log("Close called")
	return nil
}

func (i *instanceProto) CheckAuth(context.Context, *basev1beta1.CheckAuthRequest) (*basev1beta1.CheckAuthResponse, error) {
	return &basev1beta1.CheckAuthResponse{}, nil
}

func TestIntegrationInstance(t *testing.T) {
	ident := &sharedv1beta1.Package_Identity{
		Name:    "FooBar",
		Version: "0.1.0",
	}

	intgr, err := New(&sharedv1beta1.Package{
		Identity: ident,
		Icon:     "https://foo.bar/logo.png",
		Type:     sharedv1beta1.PackageType_PACKAGE_TYPE_GO,
	}, func(ctx context.Context, config *config) (*instance, error) {
		return newInstance(t, config)
	})
	if err != nil {
		t.Fatal(err)
	}

	srv := NewServer(intgr)
	defer srv.Stop()

	errCh := make(chan error)
	go func() {
		err := srv.Serve()
		if err != nil {
			errCh <- err
		}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		t.Fatalf("server failed to start: %v", err)
	}

	client, err := network.DialGRPCClientUsing(intgr.conn,
		basev1beta1.NewBaseServiceClient,
	)
	if err != nil {
		t.Fatalf("failed to dial integration: %v", err)
	}

	var (
		config = config{
			Foo: "bar",
		}
		configHash = util.AnyHash(config)
	)

	anyConfig, err := common.WrapProtoAny(config)
	if err != nil {
		t.Fatal(err)
	}

	configReq := &basev1beta1.CheckConfigRequest{
		Connection: "foobar",
		ConfigData: anyConfig,
		ConfigHash: hex.EncodeToString(configHash[:]),
		ConfigGen:  1,
	}

	checkConfig, err := client.CheckConfig(t.Context(), configReq)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("CheckConfig response: %v", checkConfig)
	}

	ctx = middleware.AddRequestToContext(t.Context(), middleware.NewRequest(
		configReq.Connection,
		configReq.ConfigHash,
		configReq.ConfigGen,
	))

	checkAuth, err := client.CheckAuth(ctx, &basev1beta1.CheckAuthRequest{})
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("CheckAuth response: %v", checkAuth)
	}

	srv.Stop()
}

func TestIntegrationInstanceProto(t *testing.T) {
	ident := &sharedv1beta1.Package_Identity{
		Name:    "FooBar",
		Version: "0.1.0",
	}

	intgr, err := New(&sharedv1beta1.Package{
		Identity: ident,
		Icon:     "https://foo.bar/logo.png",
		Type:     sharedv1beta1.PackageType_PACKAGE_TYPE_GO,
	}, func(ctx context.Context, config *corev1.Flow) (*instanceProto, error) {
		return newInstanceProto(t, config)
	})
	if err != nil {
		t.Fatal(err)
	}

	srv := NewServer(intgr)
	defer srv.Stop()

	errCh := make(chan error)
	go func() {
		err := srv.Serve()
		if err != nil {
			errCh <- err
		}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		t.Fatalf("server failed to start: %v", err)
	}

	client, err := network.DialGRPCClientUsing(intgr.conn,
		basev1beta1.NewBaseServiceClient,
	)
	if err != nil {
		t.Fatal(err)
	}

	var (
		config = &corev1.Flow{
			Name: "users/jordan/flows/test-flow",
			Uid:  uuid.New().String(),
			Spec: &corev1.FlowSpecMetadata{
				Version: &corev1.FlowSpecMetadata_V1Beta1{
					V1Beta1: &flowv1beta1.Flow{
						Name: "Test Flow",
					},
				},
			},
			Graph: &corev1.FlowGraphMetadata{
				Version: &corev1.FlowGraphMetadata_V1Beta1{
					V1Beta1: &flowv1beta1.Graph{},
				},
			},
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		}
		configHash = util.AnyHash(config)
	)

	anyConfig, err := common.WrapProtoAny(config)
	if err != nil {
		t.Fatal(err)
	}

	configReq := &basev1beta1.CheckConfigRequest{
		Connection: "foobar",
		ConfigData: anyConfig,
		ConfigHash: hex.EncodeToString(configHash[:]),
		ConfigGen:  1,
	}

	checkConfig, err := client.CheckConfig(t.Context(), configReq)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("CheckConfig response: %v", checkConfig)
	}

	ctx = middleware.AddRequestToContext(t.Context(), middleware.NewRequest(
		configReq.Connection,
		configReq.ConfigHash,
		configReq.ConfigGen,
	))

	checkAuth, err := client.CheckAuth(ctx, &basev1beta1.CheckAuthRequest{})
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("CheckAuth response: %v", checkAuth)
	}

	srv.Stop()
}
