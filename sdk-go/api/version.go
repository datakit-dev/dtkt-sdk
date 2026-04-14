package api

import (
	"fmt"
	"slices"
	"strings"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

var validVersions = versions{
	V1Beta1,
	V1Beta2,
}

type (
	Version interface {
		protodesc.Resolver
		protoregistry.ExtensionTypeResolver
		protoregistry.MessageTypeResolver
		FindMethodByName(protoreflect.FullName) (protoreflect.MethodDescriptor, error)
		GetValidator() (protovalidate.Validator, error)
		NumFiles() int
		NumFilesByPackage(name protoreflect.FullName) int
		RangeFiles(f func(protoreflect.FileDescriptor) bool)
		RangeFilesByPackage(name protoreflect.FullName, f func(protoreflect.FileDescriptor) bool)
		RangeMessages(func(protoreflect.MessageType) bool)
		RangeMethods(func(protoreflect.MethodDescriptor) bool)
		RangeServices(func(protoreflect.ServiceDescriptor) bool)
		String() string
	}
	versions []Version
)

func Versions() versions {
	return validVersions
}

func ValidVersion(v string) error {
	if !slices.Contains(validVersions.Strings(), v) {
		return fmt.Errorf("api version must be one of: %s", strings.Join(validVersions.Strings(), ", "))
	}
	return nil
}

func (v versions) Strings() []string {
	return util.SliceMap(v, func(v Version) string {
		return v.String()
	})
}
