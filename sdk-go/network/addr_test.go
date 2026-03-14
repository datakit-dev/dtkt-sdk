package network_test

import (
	"net"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
)

func TestParseAddr(t *testing.T) {
	for idx, test := range []struct {
		name    string
		input   string
		str     string
		network string
		valid   bool
	}{
		// Cloud addresses
		{
			name:    "cloud with hostname",
			input:   "cloud://api.datakit.cloud",
			str:     "api.datakit.cloud",
			network: "cloud",
			valid:   true,
		},
		{
			name:    "cloud with host:port",
			input:   "cloud://1.2.3.4:9090",
			str:     "1.2.3.4:9090",
			network: "cloud",
			valid:   true,
		},
		{
			name:    "cloud with path",
			input:   "cloud://api.datakit.cloud/connectors/my-connector",
			str:     "api.datakit.cloud",
			network: "cloud",
			valid:   true,
		},
		{
			name:    "cloud with port and path",
			input:   "cloud://api.datakit.cloud:8443/connectors/test",
			str:     "api.datakit.cloud:8443",
			network: "cloud",
			valid:   true,
		},
		{
			name:  "cloud without hostname",
			input: "cloud://",
			valid: false,
		},
		// TCP addresses
		{
			name:    "tcp with host:port",
			input:   "tcp://127.0.0.1:9090",
			str:     "127.0.0.1:9090",
			network: "tcp",
			valid:   true,
		},
		{
			name:    "tcp with port only (defaults to 127.0.0.1)",
			input:   "tcp://:9090",
			str:     "127.0.0.1:9090",
			network: "tcp",
			valid:   true,
		},
		{
			name:    "tcp with hostname:port",
			input:   "tcp://localhost:8080",
			str:     "localhost:8080",
			network: "tcp",
			valid:   true,
		},
		{
			name:  "tcp without port",
			input: "tcp://localhost",
			valid: false,
		},
		{
			name:  "tcp empty",
			input: "tcp://",
			valid: false,
		},
		// Unix socket addresses
		{
			name:    "unix with absolute path",
			input:   "unix:///tmp/test.sock",
			str:     "/tmp/test.sock",
			network: "unix",
			valid:   true,
		},
		{
			name:    "unix with absolute path (var)",
			input:   "unix:///var/run/app.sock",
			str:     "/var/run/app.sock",
			network: "unix",
			valid:   true,
		},
		{
			name:  "unix with relative path",
			input: "unix://relative/path.sock",
			valid: false,
		},
		{
			name:  "unix without path",
			input: "unix://",
			valid: false,
		},
		// SSH addresses
		{
			name:    "ssh+unix with basic format",
			input:   "ssh+unix://user@host:22/tmp/remote.sock",
			network: "ssh+unix",
			valid:   true,
		},
		{
			name:    "ssh+tcp with basic format",
			input:   "ssh+tcp://user@host:22/localhost:8080",
			network: "ssh+tcp",
			valid:   true,
		},
		// Invalid cases
		{
			name:  "invalid scheme",
			input: "foo://bar",
			valid: false,
		},
		{
			name:  "missing scheme",
			input: "localhost:8080",
			valid: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			addr, err := network.ParseAddr(test.input)
			if err != nil && test.valid {
				t.Fatalf("test %d (%s) expected to be valid, got: %s", idx, test.name, err)
			} else if err == nil && !test.valid {
				t.Fatalf("test %d (%s) expected to be invalid, but succeeded with: %s", idx, test.name, addr)
			}

			if test.valid {
				if test.str != "" && test.str != addr.String() {
					t.Errorf("test %d (%s) expected string: %s, got: %s", idx, test.name, test.str, addr.String())
				}
				if test.network != "" && test.network != addr.Network() {
					t.Errorf("test %d (%s) expected network: %s, got: %s", idx, test.name, test.network, addr.Network())
				}
				t.Logf("network: %s address: %s uri: %s", addr.Network(), addr, addr.URL())
			}
		})
	}
}

func TestAddress_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		addr  network.Address
		valid bool
	}{
		{
			name:  "valid cloud address",
			addr:  network.Addr(network.Cloud, "api.datakit.cloud"),
			valid: true,
		},
		{
			name:  "valid cloud with path",
			addr:  network.Addr(network.Cloud, "api.datakit.cloud/connectors/test"),
			valid: true,
		},
		{
			name:  "cloud empty hostname",
			addr:  network.Addr(network.Cloud, ""),
			valid: false,
		},
		{
			name:  "valid tcp address",
			addr:  network.Addr(network.TCP, "127.0.0.1:8080"),
			valid: true,
		},
		{
			name:  "tcp missing port",
			addr:  network.Addr(network.TCP, "localhost"),
			valid: false,
		},
		{
			name:  "tcp empty",
			addr:  network.Addr(network.TCP, ""),
			valid: false,
		},
		{
			name:  "valid unix socket",
			addr:  network.Addr(network.Socket, "/tmp/test.sock"),
			valid: true,
		},
		{
			name:  "unix relative path",
			addr:  network.Addr(network.Socket, "relative/path.sock"),
			valid: false,
		},
		{
			name:  "unix empty",
			addr:  network.Addr(network.Socket, ""),
			valid: false,
		},
		{
			name:  "invalid network type",
			addr:  network.Address{"invalid", "address"},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.addr.IsValid()
			if (err == nil) != tt.valid {
				if tt.valid {
					t.Errorf("expected valid, got error: %v", err)
				} else {
					t.Errorf("expected invalid, but was valid")
				}
			}
		})
	}
}

