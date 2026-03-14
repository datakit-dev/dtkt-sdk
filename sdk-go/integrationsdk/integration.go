package integrationsdk

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/env"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/middleware"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/resource"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/jhump/protoreflect/v2/sourceinfo"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ v1beta1.InstanceMux[v1beta1.InstanceType] = (*Integration[any, v1beta1.InstanceType])(nil)

type Integration[C any, I v1beta1.InstanceType] struct {
	basev1beta1.UnimplementedBaseServiceServer

	log  *slog.Logger
	name resource.Name
	addr network.Address
	conn network.Connector

	spec   *sharedv1beta1.Package
	config *v1beta1.TypeSchema[C]

	types   *v1beta1.TypeRegistry
	actions *v1beta1.ActionRegistry
	events  *v1beta1.EventRegistry
	sources *v1beta1.EventSourceRegistry

	services map[string]func(grpc.ServiceRegistrar)
	handlers map[string]http.Handler

	instances   util.SyncMap[string, *Instance[I]]
	newInstance func(context.Context, C) (I, error)

	started bool
	mut     sync.Mutex
	mux     singleflight.Group
}

func New[C any, I v1beta1.InstanceType](
	spec *sharedv1beta1.Package,
	newInstance NewInstanceFunc[C, I],
) (*Integration[C, I], error) {
	if spec.GetType() != sharedv1beta1.PackageType_PACKAGE_TYPE_GO {
		return nil, fmt.Errorf("invalid package type: %s, expected: %s", spec.Type, sharedv1beta1.PackageType_PACKAGE_TYPE_GO)
	}

	addr, err := network.DefaultAddress(network.DefaultNetwork())
	if err != nil {
		return nil, err
	}

	conn, err := network.ResolveConnector()
	if err != nil {
		return nil, err
	}

	var (
		// ident   = common.GetPackageIdentity(spec)
		name    resource.Name
		types   *v1beta1.TypeRegistry
		logOpts = []any{
			slog.String("network", addr.Network()),
			slog.String("address", addr.String()),
		}
	)
	if env.GetVar(env.ContextAddress) != "" && resource.Deployment.IsName(env.GetVar(env.DeployName)) {
		addr, err := network.ParseAddr(env.GetVar(env.ContextAddress))
		if err != nil {
			return nil, err
		}

		name = resource.Deployment.MustGetName(env.GetVar(env.DeployName))
		types = v1beta1.NewTypeRegistry(v1beta1.DefaultTypeSyncer(),
			addr.URL().JoinPath("schemas", name.String(), "types"),
			// v1beta1.WithRuntimeProtoGen(ident.ProtoPackage("integration")),
		)

		logOpts = append(logOpts, slog.String("deployment", name.String()))
	} else {
		name = resource.EmptyName()
		types = v1beta1.NewTypeRegistry(v1beta1.DefaultTypeSyncer(),
			addr.URL().JoinPath("/types"),
			// v1beta1.WithRuntimeProtoGen(ident.ProtoPackage("integration")),
		)
	}

	config, err := v1beta1.NewTypeSchemaFor[C](types, "Config")
	if err != nil {
		return nil, err
	}

	intgr := &Integration[C, I]{
		log: log.NewLogger().With(logOpts...),

		spec:   spec,
		config: config,

		name: name,
		addr: addr,
		conn: conn,

		types:   types,
		actions: &v1beta1.ActionRegistry{},
		events:  &v1beta1.EventRegistry{},
		sources: &v1beta1.EventSourceRegistry{},

		handlers: map[string]http.Handler{},
		services: map[string]func(grpc.ServiceRegistrar){},

		newInstance: newInstance,
	}

	// Register Integration as BaseService implementation
	err = intgr.addService(basev1beta1.BaseService_ServiceDesc.ServiceName, func(srv grpc.ServiceRegistrar) {
		basev1beta1.RegisterBaseServiceServer(srv, intgr)
	})
	if err != nil {
		return nil, err
	}

	return intgr, nil
}

func NewFS[C any, I v1beta1.InstanceType](
	fs embed.FS,
	newInstance NewInstanceFunc[C, I],
) (*Integration[C, I], error) {
	reader, err := fs.Open(SpecFile)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer reader.Close()

	spec, err := ReadSpec(encoding.YAML, reader)
	if err != nil {
		return nil, err
	}

	return New(spec.GetPackage(), newInstance)
}

