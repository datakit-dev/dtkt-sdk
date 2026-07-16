package integrationsdk

import (
	"fmt"
	"io"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

const (
	SpecSchemaID       = "dtkt.core.v1.PackageSpec"
	SpecSchemaFilename = SpecSchemaID + v1beta1.JSONSchemaFileExt
	SpecFile           = "package.dtkt.yaml"
)

func SpecLoader(opts ...common.JSONSchemaOption) *api.SpecLoader[*corev1.PackageSpec] {
	return api.NewSpecLoader[*corev1.PackageSpec](opts...)
}

func ReadSpec(reader io.Reader, format encoding.Format) (*corev1.PackageSpec, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	return SpecLoader().Decode(data, format)
}

func WriteSpec(writer io.Writer, format encoding.Format, spec *corev1.PackageSpec) error {
	b, err := SpecLoader().Encode(spec, format)
	if err != nil {
		return err
	}

	_, err = writer.Write(b)

	return err
}
