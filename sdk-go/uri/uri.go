package uri

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdfs "io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

var (
	validSchemePattern = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+.-]+)\:\/\/`)
	validUriPattern    = regexp.MustCompile(`^(?:(?:[a-zA-Z][a-zA-Z\d+\-.]*):)?(?://(?:[A-Za-z0-9\-\.]+(?::\d+)?))?(/[^\?#]*)?(?:\?([^\#]*))?(?:\#(.*))?$`)
)

func IsValid(uri string) bool {
	return validUriPattern.MatchString(uri)
}

func HasScheme(uri string) bool {
	return validSchemePattern.MatchString(uri)
}

func GetScheme(uri string) (string, bool) {
	if HasScheme(uri) {
		return validSchemePattern.FindStringSubmatch(uri)[1], true
	}
	return "", false
}

func TrimScheme(uri string) string {
	if scheme, ok := GetScheme(uri); ok {
		return strings.TrimPrefix(uri, scheme+"://")
	}
	return uri
}

func Parse(uriStr string) (*url.URL, error) {
	scheme, hasScheme := GetScheme(uriStr)
	if !hasScheme {
		scheme = "file"

		if strings.HasPrefix(uriStr, ".") || !strings.HasPrefix(uriStr, filepath.FromSlash("/")) {
			cwd, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("working directory (%s) error: %w", uriStr, err)
			}

			uriStr = filepath.Join(cwd, filepath.ToSlash(uriStr))
		}
	}

	uri, err := ParseWithScheme(uriStr, scheme)
	if err != nil {
		return nil, fmt.Errorf("parse uri (%s) error: %w", uriStr, err)
	}

	switch uri.Scheme {
	case "file":
		path, err := filepath.Abs(filepath.FromSlash(uri.Path))
		if err != nil {
			return nil, fmt.Errorf("absolute file path (%s) error: %w", uri.Path, err)
		}
		uri.Path = filepath.ToSlash(path)
	}

	return uri, nil
}

func ParseWithScheme(uri string, scheme string) (*url.URL, error) {
	scheme = strings.ToLower(scheme)

	if !HasScheme(uri) {
		uri = scheme + `://` + uri
	}

	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	url.Scheme = strings.ToLower(url.Scheme)

	if url.Scheme != scheme {
		return nil, fmt.Errorf("uri scheme invalid: %q, expected: %q", url.Scheme, scheme)
	}

	return url, nil
}

func Exists(ctx context.Context, uri *url.URL) (bool, error) {
	if uri == nil {
		return false, nil
	}

	switch uri.Scheme {
	case "file":
		path := filepath.FromSlash(uri.Path)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return false, nil
		} else {
			return true, nil
		}
	case "sftp", "ssh":
		// Check if SFTP file/directory exists by attempting to stat it
		fs, err := NewURIFilesystem(ctx, uri)
		if err != nil {
			return false, err
		}
		defer func() {
			if sftpFS, ok := fs.(*SFTPFileSystem); ok {
				//nolint:errcheck
				sftpFS.Close()
			}
		}()

		// Stat the path from the URI (relative to base path)
		// For sftp://user@host/path/to/file, we want to check "."
		// since the filesystem is already rooted at /path/to/file
		_, err = stdfs.Stat(fs, ".")
		return err == nil, err
	case "git", "git+https", "git+http", "git+ssh", "git+file":
		// Check if Git repository/file exists by attempting to create filesystem
		fs, err := NewURIFilesystem(ctx, uri)
		if err != nil {
			return false, err
		}

		// Stat the root path to verify access
		_, err = stdfs.Stat(fs, ".")
		return err == nil, err
	case "https", "http":
		_, err := http.NewRequestWithContext(ctx, http.MethodHead, uri.String(), nil)
		return err == nil, err
	}
	return false, fmt.Errorf("unsupported scheme: %s", uri.Scheme)
}

