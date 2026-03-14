package common_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1/geojsonwrapperpb"
	catalogv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/catalog/v1beta1"
	geojsonv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/geojson/v1beta1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func assertEqual[T any](v1 T, v2 any) (T, error) {
	v2Any, err := common.WrapProtoAny(v2)
	if err != nil {
		return v1, err
	}
	return common.UnwrapProtoAnyAs[T](v2Any)
}

func TestAnyProto(t *testing.T) {
	var (
		nativeMap = map[string]any{
			"stringVal":  "string value",
			"int32Val":   int32(123),
			"int64Val":   int64(456),
			"uint32Val":  uint32(123),
			"uint64Val":  uint64(456),
			"float32Val": float32(123.456),
			"float64Val": float64(456.789),
			"boolVal":    true,
		}
		nativeSlice = []any{
			"string value", int32(123), int64(456), uint32(123), uint64(456), float32(123.456), float64(456.789), true,
		}
		nativeGeoJSON = []byte(`{"bbox":[-115.5291675385877,48.39543825655724,-115.52740542716921,48.396931593444556],"geometry":{"type":"MultiPolygon","coordinates":[[[[-115.52740542716921,48.39571803360954],[-115.5276652663668,48.39543825655724],[-115.52770356663623,48.395470665519],[-115.5291675385877,48.396700854361015],[-115.52886123909623,48.396931593444556],[-115.52744246310665,48.39574888352531],[-115.52740542716921,48.39571803360954]]]]},"id":6015,"properties":{"AddressL_1":"","AddressLin":"NORTHWOOD AVE","Assessment":"0000007957","COUNTYCD":56,"CareOfTaxp":"","Certificat":"","CityStateZ":"LIBBY, MT 59923","Continuous":0,"CountyAbbr":"LN","CountyName":"Lincoln","DbaName":"","FallowAcre":0,"FarmsiteAc":0,"ForestAcre":0,"GISAcres":1.48001605086,"GrazingAcr":0,"IrrigatedA":0,"LegalDescr":"NORTHWOOD MANOR, S02, T30 N, R31 W, BLOCK 1, Lot 6, ACRES 1.42","LevyDistri":"56-5521-4F","NonQualAcr":0,"OwnerAdd_1":"1195 MENDOCINO DR","OwnerAdd_2":"","OwnerAddre":"C/O PATRICK FOX","OwnerCity":"HELENA","OwnerName":"BLOOMGREN BLAINE E \u0026 ELLEN L TTEES","OwnerState":"MT","OwnerZipCo":"59601","PARCELID":"56417502202070000","PropAccess":"","PropType":"Vacant Land","PropertyID":794617,"Range":"31 W","Section":"02","Shape_Area":5989.41246067,"Shape_Leng":422.544002524,"Subdivisio":"NORTHWOOD MANOR","TaxYear":2025,"TotalAcres":1.42,"TotalBuild":0,"TotalLandV":138371,"TotalValue":138371,"Township":"30 N","WildHayAcr":0},"type":"Feature"}`)
	)

	protoMap, err := structpb.NewStruct(nativeMap)
	if err != nil {
		t.Fatal(err)
	}

	protoSlice, err := structpb.NewList(nativeSlice)
	if err != nil {
		t.Fatal(err)
	}

	geoJSON, err := geojsonwrapperpb.UnmarshalGeoJSON("Feature", nativeGeoJSON)
	if err != nil {
		t.Fatal(err)
	}

	var catalog = &catalogv1beta1.Catalog{
		Name:     "test_catalog",
		Metadata: protoMap,
	}

	var tests = []struct {
		value   any
		proto   proto.Message
		compare func(any, any) bool
	}{
		{
			value: nativeMap,
			proto: protoMap,
			compare: func(v1, v2 any) bool {
				if v1, ok := v1.(map[string]any); ok {
					if v2, ok := v2.(map[string]any); ok {
						for k1, v1 := range v1 {
							if v2, ok := v2[k1]; ok {
								v3, err := assertEqual(v1, v2)
								if err != nil {
									t.Log(err, v1, v2, v3)
									return false
								}
							}
						}
					}
				}
				return true
			},
		},
		{
			value: nativeSlice,
			proto: protoSlice,
			compare: func(v1, v2 any) bool {
				if v1, ok := v1.([]any); ok {
					if v2, ok := v2.([]any); ok && len(v1) == len(v2) {
						for idx, v1 := range v1 {
							v3, err := assertEqual(v1, v2[idx])
							if err != nil {
								t.Log(err, v1, v2, v3)
								return false
							}
						}
					}
				}
				return true
			},
		},
		{
			value: nativeGeoJSON,
			proto: geoJSON,
			compare: func(v1, v2 any) bool {
				if v1, ok := v1.(*geojsonv1beta1.GeoJSON); ok {
					if v2, ok := v2.(*wrapperspb.BytesValue); ok {
						var geoJSON = &geojsonwrapperpb.GeoJSON{}
						if err := geoJSON.UnmarshalJSON(v2.Value); err != nil {
							t.Fatal(err)
						}

						v3, err := assertEqual(v1, geoJSON.GeoJSON)
						if err != nil {
							t.Log(err, v1, v2, v3)
							return false
						}
					}
				}

				return true
			},
		},
		{
			value: "string value",
			proto: wrapperspb.String("string value"),
		},
		{
			value: int32(123),
			proto: wrapperspb.Int32(123),
		},
		{
			value: int64(456),
			proto: wrapperspb.Int64(456),
		},
		{
			value: uint32(123),
			proto: wrapperspb.UInt32(123),
		},
		{
			value: uint64(456),
			proto: wrapperspb.UInt64(456),
		},
		{
			value: float32(123.456),
			proto: wrapperspb.Float(123.456),
		},
		{
			value: float64(456.789),
			proto: wrapperspb.Double(456.789),
		},
		{
			value: true,
			proto: wrapperspb.Bool(true),
		},
		{
			value: []byte("here are some bytes"),
			proto: wrapperspb.Bytes([]byte("here are some bytes")),
			compare: func(v1, v2 any) bool {
				if v1, ok := v1.([]byte); ok {
					if v2, ok := v2.([]byte); ok {
						return string(v1) == string(v2)
					}
				}
				return false
			},
		},
		{
			value: catalog,
			proto: catalog,
			compare: func(v1, v2 any) bool {
				if v1, ok := v1.(proto.Message); ok {
					if v2, ok := v2.(proto.Message); ok {
						return proto.Equal(v1, v2)
					}
				}
				return false
			},
		},
	}

	for idx, test := range tests {
		protoAny, err := common.WrapProtoAny(test.value)
		if err != nil {
			t.Fatal(err)
		}

		protoVal, err := protoAny.UnmarshalNew()
		if err != nil {
			t.Fatal(err)
		}

		anyVal, err := common.UnwrapProtoAny(protoAny)
		if err != nil {
			t.Fatal(err)
		}

		var equal bool
		if test.compare != nil {
			equal = test.compare(test.value, anyVal)
		} else {
			if protoVal.ProtoReflect().Descriptor() != test.proto.ProtoReflect().Descriptor() {
				t.Fatalf("test %d expected descriptors to be equal, %T != %T", idx, protoVal, test.proto)
			}

			equal = test.value == anyVal
		}

		if !equal {
			t.Fatalf("test %d expected values to be equal, (%T) %#v != (%T) %#v", idx, test.value, test.value, anyVal, anyVal)
		}
	}
}

