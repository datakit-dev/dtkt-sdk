package v1beta1

import (
	context "context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	eventv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/event/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

const EventSourcesPrefix = "event_sources"

var _ RegisteredPullSource = (*pullSource[any, PullSourceHandler[any]])(nil)
var _ RegisteredPushSource = (*pushSource[any, PushSourceHandler[any]])(nil)

type (
	EventSourceRegistry struct {
		m util.SyncMap[string, RegisteredSource]
	}
	RegisteredSource interface {
		SetConfig(*anypb.Any) error
		Proto() *eventv1beta1.EventSource
		Close() error
	}
	RegisteredPullSource interface {
		RegisteredSource
		HandlePullRequest(context.Context, *eventv1beta1.StreamPullEventsRequest) (*EventWithPayload, error)
		HandlePullStream(*eventv1beta1.StreamPullEventsRequest, grpc.ServerStreamingServer[eventv1beta1.StreamPullEventsResponse]) error
	}
	RegisteredPushSource interface {
		RegisteredSource
		HandlePushRequest(context.Context, *eventv1beta1.StreamPushEventsRequest) (*EventWithPayload, error)
		HandlePushStream(*eventv1beta1.StreamPushEventsRequest, grpc.BidiStreamingServer[eventv1beta1.StreamPushEventsRequest, eventv1beta1.StreamPushEventsResponse]) error
	}
	PullSourceHandler[C any] interface {
		PullEvent(context.Context, *EventRegistry) (*EventWithPayload, error)
		ConfigEqual(C) bool
		Close() error
	}
	PushSourceHandler[C any] interface {
		PushEvent(context.Context, *EventRegistry, http.Header, []byte) (*EventWithPayload, error)
		ConfigEqual(C) bool
		Close() error
	}
	RegisterSourceFunc[I InstanceType]                                   func(InstanceMux[I]) (RegisteredSource, error)
	PullSourceHandlerFunc[I InstanceType, C any, S PullSourceHandler[C]] func(context.Context, InstanceMux[I], C) (S, error)
	PushSourceHandlerFunc[I InstanceType, C any, S PushSourceHandler[C]] func(context.Context, InstanceMux[I], C) (S, error)
	pullSource[C any, S PullSourceHandler[C]]                            struct {
		configSchema *TypeSchema[C]
		events       *EventRegistry
		eventSource  *eventv1beta1.EventSource

		newHandler func(context.Context, C) (S, error)
		handlers   util.SyncMap[uuid.UUID, PullSourceHandler[C]]
		mut        sync.Mutex
	}
	pushSource[C any, S PushSourceHandler[C]] struct {
		configSchema *TypeSchema[C]
		events       *EventRegistry
		eventSource  *eventv1beta1.EventSource

		newHandler func(context.Context, C) (S, error)
		handlers   util.SyncMap[uuid.UUID, PushSourceHandler[C]]
		mut        sync.Mutex
	}
)

func RegisterSources[I InstanceType](mux InstanceMux[I], regSources ...RegisterSourceFunc[I]) error {
	for _, regSource := range regSources {
		source, err := regSource(mux)
		if err != nil {
			return err
		}
		mux.EventSources().m.Store(source.Proto().GetName(), source)
	}
	return nil
}

func NewPullSource[I InstanceType, C any, S PullSourceHandler[C]](
	displayName, description string,
	requiresConfig, continueOnError bool,
	pullFreq time.Duration,
	handlerFunc PullSourceHandlerFunc[I, C, S],
) RegisterSourceFunc[I] {
	return func(mux InstanceMux[I]) (RegisteredSource, error) {
		name := util.ToPascalCase(displayName)
		configSchema, err := NewTypeSchemaFor[C](mux.Types(), fmt.Sprintf("EventSourceConfig.%s", name))
		if err != nil {
			return nil, err
		}

		return &pullSource[C, S]{
			configSchema: configSchema,
			events:       mux.Events(),
			newHandler: func(ctx context.Context, c C) (S, error) {
				return handlerFunc(ctx, mux, c)
			},
			eventSource: &eventv1beta1.EventSource{
				Name:            fmt.Sprintf("%s/%s", EventSourcesPrefix, util.Slugify(string(displayName))),
				DisplayName:     string(displayName),
				Description:     description,
				ConfigSchema:    configSchema.ToProto(),
				RequiresConfig:  requiresConfig,
				ContinueOnError: continueOnError,
				Strategy: &eventv1beta1.EventSource_PullFreq{
					PullFreq: durationpb.New(pullFreq),
				},
			},
		}, nil
	}
}

func NewPushSource[I InstanceType, C any, S PushSourceHandler[C]](
	displayName, description string,
	requiresConfig, continueOnError bool,
	baseUrl *url.URL,
	handlerFunc PushSourceHandlerFunc[I, C, S],
) RegisterSourceFunc[I] {
	return func(mux InstanceMux[I]) (RegisteredSource, error) {
		name := util.ToPascalCase(displayName)
		configSchema, err := NewTypeSchemaFor[C](mux.Types(), fmt.Sprintf("EventSourceConfig.%s", name))
		if err != nil {
			return nil, err
		}

		name = fmt.Sprintf("%s/%s", EventSourcesPrefix, util.Slugify(string(displayName)))
		return &pushSource[C, S]{
			configSchema: configSchema,
			events:       mux.Events(),
			eventSource: &eventv1beta1.EventSource{
				Name:            name,
				DisplayName:     string(displayName),
				Description:     description,
				ConfigSchema:    configSchema.ToProto(),
				RequiresConfig:  requiresConfig,
				ContinueOnError: continueOnError,
				Strategy: &eventv1beta1.EventSource_PushUrl{
					PushUrl: baseUrl.JoinPath(name).String(),
				},
			},
			newHandler: func(ctx context.Context, c C) (S, error) {
				return handlerFunc(ctx, mux, c)
			},
		}, nil
	}
}

func (r *EventSourceRegistry) Find(name string) (RegisteredSource, error) {
	if strings.HasPrefix(name, EventSourcesPrefix+"/") {
		source, ok := r.m.Load(name)
		if ok {
			return source, nil
		}
	} else {
		_, source, ok := r.m.FindFunc(func(_ string, event RegisteredSource) bool {
			return name == event.Proto().GetDisplayName()
		})
		if ok {
			return source, nil
		}
	}
	return nil, fmt.Errorf("event source not found: %q", name)
}

func (r *EventSourceRegistry) Range(f func(string, RegisteredSource) bool) {
	r.m.Range(f)
}

func (r *EventSourceRegistry) List(_ context.Context, req *eventv1beta1.ListEventSourcesRequest) (*eventv1beta1.ListEventSourcesResponse, error) {
	// var sources []*eventv1beta1.EventSource
	// res.m.Range(func(name string, source RegisteredSource) bool {
	// 	// if req.Strategy == eventv1beta1.EventSourceStrategy_EVENT_SOURCE_STRATEGY_UNSPECIFIED || req.Strategy == source.Proto().GetStrategy() {
	// 	// 	sources = append(sources, source.Proto())
	// 	// }
	// 	return true
	// })
	return &eventv1beta1.ListEventSourcesResponse{
		EventSources: r.Protos(),
	}, nil
}

func (r *EventSourceRegistry) SetConfig(name string, config *anypb.Any) error {
	source, err := r.Find(name)
	if err != nil {
		return err
	}
	return source.SetConfig(config)
}

func (s *pullSource[C, S]) SetConfig(configAny *anypb.Any) error {
	config, err := common.UnwrapProtoAnyAs[C](configAny)
	if err != nil {
		return fmt.Errorf("pull stream %q config unmarshal error: %w", s.eventSource.GetName(), err)
	}

	err = s.configSchema.Validate(config)
	if err != nil {
		return fmt.Errorf("pull stream %q config validation error: %w", s.eventSource.GetName(), err)
	}

	s.eventSource.Config = configAny

	return nil
}

func (s *pushSource[C, S]) SetConfig(configAny *anypb.Any) error {
	config, err := common.UnwrapProtoAnyAs[C](configAny)
	if err != nil {
		return fmt.Errorf("push stream %q config unmarshal error: %w", s.eventSource.GetName(), err)
	}

	err = s.configSchema.Validate(config)
	if err != nil {
		return fmt.Errorf("push stream %q config validation error: %w", s.eventSource.GetName(), err)
	}

	s.eventSource.Config = configAny

	return nil
}

func (r *EventSourceRegistry) HandlePullStream(req *eventv1beta1.StreamPullEventsRequest, stream grpc.ServerStreamingServer[eventv1beta1.StreamPullEventsResponse]) error {
	if req == nil {
		return fmt.Errorf("pull stream events request: request cannot be nil")
	} else if req.GetName() == "" {
		return fmt.Errorf("pull stream events request: name required")
	}

	source, err := r.Find(req.GetName())
	if err != nil {
		return err
	} else if source == nil {
		return fmt.Errorf("event source %q not found", req.GetName())
	}

	pullSource, ok := source.(RegisteredPullSource)
	if !ok {
		return fmt.Errorf("event source %q is not a PULL stream: %T", source.Proto().GetName(), source)
	}

	return pullSource.HandlePullStream(req, stream)
}

func (r *EventSourceRegistry) HandlePushStream(stream grpc.BidiStreamingServer[eventv1beta1.StreamPushEventsRequest, eventv1beta1.StreamPushEventsResponse]) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	if req == nil {
		return fmt.Errorf("push stream request cannot be nil")
	} else if req.GetName() == "" {
		return fmt.Errorf("push stream events request: name required")
	}

	source, err := r.Find(req.GetName())
	if err != nil {
		return err
	} else if source == nil {
		return fmt.Errorf("event source %q not found", req.GetName())
	}

	pushSource, ok := source.(RegisteredPushSource)
	if !ok {
		return fmt.Errorf("event source %q is not a PUSH stream: %T", source.Proto().GetName(), source)
	}

	return pushSource.HandlePushStream(req, stream)
}

