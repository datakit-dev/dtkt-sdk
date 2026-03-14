package util

func ToPointer[T any](val T) *T {
	return &val
}

func FromPointer[T any](ptr *T) (val T) {
	if ptr != nil {
		val = *ptr
	}
	return
}

func IsNil[T any](v T) bool {
	return isNil(v)
}

func isNil(v any) bool {
	return v == nil
}

func NilValueOrDefault[T any](v *T, d T) T {
	if v != nil {
		return *v
	}
	return d
}
