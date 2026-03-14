package fivetran

import (
	"context"
	"slices"
)

type (
	SourceService struct {
		configs []SourceConfig
	}
)

func NewSourceService(ctx context.Context, configFuncs ...SourceConfigFunc) (*SourceService, error) {
	var configs []SourceConfig
	for configFunc := range slices.Values(configFuncs) {
		if configFunc != nil {
			if config, err := configFunc(ctx); err != nil {
				return nil, err
			} else {
				configs = append(configs, config)
			}
		}
	}

	return &SourceService{
		configs: configs,
	}, nil
}

// func (ss *SourceService) Find(ctx context.Context, replicator replicationv1beta1.ServiceType, configID string) (SourceConfig, error) {
// 	var configs = []SourceConfig{}

// 	for _, s := range ss.configs {
// 		if replicator == replicationv1beta1.ServiceType_SERVICE_TYPE_UNSPECIFIED || s.ReplicationService() == replicator {
// 			configs = append(configs, s)
// 		}
// 	}

// 	for _, source := range configs {
// 		if source.GetID() == configID {
// 			return source, nil
// 		}
// 	}

// 	return nil, fmt.Errorf(`source %s not found for service: %s`, configID, replicator)
// }

// func (ss *SourceService) List(ctx context.Context, replicator replicationv1beta1.ServiceType) ([]*replicationv1beta1.SourceConfig, error) {
// 	var configs = []*replicationv1beta1.SourceConfig{}

// 	for _, s := range ss.configs {
// 		if replicator == replicationv1beta1.ServiceType_SERVICE_TYPE_UNSPECIFIED || s.ReplicationService() == replicator {
// 			source, err := s.ToProto(ctx)
// 			if err != nil {
// 				return nil, err
// 			}
// 			configs = append(configs, source)
// 		}
// 	}

// 	return configs, nil
// }

// func (ss *SourceService) ListSourceConfigs(ctx context.Context, req *replicationv1beta1.ListSourceConfigsRequest) (*replicationv1beta1.ListSourceConfigsResponse, error) {
// 	if req.GetServiceConfig() == nil {
// 		return nil, fmt.Errorf("replication service config required")
// 	}

// 	configs, err := ss.List(ctx, req.GetServiceConfig().GetService())
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &replicationv1beta1.ListSourceConfigsResponse{
// 		SourceConfigs: configs,
// 	}, nil
// }

// func (ss *SourceService) CheckSourceAuth(ctx context.Context, req *replicationv1beta1.CheckSourceAuthRequest) (*replicationv1beta1.CheckSourceAuthResponse, error) {
// 	config, err := ss.Find(ctx, req.Service, req.ConfigId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return config.CheckSourceAuth(ctx, req)
// }

// func (ss *SourceService) GetSource(ctx context.Context, req *replicationv1beta1.GetSourceRequest) (*replicationv1beta1.GetSourceResponse, error) {
// 	config, err := ss.Find(ctx, req.ServiceConfig.Service, req.ConfigId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return config.GetSource(ctx, req)
// }

// func (ss *SourceService) CreateSource(ctx context.Context, req *replicationv1beta1.CreateSourceRequest) (*replicationv1beta1.CreateSourceResponse, error) {
// 	config, err := ss.Find(ctx, req.ServiceConfig.Service, req.ConfigId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return config.CreateSource(ctx, req)
// }

// func (ss *SourceService) UpdateSource(ctx context.Context, req *replicationv1beta1.UpdateSourceRequest) (*replicationv1beta1.UpdateSourceResponse, error) {
// 	config, err := ss.Find(ctx, req.ServiceConfig.Service, req.ConfigId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return config.UpdateSource(ctx, req)
// }

// func (ss *SourceService) DeleteSource(ctx context.Context, req *replicationv1beta1.DeleteSourceRequest) (*replicationv1beta1.DeleteSourceResponse, error) {
// 	config, err := ss.Find(ctx, req.ServiceConfig.Service, req.ConfigId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return config.DeleteSource(ctx, req)
// }

// func (ss *SourceService) StartSync(ctx context.Context, req *replicationv1beta1.StartSyncRequest) (*replicationv1beta1.StartSyncResponse, error) {
// 	config, err := ss.Find(ctx, req.ServiceConfig.Service, req.SourceConfigId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return config.StartSync(ctx, req)
// }

// func (ss *SourceService) StopSync(ctx context.Context, req *replicationv1beta1.StopSyncRequest) (*replicationv1beta1.StopSyncResponse, error) {
// 	config, err := ss.Find(ctx, req.ServiceConfig.Service, req.SourceConfigId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return config.StopSync(ctx, req)
// }
