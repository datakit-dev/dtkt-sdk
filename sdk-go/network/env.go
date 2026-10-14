package network

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"maps"
	"net"
	"os"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/env"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	"google.golang.org/protobuf/proto"
)

func AddressToEnv(addr net.Addr) map[string]string {
	return map[string]string{
		env.Network: addr.Network(),
		env.Address: addr.String(),
	}
}

func AddressFromEnv() string {
	return os.Getenv(env.Address)
}

func TypeToEnv(typ Type) map[string]string {
	return map[string]string{
		env.Network: typ.String(),
	}
}

func TypeFromEnv() string {
	return os.Getenv(env.Network)
}

func DefaultType() (typ Type) {
	typ = Type(TypeFromEnv())
	if typ.IsValid() {
		return typ
	}
	return Socket
}

func DefaultAddress(network Type) (Address, error) {
	if AddressFromEnv() != "" {
		return Addr(network, AddressFromEnv()), nil
	}
	return network.DefaultAddress()
}

func ConnectorToEnv(conn Connector) (vars map[string]string, err error) {
	vars = map[string]string{}
	if conn.Address().IsCloud() {
		if conn, ok := conn.(*CloudConnector); ok {
			b, err := proto.Marshal(conn.resource)
			if err != nil {
				return nil, err
			}

			vars[env.CloudResource] = base64.StdEncoding.EncodeToString(b)
		}
	}

	maps.Copy(vars, AddressToEnv(conn.Address()))

	return
}

func ConnectorFromEnv(addr Address) (Connector, error) {
	var opts []Option
	tlsConfig, err := TLSConfigFromEnv()
	if err != nil {
		return nil, err
	} else if tlsConfig != nil {
		opts = append(opts, WithTLSConfig(tlsConfig))
	}

	if addr.IsCloud() {
		if os.Getenv(env.CloudResource) == "" {
			return nil, fmt.Errorf("invalid %q connector: env missing %s", addr.Network(), env.CloudResource)
		}

		b, err := base64.StdEncoding.DecodeString(env.GetVar(env.CloudResource))
		if err != nil {
			return nil, fmt.Errorf("invalid %q connector: decode resource: %w", addr.Network(), err)
		}

		resource := new(corev1.Resource)
		err = proto.Unmarshal(b, resource)
		if err != nil {
			return nil, fmt.Errorf("invalid %q connector: unmarshal resource: %w", addr.Network(), err)
		}

		opts = append(opts,
			WithCloudResource(resource),
		)

		addr = AddrFromProto(resource.GetContext().GetAddress())
	}

	return NewConnector(addr, opts...)
}

func TLSConfigToEnv(caCertPEM string, bundle *corev1.TLSCertBundle) (_ map[string]string, err error) {
	if caCertPEM == "" || bundle == nil {
		return
	}

	b, err := proto.Marshal(bundle)
	if err != nil {
		return
	}

	return map[string]string{
		env.TLSCACert: caCertPEM,
		env.TLSBundle: base64.StdEncoding.EncodeToString(b),
	}, nil
}

func TLSConfigFromEnv() (_ *tls.Config, err error) {
	if env.GetVar(env.TLSCACert) == "" || env.GetVar(env.TLSBundle) == "" {
		return
	}

	b, err := base64.StdEncoding.DecodeString(env.GetVar(env.TLSBundle))
	if err != nil {
		return
	}

	bundle := new(corev1.TLSCertBundle)
	err = proto.Unmarshal(b, bundle)
	if err != nil {
		return
	}

	return TLSServerConfig(env.GetVar(env.TLSCACert), bundle)
}
