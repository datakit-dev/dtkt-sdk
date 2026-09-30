package common

import (
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/jhump/protoreflect/v2/protoprint"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ cel.Library = (*celDescriptorLib)(nil)

type (
	celDescriptorLib struct {
		printer protoprint.Printer
	}
)

func CELDescriptorLib() cel.EnvOption {
	return cel.Lib(&celDescriptorLib{})
}

func (l *celDescriptorLib) LibraryName() string {
	return "protodesc"
}

func (l *celDescriptorLib) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("protodesc.print",
			cel.Overload("protodesc.print_string",
				[]*cel.Type{cel.DynType},
				cel.StringType,
				cel.UnaryBinding(l.protoPrint),
			),
		),
	}
}

func (l *celDescriptorLib) protoPrint(value ref.Val) ref.Val {
	var desc protoreflect.Descriptor
	switch value := value.(type) {
	case *celEnum:
		desc = value.desc
	case *celMessage:
		desc = value.msg.ProtoReflect().Descriptor()
	default:
		return types.WrapErr(fmt.Errorf("argument must be a proto enum or message"))
	}

	str, err := l.printer.PrintProtoToString(desc)
	if err != nil {
		return types.WrapErr(err)
	}

	return types.String(str)
}

func (l *celDescriptorLib) ProgramOptions() []cel.ProgramOption { return nil }
