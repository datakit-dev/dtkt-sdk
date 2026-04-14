# Implementation Plan -- Completed Phases

> Active plan: `docs/plan.md`. Architecture reference: `docs/architecture.md`.

---

## Phase 1 — Dead Code & Vestigial Cleanup ✅

> Remove dead code, stubs, and vestiges of replaced systems before building new
> things on top. Small, safe deletions -- tests must still pass after each step.
>
> **Status: COMPLETE.** All items verified -- no `StateStore`, `state/` dir,
> watermill references, or dead imports remain in source. `task build` and
> `task test` pass.

### 1.1 ~~Rename `WithRegistry` → `WithMethods`~~ (done)
Renamed: `MethodRegistry` interface → `Methods`, `Registry` struct → `MethodMap`,
`WithRegistry()` option → `WithMethods()`, all `registry` fields/params → `methods`.
"Registry" is reserved for protobuf registry (populated by Connection nodes).

`rpc/rpc.go` (`rpc.Client`, `rpc.Connector`, `rpc.BidiStream`, etc.) defines
the proto-based RPC contract. `common/resolver.go` defines the minimal
`Resolver` interface (`FindMethodByName`). `rpc/mock/mock.go` provides an
in-memory implementation for tests. Connection nodes use these for dynamic
method calls.

### 1.2 Delete `state/sqlite/sqlite.go`
Panic stub (`"not implemented"` on all methods). The old `StateStore` shape
(`Save/Load` with `*expr.Value`) is dead -- state will be managed by the executor
in the same transaction as the outbox publish. No separate state package.

### 1.3 Delete `state/memory/memory.go`
Implements the `StateStore` interface which is dead. No tests depend on it.

### 1.4 Remove `StateStore` interface from `executor/executor.go`
The `StateStore` interface (`Save`/`Load` with `*expr.Value`) is disconnected from
PubSub -- no transactional guarantee. State writes will be executor logic in the
same DB transaction as `ForwarderPublisher.Publish()`. Remove the interface, the
`WithStateStore()` option, and any wiring in the runtime executor.

### 1.5 Remove Watermill reference
`TopicFor()` in `executor/executor.go` has a comment referencing "watermill topic
name". Update or remove the comment.

### 1.6 Clean up `main.go`
Remove the `statemem` import (`state/memory`) and `runtime.WithStateStore(statemem.NewStore())`
call. After 1.2-1.4 delete the `StateStore` interface and its implementations,
`main.go` will fail to compile if this isn't addressed. Confirm `task build` passes.

---

## Phase 2 — PubSub Core (`pubsub/`) ✅

> `pubsub/` provides core messaging abstractions (like Watermill's `message/`
> package). Only gochannel lives here as a backend. SQL storage is NOT a PubSub
> backend -- it belongs in the separate `outbox/` package.
>
> **Status: COMPLETE.** Message redesign (UUID, Metadata, Context, idempotent
> Ack/Nack, CopyMessage), gochannel backend (ack-before-next, nack redelivery,
> fan-out), HandlerFunc/Middleware types, 6 middleware (retry, poison, recoverer,
> throttle, timeout, correlation), and Forwarder all implemented and tested.
> Coverage: gochannel 90%, middleware 93.2%, forwarder 62.2%.
> `task build` and `task test` pass.

### 2.1 Redesign `pubsub.Message`
Previous: `Node *RunSnapshot_Node` + ack/nack channels (nack is a no-op).

Implemented:
```go
type Message struct {
    UUID     string
    Metadata map[string]string
    Payload  proto.Message
    ctx      context.Context
    ackCh    chan struct{}
    nackCh   chan struct{}
}
```
- UUID: unique per message (for dedup, outbox tracking, CloudEvents `id`)
- Metadata: correlation ID, tracing, CE fields at system boundary
- Context: per-message cancellation/deadline
- Payload: generic proto.Message (replaces `Node *RunSnapshot_Node`)
- Ack/Nack: idempotent; Nack triggers redelivery

### 2.2 Fix Nack → redelivery
Nacked messages are redelivered to the subscriber. The gochannel backend
re-enqueues nacked messages at the tail of the subscriber's buffer (immediate
re-enqueue, not delayed). This means the message is retried after all currently
buffered messages are delivered.

### 2.3 Subscriber-waits-for-ack delivery
Subscriber acks the current message before receiving the next one. Ensures
ordered processing per subscriber. Per-subscriber gating -- not a global lock.
In a fan-out topology, each subscriber independently gates on its own ack.

### 2.4 HandlerFunc and Middleware types
```go
type HandlerFunc func(msg *Message) ([]*Message, error)
type Middleware func(h HandlerFunc) HandlerFunc
```
Defined in `pubsub/handler.go`.

HandlerFunc does NOT replace node handlers. Node handlers use `Run(ctx) error`
(long-lived goroutines) and that pattern stays. HandlerFunc is a pubsub
message-level concept used for transform steps and middleware wrapping.

### 2.5 Middleware implementations (`pubsub/middleware/`)
- **Retry** -- re-run handler on error, with exponential backoff
- **Poison queue** -- after N retries, route to a dead-letter topic
- **Recoverer** -- catch panics, convert to errors
- **Throttle** -- rate-limit handler invocations
- **Timeout** -- cancel handler if it exceeds deadline
- **Correlation ID** -- propagate correlation ID via `Message.Metadata`

### 2.6 Rename + update gochannel backend
Renamed `pubsub/memory/` → `pubsub/gochannel/`. Updated:
- UUID assigned on publish
- Context propagated
- Fan-out copies share the same UUID
- Nack → redelivery (re-enqueue)
- Ack-before-next-message contract enforced

### 2.7 Forwarder (`pubsub/forwarder/`)
- `ForwarderPublisher` -- Publisher decorator. Wraps message in envelope with
  destination topic, publishes to a forwarder topic on any Publisher.
- `Forwarder` -- background daemon. Subscribes to forwarder topic on any
  Subscriber, unwraps envelopes, publishes to real broker Publisher.
- Generic: composes any Subscriber (input) + any Publisher (output). No DB
  dependency -- purely a pubsub-level relay abstraction.

---

## Phase 3 — Outbox Pattern (`outbox/`) ✅

