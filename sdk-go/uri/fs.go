package uri

import (
	"context"
	"fmt"
	stdfs "io/fs"
	"net/url"
	"os"
)

// NewURIFilesystem creates a filesystem (fs.FS) from a URI.
// Supported schemes:
//   - file:// - Returns os.DirFS for local filesystem access
//   - sftp|ssh:// - Returns SFTPFileSystem for remote SSH filesystem access
//   - git:// or git+https:// or git+ssh:// or git+file:// - Returns GitFileSystem for Git repository access
//
// For sftp|ssh:// URIs, SSH authentication and connection details can be specified
// via query parameters (identity, ssh_config, known_hosts, host_key_verify).
//
// For git:// URIs, you can specify a ref (branch, tag, or commit) via query parameters.
//
// Example usage:
//
//	// Local filesystem
//	uri, _ := url.Parse("file:///path/to/files")
//	fs, _ := NewURIFilesystem(ctx, uri)
//
//	// Remote SFTP filesystem
//	uri, _ := url.Parse("sftp|ssh://user@host/path/to/files?identity=~/.ssh/id_ed25519")
//	fs, _ := NewURIFilesystem(ctx, uri)
//
//	// Git repository filesystem
//	uri, _ := url.Parse("git+https://github.com/user/repo?ref=main")
//	fs, _ := NewURIFilesystem(ctx, uri)
func NewURIFilesystem(ctx context.Context, uri *url.URL) (stdfs.FS, error) {
	if uri == nil {
		return nil, fmt.Errorf("URI cannot be nil")
	}

	switch uri.Scheme {
	case "file":
		// For file:// URIs, return os.DirFS rooted at the path
		if uri.Path == "" {
			return nil, fmt.Errorf("file:// URI requires a path")
		}
		return os.DirFS(uri.Path), nil
	case "sftp", "ssh":
		// For sftp|ssh:// URIs, create an SFTP filesystem
		return NewSFTPFileSystem(ctx, uri)
	case "git", "git+https", "git+http", "git+ssh", "git+file":
		// For git:// URIs, create a Git filesystem
		return NewGitFileSystem(ctx, uri)

	default:
		return nil, fmt.Errorf("unsupported filesystem scheme: %s (supported: file, sftp, git)", uri.Scheme)
	}
}
