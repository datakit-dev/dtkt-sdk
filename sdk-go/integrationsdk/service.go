package integrationsdk

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	actionv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/action/v1beta1"
	eventv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/event/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

type (
	Service struct {
		info    grpc.ServiceInfo
		regSvc  RegisterServiceFunc
		getConn GetProxyConnFunc
		isProxy bool
	}
	InitServiceFunc[I v1beta1.InstanceType, S any] func(v1beta1.InstanceMux[I]) S
	RegisterServiceFunc                            func(grpc.ServiceRegistrar)
	GetProxyConnFunc                               func(context.Context) (context.Context, *grpc.ClientConn, error)
)

func NewServiceInfo(svcDesc *grpc.ServiceDesc) grpc.ServiceInfo {
	methods := make([]grpc.MethodInfo, 0, len(svcDesc.Methods)+len(svcDesc.Streams))
	for _, m := range svcDesc.Methods {
		methods = append(methods, grpc.MethodInfo{
			Name:           m.MethodName,
			IsClientStream: false,
			IsServerStream: false,
		})
	}

	for _, s := range svcDesc.Streams {
		methods = append(methods, grpc.MethodInfo{
			Name:           s.StreamName,
			IsClientStream: s.ClientStreams,
			IsServerStream: s.ServerStreams,
		})
	}

	return grpc.ServiceInfo{
		Methods:  methods,
		Metadata: svcDesc.Metadata,
	}
}

func RegisterServiceProxy[C any, I v1beta1.InstanceType](intgr *Integration[C, I], svcDesc *grpc.ServiceDesc, getConn GetProxyConnFunc) {
	err := intgr.RegisterProxy(svcDesc, getConn)
	if err != nil {
		log.Fatal(err)
	}
}

func RegisterService[S, C any, I v1beta1.InstanceType](intgr *Integration[C, I], svcDesc *grpc.ServiceDesc, initSvc InitServiceFunc[I, S]) {
	err := intgr.RegisterService(svcDesc, func(reg grpc.ServiceRegistrar) {
		reg.RegisterService(svcDesc, initSvc(intgr))
	})
	if err != nil {
		log.Fatal(err)
	}
}

func RegisterManagedActionService[S actionv1beta1.ActionServiceServer, C any, I v1beta1.InstanceType](intgr *Integration[C, I], initSvc InitServiceFunc[I, S], regActions ...v1beta1.RegisterActionFunc[I]) {
	err := intgr.RegisterService(&actionv1beta1.ActionService_ServiceDesc, func(reg grpc.ServiceRegistrar) {
		actionv1beta1.RegisterActionServiceServer(reg, initSvc(intgr))
	})
	if err != nil {
		log.Fatal(err)
	}

	err = v1beta1.RegisterActions(intgr, regActions...)
	if err != nil {
		log.Fatal(err)
	}
}

func RegisterManagedEventService[S eventv1beta1.EventServiceServer, C any, I v1beta1.InstanceType](intgr *Integration[C, I], initSvc InitServiceFunc[I, S], regEvents []v1beta1.RegisterEventFunc[I], regSources ...v1beta1.RegisterSourceFunc[I]) {
	err := intgr.RegisterService(&eventv1beta1.EventService_ServiceDesc, func(reg grpc.ServiceRegistrar) {
		eventv1beta1.RegisterEventServiceServer(reg, initSvc(intgr))
	})
	if err != nil {
		log.Fatal(err)
	}

	err = v1beta1.RegisterEvents(intgr, regEvents...)
	if err != nil {
		log.Fatal(err)
	}

	err = v1beta1.RegisterSources(intgr, regSources...)
	if err != nil {
		log.Fatal(err)
	}

	err = intgr.RegisterHandler("/"+v1beta1.EventSourcesPrefix+"/{event_source}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		log.Fatal(err)
	}
}
