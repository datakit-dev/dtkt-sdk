package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/jhump/protoreflect/v2/protoresolve"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type (
	ProtoOptions struct {
		Resolver         protoresolve.SerializationResolver
		DurationAsString bool
	}
	ProtoMapKey interface {
		bool | int32 | int64 | uint32 | uint64 | string
	}
	ProtoScalar interface {
		bool | []byte | float32 | float64 | int32 | int64 | uint32 | uint64 | string
	}
)

var protoWrapperNames = []string{
	"google.protobuf.BoolValue",
	"google.protobuf.BytesValue",
	"google.protobuf.DoubleValue",
	"google.protobuf.FloatValue",
	"google.protobuf.Int32Value",
	"google.protobuf.Int64Value",
	"google.protobuf.StringValue",
	"google.protobuf.UInt32Value",
	"google.protobuf.UInt64Value",
}

func ProtoMapKeyFrom[K ProtoMapKey](key string) (toKey K, err error) {
	switch any(toKey).(type) {
	case string:
		toKey = any(key).(K)
	case bool:
		var b bool
		b, err = strconv.ParseBool(key)
		if err != nil {
			return
		}
		toKey = any(b).(K)
	default:
		cast, ok := isNumber(toKey)
		if ok {
			var n any
			n, err = cast(key)
			if err != nil {
				return
			}
			toKey = any(n).(K)
		}
	}
	return
}

func ProtoMapKeyFor[K ProtoMapKey](key K) string {
	return util.StringFormatAny(key)
}

func AnyTypeURL[T ~string](name T) string {
	return fmt.Sprintf("type.googleapis.com/%s", name)
}

func IsWellKnownName[T ~string](name T) bool {
	return strings.HasPrefix(string(name), "google.protobuf.")
}

func IsWellKnownWrapperName[T ~string](name T) bool {
	return slices.Contains(protoWrapperNames, string(name))
}

func (w ProtoOptions) UnwrapMap(from map[string]proto.Message) (to map[string]any, err error) {
	to = make(map[string]any, len(from))
	if len(from) > 0 {
		for key, proto := range from {
			val, err := w.Unwrap(proto)
			if err != nil {
				return nil, err
			}
			to[key] = val
		}
	}
	return
}

func (w ProtoOptions) UnwrapAnyMap(from map[string]*anypb.Any) (to map[string]any, err error) {
	to = make(map[string]any, len(from))
	if len(from) > 0 {
		for key, proto := range from {
			val, err := w.UnwrapAny(proto)
			if err != nil {
				return nil, err
			}
			to[key] = val
		}
	}
	return
}

func (w ProtoOptions) WrapMap(from map[string]any) (to map[string]proto.Message, err error) {
	to = map[string]proto.Message{}
	if len(from) > 0 {
		for key, val := range from {
			proto, err := w.Wrap(val)
			if err != nil {
				return nil, err
			}
			to[key] = proto
		}
	}
	return
}

func (w ProtoOptions) WrapAnyMap(from map[string]any) (to map[string]*anypb.Any, err error) {
	to = map[string]*anypb.Any{}
	if len(from) > 0 {
		for key, val := range from {
			proto, err := w.WrapAny(val)
			if err != nil {
				return nil, err
			}
			to[key] = proto
		}
	}
	return
}

func (w ProtoOptions) UnwrapSlice(from []proto.Message) (to []any, err error) {
	to = make([]any, len(from))
	if len(from) > 0 {
		for idx, proto := range from {
			val, err := w.Unwrap(proto)
			if err != nil {
				return nil, err
			}
			to[idx] = val
		}
	}
	return
}

func (w ProtoOptions) UnwrapAnySlice(from []*anypb.Any) (to []any, err error) {
	to = make([]any, len(from))
	if len(from) > 0 {
		for idx, proto := range from {
			val, err := w.UnwrapAny(proto)
			if err != nil {
				return nil, err
			}
			to[idx] = val
		}
	}
	return
}

func (w ProtoOptions) WrapAnySlice(from []any) (to []*anypb.Any, err error) {
	to = make([]*anypb.Any, len(from))
	for idx, val := range from {
		proto, err := w.WrapAny(val)
		if err != nil {
			return nil, err
		}
		to[idx] = proto
	}
	return
}

