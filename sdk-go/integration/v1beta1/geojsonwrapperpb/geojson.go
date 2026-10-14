package geojsonwrapperpb

import (
	"encoding/json"
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"
)

type GeoJSON struct {
	GeoJSON *geojsonv1beta1.GeoJSON
}

func NewGeoJSON(g *geojsonv1beta1.GeoJSON) *GeoJSON {
	return &GeoJSON{g}
}

func (g GeoJSON) MarshalJSON() ([]byte, error) {
	if g.GeoJSON.GetGeometry() != nil {
		if g.GeoJSON.GetGeometry().GetPoint() != nil {
			return Point{Point: g.GeoJSON.GetGeometry().GetPoint()}.MarshalJSON()
		} else if g.GeoJSON.GetGeometry().GetMultiPoint() != nil {
			return MultiPoint{MultiPoint: g.GeoJSON.GetGeometry().GetMultiPoint()}.MarshalJSON()
		} else if g.GeoJSON.GetGeometry().GetLineString() != nil {
			return LineString{LineString: g.GeoJSON.GetGeometry().GetLineString()}.MarshalJSON()
		} else if g.GeoJSON.GetGeometry().GetMultiLineString() != nil {
			return MultiLineString{MultiLineString: g.GeoJSON.GetGeometry().GetMultiLineString()}.MarshalJSON()
		} else if g.GeoJSON.GetGeometry().GetPolygon() != nil {
			return Polygon{Polygon: g.GeoJSON.GetGeometry().GetPolygon()}.MarshalJSON()
		} else if g.GeoJSON.GetGeometry().GetMultiPolygon() != nil {
			return MultiPolygon{MultiPolygon: g.GeoJSON.GetGeometry().GetMultiPolygon()}.MarshalJSON()
		} else if g.GeoJSON.GetGeometry().GetGeometryCollection() != nil {
			return GeometryCollection{GeometryCollection: g.GeoJSON.GetGeometry().GetGeometryCollection()}.MarshalJSON()
		} else {
			return nil, fmt.Errorf("expected valid Geometry, got: %s", g.GeoJSON.GetGeometry())
		}
	} else if g.GeoJSON.GetFeatureCollection() != nil {
		return FeatureCollection{FeatureCollection: g.GeoJSON.GetFeatureCollection()}.MarshalJSON()
	} else if g.GeoJSON.GetFeature() != nil {
		return Feature{Feature: g.GeoJSON.GetFeature()}.MarshalJSON()
	}

	return nil, fmt.Errorf("expected valid GeoJSON, got: %s", g)
}

func (g *GeoJSON) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var geoType string
	if err := json.Unmarshal(raw["type"], &geoType); err != nil {
		return err
	}

	geoJSON, err := UnmarshalGeoJSON(geoType, data)
	if err != nil {
		return err
	}

	*g = GeoJSON{GeoJSON: geoJSON}
	return nil
}

