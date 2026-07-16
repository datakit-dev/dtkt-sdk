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
	addr network.Address
	name resource.Name
	root string

	spec     *sharedv1beta1.Package
	schema   *v1beta1.TypeSchema[C]
	registry *v1beta1.TypeRegistry
	actions  *v1beta1.ActionRegistry
	events   *v1beta1.EventRegistry
	sources  *v1beta1.EventSourceRegistry

	services map[string]Service
	handlers map[string]http.Handler

	instances   util.SyncMap[string, *instance]
	newInstance NewInstanceFunc[C, I]

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

	var (
		ident    = common.GetPackageIdentity(spec)
		name     resource.Name
		registry *v1beta1.TypeRegistry
		logOpts  = []any{
			slog.String("package", ident.String()),
			slog.String("network", addr.Network()),
			slog.String("address", addr.String()),
		}
	)

	root := os.Getenv(env.DataRoot)
	if root == "" {
		root, err = os.UserCacheDir()
		if err != nil {
			return nil, err
		}

		root = filepath.Join(root, strings.ToLower(ident.String()))
	}

	if env.GetVar(env.ContextAddress) != "" && resource.Deployment.IsName(env.GetVar(env.DeployName)) {
		addr, err := network.ParseAddr(env.GetVar(env.ContextAddress))
		if err != nil {
			return nil, err
		}

		name = resource.Deployment.MustGetName(env.GetVar(env.DeployName))
		registry = v1beta1.NewTypeRegistry(v1beta1.DefaultTypeSyncer(),
			addr.URL().JoinPath("schemas", name.String(), "types"),
			// Experimental (not enabled until stable):
			// v1beta1.WithRuntimeProtoGen(ident.ProtoPackage("integration")),
		)

		logOpts = append(logOpts, slog.String("deployment", name.String()))
	} else {
		name = resource.EmptyName()
		registry = v1beta1.NewTypeRegistry(v1beta1.DefaultTypeSyncer(),
			addr.URL().JoinPath("types"),
			// Experimental (not enabled until stable):
			// v1beta1.WithRuntimeProtoGen(ident.ProtoPackage("integration")),
		)
	}

	config, err := v1beta1.NewTypeSchemaFor[C](registry, "Config")
	if err != nil {
		return nil, err
	}

	intgr := &Integration[C, I]{
		log: log.NewLogger(
			log.WithFormat(log.FormatPretty),
			log.WithSource(true),
		).With(logOpts...),
		addr: addr,
		name: name,
		root: root,

		spec:     spec,
		schema:   config,
		registry: registry,
		actions:  &v1beta1.ActionRegistry{},
		events:   &v1beta1.EventRegistry{},
		sources:  &v1beta1.EventSourceRegistry{},

		handlers: map[string]http.Handler{},
		services: map[string]Service{},

		newInstance: newInstance,
	}

	if env.GetVar(env.AppEnv) == "dev" {
		debugPath, debugHandler := env.Handler()
		intgr.handlers[debugPath] = debugHandler
		intgr.handlers[StatsPath] = StatsHandler(intgr.log, &intgr.instances)
	}

	// Register Integration as BaseService implementation
	err = intgr.RegisterService(&basev1beta1.BaseService_ServiceDesc, func(reg grpc.ServiceRegistrar) {
		basev1beta1.RegisterBaseServiceServer(reg, intgr)
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

	spec, err := ReadSpec(reader, encoding.YAML)
	if err != nil {
		return nil, err
	}

	return New(spec.GetV1Beta1(), newInstance)
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
	return i.schema.ToProto()
}

func (i *Integration[C, I]) Types() *v1beta1.TypeRegistry {
	return i.registry
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

func (i *Integration[C, I]) GetInstance(ctx context.Context) (_ I, err error) {
	req, err := middleware.RequestFromContext(ctx)
	if err != nil {
		err = status.Error(codes.InvalidArgument, err.Error())
		return
	}

	inst, err := i.getInstance(req)
	if err != nil {
		return
	}

	return inst.InstanceType.(I), nil
}

func (i *Integration[C, I]) DataRoot() string {
	return i.root
}

func (i *Integration[C, I]) Serve() error {
	return NewServer(i).Serve()
}

func (i *Integration[C, I]) RegisterHandler(pattern string, handler http.Handler) error {
	if i.isRunning() {
		return fmt.Errorf("cannot modify handlers, server running")
	}

	i.mut.Lock()
	defer i.mut.Unlock()

	if _, ok := i.handlers[pattern]; ok {
		return fmt.Errorf("handler already registered: %s", pattern)
	}

	i.handlers[pattern] = handler

	return nil
}

func (i *Integration[C, I]) RegisterService(svcDesc *grpc.ServiceDesc, regSvc func(grpc.ServiceRegistrar)) error {
	if i.isRunning() {
		return fmt.Errorf("cannot modify services, server running")
	} else if svcDesc == nil {
		return fmt.Errorf("service descriptor cannot be nil")
	} else if svcDesc.ServiceName == "" {
		return fmt.Errorf("service name required")
	} else if regSvc == nil {
		return fmt.Errorf("service initializer cannot be nil")
	}

	i.mut.Lock()
	defer i.mut.Unlock()

	svcName := svcDesc.ServiceName
	if _, ok := i.services[svcName]; ok {
		return fmt.Errorf("service already registered: %s", svcName)
	}

	err := i.syncServiceTypes(svcName)
	if err != nil {
		return err
	}

	i.services[svcName] = NewService(svcDesc, regSvc)

	return nil
}

func (i *Integration[C, I]) RegisterProxy(svcDesc *grpc.ServiceDesc, getConn GetProxyConnFunc[I]) error {
	if i.isRunning() {
		return fmt.Errorf("cannot modify services, server running")
	} else if svcDesc == nil {
		return fmt.Errorf("service descriptor cannot be nil")
	} else if svcDesc.ServiceName == "" {
		return fmt.Errorf("service name required")
	} else if getConn == nil {
		return fmt.Errorf("get proxy connection cannot be nil")
	}

	i.mut.Lock()
	defer i.mut.Unlock()

	svcName := svcDesc.ServiceName
	if _, ok := i.services[svcName]; ok {
		return fmt.Errorf("service already registered: %s", svcName)
	}

	err := i.syncServiceTypes(svcName)
	if err != nil {
		return err
	}

	i.services[svcName] = NewServiceProxy(svcDesc, func(ctx context.Context, opts ...grpc.DialOption) (context.Context, *grpc.ClientConn, error) {
		return getConn(ctx, i, opts...)
	})

	return nil
}

func (i *Integration[C, I]) GetPackage(context.Context, *basev1beta1.GetPackageRequest) (*basev1beta1.GetPackageResponse, error) {
	return &basev1beta1.GetPackageResponse{
		Package:      i.spec,
		ConfigSchema: i.schema.ToProto(),
	}, nil
}

func (i *Integration[C, I]) CheckConfig(ctx context.Context, req *basev1beta1.CheckConfigRequest) (*basev1beta1.CheckConfigResponse, error) {
	inst, err := i.getInstance(middleware.NewRequest(req.Connection, req.ConfigHash, req.ConfigGen))
	if err != nil {
		inst, err = NewInstance(i, req)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "new instance error: %s", err)
		}

		err = i.closeInstance(inst.PrevSHA256())
		if err != nil {
			i.log.Error("Error while closing instance.", log.Err(err))
		}

		i.instances.Store(inst.HashSHA256(), inst)
	}

	var authRequired *sharedv1beta1.AuthType
	resp, _ := inst.CheckAuth(ctx, &basev1beta1.CheckAuthRequest{})
	if resp.GetAuthRequired() > 0 {
		authRequired = new(resp.GetAuthRequired())
	}

	return &basev1beta1.CheckConfigResponse{
		AuthRequired: authRequired,
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
	return i.registry.ListTypes(req)
}

func (i *Integration[C, I]) GetType(_ context.Context, req *basev1beta1.GetTypeRequest) (*basev1beta1.GetTypeResponse, error) {
	return i.registry.GetType(req)
}

func (i *Integration[C, I]) Close(ctx context.Context, req *basev1beta1.CloseRequest) (*basev1beta1.CloseResponse, error) {
	mreq, err := middleware.RequestFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	inst, err := i.getInstance(mreq)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	err = i.closeInstance(inst.HashSHA256())
	if err != nil {
		return nil, status.Error(codes.Aborted, err.Error())
	}

	return &basev1beta1.CloseResponse{}, nil
}

func (i *Integration[C, I]) syncServiceTypes(svcName string) error {
	desc, err := sourceinfo.Files.FindDescriptorByName(protoreflect.FullName(svcName))
	if err != nil {
		return fmt.Errorf("service: %q: %w", svcName, err)
	}

	svc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return fmt.Errorf("expected service descriptor for: %q, got: %T", svcName, desc)
	}

	for idx := range svc.Methods().Len() {
		method := svc.Methods().Get(idx)
		inputType, err := sourceinfo.Types.FindMessageByName(method.Input().FullName())
		if err != nil {
			return fmt.Errorf("method input: %q: %w", method.Input().FullName(), err)
		}

		_, err = v1beta1.NewTypeSchemaForProto(i.registry, inputType.New().Interface())
		if err != nil {
			return fmt.Errorf("method input: %q: %w", method.Input().FullName(), err)
		}

		outputType, err := sourceinfo.Types.FindMessageByName(method.Output().FullName())
		if err != nil {
			return fmt.Errorf("method output: %q: %w", method.Output().FullName(), err)
		}

		_, err = v1beta1.NewTypeSchemaForProto(i.registry, outputType.New().Interface())
		if err != nil {
			return fmt.Errorf("method output: %q: %w", method.Output().FullName(), err)
		}
	}

	return nil
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
			basev1beta1.BaseService_CheckConfig_FullMethodName,
			basev1beta1.BaseService_GetPackage_FullMethodName,
			basev1beta1.BaseService_GetType_FullMethodName,
			basev1beta1.BaseService_ListTypes_FullMethodName:
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
		case grpc_reflection_v1.ServerReflection_ServerReflectionInfo_FullMethodName,
			grpc_health_v1.Health_Watch_FullMethodName:
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

		err = handler(srv, &grpcServerStream{
			ServerStream: stream,
			ctx:          i.newContext(stream.Context(), req),
		})
		if err != nil {
			return grpcError(err)
		}

		return nil
	}
}

