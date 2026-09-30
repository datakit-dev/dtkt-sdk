package common

import (
	"reflect"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"google.golang.org/protobuf/proto"
)

var _ cel.Library = (*celLocaleLib)(nil)
var anySlice = reflect.TypeFor[[]any]()

type (
	celLocaleLib struct {
		FormatProtoOptions
	}
	celLocaleOpt func(*celLocaleLib)
)

func CELLocaleLib(opts ...celLocaleOpt) cel.EnvOption {
	return cel.Lib(NewCELLocal(opts...))
}

func NewCELLocal(opts ...celLocaleOpt) *celLocaleLib {
	lib := &celLocaleLib{
		FormatProtoOptions: defaultFormatProtoOptions,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(lib)
		}
	}

	if lib.locale == language.Und {
		lib.locale = language.AmericanEnglish
	}

	if lib.printer == nil {
		lib.printer = message.NewPrinter(lib.locale)
	}

	return lib
}

func WithCELLocaleTag(tag language.Tag) celLocaleOpt {
	return func(lib *celLocaleLib) {
		lib.locale = tag
	}
}

func WithCELLocalePrinter(printer *message.Printer) celLocaleOpt {
	return func(lib *celLocaleLib) {
		lib.printer = printer
	}
}

func WithCELLocale(locale string) celLocaleOpt {
	return func(lib *celLocaleLib) {
		tag, err := language.Parse(locale)
		if err == nil {
			lib.locale = tag
		}
	}
}

func (l *celLocaleLib) LibraryName() string {
	return "locale"
}

func (l *celLocaleLib) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("locale.format",
			cel.Overload("locale.format_dyn",
				[]*cel.Type{cel.StringType, cel.ListType(cel.DynType)},
				cel.StringType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					if len(args) != 2 {
						return types.NewErr(`expected format("{format}", [values...])`)
					}

					vals, err := args[1].ConvertToNative(anySlice)
					if err != nil {
						return types.WrapErr(err)
					}

					return types.String(l.Sprintf(args[0].Value(), vals.([]any)...))
				}),
			),
		),
	}
}

func (l *celLocaleLib) Sprintf(key message.Reference, vals ...any) string {
	for idx, val := range vals {
		if msg, ok := val.(proto.Message); ok {
			str, err := l.Format(msg)
			if err == nil {
				vals[idx] = str
			}
		}
	}

	return l.printer.Sprintf(key, vals...)
}

func (l *celLocaleLib) ProgramOptions() []cel.ProgramOption {
	return nil
}
