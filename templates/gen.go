package templates

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	actionv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/action/v1beta1"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
	eventv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/event/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

//go:embed go/*
var pkgTemplates embed.FS

var (
	skipDockerFiles = map[sharedv1beta1.PackageType][]string{
		sharedv1beta1.PackageType_PACKAGE_TYPE_GO: {
			"Dockerfile",
			"skaffold.yaml",
			"infra/*",
		},
	}
	skipLibFiles = map[sharedv1beta1.PackageType][]string{
		sharedv1beta1.PackageType_PACKAGE_TYPE_GO: {
			"main.go",
		},
	}
	skipProtoFiles = map[sharedv1beta1.PackageType][]string{
		sharedv1beta1.PackageType_PACKAGE_TYPE_GO: {
			"proto/*",
			"buf.*",
			"pkg/greet_service.go",
			"examples/greet-*",
		},
	}
	skipActionFiles = map[sharedv1beta1.PackageType][]string{
		sharedv1beta1.PackageType_PACKAGE_TYPE_GO: {
			"pkg/action.go",
			"examples/echo-*",
		},
	}
)

type GeneratePackageConfig struct {
	Spec     *integrationsdk.Spec
	Services []string
	Tmpls    []fs.DirEntry
	Path     string
	Lib      bool
	Proto    bool
}

func GeneratePackage(config *GeneratePackageConfig) error {
	if err := config.Spec.Validate(); err != nil {
		return err
	}

	slices.Sort(config.Services)

	ident := common.GetPackageIdentity(config.Spec.GetPackage())
	svcPkgs := common.ProtoPackages{}
	for _, svcName := range config.Services {
		svcName := common.ProtoService(svcName)
		if api.VersionContainsName(config.Spec.APIVersion(), svcName) {
			svcPkg := svcName.ProtoPkg()
			config.Spec.APIVersion().RangeFilesByPackage(protoreflect.FullName(svcPkg), func(fd protoreflect.FileDescriptor) bool {
				if !slices.Contains(svcPkgs[svcPkg], fd) {
					svcPkgs[svcPkg] = append(svcPkgs[svcPkg], fd)
				}
				return true
			})
		} else {
			return fmt.Errorf(`invalid service "%s": descriptor not found`, svcName.ProtoPkg())
		}
	}

	if config.Path == "." {
		// Check if we're in dtkt-integrations root
		var isIntegrations bool
		if _, err := os.Stat(".git/config"); err == nil {
			gitConfig, err := os.ReadFile(".git/config")
			if err == nil {
				if match, err := regexp.Match("git@github.com:datakit-dev/dtkt-integrations.git", gitConfig); err == nil && match {
					if fs, err := os.Stat("packages"); err == nil && fs.IsDir() {
						isIntegrations = true
					}
				}
			}
		}

		if isIntegrations {
			config.Path = "./packages/" + ident.Slug()
		} else {
			config.Path = "./" + ident.Slug()
		}
	}

	err := os.MkdirAll(config.Path, 0755)
	if err != nil {
		return fmt.Errorf("failed to create package directory: %v", err)
	}

	fmt.Printf("Generating %s %s package in %s\n", ident.Slug(), config.Spec.GetPackage().GetType(), config.Path)

	pkgFile, err := os.Create(config.SpecPath())
	if err != nil {
		return err
	}
	defer pkgFile.Close()

	_, err = integrationsdk.WriteSpec(config.Spec, encoding.YAML, pkgFile)
	if err != nil {
		return err
	}

	pkgTmpl := strings.ToLower(config.Type())
	protoPkg := ident.ProtoPackage("integration")
	protoPath := path.Join(strings.Split(protoPkg, ".")...)
	protoImport := strings.Join(strings.Split(ident.ProtoPackage(), "."), "")

	err = fs.WalkDir(pkgTemplates, pkgTmpl, func(readPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		fmt.Println(readPath)

		writePath := config.Path + "/" + strings.TrimSuffix(strings.TrimPrefix(readPath, pkgTmpl+"/"), ".tmpl")

		if !d.IsDir() && !config.SkipFile(writePath, &svcPkgs) {
			readFile, err := pkgTemplates.ReadFile(readPath)
			if err != nil {
				return fmt.Errorf("failed to read template %s: %w", readPath, err)
			}

			if strings.HasSuffix(readPath, ".tmpl") {
				tmpl, err := template.New(readPath).
					Funcs(svcPkgs.TmplFuncMap()).
					Parse(string(readFile))
				if err != nil {
					return fmt.Errorf("failed to parse template %s: %w", readPath, err)
				}

				data := map[string]any{
					"API_SERVICES": svcPkgs,

					"PKG_NAME":       ident.Name(),
					"PKG_VERSION":    ident.Version(),
					"PKG_SLUG":       ident.Slug(),
					"PKG_IMAGE_NAME": ident.DockerImageName(),

					"PROTO_ENABLED": config.Proto,
					"PROTO_PKG":     protoPkg,
					"PROTO_PATH":    protoPath,
					"PROTO_IMPORT":  protoImport,
				}

				filenameTmpl, err := template.New("filename").Parse(writePath)
				if err != nil {
					return fmt.Errorf("failed to parse filename template %s: %w", readPath, err)
				}

				var writePathBuf bytes.Buffer
				if err := filenameTmpl.Execute(&writePathBuf, data); err != nil {
					return fmt.Errorf("failed to execute filename template %s: %w", readPath, err)
				}

				writePath = writePathBuf.String()

				switch filepath.Base(writePath) {
				case "service.go":
					for _, fullName := range config.Services {
						if fullName == basev1beta1.BaseService_ServiceDesc.ServiceName || fullName == actionv1beta1.ActionService_ServiceDesc.ServiceName || fullName == eventv1beta1.EventService_ServiceDesc.ServiceName {
							continue
						}

						var (
							parts     = strings.Split(fullName, ".")
							shortName = parts[len(parts)-1]
							fileName  = util.ToSnakeCase(shortName)
							path      = strings.ReplaceAll(writePath, "service.go", fileName+".go")
						)

						data["SVC_NAME"] = shortName

						if err = WriteTemplate(tmpl, path, data); err != nil {
							return fmt.Errorf("failed to write file %s: %w", path, err)
						}
					}
				case "action.go":
					if slices.Contains(config.Services, actionv1beta1.ActionService_ServiceDesc.ServiceName) {
						if err := WriteTemplate(tmpl, writePath, data); err != nil {
							return fmt.Errorf("failed to write file %s: %w", writePath, err)
						}
					}
				case "event.go":
					if slices.Contains(config.Services, eventv1beta1.EventService_ServiceDesc.ServiceName) {
						if err := WriteTemplate(tmpl, writePath, data); err != nil {
							return fmt.Errorf("failed to write file %s: %w", writePath, err)
						}
					}
				case "config.proto", "greet_service.proto":
					protoDir, protoFile := filepath.Split(writePath)
					writePath = filepath.Join(protoDir, protoPath, protoFile)

					if err := WriteTemplate(tmpl, writePath, data); err != nil {
						return fmt.Errorf("failed to write file %s: %w", writePath, err)
					}
				default:
					if err := WriteTemplate(tmpl, writePath, data); err != nil {
						return fmt.Errorf("failed to write file %s: %w", writePath, err)
					}
				}
			} else if err = WriteFile(writePath, readFile); err != nil {
				return fmt.Errorf("failed to write file %s: %w", writePath, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create package: %v", err)
	}

	return InitPackage(config)
}

func WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(path), err)
	}

	fmt.Println("Writing file:", path)

	return os.WriteFile(path, data, 0644)
}

func WriteTemplate(tmpl *template.Template, path string, data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(path), err)
	}

	buf := new(bytes.Buffer)
	err := tmpl.Execute(buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template %s: %w", path, err)
	}

	fmt.Println("Writing file:", path)

	return os.WriteFile(path, buf.Bytes(), 0644)
}

