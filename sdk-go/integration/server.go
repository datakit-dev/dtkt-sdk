package integration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integration/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/env"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/datakit-dev/grpc-proxy/proxy"
	"github.com/jhump/protoreflect/v2/sourceinfo"
)

type (
	Server[C any, I v1beta1.InstanceType] struct {
		intgr  *Integration[C, I]
		health *health.Server
		http   http.Server

		services map[string]grpc.ServiceInfo
		proxies  map[string]GetConnFunc

		regOnce sync.Once // protects one-time gRPC service registration
		stopCh  chan struct{}
		doneCh  chan struct{}
	}
)

func NewServer[C any, I v1beta1.InstanceType](intgr *Integration[C, I]) *Server[C, I] {
	var (
		services = map[string]grpc.ServiceInfo{}
		proxies  = map[string]GetConnFunc{}
	)

	for name, svc := range intgr.services {
		info := NewServiceInfo(svc.desc)

		services[name] = info

		if svc.isProxy && svc.getConn != nil {
			for _, m := range info.Methods {
				proxies["/"+name+"/"+m.Name] = svc.getConn
			}
		}
	}

	return &Server[C, I]{
		intgr:  intgr,
		health: health.NewServer(),

		services: services,
		proxies:  proxies,

		stopCh: make(chan struct{}, 1),
		doneCh: make(chan struct{}, 1),
	}
}

func (s *Server[C, I]) Serve() error {
	bind, err := network.ConnectorFromEnv(s.intgr.addr)
	if err != nil {
		return err
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(
			s.intgr.unaryInterceptor(),
		),
		grpc.StreamInterceptor(
			s.intgr.streamInterceptor(),
		),
	}

	if len(s.proxies) > 0 {
		encoding.RegisterCodecV2(proxy.Codec())

		opts = append(opts,
			grpc.ForceServerCodecV2(proxy.Codec()),
			grpc.UnknownServiceHandler(s.proxyHandler()),
		)
	}

	grpc := grpc.NewServer(opts...)
	s.registerGRPCServices(grpc)

	s.http.Protocols = new(http.Protocols)
	s.http.Protocols.SetHTTP1(true)

	if env.GetVar(env.TLSCACert) != "" && env.GetVar(env.TLSBundle) != "" {
		// h2c (cleartext HTTP/2) is only safe to accept when there's no TLS
		// listener terminating the connection; ALPN negotiates h2 otherwise.
		s.http.Protocols.SetUnencryptedHTTP2(true)
	} else {
		// The client always negotiates real (ALPN) h2 over TLS connections
		// (see network.HTTPClient.DialContext); without this the server can
		// only speak HTTP/1.1 on that conn and rejects the h2 preface.
		s.http.Protocols.SetHTTP2(true)
	}

	s.http.BaseContext = func(_ net.Listener) context.Context {
		return s.intgr.ctx
	}

	mux := http.NewServeMux()
	for pattern, handler := range s.intgr.handlers {
		mux.Handle(pattern, handler)
	}

	s.http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(
			r.Header.Get("Content-Type"), "application/grpc") {
			grpc.ServeHTTP(w, r)
		} else {
			mux.ServeHTTP(w, r)
		}
	})

	s.intgr.log.Info("Integration server starting...")

	if bind.Address().Network() == network.Socket.String() {
		// Close socket before bind to ensure new socket file
		err = os.Remove(bind.Address().String())
		if err != nil {
			s.intgr.Logger().Error("remove socket file", log.Err(err))
		}
	}

	lis, err := bind.Listener(s.intgr.ctx)
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer lis.Close()

	pidFile := env.GetVar(env.PIDFile)
	if pidFile != "" {
		pid, err := util.WritePID(pidFile)
		if err != nil {
			return err
		}

		//nolint:errcheck
		defer pid.Unlock()
	}

	go s.stopWatch()

	s.intgr.setRunning(true)
	defer s.intgr.setRunning(false)

	err = s.http.Serve(lis)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.intgr.log.Error("Server encountered error. Exiting...", slog.String("err", err.Error()))
		return err
	}

	s.intgr.cancel()
	<-s.doneCh

	return nil
}

func (s *Server[C, I]) GetServiceInfo() map[string]grpc.ServiceInfo {
	return s.services
}

func (s *Server[C, I]) Stop() {
	s.stopCh <- struct{}{}
}

// registerGRPCServices registers all gRPC services, health, and reflection on
// s.grpc exactly once. It is safe to call from multiple code paths.
func (s *Server[C, I]) registerGRPCServices(grpc *grpc.Server) {
	s.regOnce.Do(func() {
		for name, svc := range s.intgr.services {
			if svc.isProxy {
				continue
			}

			svc.regSvc(grpc)
			s.health.SetServingStatus(name, grpc_health_v1.HealthCheckResponse_SERVING)
		}

		grpc_health_v1.RegisterHealthServer(grpc, s.health)
		grpc_reflection_v1.RegisterServerReflectionServer(grpc,
			reflection.NewServerV1(reflection.ServerOptions{
				Services:           s,
				DescriptorResolver: sourceinfo.Files,
				ExtensionResolver:  sourceinfo.Types,
			}),
		)

		s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	})
}

func (s *Server[C, I]) proxyHandler() grpc.StreamHandler {
	return proxy.TransparentHandler(
		func(ctx context.Context, method string) (proxy.Mode, []proxy.Backend, error) {
			getConn, ok := s.proxies[method]
			if !ok {
				return 0, nil, status.Errorf(codes.Aborted, "proxy method not found: %s", method)
			}

			return proxy.One2One, []proxy.Backend{
				&proxy.SingleBackend{
					GetConn: func(ctx context.Context) (context.Context, *grpc.ClientConn, error) {
						return getConn(ctx,
							grpc.WithTransportCredentials(
								insecure.NewCredentials(),
							),
							grpc.WithDefaultCallOptions(
								grpc.ForceCodecV2(proxy.Codec()),
								grpc.CallContentSubtype("proto"),
							),
						)
					},
				},
			}, nil
		},
	)
}

func (s *Server[C, I]) stopWatch() {
	select {
	case <-s.intgr.ctx.Done():
		s.shutdown()
	case <-s.stopCh:
		s.shutdown()
	}
}

func (s *Server[C, I]) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.intgr.log.Info("Integration server stopping...")

	if err := s.http.Shutdown(ctx); err != nil {
		if !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, context.Canceled) {
			s.intgr.log.Error("Server encountered error while shutting down.", slog.String("err", err.Error()))
		}
	}

	if err := s.intgr.close(); err != nil {
		s.intgr.log.Error(fmt.Sprintf("Error closing integration: %s", err.Error()))
	}

	s.doneCh <- struct{}{}
}