> The outbox package builds on top of `pubsub/` (including `pubsub/forwarder/`)
> and provides transactional event storage. The outbox interfaces support
> multiple backends (memory, SQLite, PostgreSQL). This phase implements the
> interfaces and the **memory backend only** -- enough to test the full
> transactional semantics end-to-end without serialization concerns.
> SQLite and PostgreSQL backends are deferred to Phase 6.5.
>
> **Status: COMPLETE.** Outbox interfaces (Storage, EventReader, Tx, TxBeginner,
> Outbox combined interface), memory backend (19 tests, 98.5% coverage),
> PublisherAdapter + SubscriberAdapter (11 tests, 88.6% coverage), runtime
> wiring (txPublisher, outboxPubSub, WithOutbox option, Forwarder daemon in
> Execute()), and end-to-end integration tests (4 tests verifying outbox
> relay through input→var→output chains, multi-hop chains, range generators,
> and snapshot captures). `task build` and `task test` pass. All 7 packages
> green.

### 3.1 Outbox storage interface (`outbox/outbox.go`)
```go
type Storage interface {
    Store(ctx context.Context, topic string, msg *pubsub.Message) error
    Read(ctx context.Context, afterUID uuid.UUID, limit int) ([]*StoredMessage, error)
    MarkForwarded(ctx context.Context, uids ...uuid.UUID) error
}

type EventReader interface {
    SnapshotAt(ctx context.Context, uid uuid.UUID) (*flowv1beta2.RunSnapshot, error)
}

type Tx interface {
    Storage() Storage
    Commit() error
    Rollback() error
}

type TxBeginner interface {
    Begin(ctx context.Context) (Tx, error)
}
```

### 3.2 In-memory outbox storage (`outbox/memory/`)
- Implements `Storage`, `EventReader`, and `TxBeginner`
- Mutex-guarded append buffer with staged writes visible only on `Commit()`
- Used in unit tests for Forwarder, outbox wiring, and executor integration

### 3.3 Outbox wiring
Per-message transaction loop in each node handler goroutine:
1. Receive message (ack-before-next contract gates delivery)
2. Process: CEL eval, RPC call, or other node-specific logic
3. Open tx via `TxBeginner.Begin(ctx)`
4. Write state in tx (node value, transform accumulator state)
5. Wrap `tx.Storage()` with `ForwarderPublisher` → handler publishes NodeEvents
   to outbox in the same tx
6. `tx.Commit()` -- atomic state + event
7. Ack the input message
8. `Forwarder` daemon reads committed rows from outbox, relays to gochannel

On error: `tx.Rollback()`, nack the message (triggers redelivery).

### 3.4 Tests
- Memory outbox: Store/Read/MarkForwarded, transaction commit visibility,
  rollback discarding, double-commit error, SnapshotAt
- Forwarder + outbox: end-to-end relay from memory outbox → gochannel

---

## Phase 4 — PubSub-Based Transforms (partial) ✅

> Sub-phases 4.0–4.3 and 4.5 are complete. Sub-phases 4.4 and 4.6 remain --
> see `docs/plan.md`.

### 4.0 Prerequisite: Generic Message Payload ✅
Renamed `pubsub.Message.Node *RunSnapshot_Node` → `Payload proto.Message`. Both
`*RunSnapshot_Node` and `*expr.Value` implement `proto.Message`. Updated all callers
with type assertions. Memory outbox `SnapshotAt` type-asserts to `*RunSnapshot_Node`.

### 4.1 Transform pipeline model ✅
Each transform step is a goroutine connected by pubsub topics:

```
node handler → publish *expr.Value → nodes.{id}.transform.input
                                          ↓ (filter goroutine)
                                     nodes.{id}.transform.0
                                          ↓ (map goroutine)
                                     nodes.{id}.transform.1
                                          ↓ (wrapper goroutine)
                                     nodes.{id}  ← *RunSnapshot_NodeEvent (OUTPUT)
```

Transform step signature: `func(*expr.Value) (*expr.Value, error)` -- nil
return means "don't publish" (filter drops, reduce accumulates silently).
EOF flows through the pipeline as a regular value; reduce reacts to EOF.

Implemented with `Start()`, `runStep()`, `runSink()`, `applyStep()` pattern.
Steps subscribe synchronously before goroutines launch (no race condition).