func InitPackage(config *GeneratePackageConfig) error {
	ident := common.GetPackageIdentity(config.Spec.GetPackage())

	switch config.Type() {
	case "GO":
		if _, err := os.Stat(config.Path + "/go.mod"); err != nil && os.IsNotExist(err) {
			pkgMod := "github.com/datakit-dev/dtkt-integrations/" + ident.Slug()
			modCmd := exec.Command("go", "mod", "init", pkgMod)
			modCmd.Dir = config.Path
			modCmd.Stderr = os.Stderr
			modCmd.Stdout = os.Stdout

			if err = modCmd.Run(); err != nil {
				return err
			}
		}

		var modCmdsArgs = [][]string{
			{"mod", "edit", "-replace", "github.com/michaelquigley/pfxlog=github.com/michaelquigley/pfxlog@v0.6.10"},
			{"get", "-tool", "golang.org/x/tools/cmd/goimports"},
			{"tool", "goimports", "-w", "."},
		}

		// ***************************************
		// TODO: remove this once SDK is published
		// var (
		// 	sdkPath = "../../../dtkt-sdk/sdk-go"
		// 	scanner = bufio.NewScanner(os.Stdin)
		// )

		// sdkPath = filepath.Join(config.Path, sdkPath)
		// for {
		// 	fi, err := os.Stat(sdkPath)
		// 	if err == nil && fi.IsDir() {
		// 		break
		// 	} else if err != nil && os.IsNotExist(err) {
		// 		fmt.Printf("Enter local path for Go SDK: ")

		// 		if ok := scanner.Scan(); !ok {
		// 			return fmt.Errorf("failed to read input")
		// 		}

		// 		sdkPath = scanner.Text()
		// 	}
		// }

		// modCmdsArgs = append(modCmdsArgs, []string{"mod", "edit", "-replace", "github.com/datakit-dev/dtkt-sdk/sdk-go=" + sdkPath})
		// ***************************************

		for _, args := range modCmdsArgs {
			modCmd := exec.Command("go", args...)
			modCmd.Dir = config.Path
			modCmd.Stderr = os.Stderr
			modCmd.Stdout = os.Stdout

			err := modCmd.Run()
			if err != nil {
				return err
			}
		}

		if _, err := os.Stat(config.Path + "/buf.yaml"); err == nil {
			bufUpdateCmd := exec.Command("buf", "dep", "update")
			bufUpdateCmd.Dir = config.Path
			bufUpdateCmd.Stderr = os.Stderr
			bufUpdateCmd.Stdout = os.Stdout

			err := bufUpdateCmd.Run()
			if err != nil {
				return err
			}

			bufGenerateCmd := exec.Command("buf", "generate")
			bufGenerateCmd.Dir = config.Path
			bufGenerateCmd.Stderr = os.Stderr
			bufGenerateCmd.Stdout = os.Stdout

			err = bufGenerateCmd.Run()
			if err != nil {
				return err
			}
		}

		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = config.Path
		tidyCmd.Stderr = os.Stderr
		tidyCmd.Stdout = os.Stdout

		err := tidyCmd.Run()
		if err != nil {
			return err
		}

	case "NODE":
		if _, err := os.Stat(config.Path + "/package.json"); err != nil && os.IsNotExist(err) {
			pkgCmd := exec.Command("npm", "init", "-y")
			pkgCmd.Dir = config.Path

			return pkgCmd.Run()
		}
	case "PYTHON":
		if _, err := os.Stat(config.Path + "/pyproject.toml"); err != nil && os.IsNotExist(err) {
			pkgCmd := exec.Command("pdm", "init", "minimal")
			pkgCmd.Stdin = os.Stdin
			pkgCmd.Stdout = os.Stdout
			pkgCmd.Stderr = os.Stderr
			pkgCmd.Dir = config.Path

			return pkgCmd.Run()
		}
	}

	return nil
}

