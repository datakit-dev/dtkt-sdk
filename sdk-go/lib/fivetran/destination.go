package fivetran

import (
	"context"
	"log"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	fivetranv1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/lib/fivetran/v1"
	replicationv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/replication/v1beta1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type (
	DestinationService[I InstanceWithDestination] struct {
		replicationv1beta1.UnimplementedDestinationServiceServer
		mux   v1beta1.InstanceMux[I]
		types []*replicationv1beta1.DestinationType
	}
	InstanceWithDestination interface {
		InstanceWithCredentials
		GetFivetranDestination(ctx context.Context, typeId string) (*fivetranv1.Destination, error)
	}
)

func NewDestinationService[I InstanceWithDestination](mux v1beta1.InstanceMux[I], types ...*replicationv1beta1.DestinationType) *DestinationService[I] {
	if len(types) == 0 {
		log.Fatal("fivetran destination service: one or more destination types required")
	}

	return &DestinationService[I]{
		mux:   mux,
		types: types,
	}
}

func GetDestinationFromInstance[I InstanceWithDestination](ctx context.Context, mux v1beta1.InstanceMux[I], typeId string) (*fivetranv1.Destination, error) {
	inst, err := mux.GetInstance(ctx)
	if err != nil {
		return nil, err
	}

	return inst.GetFivetranDestination(ctx, typeId)
}

func (s *DestinationService[I]) ListDestinationTypes(ctx context.Context, req *replicationv1beta1.ListDestinationTypesRequest) (*replicationv1beta1.ListDestinationTypesResponse, error) {
	return &replicationv1beta1.ListDestinationTypesResponse{
		DestinationTypes: s.types,
	}, nil
}

func (s *DestinationService[I]) GetDestination(ctx context.Context, req *replicationv1beta1.GetDestinationRequest) (*replicationv1beta1.GetDestinationResponse, error) {
	client, err := GetClientFromInstance(ctx, s.mux)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	details, err := client.NewDestinationDetails().
		DestinationID(req.Id).
		DoCustom(ctx)
	if err != nil {
		if details.Message == "" {
			details.Message = err.Error()
		}
		return nil, status.Errorf(codes.FailedPrecondition, "get group: %s", details.Message)
	}

	group, err := client.NewGroupDetails().GroupID(details.Data.GroupID).Do(ctx)
	if err != nil {
		if group.Message == "" {
			group.Message = err.Error()
		}
		return nil, status.Errorf(codes.FailedPrecondition, "get group details: %s", group.Message)
	}

	config, err := wrapProtoFromValue[*fivetranv1.Destination](map[string]any{
		"details": details.Data.DestinationDetailsBase,
		"config":  details.Data.Config,
	})
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "destination to proto: %s", group.Message)
	}

	return &replicationv1beta1.GetDestinationResponse{
		Destination: &replicationv1beta1.Destination{
			TypeId: details.Data.Service,
			Id:     details.Data.ID,
			Name:   group.Data.Name,
			Config: config,
		},
	}, nil
}

func (s *DestinationService[I]) CreateDestination(ctx context.Context, req *replicationv1beta1.CreateDestinationRequest) (*replicationv1beta1.CreateDestinationResponse, error) {
	client, err := GetClientFromInstance(ctx, s.mux)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	dest, err := GetDestinationFromInstance(ctx, s.mux, req.TypeId)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	if req.Config != nil && !req.Config.MessageIs(dest) {
		return nil, status.Errorf(codes.InvalidArgument, "provided config type invalid: %s, expected: %s", req.Config.MessageName(), dest.ProtoReflect().Descriptor().FullName())
	} else if req.Config != nil {
		destOverrides, err := req.Config.UnmarshalNew()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "provided config failed to unmarshal: %s", err.Error())
		}

		proto.Merge(dest, destOverrides)
	}

	b, err := protojson.Marshal(dest)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	var destMap map[string]any
	err = encoding.FromJSONV2(b, &destMap)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	group, err := client.NewGroupCreate().Name(req.Name).Do(ctx)
	if err != nil {
		if group.Message == "" {
			group.Message = err.Error()
		}
		return nil, status.Errorf(codes.FailedPrecondition, "error creating group: %s", group.Message)
	}

	resp, err := client.NewDestinationCreate().
		GroupID(group.Data.ID).
		Service(req.TypeId).
		ConfigCustom(&destMap).
		DoCustom(ctx)
	if err != nil {
		if resp.Message == "" {
			resp.Message = err.Error()
		}
		return nil, status.Errorf(codes.FailedPrecondition, "error creating destination: %s", resp.Message)
	}

	config, err := wrapProtoFromValue[*fivetranv1.Destination](map[string]any{
		"details": resp.Data.DestinationDetailsBase,
		"config":  resp.Data.Config,
	})
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "destination to proto: %s", group.Message)
	}

	return &replicationv1beta1.CreateDestinationResponse{
		Destination: &replicationv1beta1.Destination{
			TypeId: resp.Data.Service,
			Id:     resp.Data.ID,
			Name:   group.Data.Name,
			Config: config,
		},
	}, nil
}

