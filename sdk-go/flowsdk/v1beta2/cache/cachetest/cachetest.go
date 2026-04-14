package cachetest

import (
	"context"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Factory creates a fresh Cache instance for each test.
type Factory func(t *testing.T) cache.Cache

// Run executes the full cache conformance test suite against the given factory.
func Run(t *testing.T, factory Factory) {
	t.Helper()

	t.Run("GetSet", func(t *testing.T) {
		c := factory(t)
		ctx := context.Background()

		node := &flowv1beta2.RunSnapshot_VarNode{}
		node.SetId("n1")
		if err := c.Set(ctx, "key1", node); err != nil {
			t.Fatal(err)
		}

		got, ok, err := c.Get(ctx, "key1")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected cache hit")
		}

		gotNode, ok := got.(*flowv1beta2.RunSnapshot_VarNode)
		if !ok {
			t.Fatalf("got type %T, want *RunSnapshot_VarNode", got)
		}
		if gotNode.GetId() != "n1" {
			t.Errorf("got ID %q, want %q", gotNode.GetId(), "n1")
		}
	})

	t.Run("GetMiss", func(t *testing.T) {
		c := factory(t)
		ctx := context.Background()

		_, ok, err := c.Get(ctx, "nonexistent-key")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected cache miss")
		}
	})

	t.Run("SetOverwrite", func(t *testing.T) {
		c := factory(t)
		ctx := context.Background()

		node1 := &flowv1beta2.RunSnapshot_VarNode{}
		node1.SetId("v1")
		if err := c.Set(ctx, "key1", node1); err != nil {
			t.Fatal(err)
		}

		node2 := &flowv1beta2.RunSnapshot_VarNode{}
		node2.SetId("v2")
		if err := c.Set(ctx, "key1", node2); err != nil {
			t.Fatal(err)
		}

		got, ok, err := c.Get(ctx, "key1")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected cache hit")
		}
		if got.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "v2" {
			t.Errorf("got %q, want %q", got.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "v2")
		}
	})

	t.Run("IsolatedKeys", func(t *testing.T) {
		c := factory(t)
		ctx := context.Background()

		nodeA := &flowv1beta2.RunSnapshot_VarNode{}
		nodeA.SetId("a")
		nodeB := &flowv1beta2.RunSnapshot_VarNode{}
		nodeB.SetId("b")
		if err := c.Set(ctx, "keyA", nodeA); err != nil {
			t.Fatal(err)
		}
		if err := c.Set(ctx, "keyB", nodeB); err != nil {
			t.Fatal(err)
		}

		gotA, ok, err := c.Get(ctx, "keyA")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected hit for keyA")
		}
		if gotA.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "a" {
			t.Errorf("keyA: got %q, want %q", gotA.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "a")
		}

		gotB, ok, err := c.Get(ctx, "keyB")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected hit for keyB")
		}
		if gotB.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "b" {
			t.Errorf("keyB: got %q, want %q", gotB.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "b")
		}
	})

	t.Run("MultipleTypes", func(t *testing.T) {
		c := factory(t)
		ctx := context.Background()

		varNode := &flowv1beta2.RunSnapshot_VarNode{}
		varNode.SetId("var1")
		if err := c.Set(ctx, "var-key", varNode); err != nil {
			t.Fatal(err)
		}

		inputNode := &flowv1beta2.RunSnapshot_InputNode{}
		inputNode.SetId("input1")
		if err := c.Set(ctx, "input-key", inputNode); err != nil {
			t.Fatal(err)
		}

		gotVar, ok, err := c.Get(ctx, "var-key")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected hit for var-key")
		}
		if gotVar.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "var1" {
			t.Errorf("var: got %q, want %q", gotVar.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "var1")
		}

		gotInput, ok, err := c.Get(ctx, "input-key")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected hit for input-key")
		}
		if gotInput.(*flowv1beta2.RunSnapshot_InputNode).GetId() != "input1" {
			t.Errorf("input: got %q, want %q", gotInput.(*flowv1beta2.RunSnapshot_InputNode).GetId(), "input1")
		}
	})
}
