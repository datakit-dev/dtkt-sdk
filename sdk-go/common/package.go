package common

import (
	"regexp"
	"strings"
	"sync"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/invopop/jsonschema"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	PackageNamePattern    = `[a-zA-Z][a-zA-Z0-9_]+`
	PackageVersionPattern = `(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`
	PackageIconPattern    = `https:\/\/|^data:image\/[a-z]+;base64,`
)

var (
	packageNameRegex = sync.OnceValue(func() *regexp.Regexp {
		desc := packageIdentProto.ProtoReflect().Descriptor().Fields().ByName("name")
		if desc != nil {
			if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
				if proto.HasExtension(opts, validate.E_Field) {
					if rules, ok := proto.GetExtension(opts, validate.E_Field).(*validate.FieldRules); ok && rules != nil && rules.GetString() != nil && rules.GetString().GetPattern() != "" {
						return regexp.MustCompile(strings.TrimLeft(strings.TrimRight(rules.GetString().GetPattern(), "$"), "^"))
					}
				}
			}
		}
		return regexp.MustCompile("^" + PackageNamePattern + "$")
	})
	packageVersionRegex = sync.OnceValue(func() *regexp.Regexp {
		desc := packageIdentProto.ProtoReflect().Descriptor().Fields().ByName("version")
		if desc != nil {
			if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
				if proto.HasExtension(opts, validate.E_Field) {
					if rules, ok := proto.GetExtension(opts, validate.E_Field).(*validate.FieldRules); ok && rules != nil && rules.GetString() != nil && rules.GetString().GetPattern() != "" {
						return regexp.MustCompile(strings.TrimLeft(strings.TrimRight(rules.GetString().GetPattern(), "$"), "^"))
					}
				}
			}
		}
		return regexp.MustCompile("^" + PackageVersionPattern + "$")
	})
	packageIconRegex = sync.OnceValue(func() *regexp.Regexp {
		desc := packageProto.ProtoReflect().Descriptor().Fields().ByName("icon")
		if desc != nil {
			if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
				if proto.HasExtension(opts, validate.E_Field) {
					if rules, ok := proto.GetExtension(opts, validate.E_Field).(*validate.FieldRules); ok && rules != nil && rules.GetString() != nil && rules.GetString().GetPattern() != "" {
						return regexp.MustCompile(strings.TrimLeft(strings.TrimRight(rules.GetString().GetPattern(), "$"), "^"))
					}
				}
			}
		}
		return regexp.MustCompile("^" + PackageIconPattern)
	})
	packageIdentityRegex = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile("^" + packageNameRegex().String() + "@" + packageVersionRegex().String() + "$")
	})

	packageProto      *sharedv1beta1.Package
	packageIdentProto *sharedv1beta1.Package_Identity
)

type (
	PackageName                          string
	PackageIcon                          string
	PackageVersion                       string
	PackageIdentity                      string
	PackageProto[T PackageIdentityProto] interface {
		GetIdentity() T
	}
	PackageIdentityProto interface {
		GetName() string
		GetVersion() string
	}
)

func ValidPackageNameRegex() *regexp.Regexp {
	return packageNameRegex()
}

func ValidPackageVersionRegex() *regexp.Regexp {
	return packageVersionRegex()
}

func ValidPackageIconRegex() *regexp.Regexp {
	return packageIconRegex()
}

func PackageIdentityFromProto[T PackageIdentityProto](proto T) PackageIdentity {
	return PackageIdentity(proto.GetName() + "@" + proto.GetVersion())
}

func GetPackageIdentity[I PackageIdentityProto, T PackageProto[I]](proto T) PackageIdentity {
	return PackageIdentityFromProto(proto.GetIdentity())
}

func (i PackageIdentity) ToProto() *sharedv1beta1.Package_Identity {
	return &sharedv1beta1.Package_Identity{
		Name:    i.Name(),
		Version: i.Version().String(),
	}
}

func (n PackageName) String() string {
	return string(n)
}

func (i PackageIcon) String() string {
	return string(i)
}

func (v PackageVersion) String() string {
	return string(v)
}

func (v PackageVersion) Major() string {
	parts := strings.SplitN(string(v), ".", 3)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (v PackageVersion) Minor() string {
	parts := strings.SplitN(string(v), ".", 3)
	if len(parts) >= 1 {
		return parts[1]
	}
	return ""
}

func (v PackageVersion) Patch() string {
	parts := strings.SplitN(string(v), ".", 3)
	if len(parts) >= 2 {
		return parts[2]
	}
	return ""
}

func (PackageName) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Title:       "PackageName",
		Description: "Valid package name.",
		Pattern:     packageNameRegex().String(),
	}
}

func (PackageIcon) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Title:       "PackageIcon",
		Description: "Valid package image.",
		Pattern:     packageIconRegex().String(),
	}
}

func (PackageVersion) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Title:       "PackageVersion",
		Description: "Valid package semantic version.",
		Pattern:     packageVersionRegex().String(),
	}
}

func (PackageIdentity) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Title:       "PackageIdentity",
		Description: "Valid package identity, e.g.: PACKAGE_NAME[@latest|PACKAGE_VERSION]",
		Pattern:     packageIdentityRegex().String(),
	}
}

func (i PackageIdentity) String() string {
	return string(i)
}

func (i PackageIdentity) Name() string {
	parts := strings.SplitN(string(i), "@", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return string(i)
}

func (i PackageIdentity) Version() PackageVersion {
	parts := strings.Split(string(i), "@")
	if len(parts) >= 2 {
		return PackageVersion(parts[1])
	}
	return ""
}

func (i PackageIdentity) Slug() string {
	return strings.ToLower(i.Name())
}

func (i PackageIdentity) ProtoPackage(prefix ...string) string {
	prefix = util.SliceReduce(prefix, func(p string) (string, bool) {
		p = util.SlugifyWithSeparator('_', p)
		return p, len(p) > 0
	})
	return strings.Join(append(prefix, i.ProtoName(), i.ProtoVersion()), ".")
}

func (i PackageIdentity) ProtoName() string {
	return util.SlugifyWithSeparator('_', strings.ToLower(i.Name()))
}

func (i PackageIdentity) ProtoVersion() string {
	if i.Version() == "" || i.Version().Major() == "" || i.Version().Major() == "0" {
		return "v1beta"
	}
	return "v" + i.Version().Major()
}

func (i PackageIdentity) DockerImageName() string {
	return "intgr-" + i.Slug()
}

func (i PackageIdentity) DockerImageVersion() string {
	return "v" + i.Version().String()
}

func (i PackageIdentity) DockerImageRef() string {
	return i.DockerImageName() + ":" + i.DockerImageVersion()
}

func (i PackageIdentity) IsValid() bool {
	return packageIdentityRegex().MatchString(string(i))
}
