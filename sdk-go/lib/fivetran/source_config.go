package fivetran

import (
	"context"
	"fmt"

	sdkcommon "github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	replicationv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/replication/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/fivetran/go-fivetran/common"
	connectors "github.com/fivetran/go-fivetran/connections"
)

var (
// sourceConfigs []*replicationv1beta1.SourceConfig
// _ SourceConfig = (*sourceConfig[any])(nil)
)

type (
	SourceConfig interface {
		// ReplicationService() replicationv1beta1.ServiceType
		GetID() string
		ToProto(context.Context) (*replicationv1beta1.SourceType, error)
		SetConnectorAuth(ConnectorAuthFunc)
		SetConnectorClientAccess(ConnectorClientAccessFunc)
		SetConnectorConfig(ConnectorConfigFunc)
		SetAuthConfig(*sharedv1beta1.OAuthConfig)
		SetRefreshToken(OAuthTokenFunc)
		SetConfigData(ConfigDataFunc)
		CheckSourceAuth(context.Context, *replicationv1beta1.CheckSourceAuthRequest) (*replicationv1beta1.CheckSourceAuthResponse, error)
		GetSource(context.Context, *replicationv1beta1.GetSourceRequest) (*replicationv1beta1.GetSourceResponse, error)
		CreateSource(context.Context, *replicationv1beta1.CreateSourceRequest) (*replicationv1beta1.CreateSourceResponse, error)
		UpdateSource(context.Context, *replicationv1beta1.UpdateSourceRequest) (*replicationv1beta1.UpdateSourceResponse, error)
		DeleteSource(context.Context, *replicationv1beta1.DeleteSourceRequest) (*replicationv1beta1.DeleteSourceResponse, error)
		StartSync(context.Context, *replicationv1beta1.StartSyncRequest) (*replicationv1beta1.StartSyncResponse, error)
		StopSync(context.Context, *replicationv1beta1.StopSyncRequest) (*replicationv1beta1.StopSyncResponse, error)
	}
	SourceConfigFunc          func(context.Context) (SourceConfig, error)
	OAuthTokenFunc            func(context.Context) (*oauth2.Token, error)
	ConfigDataFunc            func(context.Context) (*structpb.Struct, error)
	ConnectorAuthFunc         func(context.Context, *connectors.ConnectionAuth) error
	ConnectorClientAccessFunc func(context.Context, *connectors.ConnectionAuthClientAccess) error
	ConnectorConfigFunc       func(context.Context, *connectors.ConnectionConfig) error
	sourceConfig[T any]       struct {
		id string
		// client       *Client
		configSchema *sdkcommon.JSONSchema[T]
		// authConfig   *sharedv1beta1.OAuthConfig
		// connAuth     ConnectorAuthFunc
		// connConfig   ConnectorConfigFunc
		// clientAccess ConnectorClientAccessFunc
		// refreshToken OAuthTokenFunc
		// configData   ConfigDataFunc
	}
	ConnectorType struct {
		// Id The connector type identifier within the Fivetran system
		Id string `json:"id"`
		// Name The connector service name within the Fivetran system
		Name string `json:"name"`
		// Description The description characterizing the purpose of the connector
		Description *string `json:"description,omitempty"`
		// IconUrl The icon resource URL
		IconUrl *string `json:"icon_url,omitempty"`
		// Type The connector service type
		Type string `json:"type"`
	}
	ConnectorTypesResponse struct {
		common.CommonResponse
		Data []*ConnectorType `json:"data"`
	}
)

func NewSourceConfig[T any](id string) (*sourceConfig[T], error) {
	schema, err := sdkcommon.JSONSchemaFor[T](
		sdkcommon.WithSchemaID(
			fmt.Sprintf("source/%s/config", id),
		),
	)
	if err != nil {
		return nil, err
	}

	return &sourceConfig[T]{
		id:           id,
		configSchema: schema,
	}, nil
}

// func ListSourceConfigs() ([]*replicationv1beta1.SourceConfig, error) {
// 	if sourceConfigs == nil {
// 		resp, err := http.Get("https://api.fivetran.com/public/connector-types")
// 		if err != nil {
// 			return nil, err
// 		}