func NewReader(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	switch uri.Scheme {
	case "file":
		path := filepath.FromSlash(uri.Path)
		if fi, err := os.Stat(path); err != nil {
			return nil, err
		} else if fi.IsDir() {
			return nil, fmt.Errorf("expected file, got directory: %s", path)
		}

		return os.Open(path)
	case "sftp", "ssh":
		// Open SFTP filesystem and read file
		fs, err := NewURIFilesystem(ctx, uri)
		if err != nil {
			return nil, err
		}

		// Cast to SFTP filesystem
		sftpFS, ok := fs.(*SFTPFileSystem)
		if !ok {
			return nil, fmt.Errorf("expected SFTP filesystem")
		}

		// Check if path is a directory or file
		// The filesystem is rooted at the URI path, so check "."
		info, err := stdfs.Stat(fs, ".")
		if err != nil {
			return nil, errors.Join(err, sftpFS.Close())
		}

		if info.IsDir() {
			return nil, errors.Join(fmt.Errorf("expected file, got directory: %s", uri.Path), sftpFS.Close())
		}

		// Create file for the file at the base path
		file, err := NewSFTPReader(sftpFS.sftpClient, sftpFS.basePath)
		if err != nil {
			return nil, errors.Join(err, sftpFS.Close())
		}

		// Return a reader that closes both the file and the filesystem
		return file, nil
	case "git", "git+https", "git+http", "git+ssh", "git+file":
		// Git filesystems are read-only
		fs, err := NewURIFilesystem(ctx, uri)
		if err != nil {
			return nil, err
		}

		// Check if path is a directory or file
		info, err := stdfs.Stat(fs, ".")
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			return nil, fmt.Errorf("expected file, got directory: %s", uri.Path)
		}

		// Open and read the file
		file, err := fs.Open(".")
		if err != nil {
			return nil, err
		}

		return file, nil
	case "https", "http":
		req, err := http.NewRequestWithContext(ctx, "GET", uri.String(), nil)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		return resp.Body, nil
	}

	return nil, fmt.Errorf("unsupported scheme: %s", uri.Scheme)
}

func NewWriter(ctx context.Context, uri *url.URL) (io.WriteCloser, error) {
	switch uri.Scheme {
	case "file":
		path := filepath.FromSlash(uri.Path)
		if fi, err := os.Stat(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		} else if fi != nil && fi.IsDir() {
			return nil, fmt.Errorf("expected file, got directory: %s", path)
		}

		return os.Create(path)
	case "sftp", "ssh":
		// Open SFTP filesystem and create file
		fs, err := NewURIFilesystem(ctx, uri)
		if err != nil {
			return nil, err
		}

		// Cast to SFTP filesystem
		sftpFS, ok := fs.(*SFTPFileSystem)
		if !ok {
			return nil, fmt.Errorf("expected SFTP filesystem")
		}

		// Check if path exists and is a directory
		info, err := stdfs.Stat(fs, ".")
		if err == nil && info.IsDir() {
			return nil, errors.Join(fmt.Errorf("expected file, got directory: %s", uri.Path), sftpFS.Close())
		}

		// Create file for the file at the base path
		file, err := NewSFTPWriter(sftpFS.sftpClient, sftpFS.basePath)
		if err != nil {
			return nil, errors.Join(err, sftpFS.Close())
		}

		// Return a writer that closes both the file and the filesystem
		return file, nil
	case "git", "git+https", "git+http", "git+ssh", "git+file":
		// Git filesystems are read-only
		return nil, fmt.Errorf("git filesystems are read-only, cannot write to: %s", uri.String())
	}

	return nil, fmt.Errorf("unsupported scheme: %s", uri.Scheme)
}

func GetChecksum(ctx context.Context, uriStr string) (string, error) {
	uri, err := Parse(uriStr)
	if err != nil {
		return "", err
	}

	reader, err := NewReader(ctx, uri)
	if err != nil {
		return "", err
	}
	//nolint:errcheck
	defer reader.Close()

	return util.HashSHA256Reader(reader)
}
