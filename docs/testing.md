# dtkt-sdk testing

Project-specific testing rules for `dtkt-sdk`. Universe-wide Go testing
conventions (gotest.tools usage, test-doubles philosophy, golden helper shape,
`internal/testutil` discipline) live in
[../../dtkt-dev/docs/go/testing.md](../../dtkt-dev/docs/go/testing.md) - read
that first.

This file documents what is unique to `dtkt-sdk`: the proto-heavy comparison
needs, the schemas/golden layout, and the boundary between contract tests
(proto/GraphQL) and runtime tests (the hand-written `*sdk` packages).

## Use this for that

### `sdktest` is the shared helper home

- **`sdktest`** lives at [sdk-go/sdktest/](../sdk-go/) as a public Go package -
  imported by every other Go module that needs proto comparison or golden
  assertions.
- **The two core helpers** are `AssertProtoEqual` and `Golden`, per
  [../../dtkt-dev/docs/go/testing.md](../../dtkt-dev/docs/go/testing.md).
- **Project-specific dtkt-sdk helpers** go in `sdk-go/internal/testutil/`,
  not `sdktest`. The bar for `sdktest` is "needed by at least one other
  module."

### Proto value comparison

- **`sdktest.AssertProtoEqual(t, want, got)`** for any proto comparison.
  Wraps `cmp.Diff` with `protocmp.Transform()` so unset fields, oneof
  selection, and repeated-field ordering all compare correctly.
- **Never use `reflect.DeepEqual` or `proto.Equal` directly in tests.** The
  former produces unreadable diffs for opaque API_HYBRID messages; the latter
  returns a bool with no diff context.
- **Builder construction** in test fixtures, mirroring the
  [API_HYBRID rules](../../dtkt-dev/docs/go/style.md#protobuf-api_hybrid-conventions)
  used in production code:

  ```go
  want := flowpb.Flow_builder{
      Id:   proto.String("test-flow"),
      Name: proto.String("Test"),
  }.Build()
  ```

### Golden files

- **`sdktest.Golden(t, name, got)`** is the single entry point for any golden
  assertion (proto, JSON, YAML, strings, bytes). Type-dispatches on `got`.
- **Goldens live at `testdata/golden/<name>`** next to the test that uses
  them. Examples include the JSON Schema output exercised by
  [common/json_schema_test.go](../sdk-go/common/json_schema_test.go).
- **The flow spec YAML under [sdk-go/flowsdk/v1beta2/runtime/testdata/](../sdk-go/flowsdk/v1beta2/runtime/testdata/)
  is not a golden tree.** Those files are flow spec *inputs* the runtime tests
  execute against; assertion outputs are golden files under `testdata/golden/`.
  Do not conflate the two layouts.
- **Update goldens with `task test:update-goldens`** (defined in the shared Go
  Taskfile; passes `-update` to `go test`). A golden change must be reviewed
  in the PR; bulk-updating without inspecting is a review-blocker.

### Contract vs runtime tests

- **Contract tests** (proto + GraphQL) live alongside the source files in
  [proto/](../proto/) and [graph/](../graph/). They assert on the shape of
  the contract: field numbers are stable, lint exceptions stay scoped, breaking
  rules pass. `task lint` runs `buf lint` and `buf breaking` - those *are*
  the contract tests.
- **Runtime tests** live under [sdk-go/](../sdk-go/) and [sdk-ts/](../sdk-ts/),
  one `_test.go` next to the file under test. They assert on hand-written
  wrapper behavior: `flowsdk` execution, `integrationsdk` spec resolution,
  `protostoresdk` round-trips, encoding helpers, network addr parsing.
- **Don't write contract tests in Go.** If you find yourself writing a test
  that asserts on field numbers or service names, it belongs in `buf` lint
  config, not in `_test.go`.

### Generated code is not tested directly

- **`sdk-go/proto/`, `sdk-go/cloud/graph/generated.go`, `sdk-ts/src/proto/`,
  `sdk-ts/src/graphql/generated.ts`** are codegen output. Tests live around
  them (in the hand-written `*sdk` packages) but never import the generated
  tree as the unit under test.
- **If a test of generated code seems necessary, the contract is wrong** -
  fix the proto/graphql source instead.

### `templates/` testing

- **`templates/` generates integration packages.** Tests live in
  [templates/](../templates/) and exercise `GeneratePackage()` against fixed
  `Spec` inputs.
- **Snapshot the rendered template tree as goldens.** A change to a template
  file shows up as a diff under `testdata/golden/<spec-name>/`.
- **Keep template tests independent of `sdk-go/go.mod`.** The
  [templates/go.mod](../templates/go.mod) split is intentional; do not import
  test helpers from `sdk-go/internal/testutil` into `templates/`.

### sdk-ts

- **Vitest is the runner.** `pnpm test` (which is `vitest run`) at the
  [sdk-ts](../sdk-ts/) root, or `pnpm test:watch` during development.
  Configured in [`sdk-ts/package.json`](../sdk-ts/package.json).
- **Tests live next to the code under test** as `*.test.ts`. Reference:
  [`src/protoformsdk/form/message.test.ts`](../sdk-ts/src/protoformsdk/form/message.test.ts).
- **Generated proto / GraphQL trees are not under test.**
  `src/proto/`, `src/graphql/generated.ts` are codegen output - tests live
  in the hand-written wrapper packages (`cloud/`, `integrationsdk/`,
  `protoformsdk/`).
- **No build step before testing.** `tsdown` builds for publish; tests run
  against `src/*.ts` directly. Don't introduce a pre-test compile.

### Cross-language behavior

- **Don't try to run TS tests from Go or vice versa.** Each package has its
  own `task test`. The contract layer (proto, graphql) is what cross-validates;
  if Go and TS clients see the same contract, they see the same behavior.
- **`(cd sdk-go && task test)` and `(cd sdk-ts && pnpm test)`** are the
  per-language entry points. `task test` from the repo root fans out.

## Patterns to add

> Add as runtime helpers and fixtures grow - canonical Resource/Deployment
> builders for tests across `flowsdk`/`integrationsdk`, `assertproto.Equal`
> options for ignoring `created_at`/`updated_at` timestamps, golden layout
> for the `templates/` rendered tree once that suite lands.