Topic naming:
- `nodes.{id}.transform.input` -- handler publishes raw `*expr.Value` here
- `nodes.{id}.transform.{index}` -- inter-step topics (0-indexed)
- Final step publishes to `nodes.{id}` (the node's real output topic)

### 4.2 Transform step compilation ✅
`compileTransforms()` returns a ready-to-run pipeline:
- Compiles CEL programs for map/filter expressions
- Compiles accumulator state for reduce/scan (init expression + accumulator CEL)
- Compiles group_by key expression (for grouped reduce/scan)
- Returns `transformPipeline` started with `Start(ctx, g, pubsub, id, sink)`

### 4.3 Reduce/scan EOF semantics ✅
- **reduce**: on regular value, update accumulator silently (nil). On EOF,
  emit the accumulator, then pass EOF through.
- **scan**: on regular value, update accumulator and emit it. On EOF, pass through.
- **group_by reduce**: on EOF, emit one value per group, then pass EOF through.
- No `Apply()`/`Flush()` needed. EOF is just another value flowing through.

### 4.5 Handler simplification ✅
- Handlers publish raw `*expr.Value` to `nodes.{id}.transform.input` (if
  transforms exist) or wrap in `RunSnapshot_NodeEvent` and publish to `nodes.{id}`
  directly (if no transforms)
- Removed `transformPipeline.Apply()`, `Flush()`, `accumulateState.seen`
- Removed all `Apply()`/`Flush()` call sites from var, switch, range, ticker,
  output handlers
- `compileTransforms()` returns a pipeline that manages its own goroutine
  lifecycle via errgroup

### 4.6 Flat Typed Node Messages + CEL Native Types ✅
Replace `RunSnapshot_Node` (single message with `oneof type`) with flat typed
node messages. CEL proto types are registered via `cel.Types()` for native
field access. The `value` field (an `*expr.Value` serialization wrapper) is
unwrapped to native `ref.Val` in `nodeToMap()` -- a simplified replacement
for the old `nodeToState()` type-switch.

**Proto changes:**
- Delete `RunSnapshot_Node` with its nested `Input`, `Var`, `Generator`, etc. sub-messages
- Add flat top-level messages: `RunSnapshot_InputNode`, `RunSnapshot_VarNode`,
  `RunSnapshot_GeneratorNode`, `RunSnapshot_ActionNode`, `RunSnapshot_StreamNode`,
  `RunSnapshot_OutputNode`, `RunSnapshot_InteractionNode` -- each with common fields
  (`id`, `value`, `error`, `event_time`) + type-specific fields promoted to
  the top level
- Replace `NodeEvent.node: Node` field with `oneof node { InputNode input; VarNode var; ... }`
- Update `RunSnapshot` snapshot maps to typed: `map<string, InputNode> inputs`, etc.
- Run `task generate`

**Runtime changes:**
- Define `StateNode` Go interface (`GetId`, `GetValue`, `proto.Message`)
- Replace `publishNode(pub, topic, *RunSnapshot_Node)` with type-specific helpers
  or a generic `publishEvent(pub, topic, *RunSnapshot_NodeEvent)` accepting the interface
- Delete `nodeToState()` and the `map[string]any` activation shim in `cel.go`
- Add `TypeRegistry` (`runtime/registry.go` or similar): holds `cel.EnvOption`s
  for all known proto types. Built-in flat node types are registered at init.
- Update `cel.go` `compileCEL()`: accept `TypeRegistry`, use its `cel.EnvOption`s
  to register flat types and declare typed map variables instead of `cel.DynType`
- Update `activation.Resolve()`: pass typed flat nodes directly into the CEL
  variable map under their namespace (`inputs`, `vars`, etc.)
- Update `nodeRef.recv()`: payload becomes `*RunSnapshot_NodeEvent`; extract the
  typed inner node from the `oneof`
- Update all handler `Run()` methods and wrapper functions to build the
  appropriate typed flat node instead of `*RunSnapshot_Node`
- Update outbox memory/sqlite: serialize `*RunSnapshot_NodeEvent` wrapper;
  `SnapshotAt` builds typed maps from NodeEvent oneof

---

## Phase 5 — Test Coverage Gaps (partial) ✅

### 5.1 Throttle tests ✅
- `TestGraph_Action_Throttle` -- action throttle (graph_actions_test.go)
- `TestGraph_Stream_BidiThrottle` -- bidi stream throttle (graph_streams_test.go)
- `TestGraph_Output_Throttle` -- output throttle (graph_outputs_test.go)
- `TestGraph_Generator_RangeWithRate` -- range rate (graph_generators_test.go)
- `TestGraph_Input_Cache` -- input cache with throttle (graph_inputs_resolution_test.go)
- `TestGraph_Input_TypeDefault` -- type default on throttle timeout
- `TestGraph_Input_CachePriorityOverDefault` -- cache beats default
- `TestGraph_Input_DefaultThrottleInjection` -- default throttle injection
- `TestThrottle_RateLimits` / `TestThrottle_ZeroInterval` -- pubsub middleware

### 5.2 Error path tests ✅
- `TestGraph_Error_VarEval` -- CEL division by zero at runtime
- `TestGraph_Action_MissingMethod` -- action references missing method
- `TestGraph_Action_NoMethods` -- action node with no Methods provided
- `TestGraph_Error_ContextCancellation` -- cancel context during ticker graph

### 5.4 Gochannel backend tests ✅
- `TestNackRedelivery` -- nack → redelivery
- `TestAckBeforeNext` -- ack-before-next-message contract
- `TestFanOut_UUIDPreserved`, `TestFanOut_IndependentAck` -- fan-out + UUID
- `TestContextCancellation` -- context cancellation during subscribe

### 5.5 Transform pipeline edge cases ✅
- `TestGraph_Transform_Flatten_ScalarPassthrough` -- scalar through flatten
- `TestGraph_Generator_Ticker_WithMapTransform` -- ticker + map
- `TestGraph_Generator_Ticker_WithFilterTransform` -- ticker + filter
- `TestGraph_Generator_Ticker_WithReduceTransform` -- ticker + reduce
- `TestGraph_Reduce_GroupBy_FixedWindow` -- fixed window grouping
- `TestGraph_Reduce_MultiValueNoWindow` -- bare reduce emits once on EOF
- `TestGraph_Transform_Map_Int64ToString` -- cross-type mapping
- `TestGraph_Transform_Map_EvalError` -- runtime eval error in map
- `TestGraph_Transform_ContextCancel` -- cancel mid-pipeline
- `TestGraph_Transform_InputThenVar_ChainedReduces` -- chained reduces

---

## Phase 6 — Runtime Feature Gaps (partial) ✅

### 6.0 Input Resolution Chain ✅
- **6.0.1 `constant`** -- close input after first value, downstream evaluates once
- **6.0.2 `throttle` wiring** -- Rate → timeout window, inject default when
  cache/default set without explicit throttle
- **6.0.3 `cache`** -- store last value, replay on throttle timeout
- **6.0.4 Type-level `default`** -- fallback when no cache, throttle expires
- **6.0.5 Tests** -- constant, cache, default, cache+default priority, injection
- **6.0.6 Action/Stream/Output `throttle`** -- subscribe-side rate limiting
- **6.0.7 Range `rate`** -- time.Ticker pacing for Range emission

### 6.1 `when` / `close_request_when` ✅
CEL conditions on Actions and Streams. `close_request_when` calls `CloseSend()`
on streams. Both `close_request_when` and EOF-in-request supported as ergonomic
alternatives.

### 6.2 Orphaned node warnings ✅
`Lint()` flags nodes without outgoing edges that aren't side-effect nodes
(actions/streams/outputs/interactions/connections).

### 6.3 Cron generator ✅
`Generator_Cron_` handler using `robfig/cron/v3`. Sleeps until
`cron.Next(time.Now())`, publishes NodeEvent with count and event_time.

### 6.4 Connection nodes + rpc.Client + per-connection routing ✅
- Connection nodes: skipped by `newNodeHandler()` (graph-level metadata)
- `rpc.Client` interface: `CallUnary`, `CallBidiStream`, `CallClientStream`,
  `CallServerStream` using `proto.Message` and `protoreflect.FullName`
- `rpc.Connector` struct: pairs `Client` + `common.Resolver` per connection
- `common.Resolver` interface: `FindMethodByName(protoreflect.FullName)`
- `WithConnectors(map[string]*rpc.Connector)` executor option: keyed by
  connection ID from the flow spec's `MethodCall.connection` field
- Action/stream handlers extract `call.GetConnection()` to look up the correct
  connector -- no mux/multiplexing layer needed
- `mock.Client` in `rpc/mock/mock.go` for tests (implements both `rpc.Client`
  and `common.Resolver`)

### 6.6 Interaction handler ✅
`interactionHandler` with prompt/response channels, `promptAndWait()` for
external communication, EOF propagation. Tests: Basic, MultiplePrompts,
MissingOption, ResponseClose.

### 6.7 Inputs and Outputs via PubSub ✅
Replaced `chan *InputEvent` / `chan *OutputEvent` with PubSub topics. Input
nodes subscribe to PubSub topics; output nodes publish to PubSub topics.
Callers use PubSub directly. Input resolution chain preserved.

### 6.9 CEL `EOF()` function and generator `this.count` / `this.time` ✅
`EOF()` registered as custom CEL function. Generator count scoped to
`this.count`, tick timestamp to `this.time`.

---

## Phase 7 — Cross-Cutting Features ✅

### 7.1 `memoize` support (was 6.8) ✅
Implemented with pluggable `Cache` interface (`cache/cache.go`) and in-memory
backend (`cache/memory/memory.go`). Unary handler checks cache before RPC,
stores on miss. `WithCache` executor option for shared cache; per-handler
`MemoryCache` fallback when `memoize: true` and no shared cache.

### 7.2 Transform state persistence via NodeEvent STATE events (was 4.4) ✅
Already fully implemented: `publishStateEvent()`, `newStateCallback()`,
`runStep()` onState callback, all handler wiring in place.

### 7.3 Valkey Streams PubSub backend ✅
Renamed `pubsub/gochannel` to `pubsub/memory`. Cloud PubSub backends
implemented in `cloud/pubsub/valkey/` (Valkey Streams via Valkey Glide Go) and
`cloud/pubsub/kafka/` (Kafka via franz-go). PubSub conformance test suite
(`pubsub/pubsubtest/`) wired into all three backends.

Also completed: Valkey cache backend (`cloud/cache/valkey/`), cache conformance
suite (`cache/cachetest/`), outbox conformance suite (`outbox/outboxtest/`).
All cloud backends live under the `cloud/` package tree.

### 7.4 `MethodCall.request` / `response` field handling (was 6.5) ✅
Implemented two-phase node processing (parse → compile). `parseCEL()` / `compileCEL()`
split in `runtime/cel.go`. Request tree infrastructure in `runtime/request.go`
(`compileRequestTree`, `evalRequest`, `resolveRequestValue`, `transformResponse`).
All four handler structs (`unaryHandler`, `serverStreamHandler`, `clientStreamHandler`,
`bidiStreamHandler`) have `request *compiledRequest` and `responseProg cel.Program`
fields. Lint uses `parseCEL()` for AST-only validation. Handlers fall back to
`act.FirstInputValue()` when no request tree is compiled.

---

## Phase 8 — CLI Integration (dtkt-cli, partial) ✅

> Proto definitions moved to SDK at `proto/dtkt/flow/v1beta2/`. The CLI
> integrates the executor via ent schemas and the entstore outbox backend.
> Remaining CLI work tracked in `docs/plan.md`.

### 8.1 Ent schemas ✅
`FlowRun`, `RunSnapshot`, `FlowEvent` schemas in `dtkt-cli/internal/db/schema/`.
Codegen complete -- all ent types generated (FlowRunCreate, FlowEventCreate,
RunSnapshotCreate, etc.). Migration table definitions included in `migrate/schema.go`.

### 8.2 Flow schema edge ✅
`Flow` updated with `flow_runs` edge (cascade delete via `entsql.OnDelete`).

### 8.3 Proto imports via entadapter ✅
Schemas use `entadapter.MessageField[flowv1beta2.RunSnapshot]` and
`entadapter.MessageField[flowv1beta2.RunSnapshot_NodeEvent]` from SDK. Proto
fields stored as JSON via protojson (no manual marshal/unmarshal).

### 8.4 cmd/flow2 scaffolding ✅
Scaffolded from `cmd/flow` (15 files, `package flow2`). Registered in
`cmd/root.go` via `flow2.Cmd`. Subcommands: create, get, list, update, delete,
lint, run. Package renamed but subcommands still reference Automation APIs.

### 8.5 Entstore ✅
`dtkt-cli/internal/core/flowv1beta2/outbox/entstore/entstore.go` implements
`outbox.Outbox` using ent. Uses typed `NodeEvent` field via entadapter and
ent-generated predicates (`flowevent.ForwardedEQ`, `flowevent.IDGT`, etc.).
UUIDv7 for sequence ordering. Compiles clean.

### 8.6 cmd/flowrun: start + stop ✅
`cmd/flowrun/` package with `start` and `stop` subcommands. Registered in
`cmd/root.go` via `flowrun.Cmd`. Aliases: `flowrun`, `flowruns`.

**`flowrun start NAME|PATH|URI`:** Creates a FlowRun via `CreateFlowRun` RPC,
polls until PENDING, then opens `StreamFlowRunEvents` bidi stream and sends
`StartFlowEvent`. Supports `--conns` (connection mapping), `--inputs` (initial
input values as JSON), `--timeout` (execution timeout), `--id` (custom run ID),
`--wait-ready` (poll timeout). Two paths: with-inputs (send loop + receive loop
in errgroup) and no-inputs (send start + close request + receive-only loop).
Event dispatch via `handleReceiveEvent()` handles OutputEvent (resolve + encode),
terminal FlowRun state events, and InputRequest/InteractionRequest (TODO stubs).

**`flowrun stop NAME`:** Resolves flowrun by name via `GetFlowRun`, sends
`UpdateFlowRun` with desired state STOPPING, polls until terminal state.

**`flowrun.go`:** Parent command with `Use: "flowrun"`, `Long: "Start and stop
flow runs."`. Prints help when no subcommands given.

**`helper.go`:** Empty package file (placeholder for future shared helpers).

**Backend:** `internal/core/flowrun/handler.go` implements the full CRUD surface:
`CreateFlowRun`, `GetFlowRun`, `ListFlowRuns`, `UpdateFlowRun`,
`DeleteFlowRun`, `SendFlowRunEvent`, `ReceiveFlowRunEvents`,
`StreamFlowRunEvents`. Active runs tracked in `h.runs` SyncMap. Operations
(Boot, Create, Update, Delete) manage state transitions. Handler wired in
`internal/core/server.go` via `flowrun.NewHandler(env)`.

**Remaining:** `list`, `get`, `create`, `delete`, `attach` commands -- see
`docs/plan.md` §9.13.

---

## Phase 9 -- Error Handling & Flow Lifecycle (§1, partial) ✅

### 9.1 Node phase tracking (§1.1) ✅
Enriched all node state publications with `Phase` field. All handler
constructors converted to builder pattern for consistent phase tracking.
Phases: PENDING → RUNNING → SUCCEEDED (terminal phases for error/cancel
added in later sub-phases).

Tests: 7 phase-tracking tests in `executor_phases_test.go` covering
output, var, action, stream, generator (range + ticker), and interaction
nodes. All verify PENDING → RUNNING → SUCCEEDED transitions.

`flow.started` event deferred to §1.3 (flow lifecycle events).

### 9.2 Outbox transparency testing ✅
All executor integration tests (~146 across 15 files) now run both with and
without outbox via `withAndWithoutOutbox` helper in `executor_helpers_test.go`.
Surfaced and fixed two outbox bugs:

**Memory outbox `Read` cursor fix** (`outbox/memory/memory.go`): Changed from
UUID byte comparison (`bytes.Compare`) to position-based cursor. Concurrent
transactions can commit records out of UUID order (UUIDv7 is time-based, but
commit order may differ). The byte-comparison cursor would skip records with
lower UUIDs committed after the cursor advanced. Position-based cursor finds
`afterUID` by exact match in the records slice and scans from the next index.

**Forwarder context decoupling** (`runtime/executor.go`): The forwarder
previously ran on the caller's `ctx`. When context was cancelled (test timeout,
graceful stop), `fwd.Run(ctx)` returned `ctx.Err()` which propagated as an
Execute error. Changed to use a separate `context.Background()`-derived context
(`fwdCtx`). Lifetime controlled by `CloseWhenDrained` (normal shutdown) or
deferred `fwdCancel` (early exit). Forwarder now drains cleanly regardless of
caller context state.

