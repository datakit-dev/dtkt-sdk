package env

import (
	"context"
	"encoding/json"
	"expvar"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/version"
)

const (
	Address        = "DTKT_ADDRESS"
	AppEnv         = "DTKT_APP_ENV"
	CloudConfig    = "DTKT_CLOUD_CONFIG"
	ContextAddress = "DTKT_CONTEXT_ADDRESS"
	ContextName    = "DTKT_CONTEXT_NAME"
	DataRoot       = "DTKT_DATA_ROOT"
	DebugPath      = "/debug/info"
	DeployName     = "DTKT_DEPLOY_NAME"
	LogFormat      = "DTKT_LOG_FORMAT"
	LogLevel       = "DTKT_LOG_LEVEL"
	LogSource      = "DTKT_LOG_SOURCE"
	Network        = "DTKT_NETWORK"
)

var (
	getVars = sync.OnceValue(func() Vars {
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
		}
	})
	getHandler = sync.OnceValue(func() http.Handler {
		expvar.Publish("envvars", getVars())
		expvar.Publish("version", version.GetVersionInfo())
		expvar.NewInt("numcpu").Set(int64(runtime.NumCPU()))
		return expvar.Handler()
	})
)

type (
	DebugInfo struct {
		Vars     Vars                `json:"envvars"`
		MemStats runtime.MemStats    `json:"memstats"`
		NumCPU   int                 `json:"numcpu"`
		Version  version.VersionInfo `json:"version"`
	}
	Vars map[string]string
)

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

func Handler() (string, http.Handler) {
	return DebugPath, getHandler()
}

func FetchDebugInfo(ctx context.Context, baseUrl *url.URL) (*DebugInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseUrl.JoinPath(DebugPath).String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	//nolint:errcheck
	defer resp.Body.Close()

	var info DebugInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}