// 		body, err := io.ReadAll(resp.Body)
// 		if err != nil {
// 			return nil, err
// 		}
// 		defer resp.Body.Close()

// 		var connTypesResp ConnectorTypesResponse
// 		if err := json.Unmarshal(body, &connTypesResp); err != nil {
// 			return nil, err
// 		}

// 		for connType := range slices.Values(connTypesResp.Data) {
// 			var (
// 				desc    = fmt.Sprintf("%s source type from Fivetran.", connType.Name)
// 				iconUrl string
// 			)
// 			if connType.Description != nil {
// 				desc = *connType.Description
// 			}

// 			if connType.IconUrl != nil {
// 				iconUrl = *connType.IconUrl
// 			} else {
// 				fmt.Printf("icon url missing for source config: %s\n", connType.Id)
// 				continue
// 			}

// 			sourceConfigs = append(sourceConfigs, &replicationv1beta1.SourceConfig{
// 				// Service:     replicationv1beta1.ServiceType_SERVICE_TYPE_FIVETRAN,
// 				Id:          connType.Id,
// 				Name:        connType.Name,
// 				Description: desc,
// 				IconUrl:     iconUrl,
// 				Category:    string(connType.Type),
// 			})
// 		}
// 	}

// 	return sourceConfigs, nil
// }

// func GetSourceConfig(id string) (*replicationv1beta1.SourceConfig, error) {
// 	srcConfigs, err := ListSourceConfigs()
// 	if err != nil {
// 		return nil, err
// 	}

// 	var srcConfig *replicationv1beta1.SourceConfig
// 	for srcConfig = range slices.Values(srcConfigs) {
// 		if srcConfig.Id == id {
// 			break
// 		}
// 	}

// 	if srcConfig == nil {
// 		return nil, fmt.Errorf("source config not found: %s", id)
// 	}

// 	return srcConfig, nil
// }

// func (c *sourceConfig[T]) GetID() string {
// 	return c.id
// }

// func (c *sourceConfig[T]) SetAuthConfig(config *sharedv1beta1.OAuthConfig) {
// 	c.authConfig = config
// }

// func (c *sourceConfig[T]) SetRefreshToken(f OAuthTokenFunc) {
// 	c.refreshToken = f
// }

// func (c *sourceConfig[T]) SetConfigData(f ConfigDataFunc) {
// 	c.configData = f
// }

// // func (c *sourceConfig[T]) ReplicationService() replicationv1beta1.ServiceType {
// // 	return replicationv1beta1.ServiceType_SERVICE_TYPE_FIVETRAN
// // }

// func (c *sourceConfig[T]) SetConnectorAuth(f ConnectorAuthFunc) {
// 	c.connAuth = f
// }

// func (c *sourceConfig[T]) SetConnectorClientAccess(f ConnectorClientAccessFunc) {
// 	c.clientAccess = f
// }

// func (c *sourceConfig[T]) SetConnectorConfig(f ConnectorConfigFunc) {
// 	c.connConfig = f
// }

// func (c *sourceConfig[T]) getClient(config *fivetranv1beta1.ClientConfig) (*Client, error) {
// 	if config == nil {
// 		return nil, fmt.Errorf("fivetran replication config required")
// 	}

// 	if c.client == nil {
// 		client, err := NewClient(config)
// 		if err != nil {
// 			return nil, err
// 		}
// 		c.client = client
// 	}
// 	return c.client, nil
// }

// func (c *sourceConfig[T]) connectorAuth(ctx context.Context, authFunc ConnectorAuthFunc, clientAccessFunc ConnectorClientAccessFunc) (*connectors.ConnectionAuth, error) {
// 	var auth = &connectors.ConnectionAuth{}

// 	if authFunc != nil {
// 		if err := authFunc(ctx, auth); err != nil {
// 			return nil, err
// 		}
// 	}

// 	if clientAccessFunc != nil {
// 		var clientAccess = &connectors.ConnectionAuthClientAccess{}
// 		if err := clientAccessFunc(ctx, clientAccess); err != nil {
// 			return nil, err
// 		}
// 		auth.ClientAccess(clientAccess)
// 	}

// 	return auth, nil
// }

