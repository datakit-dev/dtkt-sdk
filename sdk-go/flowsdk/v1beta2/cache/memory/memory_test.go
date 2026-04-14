package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache/cachetest"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache/memory"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func TestConformance(t *testing.T) {
	cachetest.Run(t, func(t *testing.T) cache.Cache {
		t.Helper()
		return memory.New()
	})
}

func node(id string) *flowv1beta2.RunSnapshot_VarNode {
	n := &flowv1beta2.RunSnapshot_VarNode{}
	n.SetId(id)
	return n
}

func TestLRUEviction(t *testing.T) {
	c := memory.New(memory.WithMaxEntries(3))
	ctx := context.Background()

	for _, id := range []string{"a", "b", "c"} {
		if err := c.Set(ctx, id, node(id)); err != nil {
			t.Fatal(err)
		}
	}

	// Adding a 4th should evict "a" (least recently used).
	if err := c.Set(ctx, "d", node("d")); err != nil {
		t.Fatal(err)
	}

	if _, ok, _ := c.Get(ctx, "a"); ok {
		t.Error("expected 'a' to be evicted")
	}
	for _, id := range []string{"b", "c", "d"} {
		if _, ok, _ := c.Get(ctx, id); !ok {
			t.Errorf("expected %q to be present", id)
		}
	}
}

func TestLRUEviction_AccessOrderMatters(t *testing.T) {
	c := memory.New(memory.WithMaxEntries(3))
	ctx := context.Background()

	for _, id := range []string{"a", "b", "c"} {
		if err := c.Set(ctx, id, node(id)); err != nil {
			t.Fatal(err)
		}
	}

	// Access "a" to make it recently used.
	c.Get(ctx, "a") //nolint:errcheck

	// Adding "d" should evict "b" (now the LRU), not "a".
	if err := c.Set(ctx, "d", node("d")); err != nil {
		t.Fatal(err)
	}

	if _, ok, _ := c.Get(ctx, "b"); ok {
		t.Error("expected 'b' to be evicted")
	}
	if _, ok, _ := c.Get(ctx, "a"); !ok {
		t.Error("expected 'a' to be present (was accessed recently)")
	}
}

func TestSetWithTTL_ExpiresEntry(t *testing.T) {
	c := memory.New()
	ctx := context.Background()

	if err := c.SetWithTTL(ctx, "k", node("v"), 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Should be present immediately.
	if _, ok, _ := c.Get(ctx, "k"); !ok {
		t.Fatal("expected hit before TTL expires")
	}

	// Wait for expiry.
	time.Sleep(60 * time.Millisecond)

	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Error("expected miss after TTL expires")
	}
}

func TestSetWithTTL_ZeroMeansNoExpiry(t *testing.T) {
	c := memory.New()
	ctx := context.Background()

	if err := c.SetWithTTL(ctx, "k", node("v"), 0); err != nil {
		t.Fatal(err)
	}

	// Should be present -- no expiry.
	if _, ok, _ := c.Get(ctx, "k"); !ok {
		t.Fatal("expected hit with zero TTL")
	}
}

func TestSetOverwriteUpdatesTTL(t *testing.T) {
	c := memory.New()
	ctx := context.Background()

	// Set with short TTL.
	if err := c.SetWithTTL(ctx, "k", node("v1"), 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Overwrite with no TTL.
	if err := c.Set(ctx, "k", node("v2")); err != nil {
		t.Fatal(err)
	}

	time.Sleep(60 * time.Millisecond)

	// Should still be present -- TTL was cleared by the overwrite.
	got, ok, _ := c.Get(ctx, "k")
	if !ok {
		t.Fatal("expected hit after TTL overwrite")
	}
	if got.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "v2" {
		t.Errorf("got %q, want %q", got.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "v2")
	}
}

func TestDefaultMaxEntries(t *testing.T) {
	c := memory.New()
	ctx := context.Background()

	// Default should handle many entries.
	for i := 0; i < 100; i++ {
		id := string(rune('a' + (i % 26)))
		if err := c.Set(ctx, id, node(id)); err != nil {
			t.Fatal(err)
		}
	}
}