func UnmarshalGeoJSON(geoType string, geoJSON []byte) (*geojsonv1beta1.GeoJSON, error) {
	var result = &geojsonv1beta1.GeoJSON{}
	switch geoType {
	case "Feature":
		wrapped := new(Feature)
		if err := wrapped.UnmarshalJSON(geoJSON); err != nil {
			return nil, fmt.Errorf("invalid Feature (%#v): %w", geoJSON, err)
		}

		feature, err := wrapped.ToProto()
		if err != nil {
			return nil, err
		}

		result.Geojson = &geojsonv1beta1.GeoJSON_Feature{
			Feature: feature,
		}

		return result, nil
	case "FeatureCollection":
		wrapped := new(FeatureCollection)
		if err := wrapped.UnmarshalJSON(geoJSON); err != nil {
			return nil, fmt.Errorf("invalid FeatureCollection (%#v): %w", geoJSON, err)
		}

		result.Geojson = &geojsonv1beta1.GeoJSON_FeatureCollection{
			FeatureCollection: wrapped.FeatureCollection,
		}

		return result, nil
	case "Point":
		wrapped := new(Point)
		if err := wrapped.UnmarshalJSON(geoJSON); err != nil {
			return nil, fmt.Errorf("invalid Point (%#v): %w", geoJSON, err)
		}
		result.Geojson = &geojsonv1beta1.GeoJSON_Geometry{
			Geometry: &geojsonv1beta1.Geometry{
				Geometry: &geojsonv1beta1.Geometry_Point{
					Point: wrapped.Point,
				},
			},
		}

		return result, nil
	case "MultiPoint":
		wrapped := new(MultiPoint)
		if err := wrapped.UnmarshalJSON(geoJSON); err != nil {
			return nil, fmt.Errorf("invalid MultiPoint (%#v): %w", geoJSON, err)
		}
		result.Geojson = &geojsonv1beta1.GeoJSON_Geometry{
			Geometry: &geojsonv1beta1.Geometry{
				Geometry: &geojsonv1beta1.Geometry_MultiPoint{
					MultiPoint: wrapped.MultiPoint,
				},
			},
		}

		return result, nil
	case "LineString":
		wrapped := new(LineString)
		if err := wrapped.UnmarshalJSON(geoJSON); err != nil {
			return nil, fmt.Errorf("invalid LineString (%#v): %w", geoJSON, err)
		}
		result.Geojson = &geojsonv1beta1.GeoJSON_Geometry{
			Geometry: &geojsonv1beta1.Geometry{
				Geometry: &geojsonv1beta1.Geometry_LineString{
					LineString: wrapped.LineString,
				},
			},
		}

		return result, nil
	case "MultiLineString":
		wrapped := new(MultiLineString)
		if err := wrapped.UnmarshalJSON(geoJSON); err != nil {
			return nil, fmt.Errorf("invalid MultiLineString (%#v): %w", geoJSON, err)
		}
		result.Geojson = &geojsonv1beta1.GeoJSON_Geometry{
			Geometry: &geojsonv1beta1.Geometry{
				Geometry: &geojsonv1beta1.Geometry_MultiLineString{
					MultiLineString: wrapped.MultiLineString,
				},
			},
		}

		return result, nil
	case "Polygon":
		wrapped := new(Polygon)
		if err := wrapped.UnmarshalJSON(geoJSON); err != nil {
			return nil, fmt.Errorf("invalid Polygon (%#v): %w", geoJSON, err)
		}
		result.Geojson = &geojsonv1beta1.GeoJSON_Geometry{
			Geometry: &geojsonv1beta1.Geometry{
				Geometry: &geojsonv1beta1.Geometry_Polygon{
					Polygon: wrapped.Polygon,
				},
			},
		}

		return result, nil
	case "MultiPolygon":
		wrapped := new(MultiPolygon)
		if err := wrapped.UnmarshalJSON(geoJSON); err != nil {
			return nil, fmt.Errorf("invalid MultiPolygon (%#v): %w", geoJSON, err)
		}
		result.Geojson = &geojsonv1beta1.GeoJSON_Geometry{
			Geometry: &geojsonv1beta1.Geometry{
				Geometry: &geojsonv1beta1.Geometry_MultiPolygon{
					MultiPolygon: wrapped.MultiPolygon,
				},
			},
		}

		return result, nil
	case "GeometryCollection":
		wrapped := new(GeometryCollection)
		if err := wrapped.UnmarshalJSON(geoJSON); err != nil {
			return nil, fmt.Errorf("invalid GeometryCollection (%#v): %w", geoJSON, err)
		}
		result.Geojson = &geojsonv1beta1.GeoJSON_Geometry{
			Geometry: &geojsonv1beta1.Geometry{
				Geometry: &geojsonv1beta1.Geometry_GeometryCollection{
					GeometryCollection: wrapped.GeometryCollection,
				},
			},
		}

		return result, nil
	default:
		return nil, fmt.Errorf("expected valid Geometry type, got: %s", geoType)
	}
}