// func (c *sourceConfig[T]) connectorConfig(ctx context.Context, configFunc ConnectorConfigFunc) (*connectors.ConnectionConfig, error) {
// 	var connConfig = &connectors.ConnectionConfig{}
// 	if configFunc != nil {
// 		if err := configFunc(ctx, connConfig); err != nil {
// 			return nil, err
// 		}
// 	}
// 	return connConfig, nil
// }

// func (c *sourceConfig[T]) Validate(repConfig *replicationv1beta1.ServiceConfig, checkSrcConfig bool) error {
// 	if repConfig == nil || repConfig.Service != replicationv1beta1.ServiceType_SERVICE_TYPE_FIVETRAN {
// 		return fmt.Errorf("fivetran replication config invalid")
// 	} else if repConfig.GetFivetranConfig() == nil {
// 		return fmt.Errorf("fivetran replication config required")
// 	} else if checkSrcConfig && repConfig.GetFivetranConfig().GetSourceConfig() == nil {
// 		return fmt.Errorf("fivetran replication source config required")
// 	}

// 	return nil
// }

// func (c *sourceConfig[T]) AuthCustom(ctx context.Context, authFunc ConnectorAuthFunc, clientAccessFunc ConnectorClientAccessFunc) (*map[string]any, error) {
// 	var authMap = map[string]any{}
// 	if auth, err := c.connectorAuth(ctx, authFunc, clientAccessFunc); err != nil {
// 		return nil, err
// 	} else {
// 		return auth.Merge(&authMap)
// 	}
// }

// func (c *sourceConfig[T]) ConfigCustom(ctx context.Context, configFunc ConnectorConfigFunc, configMap map[string]any) (*map[string]any, error) {
// 	configStr, err := sdkcommon.MarshalJSON[string](configMap)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if _, err := c.configSchema.ValidateString(configStr); err != nil {
// 		return nil, err
// 	}

// 	if config, err := c.connectorConfig(ctx, configFunc); err != nil {
// 		return nil, err
// 	} else {
// 		return config.Merge(&configMap)
// 	}
// }

// func (c *sourceConfig[T]) ToProto(context.Context) (*replicationv1beta1.SourceConfig, error) {
// 	config, err := GetSourceConfig(c.id)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// config.ConfigSchema = c.configSchema.String()

// 	return config, nil

// }

// func (c *sourceConfig[T]) CheckSourceAuth(ctx context.Context, req *replicationv1beta1.CheckSourceAuthRequest) (*replicationv1beta1.CheckSourceAuthResponse, error) {
// 	if c.authConfig == nil {
// 		return nil, fmt.Errorf("source config missing oAuthConfig: %s", req.ConfigId)
// 	}

// 	var (
// 		resp = &replicationv1beta1.CheckSourceAuthResponse{
// 			Type: sharedv1beta1.AuthType_AUTH_TYPE_OAUTH,
// 		}
// 	)

// 	switch req.Type {
// 	case sharedv1beta1.AuthCheck_AUTH_CHECK_OAUTH_CODE:
// 		var (
// 			codeReq      = req.GetCodeRequest()
// 			config, opts = middleware.OAuthConfigFromProto(c.authConfig)
// 		)

// 		config.RedirectURL = codeReq.RedirectUrl
// 		resp.Oauth = &replicationv1beta1.CheckSourceAuthResponse_AuthCodeUrl{
// 			AuthCodeUrl: config.AuthCodeURL(codeReq.State, opts...),
// 		}
// 		resp.Success = true

// 		return resp, nil
// 	case sharedv1beta1.AuthCheck_AUTH_CHECK_OAUTH_CALLBACK:
// 		if req.GetTokenRequest() == nil {
// 			resp.Success = false
// 			resp.Error = fmt.Sprintf("check source auth %s: missing token request", sharedv1beta1.AuthCheck_AUTH_CHECK_OAUTH_CALLBACK)
// 			return resp, nil
// 		}

// 		var config, opts = middleware.OAuthConfigFromProto(c.authConfig)
// 		config.RedirectURL = req.GetTokenRequest().RedirectUrl

// 		token, err := config.Exchange(ctx, req.GetTokenRequest().Code, opts...)
// 		if err != nil {
// 			resp.Success = false
// 			resp.Error = err.Error()
// 			return resp, nil
// 		}