### 9.3 Test file rename ✅
Renamed `graph_*_test.go` → `executor_*_test.go` across all 17 test files
in `runtime/`. Tests exercise the Executor, not the graph data structure.
The `graph/` package has its own `graph_test.go` for DAG-level tests.

### 9.4 Retry loop + `this.error` CEL binding (§1.2) ✅
Implemented retry strategy support for Action and Stream nodes:

**New file: `runtime/retry.go`** -- Core retry logic:
- `compiledRetryStrategy`: compiled CEL programs for when/skip_when/suspend_when/terminate_when + backoff config
- `compileRetryStrategy()`: compiles RetryStrategy proto into executable programs
- `executeWithRetry()`: retry loop with escalation-first evaluation (skip/suspend/terminate checked on every error before backoff)
- `backoffDelay()`: exponential backoff with configurable multiplier and max delay
- `errSkipped` sentinel: signals skip_when matched (handler continues to next item)
- `SuspendError`/`TerminateError` types: signal flow-level actions
- `lintRetryStrategy()`: validates all 4 CEL expressions without executing
- `buildRetryVars()`: augments CEL activation with `this.error.code`/`this.error.message` map
- `grpcStatusProto()`: converts Go errors to google.rpc.Status proto

**Handler modifications:**
- All 4 handler types (unary, server-stream, client-stream, bidi-stream) gain `retry *compiledRetryStrategy` field
- `handlers.go`: compiles RetryStrategy from Action/Stream proto and passes to handlers
- RPC error paths wrapped with `executeWithRetry()` call
- `errSkipped` check: handlers `continue` to next input on skip

