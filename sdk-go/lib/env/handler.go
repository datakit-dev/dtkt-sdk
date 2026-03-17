package env

import (
	"context"
	"encoding/json"
	"expvar"
	"net/http"
	"net/url"
	"runtime"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/version"
)

const DebugPath = "/debug/info"

var getHandler = sync.OnceValue(func() http.Handler {
	expvar.Publish("envvars", getVars())
	expvar.Publish("version", version.GetVersionInfo())
	expvar.NewInt("numcpu").Set(int64(runtime.NumCPU()))
	return expvar.Handler()
})

type DebugInfo struct {
	Vars     Vars                `json:"envvars"`
	MemStats runtime.MemStats    `json:"memstats"`
	NumCPU   int                 `json:"numcpu"`
	Version  version.VersionInfo `json:"version"`
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
