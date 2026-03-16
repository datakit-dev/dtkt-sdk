package uri

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdfs "io/fs"
	"net/url"
	"path"
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	"github.com/pkg/sftp"
)

// SFTPFileSystem provides filesystem access over SFTP (SSH File Transfer Protocol).
// It implements a virtual filesystem that can read files from remote SSH hosts.
type SFTPFileSystem struct {
	sshConnector *network.SSHConnector
	sftpClient   *sftp.Client
	uri          *url.URL
	basePath     string
}

// NewSFTPFileSystem creates a new SFTP filesystem from an sftp|ssh:// URI.
// The URI should be in the format: sftp|ssh://user@host[:port]/path/to/directory
//
// Authentication and connection details can be specified via query parameters:
//   - identity=/path/to/key - SSH private key path
//   - ssh_config=/path/to/config - SSH config file
//   - known_hosts=/path/to/known_hosts - Known hosts file
//   - host_key_verify=known_hosts|tofu|fingerprint|none - Verification mode
//
// Example: sftp|ssh://user@host/path/to/files?identity=~/.ssh/id_ed25519
func NewSFTPFileSystem(ctx context.Context, uri *url.URL) (*SFTPFileSystem, error) {
	if !slices.Contains([]string{"sftp", "ssh"}, uri.Scheme) {
		return nil, fmt.Errorf("expected sftp|ssh:// scheme, got %s://", uri.Scheme)
	}

	// Create SSH remote URI for connection (no specific remote address for SFTP)
	sshRemoteURI, err := network.NewSSHRemoteURI(uri, network.Address{})
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH remote URI: %w", err)
	}

	// Create SSH connector
	sshConnector, err := network.NewSSHConnector(sshRemoteURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH connector: %w", err)
	}

	// Establish SSH connection
	_, err = sshConnector.DialContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to establish SSH connection: %w", errors.Join(err, sshConnector.Close()))
	}

	// Get SSH client
	sshClient := sshConnector.SSHClient()
	if sshClient == nil {
		return nil, errors.Join(fmt.Errorf("SSH client not available"), sshConnector.Close())
	}

	// Open SFTP session on the SSH connection
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", errors.Join(err, sshConnector.Close()))
	}

	// Extract base path from URI
	basePath := uri.Path
	if basePath == "" {
		basePath = "/"
	}

	return &SFTPFileSystem{
		sshConnector: sshConnector,
		sftpClient:   sftpClient,
		uri:          uri,
		basePath:     basePath,
	}, nil
}

// Close closes the SFTP session and the underlying SSH connection.
func (fs *SFTPFileSystem) Close() error {
	var sftpErr, sshErr error
	if fs.sftpClient != nil {
		sftpErr = fs.sftpClient.Close()
	}
	if fs.sshConnector != nil {
		sshErr = fs.sshConnector.Close()
	}
	if sftpErr != nil {
		return sftpErr
	}
	return sshErr
}

// ReadFile reads a file from the remote filesystem.
func (fs *SFTPFileSystem) ReadFile(name string) ([]byte, error) {
	fullPath := fs.resolvePath(name)
	file, err := fs.sftpClient.Open(fullPath)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer file.Close()
	return io.ReadAll(file)
}

// ReadDir reads a directory from the remote filesystem.
func (fs *SFTPFileSystem) ReadDir(name string) ([]stdfs.DirEntry, error) {
	fullPath := fs.resolvePath(name)
	entries, err := fs.sftpClient.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	// Convert []os.FileInfo to []stdfs.DirEntry
	dirEntries := make([]stdfs.DirEntry, len(entries))
	for i, entry := range entries {
		dirEntries[i] = &sftpDirEntry{entry}
	}
	return dirEntries, nil
}

// Stat returns file info for a remote file.
func (fs *SFTPFileSystem) Stat(name string) (stdfs.FileInfo, error) {
	fullPath := fs.resolvePath(name)
	return fs.sftpClient.Stat(fullPath)
}

