package network

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	osuser "os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Implements net.Addr for stdlib interop
var _ net.Addr = (*SSHRemoteURI)(nil)

// HostKeyVerification defines how to verify host keys
type HostKeyVerification string

const (
	SSHUnixAddrFormat = "[user[:password]@]host[:port]/path/to/socket[?identity=key&ssh_config=file&...]"
	SSHTCPAddrFormat  = "[user[:password]@]host[:port]/remote-host:remote-port[?identity=key&ssh_config=file&...]"
)

const (
	// VerifyKnownHosts uses standard SSH known_hosts file(s)
	VerifyKnownHosts HostKeyVerification = "known_hosts"
	// VerifyTOFU trusts on first use, then remembers
	VerifyTOFU HostKeyVerification = "tofu"
	// VerifyFingerprint verifies against a specific fingerprint
	VerifyFingerprint HostKeyVerification = "fingerprint"
	// VerifyNone skips verification (INSECURE - use only for testing)
	VerifyNone HostKeyVerification = "none"
)

// SSHRemoteURI represents a parsed ssh+unix:// or ssh+tcp:// URI
type SSHRemoteURI struct {
	User     string
	Password string // Not recommended - prefer key-based auth
	Host     string
	Port     int

	// Remote address to dial via SSH (unix socket path or tcp host:port)
	RemoteAddr Address

	// Authentication
	IdentityFiles []string // SSH private key files

	// Configuration
	SSHConfigFile string             // Path to SSH config file
	sshConfig     *ssh_config.Config // Parsed SSH config

	// Host Key Verification
	HostKeyVerify   HostKeyVerification
	KnownHostsFiles []string // Paths to known_hosts files
	HostFingerprint string   // Expected host key fingerprint (for VerifyFingerprint mode)

	// Retain original parsed uri
	uri *url.URL
}

// ParseSSHRemoteURI parses a URI in the format:
// ssh+unix://[user[:password]@]host[:port]/path/to/socket[?identity=key&ssh_config=file&...]
// ssh+tcp://[user[:password]@]host[:port]/remote-host:remote-port[?identity=key&ssh_config=file&...]
//
// Supported query parameters:
//   - identity: Path to SSH private key (can be specified multiple times)
//   - ssh_config: Path to SSH config file
//   - known_hosts: Path to known_hosts file (can be specified multiple times)
//   - host_key_verify: Verification mode (known_hosts, tofu, fingerprint, none)
//   - fingerprint: Expected host key fingerprint (for fingerprint verification)
//
// Other SSH options like ProxyCommand, ServerAliveInterval, etc. are read from SSH config.
func ParseSSHRemoteURI(uri string) (*SSHRemoteURI, error) {
	// Parse the URI
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid URI: %w", err)
	}

	// Validate scheme
	if u.Scheme != SSHSocket.String() && u.Scheme != SSHTCP.String() {
		return nil, fmt.Errorf("invalid scheme: expected '%s' or '%s', got '%s'", SSHSocket, SSHTCP, u.Scheme)
	}

	// Extract and validate remote address based on scheme
	remotePath := u.Path
	if remotePath == "" {
		return nil, fmt.Errorf("remote address is required")
	}

	// Determine remote network type and parse address
	var remoteNetwork Type
	var remoteAddrStr string

	switch u.Scheme {
	case "ssh+unix":
		remoteNetwork = Socket
		remoteAddrStr = remotePath // Unix socket path
	case "ssh+tcp":
		remoteNetwork = TCP
		// Strip leading slash from path to get host:port
		remoteAddrStr = strings.TrimPrefix(remotePath, "/")
		if remoteAddrStr == "" {
			return nil, fmt.Errorf("remote TCP address is required")
		}
	}

	// Create and validate remote address
	remoteAddr := Addr(remoteNetwork, remoteAddrStr)
	if err := remoteAddr.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid remote address: %w", err)
	}

	// Delegate to NewSSHRemoteURI for common initialization
	return NewSSHRemoteURI(u, remoteAddr)
}

