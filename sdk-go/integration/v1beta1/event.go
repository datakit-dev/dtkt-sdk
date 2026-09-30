package v1beta1

import (
	context "context"
	"fmt"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	eventv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/event/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const EventsPrefix = "events"

var _ RegisteredEvent = (*registeredEvent[any])(nil)

type (
	EventRegistry struct {
		m util.SyncMap[string, RegisteredEvent]
	}
	RegisteredEvent interface {
		WithPayload(any, ...EventOption) (*EventWithPayload, error)
		Proto() *eventv1beta1.Event
	}
	RegisterEventFunc[I InstanceType] func(InstanceMux[I]) (RegisteredEvent, error)
	EventWithPayload                  struct {
		Event   *eventv1beta1.Event
		Payload *anypb.Any
		Action  *sharedv1beta1.ActionType
	}
	EventOption func(*EventWithPayload)

	registeredEvent[P any] struct {
		event         *eventv1beta1.Event
		action        *sharedv1beta1.ActionType
		payloadSchema *TypeSchema[P]
	}
)

func RegisterEvents[I InstanceType](mux InstanceMux[I], newEvents ...RegisterEventFunc[I]) error {
	for _, newEventFunc := range newEvents {
		if newEventFunc != nil {
			event, err := newEventFunc(mux)
			if err != nil {
				return err
			}
			mux.Events().m.Store(event.Proto().GetName(), event)
		}
	}
	return nil
}

func RegisterEvent[I InstanceType, P any, S ~string](
	displayName S,
	description string,
) RegisterEventFunc[I] {
	return func(mux InstanceMux[I]) (RegisteredEvent, error) {
		name := util.ToPascalCase(string(displayName))
		payloadSchema, err := NewTypeSchemaFor[P](mux.Types(), fmt.Sprintf("EventPayload.%s", name))
		if err != nil {
			return nil, err
		}

		return &registeredEvent[P]{
			event: &eventv1beta1.Event{
				Name:          fmt.Sprintf("%s/%s", EventsPrefix, util.Slugify(string(displayName))),
				DisplayName:   string(displayName),
				Description:   description,
				PayloadSchema: payloadSchema.ToProto(),
			},
			payloadSchema: payloadSchema,
		}, nil
	}
}

func RegisterEventWithAction[I InstanceType, P any, S ~string](
	displayName S,
	actionType sharedv1beta1.ActionType,
	description string,
) RegisterEventFunc[I] {
	return func(mux InstanceMux[I]) (RegisteredEvent, error) {
		name := util.ToPascalCase(string(displayName))
		payloadSchema, err := NewTypeSchemaFor[P](mux.Types(), fmt.Sprintf("EventPayload.%s", name))
		if err != nil {
			return nil, err
		}

		return &registeredEvent[P]{
			event: &eventv1beta1.Event{
				Name:          fmt.Sprintf("%s/%s", EventsPrefix, util.Slugify(string(displayName))),
				DisplayName:   string(displayName),
				Description:   description,
				PayloadSchema: payloadSchema.ToProto(),
			},
			action:        &actionType,
			payloadSchema: payloadSchema,
		}, nil
	}
}

func WithEventAction(action sharedv1beta1.ActionType) EventOption {
	return func(event *EventWithPayload) {
		if action > 0 && int(action) < len(eventv1beta1.EventSource_State_value) {
			event.Action = &action
		}
	}
}

func (e *EventRegistry) Find(name string) (RegisteredEvent, error) {
	if strings.HasPrefix(name, EventsPrefix+"/") {
		event, ok := e.m.Load(name)
		if ok {
			return event, nil
		}
	} else {
		_, event, ok := e.m.FindFunc(func(_ string, event RegisteredEvent) bool {
			return name == event.Proto().GetDisplayName()
		})
		if ok {
			return event, nil
		}
	}
	return nil, fmt.Errorf("event not found: %q", name)
}

func (e *registeredEvent[P]) WithPayload(payload any, opts ...EventOption) (*EventWithPayload, error) {
	payloadValid, err := e.payloadSchema.ValidateAny(payload)
	if err != nil {
		return nil, err
	}

	payloadAny, err := common.WrapProtoAny(payloadValid)
	if err != nil {
		return nil, err
	}

	event := &EventWithPayload{
		Event:   e.Proto(),
		Payload: payloadAny,
		Action:  e.action,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(event)
		}
	}

	return event, nil
}

func (e *EventRegistry) List(_ context.Context, req *eventv1beta1.ListEventsRequest) (*eventv1beta1.ListEventsResponse, error) {
	return &eventv1beta1.ListEventsResponse{
		Events: e.Protos(),
	}, nil
}

func (e *EventRegistry) Protos() []*eventv1beta1.Event {
	return util.SliceMap(e.m.Values(), func(e RegisteredEvent) *eventv1beta1.Event {
		return e.Proto()
	})
}

func (e *registeredEvent[P]) Proto() *eventv1beta1.Event {
	return proto.CloneOf(e.event)
}
