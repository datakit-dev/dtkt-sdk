package util

import (
	"testing"

	"github.com/google/uuid"
)

func TestParsePageTokenUUIDV7(t *testing.T) {
	uid, _ := uuid.NewV7()
	tests := []struct {
		name      string
		uid       string
		expectErr bool
	}{
		{
			name: "valid UUIDv7",
			uid:  uid.String(),
		},
		{
			name:      "invalid base64",
			uid:       "invalid-base64",
			expectErr: true,
		},
		{
			name:      "invalid UUID version",
			uid:       "017f4a80-1b2c-4e8f-9a0b-1c2d3e4f5a6b", // UUIDv4
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uid, err := uuid.Parse(tt.uid)
			if err != nil && !tt.expectErr {
				t.Fatalf("failed to parse test %s: %v", tt.name, err)
			}

			pageToken := NextPageTokenUUIDV7(uid)
			parsedUID, err := ParsePageTokenUUIDV7(pageToken)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if parsedUID != uid {
				t.Errorf("expected UID %v, got %v", uid, parsedUID)
			}

			pageToken2 := NextPageTokenUUIDV7(parsedUID)
			if pageToken != pageToken2 {
				t.Errorf("expected page token %s, got %s", pageToken, pageToken2)
			}
		})
	}
}

type testPageTokenRequest struct {
	pageSize  int32
	pageToken string
}

func (r testPageTokenRequest) GetPageSize() int32 {
	return r.pageSize
}

func (r testPageTokenRequest) GetPageToken() string {
	return r.pageToken
}

func TestParsePageTokenRequestUUIDV7(t *testing.T) {
	uid, _ := uuid.NewV7()
	pageToken := NextPageTokenUUIDV7(uid)

	tests := []struct {
		name        string
		req         testPageTokenRequest
		defaultSize int32
		minSize     int32
		maxSize     int32
		expectUID   uuid.UUID
		expectSize  int32
		expectErr   bool
	}{
		{
			name: "valid request with page size",
			req: testPageTokenRequest{
				pageSize:  50,
				pageToken: pageToken,
			},
			defaultSize: 20,
			minSize:     10,
			maxSize:     100,
			expectUID:   uid,
			expectSize:  50,
		},
		{
			name: "valid request with default page size",
			req: testPageTokenRequest{
				pageSize:  0,
				pageToken: pageToken,
			},
			defaultSize: 30,
			minSize:     10,
			maxSize:     100,
			expectUID:   uid,
			expectSize:  30,
		},
		{
			name: "invalid page token",
			req: testPageTokenRequest{
				pageSize:  20,
				pageToken: "invalid-token",
			},
			defaultSize: 20,
			minSize:     10,
			maxSize:     100,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedUID, size, err := ParsePageTokenRequestUUIDV7(tt.req, tt.defaultSize, tt.minSize, tt.maxSize)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if parsedUID != tt.expectUID {
				t.Errorf("expected UID %v, got %v", tt.expectUID, parsedUID)
			}

			if size != tt.expectSize {
				t.Errorf("expected size %d, got %d", tt.expectSize, size)
			}
		})
	}
}
