# Token-Based Pagination With Fixed Sort Fields

## Scope

This document defines keyset pagination for an API that permits clients to sort
by a fixed, server-known set of fields. It applies to forward pagination over a
relational query. Offset pagination is not part of this definition.

The central invariant is:

> The effective `ORDER BY`, the values encoded in the page token, and the
> continuation predicate MUST describe exactly the same total ordering.

A violation of this invariant can duplicate or omit records between pages.

## Terminology

- **Allowed sort field**: A server-defined field that clients may select for
  sorting.
- **Sort component**: A field, direction, and null-placement rule.
- **Effective order**: The requested sort components followed by a unique
  server-defined tie-breaker.
- **Cursor**: The effective-order values from the final returned record.
- **Page token**: An opaque encoding of the cursor and its query context.
- **Continuation predicate**: The lexicographic predicate selecting records
  strictly after the cursor in the effective order.

## Allowed Sort Fields

The server MUST maintain an allowlist. Each field definition MUST contain:

1. Its public API name.
2. Its database expression or column.
3. Its cursor value type and codec.
4. Its allowed directions.
5. Whether it is nullable.
6. Its fixed null-placement policy for each direction.
7. A function that extracts its cursor value from a returned record.

For example:

| API field | SQL column | Type | Directions | Nullable |
| --- | --- | --- | --- | --- |
| `name` | `name` | string | ascending, descending | no |
| `update_time` | `updated_at` | timestamp | ascending, descending | no |
| `priority` | `priority` | integer | ascending, descending | yes |

Unknown fields, repeated fields, unsupported directions, and invalid order
syntax MUST be rejected before executing a query.

SQL identifiers MUST come from the server allowlist. A client-provided field
name MUST NOT be interpolated directly into SQL.

## Effective Total Order

A page cursor is correct only when the query has a total order: every pair of
records must have a deterministic relative position.

Let the validated client order be:

```text
K1 D1, K2 D2, ..., Kn Dn
```

where each `Ki` is a known sort field and each `Di` is ascending or descending.
The server MUST append a unique, stable tie-breaker `U` unless the requested
components already form a unique key:

```text
K1 D1, K2 D2, ..., Kn Dn, U Du
```

The tie-breaker SHOULD be immutable. Its direction SHOULD normally match the
last requested direction, but any fixed direction is valid if cursor generation
and predicates use that same direction.

Example:

```sql
ORDER BY name ASC, updated_at DESC, rowid DESC
```

Here, `rowid` is the unique tie-breaker. Without it, two records with equal
`name` and `updated_at` would not have deterministic page membership.

When the client supplies no order, the server MUST use a documented default
effective order, such as:

```sql
ORDER BY updated_at DESC, rowid DESC
```

## Cursor Definition

For an effective order of `m` components, the cursor is the ordered tuple:

```text
C = (c1, c2, ..., cm)
```

where `ci` is the normalized value of component `Ki` from the final record
returned to the client.

Values MUST be encoded without losing comparison-relevant information. In
particular:

- timestamps MUST preserve sufficient precision and use a canonical instant
  representation;
- integers MUST not be converted through an imprecise floating-point format;
- strings MUST use comparison semantics compatible with the database collation;
- enums MUST have a stable wire representation;
- null MUST be distinct from every non-null value.

## Page Token Contents

A token SHOULD contain a versioned payload with at least:

```json
{
  "version": 1,
  "order": [
    {"field": "name", "direction": "asc", "nulls": "last"},
    {"field": "update_time", "direction": "desc", "nulls": "last"},
    {"field": "rowid", "direction": "desc", "nulls": "last"}
  ],
  "values": ["example", "2026-08-17T09:15:21.746811654Z", 704],
  "scope_hash": "..."
}
```

The token MAY also contain:

- an expiration time;
- a snapshot identifier or high-water mark;
- a tenant, parent-resource, or authorization-scope hash;
- a normalized filter hash.

The token MUST be opaque to API clients. Encoding as Base64 is only transport
encoding and does not prevent tampering. If accepting modified tokens would
create security, isolation, or resource-consumption risks, the token MUST be
authenticated with a MAC or protected with authenticated encryption.

### Protobuf Reference Design

When page tokens are protobuf-encoded, the payload SHOULD model the complete
effective-order tuple rather than hard-code one particular pair of cursor
fields. The following schema is a reference design:

```proto
syntax = "proto3";

package pagination.v1;

import "google/protobuf/timestamp.proto";

message OrderedPageToken {
  // Version of the payload and its comparison rules.
  uint32 version = 1;

  // Complete effective order, including the unique tie-breaker.
  repeated CursorComponent cursor = 2;

  // Hash of the normalized filter, parent, tenant, authorization scope,
  // and any other constraints that define the result set.
  bytes scope_hash = 3;

  // Optional expiration and snapshot metadata.
  google.protobuf.Timestamp expire_time = 4;
  bytes snapshot = 5;
}

message CursorComponent {
  // Canonical public field name from the server allowlist.
  string field = 1;

  SortDirection direction = 2;
  NullPlacement null_placement = 3;

  // Normalized value extracted from the final returned record.
  CursorValue value = 4;
}

message CursorValue {
  oneof kind {
    NullValue null_value = 1;
    string string_value = 2;
    int64 int64_value = 3;
    uint64 uint64_value = 4;
    double double_value = 5;
    bool bool_value = 6;
    bytes bytes_value = 7;
    google.protobuf.Timestamp timestamp_value = 8;
  }
}

message NullValue {}

enum SortDirection {
  SORT_DIRECTION_UNSPECIFIED = 0;
  SORT_DIRECTION_ASCENDING = 1;
  SORT_DIRECTION_DESCENDING = 2;
}

enum NullPlacement {
  NULL_PLACEMENT_UNSPECIFIED = 0;
  NULL_PLACEMENT_FIRST = 1;
  NULL_PLACEMENT_LAST = 2;
}
```

The server MUST validate this payload more strictly than protobuf decoding
alone. In particular:

- `version` MUST identify a supported token contract;
- `cursor` MUST be non-empty and contain exactly the expected effective-order
  components;
- every `field` MUST resolve through the server sort-field allowlist;
- `direction` and `null_placement` MUST be specified and supported for that
  field;
- every `value` kind MUST match the configured cursor type for that field;
- `null_value` MUST be rejected for non-nullable fields;
- `expire_time`, when present, MUST be a valid timestamp in the future;
- `scope_hash` MUST match the current normalized query context.

The null marker is an empty message rather than a boolean because oneof presence
already represents SQL `NULL`; there is no meaningful `false` null value.

Typed scalar fields SHOULD be preferred over `google.protobuf.Struct` or
`google.protobuf.Value`. Those generic JSON-compatible messages represent
numbers as double precision and can lose exact `int64` or `uint64` cursor
values. Dedicated fields MAY be added for commonly sorted types such as UUID,
decimal, date, or enum. UUID values can otherwise be represented as exactly 16
bytes, provided the database comparison uses the same byte ordering.

The order descriptors and cursor values MAY be represented as separate repeated
fields, but combining each descriptor and value into one `CursorComponent`
makes positional mismatches less likely. Regardless of the wire shape, cursor
component order is significant and MUST match the effective SQL order.

Page size SHOULD NOT be included in the token. A client may change page size
between requests without changing the position represented by the cursor,
subject to server limits.

### Serialization and Protection

A typical token pipeline is:

1. Deterministically serialize the protobuf payload.
2. Authenticate the bytes with a versioned signing key, or encrypt them with an
   authenticated-encryption scheme.
3. Combine the key identifier, payload, and authentication data in a stable
   envelope.
4. Encode the envelope using unpadded Base64 URL encoding.

Deterministic serialization is useful for reproducibility but is not a security
mechanism. Verification MUST occur before trusting any field, direction, value,
scope, or expiration from the payload.

Avoid exposing raw protobuf bytes with ordinary Base64 when tokens contain
sensitive filter or tenant context. A MAC prevents modification but does not
hide contents; authenticated encryption provides both integrity and
confidentiality.

### Compatibility and Migration

An existing token message specialized to one fixed order SHOULD NOT be given a
new, broader meaning in place. Introduce a distinct ordered-token payload and
support both formats during a bounded migration period.

One possible envelope is:

```proto
message PageTokenEnvelope {
  uint32 envelope_version = 1;

  oneof payload {
    LegacyPageToken legacy = 2;
    OrderedPageToken ordered = 3;
  }
}
```

If legacy tokens were serialized directly without an envelope, the decoder can:

1. Recognize a new envelope using an explicit prefix or framing version.
2. Decode and validate the new ordered payload when that prefix is present.
3. Otherwise decode the old payload using its original rules.
4. Emit only new-format tokens after deployment.
5. Remove legacy decoding after the maximum token lifetime and rollout window.

Do not select a format merely because protobuf unmarshalling returned no error.
Protobuf parsers tolerate unknown fields and zero values, so unrelated payloads
can sometimes appear syntactically valid. Use explicit framing and full
semantic validation.

Field numbers and enum values in a published protobuf token contract MUST NOT be
reused. Removed fields and enum values SHOULD be reserved. Any change to value
normalization, collation interpretation, null ranking, or comparison semantics
requires a new token version even when the protobuf schema itself is unchanged.

## Token Validation