// 		resp.Oauth = &replicationv1beta1.CheckSourceAuthResponse_Token{
// 			Token: middleware.OAuthTokenToProto(token),
// 		}
// 		resp.Success = true

// 		return resp, nil
// 	case sharedv1beta1.AuthCheck_AUTH_CHECK_OAUTH_REFRESH:
// 		if req.GetRefreshRequest() == nil {
// 			resp.Success = false
// 			resp.Error = fmt.Sprintf("check source auth %s: missing token request", sharedv1beta1.AuthCheck_AUTH_CHECK_OAUTH_REFRESH)
// 			return resp, nil
// 		}

// 		if c.refreshToken == nil {
// 			return nil, fmt.Errorf("refresh oauth token function not set")
// 		}

// 		token, err := c.refreshToken(ctx)
// 		if err != nil {
// 			resp.Success = false
// 			resp.Error = err.Error()
// 			return resp, nil
// 		}

// 		resp.Oauth = &replicationv1beta1.CheckSourceAuthResponse_Token{
// 			Token: middleware.OAuthTokenToProto(token),
// 		}
// 		resp.Success = true

// 		return resp, nil
// 	}

// 	if c.configData != nil {
// 		configData, err := c.configData(ctx)
// 		if err != nil {
// 			resp.Success = false
// 			resp.Error = err.Error()
// 			return resp, nil
// 		} else {
// 			resp.ConfigData = configData
// 		}
// 	}

// 	resp.Success = true
// 	return resp, nil
// }

// func (c *sourceConfig[T]) GetSource(ctx context.Context, req *replicationv1beta1.GetSourceRequest) (*replicationv1beta1.GetSourceResponse, error) {
// 	if err := c.Validate(req.ServiceConfig, false); err != nil {
// 		return nil, err
// 	}

// 	client, err := c.getClient(req.ServiceConfig.GetFivetranConfig())
// 	if err != nil {
// 		return nil, err
// 	}

// 	getResp, err := client.NewConnectionDetails().ConnectionID(req.Id).DoCustom(ctx)
// 	if err != nil {
// 		if getResp.Message != "" {
// 			return nil, fmt.Errorf("get source error: %s", getResp.Message)
// 		}
// 		return nil, err
// 	}

// 	var source = &replicationv1beta1.Source{
// 		Service:       replicationv1beta1.ServiceType_SERVICE_TYPE_FIVETRAN,
// 		DestinationId: getResp.Data.GroupID,
// 		ConfigId:      getResp.Data.Service,
// 		Id:            getResp.Data.ID,
// 		Name:          getResp.Data.Schema,
// 		SyncStatus:    SyncStatus(getResp.Data.Status.SyncState).ToProto(),
// 	}

// 	source.Config, err = structpb.NewStruct(getResp.Data.Config)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &replicationv1beta1.GetSourceResponse{
// 		Source: source,
// 	}, nil
// }

// func (c *sourceConfig[T]) CreateSource(ctx context.Context, req *replicationv1beta1.CreateSourceRequest) (*replicationv1beta1.CreateSourceResponse, error) {
// 	if err := c.Validate(req.ServiceConfig, true); err != nil {
// 		return nil, err
// 	}

// 	auth, err := c.AuthCustom(ctx, c.connAuth, c.clientAccess)
// 	if err != nil {
// 		return nil, err
// 	}

// 	config, err := c.ConfigCustom(ctx, func(ctx context.Context, cc *connectors.ConnectionConfig) error {
// 		cc.Schema(req.Name)
// 		if c.connConfig != nil {
// 			return c.connConfig(ctx, cc)
// 		}
// 		return nil
// 	}, req.Config.AsMap())
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := c.getClient(req.ServiceConfig.GetFivetranConfig())
// 	if err != nil {
// 		return nil, err
// 	}

