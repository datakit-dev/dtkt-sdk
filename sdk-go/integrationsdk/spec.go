package integrationsdk

import (
	"fmt"
	"io"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/invopop/jsonschema"
)

const (
	SpecAPIVersion     = api.V1Beta1
	SpecKind           = "Package"
	SpecSchemaID       = "dtkt.integrationsdk.v1beta1.Package"
	SpecSchemaFilename = SpecSchemaID + v1beta1.JSONSchemaFileExt
	SpecFile           = "package.dtkt.yaml"
)

type Spec struct {
	pkg *sharedv1beta1.Package
	raw []byte
}

func SpecLoader(opts ...api.SpecLoaderOpt[*Spec]) *api.SpecLoader[*Spec] {
	return api.NewLoader(new(Spec), opts...)
}

func NewSpec(pkgType sharedv1beta1.PackageType, pkgIdent *sharedv1beta1.Package_Identity, icon string) *Spec {
	return &Spec{
		pkg: &sharedv1beta1.Package{
			Type:     pkgType,
			Identity: pkgIdent,
			Icon:     icon,
		},
	}
}

func NewSpecWithPackage(pkg *sharedv1beta1.Package) *Spec {
	return &Spec{
		pkg: pkg,
	}
}

func ReadSpec(format encoding.Format, reader io.Reader) (*Spec, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	spec, err := SpecLoader().Decode(format, raw)
	if err != nil {
		return nil, err
	}

	spec.raw = raw

	return spec, nil
}

func WriteSpec(spec *Spec, format encoding.Format, writer io.Writer) (int, error) {
	b, err := SpecLoader().Encode(format, spec)
	if err != nil {
		return 0, err
	}

	n, err := writer.Write(b)
	if err != nil {
		return 0, err
	}

	spec.raw = b

	return n, nil
}

func (s *Spec) GetRaw() []byte {
	return s.raw
}

func (s *Spec) GetPackage() *sharedv1beta1.Package {
	return s.pkg
}

func (s *Spec) GetIdentity() common.PackageIdentity {
	return common.GetPackageIdentity(s.pkg)
}

func (s *Spec) Validate() error {
	if s == nil {
		return fmt.Errorf("invalid spec: failed to load")
	} else if s.pkg == nil {
		return fmt.Errorf("invalid spec: missing package")
	}
	return protovalidate.Validate(s.pkg)
}

func (*Spec) APIVersion() api.Version {
	return SpecAPIVersion
}

func (*Spec) SpecKind() string {
	return SpecKind
}

func (*Spec) SpecID() string {
	return SpecSchemaID
}

func (Spec) Filename() string {
	return SpecSchemaFilename
}

func (s *Spec) UnmarshalJSON(data []byte) error {
	pkg := new(sharedv1beta1.Package)
	if err := encoding.FromJSONV2(data, pkg); err != nil {
		return err
	}

	if pkg.Identity == nil {
		pkg.Identity = &sharedv1beta1.Package_Identity{}
	}

	*s = Spec{
		pkg: pkg,
	}

	return nil
}

func (s Spec) MarshalJSON() ([]byte, error) {
	return encoding.ToJSONV2(s.pkg)
}

func (Spec) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Ref: "dtkt.shared.v1beta1.Package.jsonschema.json",
	}
}
