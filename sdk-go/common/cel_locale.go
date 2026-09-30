package common

import (
	"reflect"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var _ cel.Library = (*celLocaleLib)(nil)
var anySlice = reflect.TypeFor[[]any]()

type (
	celLocaleLib struct {
		lang    language.Tag
		printer *message.Printer
	}
	celLocaleOpt func(*celLocaleLib)
)

func CELLocaleLib(opts ...celLocaleOpt) cel.EnvOption {
	lib := &celLocaleLib{}
	for _, opt := range opts {
		if opt != nil {
			opt(lib)
		}
	}

	if lib.lang == language.Und {
		lib.lang = language.AmericanEnglish
	}

	lib.printer = message.NewPrinter(lib.lang)

	return cel.Lib(lib)
}

func WithCELLocaleTag(tag language.Tag) celLocaleOpt {
	return func(lib *celLocaleLib) {
		lib.lang = tag
	}
}

func WithCELLocale(locale string) celLocaleOpt {
	return func(lib *celLocaleLib) {
		tag, err := language.Parse(locale)
		if err == nil {
			lib.lang = tag
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

					return types.String(l.printer.Sprintf(args[0].Value(), vals.([]any)...))
				}),
			),
		),
	}
}

func (l *celLocaleLib) ProgramOptions() []cel.ProgramOption {
	return nil
}
