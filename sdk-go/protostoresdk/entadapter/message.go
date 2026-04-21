package entadapter

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	entfield "entgo.io/ent/schema/field"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/protostore"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

func MessageField[T any, M protostore.MessageType[T]](opts ...FieldOption) *fieldType[M] {
	field := &fieldType[M]{
		field: v1beta1.NewField(nil),
	}

	var msg M
	field.desc = entfield.
		Bytes(util.ToSnakeCase(msg.ProtoReflect().Descriptor().Name())).
		GoType(msg).
		ValueScanner(MessageScanner(field)).
		Descriptor()

	field.applyOptions(opts...)

	return field
}

func EncryptedMessageField[T any, M protostore.MessageType[T]](cipher Cipher, opts ...FieldOption) *fieldType[M] {
	field := &fieldType[M]{
		field: v1beta1.NewField(nil),
	}

	var msg M
	field.desc = entfield.
		Bytes(util.ToSnakeCase(msg.ProtoReflect().Descriptor().Name())).
		GoType(msg).
		ValueScanner(EncryptedMessageScanner(field, cipher)).
		Descriptor()

	field.applyOptions(opts...)

	return field
}

func EncryptedAnyField(cipher Cipher, opts ...FieldOption) *fieldType[*anypb.Any] {
	field := &fieldType[*anypb.Any]{
		field: v1beta1.NewField(nil),
	}

	field.desc = entfield.
		Bytes("encrypted_any").
		GoType((*anypb.Any)(nil)).
		ValueScanner(EncryptedAnyScanner(field, cipher)).
		Descriptor()

	field.applyOptions(opts...)

	return field
}

func MessageScanner[M protostore.MessageType[T], T any](field *fieldType[M]) entfield.TypeValueScanner[M] {
	return entfield.ValueScannerFunc[M, *sql.Null[[]byte]]{
		V: func(msg M) (driver.Value, error) {
			b, err := encoding.ToJSONV2(msg, encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
				Resolver: protostore.GetResolver(),
			}))
			if err != nil {
				return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
			}

			return json.RawMessage(b), nil
		},
		S: func(ns *sql.Null[[]byte]) (msg M, err error) {
			msg = new(T)
			if ns.Valid {
				err = encoding.FromJSONV2(ns.V, msg, encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: protostore.GetResolver(),
				}))
				if err != nil {
					return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
				}
			}
			return
		},
	}
}

func EncryptedMessageScanner[T any, M protostore.MessageType[T]](field *fieldType[M], cipher Cipher) entfield.TypeValueScanner[M] {
	return entfield.ValueScannerFunc[M, *sql.Null[[]byte]]{
		V: func(msg M) (driver.Value, error) {
			data, err := encoding.ToJSONV2(msg, encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
				Resolver: protostore.GetResolver(),
			}))
			if err != nil {
				return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
			}

			data, err = cipher.Encrypt(data)
			if err != nil {
				return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
			}

			return json.RawMessage(data), nil
		},
		S: func(ns *sql.Null[[]byte]) (M, error) {
			if ns.Valid {
				data, err := cipher.Decrypt(ns.V)
				if err != nil {
					return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
				}

				msg := new(T)
				err = encoding.FromJSONV2(data, msg, encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
					Resolver: protostore.GetResolver(),
				}))
				if err != nil {
					return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
				}

				return msg, nil
			}

			return nil, nil
		},
	}
}

func EncryptedAnyScanner(field *fieldType[*anypb.Any], cipher Cipher) entfield.TypeValueScanner[*anypb.Any] {
	return entfield.ValueScannerFunc[*anypb.Any, *sql.Null[[]byte]]{
		V: func(msg *anypb.Any) (driver.Value, error) {
			data, err := cipher.Encrypt(msg.Value)
			if err != nil {
				return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
			}

			data, err = encoding.ToJSONV2(&corev1.EncryptedAny{
				TypeUrl: msg.TypeUrl,
				Value:   data,
			})
			if err != nil {
				return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
			}

			return json.RawMessage(data), nil
		},
		S: func(ns *sql.Null[[]byte]) (*anypb.Any, error) {
			if ns.Valid {
				msg := new(corev1.EncryptedAny)
				err := encoding.FromJSONV2(ns.V, msg)
				if err != nil {
					return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
				}

				data, err := cipher.Decrypt(msg.Value)
				if err != nil {
					return nil, fmt.Errorf("field: %s: %w", field.desc.Name, err)
				}

				return &anypb.Any{
					TypeUrl: msg.TypeUrl,
					Value:   data,
				}, nil
			}

			return nil, nil
		},
	}
}
