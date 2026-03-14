package fivetran

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	eventv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/event/v1beta1"
	"github.com/fivetran/go-fivetran/webhooks"
)

var _ v1beta1.PushSourceHandler[*WebhookConfig] = (*WebhookHandler)(nil)

const (
	WebhookEventSourceName = "FivetranWebhook"
	WebhookSignatureHeader = "x-fivetran-signature-256"
)

type (
	WebhookHandler struct {
		client  *Client
		config  *WebhookConfig
		webhook *webhooks.WebhookCommonData
	}
	WebhookConfig struct {
		DestinationType string `json:"destination_type"`
	}
	EventSourceRequest interface {
		GetEventSource() *eventv1beta1.EventSource
	}
)

func WebhookEventSource[I InstanceWithDestination](mux v1beta1.InstanceMux[I]) v1beta1.RegisterSourceFunc[I] {
	return v1beta1.NewPushSource(
		WebhookEventSourceName,
		"Fivetran Webhook delivers replication events for all connections within an account or destination.",
		true, true,
		mux.Address().URL(),
		NewWebhookHandler[I],
	)
}

func NewWebhookHandler[I InstanceWithDestination](ctx context.Context, mux v1beta1.InstanceMux[I], config *WebhookConfig) (*WebhookHandler, error) {
	client, err := GetClientFromInstance(ctx, mux)
	if err != nil {
		return nil, err
	} else if client.creds.WebhookSecret == "" {
		return nil, fmt.Errorf("webhook secret not found in credentials")
	}

	dest, err := GetDestinationFromInstance(ctx, mux, config.DestinationType)
	if err != nil {
		return nil, err
	}

	es, err := mux.EventSources().Find(WebhookEventSourceName)
	if err != nil {
		return nil, err
	}

	webhook, err := upsertWebhook(ctx, client, dest.Details.GroupId, es.Proto().GetPushUrl())
	if err != nil {
		return nil, err
	}

	return &WebhookHandler{
		client:  client,
		config:  config,
		webhook: webhook,
	}, nil
}

func (h *WebhookHandler) PushEvent(ctx context.Context, events *v1beta1.EventRegistry, headers http.Header, body []byte) (*v1beta1.EventWithPayload, error) {
	if headers.Get(WebhookSignatureHeader) == "" {
		return nil, fmt.Errorf("webhook signature header missing in request")
	}

	if !h.validSignature(headers.Get(WebhookSignatureHeader), body) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	var webhookEvent WebhookEvent
	err := encoding.FromJSONV2(body, &webhookEvent)
	if err != nil {
		return nil, err
	}

	event, err := events.Find(webhookEvent.Type.String())
	if err != nil {
		return nil, err
	}

	return event.WithPayload(webhookEvent.Payload())
}

func (h *WebhookHandler) ConfigEqual(config *WebhookConfig) bool {
	return h.config != nil && config != nil && *h.config == *config
}

func (h *WebhookHandler) Close() error {
	return nil
}

func (h *WebhookHandler) validSignature(givenSig string, body []byte) bool {
	hash := hmac.New(sha256.New, []byte(h.client.creds.WebhookSecret))
	_, err := hash.Write(body)
	if err != nil {
		return false
	}
	return strings.EqualFold(givenSig, hex.EncodeToString(hash.Sum(nil)))
}

func getWebhook(ctx context.Context, client *Client, url string) (*webhooks.WebhookCommonData, error) {
	svc := client.NewWebhookList().Limit(1000)
	for {
		resp, err := svc.Do(ctx)
		if err != nil {
			return nil, fmt.Errorf("fivetran list fivetran webhooks: %s", err.Error())
		} else if len(resp.Data.Items) == 0 {
			break
		}

		for _, w := range resp.Data.Items {
			if strings.EqualFold(w.Url, url) {
				return &w, nil
			}
		}

		if resp.Data.NextCursor != "" {
			svc = svc.Cursor(resp.Data.NextCursor)
		} else {
			break
		}
	}

	return nil, fmt.Errorf("fivetran webhook not found")
}

