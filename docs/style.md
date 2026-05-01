# dtkt-sdk style

Conventions for writing new code in `dtkt-sdk` - proto/GraphQL contract, hand-written wrappers, and the published `sdk-go` / `sdk-ts`. Operational facts (codegen pipeline, scalar bindings, release-please wiring) are in [../AGENTS.md](../AGENTS.md).

For language-level rules:
- Go: [../../dtkt-dev/docs/go/style.md](../../dtkt-dev/docs/go/style.md) (the API_HYBRID section is particularly relevant - it documents the builder/accessor pattern that the generated `sdk-go` exposes)
- TypeScript: [../../dtkt-dev/docs/typescript/style.md](../../dtkt-dev/docs/typescript/style.md)

For contributor-facing rules (commit conventions, sign-off, PR/branch naming): [../CONTRIBUTING.md](../CONTRIBUTING.md).

## Use this for that

### Adding to the contract

- **New GraphQL operation** = new file under [graph/operations/](../graph/operations/), one per resource. Both genqlient and graphql-codegen pick up the glob automatically.
- **New proto package** lives at `proto/dtkt/<area>/v<N>/`. Follow the directory convention; field numbers in a given message are append-only.
- **New proto fields use new numbers.** `buf breaking` is in `FILE` mode - renaming or reusing field numbers fails the breaking-change check.
- **Lint exceptions are file-scoped and frozen.** [buf.yaml](../buf.yaml) pins exceptions for `ENUM_VALUE_PREFIX`, `SERVICE_SUFFIX`, and `RPC_*` under `proto/dtkt/core/v1/`. New APIs follow `STANDARD`; if a new API would need an exception, fix the API instead.

### Generated vs hand-written

- **Never hand-edit generated trees.** `sdk-go/proto/`, `sdk-ts/src/proto/`, `sdk-go/cloud/graph/generated.go`, `sdk-ts/src/graphql/generated.ts` are all output. Add capability by changing the input (proto / GraphQL operations) and re-running `task generate`.
- **Hand-written wrappers live outside the generated trees.** New Go helpers go in `sdk-go/{cloud,flowsdk,integrationsdk,protoformsdk,protostoresdk,middleware,network,encoding,lib,...}`; new TS helpers go in `sdk-ts/src/{cloud,integrationsdk,protoformsdk}`.

### TypeScript exports

- **`package.json#exports` is deny-by-default.** New consumer-facing entry points must be added under `./cloud/*`, `./proto/*`, or `./protoformsdk/*` - anything else is unreachable even if it builds.
- **TS proto generation uses `target=ts`** (source, not compiled JS). Consumers compile via their own `tsconfig`. Don't add a build step for the proto output.

### Templates (`templates/`)

- **`templates/` generates *integration packages*, not the SDKs.** New template files belong under [templates/go/](../templates/go/); the embed in [templates/gen.go](../templates/gen.go) picks them up.
- **Keep `templates/go.mod` separate** from `sdk-go/go.mod` - embedded templates need to compile independently.

## Patterns to add

> Add as the contract evolves - preferred way to extend `dtkt.shared` types, conventions for streaming RPCs, scalar binding decisions for new GraphQL types, when to break out a new `dtkt.<area>` package vs extend an existing one.
