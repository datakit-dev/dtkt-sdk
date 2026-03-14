package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
)

var _ Connector = (*SSHConnector)(nil)

var ErrSSHBind = errors.New("bind not supported: SSH connections are dial-only (use Cloud Connection for bidirectional connectivity)")

// SSHConnector is a dialer to a remote address (Unix socket or TCP) on a remote machine over SSH.
type SSHConnector struct {
	uri *SSHRemoteURI

	// SSH client management
	mu        sync.RWMutex
	sshClient *ssh.Client
	connected bool
}

// NewSSHConnector creates a new SSH connector from a parsed URI.
// Supports both ssh+unix:// and ssh+tcp:// schemes.
func NewSSHConnector(addr net.Addr) (_ *SSHConnector, err error) {
	uri, ok := addr.(*SSHRemoteURI)
	if !ok {
		uri, err = ParseSSHRemoteURI(fmt.Sprintf("%s://%s", addr.Network(), addr))
		if err != nil {
			return nil, err
		}
	}

	return &SSHConnector{
		uri: uri,
	}, nil
}

// Address returns the remote address (net.Addr interface).
func (c *SSHConnector) Address() net.Addr {
	return c.uri
}

// GRPCTarget returns the target string for dialing (e.g., for gRPC).
// Uses the standard GRPCTarget function to format based on remote network type.
func (c *SSHConnector) GRPCTarget() string {
	return GRPCTarget(c.uri.RemoteAddr)
}

// DialGRPC creates a gRPC client connection over SSH to the remote address.
func (c *SSHConnector) DialGRPC(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	// Ensure SSH connection is established
	if err := c.ensureSSHConnected(); err != nil {
		return nil, err
	}

	// Prepend our context dialer
	opts = append([]grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
			return c.dialContext(ctx)
		}),
	}, opts...)

	return grpc.NewClient(c.GRPCTarget(), opts...)
}

// DialContext creates a connection to the remote address with context support.
// This is useful for HTTP transports and other context-aware dialers.
func (c *SSHConnector) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	// Ignore network and addr - we always dial the configured remote address over SSH
	return c.dialContext(ctx)
}

// dialContext is the internal dial implementation that respects context.
// If RemoteAddr is nil (as with SFTP), this only establishes the SSH connection
// and returns nil (no remote dial needed).
func (c *SSHConnector) dialContext(ctx context.Context) (net.Conn, error) {
	if err := c.ensureSSHConnected(); err != nil {
		return nil, err
	}

	// If no remote address is specified (e.g., for SFTP), we're done
	// The caller just wanted to ensure the SSH connection is established
	if len(c.uri.RemoteAddr) == 0 {
		return nil, nil
	}

	c.mu.RLock()
	client := c.sshClient
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("SSH client not connected")
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Create a channel for the connection result
	type result struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan result, 1)

	// Dial in a goroutine so we can respect context cancellation
	go func() {
		conn, err := client.Dial(c.uri.RemoteAddr.Network(), c.uri.RemoteAddr.String())
		resultCh <- result{conn: conn, err: err}
	}()

	// Wait for either the connection or context cancellation
	select {
	case <-ctx.Done():
		// Context cancelled - the connection may still complete,
		// so we need to close it if it does
		go func() {
			if res := <-resultCh; res.conn != nil {
				//nolint:errcheck
				res.conn.Close()
			}
		}()
		return nil, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			return nil, fmt.Errorf("failed to dial remote address: %w", res.err)
		}
		return res.conn, nil
	}
}

// Bind is not supported for SSH connections as they are client-only.
// SSH connections cannot listen/bind - they can only dial remote addresses.
// For bidirectional connectivity with Bind support, use Cloud Connection.
func (c *SSHConnector) Bind(context.Context) (net.Listener, error) {
	return nil, ErrSSHBind
}

// Close closes the underlying SSH connection and releases resources.
func (c *SSHConnector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.sshClient == nil {
		return nil
	}

	err := c.sshClient.Close()
	c.sshClient = nil
	c.connected = false
	return err
}

// String returns a sanitized string representation of the connection URI.
func (c *SSHConnector) String() string {
	return c.uri.String()
}

// SSHClient returns the underlying SSH client for SFTP or other SSH operations.
// This implements the uri.SSHClientProvider interface, allowing network.SSHConnector
// to be used with uri.NewSFTPFileSystem and uri.NewSFTPConnector.
// The returned client is managed by SSHConnector and should NOT be closed by the caller.
// Returns nil if not yet connected - call ensureSSHConnected first or use DialContext.
func (c *SSHConnector) SSHClient() *ssh.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sshClient
}

