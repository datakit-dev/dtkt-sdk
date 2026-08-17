package common

import (
	"fmt"
	"reflect"
	"uuid"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ cel.Library = (*celUUIDLib)(nil)
var intType = reflect.TypeFor[int]()

type celUUIDLib struct{}

func CELUUIDLib() cel.EnvOption {
	return cel.Lib(&celUUIDLib{})
}

func (l *celUUIDLib) LibraryName() string {
	return "uuid"
}

func (l *celUUIDLib) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("uuid.new",
			cel.Overload("uuid_string", nil, cel.StringType,
				cel.FunctionBinding(func(...ref.Val) ref.Val {
					return types.String(uuid.New().String())
				}),
			),
			cel.Overload("uuid_version_string", []*cel.Type{cel.IntType}, cel.StringType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					version, err := args[0].ConvertToNative(intType)
					if err != nil {
						return types.WrapErr(err)
					}

					var id uuid.UUID
					switch version {
					case 4:
						id = uuid.NewV4()
					case 7:
						id = uuid.NewV7()
					default:
						return types.WrapErr(fmt.Errorf("unsupported uuid version: %d", version))
					}

					return types.String(id.String())
				}),
			),
		),
	}
}

func (l *celUUIDLib) ProgramOptions() []cel.ProgramOption { return nil }
