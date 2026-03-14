package geojsonwrapperpb_test

import (
	"encoding/json"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1/geojsonwrapperpb"
	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test_ToFromProto(t *testing.T) {
	props, err := structpb.NewStruct(map[string]any{
		"foo": "bar",
		"bar": 1.234,
	})
	if err != nil {
		t.Fatal(err)
	}

	b, err := json.Marshal(&geojsonwrapperpb.Feature{
		Feature: &geojsonv1beta1.Feature{
			Type: "Feature",
			Id: &geojsonv1beta1.Feature_IdNum{
				IdNum: 123,
			},
			Properties: props,
			Geometry: &geojsonv1beta1.Geometry{
				Geometry: &geojsonv1beta1.Geometry_Polygon{
					Polygon: &geojsonv1beta1.Polygon{
						Type: "Polygon",
						Coordinates: []*geojsonv1beta1.LinearRing{
							{Positions: []*geojsonv1beta1.Position{
								{Values: []float64{1, 2}},
							}},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(b))

	var feat geojsonwrapperpb.Feature
	if err = json.Unmarshal(b, &feat); err != nil {
		t.Fatal(err)
	}

	t.Log(feat.ID())
	t.Log(feat.Geom())
	t.Log(feat.BBox())
	t.Log(feat.Center())
	t.Log(feat.GeoHashUint64())
	t.Log(feat.GeoHashString())
	t.Log(feat.Properties())

	f, err := feat.ToProto()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(f)

	b, err = feat.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(b))
}