**Linting:**
- `lint.go`: Action and Stream cases call `lintRetryStrategy()` on their RetryStrategy

**Tests (5 new, all with/without outbox):**
- `TestGraph_Action_RetryBackoff_Success`: UnavailableThenOK retries and succeeds
- `TestGraph_Action_RetrySkip`: NotFound (code 5) triggers skip_when, no output
- `TestGraph_Action_RetryTerminate`: Internal (code 13) triggers terminate_when
- `TestGraph_Action_RetryExhausted`: retries exhausted, error propagated
- `TestLint_InvalidRetryStrategyCEL`: invalid CEL in skip_when detected by linter

### 9.5 ErrorStrategy TERMINATE + STOP (§1.3) ✅

**New files:**
- `runtime/lifecycle.go`: `publishTerminalPhase()` publishes PHASE_ERRORED
  (or other terminal) with EOF value and `google.rpc.Status` error for any of
  the 7 node types. `isGenerator()` helper for context routing.
- `runtime/executor_lifecycle_test.go`: 7 tests (14 with outbox)
- 4 YAML fixtures: `action_error_internal.yaml`, `stop_multi_path.yaml`,
  `stop_generator_action.yaml`, `stop_terminate_error.yaml`

**Modified files:**
- `runtime/executor.go`:
  - Added `errorStrategy` field + `WithErrorStrategy(ErrorStrategy)` option
  - `Execute()`: error interception wrapper around handler goroutines
  - TERMINATE (default): publish PHASE_ERRORED → propagate error to errgroup
  - STOP: publish PHASE_ERRORED → `signalStop` (inject EOFs to inputs, cancel
    genCtx for generators) → return nil to errgroup → pipeline drains
  - TerminateError always forces TERMINATE regardless of strategy
  - Context-cancelled handlers return nil (swallowed, not treated as node errors)
  - `genCtx`: separate context for generators, child of gCtx. Cancelled by
    signalStop (STOP) or by errgroup cancel (TERMINATE, cascading).
  - `nodeProtos` map: stores `*Node` per handler for phase publishing dispatch
  - `inputNodeIDs`: collected upfront for STOP EOF injection
- `runtime/range.go`: Fixed to publish EOF on ctx.Done() (like ticker/cron).
  Previously returned `ctx.Err()` without publishing EOF, which blocked
  downstream handlers during STOP drain.
- `runtime/executor_phases_test.go`:
  - `collectPhases` now terminates on ERRORED/FAILED/CANCELLED (not just SUCCEEDED)
  - `phaseNames` extended with ERRORED/FAILED/CANCELLED labels

**Tests (7 new, all with/without outbox):**
- `TestErrorStrategy_Terminate_ActionError`: action error terminates, PHASE_ERRORED
- `TestErrorStrategy_Terminate_RetryTerminateError`: TerminateError terminates
- `TestErrorStrategy_Stop_ActionError`: STOP returns error after drain
- `TestErrorStrategy_Stop_MultiPath`: healthy path produces output despite error
- `TestErrorStrategy_Stop_Generator`: generator stops gracefully on STOP
- `TestErrorStrategy_Stop_TerminateErrorOverrides`: TerminateError bypasses STOP
- `TestErrorStrategy_Stop_NodePhase`: errored node is PHASE_ERRORED in STOP

### 9.6 Stop/Terminate commands (§1.4) ✅
- `Stop()`: graceful drain (EOF injection, generator context cancel), Execute returns nil.
- `Terminate()`: immediate cancel, Execute returns `ErrTerminated`.
- `StopNode(nodeID)` / `TerminateNode(nodeID)`: per-node control.
- Per-node contexts stored in `nodeCtxs` map. Run-state mutex-protected.
- 14 tests (28 with outbox).