func (s *DestinationService[I]) UpdateDestination(ctx context.Context, req *replicationv1beta1.UpdateDestinationRequest) (*replicationv1beta1.UpdateDestinationResponse, error) {
	// 	inst, err := s.mux.GetInstance(ctx)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	instConfig, err := s.getInstanceConfig(inst, req.ConfigId)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	configMap, err := common.ToMap(instConfig.config.Request())
	// 	if err != nil {
	// 		return nil, err
	// 	} else if configMap == nil {
	// 		// configMap = req.Config.AsMap()
	// 	} else {
	// 		// configMap = common.JSONMap(configMap).Merge(req.Config.AsMap())
	// 	}

	// 	dstConfig, client, err := s.getConfigAndClient(req.ServiceConfig)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	update := client.NewDestinationUpdate().
	// 		DestinationID(req.Id).
	// 		ConfigCustom(util.ToPointer(configMap))

	// 	// if dstConfig != nil {
	// 	// 	update.RunSetupTests(dstConfig.RunSetupTests)

	// 	// 	if dstConfig.Region != "" {
	// 	// 		update.Region(dstConfig.Region)
	// 	// 	}

	// 	// 	if dstConfig.TimeZoneOffset != "" {
	// 	// 		update.
	// 	// 			TimeZoneOffset(dstConfig.TimeZoneOffset).
	// 	// 			DaylightSavingTimeEnabled(dstConfig.DaylightSavingTimeEnabled)
	// 	// 	}
	// 	// }

	// 	updateResp, err := update.DoCustom(ctx)
	// 	if err != nil {
	// 		if updateResp.Message == "" {
	// 			updateResp.Message = err.Error()
	// 		}
	// 		return &replicationv1beta1.UpdateDestinationResponse{
	// 			Updated: false,
	// 			Error:   fmt.Sprintf("error updating destination: %s", updateResp.Message),
	// 		}, nil
	// 	}

	// 	// configStruct, err := structpb.NewStruct(updateResp.Data.Config)
	// 	// if err != nil {
	// 	// 	return nil, err
	// 	// }

	// 	groupResp, err := client.NewGroupDetails().GroupID(updateResp.Data.GroupID).Do(ctx)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	return &replicationv1beta1.UpdateDestinationResponse{
	// 		Updated: true,
	// 		Destination: &replicationv1beta1.Destination{
	// 			// Service:  replicationv1beta1.ServiceType_SERVICE_TYPE_FIVETRAN,
	// 			ConfigId: req.ConfigId,
	// 			Id:       updateResp.Data.ID,
	// 			Name:     groupResp.Data.Name,
	// 			// Config:   configStruct,
	// 		},
	// 	}, nil
	return nil, nil
}

func (s *DestinationService[I]) DeleteDestination(ctx context.Context, req *replicationv1beta1.DeleteDestinationRequest) (*replicationv1beta1.DeleteDestinationResponse, error) {
	// 	_, client, err := s.getConfigAndClient(req.ServiceConfig)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	deleteResp, err := client.NewDestinationDelete().DestinationID(req.Id).Do(ctx)
	// 	if err != nil {
	// 		if deleteResp.Message == "" {
	// 			deleteResp.Message = err.Error()
	// 		}
	// 		return &replicationv1beta1.DeleteDestinationResponse{
	// 			Deleted: false,
	// 			Error:   deleteResp.Message,
	// 		}, nil
	// 	}

	// 	return &replicationv1beta1.DeleteDestinationResponse{
	// 		Deleted: true,
	// 	}, nil

	return nil, nil
}
