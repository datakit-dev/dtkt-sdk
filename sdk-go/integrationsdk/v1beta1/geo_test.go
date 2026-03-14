package v1beta1_test

// import (
// 	"encoding/json"
// 	"testing"

// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1/geojsonwrapperpb"
// 	catalogv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/catalog/v1beta1"
// 	geov1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geo/v1beta1"
// 	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"
// 	"google.golang.org/protobuf/encoding/protojson"
// 	"google.golang.org/protobuf/proto"
// )

// const reqRaw = `{"source":{"table":{"schema":{"name":"geo_us_boundaries","catalog":{"name":"bigquery-public-data"}},"name":"states"},"geo_field":"state_geom","prop_fields":["geo_id","state_fips_code","state_name","state"]},"bounds":{"type":"BOUNDS_TYPE_COVERS","geom":{"type":"Polygon","coordinates":[[[-111.290492270536,47.5080344121109],[-111.290489466303,47.5084460198696],[-111.29069144007,47.50844717663],[-111.29069479445,47.5080353433743],[-111.290492270536,47.5080344121109]]]},"centroid":false},"limit":1}`

// const geoJSONRaw = `{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-111.290492270536,47.5080344121109],[-111.290489466303,47.5084460198696],[-111.29069144007,47.50844717663],[-111.29069479445,47.5080353433743],[-111.290492270536,47.5080344121109]]]},"properties":{"county_name":"Cascade","foo":"bar"}}`

// func Test_StreamGeoJSON_ToFromProto(t *testing.T) {
// 	var req v1beta1.StreamGeoJsonRequest
// 	err := json.Unmarshal([]byte(reqRaw), &req)
// 	if err != nil {
// 		t.Fatal(err)
// 	} else if req.Bounds.Type != geov1beta1.BoundsType_BOUNDS_TYPE_COVERS {
// 		t.Fatalf("expected %s, got: %s", geov1beta1.BoundsType_BOUNDS_TYPE_COVERS, req.Bounds.Type)
// 	} else if req.GetBounds() == nil {
// 		t.Fatal("expected bounds")
// 	}

// 	var req2 = new(geov1beta1.StreamGeoJsonRequest)
// 	req2.Source = &geov1beta1.GeoSource{
// 		Source: &geov1beta1.GeoSource_Table{
// 			Table: &catalogv1beta1.Table{
// 				Name: "foobar",
// 			},
// 		},
// 	}
// 	proto.Merge(req2, req)

// 	_, err = protojson.Marshal(req2)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req2 = new(geov1beta1.StreamGeoJsonRequest)
// 	req2.Source = &geov1beta1.GeoSource{
// 		Source: &geov1beta1.GeoSource_Table{
// 			Table: &catalogv1beta1.Table{
// 				Name: "foobar",
// 			},
// 		},
// 	}
// 	req2.Bounds = &geov1beta1.Bounds{
// 		Type: geov1beta1.BoundsType_BOUNDS_TYPE_COVERS,
// 		Geom: &geojsonv1beta1.Geometry{
// 			Geometry: &geojsonv1beta1.Geometry_Polygon{
// 				Polygon: &geojsonv1beta1.Polygon{
// 					Type: "Polygon",
// 					Coordinates: []*geojsonv1beta1.LinearRing{
// 						{Positions: []*geojsonv1beta1.Position{
// 							{Values: []float64{1.2, 3, 4}},
// 						}},
// 					},
// 				},
// 			},
// 		},
// 		Centroid: true,
// 	}

// 	_, err = json.Marshal(req2)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	geoJSON, err := geojsonwrapperpb.UnmarshalGeoJSON("Feature", []byte(geoJSONRaw))
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	var resp = new(geov1beta1.StreamGeoJsonResponse)
// 	resp.Result = geoJSON
// 	_, err = json.Marshal(resp)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = v1beta1.StreamGeoJsonResponse{StreamGeoJsonResponse: resp}.MarshalJSON()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
