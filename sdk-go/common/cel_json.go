package common

import (
	"reflect"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ cel.Library = (*celJSONLib)(nil)
var stringType = reflect.TypeFor[string]()
var bytesType = reflect.TypeFor[[]byte]()

type celJSONLib struct{}

func CELJSONLib() cel.EnvOption {
	return cel.Lib(&celJSONLib{})
}

func (l *celJSONLib) LibraryName() string {
	return "json"
}

func (l *celJSONLib) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("json.unmarshal",
			cel.Overload("json.unmarshal_string",
				[]*cel.Type{cel.StringType},
				cel.DynType,
				cel.UnaryBinding(l.unmarshalJSON),
			),
			cel.Overload("json.unmarshal_bytes",
				[]*cel.Type{cel.BytesType},
				cel.DynType,
				cel.UnaryBinding(l.unmarshalJSON),
			),
		),
		cel.Function("json.marshal",
			cel.Overload("json.marshal_dyn",
				[]*cel.Type{cel.DynType},
				cel.BytesType,
				cel.UnaryBinding(l.marshalJSON),
			),
		),
	}
}

func (l *celJSONLib) ProgramOptions() []cel.ProgramOption {
	return nil
}

func (l *celJSONLib) unmarshalJSON(value ref.Val) ref.Val {
	var bytes []byte

	native, err := value.ConvertToNative(stringType)
	if err != nil {
		native, err = value.ConvertToNative(bytesType)
		if err != nil {
			return types.NewErr("json.unmarshal argument must be a string or bytes")
		} else {
			bytes = native.([]byte)
		}
	} else {
		bytes = []byte(native.(string))
	}

	result := new(structpb.Value)
	if err := encoding.FromJSON(bytes, result); err != nil {
		return types.NewErr("json.unmarshal failed to parse JSON: %s", err.Error())
	}

	return types.DefaultTypeAdapter.NativeToValue(result)
}

func (l *celJSONLib) marshalJSON(value ref.Val) ref.Val {
	b, err := encoding.ToJSON(value.Value())
	if err != nil {
		return types.NewErr("json.marshal failed to encode value: %s", err.Error())
	}

	return types.Bytes(b)
}
