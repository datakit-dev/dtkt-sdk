package uri

import (
	"context"
	"fmt"
	stdfs "io/fs"
	"net/url"

	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/gitfs"
)

// GitFileSystem provides filesystem access to Git repositories.
// It wraps the go-fsimpl gitfs implementation to provide read-only access
// to files in Git repositories (local or remote).
type GitFileSystem struct {
	fs  stdfs.FS
	uri *url.URL
}

// NewGitFileSystem creates a new Git filesystem from a git:// URI.
// The URI should be in one of these formats:
//   - git+file:///path/to/repo - Local Git repository
//   - git+https://github.com/user/repo - Remote HTTPS Git repository
//   - git+ssh://git@github.com/user/repo - Remote SSH Git repository
//   - git://github.com/user/repo - Remote Git repository (defaults to https)
//
// Optional query parameters:
//   - ref=branch-name|tag|commit - Specify a Git reference (default: HEAD)
//   - depth=N - Shallow clone depth for remote repos
//
// The path component can specify a subdirectory within the repository:
//   - git+https://github.com/user/repo//subdir - Access only the subdir
//
// Examples:
//   - git+file:///home/user/project - Local repository
//   - git+https://github.com/user/repo - Remote repository on main/master
//   - git+https://github.com/user/repo?ref=v1.2.3 - Specific tag
//   - git+https://github.com/user/repo//docs - Only the docs subdirectory
func NewGitFileSystem(ctx context.Context, uri *url.URL) (*GitFileSystem, error) {
	if uri == nil {
		return nil, fmt.Errorf("URI cannot be nil")
	}

	// Prepare the URL for gitfs
	gitURI := uri

	// If the scheme is just "git", convert it to git+https
	if uri.Scheme == "git" {
		// Rebuild with git+https scheme
		modifiedURI := *uri
		modifiedURI.Scheme = "git+https"
		gitURI = &modifiedURI
	}

	// Create the gitfs filesystem
	fs, err := gitfs.New(gitURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create git filesystem for %s: %w", gitURI.String(), err)
	}

	return &GitFileSystem{
		fs:  fsimpl.WithContextFS(ctx, fs),
		uri: uri,
	}, nil
}

// Open opens a file for reading from the Git repository.
func (gfs *GitFileSystem) Open(name string) (stdfs.File, error) {
	return gfs.fs.Open(name)
}

// ReadFile reads the contents of a file from the Git repository.
func (gfs *GitFileSystem) ReadFile(name string) ([]byte, error) {
	if fsWithReadFile, ok := gfs.fs.(interface {
		ReadFile(string) ([]byte, error)
	}); ok {
		return fsWithReadFile.ReadFile(name)
	}
	// Fallback to standard fs.ReadFile
	return stdfs.ReadFile(gfs.fs, name)
}

// ReadDir reads a directory from the Git repository.
func (gfs *GitFileSystem) ReadDir(name string) ([]stdfs.DirEntry, error) {
	if fsWithReadDir, ok := gfs.fs.(interface {
		ReadDir(string) ([]stdfs.DirEntry, error)
	}); ok {
		return fsWithReadDir.ReadDir(name)
	}
	// Fallback to standard fs.ReadDir
	return stdfs.ReadDir(gfs.fs, name)
}

// Stat returns file info for a file in the Git repository.
func (gfs *GitFileSystem) Stat(name string) (stdfs.FileInfo, error) {
	if fsWithStat, ok := gfs.fs.(interface {
		Stat(string) (stdfs.FileInfo, error)
	}); ok {
		return fsWithStat.Stat(name)
	}

	// Fallback to using Open
	file, err := gfs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer file.Close()

	return file.Stat()
}

// URI returns the original URI.
func (gfs *GitFileSystem) URI() *url.URL {
	return gfs.uri
}

// FS returns the underlying fs.FS implementation.
func (gfs *GitFileSystem) FS() stdfs.FS {
	return gfs.fs
}