When a token is supplied, the server MUST:

1. Decode and authenticate it.
2. Validate its version.
3. Validate the number and types of cursor values.
4. Validate that all encoded order fields remain allowed.
5. Require the current requested order to equal the token order exactly.
6. Require the current query scope and filter to match the token context.
7. Reject expired tokens.

An API MAY omit `order_by` and `filter` from subsequent requests and recover
them from the token. If those request fields are supplied, they MUST match the
token rather than silently changing the query.

Page size does not need to be fixed by the token. Allowing it to change does not
alter cursor correctness, provided it remains within server limits.

## Lexicographic Continuation Predicate

For non-null cursor values, define the strict-after comparator for component
`i` as:

```text
AFTER_i(Ki, ci) = Ki > ci  when Di is ASC
                  Ki < ci  when Di is DESC
```

Define equality as:

```text
EQUAL_i(Ki, ci) = Ki = ci
```

For an effective order with `m` components, the forward continuation predicate
is:

```text
AFTER_1
OR (EQUAL_1 AND AFTER_2)
OR (EQUAL_1 AND EQUAL_2 AND AFTER_3)
...
OR (EQUAL_1 AND ... AND EQUAL_(m-1) AND AFTER_m)
```

Equivalently:

$$
P(K, C) = \bigvee_{i=1}^{m}
\left(
\left(\bigwedge_{j=1}^{i-1} K_j = c_j\right)
\land A_i(K_i, c_i)
\right)
$$

where $A_i$ is `>` for ascending order and `<` for descending order.

### Mixed-Direction Example

For:

```sql
ORDER BY name ASC, updated_at DESC, rowid DESC
```

and cursor values `:name`, `:updated_at`, and `:rowid`, the predicate is:

```sql
WHERE name > :name
   OR (name = :name AND updated_at < :updated_at)
   OR (
       name = :name
       AND updated_at = :updated_at
       AND rowid < :rowid
   )
```

The query for the next page is therefore:

```sql
SELECT ...
FROM resources
WHERE <base predicates>
  AND (
       name > :name
       OR (name = :name AND updated_at < :updated_at)
       OR (
           name = :name
           AND updated_at = :updated_at
           AND rowid < :rowid
       )
  )
ORDER BY name ASC, updated_at DESC, rowid DESC
LIMIT :page_size_plus_one;
```

The base predicates and continuation predicate MUST be combined with `AND` and
parenthesized correctly.

## Nullable Sort Fields

Nullable fields require explicit semantics because SQL three-valued logic makes
ordinary comparisons with `NULL` evaluate to unknown.

The server MUST choose `NULLS FIRST` or `NULLS LAST` for every nullable sort
component and use the same policy in:

- `ORDER BY`;
- token metadata;
- equality tests;
- strict-after tests.

A practical implementation can compare a synthetic null rank before the field
value. For `NULLS LAST`:

```sql
ORDER BY (priority IS NULL) ASC, priority ASC, rowid ASC
```

The effective cursor components become:

```text
(priority_is_null ASC, priority ASC, rowid ASC)
```

The null rank participates in the ordinary lexicographic predicate. Equality
for nullable values MUST use null-safe equality, for example:

```sql
priority IS :priority
```

in SQLite, or:

```sql
priority IS NOT DISTINCT FROM :priority
```

in PostgreSQL.

Treating null rank as an explicit cursor component is generally simpler and
less error-prone than embedding special cases throughout predicate generation.

## Query Algorithm

For the first page:

1. Validate the requested order.
2. Construct the effective order by adding the unique tie-breaker.
3. Apply base filters and authorization predicates.
4. Apply the effective `ORDER BY`.
5. Fetch `page_size + 1` records.

For a subsequent page:

1. Validate and decode the token.
2. Reconstruct the exact effective order.
3. Apply the same base filters and authorization predicates.
4. Apply the lexicographic continuation predicate.
5. Apply the effective `ORDER BY`.
6. Fetch `page_size + 1` records.

If at most `page_size` records are returned, return all records and no next-page
token. If `page_size + 1` records are returned:

1. Exclude the extra record from the response.
2. Build the next cursor from the last included record.
3. Return a token for that cursor.

The extra record proves that another page exists and avoids an unnecessary
`COUNT(*)` query.

## Filters and Query Scope

A cursor is meaningful only within the query that created it. Changing a filter,
parent resource, tenant, visibility scope, join, or authorization condition can
change the ordered result set.

The server MUST either:

- encode the normalized query context in the token; or
- encode and validate a stable hash of that context.

The normalization procedure itself is part of the token version. Semantically
equivalent but textually different filters need not share tokens unless the
server defines canonical filter normalization.

