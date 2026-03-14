package format

import (
	"archive/zip"
	"io"
	"io/fs"
	"path"
	"strings"
)

var _ fs.File = (*zipFile)(nil)

type (
	zipFS struct {
		*zip.Reader
	}
	zipEntry struct {
		*zip.File
	}
	zipFile struct {
		io.ReadCloser
		fi fs.FileInfo
	}
)

func NewZipFS(file io.ReaderAt, size int64) (*zipFS, error) {
	r, err := zip.NewReader(file, size)
	if err != nil {
		return nil, err
	}

	return &zipFS{Reader: r}, nil
}

func (f *zipFile) Stat() (fs.FileInfo, error) { return f.fi, nil }

func (z zipFS) Open(name string) (fs.File, error) {
	name = path.Clean(name)
	for _, f := range z.File {
		if path.Clean(f.Name) == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			return &zipFile{
				ReadCloser: rc,
				fi:         f.FileInfo(),
			}, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (z zipFS) ReadDir(name string) ([]fs.DirEntry, error) {
	prefix := path.Clean(name)
	if prefix == "." {
		prefix = ""
	} else {
		prefix += "/"
	}

	seen := make(map[string]*zip.File)
	for _, f := range z.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		trimmed := strings.TrimPrefix(f.Name, prefix)
		parts := strings.SplitN(trimmed, "/", 2)

		if len(parts) > 0 && parts[0] != "" {
			base := parts[0]
			// Only return top-level entries
			if _, ok := seen[base]; !ok {
				seen[base] = f
			}
		}
	}

	if len(seen) == 0 {
		return nil, fs.ErrNotExist
	}

	var entries []fs.DirEntry
	for _, f := range seen {
		entries = append(entries, zipEntry{f})
	}
	return entries, nil
}

func (z zipEntry) Name() string      { return path.Base(z.File.Name) }
func (z zipEntry) IsDir() bool       { return z.File.FileInfo().IsDir() }
func (z zipEntry) Type() fs.FileMode { return z.File.Mode().Type() }
func (z zipEntry) Info() (fs.FileInfo, error) {
	return z.FileInfo(), nil
}
