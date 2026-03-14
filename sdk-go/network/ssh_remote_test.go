package network

import (
	"testing"
)

func TestParseSSHRemoteURI(t *testing.T) {
	tests := []struct {
		name         string
		uri          string
		wantErr      bool
		wantUser     string
		wantHost     string
		wantPort     int
		wantRemote   string
		wantNetwork  string
		wantIdentity string
		wantConfig   string
		wantPass     string
		wantVerify   HostKeyVerification
	}{
		{
			name:        "basic ssh+unix URI",
			uri:         "ssh+unix://example.com/var/run/service.sock",
			wantErr:     false,
			wantHost:    "example.com",
			wantPort:    22,
			wantRemote:  "/var/run/service.sock",
			wantNetwork: "unix",
			wantVerify:  VerifyKnownHosts,
		},
		{
			name:        "basic ssh+tcp URI",
			uri:         "ssh+tcp://example.com/localhost:8080",
			wantErr:     false,
			wantHost:    "example.com",
			wantPort:    22,
			wantRemote:  "localhost:8080",
			wantNetwork: "tcp",
			wantVerify:  VerifyKnownHosts,
		},
		{
			name:        "ssh+tcp to remote host",
			uri:         "ssh+tcp://bastion.example.com/internal-db:5432",
			wantErr:     false,
			wantHost:    "bastion.example.com",
			wantPort:    22,
			wantRemote:  "internal-db:5432",
			wantNetwork: "tcp",
			wantVerify:  VerifyKnownHosts,
		},
		{
			name:        "with user",
			uri:         "ssh+unix://john@example.com/var/run/service.sock",
			wantErr:     false,
			wantUser:    "john",
			wantHost:    "example.com",
			wantPort:    22,
			wantRemote:  "/var/run/service.sock",
			wantNetwork: "unix",
		},
		{
			name:        "ssh+tcp with user and port",
			uri:         "ssh+tcp://john@example.com:2222/app:9090",
			wantErr:     false,
			wantUser:    "john",
			wantHost:    "example.com",
			wantPort:    2222,
			wantRemote:  "app:9090",
			wantNetwork: "tcp",
		},
		{
			name:        "with password",
			uri:         "ssh+unix://john:secret@example.com/var/run/service.sock",
			wantErr:     false,
			wantUser:    "john",
			wantPass:    "secret",
			wantHost:    "example.com",
			wantPort:    22,
			wantRemote:  "/var/run/service.sock",
			wantNetwork: "unix",
		},
		{
			name:         "with identity file",
			uri:          "ssh+unix://example.com/var/run/service.sock?identity=/path/to/key",
			wantErr:      false,
			wantHost:     "example.com",
			wantPort:     22,
			wantRemote:   "/var/run/service.sock",
			wantNetwork:  "unix",
			wantIdentity: "/path/to/key",
		},
		{
			name:        "with TOFU verification",
			uri:         "ssh+tcp://example.com/localhost:8080?host_key_verify=tofu",
			wantErr:     false,
			wantHost:    "example.com",
			wantPort:    22,
			wantRemote:  "localhost:8080",
			wantNetwork: "tcp",
			wantVerify:  VerifyTOFU,
		},
		{
			name:        "with fingerprint verification",
			uri:         "ssh+unix://example.com/var/run/service.sock?fingerprint=SHA256:xxxxx",
			wantErr:     false,
			wantHost:    "example.com",
			wantPort:    22,
			wantRemote:  "/var/run/service.sock",
			wantNetwork: "unix",
			wantVerify:  VerifyFingerprint,
		},
		{
			name:        "complex ssh+tcp URI",
			uri:         "ssh+tcp://admin:pass123@10.0.0.1:2222/db.internal:5432?identity=/keys/prod.pem&host_key_verify=tofu",
			wantErr:     false,
			wantUser:    "admin",
			wantPass:    "pass123",
			wantHost:    "10.0.0.1",
			wantPort:    2222,
			wantRemote:  "db.internal:5432",
			wantNetwork: "tcp",
			wantVerify:  VerifyTOFU,
		},
		{
			name:    "invalid scheme",
			uri:     "http://example.com/socket",
			wantErr: true,
		},
		{
			name:    "missing host",
			uri:     "ssh+unix:///var/run/service.sock",
			wantErr: true,
		},
		{
			name:    "missing remote address",
			uri:     "ssh+unix://example.com",
			wantErr: true,
		},
		{
			name:    "ssh+tcp missing remote address",
			uri:     "ssh+tcp://example.com/",
			wantErr: true,
		},
		{
			name:    "invalid port",
			uri:     "ssh+unix://example.com:abc/var/run/service.sock",
			wantErr: true,
		},
		{
			name:    "invalid verification mode",
			uri:     "ssh+unix://example.com/var/run/service.sock?host_key_verify=invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSSHRemoteURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSSHRemoteURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.wantUser != "" && got.User != tt.wantUser {
				t.Errorf("ParseSSHRemoteURI() User = %v, want %v", got.User, tt.wantUser)
			}
			if got.Host != tt.wantHost {
				t.Errorf("ParseSSHRemoteURI() Host = %v, want %v", got.Host, tt.wantHost)
			}
			// Check resolved port using getPort() instead of Port field directly
			if gotPort := got.getPort(); gotPort != tt.wantPort {
				t.Errorf("ParseSSHRemoteURI() Port = %v, want %v", gotPort, tt.wantPort)
			}
			if tt.wantRemote != "" && got.RemoteAddr.String() != tt.wantRemote {
				t.Errorf("ParseSSHRemoteURI() RemoteAddr = %v, want %v", got.RemoteAddr.String(), tt.wantRemote)
			}
			if tt.wantNetwork != "" && got.RemoteAddr.Network() != tt.wantNetwork {
				t.Errorf("ParseSSHRemoteURI() RemoteAddr.Network() = %v, want %v", got.RemoteAddr.Network(), tt.wantNetwork)
			}
			if tt.wantIdentity != "" && (len(got.IdentityFiles) == 0 || got.IdentityFiles[0] != tt.wantIdentity) {
				t.Errorf("ParseSSHRemoteURI() IdentityFiles = %v, want %v", got.IdentityFiles, tt.wantIdentity)
			}
			if tt.wantConfig != "" && got.SSHConfigFile != tt.wantConfig {
				t.Errorf("ParseSSHRemoteURI() SSHConfigFile = %v, want %v", got.SSHConfigFile, tt.wantConfig)
			}
			if tt.wantPass != "" && got.Password != tt.wantPass {
				t.Errorf("ParseSSHRemoteURI() Password = %v, want %v", got.Password, tt.wantPass)
			}
			if tt.wantVerify != "" && got.HostKeyVerify != tt.wantVerify {
				t.Errorf("ParseSSHRemoteURI() HostKeyVerify = %v, want %v", got.HostKeyVerify, tt.wantVerify)
			}
		})
	}
}

func TestSSHRemoteURI_SSHEndpoint(t *testing.T) {
	uri := &SSHRemoteURI{
		Host: "example.com",
		Port: 2222,
	}
	got := uri.SSHEndpoint()
	want := "example.com:2222"
	if got != want {
		t.Errorf("SSHEndpoint() = %v, want %v", got, want)
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "absolute path",
			path:    "/etc/ssh/ssh_config",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "config/ssh_config",
			wantErr: false,
		},
		{
			name:    "tilde path",
			path:    "~/.ssh/config",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.path != "" && got == "" {
				t.Errorf("ExpandPath() returned empty path for non-empty input")
			}
		})
	}
}