func (i *Integration[C, I]) String() string {
	return common.PackageIdentityFromProto(i.spec.Identity).String()
}

func (i *Integration[C, I]) Logger() *slog.Logger {
	return i.log
}

func (i *Integration[C, I]) Address() network.Address {
	return i.addr
}

func (i *Integration[C, I]) Package() *sharedv1beta1.Package {
	return i.spec
}

func (i *Integration[C, I]) ConfigSchema() *sharedv1beta1.TypeSchema {
	return i.config.ToProto()
}

func (i *Integration[C, I]) Types() *v1beta1.TypeRegistry {
	return i.types
}

func (i *Integration[C, I]) Actions() *v1beta1.ActionRegistry {
	return i.actions
}

func (i *Integration[C, I]) Events() *v1beta1.EventRegistry {
	return i.events
}

func (i *Integration[C, I]) EventSources() *v1beta1.EventSourceRegistry {
	return i.sources
}

func (i *Integration[C, I]) GetInstance(ctx context.Context) (inst I, err error) {
	req, err := middleware.RequestFromContext(ctx)
	if err != nil {
		err = status.Error(codes.InvalidArgument, err.Error())
		return
	}

	wrap, err := i.getInstance(req)
	if err != nil {
		return
	}

	return wrap.inst, nil
}

func (i *Integration[C, I]) GetDataRoot(ctx context.Context) (string, error) {
	req, err := middleware.RequestFromContext(ctx)
	if err != nil {
		return "", err
	}

	root := os.Getenv(env.DataRoot)
	if root == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}

		root = cacheDir
	}

	return filepath.Abs(filepath.Join(root, strings.ToLower(i.String()), req.ConfigHash()))
}

func (i *Integration[C, I]) Serve() error {
	return NewServer(i).Serve()
}

func (i *Integration[C, I]) addHandler(path string, handler http.Handler) error {
	if i.isRunning() {
		return fmt.Errorf("cannot modify handlers, server running")
	}

	i.mut.Lock()
	defer i.mut.Unlock()

	i.handlers[path] = handler

	return nil
}

func (i *Integration[C, I]) addService(name string, reg func(grpc.ServiceRegistrar)) error {
	if i.isRunning() {
		return fmt.Errorf("cannot modify services, server running")
	}

	i.mut.Lock()
	defer i.mut.Unlock()

	desc, err := sourceinfo.Files.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return fmt.Errorf("service: %q: %w", name, err)
	}

	svc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return fmt.Errorf("expected service descriptor for: %q, got: %T", name, desc)
	}

	for idx := range svc.Methods().Len() {
		method := svc.Methods().Get(idx)
		inputType, err := sourceinfo.Types.FindMessageByName(method.Input().FullName())
		if err != nil {
			return fmt.Errorf("method input: %q: %w", method.Input().FullName(), err)
		}

		_, err = v1beta1.NewTypeSchemaForProto(i.types, inputType.New().Interface())
		if err != nil {
			return fmt.Errorf("method input: %q: %w", method.Input().FullName(), err)
		}

		outputType, err := sourceinfo.Types.FindMessageByName(method.Output().FullName())
		if err != nil {
			return fmt.Errorf("method output: %q: %w", method.Output().FullName(), err)
		}

		_, err = v1beta1.NewTypeSchemaForProto(i.types, outputType.New().Interface())
		if err != nil {
			return fmt.Errorf("method output: %q: %w", method.Output().FullName(), err)
		}
	}

	i.services[name] = reg
	i.log.Info(fmt.Sprintf("Added service %q", name))

	return nil
}

func (i *Integration[C, I]) GetPackage(context.Context, *basev1beta1.GetPackageRequest) (*basev1beta1.GetPackageResponse, error) {
	return &basev1beta1.GetPackageResponse{
		Package:      i.spec,
		ConfigSchema: i.config.ToProto(),
	}, nil
}

func (i *Integration[C, I]) CheckConfig(ctx context.Context, req *basev1beta1.CheckConfigRequest) (*basev1beta1.CheckConfigResponse, error) {
	_, err := i.getInstance(middleware.NewRequest(req.Connection, req.ConfigHash, req.ConfigGen))
	if err != nil {
		inst, err := NewInstance(ctx, i, req)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "new instance error: %s", err)
		}

		i.invalidatePrev(inst.Prev())
		i.instances.Store(inst.String(), inst)
	}

	return &basev1beta1.CheckConfigResponse{
		Valid:   true,
		Message: "Check config succeeded.",
	}, nil
}