func (s *pullSource[C, S]) HandlePullStream(req *eventv1beta1.StreamPullEventsRequest, stream grpc.ServerStreamingServer[eventv1beta1.StreamPullEventsResponse]) error {
	resp, err := s.HandlePullRequest(stream.Context(), req)
	if err != nil {
		return err
	}

	name := s.eventSource.GetName()
	err = stream.Send(&eventv1beta1.StreamPullEventsResponse{
		EventSource: name,
		Event:       resp.Event.GetName(),
		Payload:     resp.Payload,
		Action:      resp.Action,
	})
	if err != nil {
		return err
	}

	var (
		logger        = log.FromCtx(stream.Context())
		pullFreq      time.Duration
		continueOnErr = s.eventSource.ContinueOnError
	)
	if req.GetPullFreq().AsDuration() > 0 {
		pullFreq = req.GetPullFreq().AsDuration()
	} else if s.eventSource.GetPullFreq().AsDuration() > 0 {
		pullFreq = s.eventSource.GetPullFreq().AsDuration()
	}

	logger.Info("pull event source started", slog.String("event_source", name), slog.Duration("pullFreq", pullFreq))
	defer logger.Info("pull event source done", slog.String("event_source", name))

loop:
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-time.After(pullFreq):
			resp, err = s.HandlePullRequest(stream.Context(), req)
			if err != nil {
				if continueOnErr && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
					log.FromCtx(stream.Context()).Warn("pull event source continued on error", slog.String("event_source", name), log.Err(err))
					continue loop
				} else {
					return err
				}
			} else if resp == nil || resp.Event == nil || resp.Payload == nil {
				continue loop
			}

			err = stream.Send(&eventv1beta1.StreamPullEventsResponse{
				EventSource: name,
				Event:       resp.Event.GetName(),
				Payload:     resp.Payload,
				Action:      resp.Action,
			})
			if err != nil {
				return err
			}
		}
	}
}