// NewSSHRemoteURI creates an SSHRemoteURI from an already-parsed url.URL.
// This is useful when you have a URI with a different scheme (e.g., sftp://)
// that you want to convert to an SSH connection.
//
// Unlike ParseSSHRemoteURI, this function doesn't validate the scheme,
// allowing it to work with sftp://, ssh://, or other schemes that map to SSH connections.
//
// The remoteAddr parameter specifies what to dial on the remote host after SSH connection.
// For SFTP, this should be nil or empty since SFTP uses the SSH subsystem.
func NewSSHRemoteURI(u *url.URL, remoteAddr Address) (*SSHRemoteURI, error) {
	if u == nil {
		return nil, fmt.Errorf("URL cannot be nil")
	}

	result := &SSHRemoteURI{
		Port:          0,                // Will be set from URI, SSH config, or default
		HostKeyVerify: VerifyKnownHosts, // Default to secure verification
		uri:           u,
		RemoteAddr:    remoteAddr,
	}

	// Extract user and password
	if u.User != nil {
		result.User = u.User.Username()
		if pass, ok := u.User.Password(); ok {
			result.Password = pass
		}
	}

	// Extract host and port
	result.Host = u.Hostname()
	if result.Host == "" {
		return nil, fmt.Errorf("host is required")
	}

	if portStr := u.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
		result.Port = port
	}

	// Parse query parameters
	if err := result.parseQueryParams(u.Query()); err != nil {
		return nil, err
	}

	// Initialize SSH configuration
	if err := result.initializeSSHConfig(); err != nil {
		return nil, err
	}

	// Apply defaults from SSH config or standard locations
	if err := result.applyDefaults(); err != nil {
		return nil, err
	}

	return result, nil
}

// initializeSSHConfig loads the SSH config file if specified or uses the default location
func (u *SSHRemoteURI) initializeSSHConfig() error {
	// Load SSH config file (default or specified)
	if u.SSHConfigFile == "" {
		// Use default SSH config location
		home, err := os.UserHomeDir()
		if err == nil {
			u.SSHConfigFile = filepath.Join(home, ".ssh", "config")
		}
	}

	if u.SSHConfigFile != "" {
		// Try to load config file (ignore errors if file doesn't exist)
		if _, err := os.Stat(u.SSHConfigFile); err == nil {
			if err := u.loadSSHConfig(); err != nil {
				return fmt.Errorf("failed to load SSH config: %w", err)
			}
		}
	}

	return nil
}

// URI returns retained original
func (u *SSHRemoteURI) URI() *url.URL {
	return u.uri
}

// Network returns the network type in order to fulfill net.Addr interface
func (u *SSHRemoteURI) Network() string {
	return u.uri.Scheme
}

// String returns a sanitized string representation of the address in order to
// fulfill net.Addr interface
func (u *SSHRemoteURI) String() string {
	return strings.TrimPrefix(u.uri.String(), u.uri.Scheme+"://")
}

