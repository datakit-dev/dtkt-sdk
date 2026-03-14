package geojsonwrapperpb

import (
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

type GeometryCollection struct {
	*geojsonv1beta1.GeometryCollection
	geom *geom.GeometryCollection
}

func (gc GeometryCollection) MarshalJSON() ([]byte, error) {
	var g *geom.GeometryCollection
	if gc.geom != nil {
		g = gc.geom
	} else {
		g = geom.NewGeometryCollection()
		for _, sub := range gc.Geometries {
			subGeom, err := ToGeom(sub)
			if err != nil {
				return nil, fmt.Errorf("marshal GeometryCollection: %w", err)
			}
			g.MustPush(subGeom)
		}
	}
	return geojson.Marshal(g)
}

func (gc *GeometryCollection) UnmarshalJSON(data []byte) error {
	var g geom.T
	if err := geojson.Unmarshal(data, &g); err != nil {
		return err
	}

	collection, ok := g.(*geom.GeometryCollection)
	if !ok {
		return fmt.Errorf("expected GeometryCollection, got %T", g)
	}

	gc.GeometryCollection = &geojsonv1beta1.GeometryCollection{
		Type:       "GeometryCollection",
		Geometries: []*geojsonv1beta1.Geometry{},
	}

	for _, sub := range collection.Geoms() {
		pbGeom, err := FromGeom(sub)
		if err != nil {
			return fmt.Errorf("unmarshal GeometryCollection: %w", err)
		}
		gc.Geometries = append(gc.Geometries, pbGeom)
	}

	gc.geom = collection
	return nil
}