// 	var (
// 		srcConfig = req.ServiceConfig.GetFivetranConfig().GetSourceConfig()
// 		create    = client.NewConnectionCreate().
// 				GroupID(srcConfig.GroupId).
// 				Service(c.id).
// 				AuthCustom(auth).
// 				ConfigCustom(config).
// 				Paused(srcConfig.Paused).
// 				PauseAfterTrial(srcConfig.PauseAfterTrial).
// 				RunSetupTests(srcConfig.RunSetupTests)
// 		source = &replicationv1beta1.Source{
// 			Service: replicationv1beta1.ServiceType_SERVICE_TYPE_FIVETRAN,
// 		}
// 	)

// 	if srcConfig.ScheduleType != "manual" {
// 		var syncFreq = int(srcConfig.SyncFrequency)
// 		create.SyncFrequency(&syncFreq)

// 		if syncFreq == 1440 && srcConfig.DailySyncTime != "" {
// 			create.DailySyncTime(srcConfig.DailySyncTime)
// 		}
// 	}

// 	createResp, err := create.DoCustom(ctx)
// 	if err != nil {
// 		if createResp.Message == "" {
// 			createResp.Message = err.Error()
// 		}
// 		return &replicationv1beta1.CreateSourceResponse{
// 			Created: false,
// 			Error:   fmt.Sprintf("error creating source: %s", createResp.Message),
// 		}, nil
// 	}

// 	source.DestinationId = createResp.Data.GroupID
// 	source.ConfigId = createResp.Data.Service
// 	source.Id = createResp.Data.ID
// 	source.Name = createResp.Data.Schema
// 	source.SyncStatus = SyncStatus(createResp.Data.Status.SyncState).ToProto()

// 	if createResp.Data.Config != nil {
// 		source.Config, err = structpb.NewStruct(createResp.Data.Config)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	return &replicationv1beta1.CreateSourceResponse{
// 		Created: true,
// 		Source:  source,
// 	}, nil
// }

// func (c *sourceConfig[T]) UpdateSource(ctx context.Context, req *replicationv1beta1.UpdateSourceRequest) (*replicationv1beta1.UpdateSourceResponse, error) {
// 	if err := c.Validate(req.ServiceConfig, true); err != nil {
// 		return nil, err
// 	}

// 	auth, err := c.AuthCustom(ctx, c.connAuth, c.clientAccess)
// 	if err != nil {
// 		return nil, err
// 	}

// 	config, err := c.ConfigCustom(ctx, c.connConfig, req.Config.AsMap())
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := c.getClient(req.ServiceConfig.GetFivetranConfig())
// 	if err != nil {
// 		return nil, err
// 	}

// 	var (
// 		srcConfig = req.ServiceConfig.GetFivetranConfig().GetSourceConfig()
// 		update    = client.NewConnectionUpdate().
// 				ConnectionID(req.Id).
// 				AuthCustom(auth).
// 				ConfigCustom(config).
// 				Paused(srcConfig.Paused).
// 				PauseAfterTrial(srcConfig.PauseAfterTrial).
// 				RunSetupTests(srcConfig.RunSetupTests).
// 				ScheduleType(srcConfig.ScheduleType)
// 		source = &replicationv1beta1.Source{
// 			Service: replicationv1beta1.ServiceType_SERVICE_TYPE_FIVETRAN,
// 		}
// 	)

// 	if srcConfig.ScheduleType != "manual" {
// 		var syncFreq = int(srcConfig.SyncFrequency)
// 		update.SyncFrequency(&syncFreq)

// 		if syncFreq == 1440 && srcConfig.DailySyncTime != "" {
// 			update.DailySyncTime(srcConfig.DailySyncTime)
// 		}
// 	}

// 	updateResp, err := update.DoCustom(ctx)
// 	if err != nil {
// 		if updateResp.Message == "" {
// 			updateResp.Message = err.Error()
// 		}
// 		return &replicationv1beta1.UpdateSourceResponse{
// 			Updated: false,
// 			Error:   fmt.Sprintf("error updating source: %s", updateResp.Message),
// 		}, nil
// 	}

// 	source.DestinationId = updateResp.Data.GroupID
// 	source.ConfigId = updateResp.Data.Service
// 	source.Id = updateResp.Data.ID
// 	source.Name = updateResp.Data.Schema
// 	source.SyncStatus = SyncStatus(updateResp.Data.Status.SyncState).ToProto()