// parseQueryParams parses query parameters from the URI
func (u *SSHRemoteURI) parseQueryParams(query url.Values) error {
	// Identity files (SSH keys)
	if identities := query["identity"]; len(identities) > 0 {
		for _, id := range identities {
			expanded, err := ExpandPath(id)
			if err != nil {
				return fmt.Errorf("invalid identity path %s: %w", id, err)
			}
			u.IdentityFiles = append(u.IdentityFiles, expanded)
		}
	}

	// SSH config file
	if sshConfig := query.Get("ssh_config"); sshConfig != "" {
		expanded, err := ExpandPath(sshConfig)
		if err != nil {
			return fmt.Errorf("invalid ssh_config path: %w", err)
		}
		u.SSHConfigFile = expanded
	}

	// Known hosts files
	if knownHosts := query["known_hosts"]; len(knownHosts) > 0 {
		for _, kh := range knownHosts {
			expanded, err := ExpandPath(kh)
			if err != nil {
				return fmt.Errorf("invalid known_hosts path %s: %w", kh, err)
			}
			u.KnownHostsFiles = append(u.KnownHostsFiles, expanded)
		}
	}

	// Host key verification mode
	if verify := query.Get("host_key_verify"); verify != "" {
		u.HostKeyVerify = HostKeyVerification(verify)
		switch u.HostKeyVerify {
		case VerifyKnownHosts, VerifyTOFU, VerifyFingerprint, VerifyNone:
			// Valid
		default:
			return fmt.Errorf("invalid host_key_verify: %s (must be known_hosts, tofu, fingerprint, or none)", verify)
		}
	}

	// Host key fingerprint (for fingerprint verification)
	if fingerprint := query.Get("fingerprint"); fingerprint != "" {
		u.HostFingerprint = fingerprint
		if u.HostKeyVerify != VerifyFingerprint {
			u.HostKeyVerify = VerifyFingerprint
		}
	}

	return nil
}

// loadSSHConfig loads and parses the SSH config file
func (u *SSHRemoteURI) loadSSHConfig() error {
	f, err := os.Open(u.SSHConfigFile)
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return err
	}
	u.sshConfig = cfg
	return nil
}

// applyDefaults applies minimal defaults (user is set here for convenience)
func (u *SSHRemoteURI) applyDefaults() error {
	// If no user specified, try SSH config, then environment
	// This is done upfront since user is needed frequently
	if u.User == "" {
		u.User = u.getUser()
	}

	return nil
}

// getSSHConfigString queries ssh_config for a string value with precedence
func (u *SSHRemoteURI) getSSHConfigString(key string, defaultValue string) string {
	if u.sshConfig == nil {
		return defaultValue
	}

	val, err := u.sshConfig.Get(u.Host, key)
	if err != nil || val == "" {
		// Fall back to ssh_config library default
		if libDefault := ssh_config.Default(key); libDefault != "" {
			return libDefault
		}
		return defaultValue
	}
	return val
}

// getSSHConfigInt queries ssh_config for an integer value
func (u *SSHRemoteURI) getSSHConfigInt(key string, defaultValue int) int {
	strVal := u.getSSHConfigString(key, "")
	if strVal == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(strVal)
	if err != nil {
		return defaultValue
	}
	return val
}

// getSSHConfigStrings queries ssh_config for multi-value directives
func (u *SSHRemoteURI) getSSHConfigStrings(key string) []string {
	if u.sshConfig == nil {
		return nil
	}

	vals, err := u.sshConfig.GetAll(u.Host, key)
	if err != nil {
		return nil
	}
	return vals
}

// getPort returns port with proper precedence: URI > SSH config > default
func (u *SSHRemoteURI) getPort() int {
	// URI explicitly specified port
	if u.Port != 0 {
		return u.Port
	}

	// SSH config
	return u.getSSHConfigInt("Port", 22)
}

// getUser returns user with precedence: URI > SSH config > environment
func (u *SSHRemoteURI) getUser() string {
	if u.User != "" {
		return u.User
	}

	user := u.getSSHConfigString("User", "")
	if user != "" {
		return user
	}

	// Environment fallback
	curr, err := osuser.Current()
	if err == nil && curr.Username != "" {
		return curr.Username
	}
	return os.Getenv("LOGNAME")
}

