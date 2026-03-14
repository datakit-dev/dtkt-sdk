package encoding_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1/geojsonwrapperpb"
	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"
)

var (
	testStructValid   = []byte(`{"s": "Foo Bar", "f": 123.456}`)
	testStructInvalid = []byte(`{"s": 123.456, "f": "Foo Bar"}`)
	geoJSONValid      = []byte(`{"type":"Feature","id":"1","geometry":{"type":"Polygon","coordinates":[[[-111.290492270536,47.5080344121109],[-111.290489466303,47.5084460198696],[-111.29069144007,47.50844717663],[-111.29069479445,47.5080353433743],[-111.290492270536,47.5080344121109]]]},"properties":{"county_name":"Cascade"}}`)
)

type (
	testCase struct {
		valid    bool
		body     []byte
		expected any
	}
	testStruct struct {
		S string  `json:"s"`
		F float64 `json:"f"`
	}
)

func TestJSON(t *testing.T) {
	tests := []testCase{
		{
			valid: true,
			body:  testStructValid,
			expected: testStruct{
				S: "Foo Bar",
				F: 123.456,
			},
		},
		{
			valid:    false,
			body:     testStructInvalid,
			expected: testStruct{},
		},
	}

	for idx, test := range tests {
		var s testStruct
		err := encoding.FromJSON(test.body, &s)
		if test.valid && err != nil {
			t.Fatalf("test %d FromJSON expected to be valid, got error: %s", idx, err.Error())
		} else if !test.valid && err == nil {
			t.Fatalf("test %d FromJSON expected to be invalid", idx)
		} else if test.valid && s != test.expected {
			t.Fatalf("test %d FromJSON should be valid expected: %#v, got: %#v", idx, test.expected, s)
		}

		_, err = encoding.ToJSON(test.expected)
		if test.valid && err != nil {
			t.Fatalf("test %d ToJSON expected to be valid, got error: %s", idx, err.Error())
		} else if !test.valid && err == nil && s != test.expected {
			t.Fatalf("test %d ToJSON expected to be invalid expected: %#v, got: %#v", idx, test.expected, s)
		}
	}

	var w geojsonwrapperpb.GeoJSON
	if err := encoding.FromJSON(geoJSONValid, &w); err != nil {
		t.Fatalf("geojsonwrapperpb.GeoJSON unexpected error: %s", err)
	} else if w.GeoJSON == nil || !w.GeoJSON.ProtoReflect().IsValid() {
		t.Fatalf("expected valid GeoJSON: %#v", w.GeoJSON)
	}

	_, err := encoding.ToJSON(w)
	if err != nil {
		t.Fatalf("geojsonwrapperpb.GeoJSON unexpected error: %s", err)
	}

	b, err := encoding.ToJSON(w.GeoJSON)
	if err != nil {
		t.Fatalf("geojsonv1beta1.GeoJSON unexpected error: %s", err)
	}

	g := &geojsonv1beta1.GeoJSON{}
	err = encoding.FromJSON(b, g)
	if err != nil {
		t.Fatalf("geojsonv1beta1.GeoJSON unexpected error: %s", err)
	}
}
