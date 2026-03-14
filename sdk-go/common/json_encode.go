package common

import (
	"encoding/json"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
)

func MustUnmarshalJSON[T any, D ~string | ~[]byte](data D, opts ...func(*json.Decoder)) T {
	t, err := UnmarshalJSON[T](data, opts...)
	if err != nil {
		panic(err)
	}
	return t
}

func MustMarshalJSON[D ~string | ~[]byte, T any](t T, opts ...func(*json.Encoder)) D {
	data, err := MarshalJSON[D](t, opts...)
	if err != nil {
		panic(err)
	}
	return data
}

func UnmarshalJSON[T any, D ~string | ~[]byte](data D, opts ...func(*json.Decoder)) (t T, err error) {
	err = encoding.FromJSON([]byte(data), &t, encoding.WithJSONDecoderOptions(
		opts...,
	))

	return
}

func MarshalJSON[D ~string | ~[]byte, T any](t T, opts ...func(*json.Encoder)) (data D, err error) {
	b, err := encoding.ToJSON(t, encoding.WithJSONEncoderOptions(
		opts...,
	))
	if err != nil {
		return
	}

	return D(b), nil
}