### 9.7 Suspend/Resume (§1.5) ✅
- `Suspend()` / `SuspendNode(nodeID)`: park handler goroutines.
- `Resume()` / `ResumeNode(nodeID, val)`: resume suspended nodes.
- Phase transitions: RUNNING → SUSPENDED → PENDING → RUNNING.
- Generators restart from beginning on resume.
- 14 tests (28 with outbox) in `executor_suspend_test.go`.

### 9.8 CONTINUE strategy (§1.6) ✅
- Errored node publishes PHASE_ERRORED + EOF; dependents drain.
- Independent paths continue running. TerminateError overrides.
- 6 tests (12 with outbox) in `executor_lifecycle_test.go`.

### 9.9 `continue_when` on RetryStrategy (§1.7) ✅
- `ContinueError` sentinel carries CEL-derived value.
- Evaluated before skip_when/suspend_when/terminate_when.
- 4 tests (8 with outbox) in `executor_retry_test.go`.

---

## Phase 10 -- Migrate to Shared SDK Infrastructure (§1.8) ✅

### 10.1 CEL env: use `common.NewCELEnv` (§1.8.1) ✅
`buildCELEnv()`, `buildLintCELEnv()`, `parseCEL()` use `common.NewCELEnv(opts...)`.
Standard extension set added (URLs, Encoders, Strings, Lists, Protos, CELEnumExt).

### 10.2 `refValToExpr` → `cel.ValueAsProto` (§1.8.2) ✅
`refValToExpr` rewritten to delegate to `cel.ValueAsProto(v)`. Wrapper function
retained for call-site convenience.

### 10.3 `exprToNative` → `shared.ExprValueToNative` (§1.8.3) ✅
`exprToNative` deleted. `exprToMessage` takes `shared.Env` and uses
`shared.ExprValueToNative(env, input)`. Handler types store `shared.Env`.

### 10.4 `isCELExpression` → `shared.IsValidExpr` (§1.8.4) ✅
Replaced `isCELExpression(s)` and `stripCELPrefix(s)` with `shared.IsValidExpr(s)`.

Files changed (§1.8.1-§1.8.4):
- `runtime/cel.go`, `runtime/value.go`, `runtime/unary.go`,
  `runtime/server_stream.go`, `runtime/client_stream.go`,
  `runtime/bidi_stream.go`, `runtime/lint.go`, `runtime/handlers.go`.

### 10.5 `exprToRefVal` → `cel.ProtoAsValue` (§1.8.5) ✅
`exprToRefVal` rewritten to delegate to `cel.ProtoAsValue(adapter, v)`. Adapter
threaded via `types.Adapter` field on handler structs (`rangeHandler`,
`switchHandler`, `cronHandler`, `tickerHandler`, `accumulateState`,
`transformStep`, `activation`). All call sites pass the env adapter.

### 10.6 `nativeToExpr` test-only rewrite (§1.8.6) ✅
`nativeToExpr` was already test-only (`executor_helpers_test.go`). Rewritten
from manual type switch to `types.DefaultTypeAdapter.NativeToValue(v)` →
`cel.ValueAsProto(refVal)`.

### 10.7 `copyMap` → `maps.Clone` (§1.8.7) ✅
`copyMap` deleted from `switch.go`. Single call site inlined with `maps.Clone`.

### 10.8 Proto conflict lint detection (§1.8.8) ✅
Added `CodeProtoConflict` lint code. `lintProtoConflicts()` walks resolvers by
connection ID, collects message FQNs via `collectMessages()`, warns on same FQN
from different file paths. Tests: `TestLint_ProtoConflict`,
`TestLint_ProtoNoConflictSameFile`.

---

## Phase 11 -- CLI Integration §9.1-§9.7 ✅

### 11.1 Fix v1beta2 graph in CreateFlow/UpdateFlow (§9.1) ✅
Added `FlowSpecMetadata_V1Beta2` case in both `CreateFlow` and `UpdateFlow` in
`dtkt-cli/internal/core/flow/handler.go`. Calls `graph.Build(flow)` from v1beta2
graph package, wraps result in `FlowGraphMetadata_V1Beta2`.

### 11.2 Register FlowRunService handler on server (§9.2) ✅
`dtkt-cli/internal/core/server.go`: `flowrun.NewHandler(env)`, assigned to
`env.FlowRuns.FlowRunServiceHandler`, registered with `server.WithFlowRunHandler(...)`.

### 11.3 Register v1beta2 schema type (§9.3) ✅
- SDK: `spec.go` added `SpecOptions` struct + `ExtendSchemaWithContext()`.
- CLI: `adapter.go` `syncFlowSpecV1Beta2()` passes `WithJSONSchemaCallback`.

### 11.4 cmd/flow2 CRUD commands (§9.4) ✅
`create.go`, `update.go`, `lint.go`, `spec_util.go` updated to v1beta2 SDK.
`get.go`, `list.go`, `delete.go` are version-agnostic.

### 11.5 cmd/flow2 run/io commands (§9.5) ✅
`run.go` and `io.go` migrated to v1beta2 FlowRun path (`CreateFlowRun`,
`StreamFlowRunEvents`, `SendFlowRunEvent`, bidi streaming).

### 11.6 Lint integration (§9.6) ✅
`lintMethodCall()` and `lintNodeConnection()` validate connections and method
calls. Undeclared connections produce warnings (not errors). Tests:
`TestLint_ValidConnection`, `TestLint_UndeclaredConnection`.

### 11.7 Flow execution wiring (§9.7) ✅
Executor wired in `flowrun/run.go`: `WithConnectors`, `WithInteractions`,
`runInteractionPrompts`, `dispatchEvent` (handles all event variants).

### 11.8 LRU event cache configurability (§9.7.1) ✅
`RunOption`/`WithEventCacheSize` on `NewRun`, `HandlerOption`/
`WithHandlerEventCacheSize` on `NewHandler`. Default 100 (`DefaultEventCacheSize`).

---

## Phase 12 -- Runtime Code Quality Refactoring (Phase 10, partial) ✅

### 12.1 Kill dead marker interfaces (10.1) ✅
7 marker interfaces deleted from `executor/nodes.go`. Compile-time assertions
removed from `handlers.go`. `NodeHandler` and `StateNode` retained.

