package common

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"text/template"

	"google.golang.org/protobuf/reflect/protoreflect"
)

const ProtoGoPkgBase = "github.com/datakit-dev/dtkt-sdk/sdk-go/proto"

type (
	ProtoPackage  protoreflect.FullName
	ProtoPackages map[ProtoPackage]ProtoFilesHelper
)

func (p ProtoPackage) String() string {
	return string(p)
}

func (p ProtoPackage) HasModuleAndVersion() (module, version string, ok bool) {
	module = p.Module()
	version = p.Version()
	ok = module != "" && version != ""
	return
}

func (p ProtoPackage) Parts() []string {
	return strings.SplitN(strings.ReplaceAll(p.String(), ".", "/"), "/", 3)
}

func (p ProtoPackage) Module() string {
	return p.Parts()[1]
}

func (p ProtoPackage) Version() string {
	return p.Parts()[2]
}

func (p ProtoPackage) GoPkgName() string {
	return strings.Join(p.Parts()[1:], "")
}

func (p ProtoPackage) GoPkgPath() string {
	return path.Join(ProtoGoPkgBase, strings.ReplaceAll(p.String(), ".", "/"))
}

func (p ProtoPackage) GoImport() string {
	return fmt.Sprintf(`%s "%s"`, p.GoPkgName(), p.GoPkgPath())
}

func (p ProtoPackages) TmplFuncMap() template.FuncMap {
	return template.FuncMap{
		"goImport": func(pkg ProtoPackage) string {
			return pkg.GoImport()
		},
		"goPkg": func(pkg ProtoPackage) string {
			return pkg.GoPkgName()
		},
		"svcMethodMap": func() map[ProtoPackage]map[string][]ProtoMethod {
			return p.Files().ServiceMethodMap()
		},
		"svcMethods": func(pkg ProtoPackage, svcName string) []ProtoMethod {
			return p.Files().ServiceMethodMap()[pkg][svcName]
		},
		"svcNames": func(pkg ProtoPackage) []string {
			return p.ServiceNames(pkg)
		},
		"methodNames": func(pkg ProtoPackage, svcName string) []string {
			return p.MethodNames(pkg)
		},
		"msgNames": func(pkg ProtoPackage) []string {
			return p.MessageNames(pkg)
		},
		"enumNames": func(pkg ProtoPackage) []string {
			return p.EnumNames(pkg)
		},
		"getPkgBySvcName": func(svcName string) ProtoPackage {
			for pkg := range p {
				if slices.Contains(p.ServiceNames(pkg), svcName) {
					return pkg
				}
			}
			return ProtoPackage("")
		},
		"hasPkg": func(module, version string) bool {
			return p.HasPkg(module, version)
		},
	}
}

func (p ProtoPackages) Files() (all ProtoFilesHelper) {
	for _, files := range p {
		all = append(all, files...)
	}
	return
}

func (p ProtoPackages) MessageNames(pkg ProtoPackage, skip ...string) []string {
	return p[pkg].MessageNames(skip...)
}

func (p ProtoPackages) EnumNames(pkg ProtoPackage, skip ...string) []string {
	return p[pkg].EnumNames(skip...)
}

func (p ProtoPackages) ServiceMethods(pkg ProtoPackage, skip ...string) []ProtoMethod {
	return p[pkg].ServiceMethods(skip...)
}

func (p ProtoPackages) ServiceNames(pkg ProtoPackage, skip ...string) []string {
	return p[pkg].ServiceNames(skip...)
}

func (p ProtoPackages) MethodNames(pkg ProtoPackage, skip ...string) []string {
	return p[pkg].MethodNames(skip...)
}

func (p ProtoPackages) GetPkg(module, version string) ProtoPackage {
	for pkg := range p {
		if pkg.Module() == module && pkg.Version() == version {
			return pkg
		}
	}
	return ""
}

func (p ProtoPackages) HasPkg(module, version string) bool {
	for pkg := range p {
		if pkg.Module() == module && pkg.Version() == version {
			return true
		}
	}
	return false
}
