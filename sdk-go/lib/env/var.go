package env

import (
	"encoding/json"
	"os"
	"sync"
)

const (
	Address        = "DTKT_ADDRESS"
	AppEnv         = "DTKT_APP_ENV"
	CloudConfig    = "DTKT_CLOUD_CONFIG"
	ContextAddress = "DTKT_CONTEXT_ADDRESS"
	ContextName    = "DTKT_CONTEXT_NAME"
	DataRoot       = "DTKT_DATA_ROOT"
	DeployName     = "DTKT_DEPLOY_NAME"
	LogFormat      = "DTKT_LOG_FORMAT"
	LogLevel       = "DTKT_LOG_LEVEL"
	LogSource      = "DTKT_LOG_SOURCE"
	Network        = "DTKT_NETWORK"
	PIDFile        = "DTKT_PID_FILE"
)

// getVars
var getVars = sync.OnceValue(func() Vars {
	return Vars{
		Address:        os.Getenv(Address),
		AppEnv:         os.Getenv(AppEnv),
		CloudConfig:    os.Getenv(CloudConfig),
		ContextAddress: os.Getenv(ContextAddress),
		ContextName:    os.Getenv(ContextName),
		DataRoot:       os.Getenv(DataRoot),
		DeployName:     os.Getenv(DeployName),
		LogFormat:      os.Getenv(LogFormat),
		LogLevel:       os.Getenv(LogLevel),
		LogSource:      os.Getenv(LogSource),
		Network:        os.Getenv(Network),
		PIDFile:        os.Getenv(PIDFile),
	}
})

type Vars map[string]string

func GetVar(key string) string {
	return GetVars()[key]
}

func GetVars() Vars {
	return getVars()
}

func (v Vars) String() string {
	j, _ := json.Marshal(v)
	return string(j)
}