### 12.2 Introduce compile phase (10.4) ✅
`runtime/compile.go`: concrete compiled structs per node type (`compiledVarValue`,
`compiledVarSwitch`, `compiledTicker`, `compiledRange`, `compiledCron`,
`compiledCall`, `compiledOutput`). `compileNode()` function extracts all CEL
compilation. `newNodeHandler` is a simpler factory over compiled structs.

### 12.3 Full AST edge inference (10.6) ✅
Replaced regex-based `extractRefs` + `nodeRefPattern` with CEL AST walking
(`ast.PreOrderVisit(ast.NavigateAST(...))`) in `graph/graph.go`. Minimal CEL
env built inside `graph.Build()` for reference extraction.

### 12.4 Reduce state publishing boilerplate (10.7) ✅
`nodeFactoryMap()` returns `map[any]nodeFactory` keyed by `Node.WhichType()`.
`publishTerminalPhase` and `publishPhaseChange` are 4-line functions doing map
lookup. No switch statements remain.

---

## Phase 13 -- CLI Integration: Entstore Tests (1.1) ✅

### 13.1 Outbox conformance suite wiring ✅
Wired `outbox/outboxtest` conformance suite against CLI's entstore + SQLite in
`internal/core/flowv1beta2/outbox/entstore/entstore_test.go`. Added `EventReader`
option to `outboxtest.Options` enabling `SnapshotAt` and `SnapshotAtEmpty` tests.

### 13.2 SnapshotAt on entstore ✅
Implemented `SnapshotAt(ctx, afterUID)` on entstore. Shortcut path when UID
matches the latest FlowState (returns materialized snapshot). Full path replays
FlowEvents linked to the FlowState, plus handles conformance test path where
events are stored via direct `Store()` without FlowState linkage.

### 13.3 ReadEvents on entstore ✅
Implemented `ReadEvents(ctx, afterUID, limit)` on entstore. Queries FlowEvents
after the given UID ordered by creation time. Added compile-time assertion for
`outbox.EventReader` interface.

### 13.4 Ent-specific tests ✅
Added 4 ent-specific tests:
- `TestFlowEvent_QueryByTopic` -- FlowEvent topic predicate filtering
- `TestFlowEvent_QueryByForwarded` -- FlowEvent forwarded status filtering
- `TestSnapshotAt_WithFlowState` -- SnapshotAt with FlowState shortcut path
- `TestSnapshotAt_RollbackDoesNotAffectState` -- transaction isolation

All 20 conformance + ent-specific tests pass.

---

## Phase 14 -- Cursor-Based Event Replay (6.2, partial) ✅

### 14.1 Proto: event_id on response messages ✅
Added `string event_id = 2` to `ReceiveFlowRunEventsResponse` and
`StreamFlowRunEventsResponse` proto messages. Regenerated Go code via
`task generate`.

### 14.2 CachedEvent with UUIDv7 ✅
Created `CachedEvent` struct (`ID uuid.UUID` + `Event *ReceiveFlowRunEvent`).
`cacheEvent()` assigns `uuid.Must(uuid.NewV7())` to each event. Updated
`EventReceiver` type to `func(*CachedEvent) error`.

### 14.3 Handler wiring for after_event_id ✅
Updated `ReceiveFlowRunEvents` and `StreamFlowRunEvents` handlers to parse
`after_event_id` from requests, pass to `ReceiveEvents()`, and set `event_id`
on responses. Added `parseAfterEventID()` helper.

### 14.4 EventReader interface consolidation ✅
Merged `ReadEvents()` into `outbox.EventReader` (alongside `SnapshotAt()`).
Removed separate `EventLog` interface. Implemented `ReadEvents()` on both
entstore and memory outbox backends.

### 14.5 EventBuffer ring buffer (replaces LRU) ✅
Replaced `hashicorp/golang-lru` with purpose-built `EventBuffer` ring buffer
(`flowrun/eventbuf.go`). Evicts oldest-first (not least-recently-used).
Cursor-based iteration via `After(uuid.UUID)`. Removed `recvCacheIdx
atomic.Int64` -- UUIDv7 ordering replaces sequential int indices. Simplified
`ReceiveEvents()` from index-scanning to `drain(cursor)` calling
`recvBuf.After(cursor)`.

### 14.6 FlowEvent ent schema refactor ✅
Refactored FlowEvent ent schema: removed `topic`, `metadata`, `forwarded`
columns. Added `event_type` (OUTPUT/UPDATE) and `node_type`
(input/generator/var/action/stream/output/interaction) denormalized columns
with indexes for efficient DB filtering. Updated entstore `Store()` to
populate the new columns.

---

## Phase 15 -- CLI Integration: Signal Handling & Keybinds (1.5) ✅

Replaced `signal.NotifyContext` with manual signal channel + escalation counter.

**Signals (interactive + non-interactive):**

| Signal | Action | Auto-escalation |
|--------|--------|-----------------|
| 1st Ctrl+C | Send `StopFlowEvent`, show "Stopping..." | → Terminate after `--stop-timeout` (default 30s) |
| 2nd Ctrl+C | Send `TerminateFlowEvent`, show "Terminating..." | → `os.Exit(1)` after `--terminate-timeout` (default 10s) |
| 3rd Ctrl+C | `os.Exit(1)` immediately | -- |
| Ctrl+\\ (SIGQUIT) | Dump diagnostic info + `os.Exit(1)` | -- |

**TUI keybinds (interactive mode):**

| Key | Action |
|-----|--------|
| `d` / `q` | Detach (close stream, leave flow running) |
| `s` | Send StopFlowEvent (graceful drain) |
| `t` | Send TerminateFlowEvent (hard cancel) |
| `p` | Send SuspendFlowEvent (pause) |
| `r` | Send ResumeFlowEvent (resume) |
| Ctrl+C | Escalating stop→terminate→exit |

Keybinds and signal handler share dispatch methods: `io.RequestStop(stream)`,
`io.RequestTerminate(stream)`, `io.Detach(stream)`, `io.RequestSuspend(stream)`,
`io.RequestResume(stream)`.

**Detach:** Close stream without Stop/Terminate. Flow continues running. Print
flowrun name for re-attach.

**Files:** `internal/flowio/io.go`, `internal/flowio/model.go`,
`internal/flowio/control.go`, `internal/flowio/options.go`, `cmd/flow2/run.go`.

---

## Phase 16 -- CLI Integration: Flowrun CRUD Commands (1.6) ✅

Added full CLI surface for flowrun management on top of the existing backend
handler (`internal/core/flowrun/handler.go`).

