package memory

import (
	"container/list"
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache"
)

// compile-time interface check
var _ cache.Cache = (*Cache)(nil)

// DefaultMaxEntries is the default maximum number of cache entries.
const DefaultMaxEntries = 10000

// entry is a cache entry stored in the LRU list.
type entry struct {
	key       string
	value     proto.Message
	expiresAt time.Time // zero means no expiration
}

// Cache is a concurrency-safe in-memory LRU cache with optional TTL support.
// When the maximum number of entries is reached, the least recently used
// entry is evicted. Expired entries are evicted lazily on access.
type Cache struct {
	mu         sync.Mutex
	maxEntries int
	items      map[string]*list.Element
	order      *list.List // front = most recently used
	now        func() time.Time
}

// Option configures a Cache instance.
type Option func(*Cache)

// WithMaxEntries sets the maximum number of entries before LRU eviction.
func WithMaxEntries(n int) Option {
	return func(c *Cache) {
		if n > 0 {
			c.maxEntries = n
		}
	}
}

func New(opts ...Option) *Cache {
	c := &Cache{
		maxEntries: DefaultMaxEntries,
		items:      make(map[string]*list.Element),
		order:      list.New(),
		now:        time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Cache) Get(_ context.Context, key string) (proto.Message, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false, nil
	}
	e := elem.Value.(*entry)
	if !e.expiresAt.IsZero() && c.now().After(e.expiresAt) {
		c.removeLocked(elem)
		return nil, false, nil
	}
	c.order.MoveToFront(elem)
	return e.value, true, nil
}

func (c *Cache) Set(ctx context.Context, key string, value proto.Message) error {
	return c.SetWithTTL(ctx, key, value, 0)
}

func (c *Cache) SetWithTTL(_ context.Context, key string, value proto.Message, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = c.now().Add(ttl)
	}

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		e := elem.Value.(*entry)
		e.value = value
		e.expiresAt = expiresAt
		return nil
	}

	// Evict LRU entries if at capacity.
	for c.order.Len() >= c.maxEntries {
		c.removeLocked(c.order.Back())
	}

	e := &entry{key: key, value: value, expiresAt: expiresAt}
	elem := c.order.PushFront(e)
	c.items[key] = elem
	return nil
}

func (c *Cache) removeLocked(elem *list.Element) {
	c.order.Remove(elem)
	e := elem.Value.(*entry)
	delete(c.items, e.key)
}
