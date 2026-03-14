package api

// import (
// 	"fmt"
// 	"path"
// 	"slices"
// 	"strings"

// 	"buf.build/go/protovalidate"
// 	"google.golang.org/protobuf/reflect/protoreflect"
// 	"google.golang.org/protobuf/reflect/protoregistry"
// )

// type versionGroup string

// const V1Beta = versionGroup("v1beta")

// func (v versionGroup) Package() string {
// 	return versionPackagePrefix + v.String()
// }

// func (v versionGroup) Import() string {
// 	return fmt.Sprintf(versionImportFormat, v.Package())
// }

// func (v versionGroup) ContainsName(name protoreflect.FullName) bool {
// 	return slices.ContainsFunc(validVersions, func(version Version) bool {
// 		return strings.HasPrefix(version.String(), v.String()) && VersionContainsName(version, name)
// 	})
// }

// func (v versionGroup) NumFiles() (num int) {
// 	v.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
// 		if IsWellKnownName(fd.FullName()) || v.ContainsName(fd.FullName()) {
// 			num++
// 		}
// 		return true
// 	})
// 	return
// }

// func (v versionGroup) NumFilesByPackage(name protoreflect.FullName) int {
// 	if IsWellKnownName(name) || v.ContainsName(name) {
// 		return getGlobalResolver().NumFilesByPackage(name)
// 	}
// 	return 0
// }

// func (v versionGroup) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
// 	getGlobalResolver().RangeFiles(func(fd protoreflect.FileDescriptor) bool {
// 		if IsWellKnownName(fd.FullName()) || v.ContainsName(fd.FullName()) {
// 			return f(fd)
// 		}
// 		return true
// 	})
// }

// func (v versionGroup) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
// 	return getGlobalResolver().FindFileByPath(path)
// }

// func (v versionGroup) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
// 	if IsWellKnownName(name) || v.ContainsName(name) {
// 		return getGlobalResolver().FindDescriptorByName(name)
// 	}
// 	return nil, protoregistry.NotFound
// }

// func (v versionGroup) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
// 	if IsWellKnownName(name) || v.ContainsName(name) {
// 		return getGlobalResolver().FindMessageByName(name)
// 	}
// 	return nil, protoregistry.NotFound
// }

// func (v versionGroup) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionType, error) {
// 	if IsWellKnownName(name) || v.ContainsName(name) {
// 		return getGlobalResolver().FindExtensionByName(name)
// 	}
// 	return nil, protoregistry.NotFound
// }

// func (v versionGroup) FindExtensionByNumber(name protoreflect.FullName, num protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
// 	if IsWellKnownName(name) || v.ContainsName(name) {
// 		return getGlobalResolver().FindExtensionByNumber(name, num)
// 	}
// 	return nil, protoregistry.NotFound
// }

// func (v versionGroup) FindMessageByURL(url string) (protoreflect.MessageType, error) {
// 	name := path.Base(url)
// 	if IsWellKnownName(name) || v.ContainsName(protoreflect.FullName(name)) {
// 		return getGlobalResolver().FindMessageByURL(url)
// 	}
// 	return nil, protoregistry.NotFound
// }

// func (v versionGroup) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
// 	if IsWellKnownName(name) || v.ContainsName(name) {
// 		return getGlobalResolver().FindMethodByName(name)
// 	}
// 	return nil, protoregistry.NotFound
// }

// func (v versionGroup) RangeFilesByPackage(name protoreflect.FullName, f func(protoreflect.FileDescriptor) bool) {
// 	if IsWellKnownName(name) || v.ContainsName(name) {
// 		getGlobalResolver().RangeFilesByPackage(name, f)
// 	}
// }

// func (v versionGroup) RangeMessages(f func(protoreflect.MessageType) bool) {
// 	getGlobalResolver().RangeMessages(func(mt protoreflect.MessageType) bool {
// 		if IsWellKnownName(mt.Descriptor().FullName()) || v.ContainsName(mt.Descriptor().FullName()) {
// 			return f(mt)
// 		}
// 		return true
// 	})
// }

// func (v versionGroup) RangeServices(f func(protoreflect.ServiceDescriptor) bool) {
// 	getGlobalResolver().RangeServices(func(sd protoreflect.ServiceDescriptor) bool {
// 		if IsWellKnownName(sd.FullName()) || v.ContainsName(sd.FullName()) {
// 			return f(sd)
// 		}
// 		return true
// 	})
// }

// func (v versionGroup) RangeMethods(f func(protoreflect.MethodDescriptor) bool) {
// 	getGlobalResolver().RangeMethods(func(md protoreflect.MethodDescriptor) bool {
// 		if IsWellKnownName(md.FullName()) || v.ContainsName(md.FullName()) {
// 			return f(md)
// 		}
// 		return true
// 	})
// }

// func (v versionGroup) GetValidator() (protovalidate.Validator, error) {
// 	return GlobalValidator()
// }

// func (v versionGroup) String() string {
// 	return string(v)
// }

// func (v versionGroup) IsGroup() {}
