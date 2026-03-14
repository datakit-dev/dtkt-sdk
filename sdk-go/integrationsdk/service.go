package integrationsdk

import (
	"fmt"
	"io"
	logger "log"
	"net/http"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	actionv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/action/v1beta1"
	eventv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/event/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

type NewServiceFunc[I v1beta1.InstanceType, S any] func(v1beta1.InstanceMux[I]) S

func RegisterService[S, C any, I v1beta1.InstanceType](intgr *Integration[C, I], svcDesc *grpc.ServiceDesc, initSvc NewServiceFunc[I, S]) {
	err := intgr.addService(svcDesc.ServiceName, func(reg grpc.ServiceRegistrar) {
		reg.RegisterService(svcDesc, initSvc(intgr))
	})
	if err != nil {
		logger.Fatal(err)
	}
}

func RegisterManagedActionService[S actionv1beta1.ActionServiceServer, I v1beta1.InstanceType, C any](intgr *Integration[C, I], initSvc NewServiceFunc[I, S], regActions ...v1beta1.RegisterActionFunc[I]) {
	err := intgr.addService(actionv1beta1.ActionService_ServiceDesc.ServiceName, func(reg grpc.ServiceRegistrar) {
		actionv1beta1.RegisterActionServiceServer(reg, initSvc(intgr))
	})
	if err != nil {
		logger.Fatal(err)
	}

	err = v1beta1.RegisterActions(intgr, regActions...)
	if err != nil {
		logger.Fatal(err)
	}
}

func RegisterManagedEventService[S eventv1beta1.EventServiceServer, I v1beta1.InstanceType, C any](intgr *Integration[C, I], initSvc NewServiceFunc[I, S], regEvents []v1beta1.RegisterEventFunc[I], regSources ...v1beta1.RegisterSourceFunc[I]) {
	err := intgr.addService(eventv1beta1.EventService_ServiceDesc.ServiceName, func(reg grpc.ServiceRegistrar) {
		eventv1beta1.RegisterEventServiceServer(reg, initSvc(intgr))
	})
	if err != nil {
		logger.Fatal(err)
	}

	err = v1beta1.RegisterEvents(intgr, regEvents...)
	if err != nil {
		logger.Fatal(err)
	}

	err = v1beta1.RegisterSources(intgr, regSources...)
	if err != nil {
		logger.Fatal(err)
	}

	err = intgr.addHandler("/"+v1beta1.EventSourcesPrefix+"/{event_source}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source, err := intgr.sources.Find(fmt.Sprintf("%s/%s", v1beta1.EventSourcesPrefix, r.PathValue("event_source")))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch source := source.(type) {
		case v1beta1.RegisteredPushSource:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			name := source.Proto().GetName()
			resp, err := source.HandlePushRequest(r.Context(), &eventv1beta1.StreamPushEventsRequest{
				Name:    name,
				Headers: v1beta1.HeadersToProto(r.Header),
				Body:    body,
			})
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			b, err := protojson.Marshal(&eventv1beta1.StreamPushEventsResponse{
				EventSource: name,
				Event:       resp.Event.GetName(),
				Payload:     resp.Payload,
				Action:      resp.Action,
			})
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			_, err = w.Write(b)
			if err != nil {
				intgr.Logger().Error(err.Error())
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	if err != nil {
		logger.Fatal(err)
	}
}
