package v1beta1

import (
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/resource"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/santhosh-tekuri/jsonschema/v6"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	ListTypesDefaultPageSize int32 = 25
	ListTypesMinPageSize     int32 = 1
	ListTypesMaxPageSize     int32 = 1000
)

var (
	defaultTypeRegistry = sync.OnceValue(func() *TypeRegistry {
		return NewTypeRegistry(defaultTypeSyncer(), defaultTypeBaseUri())
	})
	defaultTypeSyncer = sync.OnceValue(func() *MemoryTypeSyncer {
		return &MemoryTypeSyncer{}
	})
	defaultTypeBaseUri = sync.OnceValue(func() *url.URL {
		return &url.URL{
			Scheme: "memory",
			Path:   "/types",
		}
	})
)

type (
	// TypeRegistry contains type schemas which may or may not be native protobuf
	// types. Each entry contains a resolved type schema.
	TypeRegistry struct {
		syncer   TypeSyncer
		compiler *jsonschema.Compiler
		baseUri  *url.URL

		protoGen bool
		protoPkg string
	}
	// TypeSyncer defines the interface for storing and retrieving type schemas.
	TypeSyncer interface {
		// StoreType stores a type schema and must be retrievable through get and list.
		StoreType(*sharedv1beta1.TypeSchema) error
		// GetType returns a type schema by given fully qualified type name.
		GetType(fullName string) (*sharedv1beta1.TypeSchema, error)
		// ListTypes returns a list of type schemas modified after given index and mod
		// time with length <= given page size.
		ListTypes(lastIdx int64, modTime time.Time, pageSize int32) ([]*sharedv1beta1.TypeSchema, error)
	}
	ProtoResolver interface {
		RangeMessages(func(protoreflect.MessageType) bool)
	}
	MemoryTypeSyncer struct {
		util.SyncMap[string, *sharedv1beta1.TypeSchema]
	}
	TypeRegistryOption func(*TypeRegistry)
)

func NewTypeRegistry(syncer TypeSyncer, baseUri *url.URL, opts ...TypeRegistryOption) *TypeRegistry {
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(common.JSONSchemaLoaderFunc(func(uri string) (any, error) {
		schemaUri, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}

		if !strings.HasPrefix(schemaUri.String(), baseUri.String()) {
			uri = baseUri.JoinPath(schemaUri.Path).String()
		}

		if !strings.HasSuffix(uri, JSONSchemaFileExt) {
			uri += JSONSchemaFileExt
		}

		path, err := TypePathFromUri(uri)
		if err != nil {
			return nil, err
		}

		schema, err := syncer.GetType(path)
		if err != nil {
			return nil, err
		}

		return schema.JsonSchema.AsMap(), nil
	}))

	reg := &TypeRegistry{
		syncer:   syncer,
		compiler: compiler,
		baseUri:  baseUri,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(reg)
		}
	}

	return reg
}

func WithRuntimeProtoGen(protoPkg string) TypeRegistryOption {
	return func(r *TypeRegistry) {
		if resource.ValidProtoNameRegex().MatchString(protoPkg) {
			r.protoGen = true
			r.protoPkg = protoPkg
		}
	}
}

func DefaultTypeRegistry() *TypeRegistry {
	return defaultTypeRegistry()
}

func DefaultTypeSyncer() *MemoryTypeSyncer {
	return defaultTypeSyncer()
}

func (r *TypeRegistry) BaseUri() *url.URL {
	baseUri := *r.baseUri
	return &baseUri
}

func (r *TypeRegistry) Syncer() TypeSyncer {
	return r.syncer
}

func (r *TypeRegistry) Compiler() *jsonschema.Compiler {
	return r.compiler
}

func (r *TypeRegistry) LoadResolverTypes(resolver ProtoResolver) (err error) {
	resolver.RangeMessages(func(mt protoreflect.MessageType) bool {
		_, err = NewTypeSchemaForProto(r, mt.New().Interface())
		return err == nil
	})
	return
}

func (r *TypeRegistry) ListTypes(req *basev1beta1.ListTypesRequest) (*basev1beta1.ListTypesResponse, error) {
	lastIdx, modTime, pageSize, err := util.ParsePageTokenRequest(req, ListTypesDefaultPageSize, ListTypesMinPageSize, ListTypesMaxPageSize)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "page token: %s", err)
	}

	types, err := r.syncer.ListTypes(lastIdx, modTime, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "list types: %s", err)
	}

	var nextPage string
	if len(types) == int(pageSize) {
		last := types[len(types)-1]
		nextPage, err = util.NextPageToken(lastIdx+int64(len(types)), last.ModTime.AsTime())
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "next page token: %s", err)
		}
	}

	return &basev1beta1.ListTypesResponse{
		Types:         types,
		NextPageToken: nextPage,
	}, nil
}

func (r *TypeRegistry) GetType(req *basev1beta1.GetTypeRequest) (*basev1beta1.GetTypeResponse, error) {
	typ, err := r.syncer.GetType(req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &basev1beta1.GetTypeResponse{
		Type: typ,
	}, nil
}

func (l *MemoryTypeSyncer) ListTypes(_ int64, modTime time.Time, pageSize int32) (types []*sharedv1beta1.TypeSchema, _ error) {
	allTypes := l.Values()
	slices.SortFunc(allTypes, func(a, b *sharedv1beta1.TypeSchema) int {
		return a.ModTime.AsTime().Compare(b.ModTime.AsTime())
	})

	for _, typ := range allTypes {
		if len(types) < int(pageSize) && typ.ModTime.AsTime().After(modTime) {
			types = append(types, typ)
		}
	}

	return
}

func (l *MemoryTypeSyncer) GetType(name string) (*sharedv1beta1.TypeSchema, error) {
	typ, ok := l.Load(path.Base(name))
	if !ok {
		return nil, fmt.Errorf("type not found: %s", name)
	}
	return typ, nil
}

func (l *MemoryTypeSyncer) StoreType(typ *sharedv1beta1.TypeSchema) error {
	name, err := TypeNameFromUri(typ.GetUri())
	if err != nil {
		return err
	}

	l.Store(name, typ)
	return nil
}