// ensureSSHConnected establishes the SSH connection if not already connected.
// This is called lazily on first dial to avoid connecting until needed.
func (c *SSHConnector) ensureSSHConnected() error {
	c.mu.RLock()
	if c.connected && c.sshClient != nil {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.connected && c.sshClient != nil {
		return nil
	}

	// Get authentication methods from URI
	auths, err := c.uri.GetAuthMethods()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Get host key callback for verification
	hostKeyCallback, err := c.uri.GetHostKeyCallback()
	if err != nil {
		return fmt.Errorf("host key verification setup failed: %w", err)
	}

	// Query ConnectTimeout from SSH config
	connectTimeout := c.uri.getSSHConfigInt("ConnectTimeout", 5)

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User:            c.uri.User,
		Auth:            auths,
		HostKeyCallback: hostKeyCallback,
		Timeout:         time.Duration(connectTimeout) * time.Second,
	}

	// Establish SSH connection (with ProxyCommand support if configured)
	var client *ssh.Client
	proxyCmd := c.uri.getSSHConfigString("ProxyCommand", "")
	if proxyCmd != "" {
		// Use ProxyCommand to establish connection
		conn, err := c.dialViaProxyCommand(proxyCmd)
		if err != nil {
			return fmt.Errorf("ProxyCommand failed: %w", err)
		}

		// Perform SSH handshake over the ProxyCommand connection
		// Note: For ProxyCommand, we pass the actual remote address for host key verification
		addr := c.uri.SSHEndpoint()
		sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
		if err != nil {
			return fmt.Errorf("SSH handshake failed: %w", errors.Join(err, conn.Close()))
		}
		client = ssh.NewClient(sshConn, chans, reqs)
	} else {
		// Standard TCP connection
		var err error
		client, err = ssh.Dial("tcp", c.uri.SSHEndpoint(), sshConfig)
		if err != nil {
			return fmt.Errorf("SSH connection failed: %w", err)
		}
	}

	c.sshClient = client
	c.connected = true
	return nil
}

// dialViaProxyCommand executes the ProxyCommand and returns a connection.
func (c *SSHConnector) dialViaProxyCommand(proxyCmd string) (net.Conn, error) {
	// Expand SSH config variables in ProxyCommand
	// %h = hostname, %p = port, %r = remote username
	cmdStr := proxyCmd
	cmdStr = strings.ReplaceAll(cmdStr, "%h", c.uri.Host)
	cmdStr = strings.ReplaceAll(cmdStr, "%p", fmt.Sprintf("%d", c.uri.getPort()))
	cmdStr = strings.ReplaceAll(cmdStr, "%r", c.uri.User)

	// Parse command and arguments
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty ProxyCommand")
	}

	// Create command
	cmd := exec.Command(parts[0], parts[1:]...)

	// Get stdin/stdout pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ProxyCommand: %w", err)
	}

	// Return a connection wrapper with the remote address
	return &proxyCommandConn{
		cmd:        cmd,
		stdin:      stdin,
		stdout:     stdout,
		remoteAddr: c.uri.SSHEndpoint(),
	}, nil
}

// proxyCommandConn implements net.Conn for ProxyCommand connections.
type proxyCommandConn struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	remoteAddr string // Store the actual remote address for host key verification
}

func (c *proxyCommandConn) Read(b []byte) (int, error) {
	return c.stdout.Read(b)
}

func (c *proxyCommandConn) Write(b []byte) (int, error) {
	return c.stdin.Write(b)
}

func (c *proxyCommandConn) Close() error {
	errs := []error{
		c.stdin.Close(),
		c.stdout.Close(),
	}

	// Wait for command to finish (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		errs = append(errs,
			c.cmd.Process.Kill(),
			fmt.Errorf("ProxyCommand did not exit in time"),
		)
		return errors.Join(errs...)
	case err := <-done:
		errs = append(errs, err)
		return errors.Join(errs...)
	}
}

func (c *proxyCommandConn) LocalAddr() net.Addr {
	return &proxyCommandAddr{addr: "localhost"}
}

func (c *proxyCommandConn) RemoteAddr() net.Addr {
	// Return a TCP address that can be parsed by knownhosts
	addr, _ := net.ResolveTCPAddr("tcp", c.remoteAddr)
	if addr != nil {
		return addr
	}
	return &proxyCommandAddr{addr: c.remoteAddr}
}

func (c *proxyCommandConn) SetDeadline(t time.Time) error {
	// ProxyCommand connections don't support deadlines
	return nil
}

func (c *proxyCommandConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *proxyCommandConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// proxyCommandAddr implements net.Addr for ProxyCommand connections
type proxyCommandAddr struct {
	addr string
}

func (a *proxyCommandAddr) Network() string {
	return "proxycommand"
}

func (a *proxyCommandAddr) String() string {
	return a.addr
}
