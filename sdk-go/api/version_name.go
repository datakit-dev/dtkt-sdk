package api

import (
	"path"
	"sync"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/invopop/jsonschema"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	// Well-known types
	_ "google.golang.org/protobuf/types/known/anypb"
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/emptypb"
	_ "google.golang.org/protobuf/types/known/fieldmaskpb"
	_ "google.golang.org/protobuf/types/known/sourcecontextpb"
	_ "google.golang.org/protobuf/types/known/structpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	_ "google.golang.org/protobuf/types/known/typepb"
	_ "google.golang.org/protobuf/types/known/wrapperspb"

	// CoreV1
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"

	// V1Beta1
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/action/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/ai/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/blob/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/catalog/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/command/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/email/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/event/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geo/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/replication/v1beta1"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"

	// V1Beta2
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/catalog/v1beta2"
	_ "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

const (
	CoreV1  = versionName("core.v1")
	V1Beta1 = versionName("v1beta1")
	V1Beta2 = versionName("v1beta2")
)

var _ Version = versionName("")
var versionValidators = map[versionName]protovalidate.Validator{}
var versionMutex sync.Mutex

type versionName string

func (v versionName) GetName() string {
	return string(v)
}

func (v versionName) GetValidator() (validator protovalidate.Validator, err error) {
	versionMutex.Lock()
	defer versionMutex.Unlock()

	validator, ok := versionValidators[v]
	if ok {
		return
	}

	validator, err = protovalidate.New(protovalidate.WithExtensionTypeResolver(v))
	if err != nil {
		return nil, err
	}
	versionValidators[v] = validator

	return
}

func (v versionName) NumFiles() (num int) {
	v.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if IsWellKnownName(fd.FullName()) || VersionContainsName(v, fd.FullName()) {
			num++
		}
		return true
	})
	return
}

func (v versionName) NumFilesByPackage(name protoreflect.FullName) int {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		return getGlobalResolver().NumFilesByPackage(name)
	}
	return 0
}

func (v versionName) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	getGlobalResolver().RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if IsWellKnownName(fd.FullName()) || VersionContainsName(v, fd.FullName()) {
			return f(fd)
		}
		return true
	})
}

func (v versionName) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	return getGlobalResolver().FindFileByPath(path)
}

func (v versionName) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		return getGlobalResolver().FindDescriptorByName(name)
	}
	return nil, protoregistry.NotFound
}

func (v versionName) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		return getGlobalResolver().FindMessageByName(name)
	}
	return nil, protoregistry.NotFound
}

func (v versionName) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionType, error) {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		return getGlobalResolver().FindExtensionByName(name)
	}
	return nil, protoregistry.NotFound
}

func (v versionName) FindExtensionByNumber(name protoreflect.FullName, num protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		return getGlobalResolver().FindExtensionByNumber(name, num)
	}
	return nil, protoregistry.NotFound
}

func (v versionName) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	name := path.Base(url)
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		return getGlobalResolver().FindMessageByURL(url)
	}
	return nil, protoregistry.NotFound
}

func (v versionName) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		return getGlobalResolver().FindMethodByName(name)
	}
	return nil, protoregistry.NotFound
}

func (v versionName) RangeFilesByPackage(name protoreflect.FullName, f func(protoreflect.FileDescriptor) bool) {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		getGlobalResolver().RangeFilesByPackage(name, f)
	}
}

func (v versionName) RangeMessages(f func(protoreflect.MessageType) bool) {
	getGlobalResolver().RangeMessages(func(mt protoreflect.MessageType) bool {
		if IsWellKnownName(mt.Descriptor().FullName()) || VersionContainsName(v, mt.Descriptor().FullName()) {
			return f(mt)
		}
		return true
	})
}

func (v versionName) RangeServices(f func(protoreflect.ServiceDescriptor) bool) {
	getGlobalResolver().RangeServices(func(sd protoreflect.ServiceDescriptor) bool {
		if IsWellKnownName(sd.FullName()) || VersionContainsName(v, sd.FullName()) {
			return f(sd)
		}
		return true
	})
}

func (v versionName) RangeMethods(f func(protoreflect.MethodDescriptor) bool) {
	getGlobalResolver().RangeMethods(func(md protoreflect.MethodDescriptor) bool {
		if IsWellKnownName(md.FullName()) || VersionContainsName(v, md.FullName()) {
			return f(md)
		}
		return true
	})
}

func (v versionName) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumType, error) {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		return getGlobalResolver().FindEnumByName(name)
	}
	return nil, protoregistry.NotFound
}

func (v versionName) FindServiceByName(name protoreflect.FullName) (protoreflect.ServiceDescriptor, error) {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		return getGlobalResolver().FindServiceByName(name)
	}
	return nil, protoregistry.NotFound
}

func (v versionName) RangeEnums(f func(protoreflect.EnumType) bool) {
	getGlobalResolver().RangeEnums(func(et protoreflect.EnumType) bool {
		if IsWellKnownName(et.Descriptor().FullName()) || VersionContainsName(v, et.Descriptor().FullName()) {
			return f(et)
		}
		return true
	})
}

func (v versionName) RangeExtensionsByMessage(name protoreflect.FullName, f func(protoreflect.ExtensionType) bool) {
	if IsWellKnownName(name) || VersionContainsName(v, name) {
		getGlobalResolver().RangeExtensionsByMessage(name, f)
	}
}

func (versionName) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Title:       "API Version",
		Description: "A valid DataKit SDK API Version.",
		Enum:        util.AnySlice(Versions().Strings()),
	}
}

func (v versionName) String() string {
	return string(v)
}
