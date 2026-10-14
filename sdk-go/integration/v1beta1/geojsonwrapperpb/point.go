package geojsonwrapperpb

import (
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

type Point struct {
	*geojsonv1beta1.Point
	geom *geom.Point
}

func (p Point) MarshalJSON() ([]byte, error) {
	var g *geom.Point
	if p.geom != nil {
		g = p.geom
	} else {
		g = geom.NewPointFlat(geom.XY, p.Coordinates.Values)
	}
	return geojson.Marshal(g)
}

func (p *Point) UnmarshalJSON(data []byte) error {
	var g geom.T
	if err := geojson.Unmarshal(data, &g); err != nil {
		return err
	}

	pt, ok := g.(*geom.Point)
	if !ok {
		return fmt.Errorf("expected Point, got %T", g)
	}

	p.Point = &geojsonv1beta1.Point{
		Type:        "Point",
		Coordinates: &geojsonv1beta1.Position{Values: pt.FlatCoords()},
	}
	p.geom = pt
	return nil
}
