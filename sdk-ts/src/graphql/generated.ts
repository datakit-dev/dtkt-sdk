import { DocumentNode } from 'graphql';
import gql from 'graphql-tag';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Any: { input: unknown; output: unknown; }
  Bytes: { input: string; output: string; }
  /**
   * Define a Relay Cursor type:
   * https://relay.dev/graphql/connections.htm#sec-Cursor
   */
  Cursor: { input: string; output: string; }
  Int64: { input: number; output: number; }
  /** The builtin Map type */
  Map: { input: Record<string, unknown>; output: Record<string, unknown>; }
  /** The builtin Time type */
  Time: { input: string; output: string; }
};

/** ActionType is enum for the type of action that can be performed on a resource. */
export enum ActionType {
  Admin = 'ADMIN',
  Create = 'CREATE',
  Delete = 'DELETE',
  Read = 'READ',
  Update = 'UPDATE',
  View = 'VIEW',
  Write = 'WRITE'
}

/** AddRoleInput is used for adding a role to a User in Org & Space (optional) for the given resources. */
export type AddRoleInput = {
  readonly orgID: Scalars['ID']['input'];
  readonly resources?: InputMaybe<ReadonlyArray<RoleResourceInput>>;
  readonly spaceID?: InputMaybe<Scalars['ID']['input']>;
  readonly type: RoleType;
  readonly userIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
};

export type AuthCheck = {
  readonly __typename?: 'AuthCheck';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly oAuthUrl?: Maybe<Scalars['String']['output']>;
  readonly required: Scalars['Boolean']['output'];
  readonly success: Scalars['Boolean']['output'];
  readonly type: ServiceAuthType;
};


export type AuthCheckOAuthUrlArgs = {
  redirectURL: Scalars['String']['input'];
};

export type BaseServiceMetadata = {
  readonly __typename?: 'BaseServiceMetadata';
  readonly customActions: ReadonlyArray<CustomActionMetadata>;
};

export type Catalog = Node & {
  readonly __typename?: 'Catalog';
  readonly alias: Scalars['String']['output'];
  readonly autoSync: Scalars['Boolean']['output'];
  readonly connection: Connection;
  readonly connectionID: Scalars['ID']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly deletedAt?: Maybe<Scalars['Time']['output']>;
  readonly description?: Maybe<Scalars['String']['output']>;
  readonly id: Scalars['ID']['output'];
  readonly label: Scalars['String']['output'];
  readonly models: ModelConnection;
  readonly name: Scalars['String']['output'];
  readonly organization: Organization;
  readonly organizationID: Scalars['ID']['output'];
  readonly queryDialect: Scalars['String']['output'];
  readonly schemas: SchemaRefConnection;
  readonly sources: SourceConnection;
  readonly spaces?: Maybe<ReadonlyArray<Space>>;
  readonly syncError?: Maybe<Scalars['String']['output']>;
  readonly syncStatus: SyncStatus;
  readonly syncedAt?: Maybe<Scalars['Time']['output']>;
  readonly updatedAt: Scalars['Time']['output'];
};


export type CatalogModelsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ModelOrder>;
  where?: InputMaybe<ModelWhereInput>;
};


export type CatalogSchemasArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ReadonlyArray<SchemaRefOrder>>;
  where?: InputMaybe<SchemaRefWhereInput>;
};


export type CatalogSourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SourceOrder>;
  where?: InputMaybe<SourceWhereInput>;
};

/** A connection to a list of items. */
export type CatalogConnection = {
  readonly __typename?: 'CatalogConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<CatalogEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type CatalogCreated = {
  readonly __typename?: 'CatalogCreated';
  readonly catalog?: Maybe<Catalog>;
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
};

export type CatalogDeleted = {
  readonly __typename?: 'CatalogDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type CatalogEdge = {
  readonly __typename?: 'CatalogEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<Catalog>;
};

/** Ordering options for Catalog connections */
export type CatalogOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Catalogs. */
  readonly field: CatalogOrderField;
};

/** Properties by which Catalog connections can be ordered. */
export enum CatalogOrderField {
  Alias = 'ALIAS',
  CreatedAt = 'CREATED_AT',
  Label = 'LABEL',
  ModelsCount = 'MODELS_COUNT',
  Name = 'NAME',
  SchemasCount = 'SCHEMAS_COUNT',
  SourcesCount = 'SOURCES_COUNT',
  UpdatedAt = 'UPDATED_AT'
}

export type CatalogServiceMetadata = {
  readonly __typename?: 'CatalogServiceMetadata';
  readonly dataTypes: ReadonlyArray<DataTypeMetadata>;
  readonly defaultCatalog?: Maybe<Scalars['String']['output']>;
  readonly queryDialect: Scalars['String']['output'];
};

export type CatalogUpdated = {
  readonly __typename?: 'CatalogUpdated';
  readonly catalog?: Maybe<Catalog>;
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * CatalogWhereInput is used for filtering Catalog objects.
 * Input was generated by ent.
 */
export type CatalogWhereInput = {
  /** alias field predicates */
  readonly alias?: InputMaybe<Scalars['String']['input']>;
  readonly aliasContains?: InputMaybe<Scalars['String']['input']>;
  readonly aliasContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly aliasEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly aliasGT?: InputMaybe<Scalars['String']['input']>;
  readonly aliasGTE?: InputMaybe<Scalars['String']['input']>;
  readonly aliasHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly aliasHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly aliasIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly aliasLT?: InputMaybe<Scalars['String']['input']>;
  readonly aliasLTE?: InputMaybe<Scalars['String']['input']>;
  readonly aliasNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly aliasNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly and?: InputMaybe<ReadonlyArray<CatalogWhereInput>>;
  /** auto_sync field predicates */
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly autoSyncNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** connection_id field predicates */
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly connectionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** deleted_at field predicates */
  readonly deletedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly deletedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** connection edge predicates */
  readonly hasConnection?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionWith?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** models edge predicates */
  readonly hasModels?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasModelsWith?: InputMaybe<ReadonlyArray<ModelWhereInput>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** schemas edge predicates */
  readonly hasSchemas?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSchemasWith?: InputMaybe<ReadonlyArray<SchemaRefWhereInput>>;
  /** sources edge predicates */
  readonly hasSources?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSourcesWith?: InputMaybe<ReadonlyArray<SourceWhereInput>>;
  /** spaces edge predicates */
  readonly hasSpaces?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSpacesWith?: InputMaybe<ReadonlyArray<SpaceWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** label field predicates */
  readonly label?: InputMaybe<Scalars['String']['input']>;
  readonly labelContains?: InputMaybe<Scalars['String']['input']>;
  readonly labelContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly labelEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly labelGT?: InputMaybe<Scalars['String']['input']>;
  readonly labelGTE?: InputMaybe<Scalars['String']['input']>;
  readonly labelHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly labelHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly labelIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly labelLT?: InputMaybe<Scalars['String']['input']>;
  readonly labelLTE?: InputMaybe<Scalars['String']['input']>;
  readonly labelNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly labelNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<CatalogWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<CatalogWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** query_dialect field predicates */
  readonly queryDialect?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectContains?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectGT?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectGTE?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly queryDialectLT?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectLTE?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly queryDialectNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** sync_error field predicates */
  readonly syncError?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContains?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncErrorLT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorLTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** sync_status field predicates */
  readonly syncStatus?: InputMaybe<SyncStatus>;
  readonly syncStatusIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  readonly syncStatusNEQ?: InputMaybe<SyncStatus>;
  readonly syncStatusNotIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  /** synced_at field predicates */
  readonly syncedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type ColumnRef = Node & {
  readonly __typename?: 'ColumnRef';
  readonly alias: Scalars['String']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly deletedAt?: Maybe<Scalars['Time']['output']>;
  readonly description?: Maybe<Scalars['String']['output']>;
  readonly dtype: Scalars['String']['output'];
  readonly fields?: Maybe<ReadonlyArray<Maybe<Field>>>;
  readonly id: Scalars['ID']['output'];
  readonly name: Scalars['String']['output'];
  readonly nullable: Scalars['Boolean']['output'];
  readonly ordinal: Scalars['Int']['output'];
  readonly primaryKey: Scalars['Boolean']['output'];
  readonly repeated: Scalars['Boolean']['output'];
  readonly sqlQueries: SqlQueryConnection;
  readonly table: TableRef;
  readonly tableID: Scalars['ID']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};


export type ColumnRefSqlQueriesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SqlQueryOrder>;
  where?: InputMaybe<SqlQueryWhereInput>;
};

/** A connection to a list of items. */
export type ColumnRefConnection = {
  readonly __typename?: 'ColumnRefConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<ColumnRefEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type ColumnRefCreated = {
  readonly __typename?: 'ColumnRefCreated';
  readonly columnRef?: Maybe<ColumnRef>;
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
};

export type ColumnRefDeleted = {
  readonly __typename?: 'ColumnRefDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type ColumnRefEdge = {
  readonly __typename?: 'ColumnRefEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<ColumnRef>;
};

/** Ordering options for ColumnRef connections */
export type ColumnRefOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order ColumnRefs. */
  readonly field: ColumnRefOrderField;
};

/** Properties by which ColumnRef connections can be ordered. */
export enum ColumnRefOrderField {
  Alias = 'ALIAS',
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  Ordinal = 'ORDINAL',
  UpdatedAt = 'UPDATED_AT'
}

export type ColumnRefUpdated = {
  readonly __typename?: 'ColumnRefUpdated';
  readonly columnRef?: Maybe<ColumnRef>;
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * ColumnRefWhereInput is used for filtering ColumnRef objects.
 * Input was generated by ent.
 */
export type ColumnRefWhereInput = {
  /** alias field predicates */
  readonly alias?: InputMaybe<Scalars['String']['input']>;
  readonly aliasContains?: InputMaybe<Scalars['String']['input']>;
  readonly aliasContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly aliasEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly aliasGT?: InputMaybe<Scalars['String']['input']>;
  readonly aliasGTE?: InputMaybe<Scalars['String']['input']>;
  readonly aliasHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly aliasHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly aliasIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly aliasLT?: InputMaybe<Scalars['String']['input']>;
  readonly aliasLTE?: InputMaybe<Scalars['String']['input']>;
  readonly aliasNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly aliasNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly and?: InputMaybe<ReadonlyArray<ColumnRefWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** deleted_at field predicates */
  readonly deletedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly deletedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** dtype field predicates */
  readonly dtype?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeContains?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeGT?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeGTE?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly dtypeLT?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeLTE?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly dtypeNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** sql_queries edge predicates */
  readonly hasSQLQueries?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSQLQueriesWith?: InputMaybe<ReadonlyArray<SqlQueryWhereInput>>;
  /** table edge predicates */
  readonly hasTable?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasTableWith?: InputMaybe<ReadonlyArray<TableRefWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<ColumnRefWhereInput>;
  /** nullable field predicates */
  readonly nullable?: InputMaybe<Scalars['Boolean']['input']>;
  readonly nullableNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  readonly or?: InputMaybe<ReadonlyArray<ColumnRefWhereInput>>;
  /** ordinal field predicates */
  readonly ordinal?: InputMaybe<Scalars['Int']['input']>;
  readonly ordinalGT?: InputMaybe<Scalars['Int']['input']>;
  readonly ordinalGTE?: InputMaybe<Scalars['Int']['input']>;
  readonly ordinalIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly ordinalLT?: InputMaybe<Scalars['Int']['input']>;
  readonly ordinalLTE?: InputMaybe<Scalars['Int']['input']>;
  readonly ordinalNEQ?: InputMaybe<Scalars['Int']['input']>;
  readonly ordinalNotIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  /** primary_key field predicates */
  readonly primaryKey?: InputMaybe<Scalars['Boolean']['input']>;
  readonly primaryKeyNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** repeated field predicates */
  readonly repeated?: InputMaybe<Scalars['Boolean']['input']>;
  readonly repeatedNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** table_id field predicates */
  readonly tableID?: InputMaybe<Scalars['ID']['input']>;
  readonly tableIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly tableIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly tableIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type ConfigCheck = {
  readonly __typename?: 'ConfigCheck';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly success: Scalars['Boolean']['output'];
};

export type Connection = Node & {
  readonly __typename?: 'Connection';
  readonly catalogs: CatalogConnection;
  readonly connectionUsers?: Maybe<ReadonlyArray<ConnectionUser>>;
  readonly createdAt: Scalars['Time']['output'];
  readonly id: Scalars['ID']['output'];
  readonly integration: Integration;
  readonly integrationID: Scalars['ID']['output'];
  readonly metadata: ConnectionMetadata;
  readonly name: Scalars['String']['output'];
  readonly organizationID: Scalars['ID']['output'];
  readonly organizations?: Maybe<ReadonlyArray<Organization>>;
  readonly owner: Organization;
  readonly slug: Scalars['String']['output'];
  readonly sourceTypes: SourceTypeConnection;
  readonly sources: SourceConnection;
  readonly updatedAt: Scalars['Time']['output'];
  readonly visibility: Visibility;
};


export type ConnectionCatalogsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<CatalogOrder>;
  where?: InputMaybe<CatalogWhereInput>;
};


export type ConnectionSourceTypesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SourceTypeOrder>;
  where?: InputMaybe<SourceTypeWhereInput>;
};


export type ConnectionSourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SourceOrder>;
  where?: InputMaybe<SourceWhereInput>;
};

export type ConnectionCheck = {
  readonly __typename?: 'ConnectionCheck';
  readonly authCheck: AuthCheck;
  readonly configCheck: ConfigCheck;
};

/** A connection to a list of items. */
export type ConnectionConnection = {
  readonly __typename?: 'ConnectionConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<ConnectionEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type ConnectionCreated = {
  readonly __typename?: 'ConnectionCreated';
  readonly connection?: Maybe<Connection>;
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
};

export type ConnectionDeleted = {
  readonly __typename?: 'ConnectionDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type ConnectionEdge = {
  readonly __typename?: 'ConnectionEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<Connection>;
};

export type ConnectionMetadata = {
  readonly __typename?: 'ConnectionMetadata';
  readonly baseService: BaseServiceMetadata;
  readonly catalogService?: Maybe<CatalogServiceMetadata>;
  readonly destinationService?: Maybe<DestinationServiceMetadata>;
  readonly embeddingService?: Maybe<EmbeddingServiceMetadata>;
  readonly eventService?: Maybe<EventServiceMetadata>;
  readonly sourceService?: Maybe<SourceServiceMetadata>;
};

/** Ordering options for Connection connections */
export type ConnectionOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Connections. */
  readonly field: ConnectionOrderField;
};

/** Properties by which Connection connections can be ordered. */
export enum ConnectionOrderField {
  CatalogsCount = 'CATALOGS_COUNT',
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  SourcesCount = 'SOURCES_COUNT',
  SourceTypesCount = 'SOURCE_TYPES_COUNT',
  UpdatedAt = 'UPDATED_AT'
}

export type ConnectionUpdated = {
  readonly __typename?: 'ConnectionUpdated';
  readonly connection?: Maybe<Connection>;
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly updated: Scalars['Boolean']['output'];
};

export type ConnectionUser = Node & {
  readonly __typename?: 'ConnectionUser';
  readonly connection: Connection;
  readonly connectionID: Scalars['ID']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly id: Scalars['ID']['output'];
  readonly organization: Organization;
  readonly organizationID: Scalars['ID']['output'];
  readonly updatedAt: Scalars['Time']['output'];
  readonly user: User;
  readonly userID: Scalars['ID']['output'];
};

/** A connection to a list of items. */
export type ConnectionUserConnection = {
  readonly __typename?: 'ConnectionUserConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<ConnectionUserEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type ConnectionUserCreated = {
  readonly __typename?: 'ConnectionUserCreated';
  readonly connectionUser?: Maybe<ConnectionUser>;
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
};

export type ConnectionUserDeleted = {
  readonly __typename?: 'ConnectionUserDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type ConnectionUserEdge = {
  readonly __typename?: 'ConnectionUserEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<ConnectionUser>;
};

/** Ordering options for ConnectionUser connections */
export type ConnectionUserOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order ConnectionUsers. */
  readonly field: ConnectionUserOrderField;
};

/** Properties by which ConnectionUser connections can be ordered. */
export enum ConnectionUserOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type ConnectionUserUpdated = {
  readonly __typename?: 'ConnectionUserUpdated';
  readonly connectionUser?: Maybe<ConnectionUser>;
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * ConnectionUserWhereInput is used for filtering ConnectionUser objects.
 * Input was generated by ent.
 */
export type ConnectionUserWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<ConnectionUserWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly not?: InputMaybe<ConnectionUserWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<ConnectionUserWhereInput>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

/**
 * ConnectionWhereInput is used for filtering Connection objects.
 * Input was generated by ent.
 */
export type ConnectionWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** catalogs edge predicates */
  readonly hasCatalogs?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasCatalogsWith?: InputMaybe<ReadonlyArray<CatalogWhereInput>>;
  /** connection_users edge predicates */
  readonly hasConnectionUsers?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionUsersWith?: InputMaybe<ReadonlyArray<ConnectionUserWhereInput>>;
  /** integration edge predicates */
  readonly hasIntegration?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasIntegrationWith?: InputMaybe<ReadonlyArray<IntegrationWhereInput>>;
  /** organizations edge predicates */
  readonly hasOrganizations?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationsWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** owner edge predicates */
  readonly hasOwner?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOwnerWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** source_types edge predicates */
  readonly hasSourceTypes?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSourceTypesWith?: InputMaybe<ReadonlyArray<SourceTypeWhereInput>>;
  /** sources edge predicates */
  readonly hasSources?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSourcesWith?: InputMaybe<ReadonlyArray<SourceWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** integration_id field predicates */
  readonly integrationID?: InputMaybe<Scalars['ID']['input']>;
  readonly integrationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly integrationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly integrationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<ConnectionWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** visibility field predicates */
  readonly visibility?: InputMaybe<Visibility>;
  readonly visibilityIn?: InputMaybe<ReadonlyArray<Visibility>>;
  readonly visibilityNEQ?: InputMaybe<Visibility>;
  readonly visibilityNotIn?: InputMaybe<ReadonlyArray<Visibility>>;
};

/** CreateCatalogInput is used for create Catalog object. */
export type CreateCatalogInput = {
  readonly alias?: InputMaybe<Scalars['String']['input']>;
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly connectionID: Scalars['ID']['input'];
  readonly createDestination?: InputMaybe<Scalars['Boolean']['input']>;
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly label?: InputMaybe<Scalars['String']['input']>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
};

export type CreateColumnInput = {
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly name: Scalars['String']['input'];
  readonly nullable?: InputMaybe<Scalars['Boolean']['input']>;
  readonly primaryKey?: InputMaybe<Scalars['Boolean']['input']>;
  readonly repeated?: InputMaybe<Scalars['Boolean']['input']>;
  readonly tableID: Scalars['ID']['input'];
  readonly type: Scalars['String']['input'];
};

/** CreateConnectionInput is used to create Connection object. */
export type CreateConnectionInput = {
  readonly config: Scalars['Any']['input'];
  readonly integrationID: Scalars['ID']['input'];
  /** name is the name of the Connection. Optional: if not provided, integration name will be used. */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly ownerID: Scalars['ID']['input'];
  readonly visibility?: InputMaybe<Visibility>;
};

/**
 * CreateDestinationInput is used to create a Destination given
 * a connection ID and config overrides.
 */
export type CreateDestinationInput = {
  readonly config?: InputMaybe<Scalars['Map']['input']>;
  /** Destination connection ID. */
  readonly connectionID: Scalars['ID']['input'];
};

/**
 * CreateEventSourceInput is used for create EventSource object.
 * Input was generated by ent.
 */
export type CreateEventSourceInput = {
  readonly config: Scalars['String']['input'];
  readonly connectionID: Scalars['ID']['input'];
  readonly name: Scalars['String']['input'];
  /** Pull frequency in string format, e.g.: "1s", "2.3h" or "4h35m" */
  readonly pullFreq?: InputMaybe<Scalars['String']['input']>;
  readonly strategy: EventSourceStrategy;
};

/** CreateFlowInput is used to create a Flow. */
export type CreateFlowInput = {
  readonly body?: InputMaybe<Scalars['String']['input']>;
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly format?: InputMaybe<SpecFormat>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly organizationId?: InputMaybe<Scalars['ID']['input']>;
};

/** CreateFlowRevisionInput creates a new revision of the flow. */
export type CreateFlowRevisionInput = {
  readonly body: Scalars['String']['input'];
  readonly flowId: Scalars['ID']['input'];
  readonly format?: SpecFormat;
  readonly updateFlow?: InputMaybe<Scalars['Boolean']['input']>;
};

/** CreateFlowRunInput creates a flow run. */
export type CreateFlowRunInput = {
  readonly autoStart?: InputMaybe<Scalars['Boolean']['input']>;
  readonly config?: InputMaybe<FlowRunConfigInput>;
  readonly revisionId: Scalars['ID']['input'];
  readonly runTimeout?: InputMaybe<Scalars['String']['input']>;
  readonly stopTimeout?: InputMaybe<Scalars['String']['input']>;
};

/**
 * CreateGeoLayerInput is used for create GeoLayer object.
 * Input was generated by ent.
 */
export type CreateGeoLayerInput = {
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly geoField?: InputMaybe<Scalars['String']['input']>;
  readonly mapId: Scalars['ID']['input'];
  readonly name: Scalars['String']['input'];
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly parentID?: InputMaybe<Scalars['ID']['input']>;
  readonly propFields?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly settings?: InputMaybe<GeoLayerSettingsInput>;
  readonly sourceId?: InputMaybe<Scalars['ID']['input']>;
  readonly visibility?: InputMaybe<Visibility>;
};

/**
 * CreateGeoMapInput is used for create GeoMap object.
 * Input was generated by ent.
 */
export type CreateGeoMapInput = {
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly layerIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly name: Scalars['String']['input'];
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly settings?: InputMaybe<GeoMapSettingsInput>;
  readonly visibility?: InputMaybe<Visibility>;
};

/**
 * CreateIntegrationInput is used for create Integration object.
 * Input was generated by ent.
 */
export type CreateIntegrationInput = {
  readonly address: Scalars['String']['input'];
  readonly apiVersion: Scalars['String']['input'];
  readonly configSchema?: InputMaybe<Scalars['String']['input']>;
  readonly description: Scalars['String']['input'];
  readonly icon: Scalars['String']['input'];
  readonly name: Scalars['String']['input'];
  readonly network: Scalars['String']['input'];
  readonly organizationID: Scalars['ID']['input'];
  readonly serviceNames?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly version: Scalars['String']['input'];
};

/**
 * CreateModelInput is used for create Model object.
 * Input was generated by ent.
 */
export type CreateModelInput = {
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly catalogID: Scalars['ID']['input'];
  readonly metadata?: InputMaybe<Scalars['Map']['input']>;
  readonly sourceIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly typeID: Scalars['ID']['input'];
};

export type CreateOrganizationInput = {
  readonly name: Scalars['String']['input'];
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly tzName?: InputMaybe<Scalars['String']['input']>;
};

/** CreatePersonalAccessTokenInput is used to create a PersonalAccessToken. */
export type CreatePersonalAccessTokenInput = {
  readonly expiresAt?: InputMaybe<Scalars['Time']['input']>;
  /** Expiration in string format, e.g.: "1s", "2.3h" or "4h35m" */
  readonly expiresIn?: InputMaybe<Scalars['String']['input']>;
  readonly name: Scalars['String']['input'];
};

export type CreateSqlQueryInput = {
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  readonly dialect: Scalars['String']['input'];
  readonly params?: InputMaybe<ReadonlyArray<ParamInput>>;
  readonly spaceID: Scalars['ID']['input'];
  readonly sql: Scalars['String']['input'];
};

/**
 * CreateSchemaRefInput is used for create SchemaRef object.
 * Input was generated by ent.
 */
export type CreateSchemaRefInput = {
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly catalogID: Scalars['ID']['input'];
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly name: Scalars['String']['input'];
};

/** CreateSourceInput is used to create a replication Source from a Connection (if supported). */
export type CreateSourceInput = {
  /** When enabled, Source will sync automatically on the schedule provided by syncFreq and syncTime (if applicable). */
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  /** Catalog ID to resolve storage connection. */
  readonly catalogID: Scalars['ID']['input'];
  /** Optional config. */
  readonly config?: InputMaybe<Scalars['Map']['input']>;
  /** Source connection ID. */
  readonly connectionID: Scalars['ID']['input'];
  /** Schema name to create in catalog. */
  readonly schemaName: Scalars['String']['input'];
  /** Frequency of sync in minutes (default: 360 minutes (6 hours)) */
  readonly syncFreq?: InputMaybe<Scalars['Int']['input']>;
  /** Time of day to sync, has no effect when syncFreq is < 24 hours (default: 00:00) */
  readonly syncTime?: InputMaybe<Scalars['String']['input']>;
  /** Source type ID. */
  readonly typeID: Scalars['ID']['input'];
};

/**
 * CreateSpaceInput is used for create Space object.
 * Input was generated by ent.
 */
export type CreateSpaceInput = {
  readonly catalogIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly flowIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly flowRunIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly geoMapIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly name: Scalars['String']['input'];
  /** Timezone name (e.g. America/Los_Angeles) */
  readonly tzName?: InputMaybe<Scalars['String']['input']>;
};

export type CreateTableInput = {
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly fields?: InputMaybe<ReadonlyArray<FieldInput>>;
  readonly name: Scalars['String']['input'];
  readonly schemaID: Scalars['ID']['input'];
};

export type CustomActionMetadata = {
  readonly __typename?: 'CustomActionMetadata';
  readonly description: Scalars['String']['output'];
  readonly inputSchema: Scalars['String']['output'];
  readonly name: Scalars['String']['output'];
  readonly outputSchema: Scalars['String']['output'];
};

export type DataType = {
  readonly __typename?: 'DataType';
  readonly geoType?: Maybe<Scalars['String']['output']>;
  readonly jsonType?: Maybe<Scalars['String']['output']>;
  readonly metadata?: Maybe<Scalars['Map']['output']>;
  readonly nativeType: Scalars['String']['output'];
};

export type DataTypeMetadata = {
  readonly __typename?: 'DataTypeMetadata';
  readonly jsonType: Scalars['String']['output'];
  readonly nativeType: Scalars['String']['output'];
};

export type Destination = Node & {
  readonly __typename?: 'Destination';
  readonly connection: Connection;
  readonly connectionID: Scalars['ID']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly id: Scalars['ID']['output'];
  readonly name: Scalars['String']['output'];
  readonly organization: Organization;
  readonly organizationID: Scalars['ID']['output'];
  readonly providerID: Scalars['String']['output'];
  readonly sources?: Maybe<ReadonlyArray<Source>>;
  /** Timezone name (e.g. America/Los_Angeles) */
  readonly tzName: Scalars['String']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};

export type DestinationConfig = {
  readonly __typename?: 'DestinationConfig';
  readonly schema: Scalars['String']['output'];
  readonly service: Scalars['String']['output'];
};

/** A connection to a list of items. */
export type DestinationConnection = {
  readonly __typename?: 'DestinationConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<DestinationEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type DestinationCreated = {
  readonly __typename?: 'DestinationCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly destination?: Maybe<Destination>;
  readonly error?: Maybe<Scalars['String']['output']>;
};

export type DestinationDeleted = {
  readonly __typename?: 'DestinationDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type DestinationEdge = {
  readonly __typename?: 'DestinationEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<Destination>;
};

/** Ordering options for Destination connections */
export type DestinationOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Destinations. */
  readonly field: DestinationOrderField;
};

/** Properties by which Destination connections can be ordered. */
export enum DestinationOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type DestinationServiceMetadata = {
  readonly __typename?: 'DestinationServiceMetadata';
  readonly destinationConfigs: ReadonlyArray<DestinationConfig>;
};

export type DestinationUpdated = {
  readonly __typename?: 'DestinationUpdated';
  readonly destination?: Maybe<Destination>;
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * DestinationWhereInput is used for filtering Destination objects.
 * Input was generated by ent.
 */
export type DestinationWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<DestinationWhereInput>>;
  /** connection_id field predicates */
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly connectionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** connection edge predicates */
  readonly hasConnection?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionWith?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** sources edge predicates */
  readonly hasSources?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSourcesWith?: InputMaybe<ReadonlyArray<SourceWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<DestinationWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<DestinationWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** provider_id field predicates */
  readonly providerID?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDContains?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDGT?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDGTE?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly providerIDLT?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDLTE?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** tz_name field predicates */
  readonly tzName?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameContains?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameGT?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly tzNameLT?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type EmbeddingModelMetadata = {
  readonly __typename?: 'EmbeddingModelMetadata';
  readonly id: Scalars['String']['output'];
  readonly maxDimensions: Scalars['Int']['output'];
  readonly minDimensions: Scalars['Int']['output'];
  readonly name: Scalars['String']['output'];
};

export type EmbeddingServiceMetadata = {
  readonly __typename?: 'EmbeddingServiceMetadata';
  readonly models: ReadonlyArray<EmbeddingModelMetadata>;
};

export type EventMetadata = {
  readonly __typename?: 'EventMetadata';
  readonly description: Scalars['String']['output'];
  readonly payloadSchema: Scalars['String']['output'];
  readonly type: Scalars['String']['output'];
};

export type EventServiceMetadata = {
  readonly __typename?: 'EventServiceMetadata';
  readonly eventSources: ReadonlyArray<EventSourceMetadata>;
  readonly events: ReadonlyArray<EventMetadata>;
};

export type EventSource = Node & {
  readonly __typename?: 'EventSource';
  readonly connection: Connection;
  readonly connectionID: Scalars['ID']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly creationAt?: Maybe<Scalars['Time']['output']>;
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id: Scalars['ID']['output'];
  readonly name: Scalars['String']['output'];
  /** Pull frequency in string format, e.g.: "1s", "2.3h" or "4h35m" */
  readonly pullFreq?: Maybe<Scalars['String']['output']>;
  readonly pushUrl?: Maybe<Scalars['String']['output']>;
  readonly requiresCreation: Scalars['Boolean']['output'];
  readonly slug: Scalars['String']['output'];
  readonly startedAt?: Maybe<Scalars['Time']['output']>;
  readonly status: EventSourceStatus;
  readonly stoppedAt?: Maybe<Scalars['Time']['output']>;
  readonly strategy: EventSourceStrategy;
  readonly updatedAt: Scalars['Time']['output'];
};

/** A connection to a list of items. */
export type EventSourceConnection = {
  readonly __typename?: 'EventSourceConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<EventSourceEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type EventSourceCreated = {
  readonly __typename?: 'EventSourceCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly eventSource?: Maybe<EventSource>;
};

export type EventSourceDeleted = {
  readonly __typename?: 'EventSourceDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type EventSourceEdge = {
  readonly __typename?: 'EventSourceEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<EventSource>;
};

export type EventSourceMetadata = {
  readonly __typename?: 'EventSourceMetadata';
  readonly configSchema: Scalars['String']['output'];
  readonly description: Scalars['String']['output'];
  readonly name: Scalars['String']['output'];
  readonly strategy: EventSourceStrategy;
};

/** Ordering options for EventSource connections */
export type EventSourceOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order EventSources. */
  readonly field: EventSourceOrderField;
};

/** Properties by which EventSource connections can be ordered. */
export enum EventSourceOrderField {
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  StartedAt = 'STARTED_AT',
  StoppedAt = 'STOPPED_AT',
  UpdatedAt = 'UPDATED_AT'
}

/** EventSourceStatus is enum for the field status */
export enum EventSourceStatus {
  Cancelled = 'CANCELLED',
  Error = 'ERROR',
  Pending = 'PENDING',
  Running = 'RUNNING',
  Scheduled = 'SCHEDULED',
  Stopped = 'STOPPED'
}

/** EventSourceStrategy is enum for the field strategy */
export enum EventSourceStrategy {
  Pull = 'PULL',
  Push = 'PUSH'
}

export type EventSourceUpdated = {
  readonly __typename?: 'EventSourceUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly eventSource?: Maybe<EventSource>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * EventSourceWhereInput is used for filtering EventSource objects.
 * Input was generated by ent.
 */
export type EventSourceWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<EventSourceWhereInput>>;
  /** connection_id field predicates */
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly connectionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** creation_at field predicates */
  readonly creationAt?: InputMaybe<Scalars['Time']['input']>;
  readonly creationAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly creationAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly creationAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly creationAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly creationAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly creationAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly creationAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly creationAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly creationAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** error field predicates */
  readonly error?: InputMaybe<Scalars['String']['input']>;
  readonly errorContains?: InputMaybe<Scalars['String']['input']>;
  readonly errorContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly errorEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly errorGT?: InputMaybe<Scalars['String']['input']>;
  readonly errorGTE?: InputMaybe<Scalars['String']['input']>;
  readonly errorHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly errorHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly errorIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly errorIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly errorLT?: InputMaybe<Scalars['String']['input']>;
  readonly errorLTE?: InputMaybe<Scalars['String']['input']>;
  readonly errorNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly errorNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly errorNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** connection edge predicates */
  readonly hasConnection?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionWith?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<EventSourceWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<EventSourceWhereInput>>;
  /** requires_creation field predicates */
  readonly requiresCreation?: InputMaybe<Scalars['Boolean']['input']>;
  readonly requiresCreationNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** started_at field predicates */
  readonly startedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly startedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly startedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly startedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** status field predicates */
  readonly status?: InputMaybe<EventSourceStatus>;
  readonly statusIn?: InputMaybe<ReadonlyArray<EventSourceStatus>>;
  readonly statusNEQ?: InputMaybe<EventSourceStatus>;
  readonly statusNotIn?: InputMaybe<ReadonlyArray<EventSourceStatus>>;
  /** stopped_at field predicates */
  readonly stoppedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly stoppedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly stoppedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly stoppedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** strategy field predicates */
  readonly strategy?: InputMaybe<EventSourceStrategy>;
  readonly strategyIn?: InputMaybe<ReadonlyArray<EventSourceStrategy>>;
  readonly strategyNEQ?: InputMaybe<EventSourceStrategy>;
  readonly strategyNotIn?: InputMaybe<ReadonlyArray<EventSourceStrategy>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type ExternalDataSource = {
  readonly __typename?: 'ExternalDataSource';
  readonly id: Scalars['String']['output'];
  readonly provider: ReportProvider;
  readonly url: Scalars['String']['output'];
};

export type ExternalReport = {
  readonly __typename?: 'ExternalReport';
  readonly dataSource?: Maybe<ExternalDataSource>;
  readonly id: Scalars['String']['output'];
  readonly provider: ReportProvider;
  readonly url: Scalars['String']['output'];
};

export type Field = {
  readonly __typename?: 'Field';
  readonly description?: Maybe<Scalars['String']['output']>;
  readonly fields?: Maybe<ReadonlyArray<Field>>;
  readonly name: Scalars['String']['output'];
  readonly nullable: Scalars['Boolean']['output'];
  readonly repeated?: Maybe<Scalars['Boolean']['output']>;
  readonly type: DataType;
};

export type FieldInput = {
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly fields?: InputMaybe<ReadonlyArray<FieldInput>>;
  readonly name: Scalars['String']['input'];
  readonly nativeType: Scalars['String']['input'];
  readonly nullable: Scalars['Boolean']['input'];
  readonly repeated?: InputMaybe<Scalars['Boolean']['input']>;
};

export type Flow = Node & {
  readonly __typename?: 'Flow';
  readonly body: FlowBody;
  readonly createdAt: Scalars['Time']['output'];
  readonly description?: Maybe<Scalars['String']['output']>;
  readonly graph: FlowGraph;
  readonly id: Scalars['ID']['output'];
  readonly name: Scalars['String']['output'];
  readonly organization: Organization;
  readonly organizationID: Scalars['ID']['output'];
  readonly resources: FlowResourceConnection;
  readonly revision?: Maybe<FlowRevision>;
  readonly revisionID?: Maybe<Scalars['ID']['output']>;
  readonly revisions: FlowRevisionConnection;
  readonly runs: FlowRunConnection;
  readonly slug: Scalars['String']['output'];
  readonly spaces?: Maybe<ReadonlyArray<Space>>;
  readonly updatedAt: Scalars['Time']['output'];
  readonly visibility: FlowVisibility;
};


export type FlowBodyArgs = {
  format?: SpecFormat;
};


export type FlowResourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowResourceOrder>;
  where?: InputMaybe<FlowResourceWhereInput>;
};


