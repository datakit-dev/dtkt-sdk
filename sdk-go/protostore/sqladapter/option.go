package sqladapter

import sqlv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/sql/v1beta1"

type TableOption func(*sqlv1beta1.Table)

func WithTableName(name string) TableOption {
	return func(t *sqlv1beta1.Table) {
		t.SetName(name)
	}
}
