package network

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

const (
	// Cloud is a custom network type for binding and dialing (through a proxy)
	// an Integration on DataKit Cloud edge network. Its address must resolve
	// using DNS to a server which supports HTTP/1.X and HTTP/2 protocols over TLS.
	// Example: cloud://api.datakit.cloud
	Cloud Type = "cloud"
	// Socket is the default network type on compatible platforms. Its
	// address must be a (absolute) file path of a Unix domain socket to a server
	// which supports HTTP/1.X and HTTP/2 protocols over H2C (HTTP/2 w/o TLS).
	// Example: unix:///path/to/file.sock
	Socket Type = "unix"
	// SSHSocket combines a secure shell (SSH) connection with a Unix domain socket
	// on the server which must meet the same requirements as the socket network
	// described above.
	// Example: ssh+unix://[user[:password]@]host[:port]/server/file.sock[?[ssh_config=local/file]]
	SSHSocket Type = "ssh+unix"
	// SSHTCP combines a secure shell (SSH) connection with a TCP connection
	// on the remote server, allowing tunneling to remote TCP services.
	// Example: ssh+tcp://[user[:password]@]host[:port]/remote-host:remote-port[?[ssh_config=local/file]]
	SSHTCP Type = "ssh+tcp"
	// TCP is the default network type on platforms that do not support
	// Unix domain sockets. Its address must resolve using DNS to a server which
	// supports HTTP/1.X and HTTP/2 protocols over H2C (HTTP/2 w/o TLS).
	// Example: tcp://host[:port]
	TCP Type = "tcp"
)

type Type string

func Types() []Type {
	return []Type{
		Cloud,
		Socket,
		SSHSocket,
		SSHTCP,
		TCP,
	}
}

func (t Type) DefaultAddress() (addr Address, err error) {
	switch t {
	case SSHSocket, SSHTCP:
		err = fmt.Errorf("%s network is dial-only", t)
		return
	case Cloud:
		return Addr(t, DefaultCloudHost), nil
	case Socket:
		var file *os.File
		file, err = os.CreateTemp("", DefaultSocketPattern)
		if err != nil {
			return
		}
		//nolint:errcheck
		defer file.Close()
		//nolint:errcheck
		defer os.Remove(file.Name())

		return Addr(t, file.Name()), nil
	case TCP:
		var port int
		port, err = util.GetFreePort(DefaultTCPHost, DefaultTCPPort, DefaultTCPPort+MaxLocalPortSearch)
		if err != nil {
			return
		}

		return Addr(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(port))), nil
	}

	err = fmt.Errorf("unsupported network type: %s", t)
	return
}

func (t Type) IsSocket() bool {
	return t == Socket || t == SSHSocket
}

func (t Type) IsCloud() bool {
	return t == Cloud
}

func (t Type) Format() string {
	switch t {
	case Cloud:
		return DefaultCloudHost
	case Socket:
		return "[localhost]/path/to/socket"
	case SSHSocket:
		return SSHUnixAddrFormat
	case SSHTCP:
		return SSHTCPAddrFormat
	case TCP:
		return "[host]:port"
	}
	return ""
}

func (t Type) String() string {
	return string(t)
}

func (t Type) IsValid() bool {
	return slices.Contains(util.StringSlice(Types()), t.String())
}
