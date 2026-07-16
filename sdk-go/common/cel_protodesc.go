package common

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/jhump/protoreflect/v2/protoprint"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ cel.Library = (*celProtoDescLib)(nil)

type (
	celProtoDescLib struct {
		printer protoprint.Printer
	}
	celProtoDescLibOption func(*celProtoDescLib)
)

func CELProtoDescLib(opts ...celProtoDescLibOption) cel.EnvOption {
	lib := &celProtoDescLib{}

	for _, opt := range opts {
		if opt != nil {
			opt(lib)
		}
	}

	return cel.Lib(lib)
}

func (l *celProtoDescLib) LibraryName() string {
	return "protodesc"
}

func (l *celProtoDescLib) CompileOptions() []cel.EnvOption {
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

func (l *celProtoDescLib) protoPrint(value ref.Val) ref.Val {
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

func (l *celProtoDescLib) ProgramOptions() []cel.ProgramOption { return nil }
