# AGENTS.md

This file provides guidance for AI coding agents working in this repository.

## Overview

`dtkt-sdk` is a multi-language client SDK for the DataKit platform, generated from a shared schema contract. The contract has two halves: protobuf service definitions under [proto/](proto/) (covering Connect/gRPC services like `dtkt.action`, `dtkt.flow`, `dtkt.protostore`, etc.) and the GraphQL schema under [graph/](graph/) (the cloud-go API surface, plus operations consumed by SDK callers). Two language-specific packages are produced from that contract and published independently: a Go module ([sdk-go/](sdk-go/), `github.com/datakit-dev/dtkt-sdk/sdk-go`) and a TypeScript package ([sdk-ts/](sdk-ts/), `@dtkt/sdk` on npm). The proto/graph files are the source of truth - never hand-edit anything in `sdk-go/proto/`, `sdk-ts/src/proto/`, `sdk-ts/src/graphql/generated.ts`, or `sdk-go/cloud/graph/generated.go`.

This repo is its own independent git repository. When opened standalone (without the `dtkt-universe` siblings), only `task generate` and the per-language tasks are useful - the universe-level orchestration in [../AGENTS.md](../AGENTS.md) does not apply.

## Commands

All commands run from the repo root unless noted.

| Goal | Command |
| --- | --- |
| Install codegen toolchain (`buf`, `protoc-gen-gosrcinfo`) and per-language deps | `task setup` |
| Full codegen chain (proto + Go + TS + docs descriptor) | `task generate` |
| Lint all (buf + Go + TS) | `task lint` |
| Run all tests | `task test` |
| Build the file descriptor set used by docs | `task docs` (writes [docs/protos.binpb](docs/protos.binpb)) |
| Regenerate Go code only (`buf generate` + `genqlient`) | `(cd sdk-go && task generate)` |
| Regenerate TS code only (`graphql-codegen`) | `(cd sdk-ts && task generate)` |
| Run buf directly (proto lint / generate / breaking) | `buf lint`, `buf generate`, `buf breaking` |
| sdk-go test / lint | `(cd sdk-go && task test)` / `(cd sdk-go && task lint)` |
| sdk-ts build / test / typecheck | `(cd sdk-ts && pnpm build)` / `pnpm test` / `pnpm typecheck` |
| Refresh license notices | `(cd sdk-go && task legal:update-license-notices)` and same in `sdk-ts/` |

`task generate` is the canonical entry point. It runs in this order: `buf generate` (proto → Go and TS), then `task go:generate` (genqlient → `sdk-go/cloud/graph/generated.go`), then `task ts:generate` (graphql-codegen → `sdk-ts/src/graphql/generated.ts`), then `task docs` (binary descriptor set). See [Taskfile.yaml](Taskfile.yaml).

