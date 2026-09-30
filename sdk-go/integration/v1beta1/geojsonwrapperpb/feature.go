package geojsonwrapperpb

import (
	"encoding/json"
	"fmt"

	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"
	"github.com/mmcloughlin/geohash"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type Feature struct {
	*geojsonv1beta1.Feature
	id    any
	geom  geom.T
	bbox  []float64
	props map[string]any
}

func NewFeature(f *geojsonv1beta1.Feature) *Feature {
	return &Feature{
		Feature: f,
	}
}

func NewFeatureFromGeom(id any, geom geom.T, props map[string]any) *Feature {
	return &Feature{
		id:    id,
		geom:  geom,
		props: props,
	}
}

func (f *Feature) loadGeom() error {
	if f.geom == nil && f.Feature != nil && f.Geometry != nil {
		geom, err := ToGeom(f.Geometry)
		if err != nil {
			return err
		}
		f.geom = geom
	}

	if f.geom == nil {
		return fmt.Errorf("feature missing geometry")
	}

	return nil
}

func (f *Feature) SetID(id any) {
	f.id = id
}

func (f *Feature) ID() (any, error) {
	if f.id == nil && f.Feature != nil && f.Id != nil {
		if f.GetIdNum() != 0 {
			f.id = f.GetIdNum()
		} else if f.GetIdStr() != "" {
			f.id = f.GetIdStr()
		}
	}

	if f.id == nil {
		return nil, fmt.Errorf("feature id is nil")
	}

	return f.id, nil
}

func (f *Feature) Center() (lat, lng float64, err error) {
	bbox, err := f.BBox()
	if err != nil {
		return 0, 0, err
	}

	if len(f.bbox) > 3 {
		lat, lng = (geohash.Box{
			MinLat: bbox[0],
			MinLng: bbox[1],
			MaxLat: bbox[2],
			MaxLng: bbox[3],
		}).Center()
		return
	}
	return 0, 0, fmt.Errorf("feature missing bbox")
}

func (f *Feature) GeoHashUint64() (uint64, error) {
	lat, lng, err := f.Center()
	if err != nil {
		return 0, err
	}
	return geohash.EncodeInt(lat, lng), nil
}

func (f *Feature) GeoHashString() (string, error) {
	lat, lng, err := f.Center()
	if err != nil {
		return "", err
	}
	return geohash.Encode(lat, lng), nil
}

func (f *Feature) Geom() (geom.T, error) {
	return f.geom, nil
}

func (f *Feature) BBox() ([]float64, error) {
	if f.bbox == nil {
		if f.Feature != nil && f.Bbox != nil {
			f.bbox = f.Bbox.Coordinates
		} else if err := f.loadGeom(); err != nil {
			return nil, err
		} else {
			switch f.geom.Bounds().Layout() {
			case geom.XY:
				f.bbox = append(f.bbox,
					f.geom.Bounds().Min(0),
					f.geom.Bounds().Min(1),
					f.geom.Bounds().Max(0),
					f.geom.Bounds().Max(1),
				)
			}
		}
	}

	return f.bbox, nil
}

func (f *Feature) Properties() map[string]any {
	if f.props == nil && f.Feature.Properties != nil {
		f.props = f.Feature.Properties.AsMap()
	}

	return f.props
}

func (f Feature) MarshalJSON() ([]byte, error) {
	if err := f.loadGeom(); err != nil {
		return nil, err
	}

	geometry, err := geojson.Encode(f.geom)
	if err != nil {
		return nil, err
	}

	id, err := f.ID()
	if err != nil {
		return nil, err
	}

	bbox, err := f.BBox()
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]any{
		"type":       "Feature",
		"id":         id,
		"geometry":   geometry,
		"properties": f.Properties(),
		"bbox":       bbox,
	})
}

func (f *Feature) UnmarshalJSON(data []byte) error {
	var feat geojson.Feature
	if err := json.Unmarshal(data, &feat); err != nil {
		return err
	}

	var id any
	if feat.ID != "" {
		var idNum int64
		if _, err := fmt.Sscanf(feat.ID, "%d", &idNum); err == nil {
			id = idNum
		} else {
			id = feat.ID
		}
	}

	var bbox []float64
	if feat.BBox != nil {
		switch feat.BBox.Layout() {
		case geom.XY:
			bbox = append(bbox,
				feat.BBox.Min(0),
				feat.BBox.Min(1),
				feat.BBox.Max(0),
				feat.BBox.Max(1),
			)
		}
	}

	*f = Feature{
		id:    id,
		geom:  feat.Geometry,
		props: feat.Properties,
		bbox:  bbox,
	}

	return nil
}

func (f *Feature) ToProto() (*geojsonv1beta1.Feature, error) {
	if f.Feature == nil {
		f.Feature = &geojsonv1beta1.Feature{
			Type: "Feature",
		}
	}

	if f.Geometry == nil && f.geom != nil {
		geom, err := FromGeom(f.geom)
		if err != nil {
			return nil, err
		}
		f.Geometry = geom
	} else {
		return nil, fmt.Errorf("feature missing geometry")
	}

	if f.Feature.Properties == nil && f.props != nil {
		props, err := structpb.NewStruct(f.props)
		if err != nil {
			return nil, err
		}
		f.Feature.Properties = props
	}

	if f.Id == nil && f.id != nil {
		switch id := f.id.(type) {
		case string:
			f.Id = &geojsonv1beta1.Feature_IdStr{IdStr: id}
		case int64:
			f.Id = &geojsonv1beta1.Feature_IdNum{IdNum: id}
		}
	}

	return f.Feature, nil
}