export type FlowRevisionsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowRevisionOrder>;
  where?: InputMaybe<FlowRevisionWhereInput>;
};


export type FlowRunsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowRunOrder>;
  where?: InputMaybe<FlowRunWhereInput>;
};

export type FlowBody = {
  readonly __typename?: 'FlowBody';
  readonly format: SpecFormat;
  readonly map: Scalars['Map']['output'];
  readonly raw: Scalars['String']['output'];
};

/** A connection to a list of items. */
export type FlowConnection = {
  readonly __typename?: 'FlowConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<FlowEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type FlowCreated = {
  readonly __typename?: 'FlowCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly flow?: Maybe<Flow>;
};

export type FlowDeleted = {
  readonly __typename?: 'FlowDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type FlowEdge = {
  readonly __typename?: 'FlowEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<Flow>;
};

export type FlowEnv = {
  readonly __typename?: 'FlowEnv';
  readonly nodes: ReadonlyArray<FlowNodeState>;
};

export type FlowGraph = {
  readonly __typename?: 'FlowGraph';
  readonly edges: ReadonlyArray<FlowGraphEdge>;
  readonly nodes: ReadonlyArray<FlowGraphNode>;
};

export type FlowGraphEdge = {
  readonly __typename?: 'FlowGraphEdge';
  readonly data: Scalars['Map']['output'];
  readonly sourceId: Scalars['String']['output'];
  readonly targetId: Scalars['String']['output'];
};

export type FlowGraphNode = {
  readonly __typename?: 'FlowGraphNode';
  readonly data: Scalars['Map']['output'];
  readonly id: Scalars['String']['output'];
  readonly type: Scalars['String']['output'];
};

export type FlowNodeState = {
  readonly __typename?: 'FlowNodeState';
  readonly nodeId: Scalars['String']['output'];
  readonly nodeType: Scalars['String']['output'];
  readonly status: FlowNodeStatus;
};

export enum FlowNodeStatus {
  Completed = 'COMPLETED',
  Idle = 'IDLE',
  Pending = 'PENDING'
}

/** Ordering options for Flow connections */
export type FlowOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Flows. */
  readonly field: FlowOrderField;
};

/** Properties by which Flow connections can be ordered. */
export enum FlowOrderField {
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  UpdatedAt = 'UPDATED_AT'
}

export type FlowResource = Node & {
  readonly __typename?: 'FlowResource';
  readonly createdAt: Scalars['Time']['output'];
  readonly flow: Flow;
  readonly flowID: Scalars['ID']['output'];
  readonly id: Scalars['ID']['output'];
  readonly nodeID: Scalars['String']['output'];
  readonly nodeType: Scalars['String']['output'];
  readonly resourceID: Scalars['ID']['output'];
  readonly resourceType: Scalars['String']['output'];
  readonly revision: FlowRevision;
  readonly revisionID: Scalars['ID']['output'];
  readonly run: FlowRun;
  readonly runID: Scalars['ID']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};

/** A connection to a list of items. */
export type FlowResourceConnection = {
  readonly __typename?: 'FlowResourceConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<FlowResourceEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type FlowResourceCreated = {
  readonly __typename?: 'FlowResourceCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly flowResource?: Maybe<FlowResource>;
};

export type FlowResourceDeleted = {
  readonly __typename?: 'FlowResourceDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type FlowResourceEdge = {
  readonly __typename?: 'FlowResourceEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<FlowResource>;
};

/** Ordering options for FlowResource connections */
export type FlowResourceOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order FlowResources. */
  readonly field: FlowResourceOrderField;
};

/** Properties by which FlowResource connections can be ordered. */
export enum FlowResourceOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type FlowResourceUpdated = {
  readonly __typename?: 'FlowResourceUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly flowResource?: Maybe<FlowResource>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * FlowResourceWhereInput is used for filtering FlowResource objects.
 * Input was generated by ent.
 */
export type FlowResourceWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<FlowResourceWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** flow_id field predicates */
  readonly flowID?: InputMaybe<Scalars['ID']['input']>;
  readonly flowIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly flowIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly flowIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** flow edge predicates */
  readonly hasFlow?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasFlowWith?: InputMaybe<ReadonlyArray<FlowWhereInput>>;
  /** revision edge predicates */
  readonly hasRevision?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasRevisionWith?: InputMaybe<ReadonlyArray<FlowRevisionWhereInput>>;
  /** run edge predicates */
  readonly hasRun?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasRunWith?: InputMaybe<ReadonlyArray<FlowRunWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** node_id field predicates */
  readonly nodeID?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDContains?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDGT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nodeIDLT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nodeIDNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** node_type field predicates */
  readonly nodeType?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeContains?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeGT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nodeTypeLT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<FlowResourceWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<FlowResourceWhereInput>>;
  /** resource_id field predicates */
  readonly resourceID?: InputMaybe<Scalars['ID']['input']>;
  readonly resourceIDGT?: InputMaybe<Scalars['ID']['input']>;
  readonly resourceIDGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly resourceIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly resourceIDLT?: InputMaybe<Scalars['ID']['input']>;
  readonly resourceIDLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly resourceIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly resourceIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** resource_type field predicates */
  readonly resourceType?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeContains?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeGT?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeGTE?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly resourceTypeLT?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeLTE?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly resourceTypeNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** revision_id field predicates */
  readonly revisionID?: InputMaybe<Scalars['ID']['input']>;
  readonly revisionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly revisionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly revisionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** run_id field predicates */
  readonly runID?: InputMaybe<Scalars['ID']['input']>;
  readonly runIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly runIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly runIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type FlowRevision = Node & {
  readonly __typename?: 'FlowRevision';
  readonly body: FlowBody;
  readonly checksum: Scalars['String']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly flow: Flow;
  readonly flowID: Scalars['ID']['output'];
  readonly graph: FlowGraph;
  readonly id: Scalars['ID']['output'];
  readonly ioSchema?: Maybe<IoSchema>;
  readonly resources: FlowResourceConnection;
  readonly runs: FlowRunConnection;
  readonly updatedAt: Scalars['Time']['output'];
};


export type FlowRevisionBodyArgs = {
  format?: SpecFormat;
};


export type FlowRevisionResourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowResourceOrder>;
  where?: InputMaybe<FlowResourceWhereInput>;
};


export type FlowRevisionRunsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowRunOrder>;
  where?: InputMaybe<FlowRunWhereInput>;
};

/** A connection to a list of items. */
export type FlowRevisionConnection = {
  readonly __typename?: 'FlowRevisionConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<FlowRevisionEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type FlowRevisionCreated = {
  readonly __typename?: 'FlowRevisionCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly flowRevision?: Maybe<FlowRevision>;
};

export type FlowRevisionDeleted = {
  readonly __typename?: 'FlowRevisionDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type FlowRevisionEdge = {
  readonly __typename?: 'FlowRevisionEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<FlowRevision>;
};

/** Ordering options for FlowRevision connections */
export type FlowRevisionOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order FlowRevisions. */
  readonly field: FlowRevisionOrderField;
};

/** Properties by which FlowRevision connections can be ordered. */
export enum FlowRevisionOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type FlowRevisionUpdated = {
  readonly __typename?: 'FlowRevisionUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly flowRevision?: Maybe<FlowRevision>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * FlowRevisionWhereInput is used for filtering FlowRevision objects.
 * Input was generated by ent.
 */
export type FlowRevisionWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<FlowRevisionWhereInput>>;
  /** checksum field predicates */
  readonly checksum?: InputMaybe<Scalars['String']['input']>;
  readonly checksumContains?: InputMaybe<Scalars['String']['input']>;
  readonly checksumContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly checksumEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly checksumGT?: InputMaybe<Scalars['String']['input']>;
  readonly checksumGTE?: InputMaybe<Scalars['String']['input']>;
  readonly checksumHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly checksumHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly checksumIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly checksumLT?: InputMaybe<Scalars['String']['input']>;
  readonly checksumLTE?: InputMaybe<Scalars['String']['input']>;
  readonly checksumNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly checksumNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** flow_id field predicates */
  readonly flowID?: InputMaybe<Scalars['ID']['input']>;
  readonly flowIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly flowIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly flowIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** flow edge predicates */
  readonly hasFlow?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasFlowWith?: InputMaybe<ReadonlyArray<FlowWhereInput>>;
  /** io_schema edge predicates */
  readonly hasIoSchema?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasIoSchemaWith?: InputMaybe<ReadonlyArray<IoSchemaWhereInput>>;
  /** resources edge predicates */
  readonly hasResources?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasResourcesWith?: InputMaybe<ReadonlyArray<FlowResourceWhereInput>>;
  /** runs edge predicates */
  readonly hasRuns?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasRunsWith?: InputMaybe<ReadonlyArray<FlowRunWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly not?: InputMaybe<FlowRevisionWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<FlowRevisionWhereInput>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type FlowRun = Node & {
  readonly __typename?: 'FlowRun';
  readonly body: FlowBody;
  readonly config: FlowRunConfig;
  readonly createdAt: Scalars['Time']['output'];
  readonly env: FlowEnv;
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly flow: Flow;
  readonly flowID: Scalars['ID']['output'];
  readonly graph: FlowGraph;
  readonly id: Scalars['ID']['output'];
  readonly resources: FlowResourceConnection;
  readonly revision: FlowRevision;
  readonly revisionID: Scalars['ID']['output'];
  /** Run timeout in string format, e.g.: "1s", "2.3h" or "4h35m". (Optional; when not set, run must be stopped manually) */
  readonly runTimeout: Scalars['String']['output'];
  readonly space: Space;
  readonly spaceID: Scalars['ID']['output'];
  readonly startedAt?: Maybe<Scalars['Time']['output']>;
  readonly status: FlowRunStatus;
  /** Stop timeout in string format, e.g.: "1s", "2.3h" or "4h35m". (Default: 1s) */
  readonly stopTimeout: Scalars['String']['output'];
  readonly stoppedAt?: Maybe<Scalars['Time']['output']>;
  readonly updatedAt: Scalars['Time']['output'];
};


export type FlowRunBodyArgs = {
  format?: SpecFormat;
};


export type FlowRunResourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowResourceOrder>;
  where?: InputMaybe<FlowResourceWhereInput>;
};

export type FlowRunConfig = {
  readonly __typename?: 'FlowRunConfig';
  readonly inputs?: Maybe<Scalars['Map']['output']>;
  readonly resources?: Maybe<ReadonlyArray<FlowRunResource>>;
};

export type FlowRunConfigInput = {
  readonly inputs?: InputMaybe<Scalars['Map']['input']>;
  readonly resources?: InputMaybe<ReadonlyArray<FlowRunResourceInput>>;
};

/** A connection to a list of items. */
export type FlowRunConnection = {
  readonly __typename?: 'FlowRunConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<FlowRunEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type FlowRunCreated = {
  readonly __typename?: 'FlowRunCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly flowRun?: Maybe<FlowRun>;
};

export type FlowRunDeleted = {
  readonly __typename?: 'FlowRunDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

export type FlowRunDone = {
  readonly __typename?: 'FlowRunDone';
  readonly eventType: FlowRunEventType;
  readonly flowRun: FlowRun;
};

/** An edge in a connection. */
export type FlowRunEdge = {
  readonly __typename?: 'FlowRunEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<FlowRun>;
};

export type FlowRunError = {
  readonly __typename?: 'FlowRunError';
  readonly error: Scalars['String']['output'];
  readonly eventType: FlowRunEventType;
  readonly flowRun: FlowRun;
};

export type FlowRunEvent = FlowRunDone | FlowRunError | FlowRunNodeStatus | FlowRunOutputs | FlowRunUserAction;

export enum FlowRunEventType {
  Done = 'DONE',
  Error = 'ERROR',
  NodeStatus = 'NODE_STATUS',
  Outputs = 'OUTPUTS',
  UserAction = 'USER_ACTION'
}

export type FlowRunNodeStatus = {
  readonly __typename?: 'FlowRunNodeStatus';
  readonly eventType: FlowRunEventType;
  readonly flowRun: FlowRun;
  readonly nodeId: Scalars['String']['output'];
  readonly nodeType: Scalars['String']['output'];
  readonly status: FlowNodeStatus;
};

/** Ordering options for FlowRun connections */
export type FlowRunOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order FlowRuns. */
  readonly field: FlowRunOrderField;
};

/** Properties by which FlowRun connections can be ordered. */
export enum FlowRunOrderField {
  CreatedAt = 'CREATED_AT',
  StartedAt = 'STARTED_AT',
  StoppedAt = 'STOPPED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type FlowRunOutputs = {
  readonly __typename?: 'FlowRunOutputs';
  readonly eventType: FlowRunEventType;
  readonly flowRun: FlowRun;
  readonly value: Scalars['Map']['output'];
};

export type FlowRunResource = {
  readonly __typename?: 'FlowRunResource';
  readonly id: Scalars['ID']['output'];
  readonly nodeId: Scalars['String']['output'];
  readonly type: Scalars['String']['output'];
};

export type FlowRunResourceInput = {
  readonly id: Scalars['ID']['input'];
  readonly nodeId: Scalars['String']['input'];
  readonly type: Scalars['String']['input'];
};

/** FlowRunStatus is enum for the field status */
export enum FlowRunStatus {
  Cancelled = 'CANCELLED',
  Error = 'ERROR',
  Pending = 'PENDING',
  Running = 'RUNNING',
  Scheduled = 'SCHEDULED',
  Stopped = 'STOPPED',
  Success = 'SUCCESS'
}

/** FlowRunSubscribeInput starts a flow run subscription. */
export type FlowRunSubscribeInput = {
  readonly eventTypes: ReadonlyArray<FlowRunEventType>;
  readonly runId: Scalars['ID']['input'];
};

export type FlowRunUpdated = {
  readonly __typename?: 'FlowRunUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly flowRun?: Maybe<FlowRun>;
  readonly updated: Scalars['Boolean']['output'];
};

export type FlowRunUserAction = {
  readonly __typename?: 'FlowRunUserAction';
  readonly eventType: FlowRunEventType;
  readonly flowRun: FlowRun;
  readonly nodeId: Scalars['String']['output'];
  readonly nodeType: Scalars['String']['output'];
};

/**
 * FlowRunWhereInput is used for filtering FlowRun objects.
 * Input was generated by ent.
 */
export type FlowRunWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<FlowRunWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** error field predicates */
  readonly error?: InputMaybe<Scalars['String']['input']>;
  readonly errorContains?: InputMaybe<Scalars['String']['input']>;
  readonly errorContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly errorEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly errorGT?: InputMaybe<Scalars['String']['input']>;
  readonly errorGTE?: InputMaybe<Scalars['String']['input']>;
  readonly errorHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly errorHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly errorIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly errorIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly errorLT?: InputMaybe<Scalars['String']['input']>;
  readonly errorLTE?: InputMaybe<Scalars['String']['input']>;
  readonly errorNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly errorNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly errorNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** flow_id field predicates */
  readonly flowID?: InputMaybe<Scalars['ID']['input']>;
  readonly flowIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly flowIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly flowIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** flow edge predicates */
  readonly hasFlow?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasFlowWith?: InputMaybe<ReadonlyArray<FlowWhereInput>>;
  /** resources edge predicates */
  readonly hasResources?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasResourcesWith?: InputMaybe<ReadonlyArray<FlowResourceWhereInput>>;
  /** revision edge predicates */
  readonly hasRevision?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasRevisionWith?: InputMaybe<ReadonlyArray<FlowRevisionWhereInput>>;
  /** space edge predicates */
  readonly hasSpace?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSpaceWith?: InputMaybe<ReadonlyArray<SpaceWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly not?: InputMaybe<FlowRunWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<FlowRunWhereInput>>;
  /** revision_id field predicates */
  readonly revisionID?: InputMaybe<Scalars['ID']['input']>;
  readonly revisionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly revisionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly revisionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** space_id field predicates */
  readonly spaceID?: InputMaybe<Scalars['ID']['input']>;
  readonly spaceIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly spaceIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly spaceIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** started_at field predicates */
  readonly startedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly startedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly startedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly startedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly startedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** status field predicates */
  readonly status?: InputMaybe<FlowRunStatus>;
  readonly statusIn?: InputMaybe<ReadonlyArray<FlowRunStatus>>;
  readonly statusNEQ?: InputMaybe<FlowRunStatus>;
  readonly statusNotIn?: InputMaybe<ReadonlyArray<FlowRunStatus>>;
  /** stopped_at field predicates */
  readonly stoppedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly stoppedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly stoppedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly stoppedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly stoppedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type FlowUpdated = {
  readonly __typename?: 'FlowUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly flow?: Maybe<Flow>;
  readonly updated: Scalars['Boolean']['output'];
};

/** FlowVisibility is enum for the field visibility */
export enum FlowVisibility {
  Private = 'PRIVATE',
  Public = 'PUBLIC'
}

/**
 * FlowWhereInput is used for filtering Flow objects.
 * Input was generated by ent.
 */
export type FlowWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<FlowWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** resources edge predicates */
  readonly hasResources?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasResourcesWith?: InputMaybe<ReadonlyArray<FlowResourceWhereInput>>;
  /** revision edge predicates */
  readonly hasRevision?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasRevisionWith?: InputMaybe<ReadonlyArray<FlowRevisionWhereInput>>;
  /** revisions edge predicates */
  readonly hasRevisions?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasRevisionsWith?: InputMaybe<ReadonlyArray<FlowRevisionWhereInput>>;
  /** runs edge predicates */
  readonly hasRuns?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasRunsWith?: InputMaybe<ReadonlyArray<FlowRunWhereInput>>;
  /** spaces edge predicates */
  readonly hasSpaces?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSpacesWith?: InputMaybe<ReadonlyArray<SpaceWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<FlowWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<FlowWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** revision_id field predicates */
  readonly revisionID?: InputMaybe<Scalars['ID']['input']>;
  readonly revisionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly revisionIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly revisionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly revisionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly revisionIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** visibility field predicates */
  readonly visibility?: InputMaybe<FlowVisibility>;
  readonly visibilityIn?: InputMaybe<ReadonlyArray<FlowVisibility>>;
  readonly visibilityNEQ?: InputMaybe<FlowVisibility>;
  readonly visibilityNotIn?: InputMaybe<ReadonlyArray<FlowVisibility>>;
};

export type GeoFeature = Node & {
  readonly __typename?: 'GeoFeature';
  readonly createdAt: Scalars['Time']['output'];
  readonly geoHash: Scalars['String']['output'];
  readonly geoID: Scalars['Int']['output'];
  readonly id: Scalars['ID']['output'];
  readonly layer: GeoLayer;
  readonly layerID: Scalars['ID']['output'];
  readonly properties?: Maybe<Scalars['Map']['output']>;
  readonly updatedAt: Scalars['Time']['output'];
};

/** A connection to a list of items. */
export type GeoFeatureConnection = {
  readonly __typename?: 'GeoFeatureConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<GeoFeatureEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type GeoFeatureCreated = {
  readonly __typename?: 'GeoFeatureCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly geoFeature?: Maybe<GeoFeature>;
};

export type GeoFeatureDeleted = {
  readonly __typename?: 'GeoFeatureDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type GeoFeatureEdge = {
  readonly __typename?: 'GeoFeatureEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<GeoFeature>;
};

/** Ordering options for GeoFeature connections */
export type GeoFeatureOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order GeoFeatures. */
  readonly field: GeoFeatureOrderField;
};

/** Properties by which GeoFeature connections can be ordered. */
export enum GeoFeatureOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type GeoFeatureUpdated = {
  readonly __typename?: 'GeoFeatureUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly geoFeature?: Maybe<GeoFeature>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * GeoFeatureWhereInput is used for filtering GeoFeature objects.
 * Input was generated by ent.
 */
export type GeoFeatureWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<GeoFeatureWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** geo_hash field predicates */
  readonly geoHash?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashContains?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashGT?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashGTE?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly geoHashLT?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashLTE?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly geoHashNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** geo_id field predicates */
  readonly geoID?: InputMaybe<Scalars['Int']['input']>;
  readonly geoIDGT?: InputMaybe<Scalars['Int']['input']>;
  readonly geoIDGTE?: InputMaybe<Scalars['Int']['input']>;
  readonly geoIDIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly geoIDLT?: InputMaybe<Scalars['Int']['input']>;
  readonly geoIDLTE?: InputMaybe<Scalars['Int']['input']>;
  readonly geoIDNEQ?: InputMaybe<Scalars['Int']['input']>;
  readonly geoIDNotIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  /** layer edge predicates */
  readonly hasLayer?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasLayerWith?: InputMaybe<ReadonlyArray<GeoLayerWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** layer_id field predicates */
  readonly layerID?: InputMaybe<Scalars['ID']['input']>;
  readonly layerIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly layerIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly layerIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly not?: InputMaybe<GeoFeatureWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<GeoFeatureWhereInput>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type GeoLayer = Node & {
  readonly __typename?: 'GeoLayer';
  readonly autoSync: Scalars['Boolean']['output'];
  readonly children?: Maybe<ReadonlyArray<GeoLayer>>;
  readonly config: Scalars['Map']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly description?: Maybe<Scalars['String']['output']>;
  readonly features: GeoFeatureConnection;
  readonly geoField?: Maybe<Scalars['String']['output']>;
  readonly id: Scalars['ID']['output'];
  readonly map?: Maybe<ReadonlyArray<GeoMap>>;
  readonly name: Scalars['String']['output'];
  readonly organization: Organization;
  readonly organizationID: Scalars['ID']['output'];
  readonly parent?: Maybe<ReadonlyArray<GeoLayer>>;
  readonly parentID?: Maybe<Scalars['ID']['output']>;
  readonly propFields: ReadonlyArray<GeoPropField>;
  readonly settings: GeoLayerSettings;
  readonly slug: Scalars['String']['output'];
  readonly source?: Maybe<ReadonlyArray<GeoSource>>;
  readonly sourceID?: Maybe<Scalars['ID']['output']>;
  readonly syncError?: Maybe<Scalars['String']['output']>;
  readonly syncStatus: SyncStatus;
  readonly syncedAt?: Maybe<Scalars['Time']['output']>;
  readonly updatedAt: Scalars['Time']['output'];
  readonly visibility: Visibility;
};


export type GeoLayerFeaturesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<GeoFeatureOrder>;
  where?: InputMaybe<GeoFeatureWhereInput>;
};

/** A connection to a list of items. */
export type GeoLayerConnection = {
  readonly __typename?: 'GeoLayerConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<GeoLayerEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type GeoLayerCreated = {
  readonly __typename?: 'GeoLayerCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly geoLayer?: Maybe<GeoLayer>;
};

export type GeoLayerDeleted = {
  readonly __typename?: 'GeoLayerDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type GeoLayerEdge = {
  readonly __typename?: 'GeoLayerEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<GeoLayer>;
};

/** Ordering options for GeoLayer connections */
export type GeoLayerOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order GeoLayers. */
  readonly field: GeoLayerOrderField;
};

/** Properties by which GeoLayer connections can be ordered. */
export enum GeoLayerOrderField {
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  UpdatedAt = 'UPDATED_AT'
}

export type GeoLayerSettings = {
  readonly __typename?: 'GeoLayerSettings';
  readonly fillColor: Scalars['String']['output'];
  readonly fillOpacity: Scalars['Float']['output'];
  readonly strokeColor: Scalars['String']['output'];
  readonly strokeOpacity: Scalars['Float']['output'];
  readonly strokeWidth: Scalars['Int']['output'];
};

export type GeoLayerSettingsInput = {
  readonly fillColor?: InputMaybe<Scalars['String']['input']>;
  readonly fillOpacity?: InputMaybe<Scalars['Float']['input']>;
  readonly strokeColor?: InputMaybe<Scalars['String']['input']>;
  readonly strokeOpacity?: InputMaybe<Scalars['Float']['input']>;
  readonly strokeWidth?: InputMaybe<Scalars['Int']['input']>;
};

export type GeoLayerUpdated = {
  readonly __typename?: 'GeoLayerUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly geoLayer?: Maybe<GeoLayer>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * GeoLayerWhereInput is used for filtering GeoLayer objects.
 * Input was generated by ent.
 */
export type GeoLayerWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<GeoLayerWhereInput>>;
  /** auto_sync field predicates */
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly autoSyncNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** geo_field field predicates */
  readonly geoField?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldContains?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldGT?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldGTE?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly geoFieldIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly geoFieldLT?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldLTE?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly geoFieldNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly geoFieldNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** children edge predicates */
  readonly hasChildren?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasChildrenWith?: InputMaybe<ReadonlyArray<GeoLayerWhereInput>>;
  /** features edge predicates */
  readonly hasFeatures?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasFeaturesWith?: InputMaybe<ReadonlyArray<GeoFeatureWhereInput>>;
  /** map edge predicates */
  readonly hasMap?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasMapWith?: InputMaybe<ReadonlyArray<GeoMapWhereInput>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** parent edge predicates */
  readonly hasParent?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasParentWith?: InputMaybe<ReadonlyArray<GeoLayerWhereInput>>;
  /** source edge predicates */
  readonly hasSource?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSourceWith?: InputMaybe<ReadonlyArray<GeoSourceWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<GeoLayerWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<GeoLayerWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** parent_id field predicates */
  readonly parentID?: InputMaybe<Scalars['ID']['input']>;
  readonly parentIDGT?: InputMaybe<Scalars['ID']['input']>;
  readonly parentIDGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly parentIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly parentIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly parentIDLT?: InputMaybe<Scalars['ID']['input']>;
  readonly parentIDLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly parentIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly parentIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly parentIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** source_id field predicates */
  readonly sourceID?: InputMaybe<Scalars['ID']['input']>;
  readonly sourceIDGT?: InputMaybe<Scalars['ID']['input']>;
  readonly sourceIDGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly sourceIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly sourceIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly sourceIDLT?: InputMaybe<Scalars['ID']['input']>;
  readonly sourceIDLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly sourceIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly sourceIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly sourceIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** sync_error field predicates */
  readonly syncError?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContains?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncErrorLT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorLTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** sync_status field predicates */
  readonly syncStatus?: InputMaybe<SyncStatus>;
  readonly syncStatusIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  readonly syncStatusNEQ?: InputMaybe<SyncStatus>;
  readonly syncStatusNotIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  /** synced_at field predicates */
  readonly syncedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** visibility field predicates */
  readonly visibility?: InputMaybe<Visibility>;
  readonly visibilityIn?: InputMaybe<ReadonlyArray<Visibility>>;
  readonly visibilityNEQ?: InputMaybe<Visibility>;
  readonly visibilityNotIn?: InputMaybe<ReadonlyArray<Visibility>>;
};

export type GeoMap = Node & {
  readonly __typename?: 'GeoMap';
  readonly createdAt: Scalars['Time']['output'];
  readonly description?: Maybe<Scalars['String']['output']>;
  readonly id: Scalars['ID']['output'];
  readonly layers: GeoLayerConnection;
  readonly name: Scalars['String']['output'];
  readonly organization: Organization;
  readonly organizationID: Scalars['ID']['output'];
  readonly settings: GeoMapSettings;
  readonly slug: Scalars['String']['output'];
  readonly spaces?: Maybe<ReadonlyArray<Space>>;
  readonly updatedAt: Scalars['Time']['output'];
  readonly visibility: Visibility;
};


export type GeoMapLayersArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<GeoLayerOrder>;
  where?: InputMaybe<GeoLayerWhereInput>;
};

/** A connection to a list of items. */
export type GeoMapConnection = {
  readonly __typename?: 'GeoMapConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<GeoMapEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type GeoMapCreated = {
  readonly __typename?: 'GeoMapCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly geoMap?: Maybe<GeoMap>;
};

export type GeoMapDeleted = {
  readonly __typename?: 'GeoMapDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type GeoMapEdge = {
  readonly __typename?: 'GeoMapEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<GeoMap>;
};

/** Ordering options for GeoMap connections */
export type GeoMapOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order GeoMaps. */
  readonly field: GeoMapOrderField;
};

/** Properties by which GeoMap connections can be ordered. */
export enum GeoMapOrderField {
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  UpdatedAt = 'UPDATED_AT'
}

export type GeoMapSettings = {
  readonly __typename?: 'GeoMapSettings';
  readonly layerOrder: ReadonlyArray<Scalars['ID']['output']>;
};

export type GeoMapSettingsInput = {
  readonly layerOrder: ReadonlyArray<Scalars['ID']['input']>;
};

export type GeoMapUpdated = {
  readonly __typename?: 'GeoMapUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly geoMap?: Maybe<GeoMap>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * GeoMapWhereInput is used for filtering GeoMap objects.
 * Input was generated by ent.
 */
export type GeoMapWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<GeoMapWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** layers edge predicates */
  readonly hasLayers?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasLayersWith?: InputMaybe<ReadonlyArray<GeoLayerWhereInput>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** spaces edge predicates */
  readonly hasSpaces?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSpacesWith?: InputMaybe<ReadonlyArray<SpaceWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<GeoMapWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<GeoMapWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** visibility field predicates */
  readonly visibility?: InputMaybe<Visibility>;
  readonly visibilityIn?: InputMaybe<ReadonlyArray<Visibility>>;
  readonly visibilityNEQ?: InputMaybe<Visibility>;
  readonly visibilityNotIn?: InputMaybe<ReadonlyArray<Visibility>>;
};

export type GeoPropField = {
  readonly __typename?: 'GeoPropField';
  readonly name: Scalars['String']['output'];
  readonly type: GeoPropFieldType;
};

export enum GeoPropFieldType {
  GeoPropFieldBoolean = 'GEO_PROP_FIELD_BOOLEAN',
  GeoPropFieldNumber = 'GEO_PROP_FIELD_NUMBER',
  GeoPropFieldString = 'GEO_PROP_FIELD_STRING'
}

export type GeoSource = Node & {
  readonly __typename?: 'GeoSource';
  readonly connection: Connection;
  readonly connectionID: Scalars['ID']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly geoFields: ReadonlyArray<Scalars['String']['output']>;
  readonly id: Scalars['ID']['output'];
  readonly layers?: Maybe<ReadonlyArray<GeoLayer>>;
  readonly name: Scalars['String']['output'];
  readonly propFields: ReadonlyArray<GeoPropField>;
  readonly providerID: Scalars['String']['output'];
  readonly type: GeoSourceType;
  readonly updateFreq: Scalars['Int']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};

/** A connection to a list of items. */
export type GeoSourceConnection = {
  readonly __typename?: 'GeoSourceConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<GeoSourceEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type GeoSourceCreated = {
  readonly __typename?: 'GeoSourceCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly geoSource?: Maybe<GeoSource>;
};

export type GeoSourceDeleted = {
  readonly __typename?: 'GeoSourceDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type GeoSourceEdge = {
  readonly __typename?: 'GeoSourceEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<GeoSource>;
};

/** Ordering options for GeoSource connections */
export type GeoSourceOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order GeoSources. */
  readonly field: GeoSourceOrderField;
};

/** Properties by which GeoSource connections can be ordered. */
export enum GeoSourceOrderField {
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  UpdatedAt = 'UPDATED_AT'
}

/** GeoSourceType is enum for the field type */
export enum GeoSourceType {
  Query = 'Query',
  Table = 'Table'
}

export type GeoSourceUpdated = {
  readonly __typename?: 'GeoSourceUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly geoSource?: Maybe<GeoSource>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * GeoSourceWhereInput is used for filtering GeoSource objects.
 * Input was generated by ent.
 */
export type GeoSourceWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<GeoSourceWhereInput>>;
  /** connection_id field predicates */
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly connectionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** connection edge predicates */
  readonly hasConnection?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionWith?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** layers edge predicates */
  readonly hasLayers?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasLayersWith?: InputMaybe<ReadonlyArray<GeoLayerWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<GeoSourceWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<GeoSourceWhereInput>>;
  /** provider_id field predicates */
  readonly providerID?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDContains?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDGT?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDGTE?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly providerIDLT?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDLTE?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** type field predicates */
  readonly type?: InputMaybe<GeoSourceType>;
  readonly typeIn?: InputMaybe<ReadonlyArray<GeoSourceType>>;
  readonly typeNEQ?: InputMaybe<GeoSourceType>;
  readonly typeNotIn?: InputMaybe<ReadonlyArray<GeoSourceType>>;
  /** update_freq field predicates */
  readonly updateFreq?: InputMaybe<Scalars['Int']['input']>;
  readonly updateFreqGT?: InputMaybe<Scalars['Int']['input']>;
  readonly updateFreqGTE?: InputMaybe<Scalars['Int']['input']>;
  readonly updateFreqIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly updateFreqLT?: InputMaybe<Scalars['Int']['input']>;
  readonly updateFreqLTE?: InputMaybe<Scalars['Int']['input']>;
  readonly updateFreqNEQ?: InputMaybe<Scalars['Int']['input']>;
  readonly updateFreqNotIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type IoSchema = Node & {
  readonly __typename?: 'IOSchema';
  readonly createdAt: Scalars['Time']['output'];
  readonly id: Scalars['ID']['output'];
  readonly inputSchema?: Maybe<Scalars['String']['output']>;
  readonly nodeID: Scalars['ID']['output'];
  readonly nodeType: Scalars['String']['output'];
  readonly organization?: Maybe<Organization>;
  readonly organizationID?: Maybe<Scalars['ID']['output']>;
  readonly outputSchema?: Maybe<Scalars['String']['output']>;
  readonly updatedAt: Scalars['Time']['output'];
};

/** A connection to a list of items. */
export type IoSchemaConnection = {
  readonly __typename?: 'IOSchemaConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<IoSchemaEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type IoSchemaCreated = {
  readonly __typename?: 'IOSchemaCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly ioSchema?: Maybe<IoSchema>;
};

export type IoSchemaDeleted = {
  readonly __typename?: 'IOSchemaDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type IoSchemaEdge = {
  readonly __typename?: 'IOSchemaEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<IoSchema>;
};

/** Ordering options for IOSchema connections */
export type IoSchemaOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order IOSchemas. */
  readonly field: IoSchemaOrderField;
};