Filtering SHOULD be performed by the database before keyset pagination. If
filtering is performed in application memory, the scan cursor and response
cursor are different concepts: the implementation must continue scanning from
the last database record examined while returning a token based on enough state
to resume that scan without omissions or repeated work.

## Collations and Derived Expressions

String cursor comparisons MUST use the same collation as `ORDER BY`. For
example:

```sql
ORDER BY name COLLATE NOCASE ASC, rowid ASC
```

requires continuation comparisons using `COLLATE NOCASE` as well. Encoding only
the original string is sufficient if the database applies the identical
collation to both ordering and predicates.

For derived sorts such as `LOWER(name)`, the cursor SHOULD encode the derived
value actually used by the ordering, followed by the unique tie-breaker:

```sql
ORDER BY LOWER(name) ASC, rowid ASC
```

The same deterministic expression MUST be used in the continuation predicate.
Non-deterministic expressions are not valid cursor components.

## Mutable Sort Fields and Concurrent Writes

Keyset pagination is stable against inserts before the cursor: newly inserted
records before the current position are not repeated. It does not provide a
snapshot by itself.

When a sort field is mutable, a record updated between requests can move across
the cursor and may be omitted or observed more than once. Deletions can reduce
later pages. Inserts after the cursor can appear on later pages.

If the API requires a stable traversal, it MUST add snapshot semantics, such as:

- a database snapshot or repeatable-read transaction retained across pages;
- a materialized result-set identifier;
- a fixed high-water mark included in every query and token;
- versioned records queried as of a fixed revision.

A timestamp high-water mark only constrains changes represented by that
particular timestamp. It is not equivalent to a full database snapshot unless
the data model guarantees it.

## Backward Pagination

Backward pagination requires reversing every effective-order direction and
using the corresponding strict-after predicate in that reversed order. The
fetched records are then reversed before being returned to preserve the public
order.

Forward and backward cursor behavior, including inclusivity, MUST be defined
separately. Reusing a forward predicate unchanged for backward traversal is
incorrect.

## Index Requirements

For scalable keyset pagination, create an index whose leading columns and
collations match the effective order. For example:

```sql
CREATE INDEX resources_name_updated_rowid
ON resources (name ASC, updated_at DESC, rowid DESC);
```

Database support for mixed-direction index scans varies, but the logical cursor
contract remains the same. Base-filter columns may need to precede sort columns
when they are fixed by equality, such as:

```sql
CREATE INDEX resources_tenant_name_rowid
ON resources (tenant_id, name ASC, rowid ASC);
```

Query plans SHOULD be tested for each supported order definition.

## Implementation Architecture

Ordering metadata SHOULD have one canonical representation. From a single list
of validated sort components, the implementation should derive:

1. The effective `ORDER BY`.
2. Cursor value extraction.
3. Token serialization and type validation.
4. The lexicographic continuation predicate.
5. Index recommendations and test fixtures.

Do not implement these as unrelated switch statements. Their agreement is the
core correctness invariant.

A conceptual component definition is:

```go
type SortComponent[R any] struct {
    APIName       string
    Direction     Direction
    NullPlacement NullPlacement
    SQLExpression string
    ValueType     CursorValueType
    GetValue      func(R) CursorValue
}
```

In production, SQL expressions should be represented by typed query-builder
functions rather than unrestricted strings.

## Required Tests

Each supported ordering SHOULD be tested with:

1. Zero records.
2. Fewer than, exactly, and more than one page of records.
3. Equal primary sort values requiring the unique tie-breaker.
4. Mixed ascending and descending components.
5. Null values for every nullable component.
6. Minimum and maximum values for every cursor type.
7. Timestamp precision and timezone normalization.
8. Strings that differ under the configured collation.
9. A changed order, filter, parent, or tenant used with an old token.
10. Malformed, unsupported-version, expired, and tampered tokens.
11. Full traversal with no duplicate IDs, no omitted IDs, and no token cycles.
12. Inserts, updates, and deletes between pages, with behavior checked against
    the API's stated consistency guarantees.

Property-based testing is particularly useful: generate records with many tied
values, sort them using the declared effective order, paginate them, and assert
that concatenated pages equal the expected sequence exactly.

## Summary of Normative Requirements

A conforming implementation MUST:

- accept only known sort fields and directions;
- establish a deterministic total order;
- include a unique stable tie-breaker;
- encode all effective-order cursor values;
- generate a lexicographic predicate matching all directions and null rules;
- apply the identical order on every page;
- bind tokens to order and query scope;
- reject incompatible or malformed tokens;
- use canonical, comparison-safe value representations;
- return tokens only when another record exists.

If any effective-order component is absent from the token or continuation
predicate, the pagination algorithm is not correct.
