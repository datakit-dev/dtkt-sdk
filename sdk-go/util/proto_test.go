package util_test

import (
	"testing"

	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

func TestProtoDiff(t *testing.T) {
	pkgA, pkgB := new(sharedv1beta1.Package), new(sharedv1beta1.Package)
	mask := util.ProtoDiff(pkgA, pkgB)
	if len(mask.GetPaths()) != 0 {
		t.Fatalf("expected no difference")
	}

	pkgA.Identity = &sharedv1beta1.Package_Identity{
		Name:    "FooBar",
		Version: "0.1.0",
	}
	pkgB.Identity = &sharedv1beta1.Package_Identity{
		Name:    "FooBar",
		Version: "0.2.0",
	}

	mask = util.ProtoDiff(pkgA, pkgB)
	if len(mask.GetPaths()) != 1 {
		t.Fatalf("expected 1 difference, got: %d", len(mask.GetPaths()))
	} else {
		t.Logf("diff: %v", mask.Paths)
	}
}