// getIdentityFiles returns identity files with precedence: URI > SSH config > defaults
func (u *SSHRemoteURI) getIdentityFiles() ([]string, error) {
	// URI query params take precedence
	if len(u.IdentityFiles) > 0 {
		return u.IdentityFiles, nil
	}

	// Query SSH config
	configIdentities := u.getSSHConfigStrings("IdentityFile")
	if len(configIdentities) > 0 {
		expanded := make([]string, 0, len(configIdentities))
		for _, id := range configIdentities {
			path, err := ExpandPath(id)
			if err == nil {
				expanded = append(expanded, path)
			}
		}
		if len(expanded) > 0 {
			return expanded, nil
		}
	}

	// Fall back to default identity files
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	defaults := []string{
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	}

	found := []string{}
	for _, key := range defaults {
		if _, err := os.Stat(key); err == nil {
			found = append(found, key)
		}
	}

	return found, nil
}

// getKnownHostsFiles returns known_hosts files with precedence: URI > SSH config > defaults
func (u *SSHRemoteURI) getKnownHostsFiles() []string {
	// URI query params take precedence
	if len(u.KnownHostsFiles) > 0 {
		return u.KnownHostsFiles
	}

	// Query SSH config
	configKH := u.getSSHConfigStrings("UserKnownHostsFile")
	if len(configKH) > 0 {
		expanded := make([]string, 0, len(configKH))
		for _, kh := range configKH {
			path, err := ExpandPath(kh)
			if err == nil {
				expanded = append(expanded, path)
			}
		}
		if len(expanded) > 0 {
			return expanded
		}
	}

	// Fall back to default known_hosts
	home, err := os.UserHomeDir()
	if err == nil {
		defaultKH := filepath.Join(home, ".ssh", "known_hosts")
		if _, err := os.Stat(defaultKH); err == nil {
			return []string{defaultKH}
		}
	}

	return nil
}

// SSHEndpoint returns the host:port string for SSH connection.
// It resolves the actual hostname from SSH config's HostName directive if available.
func (u *SSHRemoteURI) SSHEndpoint() string {
	return fmt.Sprintf("%s:%d", u.getHostName(), u.getPort())
}

// getHostName returns the actual hostname to connect to.
// It checks SSH config's HostName directive first, then falls back to the URI's host.
func (u *SSHRemoteURI) getHostName() string {
	// Try to get HostName from SSH config
	hostname := u.getSSHConfigString("HostName", "")
	if hostname != "" {
		return hostname
	}
	// Fall back to the host from the URI
	return u.Host
}

// GetAuthMethods returns SSH authentication methods based on the URI and available credentials
func (u *SSHRemoteURI) GetAuthMethods() ([]ssh.AuthMethod, error) {
	auths := []ssh.AuthMethod{}

	// If password is in URI, use it (not recommended)
	if u.Password != "" {
		auths = append(auths, ssh.Password(u.Password))
	}

	// Get identity files (from URI, SSH config, or defaults)
	identityFiles, err := u.getIdentityFiles()
	if err == nil {
		// Try each identity file
		for _, keyFile := range identityFiles {
			if a, err := publicKey(keyFile); err == nil {
				auths = append(auths, a)
			}
		}
	}

	// Try SSH agent as fallback
	if a, err := agentAuth(); err == nil {
		auths = append(auths, a)
	}

	if len(auths) == 0 {
		return nil, fmt.Errorf("no authentication methods available (tried %d identity files, SSH agent)", len(identityFiles))
	}

	return auths, nil
}

// GetHostKeyCallback returns the appropriate host key callback based on verification mode
func (u *SSHRemoteURI) GetHostKeyCallback() (ssh.HostKeyCallback, error) {
	switch u.HostKeyVerify {
	case VerifyKnownHosts:
		khFiles := u.getKnownHostsFiles()
		if len(khFiles) == 0 {
			return nil, fmt.Errorf("known_hosts verification requested but no known_hosts files found")
		}
		callback, err := knownhosts.New(khFiles...)
		if err != nil {
			return nil, fmt.Errorf("failed to load known_hosts: %w", err)
		}
		return callback, nil

	case VerifyTOFU:
		// Trust On First Use - accept any key first time, then remember
		return u.tofuHostKeyCallback()

	case VerifyFingerprint:
		if u.HostFingerprint == "" {
			return nil, fmt.Errorf("fingerprint verification requested but no fingerprint provided")
		}
		return u.fingerprintHostKeyCallback()

	case VerifyNone:
		// INSECURE - only for testing
		return ssh.InsecureIgnoreHostKey(), nil
	default:
		return nil, fmt.Errorf("unknown host key verification mode: %s", u.HostKeyVerify)
	}
}

