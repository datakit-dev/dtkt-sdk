package network

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/env"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
)

const (
	DefaultCloudHost     = "api.datakit.cloud"
	DefaultSocketHost    = "localhost"
	DefaultSocketPattern = "dtkt-*.sock"
	DefaultTCPHost       = "127.0.0.1"
	DefaultTCPPort       = 50051
	MaxLocalPortSearch   = 255
)

var _ net.Addr = Address{}

type (
	Address      [2]string
	AddressProto interface {
		proto.Message
		GetNetwork() string
		GetTarget() string
	}
)

func ParseAddr(uriStr string) (addr Address, err error) {
	u, err := url.Parse(uriStr)
	if err != nil {
		return
	}

	// Validate scheme is a known network type
	netType := Type(u.Scheme)
	if !netType.IsValid() {
		err = fmt.Errorf("unsupported network type: %s", u.Scheme)
		return
	}

	// For SSH types, delegate to specialized parser
	if netType == SSHSocket || netType == SSHTCP {
		var sshURI *SSHRemoteURI
		sshURI, err = ParseSSHRemoteURI(uriStr)
		if err != nil {
			return
		}
		// Return simplified Address representation (validation happens in ParseSSHRemoteURI)
		return Addr(netType, sshURI.String()), nil
	}

	// Extract address portion based on network type
	var addrStr string
	switch netType {
	case Cloud:
		// cloud://hostname[:port][/path] - host is required, port/path optional
		if u.Host == "" {
			err = fmt.Errorf("cloud address requires hostname")
			return
		}
		addrStr = u.Host
		if u.Path != "" && u.Path != "/" {
			// Include path for resource identification
			addrStr = u.Host + u.Path
		}

	case Socket:
		// unix://[host]/path/to/socket - path is required
		// Note: url.Parse treats unix://path as Opaque, unix:///path as Path
		if u.Path == "" && u.Opaque == "" {
			err = fmt.Errorf("unix socket address requires path")
			return
		}
		// Use Opaque if Path is empty (happens with unix://path format)
		if u.Path != "" {
			addrStr = u.Path
		} else {
			addrStr = u.Opaque
		}
		// If there's a host component, it means the path is relative (unix://host/path)
		if u.Host != "" && u.Host != "localhost" {
			err = fmt.Errorf("unix socket path must be absolute (use unix://[localhost]/path/to/file, not unix://path)")
			return
		}

	case TCP:
		// tcp://[host]:port - port required, host defaults to 127.0.0.1
		if u.Host == "" {
			err = fmt.Errorf("TCP address requires at least a port (e.g., :8080)")
			return
		}

		var host, port string
		// Validate and normalize host:port format
		host, port, err = net.SplitHostPort(u.Host)
		if err != nil {
			err = fmt.Errorf("invalid TCP address format (expected [host]:port): %w", err)
			return
		}
		// Default to localhost if host is empty
		if host == "" {
			host = DefaultTCPHost
		}
		addrStr = net.JoinHostPort(host, port)

	default:
		err = fmt.Errorf("unsupported network type: %s", netType)
		return
	}

	if addrStr == "" {
		err = fmt.Errorf("address for %s network cannot be empty", netType)
		return
	}

	addr = Addr(netType, addrStr)
	err = addr.IsValid()
	return
}

func Addr(network Type, target ...string) Address {
	var addr string

	if len(target) == 1 {
		addr = target[0]
	}

	switch network {
	case Cloud, TCP:
		if len(target) == 2 {
			addr = net.JoinHostPort(target[0], target[1])
		}
	default:
		addr = path.Join(target...)
	}

	return Address{network.String(), addr}
}

func AddrFromProto(addr AddressProto) Address {
	return Address{addr.GetNetwork(), addr.GetTarget()}
}

func AddrToEnv(addr net.Addr) (vars []string) {
	vars = append(vars,
		fmt.Sprintf("%s=%s", env.Network, addr.Network()),
		fmt.Sprintf("%s=%s", env.Address, addr.String()),
	)
	return
}

func AddrFromEnv() Address {
	return Addr(Type(NetworkFromEnv()), AddressFromEnv())
}

func NetworkFromEnv() string {
	return os.Getenv(env.Network)
}

func AddressFromEnv() string {
	return os.Getenv(env.Address)
}

func (a Address) Network() string {
	return a[0]
}

func (a Address) String() string {
	switch a.Type() {
	case Cloud, TCP, SSHSocket, SSHTCP:
		if idx := strings.Index(a[1], "/"); idx > 0 {
			return a[1][:idx]
		}
	case Socket:
		if path, hasHost := strings.CutPrefix(a[1], DefaultSocketHost); hasHost {
			return path
		}
	}
	return a[1]
}