func (s *pushSource[C, S]) HandlePushStream(req *eventv1beta1.StreamPushEventsRequest, stream grpc.BidiStreamingServer[eventv1beta1.StreamPushEventsRequest, eventv1beta1.StreamPushEventsResponse]) error {
	resp, err := s.HandlePushRequest(stream.Context(), req)
	if err != nil {
		return err
	}

	if err := stream.Send(&eventv1beta1.StreamPushEventsResponse{
		EventSource: s.eventSource.GetName(),
		Event:       resp.Event.GetName(),
		Payload:     resp.Payload,
		Action:      resp.Action,
	}); err != nil {
		return err
	}

	var (
		logger        = log.FromCtx(stream.Context())
		name          = s.eventSource.GetName()
		continueOnErr = s.eventSource.ContinueOnError
	)

	logger.Info("push event source started", slog.String("event_source", name))
	defer logger.Info("push event source done", slog.String("event_source", name))

loop:
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			req, err = stream.Recv()
			if err != nil {
				return err
			}

			resp, err = s.HandlePushRequest(stream.Context(), req)
			if err != nil {
				if continueOnErr && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
					log.FromCtx(stream.Context()).Warn("push event source continued on error", slog.String("event_source", name), log.Err(err))
					continue loop
				} else {
					return err
				}
			} else if resp == nil || resp.Event == nil || resp.Payload == nil {
				continue loop
			}

			if err = stream.Send(&eventv1beta1.StreamPushEventsResponse{
				EventSource: name,
				Event:       resp.Event.GetName(),
				Payload:     resp.Payload,
				Action:      resp.Action,
			}); err != nil {
				return err
			}
		}
	}
}