/** Properties by which IOSchema connections can be ordered. */
export enum IoSchemaOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type IoSchemaUpdated = {
  readonly __typename?: 'IOSchemaUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly ioSchema?: Maybe<IoSchema>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * IOSchemaWhereInput is used for filtering IOSchema objects.
 * Input was generated by ent.
 */
export type IoSchemaWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<IoSchemaWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** input_schema field predicates */
  readonly inputSchema?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaContains?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaGT?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaGTE?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly inputSchemaIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly inputSchemaLT?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaLTE?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly inputSchemaNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly inputSchemaNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** node_id field predicates */
  readonly nodeID?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDGT?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly nodeIDLT?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** node_type field predicates */
  readonly nodeType?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeContains?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeGT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nodeTypeLT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<IoSchemaWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<IoSchemaWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** output_schema field predicates */
  readonly outputSchema?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaContains?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaGT?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaGTE?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly outputSchemaIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly outputSchemaLT?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaLTE?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly outputSchemaNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly outputSchemaNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type Integration = Node & {
  readonly __typename?: 'Integration';
  readonly address: Scalars['String']['output'];
  readonly apiVersion: Scalars['String']['output'];
  readonly configSchema?: Maybe<Scalars['String']['output']>;
  readonly connections?: Maybe<ReadonlyArray<Connection>>;
  readonly createdAt: Scalars['Time']['output'];
  readonly description: Scalars['String']['output'];
  readonly icon: Scalars['String']['output'];
  readonly id: Scalars['ID']['output'];
  readonly name: Scalars['String']['output'];
  readonly network: Scalars['String']['output'];
  readonly organization: Organization;
  readonly organizationID: Scalars['ID']['output'];
  readonly packages: IntegrationPackageConnection;
  readonly published?: Maybe<IntegrationPackage>;
  readonly publishedID?: Maybe<Scalars['ID']['output']>;
  readonly serverConfig?: Maybe<Scalars['String']['output']>;
  readonly serviceNames?: Maybe<ReadonlyArray<Scalars['String']['output']>>;
  readonly slug: Scalars['String']['output'];
  readonly updatedAt: Scalars['Time']['output'];
  readonly version: Scalars['String']['output'];
};


export type IntegrationPackagesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<IntegrationPackageOrder>;
  where?: InputMaybe<IntegrationPackageWhereInput>;
};

/** A connection to a list of items. */
export type IntegrationConnection = {
  readonly __typename?: 'IntegrationConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<IntegrationEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type IntegrationCreated = {
  readonly __typename?: 'IntegrationCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly integration?: Maybe<Integration>;
};

export type IntegrationDeleted = {
  readonly __typename?: 'IntegrationDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type IntegrationEdge = {
  readonly __typename?: 'IntegrationEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<Integration>;
};

/** Ordering options for Integration connections */
export type IntegrationOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Integrations. */
  readonly field: IntegrationOrderField;
};

/** Properties by which Integration connections can be ordered. */
export enum IntegrationOrderField {
  ApiVersion = 'API_VERSION',
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  PackagesCount = 'PACKAGES_COUNT',
  UpdatedAt = 'UPDATED_AT',
  Version = 'VERSION'
}

export type IntegrationPackage = Node & {
  readonly __typename?: 'IntegrationPackage';
  readonly author: User;
  readonly authorID: Scalars['ID']['output'];
  readonly body: Scalars['Map']['output'];
  readonly checksum: Scalars['String']['output'];
  readonly configSchema: Scalars['String']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly id: Scalars['ID']['output'];
  readonly integration: Integration;
  readonly integrationID: Scalars['ID']['output'];
  readonly serviceNames: ReadonlyArray<Scalars['String']['output']>;
  readonly spec: Scalars['Bytes']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};

/** A connection to a list of items. */
export type IntegrationPackageConnection = {
  readonly __typename?: 'IntegrationPackageConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<IntegrationPackageEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type IntegrationPackageCreated = {
  readonly __typename?: 'IntegrationPackageCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly integrationPackage?: Maybe<IntegrationPackage>;
};

export type IntegrationPackageDeleted = {
  readonly __typename?: 'IntegrationPackageDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type IntegrationPackageEdge = {
  readonly __typename?: 'IntegrationPackageEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<IntegrationPackage>;
};

/** Ordering options for IntegrationPackage connections */
export type IntegrationPackageOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order IntegrationPackages. */
  readonly field: IntegrationPackageOrderField;
};

/** Properties by which IntegrationPackage connections can be ordered. */
export enum IntegrationPackageOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type IntegrationPackageUpdated = {
  readonly __typename?: 'IntegrationPackageUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly integrationPackage?: Maybe<IntegrationPackage>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * IntegrationPackageWhereInput is used for filtering IntegrationPackage objects.
 * Input was generated by ent.
 */
export type IntegrationPackageWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<IntegrationPackageWhereInput>>;
  /** author_id field predicates */
  readonly authorID?: InputMaybe<Scalars['ID']['input']>;
  readonly authorIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly authorIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly authorIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** checksum field predicates */
  readonly checksum?: InputMaybe<Scalars['String']['input']>;
  readonly checksumContains?: InputMaybe<Scalars['String']['input']>;
  readonly checksumContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly checksumEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly checksumGT?: InputMaybe<Scalars['String']['input']>;
  readonly checksumGTE?: InputMaybe<Scalars['String']['input']>;
  readonly checksumHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly checksumHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly checksumIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly checksumLT?: InputMaybe<Scalars['String']['input']>;
  readonly checksumLTE?: InputMaybe<Scalars['String']['input']>;
  readonly checksumNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly checksumNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** config_schema field predicates */
  readonly configSchema?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaContains?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaGT?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaGTE?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly configSchemaLT?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaLTE?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** author edge predicates */
  readonly hasAuthor?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasAuthorWith?: InputMaybe<ReadonlyArray<UserWhereInput>>;
  /** integration edge predicates */
  readonly hasIntegration?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasIntegrationWith?: InputMaybe<ReadonlyArray<IntegrationWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** integration_id field predicates */
  readonly integrationID?: InputMaybe<Scalars['ID']['input']>;
  readonly integrationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly integrationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly integrationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly not?: InputMaybe<IntegrationPackageWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<IntegrationPackageWhereInput>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type IntegrationUpdated = {
  readonly __typename?: 'IntegrationUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly integration?: Maybe<Integration>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * IntegrationWhereInput is used for filtering Integration objects.
 * Input was generated by ent.
 */
export type IntegrationWhereInput = {
  /** address field predicates */
  readonly address?: InputMaybe<Scalars['String']['input']>;
  readonly addressContains?: InputMaybe<Scalars['String']['input']>;
  readonly addressContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly addressEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly addressGT?: InputMaybe<Scalars['String']['input']>;
  readonly addressGTE?: InputMaybe<Scalars['String']['input']>;
  readonly addressHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly addressHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly addressIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly addressLT?: InputMaybe<Scalars['String']['input']>;
  readonly addressLTE?: InputMaybe<Scalars['String']['input']>;
  readonly addressNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly addressNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly and?: InputMaybe<ReadonlyArray<IntegrationWhereInput>>;
  /** api_version field predicates */
  readonly apiVersion?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionContains?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionGT?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly apiVersionLT?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly apiVersionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** config_schema field predicates */
  readonly configSchema?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaContains?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaGT?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaGTE?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly configSchemaIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly configSchemaLT?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaLTE?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly configSchemaNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** connections edge predicates */
  readonly hasConnections?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionsWith?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** packages edge predicates */
  readonly hasPackages?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasPackagesWith?: InputMaybe<ReadonlyArray<IntegrationPackageWhereInput>>;
  /** published edge predicates */
  readonly hasPublished?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasPublishedWith?: InputMaybe<ReadonlyArray<IntegrationPackageWhereInput>>;
  /** icon field predicates */
  readonly icon?: InputMaybe<Scalars['String']['input']>;
  readonly iconContains?: InputMaybe<Scalars['String']['input']>;
  readonly iconContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly iconEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly iconGT?: InputMaybe<Scalars['String']['input']>;
  readonly iconGTE?: InputMaybe<Scalars['String']['input']>;
  readonly iconHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly iconHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly iconIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly iconLT?: InputMaybe<Scalars['String']['input']>;
  readonly iconLTE?: InputMaybe<Scalars['String']['input']>;
  readonly iconNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly iconNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** network field predicates */
  readonly network?: InputMaybe<Scalars['String']['input']>;
  readonly networkContains?: InputMaybe<Scalars['String']['input']>;
  readonly networkContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly networkEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly networkGT?: InputMaybe<Scalars['String']['input']>;
  readonly networkGTE?: InputMaybe<Scalars['String']['input']>;
  readonly networkHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly networkHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly networkIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly networkLT?: InputMaybe<Scalars['String']['input']>;
  readonly networkLTE?: InputMaybe<Scalars['String']['input']>;
  readonly networkNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly networkNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<IntegrationWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<IntegrationWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** published_id field predicates */
  readonly publishedID?: InputMaybe<Scalars['ID']['input']>;
  readonly publishedIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly publishedIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly publishedIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly publishedIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly publishedIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** version field predicates */
  readonly version?: InputMaybe<Scalars['String']['input']>;
  readonly versionContains?: InputMaybe<Scalars['String']['input']>;
  readonly versionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly versionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly versionGT?: InputMaybe<Scalars['String']['input']>;
  readonly versionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly versionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly versionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly versionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly versionLT?: InputMaybe<Scalars['String']['input']>;
  readonly versionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly versionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly versionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
};

export type Model = Node & {
  readonly __typename?: 'Model';
  readonly autoSync: Scalars['Boolean']['output'];
  readonly catalog: Catalog;
  readonly catalogID: Scalars['ID']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly externalReports?: Maybe<ReadonlyArray<Maybe<ExternalReport>>>;
  readonly id: Scalars['ID']['output'];
  readonly metadata?: Maybe<Scalars['Map']['output']>;
  readonly name: Scalars['String']['output'];
  readonly schema: SchemaRef;
  readonly sources: SourceConnection;
  readonly syncError?: Maybe<Scalars['String']['output']>;
  readonly syncStatus: SyncStatus;
  readonly syncedAt?: Maybe<Scalars['Time']['output']>;
  readonly type: ModelType;
  readonly typeID: Scalars['ID']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};


export type ModelSourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SourceOrder>;
  where?: InputMaybe<SourceWhereInput>;
};

/** A connection to a list of items. */
export type ModelConnection = {
  readonly __typename?: 'ModelConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<ModelEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type ModelCreated = {
  readonly __typename?: 'ModelCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly model?: Maybe<Model>;
};

export type ModelDeleted = {
  readonly __typename?: 'ModelDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type ModelEdge = {
  readonly __typename?: 'ModelEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<Model>;
};

/** Ordering options for Model connections */
export type ModelOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Models. */
  readonly field: ModelOrderField;
};

/** Properties by which Model connections can be ordered. */
export enum ModelOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type ModelType = Node & {
  readonly __typename?: 'ModelType';
  readonly category: Scalars['String']['output'];
  readonly configTemplate?: Maybe<Scalars['Map']['output']>;
  readonly createdAt: Scalars['Time']['output'];
  readonly dependencies?: Maybe<ReadonlyArray<ModelType>>;
  readonly description: Scalars['String']['output'];
  readonly id: Scalars['ID']['output'];
  readonly maxPerType: Scalars['Int']['output'];
  readonly minPerType: Scalars['Int']['output'];
  readonly minSources: Scalars['Int']['output'];
  readonly modeler: ModelerType;
  readonly models?: Maybe<ReadonlyArray<Model>>;
  readonly name: Scalars['String']['output'];
  readonly parents?: Maybe<ReadonlyArray<ModelType>>;
  readonly slug: Scalars['String']['output'];
  readonly sourceTypes: ReadonlyArray<SourceType>;
  readonly sourceURL: Scalars['String']['output'];
  readonly sourceVersion?: Maybe<Scalars['String']['output']>;
  readonly updatedAt: Scalars['Time']['output'];
};

/** A connection to a list of items. */
export type ModelTypeConnection = {
  readonly __typename?: 'ModelTypeConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<ModelTypeEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type ModelTypeCreated = {
  readonly __typename?: 'ModelTypeCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly modelType?: Maybe<ModelType>;
};

export type ModelTypeDeleted = {
  readonly __typename?: 'ModelTypeDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type ModelTypeEdge = {
  readonly __typename?: 'ModelTypeEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<ModelType>;
};

/** Ordering options for ModelType connections */
export type ModelTypeOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order ModelTypes. */
  readonly field: ModelTypeOrderField;
};

/** Properties by which ModelType connections can be ordered. */
export enum ModelTypeOrderField {
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  UpdatedAt = 'UPDATED_AT'
}

export type ModelTypeUpdated = {
  readonly __typename?: 'ModelTypeUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly modelType?: Maybe<ModelType>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * ModelTypeWhereInput is used for filtering ModelType objects.
 * Input was generated by ent.
 */
export type ModelTypeWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<ModelTypeWhereInput>>;
  /** category field predicates */
  readonly category?: InputMaybe<Scalars['String']['input']>;
  readonly categoryContains?: InputMaybe<Scalars['String']['input']>;
  readonly categoryContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly categoryEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly categoryGT?: InputMaybe<Scalars['String']['input']>;
  readonly categoryGTE?: InputMaybe<Scalars['String']['input']>;
  readonly categoryHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly categoryHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly categoryIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly categoryLT?: InputMaybe<Scalars['String']['input']>;
  readonly categoryLTE?: InputMaybe<Scalars['String']['input']>;
  readonly categoryNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly categoryNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** dependencies edge predicates */
  readonly hasDependencies?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasDependenciesWith?: InputMaybe<ReadonlyArray<ModelTypeWhereInput>>;
  /** models edge predicates */
  readonly hasModels?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasModelsWith?: InputMaybe<ReadonlyArray<ModelWhereInput>>;
  /** parents edge predicates */
  readonly hasParents?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasParentsWith?: InputMaybe<ReadonlyArray<ModelTypeWhereInput>>;
  /** source_types edge predicates */
  readonly hasSourceTypes?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSourceTypesWith?: InputMaybe<ReadonlyArray<SourceTypeWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** max_per_type field predicates */
  readonly maxPerType?: InputMaybe<Scalars['Int']['input']>;
  readonly maxPerTypeGT?: InputMaybe<Scalars['Int']['input']>;
  readonly maxPerTypeGTE?: InputMaybe<Scalars['Int']['input']>;
  readonly maxPerTypeIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly maxPerTypeLT?: InputMaybe<Scalars['Int']['input']>;
  readonly maxPerTypeLTE?: InputMaybe<Scalars['Int']['input']>;
  readonly maxPerTypeNEQ?: InputMaybe<Scalars['Int']['input']>;
  readonly maxPerTypeNotIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  /** min_per_type field predicates */
  readonly minPerType?: InputMaybe<Scalars['Int']['input']>;
  readonly minPerTypeGT?: InputMaybe<Scalars['Int']['input']>;
  readonly minPerTypeGTE?: InputMaybe<Scalars['Int']['input']>;
  readonly minPerTypeIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly minPerTypeLT?: InputMaybe<Scalars['Int']['input']>;
  readonly minPerTypeLTE?: InputMaybe<Scalars['Int']['input']>;
  readonly minPerTypeNEQ?: InputMaybe<Scalars['Int']['input']>;
  readonly minPerTypeNotIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  /** min_sources field predicates */
  readonly minSources?: InputMaybe<Scalars['Int']['input']>;
  readonly minSourcesGT?: InputMaybe<Scalars['Int']['input']>;
  readonly minSourcesGTE?: InputMaybe<Scalars['Int']['input']>;
  readonly minSourcesIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly minSourcesLT?: InputMaybe<Scalars['Int']['input']>;
  readonly minSourcesLTE?: InputMaybe<Scalars['Int']['input']>;
  readonly minSourcesNEQ?: InputMaybe<Scalars['Int']['input']>;
  readonly minSourcesNotIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  /** modeler field predicates */
  readonly modeler?: InputMaybe<ModelerType>;
  readonly modelerIn?: InputMaybe<ReadonlyArray<ModelerType>>;
  readonly modelerNEQ?: InputMaybe<ModelerType>;
  readonly modelerNotIn?: InputMaybe<ReadonlyArray<ModelerType>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<ModelTypeWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<ModelTypeWhereInput>>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** source_url field predicates */
  readonly sourceURL?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLContains?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLGT?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLGTE?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly sourceURLLT?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLLTE?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly sourceURLNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** source_version field predicates */
  readonly sourceVersion?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionContains?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionGT?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly sourceVersionIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly sourceVersionLT?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly sourceVersionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly sourceVersionNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type ModelUpdated = {
  readonly __typename?: 'ModelUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly model?: Maybe<Model>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * ModelWhereInput is used for filtering Model objects.
 * Input was generated by ent.
 */
export type ModelWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<ModelWhereInput>>;
  /** auto_sync field predicates */
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly autoSyncNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** catalog_id field predicates */
  readonly catalogID?: InputMaybe<Scalars['ID']['input']>;
  readonly catalogIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly catalogIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly catalogIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** catalog edge predicates */
  readonly hasCatalog?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasCatalogWith?: InputMaybe<ReadonlyArray<CatalogWhereInput>>;
  /** schema edge predicates */
  readonly hasSchema?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSchemaWith?: InputMaybe<ReadonlyArray<SchemaRefWhereInput>>;
  /** sources edge predicates */
  readonly hasSources?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSourcesWith?: InputMaybe<ReadonlyArray<SourceWhereInput>>;
  /** type edge predicates */
  readonly hasType?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasTypeWith?: InputMaybe<ReadonlyArray<ModelTypeWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<ModelWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<ModelWhereInput>>;
  /** sync_error field predicates */
  readonly syncError?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContains?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncErrorLT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorLTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** sync_status field predicates */
  readonly syncStatus?: InputMaybe<SyncStatus>;
  readonly syncStatusIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  readonly syncStatusNEQ?: InputMaybe<SyncStatus>;
  readonly syncStatusNotIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  /** synced_at field predicates */
  readonly syncedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** type_id field predicates */
  readonly typeID?: InputMaybe<Scalars['ID']['input']>;
  readonly typeIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly typeIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly typeIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

/** ModelerType is enum for the field modeler */
export enum ModelerType {
  Dbt = 'DBT'
}

export type Mutation = {
  readonly __typename?: 'Mutation';
  /** Add a role for user */
  readonly addRole?: Maybe<Role>;
  readonly createCatalog: CatalogCreated;
  readonly createColumn?: Maybe<ColumnRefCreated>;
  readonly createConnection?: Maybe<ConnectionCreated>;
  readonly createDestination: DestinationCreated;
  readonly createEventSource?: Maybe<EventSourceCreated>;
  readonly createFlow?: Maybe<FlowCreated>;
  readonly createFlowRevision?: Maybe<FlowRevisionCreated>;
  readonly createFlowRun?: Maybe<FlowRunCreated>;
  readonly createGeoLayer?: Maybe<GeoLayerCreated>;
  readonly createGeoMap?: Maybe<GeoMapCreated>;
  readonly createIntegration?: Maybe<IntegrationCreated>;
  readonly createModel?: Maybe<ModelCreated>;
  readonly createOrganization?: Maybe<OrganizationCreated>;
  readonly createPersonalAccessToken?: Maybe<PersonalAccessTokenCreated>;
  readonly createSQLQuery?: Maybe<SqlQueryCreated>;
  readonly createSchema?: Maybe<SchemaRefCreated>;
  readonly createSource: SourceCreated;
  readonly createSpace?: Maybe<SpaceCreated>;
  readonly createTable: TableRefCreated;
  readonly deleteCatalog: CatalogDeleted;
  readonly deleteColumn?: Maybe<ColumnRefDeleted>;
  readonly deleteConnection?: Maybe<ConnectionDeleted>;
  readonly deleteDestination: DestinationDeleted;
  readonly deleteEventSource?: Maybe<EventSourceDeleted>;
  readonly deleteFlow?: Maybe<FlowDeleted>;
  readonly deleteFlowRevision?: Maybe<FlowRevisionDeleted>;
  readonly deleteFlowRun?: Maybe<FlowRunDeleted>;
  readonly deleteGeoLayer?: Maybe<GeoLayerDeleted>;
  readonly deleteGeoMap?: Maybe<GeoMapDeleted>;
  readonly deleteModel?: Maybe<ModelDeleted>;
  readonly deleteOrganization?: Maybe<OrganizationDeleted>;
  readonly deletePersonalAccessToken?: Maybe<PersonalAccessTokenDeleted>;
  readonly deleteSQLQuery?: Maybe<SqlQueryDeleted>;
  readonly deleteSchema?: Maybe<SchemaRefDeleted>;
  readonly deleteSource: SourceDeleted;
  readonly deleteSpace?: Maybe<SpaceDeleted>;
  readonly deleteTable: TableRefDeleted;
  /** Remove a role for user */
  readonly removeRole?: Maybe<Role>;
  readonly rotatePersonalAccessToken?: Maybe<PersonalAccessTokenUpdated>;
  readonly startEventSource?: Maybe<EventSourceUpdated>;
  readonly startFlowRun?: Maybe<FlowRunUpdated>;
  readonly stopEventSource?: Maybe<EventSourceUpdated>;
  readonly stopFlowRun?: Maybe<FlowRunUpdated>;
  readonly syncCatalog: CatalogUpdated;
  readonly syncModel?: Maybe<ModelUpdated>;
  readonly syncPackage?: Maybe<IntegrationPackageUpdated>;
  readonly syncSchema?: Maybe<SchemaRefUpdated>;
  readonly syncSource: SourceUpdated;
  readonly syncTable: TableRefUpdated;
  readonly updateCatalog: CatalogUpdated;
  readonly updateConnection?: Maybe<ConnectionUpdated>;
  readonly updateDestination: DestinationUpdated;
  readonly updateEventSource?: Maybe<EventSourceUpdated>;
  readonly updateFlow?: Maybe<FlowUpdated>;
  readonly updateFlowRunUserAction?: Maybe<FlowRunNodeStatus>;
  readonly updateGeoLayer?: Maybe<GeoLayerUpdated>;
  readonly updateGeoMap?: Maybe<GeoMapUpdated>;
  readonly updateModel?: Maybe<ModelUpdated>;
  readonly updateOrganization?: Maybe<OrganizationUpdated>;
  readonly updatePersonalAccessToken?: Maybe<PersonalAccessTokenUpdated>;
  /** Update a role for user */
  readonly updateRole?: Maybe<Role>;
  readonly updateSQLQuery?: Maybe<SqlQueryUpdated>;
  readonly updateSchema?: Maybe<SchemaRefUpdated>;
  readonly updateSource: SourceUpdated;
  readonly updateSpace?: Maybe<SpaceUpdated>;
  readonly writeTable?: Maybe<WriteTableOutput>;
};


export type MutationAddRoleArgs = {
  input: AddRoleInput;
};


export type MutationCreateCatalogArgs = {
  input: CreateCatalogInput;
};


export type MutationCreateColumnArgs = {
  input: CreateColumnInput;
};


export type MutationCreateConnectionArgs = {
  input: CreateConnectionInput;
};


export type MutationCreateDestinationArgs = {
  input: CreateDestinationInput;
};


export type MutationCreateEventSourceArgs = {
  input: CreateEventSourceInput;
};


export type MutationCreateFlowArgs = {
  input: CreateFlowInput;
};


export type MutationCreateFlowRevisionArgs = {
  input: CreateFlowRevisionInput;
};


export type MutationCreateFlowRunArgs = {
  input: CreateFlowRunInput;
};


export type MutationCreateGeoLayerArgs = {
  input: CreateGeoLayerInput;
};


export type MutationCreateGeoMapArgs = {
  input: CreateGeoMapInput;
};


export type MutationCreateIntegrationArgs = {
  input: CreateIntegrationInput;
};


export type MutationCreateModelArgs = {
  input: CreateModelInput;
};


export type MutationCreateOrganizationArgs = {
  input: CreateOrganizationInput;
};


export type MutationCreatePersonalAccessTokenArgs = {
  input: CreatePersonalAccessTokenInput;
};


export type MutationCreateSqlQueryArgs = {
  input: CreateSqlQueryInput;
};


export type MutationCreateSchemaArgs = {
  input: CreateSchemaRefInput;
};


export type MutationCreateSourceArgs = {
  input: CreateSourceInput;
};


export type MutationCreateSpaceArgs = {
  input: CreateSpaceInput;
};


export type MutationCreateTableArgs = {
  input: CreateTableInput;
};


export type MutationDeleteCatalogArgs = {
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
};


export type MutationDeleteColumnArgs = {
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
};


export type MutationDeleteConnectionArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteDestinationArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteEventSourceArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteFlowArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteFlowRevisionArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteFlowRunArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteGeoLayerArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteGeoMapArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteModelArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteOrganizationArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeletePersonalAccessTokenArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  token?: InputMaybe<Scalars['String']['input']>;
};


export type MutationDeleteSqlQueryArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteSchemaArgs = {
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
};


export type MutationDeleteSourceArgs = {
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
};


export type MutationDeleteSpaceArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteTableArgs = {
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
};


export type MutationRemoveRoleArgs = {
  input: RemoveRoleInput;
};


export type MutationRotatePersonalAccessTokenArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  token?: InputMaybe<Scalars['String']['input']>;
};


export type MutationStartEventSourceArgs = {
  id: Scalars['ID']['input'];
};


export type MutationStartFlowRunArgs = {
  id: Scalars['ID']['input'];
  timeout?: InputMaybe<Scalars['String']['input']>;
};


export type MutationStopEventSourceArgs = {
  id: Scalars['ID']['input'];
};


export type MutationStopFlowRunArgs = {
  id: Scalars['ID']['input'];
  timeout?: InputMaybe<Scalars['String']['input']>;
};


export type MutationSyncCatalogArgs = {
  id: Scalars['ID']['input'];
};


export type MutationSyncModelArgs = {
  id: Scalars['ID']['input'];
};


export type MutationSyncPackageArgs = {
  input: SyncPackageInput;
};


export type MutationSyncSchemaArgs = {
  autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  id: Scalars['ID']['input'];
};


export type MutationSyncSourceArgs = {
  id: Scalars['ID']['input'];
};


export type MutationSyncTableArgs = {
  id: Scalars['ID']['input'];
};


export type MutationUpdateCatalogArgs = {
  id: Scalars['ID']['input'];
  input: UpdateCatalogInput;
};


export type MutationUpdateConnectionArgs = {
  id: Scalars['ID']['input'];
  input: UpdateConnectionInput;
};


export type MutationUpdateDestinationArgs = {
  id: Scalars['ID']['input'];
  input: UpdateDestinationInput;
};


export type MutationUpdateEventSourceArgs = {
  id: Scalars['ID']['input'];
  input: UpdateEventSourceInput;
};


export type MutationUpdateFlowArgs = {
  id: Scalars['ID']['input'];
  input: UpdateFlowInput;
};


export type MutationUpdateFlowRunUserActionArgs = {
  id: Scalars['ID']['input'];
  input: UpdateUserActionInput;
};


export type MutationUpdateGeoLayerArgs = {
  id: Scalars['ID']['input'];
  input: UpdateGeoLayerInput;
};


export type MutationUpdateGeoMapArgs = {
  id: Scalars['ID']['input'];
  input: UpdateGeoMapInput;
};


export type MutationUpdateModelArgs = {
  id: Scalars['ID']['input'];
  input: UpdateModelInput;
};


export type MutationUpdateOrganizationArgs = {
  id: Scalars['ID']['input'];
  input: UpdateOrganizationInput;
};


export type MutationUpdatePersonalAccessTokenArgs = {
  id: Scalars['ID']['input'];
  input: UpdatePersonalAccessTokenInput;
};


export type MutationUpdateRoleArgs = {
  input: UpdateRoleInput;
};


export type MutationUpdateSqlQueryArgs = {
  id: Scalars['ID']['input'];
  input: UpdateSqlQueryInput;
};


export type MutationUpdateSchemaArgs = {
  id: Scalars['ID']['input'];
  input: UpdateSchemaRefInput;
};


export type MutationUpdateSourceArgs = {
  id: Scalars['ID']['input'];
  input: UpdateSourceInput;
};


export type MutationUpdateSpaceArgs = {
  id: Scalars['ID']['input'];
  input: UpdateSpaceInput;
};


export type MutationWriteTableArgs = {
  id: Scalars['ID']['input'];
  input: WriteTableInput;
};

/**
 * An object with an ID.
 * Follows the [Relay Global Object Identification Specification](https://relay.dev/graphql/objectidentification.htm)
 */
export type Node = {
  /** The id of the object. */
  readonly id: Scalars['ID']['output'];
};

export type NodeAction = CatalogCreated | CatalogDeleted | CatalogUpdated | ColumnRefCreated | ColumnRefDeleted | ColumnRefUpdated | ConnectionCreated | ConnectionDeleted | ConnectionUpdated | ConnectionUserCreated | ConnectionUserDeleted | ConnectionUserUpdated | DestinationCreated | DestinationDeleted | DestinationUpdated | EventSourceCreated | EventSourceDeleted | EventSourceUpdated | FlowCreated | FlowDeleted | FlowResourceCreated | FlowResourceDeleted | FlowResourceUpdated | FlowRevisionCreated | FlowRevisionDeleted | FlowRevisionUpdated | FlowRunCreated | FlowRunDeleted | FlowRunUpdated | FlowUpdated | GeoFeatureCreated | GeoFeatureDeleted | GeoFeatureUpdated | GeoLayerCreated | GeoLayerDeleted | GeoLayerUpdated | GeoMapCreated | GeoMapDeleted | GeoMapUpdated | GeoSourceCreated | GeoSourceDeleted | GeoSourceUpdated | IoSchemaCreated | IoSchemaDeleted | IoSchemaUpdated | IntegrationCreated | IntegrationDeleted | IntegrationPackageCreated | IntegrationPackageDeleted | IntegrationPackageUpdated | IntegrationUpdated | ModelCreated | ModelDeleted | ModelTypeCreated | ModelTypeDeleted | ModelTypeUpdated | ModelUpdated | OrganizationCreated | OrganizationDeleted | OrganizationUpdated | PersonalAccessTokenCreated | PersonalAccessTokenDeleted | PersonalAccessTokenUpdated | SqlQueryCreated | SqlQueryDeleted | SqlQueryUpdated | SchemaRefCreated | SchemaRefDeleted | SchemaRefUpdated | SearchLexemeCreated | SearchLexemeDeleted | SearchLexemeUpdated | SearchSemanticCreated | SearchSemanticDeleted | SearchSemanticUpdated | SourceCreated | SourceDeleted | SourceTypeCreated | SourceTypeDeleted | SourceTypeUpdated | SourceUpdated | SpaceCreated | SpaceDeleted | SpaceUpdated | TableRefCreated | TableRefDeleted | TableRefUpdated | UserCreated | UserDeleted | UserUpdated;

export type NodeEvent = {
  readonly __typename?: 'NodeEvent';
  readonly action?: Maybe<NodeAction>;
  readonly actionType: ActionType;
  readonly eventType: Scalars['String']['output'];
  readonly metadata?: Maybe<Scalars['Map']['output']>;
  readonly node?: Maybe<Node>;
  readonly nodeID: Scalars['ID']['output'];
  readonly nodeType: Scalars['String']['output'];
};

export type NodeSubscribeInput = {
  /** If true, the subscription applies to all child resources of the given node type & id(s). */
  readonly isWildcard?: InputMaybe<Scalars['Boolean']['input']>;
  readonly nodeIDs: ReadonlyArray<Scalars['ID']['input']>;
  readonly nodeType: Scalars['String']['input'];
};

/** Possible directions in which to order a list of items when provided an `orderBy` argument. */
export enum OrderDirection {
  /** Specifies an ascending order for a given `orderBy` argument. */
  Asc = 'ASC',
  /** Specifies a descending order for a given `orderBy` argument. */
  Desc = 'DESC'
}

export type Organization = Node & {
  readonly __typename?: 'Organization';
  readonly catalogs: CatalogConnection;
  readonly connectionUsers: ConnectionUserConnection;
  readonly connections: ConnectionConnection;
  readonly connectionsOwned: ConnectionConnection;
  readonly createdAt: Scalars['Time']['output'];
  readonly customerID: Scalars['String']['output'];
  readonly destinations: DestinationConnection;
  readonly flows: FlowConnection;
  readonly geoLayers: GeoLayerConnection;
  readonly geoMaps: GeoMapConnection;
  readonly id: Scalars['ID']['output'];
  readonly integrations: IntegrationConnection;
  readonly name: Scalars['String']['output'];
  readonly slug: Scalars['String']['output'];
  readonly spaces: SpaceConnection;
  /** Timezone name (e.g. America/Los_Angeles) */
  readonly tzName: Scalars['String']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};


export type OrganizationCatalogsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<CatalogOrder>;
  where?: InputMaybe<CatalogWhereInput>;
};


export type OrganizationConnectionUsersArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ConnectionUserOrder>;
  where?: InputMaybe<ConnectionUserWhereInput>;
};


export type OrganizationConnectionsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ConnectionOrder>;
  where?: InputMaybe<ConnectionWhereInput>;
};


export type OrganizationConnectionsOwnedArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ConnectionOrder>;
  where?: InputMaybe<ConnectionWhereInput>;
};


export type OrganizationDestinationsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<DestinationOrder>;
  where?: InputMaybe<DestinationWhereInput>;
};


export type OrganizationFlowsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowOrder>;
  where?: InputMaybe<FlowWhereInput>;
};


export type OrganizationGeoLayersArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<GeoLayerOrder>;
  where?: InputMaybe<GeoLayerWhereInput>;
};


export type OrganizationGeoMapsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<GeoMapOrder>;
  where?: InputMaybe<GeoMapWhereInput>;
};


export type OrganizationIntegrationsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<IntegrationOrder>;
  where?: InputMaybe<IntegrationWhereInput>;
};


export type OrganizationSpacesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SpaceOrder>;
  where?: InputMaybe<SpaceWhereInput>;
};

/** A connection to a list of items. */
export type OrganizationConnection = {
  readonly __typename?: 'OrganizationConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<OrganizationEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type OrganizationCreated = {
  readonly __typename?: 'OrganizationCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly organization?: Maybe<Organization>;
};

export type OrganizationDeleted = {
  readonly __typename?: 'OrganizationDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type OrganizationEdge = {
  readonly __typename?: 'OrganizationEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<Organization>;
};

/** Ordering options for Organization connections */
export type OrganizationOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Organizations. */
  readonly field: OrganizationOrderField;
};

/** Properties by which Organization connections can be ordered. */
export enum OrganizationOrderField {
  CatalogsCount = 'CATALOGS_COUNT',
  CreatedAt = 'CREATED_AT',
  DestinationsCount = 'DESTINATIONS_COUNT',
  FlowsCount = 'FLOWS_COUNT',
  IntegrationsCount = 'INTEGRATIONS_COUNT',
  Name = 'NAME',
  SpacesCount = 'SPACES_COUNT',
  UpdatedAt = 'UPDATED_AT'
}