func (a Address) IsAvailable() bool {
	switch a.Type() {
	case SSHSocket, SSHTCP:
		// SSH types always return false as they are dial-only.
		return false
	case Cloud:
		// Cloud network always return true since binding to the edge network is
		// outside the scope of just knowing the address (requires valid config)
		// and will error on an invalid attempt .
		return true
	}

	conn, err := net.Dial(a.Network(), a.String())
	if err != nil {
		return true
	}
	//nolint:errcheck
	conn.Close()

	return false
}

func (a Address) IsValid() error {
	netType := a.Type()
	if !netType.IsValid() {
		return fmt.Errorf("unsupported network: %s", netType)
	}

	addrStr := a.String()
	if addrStr == "" {
		return fmt.Errorf("address for %s network cannot be empty", netType)
	}

	// Network-specific validation (simplified - just validate the core requirements)
	switch netType {
	case Socket:
		// Must be an absolute path
		if !filepath.IsAbs(addrStr) {
			return fmt.Errorf("unix socket path must be absolute: %s", addrStr)
		}

	case TCP:
		// Must be valid host:port format with port present
		if _, port, err := net.SplitHostPort(addrStr); err != nil {
			return fmt.Errorf("invalid TCP address (expected [host]:port): %w", err)
		} else if port == "" {
			return fmt.Errorf("TCP address requires port")
		}

	case Cloud:
		// Must have at least a hostname
		host := addrStr
		if idx := strings.Index(addrStr, "/"); idx > 0 {
			host = addrStr[:idx]
		}
		if host == "" {
			return fmt.Errorf("cloud address requires hostname")
		}
	}

	return nil
}

func (a Address) IsCloud() bool {
	return a.Type().IsCloud()
}

func (a Address) ToEnv() []string {
	return AddrToEnv(a)
}

func (a Address) ToProto() *corev1.Address {
	return &corev1.Address{
		Network: a.Network(),
		Target:  a.String(),
	}
}

func (a Address) Type() Type {
	return Type(a.Network())
}

func (a Address) URI() string {
	return a.Network() + "://" + a.String()
}

func (a Address) URL() *url.URL {
	switch a.Type() {
	case Cloud:
		return &url.URL{Scheme: "https", Host: a.String()}
	case TCP:
		return &url.URL{Scheme: "http", Host: a.String()}
	case Socket:
		return &url.URL{Scheme: a.Network(), Host: DefaultSocketHost, Path: a.String()}
	}

	addr := a.String()
	host := addr
	path := ""
	if idx := strings.Index(addr, "/"); idx > 0 {
		host = addr[:idx]
		path = addr[idx:]
	}

	return &url.URL{Scheme: a.Network(), Host: host, Path: path}
}

func (a Address) HTTP() *url.URL {
	switch a.Type() {
	case Cloud:
		return &url.URL{Scheme: "https", Host: a.String()}
	case TCP:
		return &url.URL{Scheme: "http", Host: a.String()}
	}
	// Use hash for stable, collision-free host names
	return &url.URL{Scheme: "http", Host: fmt.Sprintf("%s-%s", util.Slugify(a.Network()), util.HashShort(a.String()))}
}

func (a Address) HostPort() (string, string, error) {
	addr, err := a.Resolve()
	if err != nil {
		return "", "", err
	}
	return net.SplitHostPort(addr.String())
}

func (a Address) Resolve() (net.Addr, error) {
	addrStr := a.String()
	switch a.Type() {
	case Cloud:
		// Extract just the host:port portion (strip path if present)
		host := addrStr
		if idx := strings.Index(addrStr, "/"); idx > 0 {
			host = addrStr[:idx]
		}
		// Add default port 443 if not specified (cloud:// implies TLS)
		if _, _, err := net.SplitHostPort(host); err != nil {
			host = net.JoinHostPort(host, "443")
		}
		return net.ResolveTCPAddr("tcp", host)

	case Socket:
		// Validate socket path exists without actually dialing
		if _, err := os.Stat(addrStr); err != nil {
			return nil, fmt.Errorf("unix socket not found: %w", err)
		}
		// Return the address itself (it's already a valid net.Addr)
		return a, nil

	case TCP:
		return net.ResolveTCPAddr("tcp", addrStr)

	case SSHSocket, SSHTCP:
		// For SSH types, we need to parse the full URI to get the SSH endpoint
		uri := a.URL()
		sshURI, err := ParseSSHRemoteURI(uri.String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH address: %w", err)
		}
		// Resolve the SSH endpoint (not the remote address)
		return net.ResolveTCPAddr("tcp", sshURI.SSHEndpoint())

	default:
		return nil, fmt.Errorf("unsupported network: %s", a.Network())
	}
}
