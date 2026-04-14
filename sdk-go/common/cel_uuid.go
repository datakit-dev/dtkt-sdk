package common

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/uuid"
)

var _ cel.Library = (*celUUIDLib)(nil)

type celUUIDLib struct{}

func CELUUIDLib() cel.EnvOption {
	return cel.Lib(&celUUIDLib{})
}

func (l *celUUIDLib) LibraryName() string {
	return "uuid"
}

func (l *celUUIDLib) CompileOptions() []cel.EnvOption {
	uuid.NewV7()
	return []cel.EnvOption{
		cel.Function("uuid.new",
			cel.SingletonFunctionBinding(
				func(...ref.Val) ref.Val {
					return types.String(uuid.NewString())
				},
			),
			cel.Overload("uuid.new_string", nil, cel.TimestampType),
		),
		cel.Function("uuid.new_v7",
			cel.SingletonFunctionBinding(
				func(...ref.Val) ref.Val {
					id, err := uuid.NewV7()
					if err != nil {
						return types.WrapErr(err)
					}
					return types.String(id.String())
				},
			),
			cel.Overload("uuid.new_v7_string", nil, cel.TimestampType),
		),
	}
}

func (l *celUUIDLib) ProgramOptions() []cel.ProgramOption { return nil }
