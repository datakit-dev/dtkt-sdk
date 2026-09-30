package geojsonwrapperpb

import (
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

type Polygon struct {
	*geojsonv1beta1.Polygon
	geom *geom.Polygon
}

func (p Polygon) MarshalJSON() ([]byte, error) {
	var g *geom.Polygon
	if p.geom != nil {
		g = p.geom
	} else {
		var flatCoords []float64
		var ends []int
		for _, ring := range p.Coordinates {
			for _, pos := range ring.Positions {
				flatCoords = append(flatCoords, pos.Values...)
			}
			ends = append(ends, len(flatCoords))
		}
		g = geom.NewPolygonFlat(geom.XY, flatCoords, ends)
	}
	return geojson.Marshal(g)
}

func (p *Polygon) UnmarshalJSON(data []byte) error {
	var g geom.T
	if err := geojson.Unmarshal(data, &g); err != nil {
		return err
	}

	poly, ok := g.(*geom.Polygon)
	if !ok {
		return fmt.Errorf("expected Polygon, got %T", g)
	}

	p.Polygon = &geojsonv1beta1.Polygon{
		Type:        "Polygon",
		Coordinates: []*geojsonv1beta1.LinearRing{},
	}

	coords := poly.FlatCoords()
	offset := 0
	for _, end := range poly.Ends() {
		ring := &geojsonv1beta1.LinearRing{}
		for i := offset; i < end; i += 2 {
			ring.Positions = append(ring.Positions, &geojsonv1beta1.Position{
				Values: coords[i : i+2],
			})
		}
		p.Coordinates = append(p.Coordinates, ring)
		offset = end
	}

	p.geom = poly
	return nil
}