func (w ProtoOptions) WrapSlice(from []any) (to []any, err error) {
	to = make([]any, len(from))
	if len(from) > 0 {
		for idx, proto := range from {
			val, err := w.Wrap(proto)
			if err != nil {
				return nil, err
			}
			to[idx] = val
		}
	}
	return
}

func (w ProtoOptions) WrapAny(value any) (*anypb.Any, error) {
	msg, err := w.Wrap(value)
	if err != nil {
		return nil, err
	}

	if value, ok := msg.(*anypb.Any); ok {
		return value, nil
	}

	return anypb.New(msg)
}

func (w ProtoOptions) Wrap(value any) (proto.Message, error) {
	if value == nil {
		return structpb.NewNullValue(), nil
	}

	switch value := value.(type) {
	case *anypb.Any:
		return value, nil
	case structpb.NullValue, *structpb.Value_NullValue:
		return structpb.NewNullValue(), nil
	case *structpb.Value_ListValue:
		return value.ListValue, nil
	case *structpb.Value_StructValue:
		return value.StructValue, nil
	case *structpb.Value_NumberValue:
		return wrapperspb.Double(value.NumberValue), nil
	case *structpb.Value_BoolValue:
		return wrapperspb.Bool(value.BoolValue), nil
	case *structpb.Value_StringValue:
		return wrapperspb.String(value.StringValue), nil
	case time.Duration:
		return durationpb.New(value), nil
	case time.Time:
		return timestamppb.New(value), nil
	case string:
		return wrapperspb.String(value), nil
	case int32:
		return wrapperspb.Int32(value), nil
	case int64:
		return wrapperspb.Int64(value), nil
	case uint32:
		return wrapperspb.UInt32(value), nil
	case uint64:
		return wrapperspb.UInt64(value), nil
	case float32:
		return wrapperspb.Float(value), nil
	case float64:
		return wrapperspb.Double(value), nil
	case bool:
		return wrapperspb.Bool(value), nil
	case []byte:
		return wrapperspb.Bytes(value), nil
	case []any:
		msg, err := structpb.NewList(value)
		if err == nil {
			return msg, nil
		}
	case map[string]any:
		msg, err := structpb.NewStruct(value)
		if err == nil {
			return msg, nil
		}
	case json.Number:
		if i64Val, i64Err := value.Int64(); i64Err != nil {
			if f64Val, f64Err := value.Float64(); f64Err != nil {
				return nil, errors.Join(i64Err, f64Err)
			} else {
				return wrapperspb.Double(f64Val), nil
			}
		} else {
			return wrapperspb.Int64(i64Val), nil
		}
	case json.RawMessage:
		msg := new(structpb.Value)
		if err := encoding.FromJSONV2(value, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case proto.Message:
		return value, nil
	}

	b, err := encoding.ToJSONV2(value,
		encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
			Resolver: w.Resolver,
		}),
	)
	if err != nil {
		return nil, err
	}

	var a any
	if err = encoding.FromJSONV2(b, &a,
		encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
			Resolver: w.Resolver,
		}),
	); err != nil {
		return nil, err
	}

	return w.Wrap(a)
}

func (w ProtoOptions) UnwrapAny(from *anypb.Any) (any, error) {
	if from == nil {
		return nil, fmt.Errorf("cannot be nil")
	} else if from.TypeUrl == "" {
		return nil, fmt.Errorf("invalid type url")
	}

	return w.Unwrap(from)
}

func (w ProtoOptions) Unwrap(from proto.Message) (any, error) {
	switch value := from.(type) {
	case *anypb.Any:
		msg, err := anypb.UnmarshalNew(value, proto.UnmarshalOptions{
			Resolver: w.Resolver,
		})
		if err != nil {
			return nil, fmt.Errorf("unwrap from %s: %w", value.TypeUrl, err)
		}

		return w.Unwrap(msg)
	case *structpb.Struct:
		return value.AsMap(), nil
	case *structpb.ListValue:
		return value.AsSlice(), nil
	case *structpb.Value:
		return value.AsInterface(), nil
	case *durationpb.Duration:
		if w.DurationAsString {
			return value.AsDuration().String(), nil
		}
		return value.AsDuration(), nil
	case *timestamppb.Timestamp:
		return value.AsTime(), nil
	case *wrapperspb.StringValue:
		return value.Value, nil
	case *wrapperspb.Int32Value:
		return value.Value, nil
	case *wrapperspb.Int64Value:
		return value.Value, nil
	case *wrapperspb.UInt32Value:
		return value.Value, nil
	case *wrapperspb.UInt64Value:
		return value.Value, nil
	case *wrapperspb.FloatValue:
		return value.Value, nil
	case *wrapperspb.DoubleValue:
		return value.Value, nil
	case *wrapperspb.BoolValue:
		return value.Value, nil
	case *wrapperspb.BytesValue:
		return value.Value, nil
	}

	return from, nil
}

