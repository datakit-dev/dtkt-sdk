package cache

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"
)

// Cache stores and retrieves proto.Message values by key.
// Implementations may be in-memory (memory.Cache) or remote (e.g. Valkey).
type Cache interface {
	Get(ctx context.Context, key string) (proto.Message, bool, error)
	Set(ctx context.Context, key string, value proto.Message) error
	// SetWithTTL stores a value that expires after ttl. A zero ttl means
	// no expiration (equivalent to Set).
	SetWithTTL(ctx context.Context, key string, value proto.Message, ttl time.Duration) error
}