func (i *Integration[C, I]) newContext(ctx context.Context, req *middleware.Request) context.Context {
	if resource.Connection.IsName(req.ResourceName()) {
		ctx = log.NewCtx(ctx, i.log.With(slog.String("connection", req.ResourceName())))
	}
	return middleware.AddRequestToContext(ctx, req)
}

func (i *Integration[C, I]) getInstance(req *middleware.Request) (*instance, error) {
	inst, ok := i.instances.Load(req.HashSHA256())
	if !ok {
		value, err, _ := i.mux.Do(req.HashSHA256(), func() (any, error) {
			cached, ok := i.instances.Load(req.HashSHA256())
			if ok {
				return cached, nil
			}
			return nil, fmt.Errorf("instance not found: %s", req.ResourceName())
		})
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		inst = value.(*instance)
	}

	if req.ConfigHash() != inst.ConfigHash() {
		return nil, status.Error(codes.InvalidArgument, "invalid request: config hash mismatch")
	}

	return inst, nil
}

func (i *Integration[C, I]) closeInstance(reqHash string) (err error) {
	inst, ok := i.instances.Load(reqHash)
	if ok {
		err = inst.close()
		i.instances.Delete(reqHash)
		i.mux.Forget(reqHash)
	}
	return
}

func (i *Integration[C, I]) close() error {
	var errs []error

	if i.sources != nil {
		errs = append(errs, i.sources.Close())
	}

	i.instances.Range(func(key string, inst *instance) bool {
		errs = append(errs, inst.close())
		return true
	})
	return errors.Join(errs...)
}