func UnwrapProtoAnyAsOptions[T any](from *anypb.Any, opts ProtoOptions) (to T, err error) {
	unwrap, err := opts.UnwrapAny(from)
	if err != nil {
		return
	}

	if to, ok := unwrap.(T); ok {
		return to, nil
	}

	castFunc, ok := isNumber(to)
	if ok {
		unwrap, err = castFunc(unwrap)
		if err != nil {
			return
		}
		return unwrap.(T), nil
	}

	switch value := unwrap.(type) {
	case []byte:
		err = encoding.FromJSONV2(value, &to)
		return
	case string:
		err = encoding.FromJSONV2([]byte(value), &to)
		return
	case proto.Message:
		msg, ok := isMessage(to)
		if ok {
			var b []byte
			b, err = encoding.ToJSONV2(value,
				encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
					Resolver: opts.Resolver,
				}),
			)
			if err != nil {
				return
			}

			msg = msg.ProtoReflect().New().Interface()
			err = encoding.FromJSONV2(b, msg,
				encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: opts.Resolver,
				}),
			)
			if err != nil {
				return
			}

			to, _ = msg.(T)
			return
		}
	default:
		var b []byte
		b, err = encoding.ToJSONV2(value,
			encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
				Resolver: opts.Resolver,
			}),
		)
		if err != nil {
			return
		}

		if msg, ok := isMessage(to); ok {
			err = encoding.FromJSONV2(b, msg,
				encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: opts.Resolver,
				}),
			)
		} else {
			err = encoding.FromJSONV2(b, &to,
				encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: opts.Resolver,
				}),
			)
		}

		return
	}

	err = fmt.Errorf("invalid type, expected: %T, got: %T", to, unwrap)
	return
}

func UnwrapProtoAnyAs[T any](from *anypb.Any) (tVal T, err error) {
	return UnwrapProtoAnyAsOptions[T](from, ProtoOptions{})
}

func UnwrapProto(from proto.Message) (any, error) {
	return ProtoOptions{}.Unwrap(from)
}

func UnwrapProtoAny(from *anypb.Any) (any, error) {
	return ProtoOptions{}.UnwrapAny(from)
}

func WrapProto(from any) (proto.Message, error) {
	return ProtoOptions{}.Wrap(from)
}

func WrapProtoAny(from any) (*anypb.Any, error) {
	return ProtoOptions{}.WrapAny(from)
}

func UnwrapProtoAnyMapOptions[K ProtoMapKey, V any](from map[string]*anypb.Any, opts ProtoOptions) (to map[K]V, err error) {
	to = map[K]V{}
	for k, v := range from {
		val, err := UnwrapProtoAnyAsOptions[V](v, opts)
		if err != nil {
			return nil, err
		}

		key, err := ProtoMapKeyFrom[K](k)
		if err != nil {
			return nil, err
		}

		to[key] = val
	}
	return
}

func UnwrapProtoAnyMap[K ProtoMapKey, V any](from map[string]*anypb.Any) (map[K]V, error) {
	return UnwrapProtoAnyMapOptions[K, V](from, ProtoOptions{})
}

func UnwrapProtoMapOptions[K ProtoMapKey, V proto.Message](from map[K]V, opts ProtoOptions) (map[string]any, error) {
	to := map[string]proto.Message{}
	if len(from) > 0 {
		for k, v := range from {
			to[util.StringFormatAny(k)] = v
		}
	}
	return opts.UnwrapMap(to)
}

func UnwrapProtoMap[K ProtoMapKey, V proto.Message](from map[K]V) (map[string]any, error) {
	return UnwrapProtoMapOptions(from, ProtoOptions{})
}

