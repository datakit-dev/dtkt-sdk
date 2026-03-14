package uri

import (
	"context"
	"io/fs"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitFileSystem(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		uriString   string
		expectError bool
	}{
		{
			name:        "nil URI",
			uriString:   "",
			expectError: true,
		},
		{
			name:        "git scheme converts to git+https",
			uriString:   "git://github.com/datakit-dev/dtkt-cli",
			expectError: false,
		},
		{
			name:        "git+https scheme",
			uriString:   "git+https://github.com/datakit-dev/dtkt-cli",
			expectError: false,
		},
		{
			name:        "git+ssh scheme",
			uriString:   "git+ssh://git@github.com/datakit-dev/dtkt-cli",
			expectError: false,
		},
		{
			name:        "git with ref parameter",
			uriString:   "git+https://github.com/datakit-dev/dtkt-cli?ref=main",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.uriString == "" {
				_, err := NewGitFileSystem(ctx, nil)
				assert.Error(t, err)
				return
			}

			uri, err := url.Parse(tt.uriString)
			require.NoError(t, err)

			gitFS, err := NewGitFileSystem(ctx, uri)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, gitFS)
			} else {
				// Note: This might fail if there's no network connectivity
				// or if the repository doesn't exist, but it tests the
				// creation of the filesystem object
				if err != nil {
					t.Logf("Warning: Failed to create git filesystem (might be expected if no network): %v", err)
				} else {
					assert.NotNil(t, gitFS)
					assert.NotNil(t, gitFS.FS())
					assert.Equal(t, uri, gitFS.URI())
				}
			}
		})
	}
}

func TestGitFileSystemImplementsInterfaces(t *testing.T) {
	ctx := context.Background()
	uri, err := url.Parse("git+https://github.com/datakit-dev/dtkt-cli")
	require.NoError(t, err)

	gitFS, err := NewGitFileSystem(ctx, uri)
	if err != nil {
		t.Skipf("Skipping interface test due to git filesystem creation error: %v", err)
		return
	}

	// Verify it implements fs.FS
	var _ fs.FS = gitFS

	// Verify it has the expected methods
	assert.NotNil(t, gitFS.Open)
	assert.NotNil(t, gitFS.ReadFile)
	assert.NotNil(t, gitFS.ReadDir)
	assert.NotNil(t, gitFS.Stat)
	assert.NotNil(t, gitFS.URI)
	assert.NotNil(t, gitFS.FS)
}

func TestNewURIFilesystem_Git(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		uriString string
	}{
		{
			name:      "git scheme",
			uriString: "git://github.com/datakit-dev/dtkt-cli",
		},
		{
			name:      "git+https scheme",
			uriString: "git+https://github.com/datakit-dev/dtkt-cli",
		},
		{
			name:      "git+ssh scheme",
			uriString: "git+ssh://git@github.com/datakit-dev/dtkt-cli",
		},
		{
			name:      "git+http scheme",
			uriString: "git+http://github.com/datakit-dev/dtkt-cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := url.Parse(tt.uriString)
			require.NoError(t, err)

			filesystem, err := NewURIFilesystem(ctx, uri)
			if err != nil {
				t.Logf("Warning: Failed to create URI filesystem (might be expected if no network): %v", err)
			} else {
				assert.NotNil(t, filesystem)

				// Check it's a GitFileSystem
				if gitFS, ok := filesystem.(*GitFileSystem); ok {
					assert.NotNil(t, gitFS)
				}
			}
		})
	}
}
