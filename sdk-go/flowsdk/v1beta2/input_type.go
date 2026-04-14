package v1beta2

import (
	"encoding/json"
	"fmt"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/jhump/protoreflect/v2/protoresolve"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	InputType interface {
		InputNode
		InputNullable
		Spec() proto.Message
		IsRequired() bool
		HasDefault() bool
		GetDefault() (any, error)
		Validate(any) (any, error)
	}
	InputNode interface {
		shared.SpecNode
		GetCache() bool
	}
	InputDefault[V any] interface {
		InputNullable
		GetDefault() V
	}
	InputNullable interface {
		GetNullable() bool
	}
	inputScalar[V any] struct {
		InputNode
		InputDefault[V]
		defaultValue V
		hasDefault   bool
	}
	inputList[V any] struct {
		InputNode
		defaultValue []V
		spec         *flowv1beta2.List

		msgType   protoreflect.MessageType
		options   common.ProtoOptions
		validator protovalidate.Validator
	}
	inputMap[K common.ProtoMapKey, V any] struct {
		InputNode
		defaultValue map[K]V
		spec         *flowv1beta2.Map

		msgType   protoreflect.MessageType
		options   common.ProtoOptions
		validator protovalidate.Validator
	}
	inputMessage struct {
		InputNode
		defaultValue proto.Message
		spec         *flowv1beta2.Message

		msgType   protoreflect.MessageType
		options   common.ProtoOptions
		validator protovalidate.Validator
	}
)

func NewInputTypeWithResolver(input *flowv1beta2.Input, resolver protoresolve.SerializationResolver) (InputType, error) {
	if input.GetList() != nil {
		return newListInputType(input, resolver)
	} else if input.GetMap() != nil {
		return newMapInputType(input, resolver)
	} else if input.GetMessage() != nil {
		return newMessageInputType(input, resolver)
	}
	return newScalarInputType(input)
}

func NewInputType(input *flowv1beta2.Input) (InputType, error) {
	return NewInputTypeWithResolver(input, NewInputValidator(input))
}

func newListInputType(input *flowv1beta2.Input, resolver protoresolve.SerializationResolver) (InputType, error) {
	if input.GetList() == nil {
		return nil, fmt.Errorf("input is not list type")
	}

	switch input.GetList().GetItems() {
	case "bool":
		return &inputList[bool]{
			InputNode: input,
			spec:      input.GetList(),
		}, nil
	case "bytes":
		return &inputList[[]byte]{
			InputNode: input,
			spec:      input.GetList(),
		}, nil
	case "double":
		return &inputList[float64]{
			InputNode: input,
			spec:      input.GetList(),
		}, nil
	case "float":
		return &inputList[float32]{
			InputNode: input,
			spec:      input.GetList(),
		}, nil
	case "int64":
		return &inputList[int64]{
			InputNode: input,
			spec:      input.GetList(),
		}, nil
	case "uint64":
		return &inputList[uint64]{
			InputNode: input,
			spec:      input.GetList(),
		}, nil
	case "int32":
		return &inputList[int32]{
			InputNode: input,
			spec:      input.GetList(),
		}, nil
	case "uint32":
		return &inputList[uint32]{
			InputNode: input,
			spec:      input.GetList(),
		}, nil
	case "string":
		return &inputList[string]{
			InputNode: input,
			spec:      input.GetList(),
		}, nil
	}

	msgType, err := resolver.FindMessageByName(protoreflect.FullName(input.GetList().GetItems()))
	if err != nil {
		return nil, fmt.Errorf("list message type: '%s': %w", input.GetList().GetItems(), err)
	}

	validator, err := protovalidate.New(
		protovalidate.WithMessageDescriptors(msgType.Descriptor()),
		protovalidate.WithExtensionTypeResolver(resolver),
	)
	if err != nil {
		return nil, err
	}

	return &inputList[proto.Message]{
		InputNode: input,
		spec:      input.GetList(),

		msgType: msgType,
		options: common.ProtoOptions{
			Resolver:         resolver,
			DurationAsString: true,
		},
		validator: validator,
	}, nil
}

