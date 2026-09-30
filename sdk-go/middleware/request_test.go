package middleware

import (
	"testing"
)

type fakeConnection struct {
	name       string
	configHash string
	configGen  uint64
}

func (c fakeConnection) GetName() string       { return c.name }
func (c fakeConnection) GetConfigHash() string { return c.configHash }
func (c fakeConnection) GetConfigGen() uint64  { return c.configGen }

func TestNewRequest(t *testing.T) {
	r := NewRequest("res", "hash", 3)
	if r.ResourceName() != "res" {
		t.Fatalf("ResourceName() = %q, want %q", r.ResourceName(), "res")
	}
	if r.ConfigHash() != "hash" {
		t.Fatalf("ConfigHash() = %q, want %q", r.ConfigHash(), "hash")
	}
	if r.ConfigGen() != 3 {
		t.Fatalf("ConfigGen() = %d, want %d", r.ConfigGen(), 3)
	}
}

func TestRequest_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		req     *Request
		wantErr bool
	}{
		{"nil request", nil, true},
		{"missing resource name", NewRequest("", "hash", 1), true},
		{"missing config hash", NewRequest("res", "", 1), true},
		{"valid", NewRequest("res", "hash", 1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.IsValid()
			if (err != nil) != tt.wantErr {
				t.Fatalf("IsValid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequest_StringAndHashes(t *testing.T) {
	r1 := NewRequest("res", "hash", 5)
	r2 := NewRequest("res", "hash", 5)
	if r1.String() != r2.String() {
		t.Fatalf("String() not deterministic: %q != %q", r1.String(), r2.String())
	}
	if r1.HashSHA256() != r2.HashSHA256() {
		t.Fatalf("HashSHA256() not deterministic: %q != %q", r1.HashSHA256(), r2.HashSHA256())
	}

	other := NewRequest("res", "hash", 6)
	if r1.String() == other.String() {
		t.Fatalf("expected different configGen to change String(), both got %q", r1.String())
	}
	if r1.HashSHA256() == other.HashSHA256() {
		t.Fatalf("expected different configGen to change HashSHA256(), both got %q", r1.HashSHA256())
	}
}

func TestRequest_PrevSHA256(t *testing.T) {
	r := NewRequest("res", "hash", 5)
	prev := NewRequest("res", "hash", 4)
	if r.PrevSHA256() != prev.HashSHA256() {
		t.Fatalf("PrevSHA256() = %q, want %q", r.PrevSHA256(), prev.HashSHA256())
	}

	// configGen 0 has no predecessor, so it should floor at 0 rather than
	// wrapping/underflowing.
	zero := NewRequest("res", "hash", 0)
	if zero.PrevSHA256() != zero.HashSHA256() {
		t.Fatalf("PrevSHA256() at configGen=0 = %q, want it to floor at 0 (%q)", zero.PrevSHA256(), zero.HashSHA256())
	}
}