export type OrganizationUpdated = {
  readonly __typename?: 'OrganizationUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly organization?: Maybe<Organization>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * OrganizationWhereInput is used for filtering Organization objects.
 * Input was generated by ent.
 */
export type OrganizationWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** customer_id field predicates */
  readonly customerID?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDContains?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDGT?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDGTE?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly customerIDLT?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDLTE?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly customerIDNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** catalogs edge predicates */
  readonly hasCatalogs?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasCatalogsWith?: InputMaybe<ReadonlyArray<CatalogWhereInput>>;
  /** connection_users edge predicates */
  readonly hasConnectionUsers?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionUsersWith?: InputMaybe<ReadonlyArray<ConnectionUserWhereInput>>;
  /** connections edge predicates */
  readonly hasConnections?: InputMaybe<Scalars['Boolean']['input']>;
  /** connections_owned edge predicates */
  readonly hasConnectionsOwned?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionsOwnedWith?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  readonly hasConnectionsWith?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** destinations edge predicates */
  readonly hasDestinations?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasDestinationsWith?: InputMaybe<ReadonlyArray<DestinationWhereInput>>;
  /** flows edge predicates */
  readonly hasFlows?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasFlowsWith?: InputMaybe<ReadonlyArray<FlowWhereInput>>;
  /** geo_layers edge predicates */
  readonly hasGeoLayers?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasGeoLayersWith?: InputMaybe<ReadonlyArray<GeoLayerWhereInput>>;
  /** geo_maps edge predicates */
  readonly hasGeoMaps?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasGeoMapsWith?: InputMaybe<ReadonlyArray<GeoMapWhereInput>>;
  /** integrations edge predicates */
  readonly hasIntegrations?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasIntegrationsWith?: InputMaybe<ReadonlyArray<IntegrationWhereInput>>;
  /** spaces edge predicates */
  readonly hasSpaces?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSpacesWith?: InputMaybe<ReadonlyArray<SpaceWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<OrganizationWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** tz_name field predicates */
  readonly tzName?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameContains?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameGT?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly tzNameLT?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

/**
 * Information about pagination in a connection.
 * https://relay.dev/graphql/connections.htm#sec-undefined.PageInfo
 */
export type PageInfo = {
  readonly __typename?: 'PageInfo';
  /** When paginating forwards, the cursor to continue. */
  readonly endCursor?: Maybe<Scalars['Cursor']['output']>;
  /** When paginating forwards, are there more items? */
  readonly hasNextPage: Scalars['Boolean']['output'];
  /** When paginating backwards, are there more items? */
  readonly hasPreviousPage: Scalars['Boolean']['output'];
  /** When paginating backwards, the cursor to continue. */
  readonly startCursor?: Maybe<Scalars['Cursor']['output']>;
};

export type Param = {
  readonly __typename?: 'Param';
  readonly field: Field;
  readonly value?: Maybe<Scalars['Any']['output']>;
};

export type ParamInput = {
  readonly field: FieldInput;
  readonly value?: InputMaybe<Scalars['Any']['input']>;
};

/** Permission is a type that represents the permissions a role or user has on a resource. */
export type Permission = {
  readonly __typename?: 'Permission';
  readonly actions: ReadonlyArray<ActionType>;
  readonly resource: Node;
  readonly resourcePath: Scalars['String']['output'];
};

export type PersonalAccessToken = Node & {
  readonly __typename?: 'PersonalAccessToken';
  readonly createdAt: Scalars['Time']['output'];
  readonly expiresAt: Scalars['Time']['output'];
  readonly id: Scalars['ID']['output'];
  readonly name: Scalars['String']['output'];
  readonly token: Scalars['String']['output'];
  readonly updatedAt: Scalars['Time']['output'];
  readonly user: User;
  readonly userID: Scalars['ID']['output'];
};

/** A connection to a list of items. */
export type PersonalAccessTokenConnection = {
  readonly __typename?: 'PersonalAccessTokenConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<PersonalAccessTokenEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type PersonalAccessTokenCreated = {
  readonly __typename?: 'PersonalAccessTokenCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly personalAccessToken?: Maybe<PersonalAccessToken>;
};

export type PersonalAccessTokenDeleted = {
  readonly __typename?: 'PersonalAccessTokenDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type PersonalAccessTokenEdge = {
  readonly __typename?: 'PersonalAccessTokenEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<PersonalAccessToken>;
};

/** Ordering options for PersonalAccessToken connections */
export type PersonalAccessTokenOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order PersonalAccessTokens. */
  readonly field: PersonalAccessTokenOrderField;
};

/** Properties by which PersonalAccessToken connections can be ordered. */
export enum PersonalAccessTokenOrderField {
  CreatedAt = 'CREATED_AT',
  ExpiresAt = 'EXPIRES_AT',
  Name = 'NAME',
  UpdatedAt = 'UPDATED_AT'
}

export type PersonalAccessTokenUpdated = {
  readonly __typename?: 'PersonalAccessTokenUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly personalAccessToken?: Maybe<PersonalAccessToken>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * PersonalAccessTokenWhereInput is used for filtering PersonalAccessToken objects.
 * Input was generated by ent.
 */
export type PersonalAccessTokenWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<PersonalAccessTokenWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** expires_at field predicates */
  readonly expiresAt?: InputMaybe<Scalars['Time']['input']>;
  readonly expiresAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly expiresAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly expiresAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly expiresAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly expiresAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly expiresAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly expiresAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** user edge predicates */
  readonly hasUser?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasUserWith?: InputMaybe<ReadonlyArray<UserWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<PersonalAccessTokenWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<PersonalAccessTokenWhereInput>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** user_id field predicates */
  readonly userID?: InputMaybe<Scalars['ID']['input']>;
  readonly userIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly userIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly userIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
};

export type Query = {
  readonly __typename?: 'Query';
  readonly catalog?: Maybe<Catalog>;
  readonly catalogs: CatalogConnection;
  readonly checkConnection?: Maybe<ConnectionCheck>;
  readonly column?: Maybe<ColumnRef>;
  readonly columns: ColumnRefConnection;
  readonly columnsDeleted: ReadonlyArray<Maybe<ColumnRef>>;
  readonly connection?: Maybe<Connection>;
  readonly connections: ConnectionConnection;
  readonly destination?: Maybe<Destination>;
  readonly destinations: DestinationConnection;
  readonly eventSource?: Maybe<EventSource>;
  readonly eventSources: EventSourceConnection;
  readonly flow?: Maybe<Flow>;
  readonly flowResources: FlowResourceConnection;
  readonly flowRevision?: Maybe<FlowRevision>;
  readonly flowRevisions: FlowRevisionConnection;
  readonly flowRun?: Maybe<FlowRun>;
  readonly flowRuns: FlowRunConnection;
  readonly flows: FlowConnection;
  readonly geoFeatures: GeoFeatureConnection;
  readonly geoLayer?: Maybe<GeoLayer>;
  readonly geoLayers: GeoLayerConnection;
  readonly geoMap?: Maybe<GeoMap>;
  readonly geoMaps: GeoMapConnection;
  readonly geoSources: GeoSourceConnection;
  readonly integration?: Maybe<Integration>;
  readonly integrations: IntegrationConnection;
  readonly ioSchema?: Maybe<IoSchema>;
  readonly ioSchemas: IoSchemaConnection;
  readonly model?: Maybe<Model>;
  readonly modelType?: Maybe<ModelType>;
  readonly modelTypes: ModelTypeConnection;
  readonly models: ModelConnection;
  /** Fetches an object given its ID. */
  readonly node?: Maybe<Node>;
  /** Lookup nodes by a list of IDs. */
  readonly nodes: ReadonlyArray<Maybe<Node>>;
  readonly organization?: Maybe<Organization>;
  readonly organizations: OrganizationConnection;
  readonly packages: IntegrationPackageConnection;
  /** Permissions for the current or given user */
  readonly permissions: ReadonlyArray<Maybe<Permission>>;
  readonly personalAccessToken?: Maybe<PersonalAccessToken>;
  readonly personalAccessTokens: PersonalAccessTokenConnection;
  readonly readTable?: Maybe<ReadTableOutput>;
  /** Roles for the given resource */
  readonly resourceRoles: ReadonlyArray<Maybe<Role>>;
  readonly schema?: Maybe<SchemaRef>;
  readonly schemas: SchemaRefConnection;
  readonly schemasDeleted: ReadonlyArray<Maybe<SchemaRef>>;
  readonly search: SearchResults;
  readonly source?: Maybe<Source>;
  readonly sourceType?: Maybe<SourceType>;
  readonly sourceTypes: SourceTypeConnection;
  readonly sources: SourceConnection;
  readonly sourcesDeleted: ReadonlyArray<Maybe<Source>>;
  readonly space?: Maybe<Space>;
  /** Roles for the the current or given space */
  readonly spaceRoles: ReadonlyArray<Maybe<Role>>;
  readonly spaces: SpaceConnection;
  readonly sqlQueries: SqlQueryConnection;
  readonly sqlQuery?: Maybe<SqlQuery>;
  readonly sqlQueryValidation?: Maybe<SqlQueryValidation>;
  readonly table?: Maybe<TableRef>;
  readonly tables: TableRefConnection;
  readonly tablesDeleted: ReadonlyArray<Maybe<TableRef>>;
  /** Roles for the current or given user */
  readonly userRoles: ReadonlyArray<Maybe<Role>>;
};


export type QueryCatalogArgs = {
  id: Scalars['ID']['input'];
};


export type QueryCatalogsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<CatalogOrder>;
  where?: InputMaybe<CatalogWhereInput>;
};


export type QueryCheckConnectionArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QueryColumnArgs = {
  id: Scalars['ID']['input'];
};


export type QueryColumnsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ReadonlyArray<ColumnRefOrder>>;
  where?: InputMaybe<ColumnRefWhereInput>;
};


export type QueryConnectionArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QueryConnectionsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ConnectionOrder>;
  where?: InputMaybe<ConnectionWhereInput>;
};


export type QueryDestinationArgs = {
  id: Scalars['ID']['input'];
};


export type QueryDestinationsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<DestinationOrder>;
  where?: InputMaybe<DestinationWhereInput>;
};


export type QueryEventSourceArgs = {
  id: Scalars['ID']['input'];
};


export type QueryEventSourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<EventSourceOrder>;
  where?: InputMaybe<EventSourceWhereInput>;
};


export type QueryFlowArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QueryFlowResourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowResourceOrder>;
  where?: InputMaybe<FlowResourceWhereInput>;
};


export type QueryFlowRevisionArgs = {
  id: Scalars['ID']['input'];
};


export type QueryFlowRevisionsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowRevisionOrder>;
  where?: InputMaybe<FlowRevisionWhereInput>;
};


export type QueryFlowRunArgs = {
  id: Scalars['ID']['input'];
};


export type QueryFlowRunsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowRunOrder>;
  where?: InputMaybe<FlowRunWhereInput>;
};


export type QueryFlowsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowOrder>;
  where?: InputMaybe<FlowWhereInput>;
};


export type QueryGeoFeaturesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<GeoFeatureOrder>;
  where?: InputMaybe<GeoFeatureWhereInput>;
};


export type QueryGeoLayerArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QueryGeoLayersArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<GeoLayerOrder>;
  where?: InputMaybe<GeoLayerWhereInput>;
};


export type QueryGeoMapArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QueryGeoMapsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<GeoMapOrder>;
  where?: InputMaybe<GeoMapWhereInput>;
};


export type QueryGeoSourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<GeoSourceOrder>;
  where?: InputMaybe<GeoSourceWhereInput>;
};


export type QueryIntegrationArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QueryIntegrationsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<IntegrationOrder>;
  where?: InputMaybe<IntegrationWhereInput>;
};


export type QueryIoSchemaArgs = {
  id: Scalars['ID']['input'];
  type: Scalars['String']['input'];
};


export type QueryIoSchemasArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<IoSchemaOrder>;
  where?: InputMaybe<IoSchemaWhereInput>;
};


export type QueryModelArgs = {
  id: Scalars['ID']['input'];
};


export type QueryModelTypeArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QueryModelTypesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ModelTypeOrder>;
  where?: InputMaybe<ModelTypeWhereInput>;
};


export type QueryModelsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ModelOrder>;
  where?: InputMaybe<ModelWhereInput>;
};


export type QueryNodeArgs = {
  id: Scalars['ID']['input'];
};


export type QueryNodesArgs = {
  ids: ReadonlyArray<Scalars['ID']['input']>;
};


export type QueryOrganizationArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QueryOrganizationsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<OrganizationOrder>;
  where?: InputMaybe<OrganizationWhereInput>;
};


export type QueryPackagesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<IntegrationPackageOrder>;
  where?: InputMaybe<IntegrationPackageWhereInput>;
};


export type QueryPermissionsArgs = {
  userID?: InputMaybe<Scalars['ID']['input']>;
};


export type QueryPersonalAccessTokenArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  token?: InputMaybe<Scalars['String']['input']>;
};


export type QueryPersonalAccessTokensArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<PersonalAccessTokenOrder>;
  where?: InputMaybe<PersonalAccessTokenWhereInput>;
};


export type QueryReadTableArgs = {
  id: Scalars['ID']['input'];
  input: ReadTableInput;
};


export type QueryResourceRolesArgs = {
  id: Scalars['ID']['input'];
  type: Scalars['String']['input'];
};


export type QuerySchemaArgs = {
  id: Scalars['ID']['input'];
};


export type QuerySchemasArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ReadonlyArray<SchemaRefOrder>>;
  where?: InputMaybe<SchemaRefWhereInput>;
};


export type QuerySearchArgs = {
  by?: InputMaybe<SearchByInput>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  q?: InputMaybe<Scalars['String']['input']>;
  type?: InputMaybe<SearchType>;
  view?: InputMaybe<SearchView>;
  where?: InputMaybe<SearchWhereInput>;
};


export type QuerySourceArgs = {
  id: Scalars['ID']['input'];
};


export type QuerySourceTypeArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QuerySourceTypesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SourceTypeOrder>;
  where?: InputMaybe<SourceTypeWhereInput>;
};


export type QuerySourcesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SourceOrder>;
  where?: InputMaybe<SourceWhereInput>;
};


export type QuerySpaceArgs = {
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
};


export type QuerySpaceRolesArgs = {
  spaceID?: InputMaybe<Scalars['ID']['input']>;
};


export type QuerySpacesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SpaceOrder>;
  where?: InputMaybe<SpaceWhereInput>;
};


export type QuerySqlQueriesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SqlQueryOrder>;
  where?: InputMaybe<SqlQueryWhereInput>;
};


export type QuerySqlQueryArgs = {
  id: Scalars['ID']['input'];
};


export type QuerySqlQueryValidationArgs = {
  input: SqlQueryValidationInput;
};


export type QueryTableArgs = {
  id: Scalars['ID']['input'];
};


export type QueryTablesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ReadonlyArray<TableRefOrder>>;
  where?: InputMaybe<TableRefWhereInput>;
};


export type QueryUserRolesArgs = {
  userID?: InputMaybe<Scalars['ID']['input']>;
};

export type ReadTableInput = {
  readonly fields?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly limit?: InputMaybe<Scalars['Int64']['input']>;
  readonly offset?: InputMaybe<Scalars['Int64']['input']>;
};

export type ReadTableOutput = {
  readonly __typename?: 'ReadTableOutput';
  readonly rows: ReadonlyArray<Scalars['Map']['output']>;
  readonly table: TableRef;
};

/** RemoveRoleInput is used for removing a role from a User for the given resource. */
export type RemoveRoleInput = {
  readonly orgID: Scalars['ID']['input'];
  readonly spaceID?: InputMaybe<Scalars['ID']['input']>;
  readonly type: RoleType;
};

export enum ReportProvider {
  LookerStudio = 'LOOKER_STUDIO'
}

/** Role is a type that represents the role a user has in an organization and/or space. */
export type Role = {
  readonly __typename?: 'Role';
  readonly id: Scalars['String']['output'];
  readonly orgID: Scalars['ID']['output'];
  readonly permissions: ReadonlyArray<Maybe<Permission>>;
  readonly spaceID?: Maybe<Scalars['ID']['output']>;
  readonly type: RoleType;
  readonly users: ReadonlyArray<Maybe<User>>;
};

/** RoleResourceInput is used for adding a RBACpermission to a Role for the given resource. */
export type RoleResourceInput = {
  readonly hasWildcard?: InputMaybe<Scalars['Boolean']['input']>;
  readonly id: Scalars['ID']['input'];
  readonly type: Scalars['String']['input'];
};

/** RoleType is enum for the type of role that can be assigned to users. */
export enum RoleType {
  Editor = 'EDITOR',
  Owner = 'OWNER',
  Reader = 'READER',
  Viewer = 'VIEWER',
  Writer = 'WRITER'
}

export type SqlQuery = Node & {
  readonly __typename?: 'SQLQuery';
  readonly catalogs: ReadonlyArray<Catalog>;
  readonly columns: ColumnRefConnection;
  readonly connectionID?: Maybe<Scalars['ID']['output']>;
  readonly createdAt: Scalars['Time']['output'];
  readonly dialect: Scalars['String']['output'];
  readonly fields?: Maybe<ReadonlyArray<Maybe<Field>>>;
  readonly id: Scalars['ID']['output'];
  readonly ioSchema?: Maybe<IoSchema>;
  readonly metadata?: Maybe<Scalars['Map']['output']>;
  readonly params?: Maybe<ReadonlyArray<Maybe<Param>>>;
  readonly results?: Maybe<SqlQueryOutput>;
  readonly schemas: ReadonlyArray<SchemaRef>;
  readonly space: Space;
  readonly spaceID: Scalars['ID']['output'];
  readonly sql: Scalars['String']['output'];
  readonly sqlCanonical: Scalars['String']['output'];
  readonly tables: ReadonlyArray<TableRef>;
  readonly updatedAt: Scalars['Time']['output'];
  readonly user: User;
  readonly userID: Scalars['ID']['output'];
};


export type SqlQueryColumnsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ReadonlyArray<ColumnRefOrder>>;
  where?: InputMaybe<ColumnRefWhereInput>;
};


export type SqlQueryResultsArgs = {
  input?: InputMaybe<SqlQueryInput>;
};

/** A connection to a list of items. */
export type SqlQueryConnection = {
  readonly __typename?: 'SQLQueryConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<SqlQueryEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type SqlQueryCreated = {
  readonly __typename?: 'SQLQueryCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly sqlQuery?: Maybe<SqlQuery>;
};

export type SqlQueryDeleted = {
  readonly __typename?: 'SQLQueryDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type SqlQueryEdge = {
  readonly __typename?: 'SQLQueryEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<SqlQuery>;
};

export type SqlQueryInput = {
  readonly page?: InputMaybe<Scalars['String']['input']>;
  readonly pageSize?: InputMaybe<Scalars['Int']['input']>;
  readonly params?: InputMaybe<Scalars['Map']['input']>;
};

/** Ordering options for SQLQuery connections */
export type SqlQueryOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order SQLQueries. */
  readonly field: SqlQueryOrderField;
};

/** Properties by which SQLQuery connections can be ordered. */
export enum SqlQueryOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type SqlQueryOutput = {
  readonly __typename?: 'SQLQueryOutput';
  readonly fields?: Maybe<ReadonlyArray<Field>>;
  readonly nextPage?: Maybe<Scalars['String']['output']>;
  readonly params?: Maybe<ReadonlyArray<Param>>;
  readonly prevPage?: Maybe<Scalars['String']['output']>;
  readonly rows: ReadonlyArray<Maybe<Scalars['Map']['output']>>;
  readonly rowsCount: Scalars['Int']['output'];
  readonly totalPages: Scalars['Int']['output'];
  readonly totalRows: Scalars['Int']['output'];
};

export type SqlQueryUpdated = {
  readonly __typename?: 'SQLQueryUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly sqlQuery?: Maybe<SqlQuery>;
  readonly updated: Scalars['Boolean']['output'];
};

export type SqlQueryValidation = {
  readonly __typename?: 'SQLQueryValidation';
  readonly columnIDs?: Maybe<ReadonlyArray<Scalars['ID']['output']>>;
  readonly connectionID?: Maybe<Scalars['ID']['output']>;
  readonly dialect: Scalars['String']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly fields?: Maybe<ReadonlyArray<Field>>;
  readonly isValid: Scalars['Boolean']['output'];
  readonly params?: Maybe<ReadonlyArray<Param>>;
  readonly spaceID: Scalars['ID']['output'];
  readonly sql: Scalars['String']['output'];
};

export type SqlQueryValidationInput = {
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  readonly dialect: Scalars['String']['input'];
  readonly params?: InputMaybe<ReadonlyArray<ParamInput>>;
  readonly spaceID: Scalars['ID']['input'];
  readonly sql: Scalars['String']['input'];
};

/**
 * SQLQueryWhereInput is used for filtering SQLQuery objects.
 * Input was generated by ent.
 */
export type SqlQueryWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<SqlQueryWhereInput>>;
  /** connection_id field predicates */
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly connectionIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly connectionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly connectionIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** dialect field predicates */
  readonly dialect?: InputMaybe<Scalars['String']['input']>;
  readonly dialectContains?: InputMaybe<Scalars['String']['input']>;
  readonly dialectContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly dialectEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly dialectGT?: InputMaybe<Scalars['String']['input']>;
  readonly dialectGTE?: InputMaybe<Scalars['String']['input']>;
  readonly dialectHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly dialectHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly dialectIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly dialectLT?: InputMaybe<Scalars['String']['input']>;
  readonly dialectLTE?: InputMaybe<Scalars['String']['input']>;
  readonly dialectNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly dialectNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** columns edge predicates */
  readonly hasColumns?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasColumnsWith?: InputMaybe<ReadonlyArray<ColumnRefWhereInput>>;
  /** io_schema edge predicates */
  readonly hasIoSchema?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasIoSchemaWith?: InputMaybe<ReadonlyArray<IoSchemaWhereInput>>;
  /** space edge predicates */
  readonly hasSpace?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSpaceWith?: InputMaybe<ReadonlyArray<SpaceWhereInput>>;
  /** user edge predicates */
  readonly hasUser?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasUserWith?: InputMaybe<ReadonlyArray<UserWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly not?: InputMaybe<SqlQueryWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<SqlQueryWhereInput>>;
  /** space_id field predicates */
  readonly spaceID?: InputMaybe<Scalars['ID']['input']>;
  readonly spaceIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly spaceIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly spaceIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** sql field predicates */
  readonly sql?: InputMaybe<Scalars['String']['input']>;
  /** sql_canonical field predicates */
  readonly sqlCanonical?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalContains?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalGT?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalGTE?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly sqlCanonicalLT?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalLTE?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly sqlCanonicalNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly sqlContains?: InputMaybe<Scalars['String']['input']>;
  readonly sqlContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly sqlEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly sqlGT?: InputMaybe<Scalars['String']['input']>;
  readonly sqlGTE?: InputMaybe<Scalars['String']['input']>;
  readonly sqlHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly sqlHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly sqlIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly sqlLT?: InputMaybe<Scalars['String']['input']>;
  readonly sqlLTE?: InputMaybe<Scalars['String']['input']>;
  readonly sqlNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly sqlNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** user_id field predicates */
  readonly userID?: InputMaybe<Scalars['ID']['input']>;
  readonly userIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly userIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly userIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
};

export type SchemaRef = Node & {
  readonly __typename?: 'SchemaRef';
  readonly alias: Scalars['String']['output'];
  readonly autoSync: Scalars['Boolean']['output'];
  readonly catalog: Catalog;
  readonly catalogID: Scalars['ID']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly deletedAt?: Maybe<Scalars['Time']['output']>;
  readonly description?: Maybe<Scalars['String']['output']>;
  readonly id: Scalars['ID']['output'];
  readonly model?: Maybe<Model>;
  readonly modelID?: Maybe<Scalars['ID']['output']>;
  readonly name: Scalars['String']['output'];
  readonly source?: Maybe<Source>;
  readonly sourceID?: Maybe<Scalars['ID']['output']>;
  readonly syncError?: Maybe<Scalars['String']['output']>;
  readonly syncStatus: SyncStatus;
  readonly syncedAt?: Maybe<Scalars['Time']['output']>;
  readonly tables: TableRefConnection;
  readonly updatedAt: Scalars['Time']['output'];
};


export type SchemaRefTablesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ReadonlyArray<TableRefOrder>>;
  where?: InputMaybe<TableRefWhereInput>;
};

/** A connection to a list of items. */
export type SchemaRefConnection = {
  readonly __typename?: 'SchemaRefConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<SchemaRefEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type SchemaRefCreated = {
  readonly __typename?: 'SchemaRefCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly schemaRef?: Maybe<SchemaRef>;
};

export type SchemaRefDeleted = {
  readonly __typename?: 'SchemaRefDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type SchemaRefEdge = {
  readonly __typename?: 'SchemaRefEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<SchemaRef>;
};

/** Ordering options for SchemaRef connections */
export type SchemaRefOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order SchemaRefs. */
  readonly field: SchemaRefOrderField;
};

/** Properties by which SchemaRef connections can be ordered. */
export enum SchemaRefOrderField {
  Alias = 'ALIAS',
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  TablesCount = 'TABLES_COUNT',
  UpdatedAt = 'UPDATED_AT'
}

export type SchemaRefUpdated = {
  readonly __typename?: 'SchemaRefUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly schemaRef?: Maybe<SchemaRef>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * SchemaRefWhereInput is used for filtering SchemaRef objects.
 * Input was generated by ent.
 */
export type SchemaRefWhereInput = {
  /** alias field predicates */
  readonly alias?: InputMaybe<Scalars['String']['input']>;
  readonly aliasContains?: InputMaybe<Scalars['String']['input']>;
  readonly aliasContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly aliasEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly aliasGT?: InputMaybe<Scalars['String']['input']>;
  readonly aliasGTE?: InputMaybe<Scalars['String']['input']>;
  readonly aliasHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly aliasHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly aliasIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly aliasLT?: InputMaybe<Scalars['String']['input']>;
  readonly aliasLTE?: InputMaybe<Scalars['String']['input']>;
  readonly aliasNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly aliasNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly and?: InputMaybe<ReadonlyArray<SchemaRefWhereInput>>;
  /** auto_sync field predicates */
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly autoSyncNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** catalog_id field predicates */
  readonly catalogID?: InputMaybe<Scalars['ID']['input']>;
  readonly catalogIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly catalogIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly catalogIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** deleted_at field predicates */
  readonly deletedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly deletedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** catalog edge predicates */
  readonly hasCatalog?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasCatalogWith?: InputMaybe<ReadonlyArray<CatalogWhereInput>>;
  /** model edge predicates */
  readonly hasModel?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasModelWith?: InputMaybe<ReadonlyArray<ModelWhereInput>>;
  /** source edge predicates */
  readonly hasSource?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSourceWith?: InputMaybe<ReadonlyArray<SourceWhereInput>>;
  /** tables edge predicates */
  readonly hasTables?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasTablesWith?: InputMaybe<ReadonlyArray<TableRefWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** model_id field predicates */
  readonly modelID?: InputMaybe<Scalars['ID']['input']>;
  readonly modelIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly modelIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly modelIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly modelIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly modelIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<SchemaRefWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<SchemaRefWhereInput>>;
  /** source_id field predicates */
  readonly sourceID?: InputMaybe<Scalars['ID']['input']>;
  readonly sourceIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly sourceIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly sourceIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly sourceIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly sourceIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** sync_error field predicates */
  readonly syncError?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContains?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncErrorLT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorLTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** sync_status field predicates */
  readonly syncStatus?: InputMaybe<SyncStatus>;
  readonly syncStatusIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  readonly syncStatusNEQ?: InputMaybe<SyncStatus>;
  readonly syncStatusNotIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  /** synced_at field predicates */
  readonly syncedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type SearchByInput = {
  /** Search by node id. */
  readonly id: Scalars['ID']['input'];
  /** Include resource in search results, result rank will be 1.0. (Default: false) */
  readonly includeResult?: InputMaybe<Scalars['Boolean']['input']>;
  /** Search by node type. */
  readonly type: Scalars['String']['input'];
};

export type SearchLexeme = Node & {
  readonly __typename?: 'SearchLexeme';
  readonly createdAt: Scalars['Time']['output'];
  readonly id: Scalars['ID']['output'];
  readonly nodeID: Scalars['ID']['output'];
  readonly nodeType: Scalars['String']['output'];
  readonly organization?: Maybe<Organization>;
  readonly organizationID?: Maybe<Scalars['ID']['output']>;
  readonly searchText: Scalars['String']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};

/** A connection to a list of items. */
export type SearchLexemeConnection = {
  readonly __typename?: 'SearchLexemeConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<SearchLexemeEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type SearchLexemeCreated = {
  readonly __typename?: 'SearchLexemeCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly searchLexeme?: Maybe<SearchLexeme>;
};

export type SearchLexemeDeleted = {
  readonly __typename?: 'SearchLexemeDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type SearchLexemeEdge = {
  readonly __typename?: 'SearchLexemeEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<SearchLexeme>;
};

/** Ordering options for SearchLexeme connections */
export type SearchLexemeOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order SearchLexemes. */
  readonly field: SearchLexemeOrderField;
};

/** Properties by which SearchLexeme connections can be ordered. */
export enum SearchLexemeOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type SearchLexemeUpdated = {
  readonly __typename?: 'SearchLexemeUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly searchLexeme?: Maybe<SearchLexeme>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * SearchLexemeWhereInput is used for filtering SearchLexeme objects.
 * Input was generated by ent.
 */
export type SearchLexemeWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<SearchLexemeWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** node_id field predicates */
  readonly nodeID?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDGT?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly nodeIDLT?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** node_type field predicates */
  readonly nodeType?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeContains?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeGT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nodeTypeLT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<SearchLexemeWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<SearchLexemeWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** search_text field predicates */
  readonly searchText?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextContains?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextGT?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextGTE?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly searchTextLT?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextLTE?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type SearchResult = {
  readonly __typename?: 'SearchResult';
  /** The result node. */
  readonly node: Node;
  /** The score of the search result. */
  readonly rank: Scalars['Float']['output'];
  /** The textual representation of the node. */
  readonly text: Scalars['String']['output'];
};

export type SearchResults = {
  readonly __typename?: 'SearchResults';
  /** The search results. */
  readonly results: ReadonlyArray<Maybe<SearchResult>>;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type SearchSemantic = Node & {
  readonly __typename?: 'SearchSemantic';
  readonly autoSync: Scalars['Boolean']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly id: Scalars['ID']['output'];
  readonly nodeID: Scalars['ID']['output'];
  readonly nodeType: Scalars['String']['output'];
  readonly organization?: Maybe<Organization>;
  readonly organizationID?: Maybe<Scalars['ID']['output']>;
  readonly searchText: Scalars['String']['output'];
  readonly syncError?: Maybe<Scalars['String']['output']>;
  readonly syncStatus: SyncStatus;
  readonly syncedAt?: Maybe<Scalars['Time']['output']>;
  readonly updatedAt: Scalars['Time']['output'];
};

/** A connection to a list of items. */
export type SearchSemanticConnection = {
  readonly __typename?: 'SearchSemanticConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<SearchSemanticEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type SearchSemanticCreated = {
  readonly __typename?: 'SearchSemanticCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly searchSemantic?: Maybe<SearchSemantic>;
};

export type SearchSemanticDeleted = {
  readonly __typename?: 'SearchSemanticDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type SearchSemanticEdge = {
  readonly __typename?: 'SearchSemanticEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<SearchSemantic>;
};

/** Ordering options for SearchSemantic connections */
export type SearchSemanticOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order SearchSemantics. */
  readonly field: SearchSemanticOrderField;
};

/** Properties by which SearchSemantic connections can be ordered. */
export enum SearchSemanticOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type SearchSemanticUpdated = {
  readonly __typename?: 'SearchSemanticUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly searchSemantic?: Maybe<SearchSemantic>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * SearchSemanticWhereInput is used for filtering SearchSemantic objects.
 * Input was generated by ent.
 */
export type SearchSemanticWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<SearchSemanticWhereInput>>;
  /** auto_sync field predicates */
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly autoSyncNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** node_id field predicates */
  readonly nodeID?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDGT?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly nodeIDLT?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** node_type field predicates */
  readonly nodeType?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeContains?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeGT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nodeTypeLT?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<SearchSemanticWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<SearchSemanticWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** search_text field predicates */
  readonly searchText?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextContains?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextGT?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextGTE?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly searchTextLT?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextLTE?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly searchTextNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** sync_error field predicates */
  readonly syncError?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContains?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncErrorLT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorLTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** sync_status field predicates */
  readonly syncStatus?: InputMaybe<SyncStatus>;
  readonly syncStatusIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  readonly syncStatusNEQ?: InputMaybe<SyncStatus>;
  readonly syncStatusNotIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  /** synced_at field predicates */
  readonly syncedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export enum SearchType {
  /** Lexical search. */
  Lexical = 'LEXICAL',
  /** Semantic search. */
  Semantic = 'SEMANTIC'
}

export enum SearchView {
  /** Schema average vectors for semantic search. */
  SchemaAvgVectors = 'SCHEMA_AVG_VECTORS',
  /** Table average vectors for semantic search. */
  TableAvgVectors = 'TABLE_AVG_VECTORS'
}

export type SearchWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<SearchWhereInput>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** space edge predicates */
  readonly hasSpace?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSpaceWith?: InputMaybe<ReadonlyArray<SpaceWhereInput>>;
  /** node_id field predicates */
  readonly nodeID?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDGT?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly nodeIDLT?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly nodeIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** node_type field predicates */
  readonly nodeType?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nodeTypeIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly nodeTypeNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nodeTypeNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nodeTypeNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly not?: InputMaybe<SearchWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<SearchWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** space_id field predicates */
  readonly spaceID?: InputMaybe<Scalars['ID']['input']>;
  readonly spaceIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly spaceIDIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly spaceIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly spaceIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly spaceIDNotNil?: InputMaybe<Scalars['Boolean']['input']>;
};

export enum ServiceAuthType {
  None = 'NONE',
  Oauth = 'OAUTH'
}

export type Source = Node & {
  readonly __typename?: 'Source';
  readonly autoSync: Scalars['Boolean']['output'];
  readonly catalog: Catalog;
  readonly catalogID: Scalars['ID']['output'];
  readonly config?: Maybe<Scalars['Map']['output']>;
  readonly connection: Connection;
  readonly connectionID: Scalars['ID']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly deletedAt?: Maybe<Scalars['Time']['output']>;
  readonly destination: Destination;
  readonly destinationID: Scalars['ID']['output'];
  readonly externalReports?: Maybe<ReadonlyArray<Maybe<ExternalReport>>>;
  readonly id: Scalars['ID']['output'];
  readonly models?: Maybe<ReadonlyArray<Model>>;
  readonly name: Scalars['String']['output'];
  readonly providerID: Scalars['String']['output'];
  readonly schema: SchemaRef;
  readonly syncError?: Maybe<Scalars['String']['output']>;
  /** Frequency of sync in minutes (default: 360 minutes (6 hours)) */
  readonly syncFreq: Scalars['Int']['output'];
  readonly syncStatus: SyncStatus;
  /** Time of day to sync, has no effect when syncFreq is < 24 hours (default: 00:00) */
  readonly syncTime: Scalars['String']['output'];
  readonly syncedAt?: Maybe<Scalars['Time']['output']>;
  readonly type: SourceType;
  readonly typeID: Scalars['ID']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};

export type SourceConfig = {
  readonly __typename?: 'SourceConfig';
  readonly category: Scalars['String']['output'];
  readonly description: Scalars['String']['output'];
  readonly iconUrl: Scalars['String']['output'];
  readonly id: Scalars['String']['output'];
  readonly name: Scalars['String']['output'];
  readonly schema: Scalars['String']['output'];
  readonly service: Scalars['String']['output'];
};

/** A connection to a list of items. */
export type SourceConnection = {
  readonly __typename?: 'SourceConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<SourceEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type SourceCreated = {
  readonly __typename?: 'SourceCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly source?: Maybe<Source>;
};

export type SourceDeleted = {
  readonly __typename?: 'SourceDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type SourceEdge = {
  readonly __typename?: 'SourceEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<Source>;
};

/** Ordering options for Source connections */
export type SourceOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Sources. */
  readonly field: SourceOrderField;
};

/** Properties by which Source connections can be ordered. */
export enum SourceOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type SourceServiceMetadata = {
  readonly __typename?: 'SourceServiceMetadata';
  readonly sourceConfigs: ReadonlyArray<SourceConfig>;
};

export type SourceType = Node & {
  readonly __typename?: 'SourceType';
  readonly category: Scalars['String']['output'];
  readonly check: SourceTypeCheck;
  readonly configSchema: Scalars['String']['output'];
  readonly connection: Connection;
  readonly connectionID: Scalars['ID']['output'];
  readonly createdAt: Scalars['Time']['output'];
  readonly description: Scalars['String']['output'];
  readonly iconURL: Scalars['String']['output'];
  readonly id: Scalars['ID']['output'];
  readonly modelTypes?: Maybe<ReadonlyArray<ModelType>>;
  readonly name: Scalars['String']['output'];
  readonly providerID: Scalars['String']['output'];
  readonly slug: Scalars['String']['output'];
  readonly sources?: Maybe<ReadonlyArray<Source>>;
  readonly updatedAt: Scalars['Time']['output'];
};

export type SourceTypeCheck = {
  readonly __typename?: 'SourceTypeCheck';
  readonly authCheck: AuthCheck;
  readonly configData?: Maybe<Scalars['Map']['output']>;
};

/** A connection to a list of items. */
export type SourceTypeConnection = {
  readonly __typename?: 'SourceTypeConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<SourceTypeEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type SourceTypeCreated = {
  readonly __typename?: 'SourceTypeCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly sourceType?: Maybe<SourceType>;
};

export type SourceTypeDeleted = {
  readonly __typename?: 'SourceTypeDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type SourceTypeEdge = {
  readonly __typename?: 'SourceTypeEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<SourceType>;
};

/** Ordering options for SourceType connections */
export type SourceTypeOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order SourceTypes. */
  readonly field: SourceTypeOrderField;
};

