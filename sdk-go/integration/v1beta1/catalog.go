package v1beta1

import (
	"slices"

	catalogv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/catalog/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/types/known/structpb"
)

const CatalogLabelsKey = "dtkt_catalog"

type (
	Catalogs   []*catalogv1beta1.Catalog
	Schemas    []*catalogv1beta1.Schema
	Tables     []*catalogv1beta1.Table
	CatalogOpt func(*catalogv1beta1.Catalog)
	SchemaOpt  func(*catalogv1beta1.Schema)
	TableOpt   func(*catalogv1beta1.Table)
	QueryOpt   func(*catalogv1beta1.Query)
)

func NewCatalog(name string, opts ...CatalogOpt) *catalogv1beta1.Catalog {
	c := &catalogv1beta1.Catalog{
		Name: name,
	}
	for opt := range slices.Values(opts) {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

func NewSchema(catalog *catalogv1beta1.Catalog, name string, opts ...SchemaOpt) *catalogv1beta1.Schema {
	s := &catalogv1beta1.Schema{
		Catalog: catalog,
		Name:    name,
	}
	for opt := range slices.Values(opts) {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

func NewTable(schema *catalogv1beta1.Schema, name string, opts ...TableOpt) *catalogv1beta1.Table {
	t := &catalogv1beta1.Table{
		Schema: schema,
		Name:   name,
	}
	for opt := range slices.Values(opts) {
		if opt != nil {
			opt(t)
		}
	}
	return t
}

func NewQuery(dialect, query string, opts ...QueryOpt) *catalogv1beta1.Query {
	q := &catalogv1beta1.Query{
		Dialect: dialect,
		Query:   query,
	}
	for opt := range slices.Values(opts) {
		if opt != nil {
			opt(q)
		}
	}
	return q
}

func WithQueryFields(fields ...*sharedv1beta1.Field) QueryOpt {
	return func(q *catalogv1beta1.Query) {
		q.Fields = fields
	}
}

func WithQueryParams(params ...*sharedv1beta1.Param) QueryOpt {
	return func(q *catalogv1beta1.Query) {
		q.Params = params
	}
}

func WithCatalogDescription(desc string) CatalogOpt {
	return func(c *catalogv1beta1.Catalog) {
		c.Description = desc
	}
}

func WithCatalogMetadata(metadata *structpb.Struct) CatalogOpt {
	return func(c *catalogv1beta1.Catalog) {
		c.Metadata = metadata
	}
}

func WithSchemaDescription(desc string) SchemaOpt {
	return func(s *catalogv1beta1.Schema) {
		s.Description = desc
	}
}

func WithSchemaMetadata(metadata *structpb.Struct) SchemaOpt {
	return func(s *catalogv1beta1.Schema) {
		s.Metadata = metadata
	}
}

func WithTableDescription(desc string) TableOpt {
	return func(t *catalogv1beta1.Table) {
		t.Description = desc
	}
}

func WithTableFields(fields ...*sharedv1beta1.Field) TableOpt {
	return func(t *catalogv1beta1.Table) {
		t.Fields = fields
	}
}

func WithTableStats(stats *catalogv1beta1.TableStats) TableOpt {
	return func(t *catalogv1beta1.Table) {
		t.Stats = stats
	}
}

func WithTableMetadata(metadata *structpb.Struct) TableOpt {
	return func(t *catalogv1beta1.Table) {
		t.Metadata = metadata
	}
}

func WithQueryValid(v bool) QueryOpt {
	return func(q *catalogv1beta1.Query) {
		q.Valid = v
	}
}

func WithQueryError(err string) QueryOpt {
	return func(q *catalogv1beta1.Query) {
		q.Error = err
	}
}

func WithCatalogLabels(labels map[string]string) CatalogOpt {
	return func(c *catalogv1beta1.Catalog) {
		SetCatalogLabels(c, labels)
	}
}

func SetCatalogLabels(catalog *catalogv1beta1.Catalog, labels map[string]string) {
	if labels != nil {
		if catalog.Metadata == nil {
			catalog.Metadata = structpb.NewNullValue().GetStructValue()
		}

		if catalog.Metadata.Fields == nil {
			catalog.Metadata.Fields = map[string]*structpb.Value{}
		}

		var labelsStruct = structpb.NewNullValue().GetStructValue()
		labelsStruct.Fields = map[string]*structpb.Value{}

		for key, val := range labels {
			labelsStruct.Fields[key] = &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: val,
				},
			}
		}

		catalog.Metadata.Fields[CatalogLabelsKey] = &structpb.Value{
			Kind: &structpb.Value_StructValue{
				StructValue: labelsStruct,
			},
		}
	}
}

func GetCatalogLabels(catalog *catalogv1beta1.Catalog) map[string]string {
	// if catalog.Metadata != nil && catalog.Metadata.Values != nil {
	// 	metadata, err := AnyMapFromProto(catalog.Metadata)
	// 	if err == nil && metadata != nil {
	// 		labels, ok := metadata[CatalogLabelsKey]
	// 		if ok && labels != nil {
	// 			switch labels := labels.(type) {
	// 			case map[string]string:
	// 				return labels
	// 			case *StringMap:
	// 				if labels.Values != nil {
	// 					return labels.Values
	// 				}
	// 			}
	// 		}
	// 	}
	// }
	return map[string]string{}
}

func NewDataTypesResponse(types DataTypes) *catalogv1beta1.ListDataTypesResponse {
	return &catalogv1beta1.ListDataTypesResponse{Types: types}
}

func NewQueryDialectResponse(dialect string) *catalogv1beta1.GetQueryDialectResponse {
	return &catalogv1beta1.GetQueryDialectResponse{Dialect: dialect}
}

func NewColumnPermission(name, alias, nativeType string) *catalogv1beta1.ColumnPermission {
	return &catalogv1beta1.ColumnPermission{
		Name:  name,
		Alias: alias,
		Type:  nativeType,
	}
}

func NewTablePermission(name, alias string, columns ...*catalogv1beta1.ColumnPermission) *catalogv1beta1.TablePermission {
	return &catalogv1beta1.TablePermission{
		Name:    name,
		Alias:   alias,
		Columns: columns,
	}
}

func NewSchemaPermission(name, alias string, tables ...*catalogv1beta1.TablePermission) *catalogv1beta1.SchemaPermission {
	return &catalogv1beta1.SchemaPermission{
		Name:   name,
		Alias:  alias,
		Tables: tables,
	}
}

func NewCatalogPermission(name, alias string, schemas ...*catalogv1beta1.SchemaPermission) *catalogv1beta1.CatalogPermission {
	return &catalogv1beta1.CatalogPermission{
		Name:    name,
		Alias:   alias,
		Schemas: schemas,
	}
}

func NewValidateQueryReq(query string, accessible []*catalogv1beta1.CatalogPermission, params Params) *catalogv1beta1.ValidateQueryRequest {
	return &catalogv1beta1.ValidateQueryRequest{Query: query, Accessible: accessible, Params: params}
}

func NewValidateQueryRes(query *catalogv1beta1.Query, accessed []*catalogv1beta1.CatalogPermission) *catalogv1beta1.ValidateQueryResponse {
	return &catalogv1beta1.ValidateQueryResponse{Query: query, Accessed: accessed}
}