The Go side sets `GOEXPERIMENT=jsonv2` (per [sdk-go/Taskfile.yaml](sdk-go/Taskfile.yaml)) and pulls shared task definitions via the `TASK_PATH`-rooted `Taskfile.go.yaml` - that file lives in the universe `dtkt-dev/tasks` and is only available when this repo is checked out as part of `dtkt-universe`. Standalone clones can still run `task generate` (which doesn't depend on `go:setup`/`go:lint`/etc.) but `(cd sdk-go && task lint)` will fail without it.

The root [Taskfile.yaml](Taskfile.yaml) `includes` both subrepos under `go:` and `ts:` namespaces, so any task in [sdk-go/Taskfile.yaml](sdk-go/Taskfile.yaml) is reachable as `task go:<name>` from the repo root, and the same for `sdk-ts/` as `task ts:<name>`. Both subrepos define `setup`, `generate`, `lint`, `test`, and `legal:*` license-notice tasks; only `sdk-ts` has `build` and `publish`.

## Architecture

The contract layer:

- **[proto/](proto/)** - Protobuf source of truth. Module `buf.build/datakit-dev/dtkt-sdk` (see [buf.yaml](buf.yaml)). Subpackages cover the platform domains: `dtkt.action`, `dtkt.ai`, `dtkt.base`, `dtkt.blob`, `dtkt.catalog`, `dtkt.cli`, `dtkt.command`, `dtkt.core`, `dtkt.email`, `dtkt.event`, `dtkt.flow`, `dtkt.geo`, `dtkt.geojson`, `dtkt.protoform`, `dtkt.protostore`, `dtkt.protoui`, `dtkt.replication`, `dtkt.shared`. Depends on `protovalidate`, `cel-spec`, and `googleapis` (longrunning is regenerated locally - see [buf.gen.yaml](buf.gen.yaml)). A few lint exceptions are pinned in [buf.yaml](buf.yaml) (`ENUM_VALUE_PREFIX`, `SERVICE_SUFFIX`, and the `RPC_*` rules for `proto/dtkt/core/v1/`). Custom extension option ranges are reserved per [proto/README.md](proto/README.md) (`dtkt.protoform`: 50000-50004, `dtkt.protostore`: 50005-50009).
- **[graph/](graph/)** - GraphQL contract. [graph/schema.graphqls](graph/schema.graphqls) is **auto-mirrored from `dtkt-cloud/cloud-go/schema.generated.graphqls`** by the universe-level `task generate` (cloud-go is the source of truth for the schema; this repo is a downstream consumer that re-exports it for SDK clients). When this repo is opened standalone, the committed file is the canonical input; cross-repo refresh happens at the universe level. [graph/operations/](graph/operations/) holds the queries/mutations the SDK exposes - one file per resource (`flow`, `flow-run`, `flow-revision`, `connection`, `source`, `source-type`, `destination`, `space`, `model`, `model-type`, `catalog`, `package`, `organization`, `personal-access-token`, `roles-permissions`, `event-source`, `column`, `table`, `io-schema`, `search`, `schema`).
- **[schemas/](schemas/)** - JSON Schema files generated from the proto definitions (~600 `*.jsonschema.json` files), used by external consumers that want runtime validation without a protobuf runtime. The JSON-schema buf plugin is currently commented out in [buf.gen.yaml](buf.gen.yaml); these files are regenerated out-of-band.

The codegen layer:

- **[templates/](templates/)** - Code-generation templates for **scaffolding integration packages** (not for generating the SDKs themselves). [templates/gen.go](templates/gen.go) embeds [templates/go/](templates/go/) (Dockerfile, Taskfile, `main.go`, proto skeleton, per-service stubs) and exposes `GeneratePackage()` which is consumed by `dtkt-cli` (and indirectly `dtkt-integrations`) to bootstrap new integration repos. The package selects which template files to emit based on the `Spec`'s package type, runtimes, services, and proto/lib flags. [templates/go.mod](templates/go.mod) is intentionally separate from [sdk-go/go.mod](sdk-go/go.mod).

The published packages:

- **[sdk-go/](sdk-go/)** - Published Go module `github.com/datakit-dev/dtkt-sdk/sdk-go`. `sdk-go/proto/` and `sdk-go/cloud/graph/generated.go` are generated; everything else is hand-written: `cloud/` (GraphQL client wrapper + genqlient output), `flowsdk/`, `integrationsdk/`, `protoformsdk/`, `protostoresdk/`, `pubsub/` (publish/subscribe interface + memory backend + middleware/forwarder; cross-cutting primitive, not flow-specific), `cache/` (key/value cache interface + memory backend; cross-cutting), `middleware/`, `network/`, `encoding/`, `lib/`, `resource/`, `uri/`, `util/`, `common/`, `api/`. genqlient is invoked via `go tool genqlient` from [sdk-go/generate.go](sdk-go/generate.go).
- **[sdk-ts/](sdk-ts/)** - Published npm package `@dtkt/sdk`. `src/proto/` and `src/graphql/generated.ts` are generated; `src/cloud/`, `src/integrationsdk/`, `src/protoformsdk/` are hand-written. Built with `tsdown` to ESM. Public exports are limited via `package.json#exports` to `./cloud/*`, `./proto/*`, and `./protoformsdk/*` - `integrationsdk` is internal.

## Codegen

```
proto/**.proto ──► buf generate ──► sdk-go/proto/  (protocolbuffers/go + grpc/go + connectrpc/go + gosrcinfo)
                                 └► sdk-ts/src/proto/  (bufbuild/es + connectrpc/query-es)

graph/schema.graphqls (mirrored from cloud-go)
   + graph/operations/**.graphql
       │
       ├─► genqlient ──► sdk-go/cloud/graph/generated.go    (config: sdk-go/genqlient.yaml)
       └─► graphql-codegen ──► sdk-ts/src/graphql/generated.ts  (config: sdk-ts/codegen.ts)
```

Plugin pins live in [buf.gen.yaml](buf.gen.yaml) (e.g. `protocolbuffers/go:v1.36.10`, `bufbuild/es:v2.10.1`, `connectrpc/go:v1.19.1`). Go output uses `default_api_level=API_HYBRID` and `paths=source_relative`. TS output targets `.ts` (not compiled JS). Connect for googleapis longrunning is generated with `include_imports: false` to avoid producing Connect handlers for upstream services.

GraphQL scalar bindings differ between Go and TS and are intentional:

| Scalar | Go (genqlient) | TS (graphql-codegen) |
| --- | --- | --- |
| `ID` | `github.com/google/uuid.UUID` | (default `string`) |
| `Time`, `DateTime` | `time.Time` | `string` |
| `Int64` | `encoding/json.Number` | `number` |
| `Map` | `map[string]any` | `Record<string, unknown>` |
| `Any` | `any` | `unknown` |
| `Bytes` | `[]byte` | `string` |
| `Cursor` | (default) | `string` |

See [sdk-go/genqlient.yaml](sdk-go/genqlient.yaml) and [sdk-ts/codegen.ts](sdk-ts/codegen.ts). genqlient is configured with `optional: pointer` and `use_struct_references: true`, so optional GraphQL fields surface as `*Foo` in Go.

The `templates/` codegen flow is unrelated to the SDKs - it generates new *integration packages* and is invoked by `dtkt-cli`'s package scaffolding command.

## Publishing

Both packages ship via release-please. [release-please-config.json](release-please-config.json) declares two components, each released independently:

- **sdk-go** - `release-type: go`, tagged `sdk-go-vX.Y.Z`. Consumers depend on `github.com/datakit-dev/dtkt-sdk/sdk-go` via the Go module proxy; there is no separate publish step beyond the GitHub tag. `bump-minor-pre-major` is on, so 0.x minor bumps signal breaking changes.
- **sdk-ts** - `release-type: node`, package name `@dtkt/sdk`, tagged `sdk-ts-vX.Y.Z`. Publish happens in [.github/workflows/publish-sdk-ts.yaml](.github/workflows/publish-sdk-ts.yaml) via `(cd sdk-ts && task publish)`, which copies the root `LICENSE`/`NOTICE` (stripping the `### sdk-go` section) into the package, runs `pnpm publish`, and removes them again. Provenance is enabled (`publishConfig.provenance: true`).

Release-please workflow is wired in [.github/workflows/release-please.yaml](.github/workflows/release-please.yaml). Per-package CI lives in [.github/workflows/sdk-go.yaml](.github/workflows/sdk-go.yaml) and [.github/workflows/sdk-ts.yaml](.github/workflows/sdk-ts.yaml).

## Conventions

- **Adding a new GraphQL operation** means adding a file under [graph/operations/](graph/operations/) and re-running `task generate`. Both `genqlient` and `graphql-codegen` discover operations via `../graph/**/*.graphql` globs.
- **Adding a new proto package** requires placing it under `proto/dtkt/<area>/v<N>/` and re-running `task generate`. Lint exemptions in [buf.yaml](buf.yaml) are file-scoped - don't widen them; introduce new APIs that already follow `STANDARD` rules.
- **TS `package.json#exports` is a deny-by-default surface.** Anything not under `./cloud/*`, `./proto/*`, or `./protoformsdk/*` is unreachable by consumers, even if it builds. Add new entry points there explicitly. The dev `exports` use `.ts` and `publishConfig.exports` rewrites them to `dist/**.mjs`.
- **TS catalog deps.** All `@bufbuild/*`, `@connectrpc/*`, and `@graphql-codegen/*` versions come from the universe-level pnpm catalogs (`catalog:proto`, `catalog:graphql`, etc.). When this repo is cloned standalone, `pnpm install` will fail unless run from inside `dtkt-universe` - there's no fallback.
- **`templates/go.mod` is its own module** (separate from `sdk-go/go.mod`) so embedded templates can compile independently. Don't try to merge them; `templates/gen.go` consumes `sdk-go` as a normal import, but the `templates/go/` tree is treated as text resources, not as a buildable Go package.
- **`go_package_prefix` is overridden** in [buf.gen.yaml](buf.gen.yaml) to `github.com/datakit-dev/dtkt-sdk/sdk-go/proto`, with `go_package` disabled for the `protovalidate`/`cel-spec`/`googleapis` upstream modules so their generated code points back at the canonical paths instead of being re-vendored.
- **TS proto generation uses `target=ts` and `valid_types=protovalidate_required`.** Don't change these in [buf.gen.yaml](buf.gen.yaml) without checking that consumers compile - the SDK ships TS source and relies on the consumer's `tsconfig` for `.ts` resolution.
- **`include_imports: false` on the Connect-Go plugin** is intentional: longrunning operations from googleapis are pulled in for message types only, not for handlers. Don't flip it without reviewing the generated surface.
- Contributor-facing rules (commit conventions, sign-off, branch naming, PR rules) are in [CONTRIBUTING.md](CONTRIBUTING.md).

## Gotchas

- **Never hand-edit generated files.** That includes `sdk-go/proto/**`, `sdk-ts/src/proto/**`, `sdk-go/cloud/graph/generated.go`, and `sdk-ts/src/graphql/generated.ts`. Regenerate via `task generate`.
- **`graph/schema.graphqls` is a mirror.** If the schema looks stale, fix it upstream in `dtkt-cloud/cloud-go` and re-export - don't patch this file in isolation. The TS `codegen.ts` and Go `genqlient.yaml` both reference `../graph/schema.graphqls` directly.
- **`buf breaking` uses `FILE` mode.** Renaming or removing fields/messages will fail breaking-change checks. Add new fields with new numbers; don't reuse.
- **`schemas/` files are not regenerated by `task generate`.** The JSON-schema plugin is commented out in [buf.gen.yaml](buf.gen.yaml); refreshing those files is a manual step (uncomment, run `buf generate`, recomment).
- **`(cd sdk-ts && task publish)` mutates the working tree.** It copies `../LICENSE` and a filtered `../NOTICE` into `sdk-ts/` for the publish, then deletes them. If the publish fails partway through, those files may be left behind - clean them up before committing.
- **`task setup` installs `buf` and `protoc-gen-gosrcinfo` globally** via `go install`. Make sure `$GOBIN` (or `$GOPATH/bin`) is on `PATH` before running `task generate`, or buf plugin invocation will fail with `protoc-gen-gosrcinfo: not found`.

## Style

[Go style](../dtkt-dev/docs/go/style.md) ·
[TypeScript style](../dtkt-dev/docs/typescript/style.md) ·
[project-specific](docs/style.md). Note: the Go style guide's API_HYBRID section is
particularly relevant here - it documents the builder/accessor pattern that the
generated `sdk-go` exposes.

## Testing

[Go testing](../dtkt-dev/docs/go/testing.md) ·
[TypeScript testing](../dtkt-dev/docs/typescript/testing.md) (placeholder) ·
[project-specific](docs/testing.md) (sdk-go: proto comparison via
`sdktest.AssertProtoEqual`, unified `sdktest.Golden`, contract-vs-runtime
split; sdk-ts: vitest with tests next to source).
