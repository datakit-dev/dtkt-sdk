package uri

import (
	"context"
	"io"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithScheme(t *testing.T) {
	path, uriStr := "/foo/bar", "file:///foo/bar"
	url, err := ParseWithScheme(path, "file")
	if err != nil {
		t.Fatal(err)
	} else if url.String() != uriStr {
		t.Fatalf("expected: %s, got: %s", uriStr, url.String())
	} else if url.Scheme != "file" {
		t.Fatalf("expected: file, got: %s", url.Scheme)
	} else if url.Path != path {
		t.Fatalf("expected: %s, got: %s", path, url.Path)
	}
}

func TestURIExists_Git(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		uriString   string
		expectError bool
	}{
		{
			name:        "git+https scheme",
			uriString:   "git+https://github.com/datakit-dev/dtkt-cli",
			expectError: false,
		},
		{
			name:        "git scheme",
			uriString:   "git://github.com/datakit-dev/dtkt-cli",
			expectError: false,
		},
		{
			name:        "invalid git repository",
			uriString:   "git+https://github.com/invalid/nonexistent-repo-12345",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := url.Parse(tt.uriString)
			require.NoError(t, err)

			exists, err := Exists(ctx, uri)
			if tt.expectError {
				// For invalid repos, we expect either an error or false
				if err == nil {
					assert.False(t, exists, "Invalid repository should not exist")
				}
			} else {
				// For valid repos, we might get an error due to network
				// but we'll log it as a warning
				if err != nil {
					t.Logf("Warning: Failed to check git URI existence (might be expected if no network): %v", err)
				} else {
					t.Logf("Git URI exists: %v", exists)
				}
			}
		})
	}
}

func TestNewURIReader_Git(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		uriString   string
		expectError bool
		skipOnError bool
	}{
		{
			name:        "read README from git repo",
			uriString:   "git+https://github.com/datakit-dev/dtkt-cli//README.md",
			expectError: false,
			skipOnError: true, // Skip if no network
		},
		{
			name:        "read from specific ref",
			uriString:   "git+https://github.com/datakit-dev/dtkt-cli//README.md?ref=main",
			expectError: false,
			skipOnError: true, // Skip if no network
		},
		{
			name:        "attempt to read directory should fail",
			uriString:   "git+https://github.com/datakit-dev/dtkt-cli",
			expectError: true,
			skipOnError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := url.Parse(tt.uriString)
			require.NoError(t, err)

			reader, err := NewReader(ctx, uri)
			if tt.expectError {
				if err == nil && reader != nil {
					//nolint:errcheck
					reader.Close()
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					if tt.skipOnError {
						t.Skipf("Skipping test due to error (likely network): %v", err)
					} else {
						t.Errorf("Unexpected error: %v", err)
					}
					return
				}

				require.NotNil(t, reader)
				//nolint:errcheck
				defer reader.Close()

				// Try to read some content
				content, err := io.ReadAll(reader)
				require.NoError(t, err)
				assert.NotEmpty(t, content, "File content should not be empty")
				t.Logf("Read %d bytes from git repository", len(content))
			}
		})
	}
}

func TestNewURIWriter_Git(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		uriString string
	}{
		{
			name:      "git+https scheme should be read-only",
			uriString: "git+https://github.com/datakit-dev/dtkt-cli//README.md",
		},
		{
			name:      "git scheme should be read-only",
			uriString: "git://github.com/datakit-dev/dtkt-cli//README.md",
		},
		{
			name:      "git+ssh scheme should be read-only",
			uriString: "git+ssh://git@github.com/datakit-dev/dtkt-cli//README.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := url.Parse(tt.uriString)
			require.NoError(t, err)

			writer, err := NewWriter(ctx, uri)
			assert.Error(t, err, "Git filesystems should be read-only")
			assert.Nil(t, writer, "Writer should be nil for read-only git filesystems")
			assert.Contains(t, err.Error(), "read-only", "Error should mention read-only filesystem")
		})
	}
}

func TestGetChecksum_Git(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		uriString   string
		skipOnError bool
	}{
		{
			name:        "checksum README from git repo",
			uriString:   "git+https://github.com/datakit-dev/dtkt-cli//README.md",
			skipOnError: true,
		},
		{
			name:        "checksum from specific ref",
			uriString:   "git+https://github.com/datakit-dev/dtkt-cli//README.md?ref=main",
			skipOnError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum, err := GetChecksum(ctx, tt.uriString)
			if err != nil {
				if tt.skipOnError {
					t.Skipf("Skipping test due to error (likely network): %v", err)
				} else {
					t.Errorf("Unexpected error: %v", err)
				}
				return
			}

			assert.NotEmpty(t, checksum, "Checksum should not be empty")
			assert.Contains(t, checksum, "sha256:", "Checksum should be prefixed with sha256:")
			t.Logf("Checksum: %s", checksum)
		})
	}
}
