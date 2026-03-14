package common

import (
	"slices"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ProtoFilesHelper []protoreflect.FileDescriptor

func (f ProtoFilesHelper) MessageDescs(skip ...string) (descs []protoreflect.MessageDescriptor) {
	for _, file := range f {
		for msgIdx := range file.Messages().Len() {
			desc := file.Messages().Get(msgIdx)
			name := string(desc.FullName())
			if !slices.Contains(skip, name) {
				descs = append(descs, desc)
				skip = append(skip, name)
			}
		}
	}
	return
}

func (f ProtoFilesHelper) EnumDescs(skip ...string) (descs []protoreflect.EnumDescriptor) {
	for _, file := range f {
		for enumIdx := range file.Enums().Len() {
			enum := file.Enums().Get(enumIdx)
			name := string(enum.FullName())
			if !slices.Contains(skip, name) {
				descs = append(descs, enum)
				skip = append(skip, name)
			}
		}
	}
	return
}

func (f ProtoFilesHelper) MessageNames(skip ...string) (names []string) {
	for _, file := range f {
		for msgIdx := range file.Messages().Len() {
			name := string(file.Messages().Get(msgIdx).FullName())
			if !slices.Contains(skip, name) {
				names = append(names, name)
				skip = append(skip, name)
			}
		}
	}
	return
}

func (f ProtoFilesHelper) MessageFullNames(skip ...string) (names []string) {
	for _, file := range f {
		for msgIdx := range file.Messages().Len() {
			name := string(file.Messages().Get(msgIdx).FullName())
			if !slices.Contains(skip, name) {
				names = append(names, name)
				skip = append(skip, name)
			}
		}
	}
	return
}

func (f ProtoFilesHelper) EnumFullNames(skip ...string) (names []string) {
	for _, file := range f {
		for idx := range file.Enums().Len() {
			name := string(file.Enums().Get(idx).FullName())
			if !slices.Contains(skip, name) {
				names = append(names, name)
				skip = append(skip, name)
			}
		}
	}
	return
}

func (f ProtoFilesHelper) EnumNames(skip ...string) (names []string) {
	for _, file := range f {
		for idx := range file.Enums().Len() {
			name := string(file.Enums().Get(idx).FullName())
			if !slices.Contains(skip, name) {
				names = append(names, name)
				skip = append(skip, name)
			}
		}
	}
	return
}

func (f ProtoFilesHelper) ServiceDescs(skip ...string) (descs []protoreflect.ServiceDescriptor) {
	for _, file := range f {
		for svcIdx := range file.Services().Len() {
			desc := file.Services().Get(svcIdx)
			name := string(desc.FullName())
			if !slices.Contains(skip, name) {
				descs = append(descs, desc)
				skip = append(skip, name)
			}
		}
	}
	return
}

func (f ProtoFilesHelper) ServiceFullNames(skip ...string) (names []string) {
	for _, file := range f {
		for svcIdx := range file.Services().Len() {
			name := string(file.Services().Get(svcIdx).FullName())
			if !slices.Contains(skip, name) {
				names = append(names, name)
				skip = append(skip, name)
			}
		}
	}
	return
}

func (f ProtoFilesHelper) ServiceNames(skip ...string) (names []string) {
	for _, file := range f {
		for svcIdx := range file.Services().Len() {
			name := string(file.Services().Get(svcIdx).Name())
			if !slices.Contains(skip, name) {
				names = append(names, name)
				skip = append(skip, name)
			}
		}
	}
	return
}

func (f ProtoFilesHelper) ServiceMethods(skip ...string) (methods []ProtoMethod) {
	for _, file := range f {
		for svcIdx := range file.Services().Len() {
			svc := file.Services().Get(svcIdx)
			for methodIdx := range svc.Methods().Len() {
				method := svc.Methods().Get(methodIdx)
				if !slices.Contains(skip, string(method.FullName())) {
					methods = append(methods, NewProtoMethod(ProtoPackage(svc.ParentFile().Package()), method))
					skip = append(skip, string(method.FullName()))
				}
			}
		}
	}
	return
}

func (f ProtoFilesHelper) MethodNames(skip ...string) (names []string) {
	for _, file := range f {
		for svcIdx := range file.Services().Len() {
			for methodIdx := range file.Services().Get(svcIdx).Methods().Len() {
				method := file.Services().Get(svcIdx).Methods().Get(methodIdx)
				if !slices.Contains(skip, string(method.FullName())) {
					names = append(names, string(method.Name()))
					skip = append(skip, string(method.FullName()))
				}
			}
		}
	}
	return
}

func (f ProtoFilesHelper) ServiceMethodMap(skip ...string) (svcMethodMap map[ProtoPackage]map[string][]ProtoMethod) {
	svcMethodMap = map[ProtoPackage]map[string][]ProtoMethod{}
	for _, file := range f {
		if file.Services().Len() > 0 {
			pkg := ProtoPackage(file.Package())
			if svcMethodMap[pkg] == nil {
				svcMethodMap[pkg] = map[string][]ProtoMethod{}
			}

			for svcIdx := range file.Services().Len() {
				svc := file.Services().Get(svcIdx)

				for methodIdx := range svc.Methods().Len() {
					method := svc.Methods().Get(methodIdx)
					if !slices.Contains(skip, string(method.FullName())) {
						svcMethodMap[pkg][string(svc.Name())] = append(svcMethodMap[pkg][string(svc.Name())], NewProtoMethod(pkg, method))
						skip = append(skip, string(method.FullName()))
					}
				}
			}
		}
	}
	return
}

func (f ProtoFilesHelper) FindMessageType(name string) (protoreflect.MessageType, bool) {
	for desc := range slices.Values(f) {
		for msgIdx := range desc.Messages().Len() {
			if name == string(desc.Messages().Get(msgIdx).Name()) || name == string(desc.Messages().Get(msgIdx).FullName()) {
				return dynamicpb.NewMessageType(desc.Messages().Get(msgIdx)), true
			}
		}
	}
	return nil, false
}

func (f ProtoFilesHelper) FindEnumType(name string) (protoreflect.EnumType, bool) {
	for desc := range slices.Values(f) {
		for idx := range desc.Enums().Len() {
			if name == string(desc.Enums().Get(idx).Name()) || name == string(desc.Enums().Get(idx).FullName()) {
				return dynamicpb.NewEnumType(desc.Enums().Get(idx)), true
			}
		}
	}
	return nil, false
}

func (f ProtoFilesHelper) FindMethodDesc(name string) (protoreflect.MethodDescriptor, bool) {
	for desc := range slices.Values(f) {
		for svcIdx := range desc.Services().Len() {
			for methodIdx := range desc.Services().Get(svcIdx).Methods().Len() {
				var method = desc.Services().Get(svcIdx).Methods().Get(methodIdx)
				if (strings.HasPrefix(name, string(desc.FullName())) && strings.HasSuffix(name, string(method.Name()))) || string(method.FullName()) == name {
					return method, true
				}
			}
		}
	}
	return nil, false
}

func (f ProtoFilesHelper) MessageTypes(skip ...string) (types []protoreflect.MessageType) {
	for desc := range slices.Values(f) {
		for msgIdx := range desc.Messages().Len() {
			if !slices.Contains(skip, string(desc.Messages().Get(msgIdx).FullName())) {
				types = append(types, dynamicpb.NewMessageType(desc.Messages().Get(msgIdx)))
			}
		}
	}
	return
}

func (f ProtoFilesHelper) MethodRequestName(method string) string {
	for desc := range slices.Values(f) {
		for svcIdx := range desc.Services().Len() {
			for methodIdx := range desc.Services().Get(svcIdx).Methods().Len() {
				if method == string(desc.Services().Get(svcIdx).Methods().Get(methodIdx).FullName()) {
					return string(desc.Services().Get(svcIdx).Methods().Get(methodIdx).Input().FullName())
				}
			}
		}
	}
	return ""
}

func (f ProtoFilesHelper) MethodResponseName(method string) string {
	for desc := range slices.Values(f) {
		for svcIdx := range desc.Services().Len() {
			for methodIdx := range desc.Services().Get(svcIdx).Methods().Len() {
				if method == string(desc.Services().Get(svcIdx).Methods().Get(methodIdx).FullName()) {
					return string(desc.Services().Get(svcIdx).Methods().Get(methodIdx).Output().FullName())
				}
			}
		}
	}
	return ""
}

func (f ProtoFilesHelper) UnaryMethodNames(skip ...string) (names []string) {
	for desc := range slices.Values(f) {
		for svcIdx := range desc.Services().Len() {
			for methodIdx := range desc.Services().Get(svcIdx).Methods().Len() {
				var method = desc.Services().Get(svcIdx).Methods().Get(methodIdx)
				if !slices.Contains(skip, string(method.FullName())) && !method.IsStreamingClient() && !method.IsStreamingServer() {
					names = append(names, string(method.FullName()))
				}
			}
		}
	}
	return
}

func (f ProtoFilesHelper) ClientStreamMethodNames(skip ...string) (names []string) {
	for desc := range slices.Values(f) {
		for svcIdx := range desc.Services().Len() {
			for methodIdx := range desc.Services().Get(svcIdx).Methods().Len() {
				var method = desc.Services().Get(svcIdx).Methods().Get(methodIdx)
				if !slices.Contains(skip, string(method.FullName())) && method.IsStreamingClient() && !method.IsStreamingServer() {
					names = append(names, string(method.FullName()))
				}
			}
		}
	}
	return
}

func (f ProtoFilesHelper) ServerStreamMethodNames(skip ...string) (names []string) {
	for desc := range slices.Values(f) {
		for svcIdx := range desc.Services().Len() {
			for methodIdx := range desc.Services().Get(svcIdx).Methods().Len() {
				var method = desc.Services().Get(svcIdx).Methods().Get(methodIdx)
				if !slices.Contains(skip, string(method.FullName())) && !method.IsStreamingClient() && method.IsStreamingServer() {
					names = append(names, string(method.FullName()))
				}
			}
		}
	}
	return
}

func (f ProtoFilesHelper) BidiStreamMethodNames(skip ...string) (names []string) {
	for desc := range slices.Values(f) {
		for svcIdx := range desc.Services().Len() {
			for methodIdx := range desc.Services().Get(svcIdx).Methods().Len() {
				var method = desc.Services().Get(svcIdx).Methods().Get(methodIdx)
				if !slices.Contains(skip, string(method.FullName())) && method.IsStreamingClient() && method.IsStreamingServer() {
					names = append(names, string(method.FullName()))
				}
			}
		}
	}
	return
}