func TestAddress_URL(t *testing.T) {
	tests := []struct {
		name        string
		addr        network.Address
		expectedURI string
	}{
		{
			name:        "cloud basic",
			addr:        network.Addr(network.Cloud, "api.datakit.cloud"),
			expectedURI: "https://api.datakit.cloud",
		},
		{
			name:        "cloud with path",
			addr:        network.Addr(network.Cloud, "api.datakit.cloud/connectors/test"),
			expectedURI: "https://api.datakit.cloud",
		},
		{
			name:        "tcp address",
			addr:        network.Addr(network.TCP, "127.0.0.1:8080"),
			expectedURI: "http://127.0.0.1:8080",
		},
		{
			name:        "unix socket",
			addr:        network.Addr(network.Socket, "/tmp/test.sock"),
			expectedURI: "unix://localhost/tmp/test.sock",
		},
		{
			name:        "ssh unix socket",
			addr:        network.Addr(network.SSHSocket, "nuc.local/tmp/test.sock"),
			expectedURI: "ssh+unix://nuc.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("URI:", tt.addr.URI())
			t.Log("URL:", tt.addr.URL())
			uri := tt.addr.URL()
			if uri.String() != tt.expectedURI {
				t.Errorf("expected URL: %s, got: %s", tt.expectedURI, uri.String())
			}
		})
	}
}

func TestAddress_Resolve(t *testing.T) {
	tests := []struct {
		name      string
		addr      network.Address
		expectErr bool
		addrType  string
	}{
		{
			name:      "cloud address resolves to TCP",
			addr:      network.Addr(network.Cloud, "api.datakit.cloud"),
			expectErr: false,
			addrType:  "tcp",
		},
		{
			name:      "cloud with port",
			addr:      network.Addr(network.Cloud, "api.datakit.cloud:8443"),
			expectErr: false,
			addrType:  "tcp",
		},
		{
			name:      "cloud with path strips path",
			addr:      network.Addr(network.Cloud, "api.datakit.cloud/path"),
			expectErr: false,
			addrType:  "tcp",
		},
		{
			name:      "tcp address resolves",
			addr:      network.Addr(network.TCP, "127.0.0.1:8080"),
			expectErr: false,
			addrType:  "tcp",
		},
		{
			name:      "unix socket (nonexistent)",
			addr:      network.Addr(network.Socket, "/tmp/nonexistent.sock"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := tt.addr.Resolve()
			if (err != nil) != tt.expectErr {
				if tt.expectErr {
					t.Errorf("expected error, got nil")
				} else {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if !tt.expectErr && resolved != nil {
				if tt.addrType != "" && resolved.Network() != tt.addrType {
					t.Errorf("expected network type: %s, got: %s", tt.addrType, resolved.Network())
				}
				t.Logf("resolved %s -> %s (%s)", tt.addr, resolved, resolved.Network())
			}
		})
	}
}

func TestCloudNetwork(t *testing.T) {
	addr, err := network.Cloud.DefaultAddress()
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%s default address: %s", addr.Network(), addr.String())
	}

	if _, err = addr.Resolve(); err != nil {
		t.Fatal(err)
	}
}

func TestSocketNetwork(t *testing.T) {
	addr, err := network.Socket.DefaultAddress()
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%s default address: %s", addr.Network(), addr.String())
	}

	n, a := network.Socket, "/tmp/test.sock"
	addr = network.Addr(n, a)
	if addr.Network() != network.Socket.String() {
		t.Fatalf("expected: %s, got: %s", network.Socket, n)
	} else if addr.String() != a {
		t.Fatalf("expected: %s, got: %s", a, addr.String())
	}
}

func TestTCPNetwork(t *testing.T) {
	addr, err := network.TCP.DefaultAddress()
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%s default address: %s", addr.Network(), addr.String())
	}

	if addr.Network() != network.TCP.String() {
		t.Fatalf("expected: %s, got: %s", network.TCP, addr.Network())
	}

	host, port, err := addr.HostPort()
	if err != nil {
		t.Fatal(err)
	} else if addr.String() != net.JoinHostPort(host, port) {
		t.Fatalf("expected: %s, got: %s", host+":"+port, addr.String())
	}
}
