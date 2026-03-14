package geojsonwrapperpb

import (
	"encoding/json"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"
	"github.com/twpayne/go-geom/encoding/geojson"
)

type FeatureCollection struct {
	*geojsonv1beta1.FeatureCollection
	geom *geojson.FeatureCollection
}

func (fc FeatureCollection) MarshalJSON() ([]byte, error) {
	var collection *geojson.FeatureCollection
	if fc.geom != nil {
		collection = fc.geom
	} else {
		collection = &geojson.FeatureCollection{
			Features: []*geojson.Feature{},
		}

		for _, f := range fc.Features {
			wrapped := Feature{Feature: f}
			data, err := wrapped.MarshalJSON()
			if err != nil {
				return nil, err
			}
			var feature geojson.Feature
			if err := json.Unmarshal(data, &feature); err != nil {
				return nil, err
			}
			collection.Features = append(collection.Features, &feature)
		}
	}

	return json.Marshal(collection)
}

func (fc *FeatureCollection) UnmarshalJSON(data []byte) error {
	var collection geojson.FeatureCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return err
	}

	fc.geom = &collection
	fc.FeatureCollection = &geojsonv1beta1.FeatureCollection{
		Type:     "FeatureCollection",
		Features: []*geojsonv1beta1.Feature{},
	}

	for _, f := range collection.Features {
		// wrapped := &Feature{geom: feature}
		// pbFeature, err := wrapped.ToProto()
		feat, err := NewFeatureFromGeom(f.ID, f.Geometry, f.Properties).ToProto()
		if err != nil {
			return err
		}

		fc.Features = append(fc.Features, feat)
	}

	return nil
}
