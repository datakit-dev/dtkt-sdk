package entadapter

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"path"

	"entgo.io/ent"
	entfield "entgo.io/ent/schema/field"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostore"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

func MessageField[T any, M protostore.MessageType[T]](opts ...FieldOption) *Field[M] {
	var (
		msg   M
		field = entfield.Bytes(util.ToSnakeCase(msg.ProtoReflect().Descriptor().Name())).GoType(msg)
	)

	return NewFieldOf[M](field.ValueScanner(MessageScanner[M](field)), opts...)
}

func EncryptedMessageField[T any, M protostore.MessageType[T]](cipher Cipher, opts ...FieldOption) *Field[M] {
	var (
		msg   M
		field = entfield.Bytes(util.ToSnakeCase(msg.ProtoReflect().Descriptor().Name())).GoType(msg)
	)

	return NewFieldOf[M](field.ValueScanner(EncryptedMessageScanner[M](field, cipher)), opts...)
}

func EncryptedAnyField(cipher Cipher, opts ...FieldOption) *Field[*anypb.Any] {
	field := entfield.
		Bytes("encrypted_any").
		GoType((*anypb.Any)(nil))

	return NewFieldOf[*anypb.Any](field.ValueScanner(EncryptedAnyScanner(field, cipher)), opts...)
}

func MessageScanner[M protostore.MessageType[T], T any](field ent.Field) entfield.TypeValueScanner[M] {
	return entfield.ValueScannerFunc[M, *sql.Null[[]byte]]{
		V: func(msg M) (driver.Value, error) {
			data, err := encoding.ToJSON(msg,
				encoding.WithProtoEncodeOptions(protojson.MarshalOptions{
					Resolver: protostore.Resolver,
				}),
				encoding.WithEncodeDurationString(),
			)
			if err != nil {
				return nil, fmt.Errorf("message field: %s (%s): %w", field.Descriptor().Name, msg.ProtoReflect().Descriptor().FullName(), err)
			}

			value := json.RawMessage(data)
			err = value.Canonicalize()

			return value, err
		},
		S: func(ns *sql.Null[[]byte]) (msg M, err error) {
			msg = new(T)
			if ns.Valid {
				err = encoding.FromJSON(ns.V, msg,
					encoding.WithProtoDecodeOptions(protojson.UnmarshalOptions{
						Resolver: protostore.Resolver,
					}),
					encoding.WithDecodeDurationString(),
				)
				if err != nil {
					return nil, fmt.Errorf("message field: %s (%s): %w", field.Descriptor().Name, msg.ProtoReflect().Descriptor().FullName(), err)
				}
			}
			return
		},
	}
}

func EncryptedMessageScanner[M protostore.MessageType[T], T any](field ent.Field, cipher Cipher) entfield.TypeValueScanner[M] {
	return entfield.ValueScannerFunc[M, *sql.Null[[]byte]]{
		V: func(msg M) (driver.Value, error) {
			data, err := encoding.ToJSON(msg, encoding.WithProtoEncodeOptions(protojson.MarshalOptions{
				Resolver: protostore.Resolver,
			}))
			if err != nil {
				return nil, fmt.Errorf("encyrpt message field: %s (%s): %w", field.Descriptor().Name, msg.ProtoReflect().Descriptor().FullName(), err)
			}

			data, err = cipher.Encrypt(data)
			if err != nil {
				return nil, fmt.Errorf("encyrpt message field: %s (%s): %w", field.Descriptor().Name, msg.ProtoReflect().Descriptor().FullName(), err)
			}

			return data, nil
		},
		S: func(ns *sql.Null[[]byte]) (msg M, err error) {
			if ns.Valid {
				data, err := cipher.Decrypt(ns.V)
				if err != nil {
					return nil, fmt.Errorf("decrypt message field: %s (%s): %w", field.Descriptor().Name, msg.ProtoReflect().Descriptor().FullName(), err)
				}

				msg = new(T)
				err = encoding.FromJSON(data, msg, encoding.WithProtoDecodeOptions(protojson.UnmarshalOptions{
					Resolver: protostore.Resolver,
				}))
				if err != nil {
					return nil, fmt.Errorf("decrypt message field: %s (%s): %w", field.Descriptor().Name, msg.ProtoReflect().Descriptor().FullName(), err)
				}

				return msg, nil
			}

			return nil, nil
		},
	}
}

func EncryptedAnyScanner(field ent.Field, cipher Cipher) entfield.TypeValueScanner[*anypb.Any] {
	return entfield.ValueScannerFunc[*anypb.Any, *sql.Null[[]byte]]{
		V: func(msg *anypb.Any) (driver.Value, error) {
			data, err := cipher.Encrypt(msg.Value)
			if err != nil {
				return nil, fmt.Errorf("encypted any field: %s (%s): %w", field.Descriptor().Name, path.Base(msg.TypeUrl), err)
			}

			data, err = encoding.ToJSON(&corev1.EncryptedAny{
				TypeUrl: msg.TypeUrl,
				Value:   data,
			})
			if err != nil {
				return nil, fmt.Errorf("encypted any field: %s (%s): %w", field.Descriptor().Name, path.Base(msg.TypeUrl), err)
			}

			return json.RawMessage(data), nil
		},
		S: func(ns *sql.Null[[]byte]) (*anypb.Any, error) {
			if ns.Valid {
				msg := new(corev1.EncryptedAny)
				err := encoding.FromJSON(ns.V, msg)
				if err != nil {
					return nil, fmt.Errorf("decrypt any field: %s: %w", field.Descriptor().Name, err)
				}

				data, err := cipher.Decrypt(msg.Value)
				if err != nil {
					return nil, fmt.Errorf("decrypt any field: %s (%s): %w", field.Descriptor().Name, path.Base(msg.TypeUrl), err)
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