func newMapInputType(input *flowv1beta2.Input, resolver protoresolve.SerializationResolver) (InputType, error) {
	if input.GetMap() == nil {
		return nil, fmt.Errorf("input is not map type")
	}

	switch input.GetMap().GetKey() {
	case "bool":
		return newMapInputValueType[bool](input, resolver)
	case "int32":
		return newMapInputValueType[int32](input, resolver)
	case "int64":
		return newMapInputValueType[int64](input, resolver)
	case "uint32":
		return newMapInputValueType[uint32](input, resolver)
	case "uint64":
		return newMapInputValueType[uint64](input, resolver)
	case "string":
		return newMapInputValueType[string](input, resolver)
	}

	return nil, fmt.Errorf("invalid map key type: %s", input.GetMap().GetKey())
}

func newMessageInputType(input *flowv1beta2.Input, resolver protoresolve.SerializationResolver) (InputType, error) {
	if input.GetMessage() == nil {
		return nil, fmt.Errorf("input is not message type")
	}

	msgType, err := resolver.FindMessageByName(protoreflect.FullName(input.GetMessage().GetType()))
	if err != nil {
		return nil, err
	}

	validator, err := protovalidate.New(
		protovalidate.WithMessageDescriptors(msgType.Descriptor()),
		protovalidate.WithExtensionTypeResolver(resolver),
	)
	if err != nil {
		return nil, err
	}

	return &inputMessage{
		InputNode: input,
		spec:      input.GetMessage(),
		msgType:   msgType,
		options: common.ProtoOptions{
			Resolver:         resolver,
			DurationAsString: true,
		},
		validator: validator,
	}, nil
}

func newScalarInputType(input *flowv1beta2.Input) (InputType, error) {
	if input.GetBool() != nil {
		return &inputScalar[bool]{
			InputNode:    input,
			InputDefault: input.GetBool(),
			hasDefault:   input.GetBool().HasDefault(),
		}, nil
	} else if input.GetBytes() != nil {
		return &inputScalar[[]byte]{
			InputNode:    input,
			InputDefault: input.GetBytes(),
			hasDefault:   input.GetBytes().HasDefault(),
		}, nil
	} else if input.GetDouble() != nil {
		return &inputScalar[float64]{
			InputNode:    input,
			InputDefault: input.GetDouble(),
			hasDefault:   input.GetDouble().HasDefault(),
		}, nil
	} else if input.GetFloat() != nil {
		return &inputScalar[float32]{
			InputNode:    input,
			InputDefault: input.GetFloat(),
			hasDefault:   input.GetFloat().HasDefault(),
		}, nil
	} else if input.GetInt64() != nil {
		return &inputScalar[int64]{
			InputNode:    input,
			InputDefault: input.GetInt64(),
			hasDefault:   input.GetInt64().HasDefault(),
		}, nil
	} else if input.GetUint64() != nil {
		return &inputScalar[uint64]{
			InputNode:    input,
			InputDefault: input.GetUint64(),
			hasDefault:   input.GetUint64().HasDefault(),
		}, nil
	} else if input.GetInt32() != nil {
		return &inputScalar[int32]{
			InputNode:    input,
			InputDefault: input.GetInt32(),
			hasDefault:   input.GetInt32().HasDefault(),
		}, nil
	} else if input.GetUint32() != nil {
		return &inputScalar[uint32]{
			InputNode:    input,
			InputDefault: input.GetUint32(),
			hasDefault:   input.GetUint32().HasDefault(),
		}, nil
	} else if input.GetString() != nil {
		return &inputScalar[string]{
			InputNode:    input,
			InputDefault: input.GetString(),
			hasDefault:   input.GetString().HasDefault(),
		}, nil
	}
	return nil, fmt.Errorf("inputs.%s: unknown type", input.GetId())
}