func WrapProtoMapOptions[K ProtoMapKey, V any](from map[K]V, opts ProtoOptions) (map[string]proto.Message, error) {
	protoMap := map[string]any{}
	if len(from) > 0 {
		for k, v := range from {
			protoMap[ProtoMapKeyFor(k)] = v
		}
	}
	return opts.WrapMap(protoMap)
}

func WrapProtoMap[K ProtoMapKey, V any](from map[K]V) (map[string]proto.Message, error) {
	return WrapProtoMapOptions(from, ProtoOptions{})
}

func WrapProtoAnyMapOptions[K ProtoMapKey, V any](from map[K]V, opts ProtoOptions) (map[string]*anypb.Any, error) {
	protoMap := map[string]any{}
	if len(from) > 0 {
		for k, v := range from {
			protoMap[ProtoMapKeyFor(k)] = v
		}
	}
	return opts.WrapAnyMap(protoMap)
}

func WrapProtoAnyMap[K ProtoMapKey, V any](from map[K]V) (map[string]*anypb.Any, error) {
	return WrapProtoAnyMapOptions(from, ProtoOptions{})
}

func WrapProtoAnySliceOptions[V any](from []V, opts ProtoOptions) ([]*anypb.Any, error) {
	return opts.WrapAnySlice(util.AnySlice(from))
}

func WrapProtoAnySlice[V any](from []V) ([]*anypb.Any, error) {
	return WrapProtoAnySliceOptions(from, ProtoOptions{})
}

func UnwrapProtoAnySliceOptions[V any](from []*anypb.Any, opts ProtoOptions) ([]V, error) {
	toSlice := make([]V, len(from))
	for idx, v := range from {
		val, err := UnwrapProtoAnyAsOptions[V](v, opts)
		if err != nil {
			return nil, err
		}

		toSlice[idx] = val
	}

	return toSlice, nil
}

func UnwrapProtoAnySlice[V any](from []*anypb.Any) ([]V, error) {
	return UnwrapProtoAnySliceOptions[V](from, ProtoOptions{})
}

func UnwrapProtoSliceOptions[V proto.Message](from []V, opts ProtoOptions) ([]any, error) {
	return opts.UnwrapSlice(util.SliceMap(from, func(m V) proto.Message {
		return m
	}))
}

func UnwrapProtoSlice[V proto.Message](from []V) ([]any, error) {
	return UnwrapProtoSliceOptions(from, ProtoOptions{})
}

func isMessage(val any) (msg proto.Message, isMsg bool) {
	msg, isMsg = val.(proto.Message)
	return
}

func isNumber(to any) (func(any) (any, error), bool) {
	switch to.(type) {
	case int32:
		return func(from any) (any, error) {
			return castInt(from, util.ParseInt32)
		}, true
	case uint32:
		return func(from any) (any, error) {
			return castInt(from, util.ParseUInt32)
		}, true
	case int64:
		return func(from any) (any, error) {
			return castInt(from, util.ParseInt64)
		}, true
	case uint64:
		return func(from any) (any, error) {
			return castInt(from, util.ParseUInt64)
		}, true
	case float32:
		return func(from any) (any, error) {
			return castFloat(from, util.ParseFloat32)
		}, true
	case float64:
		return func(from any) (any, error) {
			return castFloat(from, util.ParseFloat64)
		}, true
	}
	return nil, false
}

func castInt[T int32 | uint32 | int64 | uint64](from any, castFunc func(string) (T, error)) (T, error) {
	if from, ok := from.(T); ok {
		return from, nil
	}

	var format string
	switch from := from.(type) {
	case string:
		format, _, _ = strings.Cut(from, ".")
	case []byte:
		format, _, _ = strings.Cut(string(from), ".")
	case float32, float64:
		format, _, _ = strings.Cut(fmt.Sprintf("%.2f", from), ".")
	default:
		format = fmt.Sprintf("%v", from)
	}

	return castFunc(format)
}

func castFloat[T float32 | float64](from any, castFunc func(string) (T, error)) (T, error) {
	switch from := from.(type) {
	case T:
		return from, nil
	case string:
		return castFunc(from)
	case []byte:
		return castFunc(string(from))
	}
	return castFunc(fmt.Sprintf("%v", from))
}
