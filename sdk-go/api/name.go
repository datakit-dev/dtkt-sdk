package api

import (
	"regexp"

	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	dtktTypePattern      = regexp.MustCompile(`^dtkt\.([^\.]+)\.([^\.]+)\.?(.*)`)
	wellKnownTypePattern = regexp.MustCompile(`^(google|buf)\.(validate|protobuf|api|type|geo|rpc)\.`)
)

const AnyTypeUrlPrefix = "type.googleapis.com/"

func IsKnownName[T ~string](name T) bool {
	if IsWellKnownName(name) {
		return true
	}

	for _, version := range validVersions {
		if VersionContainsName(version, name) {
			return true
		}
	}

	return false
}

func VersionContainsName[T ~string](version Version, name T) bool {
	matches := dtktTypePattern.FindStringSubmatch(string(name))
	if len(matches) < 3 {
		return false
	}

	switch version {
	case CoreV1:
		if matches[1] == "core" && matches[2] == "v1" {
			return true
		}
	}

	return version.GetName() == matches[2]
}

func VersionContainsDescriptor(version Version, desc protoreflect.Descriptor) bool {
	return VersionContainsName(version, desc.FullName())
}

func IsWellKnownName[S ~string](name S) bool {
	return wellKnownTypePattern.MatchString(string(name))
}

func IsWellKnownType(desc protoreflect.Descriptor) bool {
	return IsWellKnownName(desc.FullName())
}
