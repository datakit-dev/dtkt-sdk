package v1beta1

import (
	context "context"

	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
)

type (
	pkgCtxKey struct{}
	regCtxKey struct{}
)

func AddPackageToContext(ctx context.Context, pkg *sharedv1beta1.Package) context.Context {
	return context.WithValue(ctx, pkgCtxKey{}, pkg)
}

func PackageFromContext(ctx context.Context) (*sharedv1beta1.Package, bool) {
	pkg, ok := ctx.Value(pkgCtxKey{}).(*sharedv1beta1.Package)
	return pkg, ok
}

func AddRegistryToContext(ctx context.Context, reg *TypeRegistry) context.Context {
	return context.WithValue(ctx, regCtxKey{}, reg)
}

func RegistryFromContext(ctx context.Context) (*TypeRegistry, bool) {
	reg, ok := ctx.Value(regCtxKey{}).(*TypeRegistry)
	return reg, ok
}
