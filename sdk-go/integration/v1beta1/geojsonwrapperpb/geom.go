package geojsonwrapperpb

import (
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"
	"github.com/twpayne/go-geom"
)

func ToGeom(pbGeom *geojsonv1beta1.Geometry) (geom.T, error) {
	switch g := pbGeom.Geometry.(type) {
	case *geojsonv1beta1.Geometry_Point:
		return geom.NewPointFlat(geom.XY, g.Point.Coordinates.Values), nil

	case *geojsonv1beta1.Geometry_MultiPoint:
		var coords []float64
		for _, p := range g.MultiPoint.Coordinates {
			coords = append(coords, p.Values...)
		}
		return geom.NewMultiPointFlat(geom.XY, coords), nil

	case *geojsonv1beta1.Geometry_LineString:
		var coords []float64
		for _, p := range g.LineString.Coordinates {
			coords = append(coords, p.Values...)
		}
		return geom.NewLineStringFlat(geom.XY, coords), nil

	case *geojsonv1beta1.Geometry_MultiLineString:
		var flatCoords []float64
		var ends []int
		for _, ls := range g.MultiLineString.Coordinates {
			for _, p := range ls.Positions {
				flatCoords = append(flatCoords, p.Values...)
			}
			ends = append(ends, len(flatCoords))
		}
		return geom.NewMultiLineStringFlat(geom.XY, flatCoords, ends), nil

	case *geojsonv1beta1.Geometry_Polygon:
		var flatCoords []float64
		var ends []int
		for _, ring := range g.Polygon.Coordinates {
			for _, p := range ring.Positions {
				flatCoords = append(flatCoords, p.Values...)
			}
			ends = append(ends, len(flatCoords))
		}
		return geom.NewPolygonFlat(geom.XY, flatCoords, ends), nil

	case *geojsonv1beta1.Geometry_MultiPolygon:
		var flatCoords []float64
		var endss [][]int
		for _, poly := range g.MultiPolygon.Coordinates {
			var polyEnds []int
			for _, ring := range poly.LinearRings {
				for _, p := range ring.Positions {
					flatCoords = append(flatCoords, p.Values...)
				}
				polyEnds = append(polyEnds, len(flatCoords))
			}
			endss = append(endss, polyEnds)
		}
		return geom.NewMultiPolygonFlat(geom.XY, flatCoords, endss), nil

	case *geojsonv1beta1.Geometry_GeometryCollection:
		gc := geom.NewGeometryCollection()
		for _, sub := range g.GeometryCollection.Geometries {
			geomT, err := ToGeom(sub)
			if err != nil {
				return nil, err
			} else if err = gc.Push(geomT); err != nil {
				return nil, err
			}
		}
		return gc, nil

	default:
		return nil, fmt.Errorf("unsupported proto geometry type: %T", g)
	}
}

func FromGeom(g geom.T) (*geojsonv1beta1.Geometry, error) {
	switch geom := g.(type) {

	case *geom.Point:
		return &geojsonv1beta1.Geometry{
			Geometry: &geojsonv1beta1.Geometry_Point{
				Point: &geojsonv1beta1.Point{
					Type:        "Point",
					Coordinates: &geojsonv1beta1.Position{Values: geom.FlatCoords()},
				},
			},
		}, nil

	case *geom.MultiPoint:
		var coords []*geojsonv1beta1.Position
		for i := 0; i < len(geom.FlatCoords()); i += 2 {
			coords = append(coords, &geojsonv1beta1.Position{
				Values: geom.FlatCoords()[i : i+2],
			})
		}
		return &geojsonv1beta1.Geometry{
			Geometry: &geojsonv1beta1.Geometry_MultiPoint{
				MultiPoint: &geojsonv1beta1.MultiPoint{
					Type:        "MultiPoint",
					Coordinates: coords,
				},
			},
		}, nil

	case *geom.LineString:
		var coords []*geojsonv1beta1.Position
		for i := 0; i < len(geom.FlatCoords()); i += 2 {
			coords = append(coords, &geojsonv1beta1.Position{
				Values: geom.FlatCoords()[i : i+2],
			})
		}
		return &geojsonv1beta1.Geometry{
			Geometry: &geojsonv1beta1.Geometry_LineString{
				LineString: &geojsonv1beta1.LineString{
					Type:        "LineString",
					Coordinates: coords,
				},
			},
		}, nil

	case *geom.MultiLineString:
		var offset int
		var lines []*geojsonv1beta1.LineStringCoords
		for _, end := range geom.Ends() {
			line := &geojsonv1beta1.LineStringCoords{}
			for i := offset; i < end; i += 2 {
				line.Positions = append(line.Positions, &geojsonv1beta1.Position{
					Values: geom.FlatCoords()[i : i+2],
				})
			}
			lines = append(lines, line)
			offset = end
		}
		return &geojsonv1beta1.Geometry{
			Geometry: &geojsonv1beta1.Geometry_MultiLineString{
				MultiLineString: &geojsonv1beta1.MultiLineString{
					Type:        "MultiLineString",
					Coordinates: lines,
				},
			},
		}, nil

	case *geom.Polygon:
		var offset int
		var rings []*geojsonv1beta1.LinearRing
		for _, end := range geom.Ends() {
			ring := &geojsonv1beta1.LinearRing{}
			for i := offset; i < end; i += 2 {
				ring.Positions = append(ring.Positions, &geojsonv1beta1.Position{
					Values: geom.FlatCoords()[i : i+2],
				})
			}
			rings = append(rings, ring)
			offset = end
		}
		return &geojsonv1beta1.Geometry{
			Geometry: &geojsonv1beta1.Geometry_Polygon{
				Polygon: &geojsonv1beta1.Polygon{
					Type:        "Polygon",
					Coordinates: rings,
				},
			},
		}, nil

	case *geom.MultiPolygon:
		var offset int
		var polys []*geojsonv1beta1.PolygonCoords
		for _, polyEnds := range geom.Endss() {
			poly := &geojsonv1beta1.PolygonCoords{}
			for _, end := range polyEnds {
				ring := &geojsonv1beta1.LinearRing{}
				for i := offset; i < end; i += 2 {
					ring.Positions = append(ring.Positions, &geojsonv1beta1.Position{
						Values: geom.FlatCoords()[i : i+2],
					})
				}
				poly.LinearRings = append(poly.LinearRings, ring)
				offset = end
			}
			polys = append(polys, poly)
		}
		return &geojsonv1beta1.Geometry{
			Geometry: &geojsonv1beta1.Geometry_MultiPolygon{
				MultiPolygon: &geojsonv1beta1.MultiPolygon{
					Type:        "MultiPolygon",
					Coordinates: polys,
				},
			},
		}, nil

	case *geom.GeometryCollection:
		var geoms []*geojsonv1beta1.Geometry
		for _, sub := range geom.Geoms() {
			pbGeom, err := FromGeom(sub)
			if err != nil {
				return nil, err
			}
			geoms = append(geoms, pbGeom)
		}
		return &geojsonv1beta1.Geometry{
			Geometry: &geojsonv1beta1.Geometry_GeometryCollection{
				GeometryCollection: &geojsonv1beta1.GeometryCollection{
					Type:       "GeometryCollection",
					Geometries: geoms,
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported geom.T type: %T", g)
	}
}
