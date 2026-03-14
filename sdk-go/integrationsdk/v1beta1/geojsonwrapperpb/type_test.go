package geojsonwrapperpb_test

import (
	"fmt"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1/geojsonwrapperpb"
)

const geoJSONRaw = `{"type":"Feature","id":"1","geometry":{"type":"Polygon","coordinates":[[[-111.290492270536,47.5080344121109],[-111.290489466303,47.5084460198696],[-111.29069144007,47.50844717663],[-111.29069479445,47.5080353433743],[-111.290492270536,47.5080344121109]]]},"properties":{"county_name":"Cascade"}}`

func Test_GeoJSON_ToFromProto(t *testing.T) {
	geoJSON, err := geojsonwrapperpb.UnmarshalGeoJSON("Feature", []byte(geoJSONRaw))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(geoJSONRaw)

	bytes, err := geojsonwrapperpb.NewGeoJSON(geoJSON).MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(bytes))
}
