package common_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/invopop/jsonschema"
)

func Test_EmptyStruct(t *testing.T) {
	emptyObjSchema, err := common.JSONSchemaFor[struct{}]()
	if err != nil {
		t.Errorf("expected schema, got error: %v", err)
	}

	t.Log("emptyObjSchema:", emptyObjSchema)

	emptyObjArrSchema, err := common.JSONSchemaFor[[]struct{}]()
	if err != nil {
		t.Errorf("expected schema, got error: %v", err)
	}

	t.Log("emptyObjArrSchema:", emptyObjArrSchema)
}

type StructConfig struct {
	Key string         `json:"key,omitempty"`
	Map map[string]any `json:"map,omitempty"`
}

func Test_StructSchema(t *testing.T) {
	var structConfig = StructConfig{
		Key: "foo",
		Map: map[string]any{
			"bar": "baz",
		},
	}
	structSchema, err := common.JSONSchemaFor[StructConfig](
		common.WithReflectorOpts(func(r *jsonschema.Reflector) {
			r.BaseSchemaID = "http://foo.bar"
		}),
		common.WithSchemaID("foo"),
	)
	if err != nil {
		t.Errorf("expected schema, got error: %v", err)
	}

	t.Log(structSchema)

	if err := structSchema.Validate(structConfig); err != nil {
		t.Errorf("expected valid schema, got invalid: %v", err)
	} else if err := structSchema.ValidateAny(map[string]any{
		"key": "foo",
		"map": map[string]any{
			"bar": "baz",
		},
	}); err != nil {
		t.Errorf("expected valid schema, got invalid: %v", err)
	} else if config, err := structSchema.ValidateString(`{"key": "value"}`); err != nil {
		t.Errorf("expected valid schema, got invalid: %v", err)
	} else if config.Key != "value" {
		t.Errorf("expected config key 'value', got '%s'", config.Key)
	} else {
		t.Log(config)
	}

	structSliceSchema, err := common.JSONSchemaFor[[]StructConfig]()
	if err != nil {
		t.Errorf("expected struct slice schema, got error: %v", err)
	}

	t.Log("structSliceSchema:", structSliceSchema)

	structMapSchema, err := common.JSONSchemaFor[map[string]StructConfig]()
	if err != nil {
		t.Errorf("expected struct map schema, got error: %v", err)
	}

	t.Log("structMapSchema:", structMapSchema)
}

type MapConfig map[string]any

func Test_MapSchema(t *testing.T) {
	var mapConfig = MapConfig{}

	mapSchema, err := common.JSONSchemaFor[MapConfig]()
	if err != nil {
		t.Fatalf("expected schema, got error: %v", err)
	}

	t.Log(mapSchema)

	if err := mapSchema.Validate(mapConfig); err != nil {
		t.Errorf("expected valid, got invalid: %v", err)
	} else if err := mapSchema.ValidateAny(map[string]any{
		"foo": "bar",
	}); err != nil {
		t.Errorf("expected valid, got invalid: %v", err)
	} else if config, err := mapSchema.ValidateString(`{"key": "value"}`); err != nil {
		t.Errorf("expected valid, got invalid: %v", err)
	} else if config["key"] != "value" {
		t.Errorf("expected config key 'value', got '%s'", config["key"])
	} else {
		t.Log(config)
	}
}

type SliceConfig []string

func Test_SliceSchema(t *testing.T) {
	sliceSchema, err := common.JSONSchemaFor[SliceConfig]()
	if err != nil {
		t.Errorf("expected schema, got error: %v", err)
	}

	t.Log(sliceSchema)

	if err := sliceSchema.Validate([]string{"foo", "bar"}); err != nil {
		t.Errorf("expected valid value, got invalid: %v", err)
	} else if err := sliceSchema.ValidateAny(`["foo", "bar"]`); err != nil {
		t.Errorf("expected valid value, got invalid: %v", err)
	} else if config, err := sliceSchema.ValidateString(`["foo", "bar"]`); err != nil {
		t.Errorf("expected valid value, got invalid: %v", err)
	} else if config[0] != "foo" {
		t.Errorf("expected config key 'value', got '%s'", config[0])
	} else {
		t.Log(config)
	}
}

func Test_Scalars(t *testing.T) {
	var scalars = map[common.JSONType]any{
		common.JSONBoolean: true,
		common.JSONInteger: int64(123),
		common.JSONNumber:  456.789,
		common.JSONString:  "a",
	}

	for typ, val := range scalars {
		schema, err := common.NewJSONSchema(val)
		if err != nil {
			t.Errorf("expected schema, got error: %v", err)
		}

		t.Log(schema.String())
		t.Log(typ, val)

		if err := schema.Validate(val); err != nil {
			t.Errorf("expected valid, got invalid: %v", err)
		}
	}
}