func (c *GeneratePackageConfig) SpecPath() string {
	return path.Join(c.Path, integrationsdk.SpecFile)
}

func (c *GeneratePackageConfig) Type() string {
	return common.TrimProtoEnumPrefixFor[sharedv1beta1.PackageType](c.Spec.GetPackage().GetType().String())
}

func (c *GeneratePackageConfig) SkipFile(path string, svcPkgs *common.ProtoPackages) bool {
	path = strings.TrimPrefix(path, c.Path+"/")
	pkgType := c.Spec.GetPackage().GetType()

	if c.Lib {
		if skipTmpls, ok := skipLibFiles[pkgType]; ok {
			skip := slices.ContainsFunc(skipTmpls, func(pattern string) bool {
				match, _ := regexp.MatchString(pattern, path)
				if match {
					fmt.Println("Skipping:", path)
				}
				return match
			})

			if skip {
				return true
			}
		}
	}

	hasDocker := false
	for _, r := range c.Spec.GetPackage().GetRuntimes() {
		if r == sharedv1beta1.Runtime_RUNTIME_DOCKER {
			hasDocker = true
			break
		}
	}

	if !hasDocker {
		if skipTmpls, ok := skipDockerFiles[pkgType]; ok {
			skip := slices.ContainsFunc(skipTmpls, func(pattern string) bool {
				match, _ := regexp.MatchString(pattern, path)
				if match {
					fmt.Println("Skipping:", path)
				}
				return match
			})

			if skip {
				return true
			}
		}
	}

	if !c.Proto {
		if skipTmpls, ok := skipProtoFiles[pkgType]; ok {
			skip := slices.ContainsFunc(skipTmpls, func(pattern string) bool {
				match, _ := regexp.MatchString(pattern, path)
				if match {
					fmt.Println("Skipping:", path)
				}
				return match
			})

			if skip {
				return true
			}
		}
	}

	if !svcPkgs.HasPkg("action", "v1beta1") {
		if skipTmpls, ok := skipActionFiles[pkgType]; ok {
			skip := slices.ContainsFunc(skipTmpls, func(pattern string) bool {
				match, _ := regexp.MatchString(pattern, path)
				if match {
					fmt.Println("Skipping:", path)
				}
				return match
			})

			if skip {
				return true
			}
		}
	}

	return false
}
