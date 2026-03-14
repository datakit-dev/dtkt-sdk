package version

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

type VersionInfo struct {
	GoVersion    string `json:"goVersion" yaml:"goVersion"`
	Module       string `json:"module" yaml:"module"`
	Version      string `json:"version" yaml:"version"`
	GitCommit    string `json:"gitCommit" yaml:"gitCommit"`
	GitTreeState string `json:"gitTreeState" yaml:"gitTreeState"` // "clean" or "dirty"
	BuildTime    string `json:"buildTime" yaml:"buildTime"`
	Platform     string `json:"platform" yaml:"platform"`
	Checksum     string `json:"checksum" yaml:"checksum"`
}

func GetVersionInfo() VersionInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return VersionInfo{GoVersion: "unknown"}
	}

	vi := VersionInfo{
		GoVersion: info.GoVersion,
		Module:    info.Main.Path,
		Version:   info.Main.Version,
	}

	// Extract settings
	var goos, goarch string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "GOOS":
			goos = setting.Value
		case "GOARCH":
			goarch = setting.Value
		case "vcs.revision":
			vi.GitCommit = setting.Value
		case "vcs.time":
			vi.BuildTime = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				vi.GitTreeState = "dirty"
			} else {
				vi.GitTreeState = "clean"
			}
		}
	}

	// Combine GOOS and GOARCH into Platform
	vi.Platform = fmt.Sprintf("%s/%s", goos, goarch)

	bin, err := os.Executable()
	if err != nil {
		vi.Checksum = "<unknown>"
	}

	f, err := os.Open(bin)
	if err != nil {
		vi.Checksum = "<unknown>"
	}

	//nolint:errcheck
	defer f.Close()

	checksum, err := util.HashSHA256Reader(f)
	if err != nil {
		vi.Checksum = "<unknown>"
	}

	vi.Checksum = checksum

	return vi
}

func (v VersionInfo) String() string {
	b, err := json.Marshal(v)
	if err != nil {
		return "<invalid version info>"
	}
	return string(b)
}