func TestProtoWrapNumbers(t *testing.T) {
	var (
		floatVal = float64(1234.123124)
		intVal   = int64(floatVal)
		uintVal  = uint64(18446744073709551615)
	)

	wrapVal, err := common.WrapProtoAny(floatVal)
	if err != nil {
		t.Fatal(err)
	}

	unwrapFloat, err := common.UnwrapProtoAnyAs[float64](wrapVal)
	if err != nil {
		t.Fatal(err)
	} else if unwrapFloat != floatVal {
		t.Fatalf("expected: %v, got: %v", floatVal, unwrapFloat)
	}

	unwrapInt, err := common.UnwrapProtoAnyAs[int64](wrapVal)
	if err != nil {
		t.Fatal(err)
	} else if unwrapInt != intVal {
		t.Fatalf("expected: %v, got: %v", intVal, unwrapInt)
	}

	wrapVal, err = common.WrapProtoAny(intVal)
	if err != nil {
		t.Fatal(err)
	}

	unwrapFloat, err = common.UnwrapProtoAnyAs[float64](wrapVal)
	if err != nil {
		t.Fatal(err)
	} else if unwrapFloat != float64(intVal) {
		t.Fatalf("expected: %v, got: %v", floatVal, unwrapFloat)
	}

	unwrapInt, err = common.UnwrapProtoAnyAs[int64](wrapVal)
	if err != nil {
		t.Fatal(err)
	} else if unwrapInt != intVal {
		t.Fatalf("expected: %v, got: %v", intVal, unwrapInt)
	}

	wrapVal, err = common.WrapProtoAny(uintVal)
	if err != nil {
		t.Fatal(err)
	}

	_, err = common.UnwrapProtoAnyAs[float64](wrapVal)
	if err != nil {
		t.Fatal(err)
		// } else if uint64(unwrapFloat) != uintVal {
		// 	t.Fatalf("expected: %v, got: %v", uintVal, uint64(unwrapFloat))
	}

	unwrapUInt, err := common.UnwrapProtoAnyAs[uint64](wrapVal)
	if err != nil {
		t.Fatal(err)
	} else if unwrapUInt != uintVal {
		t.Fatalf("expected: %v, got: %v", uintVal, unwrapUInt)
	}
}
