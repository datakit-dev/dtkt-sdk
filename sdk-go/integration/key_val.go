package integration

import (
	"bytes"
	"context"
	"regexp"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integration/v1beta1"
	"github.com/nats-io/nats.go/jetstream"
)

// Lowercase letters (a-z), uppercase letters (A-Z), digits (0-9), underscores (_), dashes (-), dots (.), forward slashes (/), and equal signs (=).
var keyvalKeyRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-./=]`)

type jetStreamKeyVal struct {
	getKeyVal func() (jetstream.KeyValue, error)
}

func newJetStreamKeyVal[C any, I v1beta1.InstanceType](ctx context.Context, intgr *Integration[C, I], conn, hash string) *jetStreamKeyVal {
	return &jetStreamKeyVal{
		getKeyVal: sync.OnceValues(func() (jetstream.KeyValue, error) {
			broker, err := intgr.getJetStream()
			if err != nil {
				return nil, err
			}

			return broker.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
				Bucket: hash,
				Metadata: map[string]string{
					"deployment": intgr.name.String(),
					"connection": conn,
				},
			})
		}),
	}
}

func (s *jetStreamKeyVal) Get(ctx context.Context, key string) ([]byte, error) {
	kv, err := s.getKeyVal()
	if err != nil {
		return nil, err
	}

	key = sanitizeJetStreamKey(key)
	entry, err := kv.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return entry.Value(), nil
}

func (s *jetStreamKeyVal) Set(ctx context.Context, key string, data []byte) error {
	kv, err := s.getKeyVal()
	if err != nil {
		return err
	}

	key = sanitizeJetStreamKey(key)
	entry, err := kv.Get(ctx, key)
	if err == nil && bytes.Equal(data, entry.Value()) {
		return nil
	}

	_, err = kv.Put(ctx, key, data)
	return err
}

func (s *jetStreamKeyVal) Delete(ctx context.Context, key string) error {
	kv, err := s.getKeyVal()
	if err != nil {
		return err
	}

	return kv.Delete(ctx, sanitizeJetStreamKey(key))
}

func sanitizeJetStreamKey(key string) string {
	return keyvalKeyRegex.ReplaceAllString(key, "-")
}