/** Properties by which SourceType connections can be ordered. */
export enum SourceTypeOrderField {
  CreatedAt = 'CREATED_AT',
  UpdatedAt = 'UPDATED_AT'
}

export type SourceTypeUpdated = {
  readonly __typename?: 'SourceTypeUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly sourceType?: Maybe<SourceType>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * SourceTypeWhereInput is used for filtering SourceType objects.
 * Input was generated by ent.
 */
export type SourceTypeWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<SourceTypeWhereInput>>;
  /** category field predicates */
  readonly category?: InputMaybe<Scalars['String']['input']>;
  readonly categoryContains?: InputMaybe<Scalars['String']['input']>;
  readonly categoryContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly categoryEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly categoryGT?: InputMaybe<Scalars['String']['input']>;
  readonly categoryGTE?: InputMaybe<Scalars['String']['input']>;
  readonly categoryHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly categoryHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly categoryIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly categoryLT?: InputMaybe<Scalars['String']['input']>;
  readonly categoryLTE?: InputMaybe<Scalars['String']['input']>;
  readonly categoryNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly categoryNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** config_schema field predicates */
  readonly configSchema?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaContains?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaGT?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaGTE?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly configSchemaLT?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaLTE?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly configSchemaNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** connection_id field predicates */
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly connectionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** connection edge predicates */
  readonly hasConnection?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionWith?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** model_types edge predicates */
  readonly hasModelTypes?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasModelTypesWith?: InputMaybe<ReadonlyArray<ModelTypeWhereInput>>;
  /** sources edge predicates */
  readonly hasSources?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSourcesWith?: InputMaybe<ReadonlyArray<SourceWhereInput>>;
  /** icon_url field predicates */
  readonly iconURL?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLContains?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLGT?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLGTE?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly iconURLLT?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLLTE?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly iconURLNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<SourceTypeWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<SourceTypeWhereInput>>;
  /** provider_id field predicates */
  readonly providerID?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDContains?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDGT?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDGTE?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly providerIDLT?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDLTE?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type SourceUpdated = {
  readonly __typename?: 'SourceUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly source?: Maybe<Source>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * SourceWhereInput is used for filtering Source objects.
 * Input was generated by ent.
 */
export type SourceWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<SourceWhereInput>>;
  /** auto_sync field predicates */
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly autoSyncNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** catalog_id field predicates */
  readonly catalogID?: InputMaybe<Scalars['ID']['input']>;
  readonly catalogIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly catalogIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly catalogIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** connection_id field predicates */
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly connectionIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly connectionIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** deleted_at field predicates */
  readonly deletedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly deletedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** destination_id field predicates */
  readonly destinationID?: InputMaybe<Scalars['ID']['input']>;
  readonly destinationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly destinationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly destinationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** catalog edge predicates */
  readonly hasCatalog?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasCatalogWith?: InputMaybe<ReadonlyArray<CatalogWhereInput>>;
  /** connection edge predicates */
  readonly hasConnection?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionWith?: InputMaybe<ReadonlyArray<ConnectionWhereInput>>;
  /** destination edge predicates */
  readonly hasDestination?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasDestinationWith?: InputMaybe<ReadonlyArray<DestinationWhereInput>>;
  /** models edge predicates */
  readonly hasModels?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasModelsWith?: InputMaybe<ReadonlyArray<ModelWhereInput>>;
  /** schema edge predicates */
  readonly hasSchema?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSchemaWith?: InputMaybe<ReadonlyArray<SchemaRefWhereInput>>;
  /** type edge predicates */
  readonly hasType?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasTypeWith?: InputMaybe<ReadonlyArray<SourceTypeWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<SourceWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<SourceWhereInput>>;
  /** provider_id field predicates */
  readonly providerID?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDContains?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDGT?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDGTE?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly providerIDLT?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDLTE?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly providerIDNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** sync_error field predicates */
  readonly syncError?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContains?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncErrorLT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorLTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** sync_freq field predicates */
  readonly syncFreq?: InputMaybe<Scalars['Int']['input']>;
  readonly syncFreqGT?: InputMaybe<Scalars['Int']['input']>;
  readonly syncFreqGTE?: InputMaybe<Scalars['Int']['input']>;
  readonly syncFreqIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly syncFreqLT?: InputMaybe<Scalars['Int']['input']>;
  readonly syncFreqLTE?: InputMaybe<Scalars['Int']['input']>;
  readonly syncFreqNEQ?: InputMaybe<Scalars['Int']['input']>;
  readonly syncFreqNotIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  /** sync_status field predicates */
  readonly syncStatus?: InputMaybe<SyncStatus>;
  readonly syncStatusIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  readonly syncStatusNEQ?: InputMaybe<SyncStatus>;
  readonly syncStatusNotIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  /** sync_time field predicates */
  readonly syncTime?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeContains?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeGT?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeGTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncTimeLT?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeLTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly syncTimeNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** synced_at field predicates */
  readonly syncedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** type_id field predicates */
  readonly typeID?: InputMaybe<Scalars['ID']['input']>;
  readonly typeIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly typeIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly typeIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export type Space = Node & {
  readonly __typename?: 'Space';
  readonly catalogs: CatalogConnection;
  readonly createdAt: Scalars['Time']['output'];
  readonly flowRuns: FlowRunConnection;
  readonly flows: FlowConnection;
  readonly geoMaps: GeoMapConnection;
  readonly id: Scalars['ID']['output'];
  readonly name: Scalars['String']['output'];
  readonly organization: Organization;
  readonly organizationID: Scalars['ID']['output'];
  readonly slug: Scalars['String']['output'];
  readonly sqlQueries: SqlQueryConnection;
  /** Timezone name (e.g. America/Los_Angeles) */
  readonly tzName: Scalars['String']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};


export type SpaceCatalogsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<CatalogOrder>;
  where?: InputMaybe<CatalogWhereInput>;
};


export type SpaceFlowRunsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowRunOrder>;
  where?: InputMaybe<FlowRunWhereInput>;
};


export type SpaceFlowsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<FlowOrder>;
  where?: InputMaybe<FlowWhereInput>;
};


export type SpaceGeoMapsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<GeoMapOrder>;
  where?: InputMaybe<GeoMapWhereInput>;
};


export type SpaceSqlQueriesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SqlQueryOrder>;
  where?: InputMaybe<SqlQueryWhereInput>;
};

/** A connection to a list of items. */
export type SpaceConnection = {
  readonly __typename?: 'SpaceConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<SpaceEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type SpaceCreated = {
  readonly __typename?: 'SpaceCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly space?: Maybe<Space>;
};

export type SpaceDeleted = {
  readonly __typename?: 'SpaceDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type SpaceEdge = {
  readonly __typename?: 'SpaceEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<Space>;
};

/** Ordering options for Space connections */
export type SpaceOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Spaces. */
  readonly field: SpaceOrderField;
};

/** Properties by which Space connections can be ordered. */
export enum SpaceOrderField {
  CatalogsCount = 'CATALOGS_COUNT',
  CreatedAt = 'CREATED_AT',
  FlowsCount = 'FLOWS_COUNT',
  FlowRunsCount = 'FLOW_RUNS_COUNT',
  GeoMapsCount = 'GEO_MAPS_COUNT',
  Name = 'NAME',
  SqlQueriesCount = 'SQL_QUERIES_COUNT',
  UpdatedAt = 'UPDATED_AT'
}

export type SpaceUpdated = {
  readonly __typename?: 'SpaceUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly space?: Maybe<Space>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * SpaceWhereInput is used for filtering Space objects.
 * Input was generated by ent.
 */
export type SpaceWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<SpaceWhereInput>>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** catalogs edge predicates */
  readonly hasCatalogs?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasCatalogsWith?: InputMaybe<ReadonlyArray<CatalogWhereInput>>;
  /** flow_runs edge predicates */
  readonly hasFlowRuns?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasFlowRunsWith?: InputMaybe<ReadonlyArray<FlowRunWhereInput>>;
  /** flows edge predicates */
  readonly hasFlows?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasFlowsWith?: InputMaybe<ReadonlyArray<FlowWhereInput>>;
  /** geo_maps edge predicates */
  readonly hasGeoMaps?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasGeoMapsWith?: InputMaybe<ReadonlyArray<GeoMapWhereInput>>;
  /** organization edge predicates */
  readonly hasOrganization?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasOrganizationWith?: InputMaybe<ReadonlyArray<OrganizationWhereInput>>;
  /** sql_queries edge predicates */
  readonly hasSQLQueries?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSQLQueriesWith?: InputMaybe<ReadonlyArray<SqlQueryWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<SpaceWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<SpaceWhereInput>>;
  /** organization_id field predicates */
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly organizationIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly organizationIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** slug field predicates */
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  readonly slugContains?: InputMaybe<Scalars['String']['input']>;
  readonly slugContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly slugGT?: InputMaybe<Scalars['String']['input']>;
  readonly slugGTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly slugHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly slugIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly slugLT?: InputMaybe<Scalars['String']['input']>;
  readonly slugLTE?: InputMaybe<Scalars['String']['input']>;
  readonly slugNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly slugNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** tz_name field predicates */
  readonly tzName?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameContains?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameGT?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly tzNameLT?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

export enum SpecFormat {
  Json = 'JSON',
  Yaml = 'YAML'
}

export type Subscription = {
  readonly __typename?: 'Subscription';
  readonly flowRunSubscribe: FlowRunEvent;
  readonly nodeSubscribe: NodeEvent;
};


export type SubscriptionFlowRunSubscribeArgs = {
  input: FlowRunSubscribeInput;
};


export type SubscriptionNodeSubscribeArgs = {
  input: NodeSubscribeInput;
};

/**
 * SyncPackageInput is used to signal that an integration has changed and should be
 * updated by calling the GetPackage RPC method.
 */
export type SyncPackageInput = {
  readonly checksum: Scalars['String']['input'];
  readonly integrationID: Scalars['ID']['input'];
  readonly publish?: InputMaybe<Scalars['Boolean']['input']>;
};

/** SyncStatus is enum for the field sync_status */
export enum SyncStatus {
  Failed = 'FAILED',
  Pending = 'PENDING',
  Running = 'RUNNING',
  Scheduled = 'SCHEDULED',
  Success = 'SUCCESS',
  Unknown = 'UNKNOWN'
}

export type TableRef = Node & {
  readonly __typename?: 'TableRef';
  readonly alias: Scalars['String']['output'];
  readonly autoSync: Scalars['Boolean']['output'];
  readonly columns: ColumnRefConnection;
  readonly createdAt: Scalars['Time']['output'];
  readonly deletedAt?: Maybe<Scalars['Time']['output']>;
  readonly description?: Maybe<Scalars['String']['output']>;
  readonly fields?: Maybe<ReadonlyArray<Maybe<Field>>>;
  readonly id: Scalars['ID']['output'];
  readonly ioSchema?: Maybe<IoSchema>;
  readonly name: Scalars['String']['output'];
  readonly schema: SchemaRef;
  readonly schemaID: Scalars['ID']['output'];
  readonly syncError?: Maybe<Scalars['String']['output']>;
  readonly syncStatus: SyncStatus;
  readonly syncedAt?: Maybe<Scalars['Time']['output']>;
  readonly tableType?: Maybe<Scalars['String']['output']>;
  readonly totalBytes?: Maybe<Scalars['Int']['output']>;
  readonly totalRows?: Maybe<Scalars['Int']['output']>;
  readonly updatedAt: Scalars['Time']['output'];
};


export type TableRefColumnsArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<ReadonlyArray<ColumnRefOrder>>;
  where?: InputMaybe<ColumnRefWhereInput>;
};

/** A connection to a list of items. */
export type TableRefConnection = {
  readonly __typename?: 'TableRefConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<TableRefEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type TableRefCreated = {
  readonly __typename?: 'TableRefCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly tableRef?: Maybe<TableRef>;
};

export type TableRefDeleted = {
  readonly __typename?: 'TableRefDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type TableRefEdge = {
  readonly __typename?: 'TableRefEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<TableRef>;
};

/** Ordering options for TableRef connections */
export type TableRefOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order TableRefs. */
  readonly field: TableRefOrderField;
};

/** Properties by which TableRef connections can be ordered. */
export enum TableRefOrderField {
  Alias = 'ALIAS',
  ColumnsCount = 'COLUMNS_COUNT',
  CreatedAt = 'CREATED_AT',
  Name = 'NAME',
  TotalBytes = 'TOTAL_BYTES',
  TotalRows = 'TOTAL_ROWS',
  UpdatedAt = 'UPDATED_AT'
}

export type TableRefUpdated = {
  readonly __typename?: 'TableRefUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly tableRef?: Maybe<TableRef>;
  readonly updated: Scalars['Boolean']['output'];
};

/**
 * TableRefWhereInput is used for filtering TableRef objects.
 * Input was generated by ent.
 */
