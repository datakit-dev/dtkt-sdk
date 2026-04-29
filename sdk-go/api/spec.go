package api

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/invopop/jsonschema"
)

var _ SpecLoaderType = (*SpecLoader[SpecType])(nil)

type (
	SpecLoader[S SpecType] struct {
		apiVersion version
		kind       string
		spec       S
		schemaOpts []common.JSONSchemaOpt
	}
	SpecInstance[S SpecType] struct {
		APIVersion version `json:"apiVersion" yaml:"apiVersion"`
		Kind       string  `json:"kind" yaml:"kind"`
		Spec       S       `json:"spec" yaml:"spec"`
	}
	SpecLoaderType interface {
		APIVersion() Version
		SpecKind() string
		SpecID() string
		JSONSchema() (*jsonschema.Schema, error)
		MarshalJSONSchema() ([]byte, error)
		Filename() string
	}
	SpecType interface {
		APIVersion() Version
		SpecKind() string
		SpecID() string
		Validate() error
		Filename() string
	}
	SpecLoaderOpt[T SpecType] func(*SpecLoader[T])
)

func NewLoader[T SpecType](spec T, opts ...SpecLoaderOpt[T]) *SpecLoader[T] {
	loader := &SpecLoader[T]{
		kind:       spec.SpecKind(),
		apiVersion: spec.APIVersion().(version),
		spec:       spec,
	}
	loader.applyOptions(opts...)
	return loader
}

func (l *SpecLoader[T]) applyOptions(opts ...SpecLoaderOpt[T]) {
	for _, opt := range opts {
		if opt != nil {
			opt(l)
		}
	}
}

func (l *SpecLoader[T]) loadSchema() (*common.JSONSchema[SpecInstance[T]], error) {
	opts := append([]common.JSONSchemaOpt{
		common.WithSchemaID(l.spec.Filename()),
	}, l.schemaOpts...)

	schema, err := common.NewJSONSchema(SpecInstance[T]{
		APIVersion: l.apiVersion,
		Kind:       l.kind,
		Spec:       l.spec,
	}, opts...)
	if err != nil {
		return nil, err
	}

	schema.JSONSchema().Title = l.spec.SpecKind()

	if schema.JSONSchema().Properties != nil {
		kind := schema.JSONSchema().Properties.GetPair("kind")
		if kind != nil {
			kind.Value.Enum = util.AnySlice([]string{l.spec.SpecKind()})
		}
	}

	return schema, nil
}

func (l *SpecLoader[T]) Spec() T {
	return l.spec
}

func (l *SpecLoader[T]) Filename() string {
	return l.spec.Filename()
}

func (l *SpecLoader[T]) APIVersion() Version {
	return l.spec.APIVersion()
}

func (l *SpecLoader[T]) SpecKind() string {
	return l.spec.SpecKind()
}

func (l *SpecLoader[T]) SpecID() string {
	return l.spec.Filename()
}

func (l *SpecLoader[T]) MarshalJSONSchema() ([]byte, error) {
	schema, err := l.loadSchema()
	if err != nil {
		return nil, err
	}

	return encoding.ToJSONV2(schema.JSONSchema(),
		encoding.WithEncodeIndent("", "  "),
	)
}

func (l *SpecLoader[T]) TypedSchema() (*common.JSONSchema[SpecInstance[T]], error) {
	schema, err := l.loadSchema()
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func (l *SpecLoader[T]) JSONSchema() (*jsonschema.Schema, error) {
	schema, err := l.loadSchema()
	if err != nil {
		return nil, err
	}
	return schema.JSONSchema(), nil
}

func (l *SpecLoader[T]) Decode(format encoding.Format, raw []byte) (_ T, err error) {
	var spec SpecInstance[T]
	err = format.Decode(raw, &spec)
	if err != nil {
		return
	} else if spec.Kind != l.kind {
		err = fmt.Errorf("invalid spec kind: %s, expected: %s", spec.Kind, l.kind)
		return
	} else if spec.APIVersion != l.apiVersion {
		err = fmt.Errorf("invalid spec version: %s, expected: %s", spec.APIVersion, l.apiVersion)
		return
	}

	err = spec.Spec.Validate()
	if err != nil {
		return
	}

	return spec.Spec, nil
}

func (l *SpecLoader[T]) Encode(format encoding.Format, spec T) (raw []byte, err error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	return format.Encode(SpecInstance[T]{
		APIVersion: l.apiVersion,
		Kind:       l.kind,
		Spec:       spec,
	})
}