func newMapInputValueType[K common.ProtoMapKey](input *flowv1beta2.Input, resolver protoresolve.SerializationResolver) (InputType, error) {
	switch input.GetMap().GetValue() {
	case "bool":
		return &inputMap[K, bool]{
			InputNode: input,
			spec:      input.GetMap(),
		}, nil
	case "bytes":
		return &inputMap[K, []byte]{
			InputNode: input,
			spec:      input.GetMap(),
		}, nil
	case "double", "float64":
		return &inputMap[K, float64]{
			InputNode: input,
			spec:      input.GetMap(),
		}, nil
	case "float", "float32":
		return &inputMap[K, float32]{
			InputNode: input,
			spec:      input.GetMap(),
		}, nil
	case "int64":
		return &inputMap[K, int64]{
			InputNode: input,
			spec:      input.GetMap(),
		}, nil
	case "uint64":
		return &inputMap[K, uint64]{
			InputNode: input,
			spec:      input.GetMap(),
		}, nil
	case "int32":
		return &inputMap[K, int32]{
			InputNode: input,
			spec:      input.GetMap(),
		}, nil
	case "uint32":
		return &inputMap[K, uint32]{
			InputNode: input,
			spec:      input.GetMap(),
		}, nil
	case "string":
		return &inputMap[K, string]{
			InputNode: input,
			spec:      input.GetMap(),
		}, nil
	}

	msgType, err := resolver.FindMessageByName(protoreflect.FullName(input.GetMap().GetValue()))
	if err != nil {
		return nil, fmt.Errorf("map message type: '%s': %w", input.GetMap().GetValue(), err)
	}

	validator, err := protovalidate.New(
		protovalidate.WithMessageDescriptors(msgType.Descriptor()),
		protovalidate.WithExtensionTypeResolver(resolver),
	)
	if err != nil {
		return nil, err
	}

	return &inputMap[K, proto.Message]{
		InputNode: input,
		spec:      input.GetMap(),
		msgType:   msgType,
		options: common.ProtoOptions{
			Resolver: resolver,
		},
		validator: validator,
	}, nil
}

func (t *inputList[V]) Spec() proto.Message {
	return t.spec
}

func (t *inputList[V]) GetNullable() bool {
	return t.spec.GetNullable()
}

func (t *inputList[V]) GetDefault() (any, error) {
	if t.HasDefault() && t.defaultValue == nil {
		t.defaultValue = []V{}

		if t.msgType == nil {
			protos, err := common.WrapProtoAnySlice(t.spec.GetDefault().AsSlice())
			if err != nil {
				return nil, err
			}

			value, err := common.UnwrapProtoAnySlice[V](protos)
			if err != nil {
				return nil, err
			}

			t.defaultValue = value
		} else {
			t.defaultValue = make([]V, len(t.spec.GetDefault().GetValues()))

			for idx, val := range t.spec.GetDefault().GetValues() {
				b, err := encoding.ToJSONV2(val,
					encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
						Resolver: t.options.Resolver,
					}),
				)
				if err != nil {
					return nil, err
				}

				msg := t.msgType.New().Interface()
				err = encoding.FromJSONV2(b, msg,
					encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
						Resolver: t.options.Resolver,
					}),
				)
				if err != nil {
					return nil, err
				}

				err = t.validator.Validate(msg)
				if err != nil {
					return nil, err
				}

				t.defaultValue[idx] = msg.(V)
			}
		}
	}

	return t.defaultValue, nil
}

func (t *inputList[V]) IsRequired() bool {
	return !t.GetNullable() && !t.HasDefault()
}

func (t *inputList[V]) HasDefault() bool {
	return t.spec.GetDefault() != nil
}