**Commands implemented:**
- `dtkt flowrun list` -- `ListFlowRuns` RPC with `--state` and `--flow` filters,
  pagination via `cli.ListOptions`.
- `dtkt flowrun get NAME` -- `GetFlowRun` RPC, single flowrun detail view.
- `dtkt flowrun create NAME --flow FLOW` -- `CreateFlowRun` RPC, PENDING state,
  does not start. Supports `--conns`, `--inputs`, `--timeout`, `--id`.
- `dtkt flowrun delete NAME` -- `DeleteFlowRun` RPC, stops active runs first.
- `dtkt flowrun attach NAME` -- `StreamFlowRunEvents` bidi stream against
  existing running flowrun. Reuses `RunIO` for TUI + keybinds. Does NOT send
  `StartFlowEvent`. Snapshot-based interaction discovery for pending prompts.
- `dtkt flow2 run --detach` -- fire-and-forget: create + start + close stream.

**Interaction notification on attach:** On attach, call `GetFlowRun` to fetch
`RunSnapshot`, inspect `InteractionNode.token != ""` for pending interactions,
synthesize `InteractionRequestEvent`s. Executor's `TryDeliver` validates tokens
atomically so stale tokens are safely dropped.

**Historical event replay:** Proto supports `after_event_id` (UUIDv7 cursor).
`EventBuffer` ring buffer with oldest-first eviction. DB fallback pending (§6.2).

**Files:** `cmd/flowrun/list.go`, `cmd/flowrun/get.go`, `cmd/flowrun/create.go`,
`cmd/flowrun/delete.go`, `cmd/flowrun/attach.go`, `cmd/flowrun/flowrun.go`,
`cmd/flow2/run.go`.

---

## Phase 17 -- Runtime: Decompose Execute() (2.1) ✅

`Execute()` was ~430 lines of inline logic. Extracted to ~120-line orchestrator.

**Extracted methods (in `executor_setup.go`):**
- `wireEdges` (pubsub wiring)
- `resolveConnections` (connection resolution)
- `buildCELEnvAndValidate` (CEL environment + request schema validation)
- `buildHandlers` (handler construction loop)
- `buildInteractionHandlers` (interaction handler construction)
- `setupInputBridges` (input bridge goroutines)
- `startInteractionDemux` (interaction response demux goroutine)
- `setupRunState` (stop/suspend/resume contexts, per-node contexts, flow_control wiring)
- `launchHandlers` (suspend/resume/error interception launch loop)

Remaining inline code is outbox/forwarder setup (with scoped defers) and
post-wait error aggregation, inherently tied to Execute()'s scope.

---

## SDK Runtime Hardening ✅

- **Interface assertions:** Added compile-time assertions for `switchHandler`
  and `inputHandler` implementing `executor.NodeHandler` in `handlers.go`.
- **outboxPubSub.Close():** Changed from discarding `pub.Close()` error to
  `errors.Join(o.pub.Close(), o.sub.Close())` in `runtime/outbox.go`.
- **Phase-change publish error logging:** Replaced 4 `_ = publishPhaseChange/
  publishTerminalPhase` calls in `executor_setup.go` with `slog.Error` logging.
- **Cache TTL/eviction:** Added `SetWithTTL(ctx, key, value, ttl)` to `Cache`
  interface. Rewrote `memory.Cache` as bounded LRU with TTL support
  (`DefaultMaxEntries=10000`, `WithMaxEntries()` option, lazy expiry on access).
  Updated Valkey backend to implement `SetWithTTL` via `Expire` command.

---

## Cursor-based event replay ✅

- **Buffer stores FlowEvents directly:** Replaced
  `*corev1.ReceiveFlowRunEventsResponse` in `EventBuffer` with
  `*flowv1beta2.RunSnapshot_FlowEvent`. Added `FlowEventID(fe)` and
  `setFlowEventID(fe, id)` helpers that extract/set `event_id` from
  whichever oneof variant is set. `cacheEvent()` now stores raw
  `FlowEvent` and assigns `event_id` on the contained node message.
  Files: `internal/core/flowrun/eventbuf.go`, `internal/core/flowrun/run.go`.

- **Wire DB-backed replay fallback:** In `ReceiveEvents`, when
  `afterEventID` is not found in the in-memory buffer and `eventLog` is
  available, the method calls `eventLog.ReadEvents()` to replay from the
  persistent event log, then transitions to live buffer events.
  File: `internal/core/flowrun/run.go`.

- **Attach: pending interaction discovery:** Already handled by the
  event buffer + DB replay infrastructure -- when attaching, `ReceiveEvents`
  replays all buffered/persisted events including pending interaction
  requests, so the TUI discovers them automatically.

- **Tests:** 23 unit tests covering `FlowEventID`/`setFlowEventID`,
  `EventBuffer` operations (push, cursor lookup, wrap-around, eviction,
  concurrent access).
  File: `internal/core/flowrun/eventbuf_test.go`.

---

## Interaction form rendering ✅

- **SDK GetInteractionBinding:** Created
  `flowsdk/v1beta2/interaction.go` with `GetInteractionBinding(input)` that
  switches on element type (Confirm, Input, File, Select, MultiSelect),
  creates the appropriate binding proto, and returns a `FieldElement` +
  binding message for protoform rendering.

- **CLI HandleInteraction:** Rewrote `HandleInteraction` in
  `internal/flowio/io.go` to look up interaction spec by ID, iterate
  inputs, call `GetInteractionBinding`, create `form.NewMessage` per
  binding, compose fields via `form.NewFieldGroup`, render TUI form via
  `tuiform.NewGroup`, and wrap completed values as `*anypb.Any` for the
  response. Single-input sends value directly; multi-input wraps in
  `structpb.Struct` keyed by input ID.

---

## Control event error handling ✅

- **Propagate send errors:** Changed `sendControlEvent` to return `error`
  instead of only logging. Updated `dispatchControl` to check returned
  errors and send `controlStatusMsg{err: err}` to the TUI model.
  Added `controlErr` field to the model; the status bar now displays
  errors in red when a control command fails.
  Files: `internal/flowio/control.go`, `internal/flowio/model.go`.

- **Make stop/terminate timeouts configurable:** Added `--stop-timeout`
  and `--terminate-timeout` flags to `cmd/flowrun/attach.go` (matching
  the flags already on `cmd/flowrun/start.go`). Values plumb through
  `RunOptions.StopTimeout` / `RunOptions.TerminateTimeout`.
  File: `cmd/flowrun/attach.go`.
