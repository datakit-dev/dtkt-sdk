package integrationsdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/env"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/datakit-dev/grpc-proxy/proxy"
	"github.com/jhump/protoreflect/v2/sourceinfo"
)

type (
	Server[C any, I v1beta1.InstanceType] struct {
		intgr  *Integration[C, I]
		health *health.Server

		services     map[string]grpc.ServiceInfo
		proxyMethods map[string]GetProxyConnFunc

		mux  *http.ServeMux
		grpc *grpc.Server
		http http.Server

		stopCh chan struct{}
		doneCh chan struct{}
	}
)

func NewServer[C any, I v1beta1.InstanceType](intgr *Integration[C, I]) *Server[C, I] {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(
			intgr.unaryInterceptor(),
		),
		grpc.StreamInterceptor(
			intgr.streamInterceptor(),
		),
	}

	var (
		services     = map[string]grpc.ServiceInfo{}
		proxyMethods = map[string]GetProxyConnFunc{}
	)

	for name, svc := range intgr.services {
		services[name] = svc.info
		if svc.isProxy && svc.getConn != nil {
			for _, m := range svc.info.Methods {
				proxyMethods["/"+name+"/"+m.Name] = svc.getConn
			}
		}
	}

	srv := &Server[C, I]{
		intgr:        intgr,
		health:       health.NewServer(),
		services:     services,
		proxyMethods: proxyMethods,

		mux: http.NewServeMux(),

		stopCh: make(chan struct{}, 1),
		doneCh: make(chan struct{}, 1),
	}

	if len(proxyMethods) > 0 {
		encoding.RegisterCodecV2(proxy.Codec())

		opts = append(opts,
			grpc.ForceServerCodecV2(proxy.Codec()),
			grpc.UnknownServiceHandler(srv.proxyHandler()),
		)
	}

	srv.grpc = grpc.NewServer(opts...)

	return srv
}

func (s *Server[C, I]) Serve() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	s.http.Protocols = new(http.Protocols)
	s.http.Protocols.SetHTTP1(true)
	s.http.Protocols.SetUnencryptedHTTP2(true)
	s.http.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}

	for name, svc := range s.intgr.services {
		if svc.isProxy {
			continue
		}

		svc.regSvc(s.grpc)
		s.health.SetServingStatus(name, grpc_health_v1.HealthCheckResponse_SERVING)
	}

	grpc_health_v1.RegisterHealthServer(s.grpc, s.health)
	grpc_reflection_v1.RegisterServerReflectionServer(s.grpc,
		reflection.NewServerV1(reflection.ServerOptions{
			Services:           s,
			DescriptorResolver: sourceinfo.Files,
			ExtensionResolver:  sourceinfo.Types,
		}),
	)

	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	for pattern, handler := range s.intgr.handlers {
		s.mux.Handle(pattern, handler)
	}

	s.http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(
			r.Header.Get("Content-Type"), "application/grpc") {
			s.grpc.ServeHTTP(w, r)
		} else {
			s.mux.ServeHTTP(w, r)
		}
	})

	s.intgr.log.Info("Integration server starting...")

	if s.intgr.conn.Address().Network() == network.Socket.String() {
		// Close socket before bind to ensure new socket file
		//nolint:errcheck
		os.Remove(s.intgr.conn.Address().String())
	}
	//nolint:errcheck
	defer s.intgr.conn.Close()

	lis, err := s.intgr.conn.Bind(ctx)
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

	go s.stopWatch(ctx)

	s.intgr.setRunning(true)

	err = s.http.Serve(lis)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.intgr.log.Error("Server encountered error. Exiting...", slog.String("err", err.Error()))
		return err
	}

	cancel()
	<-s.doneCh

	return nil
}

func (s *Server[C, I]) GetServiceInfo() map[string]grpc.ServiceInfo {
	return s.services
}

func (s *Server[C, I]) Stop() {
	s.stopCh <- struct{}{}
}

func (s *Server[C, I]) proxyHandler() grpc.StreamHandler {
	return proxy.TransparentHandler(
		func(ctx context.Context, method string) (proxy.Mode, []proxy.Backend, error) {
			getConn, ok := s.proxyMethods[method]
			if !ok {
				return 0, nil, status.Errorf(codes.Aborted, "proxy method not found: %s", method)
			}

			return proxy.One2One, []proxy.Backend{
				&proxy.SingleBackend{
					GetConn: func(ctx context.Context) (context.Context, *grpc.ClientConn, error) {
						return getConn(ctx,
							network.DialGRPCInsecure,
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

func (s *Server[C, I]) stopWatch(ctx context.Context) {
	select {
	case <-ctx.Done():
		s.shutdown()
	case <-s.stopCh:
		s.shutdown()
	}
}

func (s *Server[C, I]) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer s.intgr.setRunning(false)

	s.intgr.log.Info("Integration server stopping...")

	s.grpc.GracefulStop()

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
