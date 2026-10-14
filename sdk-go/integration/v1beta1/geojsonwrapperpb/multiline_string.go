package geojsonwrapperpb

import (
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

type MultiLineString struct {
	*geojsonv1beta1.MultiLineString
	geom *geom.MultiLineString
}

func (mls MultiLineString) MarshalJSON() ([]byte, error) {
	var g *geom.MultiLineString
	if mls.geom != nil {
		g = mls.geom
	} else {
		var flatCoords []float64
		var ends []int
		for _, line := range mls.Coordinates {
			for _, pos := range line.Positions {
				flatCoords = append(flatCoords, pos.Values...)
			}
			ends = append(ends, len(flatCoords))
		}
		g = geom.NewMultiLineStringFlat(geom.XY, flatCoords, ends)
	}
	return geojson.Marshal(g)
}

func (mls *MultiLineString) UnmarshalJSON(data []byte) error {
	var g geom.T
	if err := geojson.Unmarshal(data, &g); err != nil {
		return err
	}

	mlsGeom, ok := g.(*geom.MultiLineString)
	if !ok {
		return fmt.Errorf("expected MultiLineString, got %T", g)
	}

	mls.MultiLineString = &geojsonv1beta1.MultiLineString{
		Type:        "MultiLineString",
		Coordinates: []*geojsonv1beta1.LineStringCoords{},
	}

	coords := mlsGeom.FlatCoords()
	offset := 0
	for _, end := range mlsGeom.Ends() {
		line := &geojsonv1beta1.LineStringCoords{}
		for i := offset; i < end; i += 2 {
			line.Positions = append(line.Positions, &geojsonv1beta1.Position{
				Values: coords[i : i+2],
			})
		}
		mls.Coordinates = append(mls.Coordinates, line)
		offset = end
	}

	mls.geom = mlsGeom
	return nil
}
