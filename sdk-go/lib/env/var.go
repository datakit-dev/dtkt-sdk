package env

import (
	"encoding/json"
	"os"
	"sync"
)

const (
	Address       = "DTKT_ADDRESS"
	AppEnv        = "DTKT_APP_ENV"
	CloudResource = "DTKT_CLOUD_RESOURCE"
	Context       = "DTKT_CONTEXT"
	DataRoot      = "DTKT_DATA_ROOT"
	DeployName    = "DTKT_DEPLOY_NAME"
	LogFormat     = "DTKT_LOG_FORMAT"
	LogLevel      = "DTKT_LOG_LEVEL"
	LogSource     = "DTKT_LOG_SOURCE"
	NATSAddress   = "DTKT_NATS_ADDRESS"
	Network       = "DTKT_NETWORK"
	PIDFile       = "DTKT_PID_FILE"
	TLSCACert     = "DTKT_TLS_CA_CERT"
	TLSBundle     = "DTKT_TLS_BUNDLE"
)

var getVars = sync.OnceValue(func() Vars {
	return Vars{
		Address:       os.Getenv(Address),
		AppEnv:        os.Getenv(AppEnv),
		CloudResource: os.Getenv(CloudResource),
		Context:       os.Getenv(Context),
		DataRoot:      os.Getenv(DataRoot),
		DeployName:    os.Getenv(DeployName),
		LogFormat:     os.Getenv(LogFormat),
		LogLevel:      os.Getenv(LogLevel),
		LogSource:     os.Getenv(LogSource),
		NATSAddress:   os.Getenv(NATSAddress),
		Network:       os.Getenv(Network),
		PIDFile:       os.Getenv(PIDFile),
		TLSCACert:     os.Getenv(TLSCACert),
		TLSBundle:     os.Getenv(TLSBundle),
	}
})

type Vars map[string]string

func GetVar(key string) string {
	return getVars().Get(key)
}

func GetVars() Vars {
	return getVars()
}

func (v Vars) Get(key string) string {
	return v[key]
}

func (v Vars) String() string {
	j, _ := json.Marshal(v)
	return string(j)
}
