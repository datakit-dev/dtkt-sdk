package sqladapter

import (
	"fmt"
	"strings"

	sqlv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/sql/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostore"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/go-openapi/inflect"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TableFromProto(desc protoreflect.MessageDescriptor, opts ...TableOption) (*sqlv1beta1.Table, error) {
	schemaName := util.ToSnakeCase(strings.ReplaceAll(string(desc.FullName().Parent()), ".", "_"))
	tableName := util.ToSnakeCase(inflect.Pluralize(string(desc.Name())))
	builder := sqlv1beta1.Table_builder{
		Name: new(fmt.Sprintf("%s.%s", schemaName, tableName)),
	}

	for idx := range desc.Fields().Len() {
		desc := desc.Fields().Get(idx)
		column, err := ColumnFromProto(desc)
		if err != nil {
			return nil, err
		} else if column == nil {
			continue
		}

		builder.Columns = append(builder.Columns, column)
	}

	table := builder.Build()

	for _, opt := range opts {
		if opt != nil {
			opt(table)
		}
	}

	return table, nil
}

func TableOfProto[T any, M protostore.MessageType[T]](opts ...TableOption) (*sqlv1beta1.Table, error) {
	return TableFromProto((M)(nil).ProtoReflect().Descriptor(), opts...)
}