export type TableRefWhereInput = {
  /** alias field predicates */
  readonly alias?: InputMaybe<Scalars['String']['input']>;
  readonly aliasContains?: InputMaybe<Scalars['String']['input']>;
  readonly aliasContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly aliasEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly aliasGT?: InputMaybe<Scalars['String']['input']>;
  readonly aliasGTE?: InputMaybe<Scalars['String']['input']>;
  readonly aliasHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly aliasHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly aliasIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly aliasLT?: InputMaybe<Scalars['String']['input']>;
  readonly aliasLTE?: InputMaybe<Scalars['String']['input']>;
  readonly aliasNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly aliasNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly and?: InputMaybe<ReadonlyArray<TableRefWhereInput>>;
  /** auto_sync field predicates */
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly autoSyncNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** deleted_at field predicates */
  readonly deletedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly deletedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly deletedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly deletedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** description field predicates */
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContains?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionGTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly descriptionLT?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionLTE?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly descriptionNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly descriptionNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** columns edge predicates */
  readonly hasColumns?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasColumnsWith?: InputMaybe<ReadonlyArray<ColumnRefWhereInput>>;
  /** io_schema edge predicates */
  readonly hasIoSchema?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasIoSchemaWith?: InputMaybe<ReadonlyArray<IoSchemaWhereInput>>;
  /** schema edge predicates */
  readonly hasSchema?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSchemaWith?: InputMaybe<ReadonlyArray<SchemaRefWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly not?: InputMaybe<TableRefWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<TableRefWhereInput>>;
  /** schema_id field predicates */
  readonly schemaID?: InputMaybe<Scalars['ID']['input']>;
  readonly schemaIDIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly schemaIDNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly schemaIDNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** sync_error field predicates */
  readonly syncError?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContains?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorGTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncErrorLT?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorLTE?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly syncErrorNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly syncErrorNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** sync_status field predicates */
  readonly syncStatus?: InputMaybe<SyncStatus>;
  readonly syncStatusIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  readonly syncStatusNEQ?: InputMaybe<SyncStatus>;
  readonly syncStatusNotIn?: InputMaybe<ReadonlyArray<SyncStatus>>;
  /** synced_at field predicates */
  readonly syncedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly syncedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly syncedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly syncedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** table_type field predicates */
  readonly tableType?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeContains?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeGT?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeGTE?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly tableTypeIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly tableTypeLT?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeLTE?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly tableTypeNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly tableTypeNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** total_bytes field predicates */
  readonly totalBytes?: InputMaybe<Scalars['Int']['input']>;
  readonly totalBytesGT?: InputMaybe<Scalars['Int']['input']>;
  readonly totalBytesGTE?: InputMaybe<Scalars['Int']['input']>;
  readonly totalBytesIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly totalBytesIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly totalBytesLT?: InputMaybe<Scalars['Int']['input']>;
  readonly totalBytesLTE?: InputMaybe<Scalars['Int']['input']>;
  readonly totalBytesNEQ?: InputMaybe<Scalars['Int']['input']>;
  readonly totalBytesNotIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly totalBytesNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** total_rows field predicates */
  readonly totalRows?: InputMaybe<Scalars['Int']['input']>;
  readonly totalRowsGT?: InputMaybe<Scalars['Int']['input']>;
  readonly totalRowsGTE?: InputMaybe<Scalars['Int']['input']>;
  readonly totalRowsIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly totalRowsIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly totalRowsLT?: InputMaybe<Scalars['Int']['input']>;
  readonly totalRowsLTE?: InputMaybe<Scalars['Int']['input']>;
  readonly totalRowsNEQ?: InputMaybe<Scalars['Int']['input']>;
  readonly totalRowsNotIn?: InputMaybe<ReadonlyArray<Scalars['Int']['input']>>;
  readonly totalRowsNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

/**
 * UpdateCatalogInput is used for update Catalog object.
 * Input was generated by ent.
 */
export type UpdateCatalogInput = {
  readonly addSpaceIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly alias?: InputMaybe<Scalars['String']['input']>;
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly clearDescription?: InputMaybe<Scalars['Boolean']['input']>;
  readonly clearSpaces?: InputMaybe<Scalars['Boolean']['input']>;
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly label?: InputMaybe<Scalars['String']['input']>;
  readonly removeSpaceIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
};

/** UpdateConnectionInput is used to update Connection object. */
export type UpdateConnectionInput = {
  readonly config?: InputMaybe<Scalars['Any']['input']>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly visibility?: InputMaybe<Visibility>;
};

/** UpdateDestinationInput is used to update a Destination with given config overrides. */
export type UpdateDestinationInput = {
  readonly config?: InputMaybe<Scalars['Map']['input']>;
};

/**
 * UpdateEventSourceInput is used for update EventSource object.
 * Input was generated by ent.
 */
export type UpdateEventSourceInput = {
  readonly clearPullFreq?: InputMaybe<Scalars['Boolean']['input']>;
  readonly config?: InputMaybe<Scalars['String']['input']>;
  readonly connectionID?: InputMaybe<Scalars['ID']['input']>;
  /** Pull frequency in string format, e.g.: "1s", "2.3h" or "4h35m" */
  readonly pullFreq?: InputMaybe<Scalars['String']['input']>;
};

/** UpdateFlowInput is used to update a Flow. */
export type UpdateFlowInput = {
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly revisionId?: InputMaybe<Scalars['ID']['input']>;
};

/**
 * UpdateGeoLayerInput is used for update GeoLayer object.
 * Input was generated by ent.
 */
export type UpdateGeoLayerInput = {
  readonly addFeatureIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly geoField?: InputMaybe<Scalars['String']['input']>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly propFields?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly removeFeatureIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly settings?: InputMaybe<GeoLayerSettingsInput>;
  readonly visibility?: InputMaybe<Visibility>;
};

/**
 * UpdateGeoMapInput is used for update GeoMap object.
 * Input was generated by ent.
 */
export type UpdateGeoMapInput = {
  readonly addLayerIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly removeLayerIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly settings?: InputMaybe<GeoMapSettingsInput>;
  readonly visibility?: InputMaybe<Visibility>;
};

/**
 * UpdateIntegrationInput is used for update Integration object.
 * Input was generated by ent.
 */
export type UpdateIntegrationInput = {
  readonly address?: InputMaybe<Scalars['String']['input']>;
  readonly appendServiceNames?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly clearConfigSchema?: InputMaybe<Scalars['Boolean']['input']>;
  readonly clearServiceNames?: InputMaybe<Scalars['Boolean']['input']>;
  readonly configSchema?: InputMaybe<Scalars['String']['input']>;
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly icon?: InputMaybe<Scalars['String']['input']>;
  readonly network?: InputMaybe<Scalars['String']['input']>;
  readonly organizationID?: InputMaybe<Scalars['ID']['input']>;
  readonly serviceNames?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly version?: InputMaybe<Scalars['String']['input']>;
};

/**
 * UpdateModelInput is used for update Model object.
 * Input was generated by ent.
 */
export type UpdateModelInput = {
  readonly addSourceIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly clearMetadata?: InputMaybe<Scalars['Boolean']['input']>;
  readonly clearSources?: InputMaybe<Scalars['Boolean']['input']>;
  readonly metadata?: InputMaybe<Scalars['Map']['input']>;
  readonly removeSourceIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
};

/**
 * UpdateOrganizationInput is used for update Organization object.
 * Input was generated by ent.
 */
export type UpdateOrganizationInput = {
  readonly addConnectionIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly clearConnections?: InputMaybe<Scalars['Boolean']['input']>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly removeConnectionIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly slug?: InputMaybe<Scalars['String']['input']>;
  /** Timezone name (e.g. America/Los_Angeles) */
  readonly tzName?: InputMaybe<Scalars['String']['input']>;
};

/** UpdatePersonalAccessTokenInput is used to update a PersonalAccessToken. */
export type UpdatePersonalAccessTokenInput = {
  readonly expiresAt?: InputMaybe<Scalars['Time']['input']>;
  /** Expiration in string format, e.g.: "1s", "2.3h" or "4h35m" */
  readonly expiresIn?: InputMaybe<Scalars['String']['input']>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
};

/** UpdateRoleInput is used for updating a role. */
export type UpdateRoleInput = {
  readonly addResources?: InputMaybe<ReadonlyArray<RoleResourceInput>>;
  readonly addUserIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly orgID: Scalars['ID']['input'];
  readonly removeResources?: InputMaybe<ReadonlyArray<RoleResourceInput>>;
  readonly removeUserIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly spaceID?: InputMaybe<Scalars['ID']['input']>;
  readonly type: RoleType;
};

export type UpdateSqlQueryInput = {
  readonly params?: InputMaybe<ReadonlyArray<ParamInput>>;
  readonly sql?: InputMaybe<Scalars['String']['input']>;
};

/**
 * UpdateSchemaRefInput is used for update SchemaRef object.
 * Input was generated by ent.
 */
export type UpdateSchemaRefInput = {
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  readonly clearDescription?: InputMaybe<Scalars['Boolean']['input']>;
  readonly description?: InputMaybe<Scalars['String']['input']>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
};

/** UpdateSourceInput is used to update a replication Source. */
export type UpdateSourceInput = {
  /** When enabled, Source will sync automatically on the schedule provided by syncFreq and syncTime (if applicable). */
  readonly autoSync?: InputMaybe<Scalars['Boolean']['input']>;
  /** Optional config. */
  readonly config?: InputMaybe<Scalars['Map']['input']>;
  /** Frequency of sync in minutes (default: 360 minutes (6 hours)) */
  readonly syncFreq?: InputMaybe<Scalars['Int']['input']>;
  /** Time of day to sync, has no effect when syncFreq is < 24 hours (default: 00:00) */
  readonly syncTime?: InputMaybe<Scalars['String']['input']>;
};

/**
 * UpdateSpaceInput is used for update Space object.
 * Input was generated by ent.
 */
export type UpdateSpaceInput = {
  readonly addCatalogIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly addFlowIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly addFlowRunIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly addGeoMapIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly clearCatalogs?: InputMaybe<Scalars['Boolean']['input']>;
  readonly clearFlowRuns?: InputMaybe<Scalars['Boolean']['input']>;
  readonly clearFlows?: InputMaybe<Scalars['Boolean']['input']>;
  readonly clearGeoMaps?: InputMaybe<Scalars['Boolean']['input']>;
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly removeCatalogIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly removeFlowIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly removeFlowRunIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly removeGeoMapIDs?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** Timezone name (e.g. America/Los_Angeles) */
  readonly tzName?: InputMaybe<Scalars['String']['input']>;
};

/** UpdateUserActionInput sets the value of a UserAction scoped to a FlowRun. */
export type UpdateUserActionInput = {
  readonly nodeId: Scalars['String']['input'];
  readonly value: Scalars['Map']['input'];
};

export type User = Node & {
  readonly __typename?: 'User';
  readonly banexpires?: Maybe<Scalars['Time']['output']>;
  readonly banned: Scalars['Boolean']['output'];
  readonly banreason?: Maybe<Scalars['String']['output']>;
  readonly connectionUser?: Maybe<ReadonlyArray<ConnectionUser>>;
  readonly createdAt: Scalars['Time']['output'];
  readonly email?: Maybe<Scalars['String']['output']>;
  readonly emailVerified?: Maybe<Scalars['Boolean']['output']>;
  readonly id: Scalars['ID']['output'];
  readonly image?: Maybe<Scalars['String']['output']>;
  readonly name?: Maybe<Scalars['String']['output']>;
  readonly packages?: Maybe<ReadonlyArray<IntegrationPackage>>;
  readonly personalAccessTokens?: Maybe<ReadonlyArray<PersonalAccessToken>>;
  readonly role?: Maybe<Scalars['String']['output']>;
  readonly sqlQueries: SqlQueryConnection;
  readonly termsAndPrivacyAcceptedAt?: Maybe<Scalars['Time']['output']>;
  readonly twoFactorEnabled: Scalars['Boolean']['output'];
  /** Timezone name (e.g. America/Los_Angeles) */
  readonly tzName: Scalars['String']['output'];
  readonly updatedAt: Scalars['Time']['output'];
};


export type UserSqlQueriesArgs = {
  after?: InputMaybe<Scalars['Cursor']['input']>;
  before?: InputMaybe<Scalars['Cursor']['input']>;
  first?: InputMaybe<Scalars['Int']['input']>;
  last?: InputMaybe<Scalars['Int']['input']>;
  orderBy?: InputMaybe<SqlQueryOrder>;
  where?: InputMaybe<SqlQueryWhereInput>;
};

/** A connection to a list of items. */
export type UserConnection = {
  readonly __typename?: 'UserConnection';
  /** A list of edges. */
  readonly edges?: Maybe<ReadonlyArray<Maybe<UserEdge>>>;
  /** Information to aid in pagination. */
  readonly pageInfo: PageInfo;
  /** Identifies the total count of items in the connection. */
  readonly totalCount: Scalars['Int']['output'];
};

export type UserCreated = {
  readonly __typename?: 'UserCreated';
  readonly created: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly user?: Maybe<User>;
};

export type UserDeleted = {
  readonly __typename?: 'UserDeleted';
  readonly deleted: Scalars['Boolean']['output'];
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly id?: Maybe<Scalars['ID']['output']>;
};

/** An edge in a connection. */
export type UserEdge = {
  readonly __typename?: 'UserEdge';
  /** A cursor for use in pagination. */
  readonly cursor: Scalars['Cursor']['output'];
  /** The item at the end of the edge. */
  readonly node?: Maybe<User>;
};

/** Ordering options for User connections */
export type UserOrder = {
  /** The ordering direction. */
  readonly direction?: OrderDirection;
  /** The field by which to order Users. */
  readonly field: UserOrderField;
};

/** Properties by which User connections can be ordered. */
export enum UserOrderField {
  CreatedAt = 'CREATED_AT',
  SqlQueriesCount = 'SQL_QUERIES_COUNT',
  UpdatedAt = 'UPDATED_AT'
}

export type UserUpdated = {
  readonly __typename?: 'UserUpdated';
  readonly error?: Maybe<Scalars['String']['output']>;
  readonly updated: Scalars['Boolean']['output'];
  readonly user?: Maybe<User>;
};

/**
 * UserWhereInput is used for filtering User objects.
 * Input was generated by ent.
 */
export type UserWhereInput = {
  readonly and?: InputMaybe<ReadonlyArray<UserWhereInput>>;
  /** banExpires field predicates */
  readonly banexpires?: InputMaybe<Scalars['Time']['input']>;
  readonly banexpiresGT?: InputMaybe<Scalars['Time']['input']>;
  readonly banexpiresGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly banexpiresIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly banexpiresIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly banexpiresLT?: InputMaybe<Scalars['Time']['input']>;
  readonly banexpiresLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly banexpiresNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly banexpiresNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly banexpiresNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** banned field predicates */
  readonly banned?: InputMaybe<Scalars['Boolean']['input']>;
  readonly bannedNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** banReason field predicates */
  readonly banreason?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonContains?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonGT?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonGTE?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly banreasonIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly banreasonLT?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonLTE?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly banreasonNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly banreasonNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** created_at field predicates */
  readonly createdAt?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly createdAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly createdAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  /** email field predicates */
  readonly email?: InputMaybe<Scalars['String']['input']>;
  readonly emailContains?: InputMaybe<Scalars['String']['input']>;
  readonly emailContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly emailEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly emailGT?: InputMaybe<Scalars['String']['input']>;
  readonly emailGTE?: InputMaybe<Scalars['String']['input']>;
  readonly emailHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly emailHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly emailIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly emailIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly emailLT?: InputMaybe<Scalars['String']['input']>;
  readonly emailLTE?: InputMaybe<Scalars['String']['input']>;
  readonly emailNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly emailNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly emailNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** email_verified field predicates */
  readonly emailVerified?: InputMaybe<Scalars['Boolean']['input']>;
  readonly emailVerifiedIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly emailVerifiedNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  readonly emailVerifiedNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** connection_user edge predicates */
  readonly hasConnectionUser?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasConnectionUserWith?: InputMaybe<ReadonlyArray<ConnectionUserWhereInput>>;
  /** packages edge predicates */
  readonly hasPackages?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasPackagesWith?: InputMaybe<ReadonlyArray<IntegrationPackageWhereInput>>;
  /** personal_access_tokens edge predicates */
  readonly hasPersonalAccessTokens?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasPersonalAccessTokensWith?: InputMaybe<ReadonlyArray<PersonalAccessTokenWhereInput>>;
  /** sql_queries edge predicates */
  readonly hasSQLQueries?: InputMaybe<Scalars['Boolean']['input']>;
  readonly hasSQLQueriesWith?: InputMaybe<ReadonlyArray<SqlQueryWhereInput>>;
  /** id field predicates */
  readonly id?: InputMaybe<Scalars['ID']['input']>;
  readonly idGT?: InputMaybe<Scalars['ID']['input']>;
  readonly idGTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  readonly idLT?: InputMaybe<Scalars['ID']['input']>;
  readonly idLTE?: InputMaybe<Scalars['ID']['input']>;
  readonly idNEQ?: InputMaybe<Scalars['ID']['input']>;
  readonly idNotIn?: InputMaybe<ReadonlyArray<Scalars['ID']['input']>>;
  /** image field predicates */
  readonly image?: InputMaybe<Scalars['String']['input']>;
  readonly imageContains?: InputMaybe<Scalars['String']['input']>;
  readonly imageContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly imageEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly imageGT?: InputMaybe<Scalars['String']['input']>;
  readonly imageGTE?: InputMaybe<Scalars['String']['input']>;
  readonly imageHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly imageHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly imageIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly imageIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly imageLT?: InputMaybe<Scalars['String']['input']>;
  readonly imageLTE?: InputMaybe<Scalars['String']['input']>;
  readonly imageNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly imageNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly imageNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** name field predicates */
  readonly name?: InputMaybe<Scalars['String']['input']>;
  readonly nameContains?: InputMaybe<Scalars['String']['input']>;
  readonly nameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly nameGT?: InputMaybe<Scalars['String']['input']>;
  readonly nameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly nameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly nameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly nameLT?: InputMaybe<Scalars['String']['input']>;
  readonly nameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly nameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly nameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly nameNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly not?: InputMaybe<UserWhereInput>;
  readonly or?: InputMaybe<ReadonlyArray<UserWhereInput>>;
  /** role field predicates */
  readonly role?: InputMaybe<Scalars['String']['input']>;
  readonly roleContains?: InputMaybe<Scalars['String']['input']>;
  readonly roleContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly roleEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly roleGT?: InputMaybe<Scalars['String']['input']>;
  readonly roleGTE?: InputMaybe<Scalars['String']['input']>;
  readonly roleHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly roleHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly roleIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly roleIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly roleLT?: InputMaybe<Scalars['String']['input']>;
  readonly roleLTE?: InputMaybe<Scalars['String']['input']>;
  readonly roleNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly roleNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly roleNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** terms_and_privacy_accepted_at field predicates */
  readonly termsAndPrivacyAcceptedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly termsAndPrivacyAcceptedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly termsAndPrivacyAcceptedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly termsAndPrivacyAcceptedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly termsAndPrivacyAcceptedAtIsNil?: InputMaybe<Scalars['Boolean']['input']>;
  readonly termsAndPrivacyAcceptedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly termsAndPrivacyAcceptedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly termsAndPrivacyAcceptedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly termsAndPrivacyAcceptedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly termsAndPrivacyAcceptedAtNotNil?: InputMaybe<Scalars['Boolean']['input']>;
  /** two_factor_enabled field predicates */
  readonly twoFactorEnabled?: InputMaybe<Scalars['Boolean']['input']>;
  readonly twoFactorEnabledNEQ?: InputMaybe<Scalars['Boolean']['input']>;
  /** tz_name field predicates */
  readonly tzName?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameContains?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameContainsFold?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameEqualFold?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameGT?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameGTE?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameHasPrefix?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameHasSuffix?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  readonly tzNameLT?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameLTE?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameNEQ?: InputMaybe<Scalars['String']['input']>;
  readonly tzNameNotIn?: InputMaybe<ReadonlyArray<Scalars['String']['input']>>;
  /** updated_at field predicates */
  readonly updatedAt?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtGTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
  readonly updatedAtLT?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtLTE?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNEQ?: InputMaybe<Scalars['Time']['input']>;
  readonly updatedAtNotIn?: InputMaybe<ReadonlyArray<Scalars['Time']['input']>>;
};

/** Visibility is enum for the field visibility */
export enum Visibility {
  Private = 'PRIVATE',
  Public = 'PUBLIC',
  Shared = 'SHARED'
}

export type WriteTableInput = {
  readonly rows: ReadonlyArray<Scalars['Map']['input']>;
};

export type WriteTableOutput = {
  readonly __typename?: 'WriteTableOutput';
  readonly rowsWritten: Scalars['Int64']['output'];
  readonly table: TableRef;
};

export type CatalogFragmentFragment = { readonly __typename?: 'Catalog', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly label: string, readonly queryDialect: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly connectionID: string, readonly organizationID: string };

export type GetCatalogQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetCatalogQuery = { readonly __typename?: 'Query', readonly catalog?: { readonly __typename?: 'Catalog', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly label: string, readonly queryDialect: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly connectionID: string, readonly organizationID: string } | null };

export type ListCatalogsQueryVariables = Exact<{
  orderBy?: InputMaybe<CatalogOrder>;
  where?: InputMaybe<CatalogWhereInput>;
}>;


export type ListCatalogsQuery = { readonly __typename?: 'Query', readonly catalogs: { readonly __typename?: 'CatalogConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'CatalogEdge', readonly node?: { readonly __typename?: 'Catalog', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly label: string, readonly queryDialect: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly connectionID: string, readonly organizationID: string } | null } | null> | null } };

export type CreateCatalogMutationVariables = Exact<{
  input: CreateCatalogInput;
}>;


export type CreateCatalogMutation = { readonly __typename?: 'Mutation', readonly createCatalog: { readonly __typename?: 'CatalogCreated', readonly created: boolean, readonly error?: string | null, readonly catalog?: { readonly __typename?: 'Catalog', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly label: string, readonly queryDialect: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly connectionID: string, readonly organizationID: string } | null } };

export type SyncCatalogMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type SyncCatalogMutation = { readonly __typename?: 'Mutation', readonly syncCatalog: { readonly __typename?: 'CatalogUpdated', readonly updated: boolean, readonly error?: string | null, readonly catalog?: { readonly __typename?: 'Catalog', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly label: string, readonly queryDialect: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly connectionID: string, readonly organizationID: string } | null } };

export type UpdateCatalogMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateCatalogInput;
}>;


export type UpdateCatalogMutation = { readonly __typename?: 'Mutation', readonly updateCatalog: { readonly __typename?: 'CatalogUpdated', readonly updated: boolean, readonly error?: string | null, readonly catalog?: { readonly __typename?: 'Catalog', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly label: string, readonly queryDialect: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly connectionID: string, readonly organizationID: string } | null } };

export type MutationMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type MutationMutation = { readonly __typename?: 'Mutation', readonly deleteCatalog: { readonly __typename?: 'CatalogDeleted', readonly id?: string | null } };

export type ColumnRefFragmentFragment = { readonly __typename?: 'ColumnRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly dtype: string, readonly ordinal: number, readonly nullable: boolean, readonly repeated: boolean, readonly primaryKey: boolean, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly tableID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null };

export type GetColumnQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetColumnQuery = { readonly __typename?: 'Query', readonly column?: { readonly __typename?: 'ColumnRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly dtype: string, readonly ordinal: number, readonly nullable: boolean, readonly repeated: boolean, readonly primaryKey: boolean, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly tableID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null } | null };

export type ListColumnsQueryVariables = Exact<{
  orderBy?: InputMaybe<ReadonlyArray<ColumnRefOrder> | ColumnRefOrder>;
  where?: InputMaybe<ColumnRefWhereInput>;
}>;


export type ListColumnsQuery = { readonly __typename?: 'Query', readonly columns: { readonly __typename?: 'ColumnRefConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'ColumnRefEdge', readonly node?: { readonly __typename?: 'ColumnRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly dtype: string, readonly ordinal: number, readonly nullable: boolean, readonly repeated: boolean, readonly primaryKey: boolean, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly tableID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null } | null } | null> | null } };

export type CreateColumnMutationVariables = Exact<{
  input: CreateColumnInput;
}>;


export type CreateColumnMutation = { readonly __typename?: 'Mutation', readonly createColumn?: { readonly __typename?: 'ColumnRefCreated', readonly created: boolean, readonly error?: string | null, readonly columnRef?: { readonly __typename?: 'ColumnRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly dtype: string, readonly ordinal: number, readonly nullable: boolean, readonly repeated: boolean, readonly primaryKey: boolean, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly tableID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null } | null } | null };

export type DeleteColumnMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type DeleteColumnMutation = { readonly __typename?: 'Mutation', readonly deleteColumn?: { readonly __typename?: 'ColumnRefDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } | null };

export type ConnectionFragmentFragment = { readonly __typename?: 'Connection', readonly id: string, readonly slug: string, readonly name: string, readonly visibility: Visibility, readonly createdAt: string, readonly updatedAt: string, readonly integrationID: string, readonly organizationID: string };

export type GetConnectionQueryVariables = Exact<{
  connectionId?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetConnectionQuery = { readonly __typename?: 'Query', readonly connection?: { readonly __typename?: 'Connection', readonly id: string, readonly slug: string, readonly name: string, readonly visibility: Visibility, readonly createdAt: string, readonly updatedAt: string, readonly integrationID: string, readonly organizationID: string } | null };

export type ListConnectionsQueryVariables = Exact<{
  orderBy?: InputMaybe<ConnectionOrder>;
  where?: InputMaybe<ConnectionWhereInput>;
}>;


export type ListConnectionsQuery = { readonly __typename?: 'Query', readonly connections: { readonly __typename?: 'ConnectionConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'ConnectionEdge', readonly node?: { readonly __typename?: 'Connection', readonly id: string, readonly slug: string, readonly name: string, readonly visibility: Visibility, readonly createdAt: string, readonly updatedAt: string, readonly integrationID: string, readonly organizationID: string } | null } | null> | null } };

export type CheckConnectionQueryVariables = Exact<{
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
}>;


export type CheckConnectionQuery = { readonly __typename?: 'Query', readonly checkConnection?: { readonly __typename?: 'ConnectionCheck', readonly authCheck: { readonly __typename?: 'AuthCheck', readonly type: ServiceAuthType, readonly required: boolean, readonly success: boolean, readonly error?: string | null }, readonly configCheck: { readonly __typename?: 'ConfigCheck', readonly success: boolean, readonly error?: string | null } } | null };

export type CreateConnectionMutationVariables = Exact<{
  input: CreateConnectionInput;
}>;


export type CreateConnectionMutation = { readonly __typename?: 'Mutation', readonly createConnection?: { readonly __typename?: 'ConnectionCreated', readonly created: boolean, readonly error?: string | null, readonly connection?: { readonly __typename?: 'Connection', readonly id: string, readonly slug: string, readonly name: string, readonly visibility: Visibility, readonly createdAt: string, readonly updatedAt: string, readonly integrationID: string, readonly organizationID: string } | null } | null };

export type UpdateConnectionMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateConnectionInput;
}>;


export type UpdateConnectionMutation = { readonly __typename?: 'Mutation', readonly updateConnection?: { readonly __typename?: 'ConnectionUpdated', readonly updated: boolean, readonly error?: string | null, readonly connection?: { readonly __typename?: 'Connection', readonly id: string, readonly slug: string, readonly name: string, readonly visibility: Visibility, readonly createdAt: string, readonly updatedAt: string, readonly integrationID: string, readonly organizationID: string } | null } | null };

export type DeleteConnectionMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteConnectionMutation = { readonly __typename?: 'Mutation', readonly deleteConnection?: { readonly __typename?: 'ConnectionDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } | null };

export type DestinationFragmentFragment = { readonly __typename?: 'Destination', readonly id: string, readonly name: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string, readonly providerID: string, readonly connectionID: string, readonly organizationID: string };

export type GetDestinationQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetDestinationQuery = { readonly __typename?: 'Query', readonly destination?: { readonly __typename?: 'Destination', readonly id: string, readonly name: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string, readonly providerID: string, readonly connectionID: string, readonly organizationID: string } | null };

export type ListDestinationsQueryVariables = Exact<{
  orderBy?: InputMaybe<DestinationOrder>;
  where?: InputMaybe<DestinationWhereInput>;
}>;


export type ListDestinationsQuery = { readonly __typename?: 'Query', readonly destinations: { readonly __typename?: 'DestinationConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'DestinationEdge', readonly node?: { readonly __typename?: 'Destination', readonly id: string, readonly name: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string, readonly providerID: string, readonly connectionID: string, readonly organizationID: string } | null } | null> | null } };

export type CreateDestinationMutationVariables = Exact<{
  input: CreateDestinationInput;
}>;


export type CreateDestinationMutation = { readonly __typename?: 'Mutation', readonly createDestination: { readonly __typename?: 'DestinationCreated', readonly created: boolean, readonly error?: string | null, readonly destination?: { readonly __typename?: 'Destination', readonly id: string, readonly name: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string, readonly providerID: string, readonly connectionID: string, readonly organizationID: string } | null } };

export type DeleteDestinationMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteDestinationMutation = { readonly __typename?: 'Mutation', readonly deleteDestination: { readonly __typename?: 'DestinationDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } };

export type EventSourceFragmentFragment = { readonly __typename?: 'EventSource', readonly id: string, readonly slug: string, readonly name: string, readonly status: EventSourceStatus, readonly error?: string | null, readonly strategy: EventSourceStrategy, readonly pullFreq?: string | null, readonly pushUrl?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly connectionID: string };

export type GetEventSourceQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetEventSourceQuery = { readonly __typename?: 'Query', readonly eventSource?: { readonly __typename?: 'EventSource', readonly id: string, readonly slug: string, readonly name: string, readonly status: EventSourceStatus, readonly error?: string | null, readonly strategy: EventSourceStrategy, readonly pullFreq?: string | null, readonly pushUrl?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly connectionID: string } | null };

export type ListEventSourcesQueryVariables = Exact<{
  orderBy?: InputMaybe<EventSourceOrder>;
  where?: InputMaybe<EventSourceWhereInput>;
}>;


export type ListEventSourcesQuery = { readonly __typename?: 'Query', readonly eventSources: { readonly __typename?: 'EventSourceConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'EventSourceEdge', readonly node?: { readonly __typename?: 'EventSource', readonly id: string, readonly slug: string, readonly name: string, readonly status: EventSourceStatus, readonly error?: string | null, readonly strategy: EventSourceStrategy, readonly pullFreq?: string | null, readonly pushUrl?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly connectionID: string } | null } | null> | null } };

export type CreateEventSourceMutationVariables = Exact<{
  input: CreateEventSourceInput;
}>;


export type CreateEventSourceMutation = { readonly __typename?: 'Mutation', readonly createEventSource?: { readonly __typename?: 'EventSourceCreated', readonly created: boolean, readonly error?: string | null, readonly eventSource?: { readonly __typename?: 'EventSource', readonly id: string, readonly slug: string, readonly name: string, readonly status: EventSourceStatus, readonly error?: string | null, readonly strategy: EventSourceStrategy, readonly pullFreq?: string | null, readonly pushUrl?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly connectionID: string } | null } | null };

export type UpdateEventSourceMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateEventSourceInput;
}>;


export type UpdateEventSourceMutation = { readonly __typename?: 'Mutation', readonly updateEventSource?: { readonly __typename?: 'EventSourceUpdated', readonly updated: boolean, readonly error?: string | null, readonly eventSource?: { readonly __typename?: 'EventSource', readonly id: string, readonly slug: string, readonly name: string, readonly status: EventSourceStatus, readonly error?: string | null, readonly strategy: EventSourceStrategy, readonly pullFreq?: string | null, readonly pushUrl?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly connectionID: string } | null } | null };

export type DeleteEventSourceMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteEventSourceMutation = { readonly __typename?: 'Mutation', readonly deleteEventSource?: { readonly __typename?: 'EventSourceDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } | null };

export type FlowRevisionFragmentFragment = { readonly __typename?: 'FlowRevision', readonly id: string, readonly checksum: string, readonly createdAt: string, readonly updatedAt: string, readonly flowID: string, readonly body: { readonly __typename?: 'FlowBody', readonly raw: string } };

export type ListFlowRevisionsQueryVariables = Exact<{
  orderBy?: InputMaybe<FlowRevisionOrder>;
  where?: InputMaybe<FlowRevisionWhereInput>;
}>;


export type ListFlowRevisionsQuery = { readonly __typename?: 'Query', readonly flowRevisions: { readonly __typename?: 'FlowRevisionConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'FlowRevisionEdge', readonly node?: { readonly __typename?: 'FlowRevision', readonly id: string, readonly checksum: string, readonly createdAt: string, readonly updatedAt: string, readonly flowID: string, readonly body: { readonly __typename?: 'FlowBody', readonly raw: string } } | null } | null> | null } };

export type CreateFlowRevisionMutationVariables = Exact<{
  input: CreateFlowRevisionInput;
}>;


export type CreateFlowRevisionMutation = { readonly __typename?: 'Mutation', readonly createFlowRevision?: { readonly __typename?: 'FlowRevisionCreated', readonly created: boolean, readonly error?: string | null, readonly flowRevision?: { readonly __typename?: 'FlowRevision', readonly id: string, readonly checksum: string, readonly createdAt: string, readonly updatedAt: string, readonly flowID: string, readonly body: { readonly __typename?: 'FlowBody', readonly raw: string } } | null } | null };

export type DeleteFlowRevisionMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteFlowRevisionMutation = { readonly __typename?: 'Mutation', readonly deleteFlowRevision?: { readonly __typename?: 'FlowRevisionDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } | null };

export type FlowRunFragmentFragment = { readonly __typename?: 'FlowRun', readonly id: string, readonly status: FlowRunStatus, readonly error?: string | null, readonly runTimeout: string, readonly stopTimeout: string, readonly createdAt: string, readonly updatedAt: string, readonly flowID: string, readonly revisionID: string, readonly config: { readonly __typename?: 'FlowRunConfig', readonly inputs?: Record<string, unknown> | null, readonly resources?: ReadonlyArray<{ readonly __typename?: 'FlowRunResource', readonly type: string, readonly id: string, readonly nodeId: string }> | null } };

export type GetFlowRunQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetFlowRunQuery = { readonly __typename?: 'Query', readonly flowRun?: { readonly __typename?: 'FlowRun', readonly id: string, readonly status: FlowRunStatus, readonly error?: string | null, readonly runTimeout: string, readonly stopTimeout: string, readonly createdAt: string, readonly updatedAt: string, readonly flowID: string, readonly revisionID: string, readonly config: { readonly __typename?: 'FlowRunConfig', readonly inputs?: Record<string, unknown> | null, readonly resources?: ReadonlyArray<{ readonly __typename?: 'FlowRunResource', readonly type: string, readonly id: string, readonly nodeId: string }> | null } } | null };

export type ListFlowRunsQueryVariables = Exact<{
  orderBy?: InputMaybe<FlowRunOrder>;
  where?: InputMaybe<FlowRunWhereInput>;
}>;


export type ListFlowRunsQuery = { readonly __typename?: 'Query', readonly flowRuns: { readonly __typename?: 'FlowRunConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'FlowRunEdge', readonly node?: { readonly __typename?: 'FlowRun', readonly id: string, readonly status: FlowRunStatus, readonly error?: string | null, readonly runTimeout: string, readonly stopTimeout: string, readonly createdAt: string, readonly updatedAt: string, readonly flowID: string, readonly revisionID: string, readonly config: { readonly __typename?: 'FlowRunConfig', readonly inputs?: Record<string, unknown> | null, readonly resources?: ReadonlyArray<{ readonly __typename?: 'FlowRunResource', readonly type: string, readonly id: string, readonly nodeId: string }> | null } } | null } | null> | null } };

export type CreateFlowRunMutationVariables = Exact<{
  input: CreateFlowRunInput;
}>;


export type CreateFlowRunMutation = { readonly __typename?: 'Mutation', readonly createFlowRun?: { readonly __typename?: 'FlowRunCreated', readonly created: boolean, readonly error?: string | null, readonly flowRun?: { readonly __typename?: 'FlowRun', readonly id: string, readonly status: FlowRunStatus, readonly error?: string | null, readonly runTimeout: string, readonly stopTimeout: string, readonly createdAt: string, readonly updatedAt: string, readonly flowID: string, readonly revisionID: string, readonly config: { readonly __typename?: 'FlowRunConfig', readonly inputs?: Record<string, unknown> | null, readonly resources?: ReadonlyArray<{ readonly __typename?: 'FlowRunResource', readonly type: string, readonly id: string, readonly nodeId: string }> | null } } | null } | null };

export type StartFlowRunMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  timeout?: InputMaybe<Scalars['String']['input']>;
}>;


export type StartFlowRunMutation = { readonly __typename?: 'Mutation', readonly startFlowRun?: { readonly __typename?: 'FlowRunUpdated', readonly flowRun?: { readonly __typename?: 'FlowRun', readonly id: string, readonly status: FlowRunStatus, readonly error?: string | null, readonly runTimeout: string, readonly stopTimeout: string, readonly createdAt: string, readonly updatedAt: string, readonly flowID: string, readonly revisionID: string, readonly config: { readonly __typename?: 'FlowRunConfig', readonly inputs?: Record<string, unknown> | null, readonly resources?: ReadonlyArray<{ readonly __typename?: 'FlowRunResource', readonly type: string, readonly id: string, readonly nodeId: string }> | null } } | null } | null };

export type StopFlowRunMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  timeout: Scalars['String']['input'];
}>;


export type StopFlowRunMutation = { readonly __typename?: 'Mutation', readonly stopFlowRun?: { readonly __typename?: 'FlowRunUpdated', readonly flowRun?: { readonly __typename?: 'FlowRun', readonly id: string, readonly status: FlowRunStatus, readonly error?: string | null, readonly runTimeout: string, readonly stopTimeout: string, readonly createdAt: string, readonly updatedAt: string, readonly flowID: string, readonly revisionID: string, readonly config: { readonly __typename?: 'FlowRunConfig', readonly inputs?: Record<string, unknown> | null, readonly resources?: ReadonlyArray<{ readonly __typename?: 'FlowRunResource', readonly type: string, readonly id: string, readonly nodeId: string }> | null } } | null } | null };

export type FlowFragmentFragment = { readonly __typename?: 'Flow', readonly id: string, readonly name: string, readonly description?: string | null, readonly createdAt: string, readonly updatedAt: string };

export type GetFlowQueryVariables = Exact<{
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetFlowQuery = { readonly __typename?: 'Query', readonly flow?: { readonly __typename?: 'Flow', readonly id: string, readonly name: string, readonly description?: string | null, readonly createdAt: string, readonly updatedAt: string } | null };

export type ListFlowsQueryVariables = Exact<{
  orderBy?: InputMaybe<FlowOrder>;
  where?: InputMaybe<FlowWhereInput>;
}>;


export type ListFlowsQuery = { readonly __typename?: 'Query', readonly flows: { readonly __typename?: 'FlowConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'FlowEdge', readonly node?: { readonly __typename?: 'Flow', readonly id: string, readonly name: string, readonly description?: string | null, readonly createdAt: string, readonly updatedAt: string } | null } | null> | null } };

export type CreateFlowMutationVariables = Exact<{
  input: CreateFlowInput;
}>;


export type CreateFlowMutation = { readonly __typename?: 'Mutation', readonly createFlow?: { readonly __typename?: 'FlowCreated', readonly created: boolean, readonly error?: string | null, readonly flow?: { readonly __typename?: 'Flow', readonly id: string, readonly name: string, readonly description?: string | null, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type IntegrationFragmentFragment = { readonly __typename?: 'Integration', readonly id: string, readonly slug: string, readonly name: string, readonly apiVersion: string, readonly version: string, readonly description: string, readonly icon: string, readonly serviceNames?: ReadonlyArray<string> | null, readonly configSchema?: string | null, readonly serverConfig?: string | null, readonly organizationID: string, readonly createdAt: string, readonly updatedAt: string, readonly published?: { readonly __typename?: 'IntegrationPackage', readonly id: string, readonly checksum: string, readonly spec: string, readonly configSchema: string, readonly serviceNames: ReadonlyArray<string>, readonly integrationID: string, readonly authorID: string, readonly createdAt: string, readonly updatedAt: string } | null };

export type GetIntegrationQueryVariables = Exact<{
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetIntegrationQuery = { readonly __typename?: 'Query', readonly integration?: { readonly __typename?: 'Integration', readonly id: string, readonly slug: string, readonly name: string, readonly apiVersion: string, readonly version: string, readonly description: string, readonly icon: string, readonly serviceNames?: ReadonlyArray<string> | null, readonly configSchema?: string | null, readonly serverConfig?: string | null, readonly organizationID: string, readonly createdAt: string, readonly updatedAt: string, readonly published?: { readonly __typename?: 'IntegrationPackage', readonly id: string, readonly checksum: string, readonly spec: string, readonly configSchema: string, readonly serviceNames: ReadonlyArray<string>, readonly integrationID: string, readonly authorID: string, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type ListIntegrationsQueryVariables = Exact<{
  orderBy?: InputMaybe<IntegrationOrder>;
  where?: InputMaybe<IntegrationWhereInput>;
}>;


export type ListIntegrationsQuery = { readonly __typename?: 'Query', readonly integrations: { readonly __typename?: 'IntegrationConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'IntegrationEdge', readonly node?: { readonly __typename?: 'Integration', readonly id: string, readonly slug: string, readonly name: string, readonly apiVersion: string, readonly version: string, readonly description: string, readonly icon: string, readonly serviceNames?: ReadonlyArray<string> | null, readonly configSchema?: string | null, readonly serverConfig?: string | null, readonly organizationID: string, readonly createdAt: string, readonly updatedAt: string, readonly published?: { readonly __typename?: 'IntegrationPackage', readonly id: string, readonly checksum: string, readonly spec: string, readonly configSchema: string, readonly serviceNames: ReadonlyArray<string>, readonly integrationID: string, readonly authorID: string, readonly createdAt: string, readonly updatedAt: string } | null } | null } | null> | null } };

export type CreateIntegrationMutationVariables = Exact<{
  input: CreateIntegrationInput;
}>;


export type CreateIntegrationMutation = { readonly __typename?: 'Mutation', readonly createIntegration?: { readonly __typename?: 'IntegrationCreated', readonly created: boolean, readonly error?: string | null, readonly integration?: { readonly __typename?: 'Integration', readonly id: string, readonly slug: string, readonly name: string, readonly apiVersion: string, readonly version: string, readonly description: string, readonly icon: string, readonly serviceNames?: ReadonlyArray<string> | null, readonly configSchema?: string | null, readonly serverConfig?: string | null, readonly organizationID: string, readonly createdAt: string, readonly updatedAt: string, readonly published?: { readonly __typename?: 'IntegrationPackage', readonly id: string, readonly checksum: string, readonly spec: string, readonly configSchema: string, readonly serviceNames: ReadonlyArray<string>, readonly integrationID: string, readonly authorID: string, readonly createdAt: string, readonly updatedAt: string } | null } | null } | null };

export type IoSchemaFragmentFragment = { readonly __typename?: 'IOSchema', readonly id: string, readonly nodeType: string, readonly nodeID: string, readonly inputSchema?: string | null, readonly outputSchema?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly organizationID?: string | null };

export type GetIoSchemaQueryVariables = Exact<{
  nodeType: Scalars['String']['input'];
  nodeId: Scalars['ID']['input'];
}>;


export type GetIoSchemaQuery = { readonly __typename?: 'Query', readonly ioSchema?: { readonly __typename?: 'IOSchema', readonly id: string, readonly nodeType: string, readonly nodeID: string, readonly inputSchema?: string | null, readonly outputSchema?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly organizationID?: string | null } | null };

export type ListIoSchemasQueryVariables = Exact<{
  orderBy?: InputMaybe<IoSchemaOrder>;
  where?: InputMaybe<IoSchemaWhereInput>;
}>;


export type ListIoSchemasQuery = { readonly __typename?: 'Query', readonly ioSchemas: { readonly __typename?: 'IOSchemaConnection', readonly edges?: ReadonlyArray<{ readonly __typename?: 'IOSchemaEdge', readonly node?: { readonly __typename?: 'IOSchema', readonly id: string, readonly nodeType: string, readonly nodeID: string, readonly inputSchema?: string | null, readonly outputSchema?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly organizationID?: string | null } | null } | null> | null } };

export type ModelTypeFragmentFragment = { readonly __typename?: 'ModelType', readonly id: string, readonly slug: string, readonly name: string, readonly description: string, readonly category: string, readonly sourceURL: string, readonly sourceVersion?: string | null, readonly minSources: number, readonly minPerType: number, readonly maxPerType: number, readonly modeler: ModelerType, readonly configTemplate?: Record<string, unknown> | null, readonly createdAt: string, readonly updatedAt: string };

export type GetModelTypeQueryVariables = Exact<{
  modelTypeId?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetModelTypeQuery = { readonly __typename?: 'Query', readonly modelType?: { readonly __typename?: 'ModelType', readonly id: string, readonly slug: string, readonly name: string, readonly description: string, readonly category: string, readonly sourceURL: string, readonly sourceVersion?: string | null, readonly minSources: number, readonly minPerType: number, readonly maxPerType: number, readonly modeler: ModelerType, readonly configTemplate?: Record<string, unknown> | null, readonly createdAt: string, readonly updatedAt: string } | null };

export type ListModelTypesQueryVariables = Exact<{
  orderBy?: InputMaybe<ModelTypeOrder>;
  where?: InputMaybe<ModelTypeWhereInput>;
}>;


export type ListModelTypesQuery = { readonly __typename?: 'Query', readonly modelTypes: { readonly __typename?: 'ModelTypeConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'ModelTypeEdge', readonly node?: { readonly __typename?: 'ModelType', readonly id: string, readonly slug: string, readonly name: string, readonly description: string, readonly category: string, readonly sourceURL: string, readonly sourceVersion?: string | null, readonly minSources: number, readonly minPerType: number, readonly maxPerType: number, readonly modeler: ModelerType, readonly configTemplate?: Record<string, unknown> | null, readonly createdAt: string, readonly updatedAt: string } | null } | null> | null } };

export type ModelFragmentFragment = { readonly __typename?: 'Model', readonly id: string, readonly name: string, readonly metadata?: Record<string, unknown> | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly typeID: string, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string };

export type GetModelQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetModelQuery = { readonly __typename?: 'Query', readonly model?: { readonly __typename?: 'Model', readonly id: string, readonly name: string, readonly metadata?: Record<string, unknown> | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly typeID: string, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string } | null };

export type ListModelsQueryVariables = Exact<{
  orderBy?: InputMaybe<ModelOrder>;
  where?: InputMaybe<ModelWhereInput>;
}>;


export type ListModelsQuery = { readonly __typename?: 'Query', readonly models: { readonly __typename?: 'ModelConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'ModelEdge', readonly node?: { readonly __typename?: 'Model', readonly id: string, readonly name: string, readonly metadata?: Record<string, unknown> | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly typeID: string, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string } | null } | null> | null } };

export type CreateModelMutationVariables = Exact<{
  input: CreateModelInput;
}>;


export type CreateModelMutation = { readonly __typename?: 'Mutation', readonly createModel?: { readonly __typename?: 'ModelCreated', readonly created: boolean, readonly error?: string | null, readonly model?: { readonly __typename?: 'Model', readonly id: string, readonly name: string, readonly metadata?: Record<string, unknown> | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly typeID: string, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type SyncModelMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type SyncModelMutation = { readonly __typename?: 'Mutation', readonly syncModel?: { readonly __typename?: 'ModelUpdated', readonly model?: { readonly __typename?: 'Model', readonly id: string, readonly name: string, readonly metadata?: Record<string, unknown> | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly typeID: string, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type UpdateModelMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateModelInput;
}>;


export type UpdateModelMutation = { readonly __typename?: 'Mutation', readonly updateModel?: { readonly __typename?: 'ModelUpdated', readonly updated: boolean, readonly error?: string | null, readonly model?: { readonly __typename?: 'Model', readonly id: string, readonly name: string, readonly metadata?: Record<string, unknown> | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly typeID: string, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type DeleteModelMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteModelMutation = { readonly __typename?: 'Mutation', readonly deleteModel?: { readonly __typename?: 'ModelDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } | null };

export type OrganizationFragmentFragment = { readonly __typename?: 'Organization', readonly id: string, readonly name: string, readonly slug: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string };

export type GetOrganizationQueryVariables = Exact<{
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetOrganizationQuery = { readonly __typename?: 'Query', readonly organization?: { readonly __typename?: 'Organization', readonly id: string, readonly name: string, readonly slug: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string } | null };

export type ListOrganizationsQueryVariables = Exact<{
  orderBy?: InputMaybe<OrganizationOrder>;
  where?: InputMaybe<OrganizationWhereInput>;
}>;


export type ListOrganizationsQuery = { readonly __typename?: 'Query', readonly organizations: { readonly __typename?: 'OrganizationConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'OrganizationEdge', readonly node?: { readonly __typename?: 'Organization', readonly id: string, readonly name: string, readonly slug: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string } | null } | null> | null } };

export type CreateOrganizationMutationVariables = Exact<{
  input: CreateOrganizationInput;
}>;


export type CreateOrganizationMutation = { readonly __typename?: 'Mutation', readonly createOrganization?: { readonly __typename?: 'OrganizationCreated', readonly created: boolean, readonly error?: string | null, readonly organization?: { readonly __typename?: 'Organization', readonly id: string, readonly name: string, readonly slug: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type UpdateOrganizationMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateOrganizationInput;
}>;


export type UpdateOrganizationMutation = { readonly __typename?: 'Mutation', readonly updateOrganization?: { readonly __typename?: 'OrganizationUpdated', readonly updated: boolean, readonly error?: string | null, readonly organization?: { readonly __typename?: 'Organization', readonly id: string, readonly name: string, readonly slug: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type DeleteOrganizationMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteOrganizationMutation = { readonly __typename?: 'Mutation', readonly deleteOrganization?: { readonly __typename?: 'OrganizationDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } | null };

export type IntegrationPackageFragmentFragment = { readonly __typename?: 'IntegrationPackage', readonly id: string, readonly checksum: string, readonly spec: string, readonly configSchema: string, readonly serviceNames: ReadonlyArray<string>, readonly integrationID: string, readonly authorID: string, readonly createdAt: string, readonly updatedAt: string };

export type ListPackagesQueryVariables = Exact<{
  orderBy?: InputMaybe<IntegrationPackageOrder>;
  where?: InputMaybe<IntegrationPackageWhereInput>;
}>;


export type ListPackagesQuery = { readonly __typename?: 'Query', readonly packages: { readonly __typename?: 'IntegrationPackageConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'IntegrationPackageEdge', readonly node?: { readonly __typename?: 'IntegrationPackage', readonly id: string, readonly checksum: string, readonly spec: string, readonly configSchema: string, readonly serviceNames: ReadonlyArray<string>, readonly integrationID: string, readonly authorID: string, readonly createdAt: string, readonly updatedAt: string } | null } | null> | null } };

export type SyncPackageMutationVariables = Exact<{
  input: SyncPackageInput;
}>;


export type SyncPackageMutation = { readonly __typename?: 'Mutation', readonly syncPackage?: { readonly __typename?: 'IntegrationPackageUpdated', readonly integrationPackage?: { readonly __typename?: 'IntegrationPackage', readonly id: string, readonly checksum: string, readonly spec: string, readonly configSchema: string, readonly serviceNames: ReadonlyArray<string>, readonly integrationID: string, readonly authorID: string, readonly createdAt: string, readonly updatedAt: string, readonly integration: { readonly __typename?: 'Integration', readonly id: string, readonly slug: string, readonly name: string, readonly apiVersion: string, readonly version: string, readonly description: string, readonly icon: string, readonly serviceNames?: ReadonlyArray<string> | null, readonly configSchema?: string | null, readonly serverConfig?: string | null, readonly organizationID: string, readonly createdAt: string, readonly updatedAt: string, readonly published?: { readonly __typename?: 'IntegrationPackage', readonly id: string, readonly checksum: string, readonly spec: string, readonly configSchema: string, readonly serviceNames: ReadonlyArray<string>, readonly integrationID: string, readonly authorID: string, readonly createdAt: string, readonly updatedAt: string } | null } } | null } | null };

export type PersonalAccessTokenFragmentFragment = { readonly __typename?: 'PersonalAccessToken', readonly id: string, readonly name: string, readonly token: string, readonly userID: string, readonly expiresAt: string, readonly createdAt: string, readonly updatedAt: string };

export type GetPersonalAccessTokenQueryVariables = Exact<{
  id?: InputMaybe<Scalars['ID']['input']>;
  token?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetPersonalAccessTokenQuery = { readonly __typename?: 'Query', readonly personalAccessToken?: { readonly __typename?: 'PersonalAccessToken', readonly id: string, readonly name: string, readonly token: string, readonly userID: string, readonly expiresAt: string, readonly createdAt: string, readonly updatedAt: string } | null };

export type ListPersonalAccessTokensQueryVariables = Exact<{
  orderBy?: InputMaybe<PersonalAccessTokenOrder>;
  where?: InputMaybe<PersonalAccessTokenWhereInput>;
}>;


export type ListPersonalAccessTokensQuery = { readonly __typename?: 'Query', readonly personalAccessTokens: { readonly __typename?: 'PersonalAccessTokenConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'PersonalAccessTokenEdge', readonly node?: { readonly __typename?: 'PersonalAccessToken', readonly id: string, readonly name: string, readonly token: string, readonly userID: string, readonly expiresAt: string, readonly createdAt: string, readonly updatedAt: string } | null } | null> | null } };

export type CreatePersonalAccessTokenMutationVariables = Exact<{
  input: CreatePersonalAccessTokenInput;
}>;


export type CreatePersonalAccessTokenMutation = { readonly __typename?: 'Mutation', readonly createPersonalAccessToken?: { readonly __typename?: 'PersonalAccessTokenCreated', readonly created: boolean, readonly error?: string | null, readonly personalAccessToken?: { readonly __typename?: 'PersonalAccessToken', readonly id: string, readonly name: string, readonly token: string, readonly userID: string, readonly expiresAt: string, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type UpdatePersonalAccessTokenMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdatePersonalAccessTokenInput;
}>;


export type UpdatePersonalAccessTokenMutation = { readonly __typename?: 'Mutation', readonly updatePersonalAccessToken?: { readonly __typename?: 'PersonalAccessTokenUpdated', readonly updated: boolean, readonly error?: string | null, readonly personalAccessToken?: { readonly __typename?: 'PersonalAccessToken', readonly id: string, readonly name: string, readonly token: string, readonly userID: string, readonly expiresAt: string, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type RotatePersonalAccessTokenMutationVariables = Exact<{
  id?: InputMaybe<Scalars['ID']['input']>;
  token?: InputMaybe<Scalars['String']['input']>;
}>;


export type RotatePersonalAccessTokenMutation = { readonly __typename?: 'Mutation', readonly rotatePersonalAccessToken?: { readonly __typename?: 'PersonalAccessTokenUpdated', readonly updated: boolean, readonly error?: string | null, readonly personalAccessToken?: { readonly __typename?: 'PersonalAccessToken', readonly id: string, readonly name: string, readonly token: string, readonly userID: string, readonly expiresAt: string, readonly createdAt: string, readonly updatedAt: string } | null } | null };

export type DeletePersonalAccessTokenMutationVariables = Exact<{
  token: Scalars['String']['input'];
}>;


export type DeletePersonalAccessTokenMutation = { readonly __typename?: 'Mutation', readonly deletePersonalAccessToken?: { readonly __typename?: 'PersonalAccessTokenDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } | null };

export type SchemaRefFragmentFragment = { readonly __typename?: 'SchemaRef', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null };

export type GetSchemaQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetSchemaQuery = { readonly __typename?: 'Query', readonly schema?: { readonly __typename?: 'SchemaRef', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null } | null };

export type ListSchemasQueryVariables = Exact<{
  orderBy?: InputMaybe<ReadonlyArray<SchemaRefOrder> | SchemaRefOrder>;
  where?: InputMaybe<SchemaRefWhereInput>;
}>;


export type ListSchemasQuery = { readonly __typename?: 'Query', readonly schemas: { readonly __typename?: 'SchemaRefConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'SchemaRefEdge', readonly node?: { readonly __typename?: 'SchemaRef', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null } | null } | null> | null } };

export type SchemasDeletedQueryVariables = Exact<{ [key: string]: never; }>;


export type SchemasDeletedQuery = { readonly __typename?: 'Query', readonly schemasDeleted: ReadonlyArray<{ readonly __typename?: 'SchemaRef', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null } | null> };

export type CreateSchemaMutationVariables = Exact<{
  input: CreateSchemaRefInput;
}>;


export type CreateSchemaMutation = { readonly __typename?: 'Mutation', readonly createSchema?: { readonly __typename?: 'SchemaRefCreated', readonly created: boolean, readonly error?: string | null, readonly schemaRef?: { readonly __typename?: 'SchemaRef', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null } | null } | null };

export type SyncSchemaMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  autoSync?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type SyncSchemaMutation = { readonly __typename?: 'Mutation', readonly syncSchema?: { readonly __typename?: 'SchemaRefUpdated', readonly updated: boolean, readonly error?: string | null, readonly schemaRef?: { readonly __typename?: 'SchemaRef', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null } | null } | null };

export type UpdateSchemaMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateSchemaRefInput;
}>;


export type UpdateSchemaMutation = { readonly __typename?: 'Mutation', readonly updateSchema?: { readonly __typename?: 'SchemaRefUpdated', readonly updated: boolean, readonly error?: string | null, readonly schemaRef?: { readonly __typename?: 'SchemaRef', readonly id: string, readonly name: string, readonly description?: string | null, readonly alias: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly catalogID: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null } | null } | null };

export type DeleteSchemaMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type DeleteSchemaMutation = { readonly __typename?: 'Mutation', readonly deleteSchema?: { readonly __typename?: 'SchemaRefDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } | null };

export type SearchQueryVariables = Exact<{
  q: Scalars['String']['input'];
  limit: Scalars['Int']['input'];
  offset: Scalars['Int']['input'];
  stype: SearchType;
  by?: InputMaybe<SearchByInput>;
  where?: InputMaybe<SearchWhereInput>;
  view?: InputMaybe<SearchView>;
}>;


export type SearchQuery = { readonly __typename?: 'Query', readonly search: { readonly __typename?: 'SearchResults', readonly totalCount: number, readonly results: ReadonlyArray<{ readonly __typename?: 'SearchResult', readonly rank: number, readonly text: string, readonly node:
        | { readonly __typename?: 'Catalog', readonly id: string, readonly type: 'Catalog' }
        | { readonly __typename?: 'ColumnRef', readonly id: string, readonly type: 'ColumnRef' }
        | { readonly __typename?: 'Connection', readonly id: string, readonly type: 'Connection' }
        | { readonly __typename?: 'ConnectionUser', readonly id: string, readonly type: 'ConnectionUser' }
        | { readonly __typename?: 'Destination', readonly id: string, readonly type: 'Destination' }
        | { readonly __typename?: 'EventSource', readonly id: string, readonly type: 'EventSource' }
        | { readonly __typename?: 'Flow', readonly id: string, readonly type: 'Flow' }
        | { readonly __typename?: 'FlowResource', readonly id: string, readonly type: 'FlowResource' }
        | { readonly __typename?: 'FlowRevision', readonly id: string, readonly type: 'FlowRevision' }
        | { readonly __typename?: 'FlowRun', readonly id: string, readonly type: 'FlowRun' }
        | { readonly __typename?: 'GeoFeature', readonly id: string, readonly type: 'GeoFeature' }
        | { readonly __typename?: 'GeoLayer', readonly id: string, readonly type: 'GeoLayer' }
        | { readonly __typename?: 'GeoMap', readonly id: string, readonly type: 'GeoMap' }
        | { readonly __typename?: 'GeoSource', readonly id: string, readonly type: 'GeoSource' }
        | { readonly __typename?: 'IOSchema', readonly id: string, readonly type: 'IOSchema' }
        | { readonly __typename?: 'Integration', readonly id: string, readonly type: 'Integration' }
        | { readonly __typename?: 'IntegrationPackage', readonly id: string, readonly type: 'IntegrationPackage' }
        | { readonly __typename?: 'Model', readonly id: string, readonly type: 'Model' }
        | { readonly __typename?: 'ModelType', readonly id: string, readonly type: 'ModelType' }
        | { readonly __typename?: 'Organization', readonly id: string, readonly type: 'Organization' }
        | { readonly __typename?: 'PersonalAccessToken', readonly id: string, readonly type: 'PersonalAccessToken' }
        | { readonly __typename?: 'SQLQuery', readonly id: string, readonly type: 'SQLQuery' }
        | { readonly __typename?: 'SchemaRef', readonly id: string, readonly type: 'SchemaRef' }
        | { readonly __typename?: 'SearchLexeme', readonly id: string, readonly type: 'SearchLexeme' }
        | { readonly __typename?: 'SearchSemantic', readonly id: string, readonly type: 'SearchSemantic' }
        | { readonly __typename?: 'Source', readonly id: string, readonly type: 'Source' }
        | { readonly __typename?: 'SourceType', readonly id: string, readonly type: 'SourceType' }
        | { readonly __typename?: 'Space', readonly id: string, readonly type: 'Space' }
        | { readonly __typename?: 'TableRef', readonly id: string, readonly type: 'TableRef' }
        | { readonly __typename?: 'User', readonly id: string, readonly type: 'User' }
       } | null> } };

export type SourceTypeFragmentFragment = { readonly __typename?: 'SourceType', readonly id: string, readonly slug: string, readonly name: string, readonly description: string, readonly category: string, readonly iconURL: string, readonly configSchema: string, readonly createdAt: string, readonly updatedAt: string, readonly providerID: string };

export type GetSourceTypeQueryVariables = Exact<{
  id?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetSourceTypeQuery = { readonly __typename?: 'Query', readonly sourceType?: { readonly __typename?: 'SourceType', readonly id: string, readonly slug: string, readonly name: string, readonly description: string, readonly category: string, readonly iconURL: string, readonly configSchema: string, readonly createdAt: string, readonly updatedAt: string, readonly providerID: string } | null };

export type ListSourceTypesQueryVariables = Exact<{
  orderBy?: InputMaybe<SourceTypeOrder>;
  where?: InputMaybe<SourceTypeWhereInput>;
}>;


export type ListSourceTypesQuery = { readonly __typename?: 'Query', readonly sourceTypes: { readonly __typename?: 'SourceTypeConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'SourceTypeEdge', readonly node?: { readonly __typename?: 'SourceType', readonly id: string, readonly slug: string, readonly name: string, readonly description: string, readonly category: string, readonly iconURL: string, readonly configSchema: string, readonly createdAt: string, readonly updatedAt: string, readonly providerID: string } | null } | null> | null } };

export type SourceFragmentFragment = { readonly __typename?: 'Source', readonly id: string, readonly name: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly syncFreq: number, readonly syncTime: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly providerID: string, readonly destinationID: string, readonly catalogID: string, readonly connectionID: string, readonly externalReports?: ReadonlyArray<{ readonly __typename?: 'ExternalReport', readonly provider: ReportProvider, readonly id: string, readonly url: string, readonly dataSource?: { readonly __typename?: 'ExternalDataSource', readonly provider: ReportProvider, readonly id: string, readonly url: string } | null } | null> | null };

export type GetSourceQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetSourceQuery = { readonly __typename?: 'Query', readonly source?: { readonly __typename?: 'Source', readonly id: string, readonly name: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly syncFreq: number, readonly syncTime: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly providerID: string, readonly destinationID: string, readonly catalogID: string, readonly connectionID: string, readonly externalReports?: ReadonlyArray<{ readonly __typename?: 'ExternalReport', readonly provider: ReportProvider, readonly id: string, readonly url: string, readonly dataSource?: { readonly __typename?: 'ExternalDataSource', readonly provider: ReportProvider, readonly id: string, readonly url: string } | null } | null> | null } | null };

export type ListSourcesQueryVariables = Exact<{
  orderBy?: InputMaybe<SourceOrder>;
  where?: InputMaybe<SourceWhereInput>;
}>;


export type ListSourcesQuery = { readonly __typename?: 'Query', readonly sources: { readonly __typename?: 'SourceConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'SourceEdge', readonly node?: { readonly __typename?: 'Source', readonly id: string, readonly name: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly syncFreq: number, readonly syncTime: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly providerID: string, readonly destinationID: string, readonly catalogID: string, readonly connectionID: string, readonly externalReports?: ReadonlyArray<{ readonly __typename?: 'ExternalReport', readonly provider: ReportProvider, readonly id: string, readonly url: string, readonly dataSource?: { readonly __typename?: 'ExternalDataSource', readonly provider: ReportProvider, readonly id: string, readonly url: string } | null } | null> | null } | null } | null> | null } };

export type CreateSourceMutationVariables = Exact<{
  input: CreateSourceInput;
}>;


export type CreateSourceMutation = { readonly __typename?: 'Mutation', readonly createSource: { readonly __typename?: 'SourceCreated', readonly created: boolean, readonly error?: string | null, readonly source?: { readonly __typename?: 'Source', readonly id: string, readonly name: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly syncFreq: number, readonly syncTime: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly providerID: string, readonly destinationID: string, readonly catalogID: string, readonly connectionID: string, readonly externalReports?: ReadonlyArray<{ readonly __typename?: 'ExternalReport', readonly provider: ReportProvider, readonly id: string, readonly url: string, readonly dataSource?: { readonly __typename?: 'ExternalDataSource', readonly provider: ReportProvider, readonly id: string, readonly url: string } | null } | null> | null } | null } };

export type SyncSourceMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type SyncSourceMutation = { readonly __typename?: 'Mutation', readonly syncSource: { readonly __typename?: 'SourceUpdated', readonly updated: boolean, readonly error?: string | null, readonly source?: { readonly __typename?: 'Source', readonly id: string, readonly name: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly syncFreq: number, readonly syncTime: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly providerID: string, readonly destinationID: string, readonly catalogID: string, readonly connectionID: string, readonly externalReports?: ReadonlyArray<{ readonly __typename?: 'ExternalReport', readonly provider: ReportProvider, readonly id: string, readonly url: string, readonly dataSource?: { readonly __typename?: 'ExternalDataSource', readonly provider: ReportProvider, readonly id: string, readonly url: string } | null } | null> | null } | null } };

export type UpdateSourceMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateSourceInput;
}>;


export type UpdateSourceMutation = { readonly __typename?: 'Mutation', readonly updateSource: { readonly __typename?: 'SourceUpdated', readonly updated: boolean, readonly error?: string | null, readonly source?: { readonly __typename?: 'Source', readonly id: string, readonly name: string, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly syncFreq: number, readonly syncTime: string, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly providerID: string, readonly destinationID: string, readonly catalogID: string, readonly connectionID: string, readonly externalReports?: ReadonlyArray<{ readonly __typename?: 'ExternalReport', readonly provider: ReportProvider, readonly id: string, readonly url: string, readonly dataSource?: { readonly __typename?: 'ExternalDataSource', readonly provider: ReportProvider, readonly id: string, readonly url: string } | null } | null> | null } | null } };

export type DeleteSourceMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type DeleteSourceMutation = { readonly __typename?: 'Mutation', readonly deleteSource: { readonly __typename?: 'SourceDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } };

export type SpaceFragmentFragment = { readonly __typename?: 'Space', readonly id: string, readonly slug: string, readonly name: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string, readonly organizationID: string };

export type GetSpaceQueryVariables = Exact<{
  spaceId?: InputMaybe<Scalars['ID']['input']>;
  slug?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetSpaceQuery = { readonly __typename?: 'Query', readonly space?: { readonly __typename?: 'Space', readonly id: string, readonly slug: string, readonly name: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string, readonly organizationID: string } | null };

export type ListSpacesQueryVariables = Exact<{
  orderBy?: InputMaybe<SpaceOrder>;
  where?: InputMaybe<SpaceWhereInput>;
}>;


export type ListSpacesQuery = { readonly __typename?: 'Query', readonly spaces: { readonly __typename?: 'SpaceConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'SpaceEdge', readonly node?: { readonly __typename?: 'Space', readonly id: string, readonly slug: string, readonly name: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string, readonly organizationID: string } | null } | null> | null } };

export type CreateSpaceMutationVariables = Exact<{
  input: CreateSpaceInput;
}>;


export type CreateSpaceMutation = { readonly __typename?: 'Mutation', readonly createSpace?: { readonly __typename?: 'SpaceCreated', readonly created: boolean, readonly error?: string | null, readonly space?: { readonly __typename?: 'Space', readonly id: string, readonly slug: string, readonly name: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string, readonly organizationID: string } | null } | null };

export type UpdateSpaceMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateSpaceInput;
}>;


export type UpdateSpaceMutation = { readonly __typename?: 'Mutation', readonly updateSpace?: { readonly __typename?: 'SpaceUpdated', readonly updated: boolean, readonly error?: string | null, readonly space?: { readonly __typename?: 'Space', readonly id: string, readonly slug: string, readonly name: string, readonly tzName: string, readonly createdAt: string, readonly updatedAt: string, readonly organizationID: string } | null } | null };

export type DeleteSpaceMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteSpaceMutation = { readonly __typename?: 'Mutation', readonly deleteSpace?: { readonly __typename?: 'SpaceDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } | null };

export type TableRefFragmentFragment = { readonly __typename?: 'TableRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly tableType?: string | null, readonly totalRows?: number | null, readonly totalBytes?: number | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly schemaID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null };

export type GetTableQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetTableQuery = { readonly __typename?: 'Query', readonly table?: { readonly __typename?: 'TableRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly tableType?: string | null, readonly totalRows?: number | null, readonly totalBytes?: number | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly schemaID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null } | null };

export type ListTablesQueryVariables = Exact<{
  orderBy?: InputMaybe<ReadonlyArray<TableRefOrder> | TableRefOrder>;
  where?: InputMaybe<TableRefWhereInput>;
}>;


export type ListTablesQuery = { readonly __typename?: 'Query', readonly tables: { readonly __typename?: 'TableRefConnection', readonly totalCount: number, readonly edges?: ReadonlyArray<{ readonly __typename?: 'TableRefEdge', readonly node?: { readonly __typename?: 'TableRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly tableType?: string | null, readonly totalRows?: number | null, readonly totalBytes?: number | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly schemaID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null } | null } | null> | null } };

export type TablesDeletedQueryVariables = Exact<{ [key: string]: never; }>;


export type TablesDeletedQuery = { readonly __typename?: 'Query', readonly tablesDeleted: ReadonlyArray<{ readonly __typename?: 'TableRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly tableType?: string | null, readonly totalRows?: number | null, readonly totalBytes?: number | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly schemaID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null } | null> };

export type CreateTableMutationVariables = Exact<{
  input: CreateTableInput;
}>;


export type CreateTableMutation = { readonly __typename?: 'Mutation', readonly createTable: { readonly __typename?: 'TableRefCreated', readonly created: boolean, readonly error?: string | null, readonly tableRef?: { readonly __typename?: 'TableRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly tableType?: string | null, readonly totalRows?: number | null, readonly totalBytes?: number | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly schemaID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null } | null } };

export type SyncTableMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type SyncTableMutation = { readonly __typename?: 'Mutation', readonly syncTable: { readonly __typename?: 'TableRefUpdated', readonly updated: boolean, readonly error?: string | null, readonly tableRef?: { readonly __typename?: 'TableRef', readonly id: string, readonly name: string, readonly alias: string, readonly description?: string | null, readonly tableType?: string | null, readonly totalRows?: number | null, readonly totalBytes?: number | null, readonly autoSync: boolean, readonly syncStatus: SyncStatus, readonly syncError?: string | null, readonly syncedAt?: string | null, readonly createdAt: string, readonly updatedAt: string, readonly deletedAt?: string | null, readonly schemaID: string, readonly fields?: ReadonlyArray<{ readonly __typename?: 'Field', readonly name: string, readonly description?: string | null, readonly nullable: boolean, readonly repeated?: boolean | null, readonly type: { readonly __typename?: 'DataType', readonly nativeType: string, readonly jsonType?: string | null, readonly geoType?: string | null, readonly metadata?: Record<string, unknown> | null } } | null> | null } | null } };

export type DeleteTableMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  softDelete?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type DeleteTableMutation = { readonly __typename?: 'Mutation', readonly deleteTable: { readonly __typename?: 'TableRefDeleted', readonly deleted: boolean, readonly error?: string | null, readonly id?: string | null } };

export const CatalogFragmentFragmentDoc = gql`
    fragment CatalogFragment on Catalog {
  id
  name
  description
  alias
  label
  queryDialect
  autoSync
  syncStatus
  syncError
  syncedAt
  createdAt
  updatedAt
  deletedAt
  connectionID
  organizationID
}
    `;
export const ColumnRefFragmentFragmentDoc = gql`
    fragment ColumnRefFragment on ColumnRef {
  id
  name
  alias
  description
  dtype
  ordinal
  nullable
  repeated
  primaryKey
  fields {
    name
    description
    type {
      nativeType
      jsonType
      geoType
      metadata
    }
    nullable
    repeated
  }
  createdAt
  updatedAt
  deletedAt
  tableID
}
    `;
export const ConnectionFragmentFragmentDoc = gql`
    fragment ConnectionFragment on Connection {
  id
  slug
  name
  visibility
  createdAt
  updatedAt
  integrationID
  organizationID
}
    `;
export const DestinationFragmentFragmentDoc = gql`
    fragment DestinationFragment on Destination {
  id
  name
  tzName
  createdAt
  updatedAt
  providerID
  connectionID
  organizationID
}
    `;
export const EventSourceFragmentFragmentDoc = gql`
    fragment EventSourceFragment on EventSource {
  id
  slug
  name
  status
  error
  strategy
  pullFreq
  pushUrl
  createdAt
  updatedAt
  connectionID
}
    `;
export const FlowRevisionFragmentFragmentDoc = gql`
    fragment FlowRevisionFragment on FlowRevision {
  id
  checksum
  body(format: YAML) {
    raw
  }
  createdAt
  updatedAt
  flowID
}
    `;
export const FlowRunFragmentFragmentDoc = gql`
    fragment FlowRunFragment on FlowRun {
  id
  status
  error
  runTimeout
  stopTimeout
  config {
    inputs
    resources {
      type
      id
      nodeId
    }
  }
  createdAt
  updatedAt
  flowID
  revisionID
}
    `;
export const FlowFragmentFragmentDoc = gql`
    fragment FlowFragment on Flow {
  id
  name
  description
  createdAt
  updatedAt
}
    `;
export const IntegrationPackageFragmentFragmentDoc = gql`
    fragment IntegrationPackageFragment on IntegrationPackage {
  id
  checksum
  spec
  configSchema
  serviceNames
  integrationID
  authorID
  createdAt
  updatedAt
}
    `;
export const IntegrationFragmentFragmentDoc = gql`
    fragment IntegrationFragment on Integration {
  id
  slug
  name
  apiVersion
  version
  description
  icon
  serviceNames
  configSchema
  serverConfig
  published {
    ...IntegrationPackageFragment
  }
  organizationID
  createdAt
  updatedAt
}
    ${IntegrationPackageFragmentFragmentDoc}`;
export const IoSchemaFragmentFragmentDoc = gql`
    fragment IoSchemaFragment on IOSchema {
  id
  nodeType
  nodeID
  inputSchema
  outputSchema
  createdAt
  updatedAt
  organizationID
}
    `;
export const ModelTypeFragmentFragmentDoc = gql`
    fragment ModelTypeFragment on ModelType {
  id
  slug
  name
  description
  category
  sourceURL
  sourceVersion
  minSources
  minPerType
  maxPerType
  modeler
  configTemplate
  createdAt
  updatedAt
}
    `;
export const ModelFragmentFragmentDoc = gql`
    fragment ModelFragment on Model {
  id
  name
  metadata
  autoSync
  syncStatus
  syncError
  syncedAt
  typeID
  catalogID
  createdAt
  updatedAt
}
    `;
export const OrganizationFragmentFragmentDoc = gql`
    fragment OrganizationFragment on Organization {
  id
  name
  slug
  tzName
  createdAt
  updatedAt
}
    `;
export const PersonalAccessTokenFragmentFragmentDoc = gql`
    fragment PersonalAccessTokenFragment on PersonalAccessToken {
  id
  name
  token
  userID
  expiresAt
  createdAt
  updatedAt
}
    `;
export const SchemaRefFragmentFragmentDoc = gql`
    fragment SchemaRefFragment on SchemaRef {
  id
  name
  description
  alias
  autoSync
  syncStatus
  syncError
  syncedAt
  catalogID
  createdAt
  updatedAt
  deletedAt
}
    `;
export const SourceTypeFragmentFragmentDoc = gql`
    fragment SourceTypeFragment on SourceType {
  id
  slug
  name
  description
  category
  iconURL
  configSchema
  createdAt
  updatedAt
  providerID
}
    `;
export const SourceFragmentFragmentDoc = gql`
    fragment SourceFragment on Source {
  id
  name
  externalReports {
    provider
    id
    url
    dataSource {
      provider
      id
      url
    }
  }
  autoSync
  syncStatus
  syncError
  syncedAt
  syncFreq
  syncTime
  createdAt
  updatedAt
  deletedAt
  providerID
  destinationID
  catalogID
  connectionID
}
    `;
export const SpaceFragmentFragmentDoc = gql`
    fragment SpaceFragment on Space {
  id
  slug
  name
  tzName
  createdAt
  updatedAt
  organizationID
}
    `;
export const TableRefFragmentFragmentDoc = gql`
    fragment TableRefFragment on TableRef {
  id
  name
  alias
  description
  fields {
    name
    description
    type {
      nativeType
      jsonType
      geoType
      metadata
    }
    nullable
    repeated
  }
  tableType
  totalRows
  totalBytes
  autoSync
  syncStatus
  syncError
  syncedAt
  createdAt
  updatedAt
  deletedAt
  schemaID
}
    `;
export const GetCatalogDocument = gql`
    query GetCatalog($id: ID!) {
  catalog(id: $id) {
    ...CatalogFragment
  }
}
    ${CatalogFragmentFragmentDoc}`;
export const ListCatalogsDocument = gql`
    query ListCatalogs($orderBy: CatalogOrder, $where: CatalogWhereInput) {
  catalogs(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...CatalogFragment
      }
    }
    totalCount
  }
}
    ${CatalogFragmentFragmentDoc}`;
export const CreateCatalogDocument = gql`
    mutation CreateCatalog($input: CreateCatalogInput!) {
  createCatalog(input: $input) {
    created
    error
    catalog {
      ...CatalogFragment
    }
  }
}
    ${CatalogFragmentFragmentDoc}`;
export const SyncCatalogDocument = gql`
    mutation SyncCatalog($id: ID!) {
  syncCatalog(id: $id) {
    updated
    error
    catalog {
      ...CatalogFragment
    }
  }
}
    ${CatalogFragmentFragmentDoc}`;
export const UpdateCatalogDocument = gql`
    mutation UpdateCatalog($id: ID!, $input: UpdateCatalogInput!) {
  updateCatalog(id: $id, input: $input) {
    updated
    error
    catalog {
      ...CatalogFragment
    }
  }
}
    ${CatalogFragmentFragmentDoc}`;
export const MutationDocument = gql`
    mutation Mutation($id: ID!, $softDelete: Boolean) {
  deleteCatalog(id: $id, softDelete: $softDelete) {
    id
  }
}
    `;
export const GetColumnDocument = gql`
    query GetColumn($id: ID!) {
  column(id: $id) {
    ...ColumnRefFragment
  }
}
    ${ColumnRefFragmentFragmentDoc}`;
export const ListColumnsDocument = gql`
    query ListColumns($orderBy: [ColumnRefOrder!], $where: ColumnRefWhereInput) {
  columns(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...ColumnRefFragment
      }
    }
    totalCount
  }
}
    ${ColumnRefFragmentFragmentDoc}`;
export const CreateColumnDocument = gql`
    mutation CreateColumn($input: CreateColumnInput!) {
  createColumn(input: $input) {
    created
    error
    columnRef {
      ...ColumnRefFragment
    }
  }
}
    ${ColumnRefFragmentFragmentDoc}`;
export const DeleteColumnDocument = gql`
    mutation DeleteColumn($id: ID!, $softDelete: Boolean) {
  deleteColumn(id: $id, softDelete: $softDelete) {
    deleted
    error
    id
  }
}
    `;
export const GetConnectionDocument = gql`
    query GetConnection($connectionId: ID, $slug: String) {
  connection(id: $connectionId, slug: $slug) {
    ...ConnectionFragment
  }
}
    ${ConnectionFragmentFragmentDoc}`;
export const ListConnectionsDocument = gql`
    query ListConnections($orderBy: ConnectionOrder, $where: ConnectionWhereInput) {
  connections(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...ConnectionFragment
      }
    }
    totalCount
  }
}
    ${ConnectionFragmentFragmentDoc}`;
export const CheckConnectionDocument = gql`
    query CheckConnection($id: ID, $slug: String) {
  checkConnection(id: $id, slug: $slug) {
    authCheck {
      type
      required
      success
      error
    }
    configCheck {
      success
      error
    }
  }
}
    `;
export const CreateConnectionDocument = gql`
    mutation CreateConnection($input: CreateConnectionInput!) {
  createConnection(input: $input) {
    created
    error
    connection {
      ...ConnectionFragment
    }
  }
}
    ${ConnectionFragmentFragmentDoc}`;
export const UpdateConnectionDocument = gql`
    mutation UpdateConnection($id: ID!, $input: UpdateConnectionInput!) {
  updateConnection(id: $id, input: $input) {
    updated
    error
    connection {
      ...ConnectionFragment
    }
  }
}
    ${ConnectionFragmentFragmentDoc}`;
export const DeleteConnectionDocument = gql`
    mutation DeleteConnection($id: ID!) {
  deleteConnection(id: $id) {
    deleted
    error
    id
  }
}
    `;
export const GetDestinationDocument = gql`
    query GetDestination($id: ID!) {
  destination(id: $id) {
    ...DestinationFragment
  }
}
    ${DestinationFragmentFragmentDoc}`;
export const ListDestinationsDocument = gql`
    query ListDestinations($orderBy: DestinationOrder, $where: DestinationWhereInput) {
  destinations(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...DestinationFragment
      }
    }
    totalCount
  }
}
    ${DestinationFragmentFragmentDoc}`;
export const CreateDestinationDocument = gql`
    mutation CreateDestination($input: CreateDestinationInput!) {
  createDestination(input: $input) {
    created
    error
    destination {
      ...DestinationFragment
    }
  }
}
    ${DestinationFragmentFragmentDoc}`;
export const DeleteDestinationDocument = gql`
    mutation DeleteDestination($id: ID!) {
  deleteDestination(id: $id) {
    deleted
    error
    id
  }
}
    `;
export const GetEventSourceDocument = gql`
    query GetEventSource($id: ID!) {
  eventSource(id: $id) {
    ...EventSourceFragment
  }
}
    ${EventSourceFragmentFragmentDoc}`;
export const ListEventSourcesDocument = gql`
    query ListEventSources($orderBy: EventSourceOrder, $where: EventSourceWhereInput) {
  eventSources(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...EventSourceFragment
      }
    }
    totalCount
  }
}
    ${EventSourceFragmentFragmentDoc}`;
export const CreateEventSourceDocument = gql`
    mutation CreateEventSource($input: CreateEventSourceInput!) {
  createEventSource(input: $input) {
    created
    error
    eventSource {
      ...EventSourceFragment
    }
  }
}
    ${EventSourceFragmentFragmentDoc}`;
export const UpdateEventSourceDocument = gql`
    mutation UpdateEventSource($id: ID!, $input: UpdateEventSourceInput!) {
  updateEventSource(id: $id, input: $input) {
    updated
    error
    eventSource {
      ...EventSourceFragment
    }
  }
}
    ${EventSourceFragmentFragmentDoc}`;
export const DeleteEventSourceDocument = gql`
    mutation DeleteEventSource($id: ID!) {
  deleteEventSource(id: $id) {
    deleted
    error
    id
  }
}
    `;
export const ListFlowRevisionsDocument = gql`
    query ListFlowRevisions($orderBy: FlowRevisionOrder, $where: FlowRevisionWhereInput) {
  flowRevisions(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...FlowRevisionFragment
      }
    }
    totalCount
  }
}
    ${FlowRevisionFragmentFragmentDoc}`;
export const CreateFlowRevisionDocument = gql`
    mutation CreateFlowRevision($input: CreateFlowRevisionInput!) {
  createFlowRevision(input: $input) {
    created
    error
    flowRevision {
      ...FlowRevisionFragment
    }
  }
}
    ${FlowRevisionFragmentFragmentDoc}`;
export const DeleteFlowRevisionDocument = gql`
    mutation DeleteFlowRevision($id: ID!) {
  deleteFlowRevision(id: $id) {
    deleted
    error
    id
  }
}
    `;
export const GetFlowRunDocument = gql`
    query GetFlowRun($id: ID!) {
  flowRun(id: $id) {
    ...FlowRunFragment
  }
}
    ${FlowRunFragmentFragmentDoc}`;
export const ListFlowRunsDocument = gql`
    query ListFlowRuns($orderBy: FlowRunOrder, $where: FlowRunWhereInput) {
  flowRuns(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...FlowRunFragment
      }
    }
    totalCount
  }
}
    ${FlowRunFragmentFragmentDoc}`;
export const CreateFlowRunDocument = gql`
    mutation CreateFlowRun($input: CreateFlowRunInput!) {
  createFlowRun(input: $input) {
    created
    error
    flowRun {
      ...FlowRunFragment
    }
  }
}
    ${FlowRunFragmentFragmentDoc}`;
export const StartFlowRunDocument = gql`
    mutation StartFlowRun($id: ID!, $timeout: String) {
  startFlowRun(id: $id, timeout: $timeout) {
    flowRun {
      ...FlowRunFragment
    }
  }
}
    ${FlowRunFragmentFragmentDoc}`;
export const StopFlowRunDocument = gql`
    mutation StopFlowRun($id: ID!, $timeout: String!) {
  stopFlowRun(id: $id, timeout: $timeout) {
    flowRun {
      ...FlowRunFragment
    }
  }
}
    ${FlowRunFragmentFragmentDoc}`;
export const GetFlowDocument = gql`
    query GetFlow($id: ID, $slug: String) {
  flow(id: $id, slug: $slug) {
    ...FlowFragment
  }
}
    ${FlowFragmentFragmentDoc}`;
export const ListFlowsDocument = gql`
    query ListFlows($orderBy: FlowOrder, $where: FlowWhereInput) {
  flows(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...FlowFragment
      }
    }
    totalCount
  }
}
    ${FlowFragmentFragmentDoc}`;
export const CreateFlowDocument = gql`
    mutation CreateFlow($input: CreateFlowInput!) {
  createFlow(input: $input) {
    created
    error
    flow {
      ...FlowFragment
    }
  }
}
    ${FlowFragmentFragmentDoc}`;
export const GetIntegrationDocument = gql`
    query GetIntegration($id: ID, $slug: String) {
  integration(id: $id, slug: $slug) {
    ...IntegrationFragment
  }
}
    ${IntegrationFragmentFragmentDoc}`;
export const ListIntegrationsDocument = gql`
    query ListIntegrations($orderBy: IntegrationOrder, $where: IntegrationWhereInput) {
  integrations(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...IntegrationFragment
      }
    }
    totalCount
  }
}
    ${IntegrationFragmentFragmentDoc}`;
export const CreateIntegrationDocument = gql`
    mutation CreateIntegration($input: CreateIntegrationInput!) {
  createIntegration(input: $input) {
    created
    error
    integration {
      ...IntegrationFragment
    }
  }
}
    ${IntegrationFragmentFragmentDoc}`;
export const GetIoSchemaDocument = gql`
    query GetIoSchema($nodeType: String!, $nodeId: ID!) {
  ioSchema(type: $nodeType, id: $nodeId) {
    ...IoSchemaFragment
  }
}
    ${IoSchemaFragmentFragmentDoc}`;
export const ListIoSchemasDocument = gql`
    query ListIoSchemas($orderBy: IOSchemaOrder, $where: IOSchemaWhereInput) {
  ioSchemas(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...IoSchemaFragment
      }
    }
  }
}
    ${IoSchemaFragmentFragmentDoc}`;
export const GetModelTypeDocument = gql`
    query GetModelType($modelTypeId: ID, $slug: String) {
  modelType(id: $modelTypeId, slug: $slug) {
    ...ModelTypeFragment
  }
}
    ${ModelTypeFragmentFragmentDoc}`;
export const ListModelTypesDocument = gql`
    query ListModelTypes($orderBy: ModelTypeOrder, $where: ModelTypeWhereInput) {
  modelTypes(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...ModelTypeFragment
      }
    }
    totalCount
  }
}
    ${ModelTypeFragmentFragmentDoc}`;
export const GetModelDocument = gql`
    query GetModel($id: ID!) {
  model(id: $id) {
    ...ModelFragment
  }
}
    ${ModelFragmentFragmentDoc}`;
export const ListModelsDocument = gql`
    query ListModels($orderBy: ModelOrder, $where: ModelWhereInput) {
  models(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...ModelFragment
      }
    }
    totalCount
  }
}
    ${ModelFragmentFragmentDoc}`;
export const CreateModelDocument = gql`
    mutation CreateModel($input: CreateModelInput!) {
  createModel(input: $input) {
    created
    error
    model {
      ...ModelFragment
    }
  }
}
    ${ModelFragmentFragmentDoc}`;
export const SyncModelDocument = gql`
    mutation SyncModel($id: ID!) {
  syncModel(id: $id) {
    model {
      ...ModelFragment
    }
  }
}
    ${ModelFragmentFragmentDoc}`;
export const UpdateModelDocument = gql`
    mutation UpdateModel($id: ID!, $input: UpdateModelInput!) {
  updateModel(id: $id, input: $input) {
    updated
    error
    model {
      ...ModelFragment
    }
  }
}
    ${ModelFragmentFragmentDoc}`;
export const DeleteModelDocument = gql`
    mutation DeleteModel($id: ID!) {
  deleteModel(id: $id) {
    deleted
    error
    id
  }
}
    `;
export const GetOrganizationDocument = gql`
    query GetOrganization($id: ID, $slug: String) {
  organization(id: $id, slug: $slug) {
    ...OrganizationFragment
  }
}
    ${OrganizationFragmentFragmentDoc}`;
export const ListOrganizationsDocument = gql`
    query ListOrganizations($orderBy: OrganizationOrder, $where: OrganizationWhereInput) {
  organizations(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...OrganizationFragment
      }
    }
    totalCount
  }
}
    ${OrganizationFragmentFragmentDoc}`;
export const CreateOrganizationDocument = gql`
    mutation CreateOrganization($input: CreateOrganizationInput!) {
  createOrganization(input: $input) {
    created
    error
    organization {
      ...OrganizationFragment
    }
  }
}
    ${OrganizationFragmentFragmentDoc}`;
export const UpdateOrganizationDocument = gql`
    mutation UpdateOrganization($id: ID!, $input: UpdateOrganizationInput!) {
  updateOrganization(id: $id, input: $input) {
    updated
    error
    organization {
      ...OrganizationFragment
    }
  }
}
    ${OrganizationFragmentFragmentDoc}`;
export const DeleteOrganizationDocument = gql`
    mutation DeleteOrganization($id: ID!) {
  deleteOrganization(id: $id) {
    deleted
    error
    id
  }
}
    `;
export const ListPackagesDocument = gql`
    query ListPackages($orderBy: IntegrationPackageOrder, $where: IntegrationPackageWhereInput) {
  packages(orderBy: $orderBy, where: $where) {
    totalCount
    edges {
      node {
        ...IntegrationPackageFragment
      }
    }
  }
}
    ${IntegrationPackageFragmentFragmentDoc}`;
export const SyncPackageDocument = gql`
    mutation SyncPackage($input: SyncPackageInput!) {
  syncPackage(input: $input) {
    integrationPackage {
      ...IntegrationPackageFragment
      integration {
        ...IntegrationFragment
      }
    }
  }
}
    ${IntegrationPackageFragmentFragmentDoc}
${IntegrationFragmentFragmentDoc}`;
export const GetPersonalAccessTokenDocument = gql`
    query GetPersonalAccessToken($id: ID, $token: String) {
  personalAccessToken(id: $id, token: $token) {
    ...PersonalAccessTokenFragment
  }
}
    ${PersonalAccessTokenFragmentFragmentDoc}`;
export const ListPersonalAccessTokensDocument = gql`
    query ListPersonalAccessTokens($orderBy: PersonalAccessTokenOrder, $where: PersonalAccessTokenWhereInput) {
  personalAccessTokens(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...PersonalAccessTokenFragment
      }
    }
    totalCount
  }
}
    ${PersonalAccessTokenFragmentFragmentDoc}`;
export const CreatePersonalAccessTokenDocument = gql`
    mutation CreatePersonalAccessToken($input: CreatePersonalAccessTokenInput!) {
  createPersonalAccessToken(input: $input) {
    created
    error
    personalAccessToken {
      ...PersonalAccessTokenFragment
    }
  }
}
    ${PersonalAccessTokenFragmentFragmentDoc}`;
export const UpdatePersonalAccessTokenDocument = gql`
    mutation UpdatePersonalAccessToken($id: ID!, $input: UpdatePersonalAccessTokenInput!) {
  updatePersonalAccessToken(id: $id, input: $input) {
    updated
    error
    personalAccessToken {
      ...PersonalAccessTokenFragment
    }
  }
}
    ${PersonalAccessTokenFragmentFragmentDoc}`;
export const RotatePersonalAccessTokenDocument = gql`
    mutation RotatePersonalAccessToken($id: ID, $token: String) {
  rotatePersonalAccessToken(id: $id, token: $token) {
    updated
    error
    personalAccessToken {
      ...PersonalAccessTokenFragment
    }
  }
}
    ${PersonalAccessTokenFragmentFragmentDoc}`;
export const DeletePersonalAccessTokenDocument = gql`
    mutation DeletePersonalAccessToken($token: String!) {
  deletePersonalAccessToken(token: $token) {
    deleted
    error
    id
  }
}
    `;
export const GetSchemaDocument = gql`
    query GetSchema($id: ID!) {
  schema(id: $id) {
    ...SchemaRefFragment
  }
}
    ${SchemaRefFragmentFragmentDoc}`;
export const ListSchemasDocument = gql`
    query ListSchemas($orderBy: [SchemaRefOrder!], $where: SchemaRefWhereInput) {
  schemas(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...SchemaRefFragment
      }
    }
    totalCount
  }
}
    ${SchemaRefFragmentFragmentDoc}`;
export const SchemasDeletedDocument = gql`
    query SchemasDeleted {
  schemasDeleted {
    ...SchemaRefFragment
  }
}
    ${SchemaRefFragmentFragmentDoc}`;
export const CreateSchemaDocument = gql`
    mutation CreateSchema($input: CreateSchemaRefInput!) {
  createSchema(input: $input) {
    created
    error
    schemaRef {
      ...SchemaRefFragment
    }
  }
}
    ${SchemaRefFragmentFragmentDoc}`;
export const SyncSchemaDocument = gql`
    mutation SyncSchema($id: ID!, $autoSync: Boolean) {
  syncSchema(id: $id, autoSync: $autoSync) {
    updated
    error
    schemaRef {
      ...SchemaRefFragment
    }
  }
}
    ${SchemaRefFragmentFragmentDoc}`;
export const UpdateSchemaDocument = gql`
    mutation UpdateSchema($id: ID!, $input: UpdateSchemaRefInput!) {
  updateSchema(id: $id, input: $input) {
    updated
    error
    schemaRef {
      ...SchemaRefFragment
    }
  }
}
    ${SchemaRefFragmentFragmentDoc}`;
export const DeleteSchemaDocument = gql`
    mutation DeleteSchema($id: ID!, $softDelete: Boolean) {
  deleteSchema(id: $id, softDelete: $softDelete) {
    deleted
    error
    id
  }
}
    `;
export const SearchDocument = gql`
    query Search($q: String!, $limit: Int!, $offset: Int!, $stype: SearchType!, $by: SearchByInput, $where: SearchWhereInput, $view: SearchView) {
  search(
    q: $q
    limit: $limit
    offset: $offset
    by: $by
    type: $stype
    where: $where
    view: $view
  ) {
    totalCount
    results {
      rank
      text
      node {
        type: __typename
        id
      }
    }
  }
}
    `;
export const GetSourceTypeDocument = gql`
    query GetSourceType($id: ID, $slug: String) {
  sourceType(id: $id, slug: $slug) {
    ...SourceTypeFragment
  }
}
    ${SourceTypeFragmentFragmentDoc}`;
export const ListSourceTypesDocument = gql`
    query ListSourceTypes($orderBy: SourceTypeOrder, $where: SourceTypeWhereInput) {
  sourceTypes(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...SourceTypeFragment
      }
    }
    totalCount
  }
}
    ${SourceTypeFragmentFragmentDoc}`;
export const GetSourceDocument = gql`
    query GetSource($id: ID!) {
  source(id: $id) {
    ...SourceFragment
  }
}
    ${SourceFragmentFragmentDoc}`;
export const ListSourcesDocument = gql`
    query ListSources($orderBy: SourceOrder, $where: SourceWhereInput) {
  sources(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...SourceFragment
      }
    }
    totalCount
  }
}
    ${SourceFragmentFragmentDoc}`;
export const CreateSourceDocument = gql`
    mutation CreateSource($input: CreateSourceInput!) {
  createSource(input: $input) {
    created
    error
    source {
      ...SourceFragment
    }
  }
}
    ${SourceFragmentFragmentDoc}`;
export const SyncSourceDocument = gql`
    mutation SyncSource($id: ID!) {
  syncSource(id: $id) {
    updated
    error
    source {
      ...SourceFragment
    }
  }
}
    ${SourceFragmentFragmentDoc}`;
export const UpdateSourceDocument = gql`
    mutation UpdateSource($id: ID!, $input: UpdateSourceInput!) {
  updateSource(id: $id, input: $input) {
    updated
    error
    source {
      ...SourceFragment
    }
  }
}
    ${SourceFragmentFragmentDoc}`;
export const DeleteSourceDocument = gql`
    mutation DeleteSource($id: ID!, $softDelete: Boolean) {
  deleteSource(id: $id, softDelete: $softDelete) {
    deleted
    error
    id
  }
}
    `;
export const GetSpaceDocument = gql`
    query GetSpace($spaceId: ID, $slug: String) {
  space(id: $spaceId, slug: $slug) {
    ...SpaceFragment
  }
}
    ${SpaceFragmentFragmentDoc}`;
export const ListSpacesDocument = gql`
    query ListSpaces($orderBy: SpaceOrder, $where: SpaceWhereInput) {
  spaces(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...SpaceFragment
      }
    }
    totalCount
  }
}
    ${SpaceFragmentFragmentDoc}`;
export const CreateSpaceDocument = gql`
    mutation CreateSpace($input: CreateSpaceInput!) {
  createSpace(input: $input) {
    created
    error
    space {
      ...SpaceFragment
    }
  }
}
    ${SpaceFragmentFragmentDoc}`;
export const UpdateSpaceDocument = gql`
    mutation UpdateSpace($id: ID!, $input: UpdateSpaceInput!) {
  updateSpace(id: $id, input: $input) {
    updated
    error
    space {
      ...SpaceFragment
    }
  }
}
    ${SpaceFragmentFragmentDoc}`;
export const DeleteSpaceDocument = gql`
    mutation DeleteSpace($id: ID!) {
  deleteSpace(id: $id) {
    deleted
    error
    id
  }
}
    `;
export const GetTableDocument = gql`
    query GetTable($id: ID!) {
  table(id: $id) {
    ...TableRefFragment
  }
}
    ${TableRefFragmentFragmentDoc}`;
export const ListTablesDocument = gql`
    query ListTables($orderBy: [TableRefOrder!], $where: TableRefWhereInput) {
  tables(orderBy: $orderBy, where: $where) {
    edges {
      node {
        ...TableRefFragment
      }
    }
    totalCount
  }
}
    ${TableRefFragmentFragmentDoc}`;
export const TablesDeletedDocument = gql`
    query TablesDeleted {
  tablesDeleted {
    ...TableRefFragment
  }
}
    ${TableRefFragmentFragmentDoc}`;
export const CreateTableDocument = gql`
    mutation CreateTable($input: CreateTableInput!) {
  createTable(input: $input) {
    created
    error
    tableRef {
      ...TableRefFragment
    }
  }
}
    ${TableRefFragmentFragmentDoc}`;
export const SyncTableDocument = gql`
    mutation SyncTable($id: ID!) {
  syncTable(id: $id) {
    updated
    error
    tableRef {
      ...TableRefFragment
    }
  }
}
    ${TableRefFragmentFragmentDoc}`;
export const DeleteTableDocument = gql`
    mutation DeleteTable($id: ID!, $softDelete: Boolean) {
  deleteTable(id: $id, softDelete: $softDelete) {
    deleted
    error
    id
  }
}
    `;
export type Requester<C = {}> = <R, V>(doc: DocumentNode, vars?: V, options?: C) => Promise<R> | AsyncIterable<R>
export function getSdk<C>(requester: Requester<C>) {
  return {
    GetCatalog(variables: GetCatalogQueryVariables, options?: C): Promise<GetCatalogQuery> {
      return requester<GetCatalogQuery, GetCatalogQueryVariables>(GetCatalogDocument, variables, options) as Promise<GetCatalogQuery>;
    },
    ListCatalogs(variables?: ListCatalogsQueryVariables, options?: C): Promise<ListCatalogsQuery> {
      return requester<ListCatalogsQuery, ListCatalogsQueryVariables>(ListCatalogsDocument, variables, options) as Promise<ListCatalogsQuery>;
    },
    CreateCatalog(variables: CreateCatalogMutationVariables, options?: C): Promise<CreateCatalogMutation> {
      return requester<CreateCatalogMutation, CreateCatalogMutationVariables>(CreateCatalogDocument, variables, options) as Promise<CreateCatalogMutation>;
    },
    SyncCatalog(variables: SyncCatalogMutationVariables, options?: C): Promise<SyncCatalogMutation> {
      return requester<SyncCatalogMutation, SyncCatalogMutationVariables>(SyncCatalogDocument, variables, options) as Promise<SyncCatalogMutation>;
    },
    UpdateCatalog(variables: UpdateCatalogMutationVariables, options?: C): Promise<UpdateCatalogMutation> {
      return requester<UpdateCatalogMutation, UpdateCatalogMutationVariables>(UpdateCatalogDocument, variables, options) as Promise<UpdateCatalogMutation>;
    },
    Mutation(variables: MutationMutationVariables, options?: C): Promise<MutationMutation> {
      return requester<MutationMutation, MutationMutationVariables>(MutationDocument, variables, options) as Promise<MutationMutation>;
    },
    GetColumn(variables: GetColumnQueryVariables, options?: C): Promise<GetColumnQuery> {
      return requester<GetColumnQuery, GetColumnQueryVariables>(GetColumnDocument, variables, options) as Promise<GetColumnQuery>;
    },
    ListColumns(variables?: ListColumnsQueryVariables, options?: C): Promise<ListColumnsQuery> {
      return requester<ListColumnsQuery, ListColumnsQueryVariables>(ListColumnsDocument, variables, options) as Promise<ListColumnsQuery>;
    },
    CreateColumn(variables: CreateColumnMutationVariables, options?: C): Promise<CreateColumnMutation> {
      return requester<CreateColumnMutation, CreateColumnMutationVariables>(CreateColumnDocument, variables, options) as Promise<CreateColumnMutation>;
    },
    DeleteColumn(variables: DeleteColumnMutationVariables, options?: C): Promise<DeleteColumnMutation> {
      return requester<DeleteColumnMutation, DeleteColumnMutationVariables>(DeleteColumnDocument, variables, options) as Promise<DeleteColumnMutation>;
    },
    GetConnection(variables?: GetConnectionQueryVariables, options?: C): Promise<GetConnectionQuery> {
      return requester<GetConnectionQuery, GetConnectionQueryVariables>(GetConnectionDocument, variables, options) as Promise<GetConnectionQuery>;
    },
    ListConnections(variables?: ListConnectionsQueryVariables, options?: C): Promise<ListConnectionsQuery> {
      return requester<ListConnectionsQuery, ListConnectionsQueryVariables>(ListConnectionsDocument, variables, options) as Promise<ListConnectionsQuery>;
    },
    CheckConnection(variables?: CheckConnectionQueryVariables, options?: C): Promise<CheckConnectionQuery> {
      return requester<CheckConnectionQuery, CheckConnectionQueryVariables>(CheckConnectionDocument, variables, options) as Promise<CheckConnectionQuery>;
    },
    CreateConnection(variables: CreateConnectionMutationVariables, options?: C): Promise<CreateConnectionMutation> {
      return requester<CreateConnectionMutation, CreateConnectionMutationVariables>(CreateConnectionDocument, variables, options) as Promise<CreateConnectionMutation>;
    },
    UpdateConnection(variables: UpdateConnectionMutationVariables, options?: C): Promise<UpdateConnectionMutation> {
      return requester<UpdateConnectionMutation, UpdateConnectionMutationVariables>(UpdateConnectionDocument, variables, options) as Promise<UpdateConnectionMutation>;
    },
    DeleteConnection(variables: DeleteConnectionMutationVariables, options?: C): Promise<DeleteConnectionMutation> {
      return requester<DeleteConnectionMutation, DeleteConnectionMutationVariables>(DeleteConnectionDocument, variables, options) as Promise<DeleteConnectionMutation>;
    },
    GetDestination(variables: GetDestinationQueryVariables, options?: C): Promise<GetDestinationQuery> {
      return requester<GetDestinationQuery, GetDestinationQueryVariables>(GetDestinationDocument, variables, options) as Promise<GetDestinationQuery>;
    },
    ListDestinations(variables?: ListDestinationsQueryVariables, options?: C): Promise<ListDestinationsQuery> {
      return requester<ListDestinationsQuery, ListDestinationsQueryVariables>(ListDestinationsDocument, variables, options) as Promise<ListDestinationsQuery>;
    },
    CreateDestination(variables: CreateDestinationMutationVariables, options?: C): Promise<CreateDestinationMutation> {
      return requester<CreateDestinationMutation, CreateDestinationMutationVariables>(CreateDestinationDocument, variables, options) as Promise<CreateDestinationMutation>;
    },
    DeleteDestination(variables: DeleteDestinationMutationVariables, options?: C): Promise<DeleteDestinationMutation> {
      return requester<DeleteDestinationMutation, DeleteDestinationMutationVariables>(DeleteDestinationDocument, variables, options) as Promise<DeleteDestinationMutation>;
    },
    GetEventSource(variables: GetEventSourceQueryVariables, options?: C): Promise<GetEventSourceQuery> {
      return requester<GetEventSourceQuery, GetEventSourceQueryVariables>(GetEventSourceDocument, variables, options) as Promise<GetEventSourceQuery>;
    },
    ListEventSources(variables?: ListEventSourcesQueryVariables, options?: C): Promise<ListEventSourcesQuery> {
      return requester<ListEventSourcesQuery, ListEventSourcesQueryVariables>(ListEventSourcesDocument, variables, options) as Promise<ListEventSourcesQuery>;
    },
    CreateEventSource(variables: CreateEventSourceMutationVariables, options?: C): Promise<CreateEventSourceMutation> {
      return requester<CreateEventSourceMutation, CreateEventSourceMutationVariables>(CreateEventSourceDocument, variables, options) as Promise<CreateEventSourceMutation>;
    },
    UpdateEventSource(variables: UpdateEventSourceMutationVariables, options?: C): Promise<UpdateEventSourceMutation> {
      return requester<UpdateEventSourceMutation, UpdateEventSourceMutationVariables>(UpdateEventSourceDocument, variables, options) as Promise<UpdateEventSourceMutation>;
    },
    DeleteEventSource(variables: DeleteEventSourceMutationVariables, options?: C): Promise<DeleteEventSourceMutation> {
      return requester<DeleteEventSourceMutation, DeleteEventSourceMutationVariables>(DeleteEventSourceDocument, variables, options) as Promise<DeleteEventSourceMutation>;
    },
    ListFlowRevisions(variables?: ListFlowRevisionsQueryVariables, options?: C): Promise<ListFlowRevisionsQuery> {
      return requester<ListFlowRevisionsQuery, ListFlowRevisionsQueryVariables>(ListFlowRevisionsDocument, variables, options) as Promise<ListFlowRevisionsQuery>;
    },
    CreateFlowRevision(variables: CreateFlowRevisionMutationVariables, options?: C): Promise<CreateFlowRevisionMutation> {
      return requester<CreateFlowRevisionMutation, CreateFlowRevisionMutationVariables>(CreateFlowRevisionDocument, variables, options) as Promise<CreateFlowRevisionMutation>;
    },
    DeleteFlowRevision(variables: DeleteFlowRevisionMutationVariables, options?: C): Promise<DeleteFlowRevisionMutation> {
      return requester<DeleteFlowRevisionMutation, DeleteFlowRevisionMutationVariables>(DeleteFlowRevisionDocument, variables, options) as Promise<DeleteFlowRevisionMutation>;
    },
    GetFlowRun(variables: GetFlowRunQueryVariables, options?: C): Promise<GetFlowRunQuery> {
      return requester<GetFlowRunQuery, GetFlowRunQueryVariables>(GetFlowRunDocument, variables, options) as Promise<GetFlowRunQuery>;
    },
    ListFlowRuns(variables?: ListFlowRunsQueryVariables, options?: C): Promise<ListFlowRunsQuery> {
      return requester<ListFlowRunsQuery, ListFlowRunsQueryVariables>(ListFlowRunsDocument, variables, options) as Promise<ListFlowRunsQuery>;
    },
    CreateFlowRun(variables: CreateFlowRunMutationVariables, options?: C): Promise<CreateFlowRunMutation> {
      return requester<CreateFlowRunMutation, CreateFlowRunMutationVariables>(CreateFlowRunDocument, variables, options) as Promise<CreateFlowRunMutation>;
    },
    StartFlowRun(variables: StartFlowRunMutationVariables, options?: C): Promise<StartFlowRunMutation> {
      return requester<StartFlowRunMutation, StartFlowRunMutationVariables>(StartFlowRunDocument, variables, options) as Promise<StartFlowRunMutation>;
    },
    StopFlowRun(variables: StopFlowRunMutationVariables, options?: C): Promise<StopFlowRunMutation> {
      return requester<StopFlowRunMutation, StopFlowRunMutationVariables>(StopFlowRunDocument, variables, options) as Promise<StopFlowRunMutation>;
    },
    GetFlow(variables?: GetFlowQueryVariables, options?: C): Promise<GetFlowQuery> {
      return requester<GetFlowQuery, GetFlowQueryVariables>(GetFlowDocument, variables, options) as Promise<GetFlowQuery>;
    },
    ListFlows(variables?: ListFlowsQueryVariables, options?: C): Promise<ListFlowsQuery> {
      return requester<ListFlowsQuery, ListFlowsQueryVariables>(ListFlowsDocument, variables, options) as Promise<ListFlowsQuery>;
    },
    CreateFlow(variables: CreateFlowMutationVariables, options?: C): Promise<CreateFlowMutation> {
      return requester<CreateFlowMutation, CreateFlowMutationVariables>(CreateFlowDocument, variables, options) as Promise<CreateFlowMutation>;
    },
    GetIntegration(variables?: GetIntegrationQueryVariables, options?: C): Promise<GetIntegrationQuery> {
      return requester<GetIntegrationQuery, GetIntegrationQueryVariables>(GetIntegrationDocument, variables, options) as Promise<GetIntegrationQuery>;
    },
    ListIntegrations(variables?: ListIntegrationsQueryVariables, options?: C): Promise<ListIntegrationsQuery> {
      return requester<ListIntegrationsQuery, ListIntegrationsQueryVariables>(ListIntegrationsDocument, variables, options) as Promise<ListIntegrationsQuery>;
    },
    CreateIntegration(variables: CreateIntegrationMutationVariables, options?: C): Promise<CreateIntegrationMutation> {
      return requester<CreateIntegrationMutation, CreateIntegrationMutationVariables>(CreateIntegrationDocument, variables, options) as Promise<CreateIntegrationMutation>;
    },
    GetIoSchema(variables: GetIoSchemaQueryVariables, options?: C): Promise<GetIoSchemaQuery> {
      return requester<GetIoSchemaQuery, GetIoSchemaQueryVariables>(GetIoSchemaDocument, variables, options) as Promise<GetIoSchemaQuery>;
    },
    ListIoSchemas(variables?: ListIoSchemasQueryVariables, options?: C): Promise<ListIoSchemasQuery> {
      return requester<ListIoSchemasQuery, ListIoSchemasQueryVariables>(ListIoSchemasDocument, variables, options) as Promise<ListIoSchemasQuery>;
    },
    GetModelType(variables?: GetModelTypeQueryVariables, options?: C): Promise<GetModelTypeQuery> {
      return requester<GetModelTypeQuery, GetModelTypeQueryVariables>(GetModelTypeDocument, variables, options) as Promise<GetModelTypeQuery>;
    },
    ListModelTypes(variables?: ListModelTypesQueryVariables, options?: C): Promise<ListModelTypesQuery> {
      return requester<ListModelTypesQuery, ListModelTypesQueryVariables>(ListModelTypesDocument, variables, options) as Promise<ListModelTypesQuery>;
    },
    GetModel(variables: GetModelQueryVariables, options?: C): Promise<GetModelQuery> {
      return requester<GetModelQuery, GetModelQueryVariables>(GetModelDocument, variables, options) as Promise<GetModelQuery>;
    },
    ListModels(variables?: ListModelsQueryVariables, options?: C): Promise<ListModelsQuery> {
      return requester<ListModelsQuery, ListModelsQueryVariables>(ListModelsDocument, variables, options) as Promise<ListModelsQuery>;
    },
    CreateModel(variables: CreateModelMutationVariables, options?: C): Promise<CreateModelMutation> {
      return requester<CreateModelMutation, CreateModelMutationVariables>(CreateModelDocument, variables, options) as Promise<CreateModelMutation>;
    },
    SyncModel(variables: SyncModelMutationVariables, options?: C): Promise<SyncModelMutation> {
      return requester<SyncModelMutation, SyncModelMutationVariables>(SyncModelDocument, variables, options) as Promise<SyncModelMutation>;
    },
    UpdateModel(variables: UpdateModelMutationVariables, options?: C): Promise<UpdateModelMutation> {
      return requester<UpdateModelMutation, UpdateModelMutationVariables>(UpdateModelDocument, variables, options) as Promise<UpdateModelMutation>;
    },
    DeleteModel(variables: DeleteModelMutationVariables, options?: C): Promise<DeleteModelMutation> {
      return requester<DeleteModelMutation, DeleteModelMutationVariables>(DeleteModelDocument, variables, options) as Promise<DeleteModelMutation>;
    },
    GetOrganization(variables?: GetOrganizationQueryVariables, options?: C): Promise<GetOrganizationQuery> {
      return requester<GetOrganizationQuery, GetOrganizationQueryVariables>(GetOrganizationDocument, variables, options) as Promise<GetOrganizationQuery>;
    },
    ListOrganizations(variables?: ListOrganizationsQueryVariables, options?: C): Promise<ListOrganizationsQuery> {
      return requester<ListOrganizationsQuery, ListOrganizationsQueryVariables>(ListOrganizationsDocument, variables, options) as Promise<ListOrganizationsQuery>;
    },
    CreateOrganization(variables: CreateOrganizationMutationVariables, options?: C): Promise<CreateOrganizationMutation> {
      return requester<CreateOrganizationMutation, CreateOrganizationMutationVariables>(CreateOrganizationDocument, variables, options) as Promise<CreateOrganizationMutation>;
    },
    UpdateOrganization(variables: UpdateOrganizationMutationVariables, options?: C): Promise<UpdateOrganizationMutation> {
      return requester<UpdateOrganizationMutation, UpdateOrganizationMutationVariables>(UpdateOrganizationDocument, variables, options) as Promise<UpdateOrganizationMutation>;
    },
    DeleteOrganization(variables: DeleteOrganizationMutationVariables, options?: C): Promise<DeleteOrganizationMutation> {
      return requester<DeleteOrganizationMutation, DeleteOrganizationMutationVariables>(DeleteOrganizationDocument, variables, options) as Promise<DeleteOrganizationMutation>;
    },
    ListPackages(variables?: ListPackagesQueryVariables, options?: C): Promise<ListPackagesQuery> {
      return requester<ListPackagesQuery, ListPackagesQueryVariables>(ListPackagesDocument, variables, options) as Promise<ListPackagesQuery>;
    },
    SyncPackage(variables: SyncPackageMutationVariables, options?: C): Promise<SyncPackageMutation> {
      return requester<SyncPackageMutation, SyncPackageMutationVariables>(SyncPackageDocument, variables, options) as Promise<SyncPackageMutation>;
    },
    GetPersonalAccessToken(variables?: GetPersonalAccessTokenQueryVariables, options?: C): Promise<GetPersonalAccessTokenQuery> {
      return requester<GetPersonalAccessTokenQuery, GetPersonalAccessTokenQueryVariables>(GetPersonalAccessTokenDocument, variables, options) as Promise<GetPersonalAccessTokenQuery>;
    },
    ListPersonalAccessTokens(variables?: ListPersonalAccessTokensQueryVariables, options?: C): Promise<ListPersonalAccessTokensQuery> {
      return requester<ListPersonalAccessTokensQuery, ListPersonalAccessTokensQueryVariables>(ListPersonalAccessTokensDocument, variables, options) as Promise<ListPersonalAccessTokensQuery>;
    },
    CreatePersonalAccessToken(variables: CreatePersonalAccessTokenMutationVariables, options?: C): Promise<CreatePersonalAccessTokenMutation> {
      return requester<CreatePersonalAccessTokenMutation, CreatePersonalAccessTokenMutationVariables>(CreatePersonalAccessTokenDocument, variables, options) as Promise<CreatePersonalAccessTokenMutation>;
    },
    UpdatePersonalAccessToken(variables: UpdatePersonalAccessTokenMutationVariables, options?: C): Promise<UpdatePersonalAccessTokenMutation> {
      return requester<UpdatePersonalAccessTokenMutation, UpdatePersonalAccessTokenMutationVariables>(UpdatePersonalAccessTokenDocument, variables, options) as Promise<UpdatePersonalAccessTokenMutation>;
    },
    RotatePersonalAccessToken(variables?: RotatePersonalAccessTokenMutationVariables, options?: C): Promise<RotatePersonalAccessTokenMutation> {
      return requester<RotatePersonalAccessTokenMutation, RotatePersonalAccessTokenMutationVariables>(RotatePersonalAccessTokenDocument, variables, options) as Promise<RotatePersonalAccessTokenMutation>;
    },
    DeletePersonalAccessToken(variables: DeletePersonalAccessTokenMutationVariables, options?: C): Promise<DeletePersonalAccessTokenMutation> {
      return requester<DeletePersonalAccessTokenMutation, DeletePersonalAccessTokenMutationVariables>(DeletePersonalAccessTokenDocument, variables, options) as Promise<DeletePersonalAccessTokenMutation>;
    },
    GetSchema(variables: GetSchemaQueryVariables, options?: C): Promise<GetSchemaQuery> {
      return requester<GetSchemaQuery, GetSchemaQueryVariables>(GetSchemaDocument, variables, options) as Promise<GetSchemaQuery>;
    },
    ListSchemas(variables?: ListSchemasQueryVariables, options?: C): Promise<ListSchemasQuery> {
      return requester<ListSchemasQuery, ListSchemasQueryVariables>(ListSchemasDocument, variables, options) as Promise<ListSchemasQuery>;
    },
    SchemasDeleted(variables?: SchemasDeletedQueryVariables, options?: C): Promise<SchemasDeletedQuery> {
      return requester<SchemasDeletedQuery, SchemasDeletedQueryVariables>(SchemasDeletedDocument, variables, options) as Promise<SchemasDeletedQuery>;
    },
    CreateSchema(variables: CreateSchemaMutationVariables, options?: C): Promise<CreateSchemaMutation> {
      return requester<CreateSchemaMutation, CreateSchemaMutationVariables>(CreateSchemaDocument, variables, options) as Promise<CreateSchemaMutation>;
    },
    SyncSchema(variables: SyncSchemaMutationVariables, options?: C): Promise<SyncSchemaMutation> {
      return requester<SyncSchemaMutation, SyncSchemaMutationVariables>(SyncSchemaDocument, variables, options) as Promise<SyncSchemaMutation>;
    },
    UpdateSchema(variables: UpdateSchemaMutationVariables, options?: C): Promise<UpdateSchemaMutation> {
      return requester<UpdateSchemaMutation, UpdateSchemaMutationVariables>(UpdateSchemaDocument, variables, options) as Promise<UpdateSchemaMutation>;
    },
    DeleteSchema(variables: DeleteSchemaMutationVariables, options?: C): Promise<DeleteSchemaMutation> {
      return requester<DeleteSchemaMutation, DeleteSchemaMutationVariables>(DeleteSchemaDocument, variables, options) as Promise<DeleteSchemaMutation>;
    },
    Search(variables: SearchQueryVariables, options?: C): Promise<SearchQuery> {
      return requester<SearchQuery, SearchQueryVariables>(SearchDocument, variables, options) as Promise<SearchQuery>;
    },
    GetSourceType(variables?: GetSourceTypeQueryVariables, options?: C): Promise<GetSourceTypeQuery> {
      return requester<GetSourceTypeQuery, GetSourceTypeQueryVariables>(GetSourceTypeDocument, variables, options) as Promise<GetSourceTypeQuery>;
    },
    ListSourceTypes(variables?: ListSourceTypesQueryVariables, options?: C): Promise<ListSourceTypesQuery> {
      return requester<ListSourceTypesQuery, ListSourceTypesQueryVariables>(ListSourceTypesDocument, variables, options) as Promise<ListSourceTypesQuery>;
    },
    GetSource(variables: GetSourceQueryVariables, options?: C): Promise<GetSourceQuery> {
      return requester<GetSourceQuery, GetSourceQueryVariables>(GetSourceDocument, variables, options) as Promise<GetSourceQuery>;
    },
    ListSources(variables?: ListSourcesQueryVariables, options?: C): Promise<ListSourcesQuery> {
      return requester<ListSourcesQuery, ListSourcesQueryVariables>(ListSourcesDocument, variables, options) as Promise<ListSourcesQuery>;
    },
    CreateSource(variables: CreateSourceMutationVariables, options?: C): Promise<CreateSourceMutation> {
      return requester<CreateSourceMutation, CreateSourceMutationVariables>(CreateSourceDocument, variables, options) as Promise<CreateSourceMutation>;
    },
    SyncSource(variables: SyncSourceMutationVariables, options?: C): Promise<SyncSourceMutation> {
      return requester<SyncSourceMutation, SyncSourceMutationVariables>(SyncSourceDocument, variables, options) as Promise<SyncSourceMutation>;
    },
    UpdateSource(variables: UpdateSourceMutationVariables, options?: C): Promise<UpdateSourceMutation> {
      return requester<UpdateSourceMutation, UpdateSourceMutationVariables>(UpdateSourceDocument, variables, options) as Promise<UpdateSourceMutation>;
    },
    DeleteSource(variables: DeleteSourceMutationVariables, options?: C): Promise<DeleteSourceMutation> {
      return requester<DeleteSourceMutation, DeleteSourceMutationVariables>(DeleteSourceDocument, variables, options) as Promise<DeleteSourceMutation>;
    },
    GetSpace(variables?: GetSpaceQueryVariables, options?: C): Promise<GetSpaceQuery> {
      return requester<GetSpaceQuery, GetSpaceQueryVariables>(GetSpaceDocument, variables, options) as Promise<GetSpaceQuery>;
    },
    ListSpaces(variables?: ListSpacesQueryVariables, options?: C): Promise<ListSpacesQuery> {
      return requester<ListSpacesQuery, ListSpacesQueryVariables>(ListSpacesDocument, variables, options) as Promise<ListSpacesQuery>;
    },
    CreateSpace(variables: CreateSpaceMutationVariables, options?: C): Promise<CreateSpaceMutation> {
      return requester<CreateSpaceMutation, CreateSpaceMutationVariables>(CreateSpaceDocument, variables, options) as Promise<CreateSpaceMutation>;
    },
    UpdateSpace(variables: UpdateSpaceMutationVariables, options?: C): Promise<UpdateSpaceMutation> {
      return requester<UpdateSpaceMutation, UpdateSpaceMutationVariables>(UpdateSpaceDocument, variables, options) as Promise<UpdateSpaceMutation>;
    },
    DeleteSpace(variables: DeleteSpaceMutationVariables, options?: C): Promise<DeleteSpaceMutation> {
      return requester<DeleteSpaceMutation, DeleteSpaceMutationVariables>(DeleteSpaceDocument, variables, options) as Promise<DeleteSpaceMutation>;
    },
    GetTable(variables: GetTableQueryVariables, options?: C): Promise<GetTableQuery> {
      return requester<GetTableQuery, GetTableQueryVariables>(GetTableDocument, variables, options) as Promise<GetTableQuery>;
    },
    ListTables(variables?: ListTablesQueryVariables, options?: C): Promise<ListTablesQuery> {
      return requester<ListTablesQuery, ListTablesQueryVariables>(ListTablesDocument, variables, options) as Promise<ListTablesQuery>;
    },
    TablesDeleted(variables?: TablesDeletedQueryVariables, options?: C): Promise<TablesDeletedQuery> {
      return requester<TablesDeletedQuery, TablesDeletedQueryVariables>(TablesDeletedDocument, variables, options) as Promise<TablesDeletedQuery>;
    },
    CreateTable(variables: CreateTableMutationVariables, options?: C): Promise<CreateTableMutation> {
      return requester<CreateTableMutation, CreateTableMutationVariables>(CreateTableDocument, variables, options) as Promise<CreateTableMutation>;
    },
    SyncTable(variables: SyncTableMutationVariables, options?: C): Promise<SyncTableMutation> {
      return requester<SyncTableMutation, SyncTableMutationVariables>(SyncTableDocument, variables, options) as Promise<SyncTableMutation>;
    },
    DeleteTable(variables: DeleteTableMutationVariables, options?: C): Promise<DeleteTableMutation> {
      return requester<DeleteTableMutation, DeleteTableMutationVariables>(DeleteTableDocument, variables, options) as Promise<DeleteTableMutation>;
    }
  };
}
export type Sdk = ReturnType<typeof getSdk>;