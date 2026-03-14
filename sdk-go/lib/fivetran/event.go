package fivetran

import (
	"strings"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"

	replicationv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/replication/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/proto"
)

const (
	AccountWebhookType        WebhookType      = "account"
	GroupWebhookType          WebhookType      = "group"
	CreateConnectorEvent      WebhookEventType = "create_connector"
	PauseConnectorEvent       WebhookEventType = "pause_connector"
	ResumeConnectorEvent      WebhookEventType = "resume_connector"
	EditConnectorEvent        WebhookEventType = "edit_connector"
	DeleteConnectorEvent      WebhookEventType = "delete_connector"
	ForceUpdateConnectorEvent WebhookEventType = "force_update_connector"
	ResyncConnectorEvent      WebhookEventType = "resync_connector"
	ResyncTableEvent          WebhookEventType = "resync_table"
	ConnectionSuccessEvent    WebhookEventType = "connection_successful"
	ConnectionFailedEvent     WebhookEventType = "connection_failure"
	SyncStartEvent            WebhookEventType = "sync_start"
	SyncEndEvent              WebhookEventType = "sync_end"
	DBTRunStartEvent          WebhookEventType = "dbt_run_start"
	DBTRunSuccessEvent        WebhookEventType = "dbt_run_succeeded"
	DBTRunFailedEvent         WebhookEventType = "dbt_run_failed"
	SyncRescheduled           SyncStatus       = "RESCHEDULED"
	SyncSuccess               SyncStatus       = "SUCCESSFUL"
	SyncFailed                SyncStatus       = "FAILED"
	SyncFailedWithTask        SyncStatus       = "FAILURE_WITH_TASK"
)

var (
	webhookEventTypes = WebhookEventTypes{
		CreateConnectorEvent,
		PauseConnectorEvent,
		ResumeConnectorEvent,
		EditConnectorEvent,
		DeleteConnectorEvent,
		ForceUpdateConnectorEvent,
		ResyncConnectorEvent,
		ResyncTableEvent,
		ConnectionSuccessEvent,
		ConnectionFailedEvent,
		SyncStartEvent,
		SyncEndEvent,
		DBTRunStartEvent,
		DBTRunSuccessEvent,
		DBTRunFailedEvent,
	}
)

type (
	WebhookEvent struct {
		Type               WebhookEventType  `json:"event"`
		Created            *time.Time        `json:"created"`              //	The event generation date
		ConnectorType      string            `json:"connector_type"`       //	The type of the connector for which the webhooks is sent
		ConnectorId        string            `json:"connector_id"`         //	The ID of the connector for which the webhooks is sent
		ConnectorName      string            `json:"connector_name"`       //	The name of the connector for which the webhooks is sent
		SyncId             string            `json:"sync_id"`              //	The sync for which the webhooks is sent
		DestinationGroupId string            `json:"destination_group_id"` //	The destination group ID of the connector for which the webhooks is sent
		Data               *WebhookEventData `json:"data,omitempty"`       //	The response payload object. The object fields vary depending on the event type.
	}
	WebhookEventData struct {
		Status        SyncStatus `json:"status"`
		Reason        *string    `json:"reason,omitempty"`
		TaskType      *string    `json:"taskType,omitempty"`
		RescheduledAt *time.Time `json:"rescheduledAt,omitempty"`
	}
	WebhookType       string
	WebhookEventType  string
	WebhookEventTypes []WebhookEventType
	SyncStatus        string
)

