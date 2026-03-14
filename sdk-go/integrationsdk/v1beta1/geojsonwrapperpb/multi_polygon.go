package geojsonwrapperpb

import (
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

type MultiPolygon struct {
	*geojsonv1beta1.MultiPolygon
	geom *geom.MultiPolygon
}

func (mp MultiPolygon) MarshalJSON() ([]byte, error) {
	var g *geom.MultiPolygon
	if mp.geom != nil {
		g = mp.geom
	} else {
		var flatCoords []float64
		var endss [][]int

		for _, poly := range mp.Coordinates {
			var polyEnds []int
			for _, ring := range poly.LinearRings {
				for _, pos := range ring.Positions {
					flatCoords = append(flatCoords, pos.Values...)
				}
				polyEnds = append(polyEnds, len(flatCoords))
			}
			endss = append(endss, polyEnds)
		}

		g = geom.NewMultiPolygonFlat(geom.XY, flatCoords, endss)
	}
	return geojson.Marshal(g)
}

func (mp *MultiPolygon) UnmarshalJSON(data []byte) error {
	var g geom.T
	if err := geojson.Unmarshal(data, &g); err != nil {
		return err
	}

	mpGeom, ok := g.(*geom.MultiPolygon)
	if !ok {
		return fmt.Errorf("expected MultiPolygon, got %T", g)
	}

	mp.MultiPolygon = &geojsonv1beta1.MultiPolygon{
		Type:        "MultiPolygon",
		Coordinates: []*geojsonv1beta1.PolygonCoords{},
	}

	coords := mpGeom.FlatCoords()
	endss := mpGeom.Endss()
	offset := 0

	for _, polyEnds := range endss {
		pbPoly := &geojsonv1beta1.PolygonCoords{}
		for _, end := range polyEnds {
			ring := &geojsonv1beta1.LinearRing{}
			for i := offset; i < end; i += 2 {
				ring.Positions = append(ring.Positions, &geojsonv1beta1.Position{
					Values: coords[i : i+2],
				})
			}
			pbPoly.LinearRings = append(pbPoly.LinearRings, ring)
			offset = end
		}
		mp.Coordinates = append(mp.Coordinates, pbPoly)
	}

	mp.geom = mpGeom
	return nil
}
