package common

import (
	"github.com/invopop/jsonschema"
	compileschema "github.com/santhosh-tekuri/jsonschema/v6"
)

type (
	JSONSchemaOpt          func(JSONSchemaType)
	JSONSchemaCallbackFunc func(*jsonschema.Schema) error
	JSONSchemaLoaderFunc   func(string) (any, error)
)

func WithJSONSchemaCallback(f JSONSchemaCallbackFunc) JSONSchemaOpt {
	return func(s JSONSchemaType) {
		s.setCallback(f)
	}
}

func WithRawSchema(raw []byte) JSONSchemaOpt {
	return func(s JSONSchemaType) {
		s.setRawSchema(raw)
	}
}

func WithSchemaID(id string) JSONSchemaOpt {
	return func(s JSONSchemaType) {
		s.setID(id)
	}
}

func WithReflectorOpts(opts ...func(*jsonschema.Reflector)) JSONSchemaOpt {
	return func(s JSONSchemaType) {
		s.setReflectorOpts(opts...)
	}
}

func WithSchemaCompiler(c *compileschema.Compiler) JSONSchemaOpt {
	return func(s JSONSchemaType) {
		s.setCompiler(c)
	}
}

func (l JSONSchemaLoaderFunc) Load(url string) (any, error) {
	return l(url)
}