// Open opens a file for reading.
func (fs *SFTPFileSystem) Open(name string) (stdfs.File, error) {
	fullPath := fs.resolvePath(name)
	return fs.sftpClient.Open(fullPath)
}

// Walk walks the remote directory tree.
func (fs *SFTPFileSystem) Walk(root string, walkFn stdfs.WalkDirFunc) error {
	fullPath := fs.resolvePath(root)
	return fs.walk(fullPath, walkFn)
}

// walk is the internal implementation of Walk
func (fs *SFTPFileSystem) walk(dir string, walkFn stdfs.WalkDirFunc) error {
	entries, err := fs.sftpClient.ReadDir(dir)
	if err != nil {
		return walkFn(dir, nil, err)
	}

	for _, entry := range entries {
		fullPath := path.Join(dir, entry.Name())
		dirEntry := sftpDirEntry{entry}

		err := walkFn(fullPath, dirEntry, nil)
		if err != nil {
			if err == stdfs.SkipDir {
				if entry.IsDir() {
					continue
				}
			}
			return err
		}

		if entry.IsDir() {
			if err := fs.walk(fullPath, walkFn); err != nil {
				return err
			}
		}
	}

	return nil
}

// resolvePath resolves a relative path against the base path
func (fs *SFTPFileSystem) resolvePath(name string) string {
	if path.IsAbs(name) {
		return name
	}
	return path.Join(fs.basePath, name)
}

// BasePath returns the base path of the filesystem
func (fs *SFTPFileSystem) BasePath() string {
	return fs.basePath
}

// URI returns the original URI
func (fs *SFTPFileSystem) URI() *url.URL {
	return fs.uri
}

// sftpDirEntry wraps stdfs.FileInfo to implement stdfs.DirEntry
type sftpDirEntry struct {
	info stdfs.FileInfo
}

func (d sftpDirEntry) Name() string {
	return d.info.Name()
}

func (d sftpDirEntry) IsDir() bool {
	return d.info.IsDir()
}

func (d sftpDirEntry) Type() stdfs.FileMode {
	return d.info.Mode().Type()
}

func (d sftpDirEntry) Info() (stdfs.FileInfo, error) {
	return d.info, nil
}

// SFTPReader wraps an SFTP file reader with io.ReadCloser interface
type SFTPReader struct {
	file *sftp.File
	fs   *SFTPFileSystem
}

// NewSFTPReader creates a reader for a remote file over SFTP
func NewSFTPReader(sftpClient *sftp.Client, remotePath string) (io.ReadCloser, error) {
	file, err := sftpClient.Open(remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote file: %w", err)
	}
	return &SFTPReader{file: file}, nil
}

func (r *SFTPReader) Read(p []byte) (n int, err error) {
	return r.file.Read(p)
}

func (r *SFTPReader) Close() error {
	var fileErr, fsErr error
	if r.file != nil {
		fileErr = r.file.Close()
	}
	if r.fs != nil {
		fsErr = r.fs.Close()
	}
	if fileErr != nil {
		return fileErr
	}
	return fsErr
}

// SFTPWriter wraps an SFTP file writer with io.WriteCloser interface
type SFTPWriter struct {
	file *sftp.File
	fs   *SFTPFileSystem
}

// NewSFTPWriter creates a writer for a remote file over SFTP
func NewSFTPWriter(sftpClient *sftp.Client, remotePath string) (io.WriteCloser, error) {
	file, err := sftpClient.Create(remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote file: %w", err)
	}
	return &SFTPWriter{file: file}, nil
}

func (w *SFTPWriter) Write(p []byte) (n int, err error) {
	return w.file.Write(p)
}

func (w *SFTPWriter) Close() error {
	var fileErr, fsErr error
	if w.file != nil {
		fileErr = w.file.Close()
	}
	if w.fs != nil {
		fsErr = w.fs.Close()
	}
	if fileErr != nil {
		return fileErr
	}
	return fsErr
}
