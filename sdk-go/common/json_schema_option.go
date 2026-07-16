package common

import (
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
)

type (
	JSONSchemaOptions struct {
		jsonschema.ResolveOptions
		ID       string
		Bytes    []byte
		Callback JSONSchemaCallbackFunc
	}
	JSONSchemaOption       func(*JSONSchemaOptions)
	JSONSchemaLoaderFunc   func(*url.URL) (*jsonschema.Schema, error)
	JSONSchemaCallbackFunc func(*jsonschema.Schema, *jsonschema.ResolveOptions) error
)

func WithJSONSchemaBaseURI(baseUri string) JSONSchemaOption {
	return func(opts *JSONSchemaOptions) {
		opts.ResolveOptions.BaseURI = baseUri
	}
}

func WithJSONSchemaLoader(loader JSONSchemaLoaderFunc) JSONSchemaOption {
	return func(opts *JSONSchemaOptions) {
		opts.ResolveOptions.Loader = jsonschema.Loader(loader)
	}
}

func WithJSONSchemaValidateDefaults(validateDefaults bool) JSONSchemaOption {
	return func(opts *JSONSchemaOptions) {
		opts.ResolveOptions.ValidateDefaults = validateDefaults
	}
}

func WithJSONSchemaID(id string) JSONSchemaOption {
	return func(opts *JSONSchemaOptions) {
		opts.ID = id
	}
}

func WithJSONSchemaCallback(callback JSONSchemaCallbackFunc) JSONSchemaOption {
	return func(opts *JSONSchemaOptions) {
		opts.Callback = callback
	}
}

func WithJSONSchemaBytes(bytes []byte) JSONSchemaOption {
	return func(opts *JSONSchemaOptions) {
		if len(bytes) > 0 {
			opts.Bytes = bytes
		}
	}
}

func (l JSONSchemaLoaderFunc) Load(url *url.URL) (*jsonschema.Schema, error) {
	return l(url)
}