// 	if updateResp.Data.Config != nil {
// 		source.Config, err = structpb.NewStruct(updateResp.Data.Config)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	return &replicationv1beta1.UpdateSourceResponse{
// 		Updated: true,
// 		Source:  source,
// 	}, nil
// }

// func (c *sourceConfig[T]) DeleteSource(ctx context.Context, req *replicationv1beta1.DeleteSourceRequest) (*replicationv1beta1.DeleteSourceResponse, error) {
// 	if err := c.Validate(req.ServiceConfig, false); err != nil {
// 		return nil, err
// 	}

// 	client, err := c.getClient(req.ServiceConfig.GetFivetranConfig())
// 	if err != nil {
// 		return nil, err
// 	}

// 	delResp, err := client.NewConnectionDelete().ConnectionID(req.Id).Do(ctx)
// 	if err != nil {
// 		if delResp.Message == "" {
// 			delResp.Message = err.Error()
// 		}
// 		return &replicationv1beta1.DeleteSourceResponse{
// 			Deleted: false,
// 			Error:   fmt.Sprintf("error deleting source: %s", delResp.Message),
// 		}, nil
// 	}

// 	return &replicationv1beta1.DeleteSourceResponse{
// 		Deleted: true,
// 	}, nil
// }

// func (c *sourceConfig[T]) StartSync(ctx context.Context, req *replicationv1beta1.StartSyncRequest) (*replicationv1beta1.StartSyncResponse, error) {
// 	if err := c.Validate(req.ServiceConfig, false); err != nil {
// 		return nil, err
// 	}

// 	client, err := c.getClient(req.ServiceConfig.GetFivetranConfig())
// 	if err != nil {
// 		return nil, err
// 	}

// 	getResp, err := c.GetSource(ctx, &replicationv1beta1.GetSourceRequest{
// 		ServiceConfig: req.ServiceConfig,
// 		ConfigId:      req.SourceConfigId,
// 		Id:            req.SourceId,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	var source = getResp.Source

// 	syncResp, err := client.NewConnectionSync().ConnectionID(req.SourceId).Do(ctx)
// 	if err != nil {
// 		if syncResp.Message == "" {
// 			syncResp.Message = err.Error()
// 		}
// 		source.SyncStatus = replicationv1beta1.SyncStatus_SYNC_STATUS_FAILED
// 		source.SyncError = syncResp.Message

// 		return &replicationv1beta1.StartSyncResponse{
// 			Source: source,
// 		}, nil
// 	}

// 	source.SyncStatus = replicationv1beta1.SyncStatus_SYNC_STATUS_SCHEDULED

// 	return &replicationv1beta1.StartSyncResponse{
// 		Source: source,
// 	}, nil
// }

// func (c *sourceConfig[T]) StopSync(ctx context.Context, req *replicationv1beta1.StopSyncRequest) (*replicationv1beta1.StopSyncResponse, error) {
// 	if err := c.Validate(req.ServiceConfig, true); err != nil {
// 		return nil, err
// 	}

// 	client, err := c.getClient(req.ServiceConfig.GetFivetranConfig())
// 	if err != nil {
// 		return nil, err
// 	}

// 	getResp, err := c.GetSource(ctx, &replicationv1beta1.GetSourceRequest{
// 		ServiceConfig: req.ServiceConfig,
// 		ConfigId:      req.SourceConfigId,
// 		Id:            req.SourceId,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	var (
// 		update = client.NewConnectionUpdate().
// 			ConnectionID(req.SourceId).
// 			Paused(true)
// 		source = getResp.Source
// 	)

// 	updateResp, err := update.Do(ctx)
// 	if err != nil {
// 		if updateResp.Message == "" {
// 			updateResp.Message = err.Error()
// 		}
// 		source.SyncStatus = replicationv1beta1.SyncStatus_SYNC_STATUS_FAILED
// 		source.SyncError = updateResp.Message

// 		return &replicationv1beta1.StopSyncResponse{
// 			Source: source,
// 		}, nil
// 	}

// 	source.SyncStatus = replicationv1beta1.SyncStatus_SYNC_STATUS_SUCCESS

// 	return &replicationv1beta1.StopSyncResponse{
// 		Source: source,
// 	}, nil
// }
