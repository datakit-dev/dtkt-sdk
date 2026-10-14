package geojsonwrapperpb

import (
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

type LineString struct {
	*geojsonv1beta1.LineString
	geom *geom.LineString
}

func (ls LineString) MarshalJSON() ([]byte, error) {
	var g *geom.LineString
	if ls.geom != nil {
		g = ls.geom
	} else {
		coords := []float64{}
		for _, pos := range ls.Coordinates {
			coords = append(coords, pos.Values...)
		}
		g = geom.NewLineStringFlat(geom.XY, coords)
	}
	return geojson.Marshal(g)
}

func (ls *LineString) UnmarshalJSON(data []byte) error {
	var g geom.T
	if err := geojson.Unmarshal(data, &g); err != nil {
		return err
	}

	line, ok := g.(*geom.LineString)
	if !ok {
		return fmt.Errorf("expected LineString, got %T", g)
	}

	ls.LineString = &geojsonv1beta1.LineString{
		Type:        "LineString",
		Coordinates: []*geojsonv1beta1.Position{},
	}
	for i := 0; i < len(line.FlatCoords()); i += 2 {
		ls.Coordinates = append(ls.Coordinates, &geojsonv1beta1.Position{
			Values: line.FlatCoords()[i : i+2],
		})
	}
	ls.geom = line
	return nil
}
