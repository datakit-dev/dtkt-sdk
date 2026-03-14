package geojsonwrapperpb

import (
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

type MultiPoint struct {
	*geojsonv1beta1.MultiPoint
	geom *geom.MultiPoint
}

func (mp MultiPoint) MarshalJSON() ([]byte, error) {
	var g *geom.MultiPoint
	if mp.geom != nil {
		g = mp.geom
	} else {
		var coords []float64
		for _, pos := range mp.Coordinates {
			coords = append(coords, pos.Values...)
		}
		g = geom.NewMultiPointFlat(geom.XY, coords)
	}
	return geojson.Marshal(g)
}

func (mp *MultiPoint) UnmarshalJSON(data []byte) error {
	var g geom.T
	if err := geojson.Unmarshal(data, &g); err != nil {
		return err
	}

	mpGeom, ok := g.(*geom.MultiPoint)
	if !ok {
		return fmt.Errorf("expected MultiPoint, got %T", g)
	}

	mp.MultiPoint = &geojsonv1beta1.MultiPoint{
		Type:        "MultiPoint",
		Coordinates: []*geojsonv1beta1.Position{},
	}

	coords := mpGeom.FlatCoords()
	for i := 0; i < len(coords); i += 2 {
		mp.Coordinates = append(mp.Coordinates, &geojsonv1beta1.Position{
			Values: coords[i : i+2],
		})
	}

	mp.geom = mpGeom
	return nil
}
