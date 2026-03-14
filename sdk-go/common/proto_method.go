package common

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoMethod struct {
	Pkg            ProtoPackage
	FullName       protoreflect.FullName
	Name           string
	InputFullName  protoreflect.FullName
	InputName      string
	OutputFullName protoreflect.FullName
	OutputName     string
	Unary          bool
	ClientStream   bool
	ServerStream   bool
	BidiStream     bool
}

func NewProtoMethod(pkg ProtoPackage, method protoreflect.MethodDescriptor) ProtoMethod {
	return ProtoMethod{
		Pkg:            pkg,
		FullName:       method.FullName(),
		InputFullName:  method.Input().FullName(),
		OutputFullName: method.Output().FullName(),
		Name:           string(method.Name()),
		InputName:      string(method.Input().Name()),
		OutputName:     string(method.Output().Name()),
		Unary:          !method.IsStreamingClient() && !method.IsStreamingServer(),
		ClientStream:   method.IsStreamingClient() && !method.IsStreamingServer(),
		ServerStream:   !method.IsStreamingClient() && method.IsStreamingServer(),
		BidiStream:     method.IsStreamingClient() && method.IsStreamingServer(),
	}
}

func MethodPathFromName(name protoreflect.FullName) string {
	return fmt.Sprintf("/%s/%s", name.Parent(), name.Name())
}

func MethodNameFromPath(path string) protoreflect.FullName {
	return protoreflect.FullName(strings.TrimPrefix(strings.ReplaceAll(path, "/", "."), "."))
}

func ServiceAndMethodNameFromPath(path string) (protoreflect.FullName, protoreflect.FullName) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		return "", ""
	}
	return protoreflect.FullName(parts[0]), protoreflect.FullName(parts[1])
}
