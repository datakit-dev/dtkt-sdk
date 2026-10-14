package v1beta1

import (
	"net/http"
	"slices"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	DataTypes   []*sharedv1beta1.DataType
	Params      []*sharedv1beta1.Param
	Fields      []*sharedv1beta1.Field
	DataTypeOpt func(*sharedv1beta1.DataType)
	ParamOpt    func(*sharedv1beta1.Param)
	FieldOpt    func(*sharedv1beta1.Field)
)

func NewDataTypes(dataTypes ...*sharedv1beta1.DataType) DataTypes {
	return dataTypes
}

func NewDataType[T1 ~string](nativeType T1, opts ...DataTypeOpt) *sharedv1beta1.DataType {
	var t = &sharedv1beta1.DataType{NativeType: string(nativeType)}
	for opt := range slices.Values(opts) {
		if opt != nil {
			opt(t)
		}
	}
	return t
}

func NewParams(params ...*sharedv1beta1.Param) Params {
	return params
}

func NewParam(field *sharedv1beta1.Field, opts ...ParamOpt) (param *sharedv1beta1.Param) {
	param = &sharedv1beta1.Param{Field: field}
	for opt := range slices.Values(opts) {
		if opt != nil {
			opt(param)
		}
	}
	return
}

func NewField(name string, dataType *sharedv1beta1.DataType, opts ...FieldOpt) (field *sharedv1beta1.Field) {
	field = &sharedv1beta1.Field{
		Name: name,
		Type: dataType,
	}
	for opt := range slices.Values(opts) {
		if opt != nil {
			opt(field)
		}
	}
	return
}

func NewParamWithValue(field *sharedv1beta1.Field, native any, opts ...ParamOpt) (param *sharedv1beta1.Param, err error) {
	param = NewParam(field, opts...)
	value, err := common.WrapProtoAny(native)
	if err != nil {
		return nil, err
	}

	param.Value = value

	return param, nil
}

func (p Params) Fields() (f Fields) {
	for _, param := range p {
		f = append(f, param.Field)
	}
	return
}

func (nt DataTypes) Find(nativeType string) (*sharedv1beta1.DataType, bool) {
	for _, t := range nt {
		if strings.EqualFold(t.NativeType, nativeType) {
			return t, true
		}
	}
	return nil, false
}

func WithJSONType(t sharedv1beta1.JSONType) DataTypeOpt {
	return func(dt *sharedv1beta1.DataType) {
		dt.JsonType = t
	}
}

func WithGeoType(t sharedv1beta1.GeoType) DataTypeOpt {
	return func(dt *sharedv1beta1.DataType) {
		dt.GeoType = t
	}
}

func WithDataTypeMetadata(m *structpb.Struct) DataTypeOpt {
	return func(dt *sharedv1beta1.DataType) {
		dt.Metadata = m
	}
}

func WithParamValue(v *anypb.Any) ParamOpt {
	return func(p *sharedv1beta1.Param) {
		p.Value = v
	}
}

func WithFieldNullable(nullable bool) FieldOpt {
	return func(f *sharedv1beta1.Field) {
		f.Nullable = nullable
	}
}

func WithFieldDescription(desc string) FieldOpt {
	return func(f *sharedv1beta1.Field) {
		f.Description = desc
	}
}

func WithFieldRepeated(repeated bool) FieldOpt {
	return func(f *sharedv1beta1.Field) {
		f.Repeated = repeated
	}
}

func WithFields(fields ...*sharedv1beta1.Field) FieldOpt {
	return func(f *sharedv1beta1.Field) {
		f.Fields = fields
	}
}

func HeadersFromProto(protoMap map[string]*sharedv1beta1.StringList) (headers http.Header) {
	headers = http.Header{}
	for key, val := range protoMap {
		for _, val := range val.GetValues() {
			headers.Add(key, val)
		}
	}
	return
}

func HeadersToProto(headers http.Header) (protoMap map[string]*sharedv1beta1.StringList) {
	protoMap = map[string]*sharedv1beta1.StringList{}
	for key := range headers {
		protoMap[key] = &sharedv1beta1.StringList{
			Values: headers.Values(key),
		}
	}
	return
}
