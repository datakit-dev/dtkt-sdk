package common_test

import (
	"path"
	"strings"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
)

var packages = []common.ProtoPackage{
	"dtkt.ai.v1beta1",
	"dtkt.base.v1beta1",
	"dtkt.blob.v1beta1",
	"dtkt.catalog.v1beta1",
	"dtkt.email.v1beta1",
	"dtkt.event.v1beta1",
	"dtkt.geo.v1beta1",
	"dtkt.geojson.v1beta1",
	"dtkt.replication.v1beta1",
	"dtkt.shared.v1beta1",
}

func TestProtoPackage(t *testing.T) {
	for _, pkg := range packages {
		parts := strings.Split(pkg.String(), ".")
		if pkg.Module() != parts[1] {
			t.Fatalf("expected: %s, got: %s", parts[1], pkg.Module())
		}

		var goPkgName = strings.Join(parts[1:], "")
		if pkg.GoPkgName() != goPkgName {
			t.Fatalf("expected: %s, got: %s", goPkgName, pkg.GoPkgName())
		}

		goPkgPath := path.Join(append([]string{common.ProtoGoPkgBase}, parts...)...)
		parts = append(parts, goPkgName+"connect")

		if pkg.GoPkgPath() != goPkgPath {
			t.Fatalf("expected: %s, got: %s", goPkgPath, pkg.GoPkgPath())
		}

		if common.IsWellKnownName(pkg) {
			continue
		}

		if pkg.Version() != parts[2] {
			t.Fatalf("expected: %s, got: %s", parts[2], pkg.Version())
		}
	}
}