func WebhookEvents[I v1beta1.InstanceType]() []v1beta1.RegisterEventFunc[I] {
	return []v1beta1.RegisterEventFunc[I]{
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			CreateConnectorEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_CREATE,
			"Event triggered when a connector is created",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			PauseConnectorEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a connector is paused",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			ResumeConnectorEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a connector is resumed",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			EditConnectorEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a connector is edited",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			DeleteConnectorEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_DELETE,
			"Event triggered when a connector is deleted",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			ForceUpdateConnectorEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a connector is force updated",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			ConnectionSuccessEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a connection is successful",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			ConnectionFailedEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a connection fails",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			ResyncConnectorEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a connector is resynced",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			ResyncTableEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a table is resynced",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			SyncStartEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a sync starts",
		),
		v1beta1.RegisterEventWithAction[I, *replicationv1beta1.Source](
			SyncEndEvent,
			sharedv1beta1.ActionType_ACTION_TYPE_UPDATE,
			"Event triggered when a sync ends",
		),
		// v1beta1.RegisterEventWithAction[I, WebhookEvent](
		// 	sharedv1beta1.ActionType_ACTION_TYPETYPE_MODEL,
		// 	DBTRunStartEvent,_UPDATE,
		// 	sharedv1beta1.ResourceType_RESOURCE_
		// 	"Event triggered when a dbt run starts",
		// ),
		// v1beta1.RegisterEventWithAction[I, WebhookEvent](
		// 	sharedv1beta1.ActionType_ACTION_TYPETYPE_MODEL,
		// 	DBTRunSuccessEvent,_UPDATE,
		// 	sharedv1beta1.ResourceType_RESOURCE_
		// 	"Event triggered when a dbt run succeeds",
		// ),
		// v1beta1.RegisterEventWithAction[I, WebhookEvent](
		// 	sharedv1beta1.ActionType_ACTION_TYPETYPE_MODEL,
		// 	DBTRunFailedEvent,_UPDATE,
		// 	sharedv1beta1.ResourceType_RESOURCE_
		// 	"Event triggered when a dbt run fails",
		// ),
	}
}

func (e WebhookEventType) String() string {
	return string(e)
}

func (e WebhookEventTypes) Strings() []string {
	events := make([]string, len(e))
	for i, event := range e {
		events[i] = string(event)
	}
	return events
}

func (s SyncStatus) ToProto() replicationv1beta1.SyncStatus {
	switch s {
	case SyncRescheduled:
		return replicationv1beta1.SyncStatus_SYNC_STATUS_SCHEDULED
	case SyncSuccess:
		return replicationv1beta1.SyncStatus_SYNC_STATUS_SUCCESS
	case SyncFailed, SyncFailedWithTask:
		return replicationv1beta1.SyncStatus_SYNC_STATUS_FAILED
	}

	return replicationv1beta1.SyncStatus_SYNC_STATUS_UNSPECIFIED
}

func (e *WebhookEvent) SyncStatus() replicationv1beta1.SyncStatus {
	switch e.Type {
	case ConnectionSuccessEvent:
		return replicationv1beta1.SyncStatus_SYNC_STATUS_SCHEDULED
	case SyncStartEvent:
		return replicationv1beta1.SyncStatus_SYNC_STATUS_RUNNING
	case SyncEndEvent:
		if e.Data == nil {
			return replicationv1beta1.SyncStatus_SYNC_STATUS_UNSPECIFIED
		}
		return e.Data.Status.ToProto()
	default:
		return replicationv1beta1.SyncStatus_SYNC_STATUS_UNSPECIFIED
	}
}

func (e *WebhookEvent) SyncError() string {
	if e.Data == nil {
		return ""
	}

	var msgs = []string{}
	if e.Data.TaskType != nil {
		msgs = append(msgs, *e.Data.TaskType)
	}

	if e.Data.Reason != nil {
		msgs = append(msgs, *e.Data.Reason)
	}

	if len(msgs) == 0 {
		return ""
	}

	return strings.Join(msgs, ": ")
}

func (e *WebhookEvent) Payload() proto.Message {
	switch e.Type {
	case CreateConnectorEvent, PauseConnectorEvent, ResumeConnectorEvent,
		EditConnectorEvent, DeleteConnectorEvent, ForceUpdateConnectorEvent,
		ConnectionSuccessEvent, ConnectionFailedEvent, ResyncConnectorEvent,
		ResyncTableEvent, SyncStartEvent, SyncEndEvent:
		return &replicationv1beta1.Source{
			Id:            e.ConnectorId,
			TypeId:        e.ConnectorType,
			DestinationId: e.DestinationGroupId,
			Name:          e.ConnectorName,
			SyncStatus:    e.SyncStatus(),
			SyncError:     e.SyncError(),
		}
	}

	return nil
}
