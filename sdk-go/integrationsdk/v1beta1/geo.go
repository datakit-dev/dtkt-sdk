package v1beta1

import (
	"encoding/json"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1/geojsonwrapperpb"
	catalogv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/catalog/v1beta1"
	geov1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geo/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
)

type (
	StreamGeoJsonRequest struct {
		*geov1beta1.StreamGeoJsonRequest
	}
	StreamGeoJsonResponse struct {
		*geov1beta1.StreamGeoJsonResponse
	}
	GeoSource struct {
		*geov1beta1.GeoSource
	}
	Bounds struct {
		*geov1beta1.Bounds
	}
)

func GeoPropertyTypeFromDataType(typ *sharedv1beta1.DataType) (geov1beta1.PropertyType, bool) {
	switch typ.JsonType {
	case sharedv1beta1.JSONType_JSON_TYPE_STRING:
		return geov1beta1.PropertyType_PROPERTY_TYPE_STRING, true
	case sharedv1beta1.JSONType_JSON_TYPE_NUMBER:
		return geov1beta1.PropertyType_PROPERTY_TYPE_NUMBER, true
	case sharedv1beta1.JSONType_JSON_TYPE_BOOLEAN:
		return geov1beta1.PropertyType_PROPERTY_TYPE_BOOLEAN, true
	}

	return 0, false
}

func (r *StreamGeoJsonRequest) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var req = new(geov1beta1.StreamGeoJsonRequest)
	for key, val := range raw {
		switch key {
		case "geo_field":
			if err := json.Unmarshal(val, &req.GeoField); err != nil {
				return err
			}
		case "prop_fields":
			if err := json.Unmarshal(val, &req.PropFields); err != nil {
				return err
			}
		case "bounds":
			var bounds Bounds
			if err := json.Unmarshal(val, &bounds); err != nil {
				return err
			}
			req.Bounds = bounds.Bounds
		case "source":
			var source GeoSource
			if err := json.Unmarshal(val, &source); err != nil {
				return err
			}
			req.Source = source.GeoSource
		}
	}

	*r = StreamGeoJsonRequest{StreamGeoJsonRequest: req}
	return nil
}

func (r StreamGeoJsonResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"result": geojsonwrapperpb.NewGeoJSON(r.Result),
	})
}

func (s *GeoSource) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var source = new(geov1beta1.GeoSource)
	for key, val := range raw {
		switch key {
		case "geo_fields":
			if err := json.Unmarshal(val, &source.GeoFields); err != nil {
				return err
			}
		case "prop_fields":
			if err := json.Unmarshal(val, &source.PropFields); err != nil {
				return err
			}
		case "table":
			var table = new(catalogv1beta1.Table)
			if err := json.Unmarshal(val, &table); err != nil {
				return err
			}
			source.Source = &geov1beta1.GeoSource_Table{
				Table: table,
			}
		case "query":
			var query = new(catalogv1beta1.Query)
			if err := json.Unmarshal(val, &query); err != nil {
				return err
			}
			source.Source = &geov1beta1.GeoSource_Query{
				Query: query,
			}
		}
	}

	*s = GeoSource{GeoSource: source}

	return nil
}

func (b *Bounds) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var bounds = new(geov1beta1.Bounds)
	for key, val := range raw {
		switch key {
		case "type":
			var boundsType string
			if err := json.Unmarshal(val, &boundsType); err != nil {
				return err
			} else if enumVal, ok := geov1beta1.BoundsType_value[boundsType]; ok {
				bounds.Type = geov1beta1.BoundsType(enumVal)
			}
		case "centroid":
			if err := json.Unmarshal(val, &bounds.Centroid); err != nil {
				return err
			}
		}
	}

	if raw["geom"] != nil {
		var geoJSON geojsonwrapperpb.GeoJSON
		if err := json.Unmarshal(raw["geom"], &geoJSON); err != nil {
			return err
		}
		bounds.Geom = geoJSON.GeoJSON.GetGeometry()
	}

	*b = Bounds{Bounds: bounds}

	return nil
}
