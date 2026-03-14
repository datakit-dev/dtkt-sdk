package spec

import (
	"fmt"
	"path"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/jhump/protoreflect/v2/protoresolve"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type (
	InputValidator struct {
		protoresolve.SerializationResolver
		input     *flowv1beta1.Input
		inputType InputType
		message   *MessageValidator
		err       error
	}
	MessageValidator struct {
		protoreflect.MessageType
		resolver protoresolve.SerializationResolver
		name     protoreflect.FullName
	}
)

func NewInputValidator(input *flowv1beta1.Input) *InputValidator {
	v := &InputValidator{
		input: input,
	}
	v.inputType, v.err = NewInputTypeWithResolver(v.input, v)
	return v
}

func NewInputValidatorWithResolver(input *flowv1beta1.Input, resolver protoresolve.SerializationResolver) *InputValidator {
	v := &InputValidator{
		input: input,
	}
	v.SerializationResolver = resolver
	v.inputType, v.err = NewInputTypeWithResolver(v.input, v)
	return v
}

func NewMessageValidator(name protoreflect.FullName) *MessageValidator {
	return &MessageValidator{
		name: name,
	}
}

func NewMessageValidatorWithResolver(name protoreflect.FullName, resolver protoresolve.SerializationResolver) *MessageValidator {
	return &MessageValidator{
		name:     name,
		resolver: resolver,
	}
}

func (v *InputValidator) TypeName() string {
	if v.message == nil {
		return ""
	}
	return string(v.message.name)
}

func (v *InputValidator) Validate(value any) (any, error) {
	if v.err != nil {
		return nil, v.err
	} else if v.inputType == nil {
		return nil, fmt.Errorf("unknown type")
	}

	val, err := v.inputType.Validate(value)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (r *InputValidator) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	message := NewMessageValidatorWithResolver(name, r.SerializationResolver)
	if r.message == nil {
		r.message = message
	}
	return message.ResolveMessageType()
}

func (r *InputValidator) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	message := NewMessageValidatorWithResolver(protoreflect.FullName(path.Base(url)), r.SerializationResolver)
	if r.message == nil {
		r.message = message
	}
	return message.ResolveMessageType()
}

func (r *InputValidator) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func (r *InputValidator) FindExtensionByNumber(name protoreflect.FullName, num protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func (m *MessageValidator) ResolveMessageType() (protoreflect.MessageType, error) {
	if m.MessageType == nil {
		if m.resolver != nil {
			msgType, err := m.resolver.FindMessageByName(m.name)
			if err != nil {
				return nil, err
			}
			m.MessageType = msgType
		} else if api.IsKnownName(m.name) {
			msgType, err := api.GlobalResolver().FindMessageByName(m.name)
			if err != nil {
				return nil, err
			}
			m.MessageType = msgType
		} else {
			msgType, err := protoschema.NewParser(protoschema.ParserOptions{
				PackageName: string(m.name.Parent()),
				MessageName: string(m.name.Name()),
			}).ParseMessageTypeMap(map[string]any{
				"type": "object",
			})
			if err != nil {
				return nil, err
			}
			m.MessageType = msgType
		}
	}
	return m, nil
}