func (t *inputList[V]) Validate(value any) (any, error) {
	if value == nil && t.HasDefault() {
		return t.GetDefault()
	} else if value == nil && t.GetNullable() {
		return nil, nil
	} else if value == nil {
		return nil, fmt.Errorf("cannot be null")
	} else if t.msgType == nil {
		switch value := value.(type) {
		case []V:
			return value, nil
		case []any:
			protoSlice, err := common.WrapProtoAnySliceOptions(value, t.options)
			if err != nil {
				return nil, err
			}

			return common.UnwrapProtoAnySliceOptions[V](protoSlice, t.options)
		}

		b, err := encoding.ToJSONV2(value,
			encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
				Resolver: t.options.Resolver,
			}),
		)
		if err != nil {
			return nil, err
		}

		var list []V
		err = encoding.FromJSONV2(b, &list,
			encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
				Resolver: t.options.Resolver,
			}),
		)
		if err != nil {
			return nil, err
		}

		return list, nil
	}

	var jsonSlice []json.RawMessage
	b, err := encoding.ToJSONV2(value,
		encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
			Resolver: t.options.Resolver,
		}),
	)
	if err != nil {
		return nil, err
	}

	err = encoding.FromJSONV2(b, &jsonSlice,
		encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
			Resolver: t.options.Resolver,
		}),
	)
	if err != nil {
		return nil, err
	}

	msgSlice := make([]V, len(jsonSlice))
	for idx, jsonVal := range jsonSlice {
		msg := t.msgType.New().Interface()
		err = encoding.FromJSONV2(jsonVal, msg,
			encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
				Resolver: t.options.Resolver,
			}),
		)
		if err != nil {
			return nil, err
		}

		err = t.validator.Validate(msg)
		if err != nil {
			return nil, err
		}

		msgSlice[idx] = msg.(V)
	}

	return msgSlice, nil
}

func (t *inputMap[K, V]) Spec() proto.Message {
	return t.spec
}

func (t *inputMap[K, V]) GetNullable() bool {
	return t.spec.GetNullable()
}

func (t *inputMap[K, V]) IsRequired() bool {
	return !t.GetNullable() && !t.HasDefault()
}

func (t *inputMap[K, V]) HasDefault() bool {
	return t.spec.GetDefault() != nil
}

func (t *inputMap[K, V]) GetDefault() (any, error) {
	if t.HasDefault() && t.defaultValue == nil {
		t.defaultValue = map[K]V{}

		if t.msgType == nil {
			protoMap, err := common.WrapProtoAnyMap(t.spec.GetDefault().AsMap())
			if err != nil {
				return nil, err
			}

			value, err := common.UnwrapProtoAnyMap[K, V](protoMap)
			if err != nil {
				return nil, err
			}

			t.defaultValue = value
		} else {
			for key, val := range t.spec.GetDefault().AsMap() {
				b, err := encoding.ToJSONV2(val,
					encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
						Resolver: t.options.Resolver,
					}),
				)
				if err != nil {
					return nil, err
				}

				msg := t.msgType.New().Interface()
				err = encoding.FromJSONV2(b, msg,
					encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
						Resolver: t.options.Resolver,
					}),
					encoding.WithDecodeDurationString(),
				)
				if err != nil {
					return nil, err
				}

				err = t.validator.Validate(msg)
				if err != nil {
					return nil, err
				}

				key, err := common.ProtoMapKeyFrom[K](key)
				if err != nil {
					return nil, err
				}

				t.defaultValue[key] = msg.(V)
			}
		}
	}

	return t.defaultValue, nil
}

