package util

import (
	"strconv"
)

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

func IntMap[V any, T Number](num T, f func(int) V) (v []V) {
	if num <= 0 {
		return
	}

	v = make([]V, int(num))
	for idx := range int(num) {
		v[idx] = f(idx)
	}

	return
}

func ParseInt32(from string) (to int32, err error) {
	i, err := strconv.ParseInt(from, 10, 32)
	if err != nil {
		return
	}
	return int32(i), nil
}

func FormatInt32(i int32) string {
	return strconv.FormatInt(int64(i), 10)
}

func ParseInt64(from string) (to int64, err error) {
	i, err := strconv.ParseInt(from, 10, 64)
	if err != nil {
		return
	}
	return int64(i), nil
}

func FormatInt64(i int64) string {
	return strconv.FormatInt(i, 10)
}

func ParseUInt32(from string) (to uint32, err error) {
	i, err := strconv.ParseUint(from, 10, 32)
	if err != nil {
		return
	}
	return uint32(i), nil
}

func FormatUInt32(i uint32) string {
	return strconv.FormatUint(uint64(i), 10)
}

func ParseUInt64(from string) (to uint64, err error) {
	i, err := strconv.ParseUint(from, 10, 64)
	if err != nil {
		return
	}
	return uint64(i), nil
}

func FormatUInt64(i uint64) string {
	return strconv.FormatUint(i, 10)
}

func ParseFloat32(from string) (to float32, err error) {
	f, err := strconv.ParseFloat(from, 32)
	if err != nil {
		return
	}
	return float32(f), nil
}

func ParseFloat64(from string) (to float64, err error) {
	f, err := strconv.ParseFloat(from, 64)
	if err != nil {
		return
	}
	return float64(f), nil
}