func (i *Integration[C, I]) CheckAuth(ctx context.Context, req *basev1beta1.CheckAuthRequest) (*basev1beta1.CheckAuthResponse, error) {
	inst, err := i.GetInstance(ctx)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	return inst.CheckAuth(ctx, req)
}

func (i *Integration[C, I]) ListTypes(_ context.Context, req *basev1beta1.ListTypesRequest) (*basev1beta1.ListTypesResponse, error) {
	return i.types.ListTypes(req)
}

func (i *Integration[C, I]) GetType(_ context.Context, req *basev1beta1.GetTypeRequest) (*basev1beta1.GetTypeResponse, error) {
	return i.types.GetType(req)
}

func (i *Integration[C, I]) isRunning() bool {
	i.mut.Lock()
	defer i.mut.Unlock()
	return i.started
}

func (i *Integration[C, I]) setRunning(running bool) {
	i.mut.Lock()
	defer i.mut.Unlock()
	i.started = running
}

func (i *Integration[C, I]) unaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = grpcRecover(ctx, info.FullMethod, r)
			}
		}()

		switch info.FullMethod {
		case grpc_health_v1.Health_Check_FullMethodName,
			grpc_health_v1.Health_List_FullMethodName,
			basev1beta1.BaseService_GetType_FullMethodName,
			basev1beta1.BaseService_ListTypes_FullMethodName,
			basev1beta1.BaseService_GetPackage_FullMethodName,
			basev1beta1.BaseService_CheckConfig_FullMethodName:
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.FailedPrecondition, "request headers not found")
		}

		mreq := middleware.RequestFromGRPC(md)
		if err := mreq.IsValid(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "request invalid: %s", err)
		}

		res, err := handler(i.newContext(ctx, mreq), req)
		if err != nil {
			return nil, grpcError(err)
		}

		return res, nil
	}
}

func (i *Integration[C, I]) streamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = grpcRecover(stream.Context(), info.FullMethod, r)
			}
		}()

		switch info.FullMethod {
		case grpc_reflection_v1.ServerReflection_ServerReflectionInfo_FullMethodName, grpc_health_v1.Health_Watch_FullMethodName:
			return handler(srv, stream)
		}

		md, ok := metadata.FromIncomingContext(stream.Context())
		if !ok {
			return status.Error(codes.FailedPrecondition, "request headers not found")
		}

		req := middleware.RequestFromGRPC(md)
		if err := req.IsValid(); err != nil {
			return status.Errorf(codes.InvalidArgument, "request invalid: %s", err)
		}

		err = handler(srv, &grpcServerStream{ServerStream: stream, ctx: i.newContext(stream.Context(), req)})
		if err != nil {
			return grpcError(err)
		}

		return nil
	}
}

func (i *Integration[C, I]) newContext(ctx context.Context, req *middleware.Request) context.Context {
	ctx = log.NewCtx(ctx, i.log.With(slog.String("connection", req.AddrName())))
	return middleware.AddRequestToContext(
		v1beta1.AddPackageToContext(
			v1beta1.AddRegistryToContext(ctx, i.types), i.Package(),
		), req,
	)
}

func (i *Integration[C, I]) getInstance(req *middleware.Request) (*Instance[I], error) {
	inst, ok := i.instances.Load(req.String())
	if !ok {
		value, err, _ := i.mux.Do(req.String(), func() (any, error) {
			cached, ok := i.instances.Load(req.String())
			if ok {
				return cached, nil
			}
			return nil, fmt.Errorf("instance not found: %s", req.AddrName())
		})
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		inst = value.(*Instance[I])
	}

	if req.ConfigHash() != inst.ConfigHash() {
		return nil, status.Error(codes.InvalidArgument, "invalid request: config hash mismatch")
	}

	return inst, nil
}

func (i *Integration[C, I]) invalidatePrev(reqHash string) {
	if inst, ok := i.instances.Load(reqHash); ok {
		err := inst.Close()
		if err != nil {
			i.log.Error("Error while closing instance.", log.Err(err))
		}
		i.instances.Delete(reqHash)
		i.mux.Forget(reqHash)
	}
}

func (i *Integration[C, I]) close() error {
	var errs []error

	if i.sources != nil {
		errs = append(errs, i.sources.Close())
	}

	i.instances.Range(func(key string, inst *Instance[I]) bool {
		errs = append(errs, inst.Close())
		return true
	})
	return errors.Join(errs...)
}
