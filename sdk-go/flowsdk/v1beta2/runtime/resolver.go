package runtime

import (
	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
)

// flowUnionResolver is the flow-global type-resolution surface: the
// explicit union of the run's declared-connector resolvers (in spec
// order) followed by a platform-types resolver (api.GlobalResolver() by
// default - the SDK's named, version-bound platform layer; what v1beta1
// uses as its default resolver in `flowsdk/v1beta1/runtime/env.go`).
// Built once per Execute by buildCELEnv and shared by every in-process
// component that converts/decodes proto values: runtimeEnv.resolver AND
// every per-action handler env. One resolver, one explicit type universe
// per run.
//
// Why a hand-written union instead of upstream `protoresolve.Combine`:
// `protoresolve.Combine` returns a `protoresolve.Resolver` (operates on
// protoreflect.MessageDescriptor / protoreflect.ExtensionDescriptor),
// whereas this SDK's `shared.Resolver` interface uses the std
// protoregistry.MessageTypeResolver / ExtensionTypeResolver
// (protoreflect.MessageType / ExtensionType). Adapting in both
// directions would not be smaller. We mirror Combine's exact iteration
// semantics (NotFound fallthrough; path-dedupe on RangeFiles) and keep
// the type compatible with `shared.Resolver` directly.
//
// Why spec-ordered: deterministic precedence. Iteration in Go-map order
// would be nondeterministic across runs. Spec order also matches a
// reader's mental model of "the first connection wins."
//
// Why explicit platform layer (not protoregistry.GlobalTypes): the
// platform types are a deliberate, named input - travels with the SDK,
// survives a future per-runtime-boundary deployment (a node in its own
// pod still imports the SDK and gets api.GlobalResolver() the same way),
// is testable.
type flowUnionResolver struct {
	connectors []shared.Resolver
	platform   shared.Resolver
}

// newFlowUnionResolver builds the union from spec-ordered connectors +
// the platform resolver. `orderedConnectors` MUST be in the flow spec's
// declared connection order (the caller orders from graph/spec); this
// type does not re-order. nil entries are skipped. If `platform` is nil
// the caller has not threaded the WithPlatformResolver default; that is
// a wiring bug at the buildCELEnv level, not a runtime concern handled
// here.
func newFlowUnionResolver(orderedConnectors []*rpc.Connector, platform shared.Resolver) *flowUnionResolver {
	rs := make([]shared.Resolver, 0, len(orderedConnectors))
	for _, c := range orderedConnectors {
		if c != nil && c.Resolver != nil {
			rs = append(rs, c.Resolver)
		}
	}
	return &flowUnionResolver{connectors: rs, platform: platform}
}

// findOne iterates members trying `find` until one returns nil error
// (NotFound triggers fallthrough; any other error short-circuits and
// propagates). Mirrors the protoresolve.Combine semantics.
func findOne[T any](members []shared.Resolver, platform shared.Resolver, find func(shared.Resolver) (T, error)) (T, error) {
	for _, c := range members {
		v, err := find(c)
		if err == nil {
			return v, nil
		}
		// Any non-NotFound error propagates (e.g. transient resolver
		// failure should not be silently masked by NotFound semantics).
		if !errIsNotFound(err) {
			return v, err
		}
	}
	return find(platform)
}

func errIsNotFound(err error) bool {
	return err == protoregistry.NotFound || (err != nil && err.Error() == protoregistry.NotFound.Error())
}

func (r *flowUnionResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	return findOne(r.connectors, r.platform, func(c shared.Resolver) (protoreflect.MessageType, error) {
		return c.FindMessageByName(name)
	})
}

func (r *flowUnionResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return findOne(r.connectors, r.platform, func(c shared.Resolver) (protoreflect.MessageType, error) {
		return c.FindMessageByURL(url)
	})
}

func (r *flowUnionResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return findOne(r.connectors, r.platform, func(c shared.Resolver) (protoreflect.ExtensionType, error) {
		return c.FindExtensionByName(field)
	})
}

func (r *flowUnionResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return findOne(r.connectors, r.platform, func(c shared.Resolver) (protoreflect.ExtensionType, error) {
		return c.FindExtensionByNumber(message, field)
	})
}

func (r *flowUnionResolver) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	return findOne(r.connectors, r.platform, func(c shared.Resolver) (protoreflect.MethodDescriptor, error) {
		return c.FindMethodByName(name)
	})
}

// rangeAll iterates `each(c) bool` over all members (connectors then
// platform). Stops as soon as one returns false. The callback `each`
// must return true to continue, false to stop.
func rangeAll(members []shared.Resolver, platform shared.Resolver, each func(shared.Resolver) bool) {
	for _, c := range members {
		if !each(c) {
			return
		}
	}
	_ = each(platform)
}

func (r *flowUnionResolver) RangeServices(fn func(protoreflect.ServiceDescriptor) bool) {
	rangeAll(r.connectors, r.platform, func(c shared.Resolver) bool {
		keep := true
		c.RangeServices(func(sd protoreflect.ServiceDescriptor) bool {
			if !fn(sd) {
				keep = false
				return false
			}
			return true
		})
		return keep
	})
}

func (r *flowUnionResolver) RangeMethods(fn func(protoreflect.MethodDescriptor) bool) {
	rangeAll(r.connectors, r.platform, func(c shared.Resolver) bool {
		keep := true
		c.RangeMethods(func(md protoreflect.MethodDescriptor) bool {
			if !fn(md) {
				keep = false
				return false
			}
			return true
		})
		return keep
	})
}

// RangeFiles dedupes by file path. Two connectors (or a connector and
// the platform) sharing google/protobuf/struct.proto would otherwise
// yield it twice; downstream common.NewCELTypes registers each file
// into a cel-go types.Registry (RegisterDescriptor) which errors on
// duplicate registration. Mirrors protoresolve.Combine's RangeFiles
// dedupe semantics.
func (r *flowUnionResolver) RangeFiles(fn func(protoreflect.FileDescriptor) bool) {
	seen := make(map[string]struct{})
	rangeAll(r.connectors, r.platform, func(c shared.Resolver) bool {
		keep := true
		c.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			if _, dup := seen[fd.Path()]; dup {
				return true
			}
			seen[fd.Path()] = struct{}{}
			if !fn(fd) {
				keep = false
				return false
			}
			return true
		})
		return keep
	})
}

// GetValidator returns a protovalidate.Validator that can validate any
// proto type known to the union, by composing through this resolver as
// the extension type resolver. This is the established SDK pattern
// (api/validator.go, api/version.go, flowsdk/v1beta2/input_type.go) and
// the correct semantics: protovalidate needs the types of the messages
// it validates, and those may come from any member of the union.
// First-connector-wins would be wrong - a validator that only knows one
// connector cannot validate a message from another connector.
func (r *flowUnionResolver) GetValidator() (protovalidate.Validator, error) {
	return protovalidate.New(protovalidate.WithExtensionTypeResolver(r))
}
