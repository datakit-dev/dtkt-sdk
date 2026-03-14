package network

import (
	"fmt"
	"os"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/env"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

func ConnectorToEnv(conn Connector) (vars []string) {
	vars = append(vars, AddrToEnv(conn.Address())...)

	if Type(conn.Address().Network()).IsCloud() {
		if conn, ok := conn.(*CloudConnector); ok {
			b, err := encoding.ToJSONV2(conn.opts.cloud)
			if err == nil && len(b) > 0 {
				vars = append(vars,
					fmt.Sprintf("%s=%s", env.CloudConfig, string(b)),
				)
			}
		}
	}

	return
}

func CloudConfigFromEnv() string {
	return os.Getenv(env.CloudConfig)
}

func DefaultNetwork() (network Type) {
	network = Type(NetworkFromEnv())
	if network.IsValid() {
		return network
	}
	return Socket
}

func DefaultAddress(network Type) (Address, error) {
	if AddressFromEnv() != "" {
		return Addr(network, AddressFromEnv()), nil
	}
	return network.DefaultAddress()
}

func ResolveConnector() (Connector, error) {
	addr, err := DefaultAddress(DefaultNetwork())
	if err != nil {
		return nil, err
	}

	if addr.Type().IsCloud() {
		if CloudConfigFromEnv() == "" {
			return nil, fmt.Errorf("invalid %q connector: missing cloud config", addr.Network())
		}

		config := &corev1.Resource{}
		if err := encoding.FromJSONV2([]byte(CloudConfigFromEnv()), config); err != nil {
			return nil, fmt.Errorf("invalid %q connector: unable to parse cloud config: %w", addr.Network(), err)
		}

		addr = AddrFromProto(config.GetContext().GetAddress())
		conn, err := NewCloudConnector(addr,
			WithCloudConfig(config),
		)
		if err != nil {
			return nil, fmt.Errorf("invalid %q connector: %w", addr.Network(), err)
		}

		return conn, nil
	}

	return NewConnector(addr)
}