func (t *inputMap[K, V]) Validate(value any) (any, error) {
	if value == nil && t.HasDefault() {
		return t.GetDefault()
	} else if value == nil && t.GetNullable() {
		return nil, nil
	} else if value == nil {
		return nil, fmt.Errorf("cannot be null")
	} else if value, ok := value.(map[K]V); ok && t.msgType == nil {
		return value, nil
	}

	var anyMap map[K]any
	b, err := encoding.ToJSONV2(value,
		encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
			Resolver: t.options.Resolver,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("encode value: %w", err)
	}

	err = encoding.FromJSONV2(b, &anyMap,
		encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
			Resolver: t.options.Resolver,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("decode value: %w", err)
	}

	if t.msgType == nil {
		protoMap, err := common.WrapProtoAnyMapOptions(anyMap, t.options)
		if err != nil {
			return nil, err
		}

		return common.UnwrapProtoAnyMapOptions[K, V](protoMap, t.options)
	}

	protoMap, err := common.WrapProtoMapOptions(anyMap, t.options)
	if err != nil {
		return nil, fmt.Errorf("proto wrap value: %w", err)
	}

	msgMap := map[K]V{}
	for key, msg := range protoMap {
		if msg.ProtoReflect().Descriptor().FullName() != t.msgType.Descriptor().FullName() {
			b, err := encoding.ToJSONV2(msg,
				encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
					Resolver: t.options.Resolver,
				}),
			)
			if err != nil {
				return nil, fmt.Errorf("encode proto value: %w", err)
			}

			msg = t.msgType.New().Interface()
			err = encoding.FromJSONV2(b, msg,
				encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: t.options.Resolver,
				}),
			)
			if err != nil {
				return nil, fmt.Errorf("decode proto value: %w", err)
			}
		}

		err = t.validator.Validate(msg)
		if err != nil {
			return nil, err
		}

		key, err := common.ProtoMapKeyFrom[K](key)
		if err != nil {
			return nil, err
		}

		msgMap[key] = msg.(V)
	}

	return msgMap, nil
}

func (t *inputMessage) Spec() proto.Message {
	return t.spec
}

func (t *inputMessage) IsRequired() bool {
	return !t.spec.GetNullable() && t.spec.GetDefault() == nil
}

func (t *inputMessage) GetNullable() bool {
	return t.spec.GetNullable()
}

func (t *inputMessage) HasDefault() bool {
	return t.spec.GetDefault() != nil
}

func (t *inputMessage) GetDefault() (any, error) {
	if t.defaultValue == nil {
		t.defaultValue = t.msgType.New().Interface()

		if t.HasDefault() {
			b, err := encoding.ToJSONV2(t.spec.GetDefault())
			if err != nil {
				return nil, err
			}

			err = encoding.FromJSONV2(b, t.defaultValue,
				encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: t.options.Resolver,
				}),
			)
			if err != nil {
				return nil, err
			}
		}

		err := t.validator.Validate(t.defaultValue)
		if err != nil {
			return nil, err
		}
	}

	return t.defaultValue, nil
}

func (t *inputMessage) Validate(value any) (_ any, err error) {
	if value == nil && t.HasDefault() {
		return t.GetDefault()
	} else if value == nil && t.GetNullable() {
		return t.msgType.New().Interface(), nil
	} else if value == nil {
		return nil, fmt.Errorf("cannot be null")
	}

	msg, isProto := value.(proto.Message)
	if !isProto || msg.ProtoReflect().Descriptor().FullName() != t.msgType.Descriptor().FullName() {
		b, err := encoding.ToJSONV2(value,
			encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
				Resolver: t.options.Resolver,
			}),
		)
		if err != nil {
			return nil, err
		}

		msg = t.msgType.New().Interface()
		err = encoding.FromJSONV2(b, msg,
			encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
				Resolver: t.options.Resolver,
			}),
		)
		if err != nil {
			return nil, err
		}
	}

	err = t.validator.Validate(msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (t *inputScalar[V]) Spec() proto.Message {
	return t.InputDefault.(proto.Message)
}

func (t *inputScalar[V]) IsRequired() bool {
	return !t.GetNullable() && !t.HasDefault()
}

func (t *inputScalar[V]) HasDefault() bool {
	return t.hasDefault
}

func (t *inputScalar[V]) GetDefault() (any, error) {
	if t.HasDefault() {
		return t.InputDefault.GetDefault(), nil
	}
	return t.defaultValue, nil
}

func (t *inputScalar[V]) Validate(value any) (any, error) {
	if value == nil && t.HasDefault() {
		return t.GetDefault()
	} else if value == nil && t.GetNullable() {
		var v V
		return v, nil
	} else if value == nil {
		return nil, fmt.Errorf("cannot be null")
	} else if value, ok := value.(V); ok {
		return value, nil
	}

	wrap, err := common.WrapProtoAny(value)
	if err != nil {
		return nil, err
	}

	return common.UnwrapProtoAnyAs[V](wrap)
}
