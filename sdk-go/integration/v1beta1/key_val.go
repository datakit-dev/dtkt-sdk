package v1beta1

import (
	"bytes"
	context "context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type (
	KeyVal struct {
		store KeyValStore
	}
	KeyValStore interface {
		Get(context.Context, string) ([]byte, error)
		Set(context.Context, string, []byte) error
		Delete(context.Context, string) error
	}
)

func NewKeyVal(store KeyValStore) *KeyVal {
	return &KeyVal{
		store: store,
	}
}

func (s *KeyVal) SetValue[T any](ctx context.Context, key string, val T) (err error) {
	wrapper, err := common.WrapProtoAny(val)
	if err != nil {
		return
	}

	data, err := proto.Marshal(wrapper)
	if err != nil {
		return
	}

	curr, err := s.store.Get(ctx, key)
	if err == nil && bytes.Equal(curr, data) {
		return
	}

	return s.store.Set(ctx, key, data)
}

func (s *KeyVal) GetValue[T any](ctx context.Context, key string) (val T, err error) {
	data, err := s.store.Get(ctx, key)
	if err != nil {
		return
	}

	wrapper := new(anypb.Any)
	err = proto.Unmarshal(data, wrapper)
	if err != nil {
		return
	}

	return common.UnwrapProtoAnyAs[T](wrapper)
}

func (s *KeyVal) DeleteValue(ctx context.Context, key string) error {
	return s.store.Delete(ctx, key)
}