// tofuHostKeyCallback implements Trust On First Use
func (u *SSHRemoteURI) tofuHostKeyCallback() (ssh.HostKeyCallback, error) {
	// Get or create TOFU known_hosts file
	tofuFile := u.getTOFUKnownHostsFile()

	// Ensure directory exists
	dir := filepath.Dir(tofuFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create TOFU directory: %w", err)
	}

	// If file doesn't exist, create it
	if _, err := os.Stat(tofuFile); os.IsNotExist(err) {
		if f, err := os.OpenFile(tofuFile, os.O_CREATE|os.O_WRONLY, 0600); err == nil {
			//nolint:errcheck
			f.Close()
		}
	}

	// Load existing known_hosts
	callback, err := knownhosts.New(tofuFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TOFU known_hosts: %w", err)
	}

	// Wrap to add new keys on first use
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := callback(hostname, remote, key)
		if err != nil {
			// Check if this is a "not in known_hosts" error
			if _, ok := err.(*knownhosts.KeyError); ok {
				// Add the key to known_hosts
				if addErr := u.addToKnownHosts(tofuFile, hostname, key); addErr != nil {
					return fmt.Errorf("failed to add host key: %w", addErr)
				}
				return nil // Accept the key
			}
			return err
		}
		return nil
	}, nil
}

// fingerprintHostKeyCallback verifies against a specific fingerprint
func (u *SSHRemoteURI) fingerprintHostKeyCallback() (ssh.HostKeyCallback, error) {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		// Also try legacy MD5 format
		legacyFingerprint := fingerprintMD5(key)

		if fingerprint == u.HostFingerprint || legacyFingerprint == u.HostFingerprint {
			return nil
		}
		return fmt.Errorf("host key fingerprint mismatch: expected %s, got %s (SHA256) or %s (MD5)",
			u.HostFingerprint, fingerprint, legacyFingerprint)
	}, nil
}

// getTOFUKnownHostsFile returns the path for TOFU known_hosts
func (u *SSHRemoteURI) getTOFUKnownHostsFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/ssh_tofu_known_hosts"
	}
	return filepath.Join(home, ".ssh", "known_hosts_tofu")
}

// addToKnownHosts adds a host key to the known_hosts file
func (u *SSHRemoteURI) addToKnownHosts(file, hostname string, key ssh.PublicKey) error {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer f.Close()

	line := knownhosts.Line([]string{hostname}, key)
	_, err = f.WriteString(line + "\n")
	return err
}

// fingerprintMD5 returns the legacy MD5 fingerprint
func fingerprintMD5(key ssh.PublicKey) string {
	hash := md5.Sum(key.Marshal())
	hexHash := hex.EncodeToString(hash[:])
	// Format as MD5:xx:xx:xx:...
	var formatted strings.Builder
	formatted.WriteString("MD5:")
	for i := 0; i < len(hexHash); i += 2 {
		if i > 0 {
			formatted.WriteString(":")
		}
		formatted.WriteString(hexHash[i : i+2])
	}
	return formatted.String()
}

// ExpandPath expands ~ to home directory and resolves relative paths
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Make absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	return absPath, nil
}

// publicKey loads an SSH private key file and returns an auth method
func publicKey(privateKeyFile string) (ssh.AuthMethod, error) {
	k, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(k)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}

// agentAuth returns an SSH agent-based authentication method
func agentAuth() (ssh.AuthMethod, error) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}

	client := agent.NewClient(conn)
	return ssh.PublicKeysCallback(client.Signers), nil
}