func upsertWebhook(ctx context.Context, client *Client, groupID, url string) (*webhooks.WebhookCommonData, error) {
	webhook, err := getWebhook(ctx, client, url)
	if err != nil || webhook == nil {
		createResp, err := client.NewWebhookGroupCreate().
			GroupId(groupID).
			Events(webhookEventTypes.Strings()).
			Url(url).
			Secret(client.creds.WebhookSecret).
			Active(true).
			Do(ctx)
		if err != nil {
			if createResp.Message == "" {
				createResp.Message = err.Error()
			}
			return nil, fmt.Errorf("error creating webhook: %s", createResp.Message)
		}

		webhook = &createResp.Data.WebhookCommonData
	}

	if webhook != nil {
		update := client.NewWebhookUpdate().
			WebhookId(webhook.Id)

		if !webhook.Active {
			update.Active(true)
		}

		if webhook.Url != url {
			update.Url(url)
		}

		updateResp, err := update.Do(ctx)
		if err != nil {
			if updateResp.Message == "" {
				updateResp.Message = err.Error()
			}
			return nil, fmt.Errorf("error updating webhook: %s", updateResp.Message)
		}
		webhook = &updateResp.Data.WebhookCommonData
	}

	return webhook, nil
}

// func eventSourceFromRequest(req EventSourceRequest) (*eventv1beta1.EventSource, error) {
// 	if req == nil || req.GetEventSource() == nil {
// 		return nil, fmt.Errorf("event source not found in request: %v", req)
// 	} else if req.GetEventSource().GetDisplayName() != WebhookEventSourceName {
// 		return nil, fmt.Errorf("event source invalid, expected: %s, got: %s", WebhookEventSourceName, req.GetEventSource().GetDisplayName())
// 	}

// 	return req.GetEventSource(), nil
// }

// func validateEventSource(eventSource *eventv1beta1.EventSource) (*WebhookConfig, error) {
// 	if eventSource == nil {
// 		return nil, fmt.Errorf("event source is required")
// 	} else if eventSource.GetPushUrl() == "" {
// 		return nil, fmt.Errorf("event source invalid: %q is required", "push_url")
// 	} else if eventSource.GetConfig() == nil || !eventSource.GetConfig().MessageName().IsValid() {
// 		return nil, fmt.Errorf("event source config is invalid")
// 	}

// 	config, err := common.UnwrapProtoAnyAs[*WebhookConfig](eventSource.GetConfig())
// 	if err != nil {
// 		return nil, err
// 	}

// 	return config, nil
// }

// func (s *DestinationService[I]) CreateEventSource(ctx context.Context, req *eventv1beta1.CreateEventSourceRequest) (*eventv1beta1.CreateEventSourceResponse, error) {
// 	// inst, err := s.mux.GetInstance(ctx)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// dest, err := inst.FivetranDestination()
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// client, err := NewClient(dest.Client)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// eventSource, err := eventSourceFromRequest(req)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// config, err := validateEventSource(eventSource)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// group, err := client.NewGroupDetails().GroupID(config.DestinationID).Do(ctx)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// webhook, err := upsertWebhook(ctx, client, group.Data.ID, eventSource.GetPushUrl())
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// dest.webhook = webhook

// 	return &eventv1beta1.CreateEventSourceResponse{
// 		// EventSource: eventSource,
// 	}, nil
// }

// func (s *DestinationService[I]) UpdateEventSource(ctx context.Context, req *eventv1beta1.UpdateEventSourceRequest) (*eventv1beta1.UpdateEventSourceResponse, error) {
// 	// inst, err := s.mux.GetInstance(ctx)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// dest, err := inst.FivetranDestination()
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// client, err := NewClient(dest.Client)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// eventSource, err := eventSourceFromRequest(req)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// config, err := validateEventSource(eventSource)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// group, err := client.NewGroupDetails().GroupID(config.DestinationID).Do(ctx)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// webhook, err := upsertWebhook(ctx, client, group.Data.ID, eventSource.GetPushUrl())
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// dest.webhook = webhook

// 	return &eventv1beta1.UpdateEventSourceResponse{
// 		// EventSource: eventSource,
// 	}, nil
// }

// func (s *DestinationService[I]) DeleteEventSource(ctx context.Context, req *eventv1beta1.DeleteEventSourceRequest) (*eventv1beta1.DeleteEventSourceResponse, error) {
// 	// inst, err := s.mux.GetInstance(ctx)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// dest, err := inst.FivetranDestination()
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// client, err := NewClient(dest.Client)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// eventSource, err := eventSourceFromRequest(req)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// webhook, err := getWebhook(ctx, client, eventSource.PushUrl)
// 	// if err != nil {
// 	// 	return nil, status.Error(codes.FailedPrecondition, err.Error())
// 	// }

// 	// deleteResp, err := client.NewWebhookDelete().WebhookId(webhook.Id).Do(ctx)
// 	// if err != nil {
// 	// 	if deleteResp.Message == "" {
// 	// 		deleteResp.Message = err.Error()
// 	// 	}

// 	// 	return nil, status.Error(codes.FailedPrecondition, deleteResp.Message)
// 	// }

// 	return &eventv1beta1.DeleteEventSourceResponse{}, nil
// }