func (s *pullSource[C, S]) HandlePullRequest(ctx context.Context, req *eventv1beta1.StreamPullEventsRequest) (*EventWithPayload, error) {
	if req == nil || req.GetName() == "" {
		return nil, fmt.Errorf("pull stream name required")
	}

	var configAny *anypb.Any
	if req.GetConfig() != nil && req.GetConfig().MessageName().IsValid() {
		configAny = req.GetConfig()
	} else if s.eventSource.GetConfig() != nil && s.eventSource.GetConfig().MessageName().IsValid() {
		configAny = s.eventSource.GetConfig()
	} else if s.eventSource.RequiresConfig && !s.configSchema.IsEmpty() {
		return nil, fmt.Errorf("pull stream event source config required")
	}

	var (
		config C
		err    error
	)

	if configAny == nil && s.eventSource.RequiresConfig {
		config, err = common.UnwrapProtoAnyAs[C](configAny)
		if err != nil {
			return nil, fmt.Errorf("pull stream %q config unmarshal error: %w", s.eventSource.GetName(), err)
		}

		err = s.configSchema.Validate(config)
		if err != nil {
			return nil, fmt.Errorf("pull stream %q config validation error: %w", s.eventSource.GetName(), err)
		}
	}

	handler, err := s.GetHandler(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("pull stream event source handler error: %w", err)
	}

	return handler.PullEvent(ctx, s.events)
}

func (s *pushSource[C, S]) HandlePushRequest(ctx context.Context, req *eventv1beta1.StreamPushEventsRequest) (*EventWithPayload, error) {
	if req == nil || req.GetName() == "" {
		return nil, fmt.Errorf("push stream name required")
	}

	var config C
	if s.eventSource.GetConfig() == nil && !s.configSchema.IsEmpty() {
		return nil, fmt.Errorf("push stream %s: config is not set", s.eventSource.Name)
	} else if s.eventSource.GetConfig() != nil {
		var err error
		config, err = common.UnwrapProtoAnyAs[C](s.eventSource.GetConfig())
		if err != nil {
			return nil, err
		}
	}

	handler, err := s.GetHandler(ctx, config)
	if err != nil {
		return nil, err
	}

	return handler.PushEvent(ctx, s.events, HeadersFromProto(req.Headers), req.Body)
}

func (s *pullSource[C, S]) GetHandler(ctx context.Context, config C) (handler PullSourceHandler[C], err error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	_, handler, found := s.handlers.FindFunc(func(_ uuid.UUID, handler PullSourceHandler[C]) bool {
		return handler.ConfigEqual(config)
	})
	if found {
		return
	}

	handler, err = s.newHandler(ctx, config)
	if err != nil {
		return
	}

	s.handlers.Store(uuid.New(), handler)

	return
}

func (s *pushSource[C, S]) GetHandler(ctx context.Context, config C) (handler PushSourceHandler[C], err error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	_, handler, found := s.handlers.FindFunc(func(_ uuid.UUID, handler PushSourceHandler[C]) bool {
		return handler.ConfigEqual(config)
	})
	if found {
		return
	}

	handler, err = s.newHandler(ctx, config)
	if err != nil {
		return
	}

	s.handlers.Store(uuid.New(), handler)

	return
}

func (s *pullSource[C, S]) Proto() *eventv1beta1.EventSource {
	return proto.CloneOf(s.eventSource)
}

func (s *pushSource[C, S]) Proto() *eventv1beta1.EventSource {
	return proto.CloneOf(s.eventSource)
}

func (r *EventSourceRegistry) Protos() []*eventv1beta1.EventSource {
	return util.SliceMap(r.m.Values(), func(s RegisteredSource) *eventv1beta1.EventSource {
		return s.Proto()
	})
}

func (s *pullSource[C, S]) Close() error {
	s.mut.Lock()
	defer s.mut.Unlock()
	defer s.handlers.Clear()

	var errs []error
	s.handlers.Range(func(_ uuid.UUID, handler PullSourceHandler[C]) bool {
		errs = append(errs, handler.Close())
		return true
	})
	return errors.Join(errs...)
}

func (s *pushSource[C, S]) Close() error {
	s.mut.Lock()
	defer s.mut.Unlock()
	defer s.handlers.Clear()

	var errs []error
	s.handlers.Range(func(_ uuid.UUID, handler PushSourceHandler[C]) bool {
		errs = append(errs, handler.Close())
		return true
	})
	return errors.Join(errs...)
}

func (r *EventSourceRegistry) Close() error {
	var (
		sources = r.m.Values()
		errs    []error
	)
	for _, source := range sources {
		errs = append(errs, source.Close())
	}
	return errors.Join(errs...)
}
