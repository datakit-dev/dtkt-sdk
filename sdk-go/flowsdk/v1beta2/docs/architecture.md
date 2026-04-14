# Architecture Decisions

## Proto Location

Proto definitions live in the SDK repo at `proto/dtkt/flow/v1beta2/`:
- Package: `dtkt.flow.v1beta2`
- Go import: `github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2` (alias `flowv1beta2`)
- Source: `github.com/datakit-dev/dtkt-universe/dtkt-sdk/proto/dtkt/flow/v1beta2/`

The executor runtime lives alongside the SDK in `sdk-go/flowsdk/v1beta2/`
(same Go module). Proto codegen (`buf generate`) runs in the SDK repo.
The `Runtime` message was renamed to `RunSnapshot` in v1beta2.

## CLI Integration (dtkt-cli)

The CLI (dtkt-cli) uses Ent ORM for persistence with three flow-related schemas:

- **FlowRun** -- the v1beta2 version of the automation resource. Has connections,
  inputs, timeout, state, and references the Flow spec. Replaces `Automation`
  for v1beta2 flows. Edge to FlowState (1:1) and Flow (M:1).
- **FlowState** -- materialized state (`entadapter.MessageField[flowv1beta2.RunSnapshot]`).
  Updated atomically with each FlowEvent in the same ent transaction (outbox pattern).
  Linked to FlowRun via `flow_run_id` FK. Tracks `last_event_uid` for cursor.
- **FlowEvent** -- event log / outbox table. Each row stores a typed
  `entadapter.MessageField[flowv1beta2.RunSnapshot_FlowEvent]` plus denormalized
  `event_type` (OUTPUT/UPDATE) and `node_type` (input/generator/var/action/stream/
  output/interaction) columns for efficient DB filtering. UUIDv7 primary key
  provides time-sortable ordering. Indexed on `event_type` and `node_type`.

The `cmd/flow2` package is the v1beta2 CLI command surface. `cmd/flow` (v1beta1)
 continues to work while flow2 is built and tested.

The entstore (`internal/core/flowv1beta2/outbox/entstore/`) implements the
`outbox.Outbox` interface using ent. `entadapter.MessageField`
handles proto serialization (JSON via protojson) -- no manual
`proto.Marshal`/`Unmarshal`.

## FlowRun Lifecycle: Create, Start, Stream

FlowRun creation is a two-step process. This eliminates the race between
executor startup and client stream connection.

### Sequence

```
Client                              Server (Handler)
  |                                    |
  |-- CreateFlowRun ------------------>|  1. Create DB record (state=PENDING)
  |<--- Operation (done, PENDING) -----|  2. Store Run in memory map
  |                                    |     (executor NOT started yet)
  |-- StreamFlowRunEvents ------------>|  3. Open bidi stream
  |   (name + StartFlowEvent)         |  4. Handler receives first message
  |                                    |  5. Sees start event -> run.Start()
  |                                    |     (executor starts, outputs flow)
  |<--- OutputEvent -------------------|  6. Events stream to connected client
  |<--- OutputEvent -------------------|     (no events missed)
  |<--- FlowRun(SUCCEEDED) -----------|  7. Terminal state event
```

### States

- **PENDING** -- Run created, executor loaded in memory, not yet started.
  Waiting for client to connect and send `StartFlowEvent`.
- **RUNNING** -- Executor running, processing nodes, producing events.
- **STOPPING** -- Inputs closed, pipeline draining.
- **SUCCEEDED/FAILED/ERRORED/CANCELLED** -- terminal states.

### CLI Commands

```
# FlowRun CRUD
dtkt flowrun create NAME --flow <flow>   # state: PENDING (no stream opened)
dtkt flowrun get NAME                    # retrieve flowrun details
dtkt flowrun list                        # list flowruns (--state, --flow filters)
dtkt flowrun delete NAME                 # delete flowrun (stops if active)

# FlowRun execution
dtkt flowrun start NAME|PATH|URI         # create + open stream + start execution
dtkt flowrun stop NAME                   # graceful stop (desired state: STOPPING)
dtkt flowrun attach NAME                 # attach to running flow's event stream

# Sugar
dtkt flow2 run <flow>                    # create + start in one command
dtkt flow2 run <flow> --detach           # create + start, close stream immediately
```

**Implementation status:** All commands implemented.
- `start`, `stop`: `cmd/flowrun/start.go`, `cmd/flowrun/stop.go`
- `create`, `get`, `list`, `delete`, `attach`: `cmd/flowrun/{create,get,list,delete,attach}.go`
- `--detach` flag: on both `flow2 run` and `flowrun start`
- Shared IO layer: `internal/flowio/` (TUI, signal handling, control dispatch)

Backend RPCs (`internal/core/flowrun/handler.go`) are fully implemented for all
operations: `CreateFlowRun`, `GetFlowRun`, `ListFlowRuns`, `UpdateFlowRun`,
`DeleteFlowRun`, `SendFlowRunEvent`, `ReceiveFlowRunEvents`,
`StreamFlowRunEvents`. FlowRun records persist permanently in the ent database.

### Why Two Steps?

The original single-step approach (CreateFlowRun targets RUNNING) had a race:
the executor could produce outputs before the client called StreamFlowRunEvents.
With the two-step approach, the client controls when execution begins by sending
the start event _after_ the stream is connected.

### Persistence (opt-in)

Event persistence is optional. Without persistence, the in-memory LRU event
cache serves live subscribers. With persistence enabled (via entstore), events
are also written to the FlowEvent table, enabling:
- **Historical replay**: stream events for completed runs from the DB
- **Cursor-based resume**: `after_event_id` field on stream requests
- **Cross-restart continuity**: events survive daemon restart

### Attach: Interaction Discovery & Event Replay

When a client attaches to a running flowrun (`dtkt flowrun attach`), it needs
to discover pending interactions and optionally replay historical events.

**Pending interaction discovery (snapshot-based):**

The `RunSnapshot` is the source of truth for pending interactions. Each
`InteractionNode` has a `token` field:
- `token != ""` → interaction is **pending** (waiting for response)
- `token == ""` → interaction is **fulfilled** or never requested

On attach, the client calls `GetFlowRun` to fetch the current snapshot, then
inspects `snapshot.interactions` for nodes with non-empty tokens. For each
pending interaction, the client synthesizes an `InteractionRequestEvent{id,
token}` and prompts the user. The response uses the same
`InteractionResponseEvent{id, token, value}` as a live prompt -- the executor's
`TryDeliver` validates the token atomically, so stale tokens (fulfilled by
another client between GetFlowRun and stream open) are safely dropped.

This approach is preferred over correlating request/response pairs from the
event stream because it uses a single source of truth (the snapshot) and works
regardless of LRU cache size or event replay window.

**Event replay (partially implemented):**

The proto supports `after_event_id` (UUIDv7 cursor) on
`ReceiveFlowRunEventsRequest` and `StreamFlowRunEventsRequest`. Current state:
- Both `ReceiveFlowRunEvents` and `StreamFlowRunEvents` handlers parse
  `after_event_id` from requests and pass it to `ReceiveEvents()`.
- Events are cached in a purpose-built `EventBuffer` ring buffer
  (`flowrun/eventbuf.go`) with oldest-first eviction. `cacheEvent()` assigns
  a fresh UUIDv7 `event_id` to each event at cache time.
- `ReceiveEvents()` uses UUID-based cursor via `EventBuffer.After(afterID)`.

Not yet implemented:
- **DB-backed replay fallback**: when `after_event_id` is not in the buffer,
  `eventLog.ReadEvents()` should be called to replay from persistent storage.
  The `eventLog` field exists on `Run` but is not used in `ReceiveEvents()`.
- **Buffer stores the wrong type**: `EventBuffer` stores
  `*ReceiveFlowRunEventsResponse` (the client-facing projection). It should
  store `*RunSnapshot_FlowEvent` (the canonical event) and project at send
  time. See plan §6.2.2.

**Default attach behavior:** stream live events from the buffer cursor +
synthesize prompts for pending interactions from the snapshot. DB-backed
historical replay is planned (§6.2.3).

## JSON Schema Enrichment

The `Spec` struct has only unexported fields (`flow`, `raw`), so Go reflection
produces an empty JSON schema. `SpecOptions.ExtendSchemaWithContext()` enriches
the schema at sync time by loading proto type schemas from the `TypeSyncer` and
replacing the empty schema with proper `$defs`:

1. **Connection schema**: loads `dtkt.flow.v1beta2.Connection` proto schema,
   enriches `services` items with an enum of all registered service names
   (via `Resolver.RangeServices`).
2. **Action/Stream schemas**: loads `dtkt.flow.v1beta2.Action`/`Stream` proto
   schemas, then loads `dtkt.flow.v1beta2.MethodCall` schema. Enriches the
   `method` field on the call schema with valid method names (unary for actions,
   streaming for streams). Stores as `$defs/actions.call` and `$defs/streams.call`.
3. **Flow schema**: loads `dtkt.flow.v1beta2.Flow` proto schema, rewires `$ref`
   pointers from node array items to local `$defs`, stores as `$defs/Spec`.

The CLI adapter (`syncFlowSpecV1Beta2` in `types/adapter.go`) passes the
callback via `common.WithJSONSchemaCallback` so the schema is enriched before
being stored in the types database and served at
`/schemas/types/dtkt.flowsdk.v1beta2.Flow.jsonschema.json`.

## Lint: Connection & Method Validation

`runtime.Lint()` validates connection references and method call fields:

- **Connection references**: collects declared connection IDs from graph nodes,
  then checks each action/stream `call.connection` against the set. Undeclared
  connections produce **warnings** (not errors) to support mocked/externally-
  provided connections that are injected at runtime.
- **Method call fields**: validates `call.connection` and `call.method` are
  non-empty. Proto validation (`protovalidate`) catches these at decode time,
  but lint provides defense-in-depth for programmatic callers.

## Package Layout

All executor code lives in `sdk-go/flowsdk/v1beta2/` within the SDK repo
(`dtkt-sdk`). This eliminates the import cycle that would exist if the executor
were a separate module importing the SDK while the SDK's `Spec.Lint()` needed
the executor's graph builder and linter.

```
spec.go                  # Spec type, ReadSpec, WriteSpec, Validate, Lint

executor/                # Interfaces & contracts (Executor, NodeHandler, PubSub, topic helpers)

runtime/                 # Concrete executor implementation (local, non-resumable, gochannel-based)
  executor.go            # Executor struct, Execute(), option wiring, outbox/relay lifecycle
  cel.go                 # CEL compilation & evaluation (delegates to common.NewCELEnv)
  compile.go             # compileNode() -- CEL programs, request trees, retry, cron (compiled* types)
  handlers.go            # newHandler() -- wires compiled artifacts with pubsub/channels
  executor_setup.go      # Extracted setup methods: wireEdges, resolveConnections, buildInteractionHandlers, setupInputBridges
  input.go               # Input resolution chain (throttle, cache, default, constant)
  output.go              # Output evaluation & publishing
  transform.go           # Transform pipeline (map, filter, reduce, scan, flatten, group_by)
  var.go, range.go, switch.go, ticker.go  # Node-type handlers
  unary.go, server_stream.go, client_stream.go, bidi_stream.go  # RPC handlers
  value.go               # Value conversion (delegates to shared + cel-go library)
  registry.go            # CEL type registry for flat typed node messages
  outbox.go              # txPublisher, outboxPubSub (outbox wiring for Execute)
  lint.go                # Static flow validation
  executor_*_test.go     # Executor integration tests (all run with and without outbox)

pubsub/                  # Core messaging (like Watermill's message/)
  pubsub.go              # Message, Publisher, Subscriber interfaces
  handler.go             # HandlerFunc, Middleware types
  memory/                # In-process backend (tests + local executor)
  forwarder/             # ForwarderPublisher (decorator) + Forwarder (daemon) -- legacy, not used by v1beta2
  middleware/             # retry, poison queue, recoverer, throttle, timeout, correlation ID
  pubsubtest/            # Conformance test suite for PubSub implementations

rpc/                     # RPC client abstraction
  rpc.go                 # Client, Connector, BidiStream, ClientStream, ServerStream, MethodKind
  mock/                  # In-memory mock backed by registered Go functions (tests)

cache/                   # Pluggable caching (memoize support)
  cache.go               # Cache interface (Get/Set)
  memory/                # In-process LRU cache
  cachetest/             # Conformance test suite for Cache implementations

outbox/                  # Outbox pattern = pubsub + DB storage (depends on databases)
  outbox.go              # Storage interface, outbox pattern wiring
  memory/                # In-memory outbox with tx support (tests)
  outboxtest/            # Conformance test suite for Outbox implementations

cloud/                   # Cloud-tier backends (external dependencies, testcontainers)
  pubsub/
    valkey/              # Valkey Streams PubSub (valkey-glide)
    kafka/               # Kafka PubSub (franz-go)
  cache/
    valkey/              # Valkey Cache (valkey-glide)
```

Outbox DB backends live in their respective deployment repos as ent-based
implementations (`entstore`):
- **dtkt-cli**: `internal/core/flowv1beta2/outbox/entstore/` -- ent + SQLite
- **dtkt-cloud-go**: will have its own entstore -- ent + PostgreSQL

## Relay is Part of Outbox

The outbox Relay daemon reads committed events from the outbox subscriber and
publishes them to the real message broker (gochannel for local, Kafka/Valkey
for cloud). It extracts the destination topic from message metadata
(`TopicMetadataKey`) persisted by `txPublisher` at write time.

The `pubsub/forwarder/` package exists as a generic envelope-based relay
abstraction but is **not used** by the v1beta2 executor. The outbox Relay
(`outbox/relay.go`) replaced it for outbox-to-broker relay because it operates
directly on committed outbox events without envelope wrapping.

## SQL is NOT a PubSub Backend

- Watermill's watermill-sql is a full SQL-based PubSub (polling, offsets) -- different
- We use SQL purely as transactional outbox storage for the Forwarder
- PubSub backends = actual brokers (gochannel, Kafka, NATS, Valkey)
- Outbox storage = ent ORM (SQLite for CLI, PostgreSQL for cloud)

## Outbox Pattern: How It Works

The outbox pattern ensures atomic state + event writes by co-locating both in
the same database transaction. The Relay daemon asynchronously relays
committed events from the outbox to the real message broker.

```
DB transaction (atomic):
  ├── write state (row update -- node value, accumulator, etc.)
  └── write event to outbox table (via outbox Publisher)

Relay daemon (async, background):
  outbox table ──► Kafka / Valkey Streams / NATS / gochannel
```

A tx-bound Publisher writes messages to the outbox in the same transaction as
state writes. The Relay subscribes to the outbox and publishes committed events
to the real broker. Downstream nodes subscribe to topics on the broker, not the
outbox.

### Memory backend (local dev / tests)

The memory outbox exercises the same tx code path (Begin → Storage().Store() →
Commit/Rollback) but everything is in-process. There is no persistent state to
write atomically -- the outbox is the only thing in the transaction. This is
sufficient for Phase 3 because it validates:
- Handler wiring (tx lifecycle, deferred ack, rollback on error)
- Publisher → outbox → Relay end-to-end
- Transaction visibility semantics (staged writes visible only after Commit)

Real atomic state + event writes require a real DB (SQLite/PostgreSQL).

### DB backends (ent + SQLite / PostgreSQL)

DB-backed outbox implementations use ent ORM, not raw SQL. Each deployment
target has its own `entstore` package that implements the `outbox.Outbox`
interface using ent:

- **CLI (dtkt-cli)**: `internal/core/flowv1beta2/outbox/entstore/` wraps
  `ent.Client` transactions. `Store()` uses `FlowEventClient.Create()` with
  typed `SetFlowEvent()` via entadapter, plus denormalized `SetEventType()` and
  `SetNodeType()` for efficient queries. Ordering uses UUIDv7
  (time-sortable, no AUTOINCREMENT).
- **Cloud (dtkt-cloud-go)**: will have its own entstore -- same interface,
  ent + PostgreSQL.

`TxBeginner.Begin()` wraps `ent.Client.Tx()`. The returned `Tx` provides
`Storage()` bound to the ent transaction. State writes and outbox inserts
use the same ent transaction -- `tx.Commit()` is atomic.

## FlowEvent is the Core Event

Every node handler produces `FlowEvent` messages as pubsub events. `FlowEvent`
is the unit of transport, persistence, and external observation. A `FlowEvent`
carries either a node-level event (output, state update) or a flow-level state
transition, discriminated by `event_type`.

### Flat Typed Node Messages

Instead of a single `RunSnapshot_Node` with a `oneof type` (which creates nested
field access in CEL, e.g. `node.input.closed`), each node kind has its own flat
message with common fields promoted directly onto it:

- `RunSnapshot_InputNode`: `id`, `event_id`, `value`, `error`, `event_time`, `closed`, `transforms`, `phase`
- `RunSnapshot_VarNode`: `id`, `event_id`, `value`, `error`, `event_time`, `eval_count`, `transforms`, `phase`
- `RunSnapshot_GeneratorNode`: `id`, `event_id`, `value`, `error`, `event_time`, `done`, `eval_count`, `transforms`, `phase`
- `RunSnapshot_ActionNode`: `id`, `event_id`, `value`, `error`, `event_time`, `eval_count`, `start`, `end`, `phase`
- `RunSnapshot_StreamNode`: `id`, `event_id`, `value`, `error`, `event_time`, `request_closed`, `response_closed`, `request_count`, `response_count`, `start`, `end`, `phase`
- `RunSnapshot_OutputNode`: `id`, `event_id`, `value`, `error`, `event_time`, `eval_count`, `transforms`, `closed`, `phase`
- `RunSnapshot_InteractionNode`: `id`, `event_id`, `value`, `error`, `event_time`, `submitted`, `transforms`, `phase`, `token`

**Why flat?** CEL has native proto support. With flat types registered via
`cel.Types(...)`, flow authors write:

```
= inputs.number.closed          // direct field on RunSnapshot_InputNode
= vars.evenNumbers.eval_count   // direct field on RunSnapshot_VarNode
= streams.echo.response_closed  // direct field on RunSnapshot_StreamNode
```

No `oneof` indirection, no `map[string]any` shim in the runtime. CEL gets
compile-time type checking instead of `cel.DynType`.

### FlowEvent: Transport Envelope

`RunSnapshot_FlowEvent` wraps a typed node (or flow-level state) in a `oneof`
and adds `event_type`:

```protobuf
message FlowEvent {
  enum EventType {
    EVENT_TYPE_UNSPECIFIED = 0;
    EVENT_TYPE_NODE_OUTPUT = 1;  // node produced a value; downstream CEL evaluates
    EVENT_TYPE_NODE_UPDATE = 2;  // accumulator updated, no output; downstream skips
    EVENT_TYPE_FLOW_UPDATE = 3;  // flow-level state transition (RUNNING, SUCCEEDED, etc.)
  }
  EventType event_type = 1;
  oneof data {
    InputNode       input       = 2;
    VarNode         var         = 3;
    GeneratorNode   generator   = 4;
    ActionNode      action      = 5;
    StreamNode      stream      = 6;
    OutputNode      output      = 7;
    InteractionNode interaction = 8;
    FlowState       flow        = 9;
  }
}
```

`EVENT_TYPE_NODE_OUTPUT` events carry a real value that downstream CEL subscribers react to.
`EVENT_TYPE_NODE_UPDATE` events carry updated accumulator state (from reduce/scan steps) but no
output value -- downstream subscribers skip them while the outbox captures both
for persistence and checkpoint/resume.
`EVENT_TYPE_FLOW_UPDATE` events carry flow-level state transitions (e.g. RUNNING → SUCCEEDED)
published on the `FlowTopic` (`"flow.state"`).

### Go Interface for Shared Fields

Since all flat types share common fields, a Go interface makes generic runtime
code (publishNode, outbox serialization) type-agnostic:

```go
type StateNode interface {
    GetId() string
    GetValue() *expr.Value
    proto.Message
}
```

All generated `*RunSnapshot_InputNode`, `*RunSnapshot_VarNode`, etc. satisfy this
automatically via their generated getter methods.

### Snapshot

The snapshot maps are typed per node kind:

```protobuf
message RunSnapshot {
  map<string, InputNode>       inputs       = 2;
  map<string, GeneratorNode>   generators   = 3;
  map<string, VarNode>         vars         = 4;
  map<string, ActionNode>      actions      = 5;
  map<string, StreamNode>      streams      = 6;
  map<string, OutputNode>      outputs      = 7;
  map<string, InteractionNode> interactions = 8;
}
```

`SnapshotAt(uid)` applies committed `FlowEvent`s (both OUTPUT and STATE) up to
`uid`, keeping the latest node state per ID in the appropriate typed map.
`outbox.ApplyFlowEvent()` dispatches each `RunSnapshot_FlowEvent` into the
correct typed map on the `RunSnapshot` snapshot.

### CEL Environment Options

`celEnvOptions()` returns the v1beta2-specific `[]cel.EnvOption` (variable
declarations, `EOF()` function, flat node type registration). The actual CEL
environment is created via `common.NewCELEnv(opts...)` which adds the standard
extension set (URL validation, encoders, string v4, list, proto, enum). This
ensures v1beta2 CEL has the same capabilities as v1beta1 and the rest of the SDK.

```go
func celEnvOptions() []cel.EnvOption {
    return []cel.EnvOption{
        cel.Types(
            &flowv1beta2.RunSnapshot_InputNode{},
            &flowv1beta2.RunSnapshot_VarNode{},
            // ...one per node kind
        ),
        cel.Variable("this", cel.DynType),
        cel.Variable("inputs", cel.DynType),
        cel.Variable("vars", cel.DynType),
        // ...one per namespace
        cel.Function("EOF", ...),
    }
}
```

`buildCELEnv()` (used by Executor) and `buildLintCELEnv()` (used by Lint) both
call `common.NewCELEnv(celEnvOptions()...)` so the standard extensions are
always available. `buildCELEnv` additionally registers connection proto types
via `common.NewCELTypes(resolver)` for resolver-aware type adaptation.

All namespace variables are currently `cel.DynType`. A future improvement is
to declare typed map variables (e.g. `cel.MapType(cel.StringType,
cel.ObjectType("dtkt.flow.v1beta2.RunSnapshot.InputNode"))`) to get compile-time
type checking in CEL expressions instead of runtime errors.

### Shared Value Conversion

Value conversion between `*expr.Value`, `ref.Val`, and Go native types delegates
to existing SDK infrastructure instead of reimplementing it:

| Operation | Implementation | Package |
|---|---|---|
| `*expr.Value` → Go native | `shared.ExprValueToNative(env, val)` | `flowsdk/shared` |
| `ref.Val` → `*expr.Value` | `cel.ValueAsProto(val)` | `cel-go` library |
| `*expr.Value` → `ref.Val` | `cel.ProtoAsValue(adapter, val)` | `cel-go` library |
| CEL expression check | `shared.IsValidExpr(s)` | `flowsdk/shared` |
| Go native → `*expr.Value` | `nativeToExpr(val)` -- test-only helper | `runtime` (test) |
| `proto.Message` → `*expr.Value` | `protoToExpr(msg)` -- v1beta2-specific | `runtime` |

Handler types store a `shared.Env` (threaded from the executor) so they can
call `shared.ExprValueToNative` with resolver-aware ObjectValue unmarshalling.
`exprToMessage()` uses `shared.ExprValueToNative` instead of a local reimplementation.

The old `nodeToState` `map[string]any` shim is eliminated -- `nodeToMap()`
converts each `StateNode` into a map with native `ref.Val` values for CEL.

## Two-Tier Architecture

- **Local (this repo)**: gochannel (ephemeral) + entstore outbox (event log via ent + SQLite in CLI)
- **Cloud (separate repo)**: Kafka/Valkey Streams as broker + entstore outbox (ent + PostgreSQL)
- Shared: pubsub/ interfaces, pubsub/forwarder/, typed flat node protos

### Local = history viewing, not resumable
The outbox (ent + SQLite in CLI) is a durable event log on disk. `SnapshotAt(uid)`
replays committed `FlowEvent`s up to any event UID and returns the state
at that point. This enables **history viewing** (scrubbing) -- useful for
debugging, visualization, and post-mortem inspection.

Local does **not** support replay or resume. gochannel is in-memory -- messages
between nodes are lost on crash. Even without a crash, there is no way to
re-deliver messages through the node pipeline from the outbox because the
inter-node pubsub transport (Go channels) is ephemeral. The outbox captures
what each node _produced_, but cannot re-drive the flow. On crash you restart
from scratch; the old outbox remains readable for history viewing.

### Cloud = crash recovery + historical resume
The cloud tier adds a durable pubsub broker (Kafka, Valkey Streams) and a
durable outbox (PostgreSQL via ent). These serve **different roles**:

**Broker** (Kafka / Valkey Streams) -- live transport:
- Inter-node message delivery during execution
- Crash recovery (continue from where we were): on pod eviction, OOM, or
  network partition the executor restarts and broker consumers reconnect from
  their last committed offsets. Sub-second, zero outbox queries. This is the
  99% recovery case in production.
- Horizontal scaling: distribute node handlers across instances via consumer
  groups. gochannel is in-process only.
- External real-time consumers: monitoring dashboards, audit services, other
  microservices subscribe to broker topics directly.
- Backpressure and delivery guarantees: ordered at-least-once delivery,
  consumer group rebalancing, dead letter queues.

**Outbox** (PostgreSQL via ent) -- durable event log:
- History viewing (same as local, `SnapshotAt`)
- Historical resume (rollback to a past point): query the outbox to
  reconstruct state and compute in-flight messages per subscription. This is
  a more expensive operation (seconds to minutes depending on event volume)
  but it is operator-initiated, not automatic.

| Scenario                      | Source of truth        | Broker role                  |
|-------------------------------|------------------------|------------------------------|
| History viewing (slider)      | Outbox                 | Not involved                 |
| Historical resume (rollback)  | Outbox (reconstruct)   | Not involved (fresh queues)  |
| Crash recovery (continue)     | Broker (offsets)       | Primary -- just restart      |
| Live execution                | Broker (transport)     | Primary -- message delivery  |
| Multi-instance scaling        | Broker (shared)        | Required                     |
| External consumers            | Broker (subscriptions) | Required                     |

The outbox handles the **hard problem** (historical reconstruction). The broker
handles the **operational problem** (keep things running, recover from crashes).

### What is lost on local crash
- In-flight gochannel messages (inter-node messages not yet delivered)
- Transform accumulator state not yet committed to the outbox
- Any handler work between the last outbox commit and the crash
- Committed outbox events are safe on disk (SQLite WAL) -- viewable but not replayable

### History Viewing vs Resume

History viewing and resume are fundamentally different operations because of the
inter-node message topology (see "Inter-Node Message Topology" above). The outbox
stores node *outputs* but not subscriber *cursors* -- unless we add last-acked
tracking. Resume is a **cloud-only** feature (requires durable outbox +
operational infrastructure). Local mode (gochannel + SQLite) only supports
history viewing.

#### History viewing

`SnapshotAt(uid)` applies committed `FlowEvent`s up to `uid` and returns the
reconstructed `RunSnapshot`. Works on both tiers. No pubsub involvement.

#### Resume via outbox replay + last-acked tracking (cloud only)

Each `FlowEvent` records the **last acked upstream message UUID** per subscription
at the time of the event's commit. This is stored as a map of upstream topic to
UUIDv7 on the `FlowEvent`. The outbox is the single source of truth for resume --
no broker seeking is needed. Fresh pubsub queues are rebuilt and seeded from the
outbox event log.

```
                                        rollback cursor: A5
                                                 ↓
Node A outbox:  A1 ─── A2 ─── A3 ─── A4 ─── A5 | A6 ─── A7  (discard)
                               │
Node B outbox:  B1 ─── B2 ── B3 |                              (discard)
                (acked A1) (acked A2) (acked A3)

In-flight for B: [A4, A5]  ← read from A's outbox between A3 and A5
New queue for "node.A" → seed with [A4, A5]
```

**Resume procedure**:
1. Pick a rollback point (some event UID X)
2. For each node, find its last event at or before X -- that's the node's
   rollback cursor
3. `SnapshotAt` each node's rollback cursor for state
4. For each node's subscriptions, read `last_acked` from its last event --
   that's the consumption cursor per upstream topic
5. Query the upstream node's outbox for events **between** `last_acked` and
   the upstream's rollback cursor -- these are the in-flight messages
6. Create **fresh** pubsub queues, seed them with those in-flight messages
7. Start node handlers with restored state + seeded queues
8. Everything after the rollback cursor on every node is discarded

This works because `B.last_acked["node.A"]` is guaranteed `<= A.rollback` --
you cannot ack a message that was never published. The in-flight set is always
well-defined and bounded.

**Inputs**: Input events come from outside the flow. On resume, two options:
- **Replay from outbox**: treat input events like any other node's events.
  The outbox has `InputNode` events up to the rollback cursor. In-flight input
  events are seeded into the queue like any other.
- **Re-prompt**: discard input events after the rollback point and prompt the
  user for fresh input. Appropriate when the user wants to "go back and try
  different input."
- This is likely a per-resume-request choice (e.g. `resume --replay-inputs`
  vs default re-prompt behavior).

**Multi-upstream nodes**: a node subscribing to both A and C records
`last_acked["node.A"]` and `last_acked["node.C"]` independently. Each
upstream's in-flight messages are computed and seeded separately.

**Transforms**: last-acked tracking happens at the node-to-node boundary, not
within transform steps. On resume the transform pipeline restarts empty --
in-flight messages re-enter the pipeline from the seeded queues. Reduce/scan
accumulators are already in the snapshot via `STATE` events.

**Idempotency**: a node may have partially processed a message before the
snapshot point (consumed but not committed its own event). On resume that
message is re-delivered. Handlers must be idempotent -- dedup by message UUID.
This is already a requirement for any at-least-once delivery system.

**No broker dependency for resume**: the outbox event log is the source of truth
for rebuilding queues. The broker (Kafka / Valkey Streams) is needed for *live*
inter-node messaging during execution, but resume bootstraps entirely from the
outbox. Fresh pubsub queues are seeded with in-flight messages, then live
messaging takes over for new events.

**What this adds to the proto**: a `map<string, bytes> last_acked` field on
`FlowEvent` (or equivalent metadata on the pubsub message). Keys are upstream
topic names, values are the UUIDv7 of the last consumed message from that topic.

## State Writes = Executor Logic

- Executor opens tx → writes state → publishes to outbox in same tx → commit
- Not a pubsub or outbox concern -- outbox provides storage, executor composes everything
- Both tiers do state + event atomically in same tx
- For memory backend: no persistent state, tx only wraps outbox writes
- For DB backends: same ent tx used for both state writes and outbox inserts

Implemented in `runtime/outbox.go` as `txPublisher`: wraps `outbox.TxBeginner`,
calls `Begin()` → `outbox.NewPublisher(tx.Storage()).Publish()` → `ApplyFlowEvent()`
→ `tx.StateWriter().WriteState()` → `tx.Commit()`. The materialized `RunSnapshot`
is updated in-place with each event so current-state reads are always a single-row
lookup.

### Execution loop
Node handlers (`Run(ctx) error`, long-lived goroutines) drive a per-message
transaction loop. Each node handler goroutine:
1. Receive message from subscriber (ack-before-next)
2. Process: CEL eval, RPC call, or other node-specific logic
3. Open tx via `TxBeginner.Begin(ctx)`
4. Write state in tx (node value, transform accumulator state)
5. Create outbox storage bound to tx
6. Publish `FlowEvent` events → outbox writes state + event in same tx
7. Commit -- atomic state + event
8. Ack the input message
9. Relay daemon relays outbox events to gochannel (local) or broker (cloud)

Node handlers STAY as `Run(ctx) error` goroutines. HandlerFunc is a separate
concept used for transform steps and middleware (see below).

## Outbox Table = Event Log

- **Both tiers**: The outbox table IS the durable event log.
  `SnapshotAt(uid)` reconstructs state at any point for history viewing.
- **Local**: Read-only after crash -- gochannel messages are lost, so you
  cannot continue execution. Start a new flow run; the old outbox remains
  readable for history viewing.
- **Cloud**: The outbox is the source of truth for **historical resume**
  (rollback to a past point -- reconstruct state + in-flight from outbox,
  seed fresh queues). The **broker** handles crash recovery (continue from
  committed offsets, no outbox involvement).

### Per-Transaction Storage

Each outbox commit writes **two things** atomically in the same ent transaction:

1. **RunSnapshot row** (materialized current state) -- updated in-place, one row
   per FlowRun. Always reflects the latest committed state. This is NOT event
   sourcing -- you never need to rebuild current state from events.
   Includes `last_event_uid` (UUIDv7 of the most recently applied FlowEvent)
   so consumers can determine exactly which event the current state reflects.
2. **FlowEvent row** (event log / outbox entry) -- appended, never updated.
   Stores the `FlowEvent` plus denormalized `event_type` and `node_type` for
   efficient DB filtering.

Reading current state is always a single-row lookup (`WHERE flow_run_id = ?`).
`SnapshotAt(uid)` is only needed for historical access -- it replays FlowEvents
to reconstruct past state.

### Periodic Snapshots for Efficient History Access

Currently `SnapshotAt(uid)` replays all events from event 0 to `uid`. This is
O(N) on the total event count. For long-running flows this becomes expensive.

Periodic snapshots solve this with a single-table design on RunSnapshot:

**Schema additions to RunSnapshot**:
- `snapshot bool` -- false for the live row, true for frozen checkpoints
- `events_since_snapshot int` -- counter incremented on each commit, reset on snapshot
- `last_event_uid bytes` -- UUIDv7 of the last FlowEvent applied to this row

**Invariant**: exactly one `snapshot = false` row exists per FlowRun at all
times. This is the live row used for current state reads.

**Normal transaction** (every commit):
1. Update the `snapshot = false` row (RunSnapshot) with new state
2. Increment `events_since_snapshot`
3. Set `last_event_uid` to the new FlowEvent's UID
4. Insert FlowEvent row (as usual)

**Snapshot transaction** (every N events, e.g. N = 100):
1. Mark the current `snapshot = false` row as `snapshot = true` (freeze it)
2. Insert a new `snapshot = false` row with the same state,
   `events_since_snapshot = 0`, and `last_event_uid` set to the new FlowEvent's UID
3. Insert FlowEvent row (as usual)

This adds one extra row write every N events -- negligible overhead.

**Current state** (hot path, unchanged):
```sql
SELECT * FROM flow_states WHERE flow_run_id = ? AND snapshot = false
```
Always returns exactly one row.

**`SnapshotAt(uid)`** (history access, now O(N/checkpoint_interval)):
1. Find the nearest `snapshot = true` row with `uid <= target_uid`
2. Replay only the FlowEvents between that checkpoint and `target_uid`

Worst case replays N events (one checkpoint interval) instead of the entire
history.

**Retention GC**:
- Delete old `snapshot = true` rows and their preceding FlowEvents
- Keep the most recent K snapshots for history scrubbing
- The `snapshot = false` (live) row is never deleted

This is a future optimization. The current replay-from-0 approach is correct
and sufficient for initial implementation. Periodic snapshots become valuable
when flows produce thousands of events or when rolling retention windows are
needed.

## Message Contract

- `Payload proto.Message` -- generic protobuf payload, not domain-typed
  - Between nodes: `*RunSnapshot_FlowEvent` (wraps typed flat node + event_type)
  - Between transform steps: `*expr.Value`
  - Both implement `proto.Message` -- no wrapping or Any needed
- UUID, Metadata map, Context per message
- Idempotent Ack/Nack; Nack triggers redelivery
- Subscriber-paced: ack before next message, publisher never blocks on subscriber

The pubsub package has NO knowledge of `RunSnapshot_Node` or any domain type.
`pubsub.NewMessage(payload proto.Message)` accepts any proto message.

## Node Handlers vs HandlerFunc

These are two different things. Do not conflate them.

**Node handlers** (`Run(ctx) error`): Long-lived goroutines, one per node in the
graph. Each goroutine runs for the lifetime of the flow execution. It receives
messages from its subscriber, processes them (CEL eval, RPC, etc.), and publishes
results. This is the existing pattern in `runtime/handlers.go` and it stays.

**HandlerFunc** (`func(msg *Message) ([]*Message, error)`): A stateless,
per-message function. Used for:
- **Transform steps**: each transform (filter, map, reduce, etc.) is a
  HandlerFunc that subscribes to an input topic and publishes to an output topic
- **Middleware**: retry, throttle, timeout, etc. wrap a HandlerFunc

A node handler publishes its raw value to the transform pipeline's input topic.
The transform pipeline is a chain of HandlerFuncs connected by internal topics.
The final HandlerFunc publishes the `RunSnapshot_FlowEvent` to the node's output topic.
If a node has no transforms, the handler publishes directly to the output topic.

## Inter-Node Message Topology

At runtime a flow is a DAG of concurrent node handlers connected by pubsub
topics. Each node handler subscribes to its upstream topics and publishes
`FlowEvent` messages to its own output topic. Transform pipelines add
additional internal topics *within* a node. At any given moment there are
**N independent subscriber cursors** -- one per node handler, plus one per
transform step -- each at a different position in its topic.

```
[Input A]  ──publish FlowEvent──►  topic "node.A"
                                        │
                              ┌─────────┴──────────┐
                              ▼                    ▼
                        [Var B]              [Var C]
                        subscribes to        subscribes to
                        "node.A"             "node.A"
                              │                    │
                    (transform pipeline)     (no transforms)
                    ┌─────────┘                    │
                    ▼                              ▼
              "node.B.transform.input"       topic "node.C"
                    │ (filter)
                    ▼
              "node.B.transform.0"
                    │ (reduce -- STATEFUL)
                    ▼
              "node.B.transform.1"
                    │ (wrapper)
                    ▼
              topic "node.B"
```

### What the outbox captures vs. what it doesn't

**Captured**: Every node's *output* -- the `FlowEvent` published to
`node.{id}`. Both `OUTPUT` (value produced) and `STATE` (accumulator updated)
event types are stored with a UUIDv7 UID.

**NOT captured** (currently):
- Which messages each subscriber *consumed* (no consumption cursors)
- Where each subscriber's cursor was at any point in time
- In-flight messages inside transform pipelines (`node.{id}.transform.*`)

Note: causal links ("Node B consumed event X from A, producing event Y") are
only meaningful for unary calls. Streams produce multiple outputs per input
with no 1:1 mapping, so a causal DAG is not a viable general model.

This is the fundamental gap between **history viewing** (state reconstruction
from the outbox) and **resume** (re-establishing the live message flow).
See "History Viewing vs Resume" below for the concrete scenarios.

## Transform Sub-Node Model

Transforms are a structure INSIDE a node. Each transform step processes
`*expr.Value` → `*expr.Value` with publish/subscribe semantics between steps.
Steps are goroutines connected by pubsub topics:

```
node handler → publish *expr.Value → node.{id}.transform.input
                                          ↓ (filter goroutine)
                                     node.{id}.transform.0
                                          ↓ (map goroutine)
                                     node.{id}.transform.1
                                          ↓ (wrapper goroutine)
                                     node.{id}  ← *RunSnapshot_FlowEvent
```

- Each step goroutine subscribes to its input topic, processes `*expr.Value`
  payloads, and publishes to its output topic
- Transform step signature: `func(*expr.Value) (*expr.Value, error)` -- nil
  return means "don't publish" (e.g. filter drops, reduce accumulates silently)
- EOF flows through the pipeline as a regular value. Reduce reacts to EOF by
  emitting its accumulator; other steps pass EOF through.
- The last step in the chain is a "wrapper" that takes `*expr.Value`, wraps it
  in a `RunSnapshot_FlowEvent{event_type: NODE_OUTPUT}` (with the appropriate typed flat
  node message populated), and publishes to the node's output topic
- On each reduce/scan accumulation without output, the wrapper publishes a
  `RunSnapshot_FlowEvent{event_type: NODE_UPDATE}` so the outbox captures the updated
  accumulator without triggering downstream CEL evaluation
- If a node has no transforms, the handler publishes `RunSnapshot_FlowEvent` directly to
  `node.{id}` (no pipeline, no wrapper goroutine)
- Middleware (retry, timeout) applies per-step, not per-pipeline
- Stateful transforms (reduce, scan) persist accumulators in the outbox via the
  same per-message transaction loop (see "Execution loop")

## Inputs and Interactions via PubSub

Inputs and Interactions both represent "I need a value from outside the flow."
The key distinction: **Inputs are entry points with no graph dependencies.
Interactions are graph-aware nodes with upstream edges** (their prompts can
reference upstream state via CEL).

Both use a PubSub-based event model with distinct event types:
- **InputEvent** / **InputRequestEvent**: unsolicited push (InputEvent) or
  executor-solicited request (InputRequestEvent) for input data.
- **InteractionRequestEvent** / **InteractionResponseEvent**: token-based
  request/response pair for interactions. See "Interaction request/response model".

### Input resolution

Input nodes subscribe to a PubSub topic (e.g. `inputs.number`) instead of raw Go
channels. The executor publishes a "need input" event when a node is blocked
(demand-driven, not eager). Resolution follows a priority chain:

1. **Value already available** on the topic (piped data, API call) -- use it
2. **Cached value** from earlier in this run (`cache: true`) -- reuse it
3. **Type default** (`default` field on the type, e.g. `Int64.default`) -- use it
4. **Block and request** -- publish "need input" event, wait for external provider

CLI mode: an external goroutine reads stdin and publishes to the input topic.
Piped mode (`cat data.jsonl | executor`): no prompts, values fed directly.

### Input field semantics

- **`constant`**: bypasses PubSub entirely. The value is provided once and never
  changes for the flow's lifetime. The input channel closes after the first
  value. Downstream nodes evaluate once and don't need to re-subscribe.
  `constant` is implicitly cached, but a cached input isn't necessarily constant.
- **`cache`**: if the same input is requested again within this flow run, reuse
  the last provided value instead of blocking for a fresh one.
- **`default`** (on the type, not Input): fallback value when the input is
  requested but not available and not cached. Defined on the underlying type
  message (e.g., `Int64.default`, `String.default`, `List.default`). The type
  determines the value's shape and validation.
- **`throttle`**: controls how often the input resolves a fresh value. The wait
  window per attempt is `interval / count`. If no value arrives within the
  window, falls back to cache (if enabled), then default, then sends a request
  event to the client.

### Throttle injection on Inputs

`throttle` defines the wait window before the resolution chain advances to
fallbacks. Without it, the input blocks indefinitely until a value arrives.

**`cache` and type-level `default` require a throttle.** Without a throttle
window, there is no point at which the runtime advances to the fallback:
- `cache: true` without `throttle` = blocks forever on first resolution (no
  cached value yet), then never refreshes (equivalent to `constant`)
- type `default` without `throttle` = blocks forever, default never fires

The runtime injects a default throttle when `cache` or the type's `default` is
set but `throttle` is omitted. This default is an executor-level configuration
option (`WithDefaultInputThrottle(rate)`), not a hardcoded value. Flows that
set `cache` or a type-level `default` without an explicit `throttle` inherit
the executor's configured default. Flows that set neither `cache` nor a type
`default` block until a value arrives (no injection).

### Interaction nodes

- Can depend on other flow nodes (participate in the DAG)
- Always need a prompt (title, description, form fields)
- Prompts can be dynamic CEL expressions referencing upstream state
- Token-based request/response: executor emits InteractionRequestEvent with
  UUIDv7 token, stores token on InteractionNode, blocks until matching
  InteractionResponseEvent arrives. Delivered via PubSub topic.
- Neither `constant` nor `default` apply (Interactions are always prompted)
- `interactions.<id>.token` available in CEL -- non-empty while waiting for response

### `cache` vs `memoize`

- **`cache`** (Input, Var, Action): reuse the last result if re-triggered,
  regardless of whether the trigger/request changed.
- **`memoize`** (Action only): skip the unary RPC if the same request input
  has been seen before and return the previously cached response. Keyed by
  deterministic hash of the resolved request value. Different request = new RPC.
  Only applicable to unary RPCs; streaming RPCs have no stable
  request-to-response mapping.

`cache` is about value availability. `memoize` is about avoiding redundant
side effects for identical requests.

### Outputs via PubSub

- Output nodes publish to a PubSub topic (e.g. `outputs.evenSum`)
- Consumers subscribe to output topics via PubSub

## Middleware

Implemented in `pubsub/middleware/`:
- retry, poison queue, recoverer, throttle, timeout, correlation ID

### Throttle and timeout

Throttle and timeout are **node-level execution concerns**, not pubsub middleware:

- **Throttle on Actions/Streams/Outputs**: subscribe-side rate limiting on the
  node's input subscription. Uses the pubsub `Throttle` middleware injected into
  the subscriber pipeline when configured. Limits how often the handler fires.
- **Throttle on Generators (Range)**: a `Rate` field on Range (`count` per
  `interval`). Uses a `time.Ticker` internally when set. Without it, Range emits
  as fast as possible. Ticker already has `interval` for pacing.
- **Timeout**: per-evaluation-cycle `context.WithTimeout` around CEL eval or
  RPC call inside the handler. Pubsub middleware has no visibility into handler
  internals.

The pubsub `Throttle` middleware remains available for transport-level use
(Forwarder to external systems, etc.).

The global `WithThrottle()` executor option (bare `time.Sleep` in handlers)
has been removed. Node-level throttle uses pubsub `Throttle` middleware.

## Error Handling & Flow Lifecycle

Inspired by Argo Workflows but adapted to our streaming DAG model (Argo runs
batch jobs; we run long-lived streaming nodes).

### Error categories
- **Transient**: network timeout, temporary unavailability -- retryable
- **Expected/operational**: "table already exists", constraint violation -- handler
  may return a structured error the graph can branch on (like Argo's `.Failed` vs `.Errored`)
- **Fatal**: invalid CEL expression at eval time, panic -- not retryable, fail the node

### Per-node error behavior: `retry_strategy`

Configured via `retry_strategy` field on Action and Stream proto messages.
Replaces the generic retry+onError pattern with a CEL-driven decision pipeline.

**No retry_strategy = fail-fast default.** If no retry_strategy is defined, any
error kills the node immediately. Combined with flow-level fail-fast (default),
this terminates the entire flow.

**Proto structure** (see `flow.proto` for full definition):
```yaml
retry_strategy:
  when: "= CEL"         # activates on this.response / this.error; omit = always
  backoff:               # automated retries
    initial_backoff: 1s
    backoff_multiplier: 2.0
    max_backoff: 30s
    max_attempts: 3
  skip_when: "= CEL"      # true -> skip this item, continue processing
  suspend_when: "= CEL"   # true -> pause flow until external resume
  terminate_when: "= CEL" # true -> terminate flow
```

**Execution order:**
1. Action/Stream RPC returns -- `when` evaluates against `this.response` and `this.error`
2. If `when` is false (or absent and no error), publish result normally
3. If `when` is true (or absent and there IS an error):
   a. Evaluate `skip_when` / `suspend_when` / `terminate_when` -- if any match,
      act immediately (skip item, suspend flow, or terminate flow)
   b. If none match, attempt `backoff` retry (if defined and max_attempts not
      exhausted)
   c. After retries exhausted and still no escalation expression matches, the
      node fails

Escalation expressions evaluate on **every error**, not just after retries are
exhausted. This allows immediate early-exit for known-unrecoverable errors
(e.g., INVALID_ARGUMENT → terminate, NOT_FOUND → skip) without wasting retry
attempts on errors that will never succeed.

**CEL scoping inside retry_strategy:**
- `this.response` -- the just-returned RPC result (current invocation only)
- `this.error` -- google.rpc.Status from the RPC call (code, message, details)
- `actions.foo.value` / `streams.bar.value` -- previous published values from
  other nodes (via normal dependency resolution, NOT the current invocation)

**Blocking model:** The action handler blocks while the retry loop runs.
Error -> retry all happen synchronously within the handler. This preserves
message ordering (no out-of-order publishes). The handler only publishes a
result (or fails) after the retry strategy resolves.

### Interaction request/response model

Interaction nodes are standalone graph nodes (`Flow.interactions`) with upstream
dependencies, transforms, and their own topic. Used for dependency-driven prompts
(e.g., "approve before proceeding").

The interaction protocol uses a token-based request/response pattern:

1. **InteractionRequestEvent** (executor -> external): emitted when the interaction
   node needs input. Contains `id` (node ID) and `token` (single-use UUIDv7).
2. **InteractionResponseEvent** (external -> executor): sent by the external
   source with matching `id`, `token`, `value` (google.protobuf.Any), and `actor`
   (resource name, e.g. "users/bob", "agents/claude").

**External event boundary:** External events (InputEvent, OutputEvent,
InteractionRequestEvent, InteractionResponseEvent, ResumeNodeEvent) use
`google.protobuf.Any` for values -- never `cel.expr.Value`. The `Any` carries
well-known wrapper types (e.g. `google.protobuf.Int64Value`) or domain protos.
The executor demux wraps the `Any` in `expr.Value_ObjectValue` for internal
transport; CEL automatically unwraps well-known types during evaluation.
Conversely, `cel.expr.Value` is internal node-to-node transport only -- it
never appears on external-facing events.

**Token semantics:**
- The executor generates a UUIDv7 token per request and stores it on the
  `InteractionNode.token` field in the RunSnapshot.
- Token presence on InteractionNode = waiting for a response.
- The responder must echo the token back. Mismatched or reused tokens are rejected.
- When a matching response arrives, the token is cleared from InteractionNode.
- The same interaction node can fire multiple times (retry, loop, re-suspend) --
  each firing generates a new token, invalidating any previous outstanding token.

**No inline interaction on RetryStrategy.** Interactions are always standalone
graph nodes. The old `RetryStrategy.interaction` and `retry_with` fields have
been removed (reserved in proto). Error escalation to humans is handled by
suspending the flow (`suspend_when`) and resuming externally.

### Node phases
Each node has a lifecycle phase (see `RunSnapshot.Phase`):

| Phase | Meaning | Terminal? |
|---|---|---|
| `PENDING` | Registered, no messages received yet | No |
| `RUNNING` | Actively processing messages | No |
| `STOPPING` | Draining: stop requested, waiting for in-flight operations | No |
| `SUCCEEDED` | Completed normally (EOF, graceful stop) | Yes |
| `FAILED` | Expected/operational error. Downstream can branch on it | Yes |
| `ERRORED` | Unexpected error (retries exhausted, fatal). Triggers `error_strategy` | Yes (resumable) |
| `SUSPENDED` | Paused by `suspend_when` or operator `SuspendNodeEvent` | No (resumable) |
| `CANCELLED` | Terminated by operator action. May carry a CANCELLED error if interrupted mid-operation | Yes (resumable) |

**Resumable phases:** SUSPENDED, ERRORED, and CANCELLED can be resumed via
`ResumeNodeEvent` or `ResumeFlowEvent`. On resume, the node returns to PENDING
and re-enters the execution loop. SUCCEEDED and FAILED are final -- resuming
them is a no-op.

**Unmatched switch branches:** In a streaming DAG, nodes on unmatched switch
branches simply never receive messages -- they stay PENDING. There is no
"skipped" phase (unlike Argo's batch model where tasks are scheduled upfront).

### Flow-level error strategy (`ErrorStrategy` on `Flow`)

When a node enters `PHASE_ERRORED`, the flow's `error_strategy` field
determines what happens next:

| Strategy | Behavior | Flow transition |
|---|---|---|
| **TERMINATE** (default) | Kill context immediately. No drain. | RUNNING → TERMINATING → ERRORED |
| **STOP** | Graceful drain: close input EOFs, flush in-flight. | RUNNING → STOPPING → FAILED |
| **CONTINUE** | Errored node publishes PHASE_ERRORED + EOF; dependents drain. Independent paths keep running. Errors collected and returned. | Stays RUNNING → returns collected errors |

**Partial execution (CONTINUE):** An errored node publishes PHASE_ERRORED
with an EOF value, causing dependents to drain via normal EOF propagation.
Independent branches (different generator/input paths) continue normally.
Generators on independent paths are NOT cancelled (unlike STOP). Node errors
are collected via `errors.Join` and returned after all goroutines complete.
`TerminateError` still overrides CONTINUE (immediate termination).

**`PHASE_FAILED` vs `PHASE_ERRORED` at flow level:** Failed nodes do NOT
trigger `error_strategy`. A Failed node is an expected domain error -- downstream
edges can branch on it (`continueOn`). Only Errored nodes (unexpected) trigger
the strategy.

### Flow and node commands (events.proto)

External operators control the flow via command events. There are two levels:
flow-level (affects all nodes) and node-level (targets a specific node by ID).

#### Flow-level commands

| Command | Behavior | Flow transition |
|---|---|---|
| **StopFlowEvent** | Close all input EOFs, generators stop, let pipeline drain. Nodes finish current work → PHASE_SUCCEEDED. | RUNNING → STOPPING → SUCCEEDED |
| **TerminateFlowEvent** | Cancel flow context immediately. Running nodes get context cancelled → PHASE_CANCELLED (with error if mid-operation). | RUNNING → TERMINATING → CANCELLED |
| **SuspendFlowEvent** | All nodes finish current operation, then pause. No new messages processed. | RUNNING → SUSPENDED |
| **ResumeFlowEvent** | Resume all SUSPENDED/ERRORED/CANCELLED nodes → PHASE_PENDING. Succeeded/Failed nodes left as-is. | SUSPENDED/ERRORED → RUNNING |

**SuspendFlowEvent details:**
- Nodes in PENDING/RUNNING finish their current in-flight operation (suspend-
  after-current), then transition to PHASE_SUSPENDED.
- Nodes already in terminal phases (SUCCEEDED, FAILED, ERRORED, CANCELLED)
  are left as-is.
- Generators stop publishing. Streams stay open but pause send/receive.

**ResumeFlowEvent details:**
- Only touches nodes in PHASE_SUSPENDED, PHASE_ERRORED, or PHASE_CANCELLED.
- SUCCEEDED and FAILED nodes are not restarted (nothing to resume).
- If a resumed errored node hits the same error, it re-enters PHASE_ERRORED
  and `error_strategy` applies again.
- Downstream nodes that were starved (blocked on a suspended/errored upstream)
  will unblock once the resumed node produces output.

#### Node-level commands

| Command | Behavior | Node transition |
|---|---|---|
| **StopNodeEvent** | Graceful shutdown of a single node. | → PHASE_SUCCEEDED |
| **TerminateNodeEvent** | Immediate cancellation of a single node. | → PHASE_CANCELLED |
| **SuspendNodeEvent** | Pause a single node after current operation. | → PHASE_SUSPENDED |
| **ResumeNodeEvent** | Resume a SUSPENDED/ERRORED/CANCELLED node. Optional `value` replaces pending input. | → PHASE_PENDING |

**StopNodeEvent by node type:**
- Input: close input (EOF), mark SUCCEEDED when drained
- Generator: stop firing, SUCCEEDED
- Var/Output: stop evaluating, SUCCEEDED
- Action (unary): if idle, SUCCEEDED. If mid-RPC, wait for completion, then SUCCEEDED
- Stream: close request side (EOF), wait for response close, then SUCCEEDED
- Interaction: cancel outstanding token, SUCCEEDED

**TerminateNodeEvent by node type:**
- Input/Generator/Var/Output: CANCELLED immediately
- Action: if idle, CANCELLED. If mid-RPC, cancel RPC context → CANCELLED with error
- Stream: cancel stream, close both sides → CANCELLED with error if active
- Interaction: cancel outstanding token, CANCELLED

**Node-level stop/terminate do NOT trigger `error_strategy`.** They are
operator-initiated clean transitions, not unexpected errors.

**ResumeNodeEvent value injection:**
- If `value` is provided (google.protobuf.Any), it replaces the node's pending
  input for the next evaluation. Example: resuming an errored action with a
  corrected RPC request proto.
- If `value` is absent, the node re-evaluates from upstream dependencies.
- SUCCEEDED and FAILED nodes cannot be resumed (no-op).

### Flow lifecycle events
Published on the `FlowTopic` (`"flow.state"`) as `FlowEvent` messages with
`EVENT_TYPE_FLOW_UPDATE`. The `FlowState` oneof carries the flow-level state:
- `dtkt.flow.started` -- Execute() begins
- `dtkt.flow.completed` -- all nodes Succeeded, flow Succeeded
- `dtkt.flow.failed` -- graceful drain after error (STOP strategy), flow Failed
- `dtkt.flow.errored` -- node Errored with TERMINATE, or all paths blocked (CONTINUE)
- `dtkt.flow.cancelled` -- context cancelled externally
- `dtkt.flow.suspended` -- flow suspended (all runnable nodes paused)
- `dtkt.flow.resumed` -- flow resumed from SUSPENDED/ERRORED

### Flow termination
- **Normal**: all leaf nodes (outputs) have completed -- nothing left to consume
- **Error (TERMINATE)**: node Errored → cancel context → TERMINATING → ERRORED
- **Error (STOP)**: node Errored → close input EOFs → STOPPING → FAILED
- **Error (CONTINUE)**: node Errored → node idle, flow stays RUNNING
- **Stop (operator)**: StopFlowEvent → close input EOFs, drain → STOPPING → SUCCEEDED
- **Terminate (operator)**: TerminateFlowEvent → cancel context → TERMINATING → CANCELLED
- **Suspend (operator)**: SuspendFlowEvent → pause all nodes → SUSPENDED
- **External cancel**: caller cancels the context (Ctrl+C, API terminate)

`FlowRun.State` includes `STOPPING = 3` (graceful drain in progress) and
`TERMINATING = 9` (context cancellation in progress). Both are transient states
that resolve to terminal states (SUCCEEDED, FAILED, CANCELLED, or ERRORED
depending on the trigger).

### Lifecycle hooks (future)
Like Argo's `LifecycleHook` -- trigger actions on status changes (e.g. notify
on `flow.errored`). Simple version: callback functions registered on the executor.

## Components NOT Needed

- Router (our executor IS the router)
- CQRS
- FanIn (executor handles joins via CEL activation)

## Library Decisions

- **SQLite**: `modernc.org/sqlite` -- pure Go, no CGO. Used by ent as the
  database driver for local persistence (CLI). Ent manages schema and queries.
- **Cron**: `github.com/robfig/cron/v3` -- same scheduling syntax as k8s CronJob.
  Used by the cron generator handler.
- **Proto codegen**: `buf` (already configured in `buf.gen.yaml`).
- **No external pubsub library**: custom `pubsub/` package (replaced Watermill).
- **gRPC status**: `google.golang.org/grpc` for `status.Error()` / `codes` --
  `this.error` in CEL is `google.rpc.Status` (code, message, details).

## Sentinel Values & CEL Functions

### EOF sentinel
`EOF` signals end-of-stream. Represented internally as a `flowv1beta2.EOF` proto
message wrapped in `cel.expr.Value.object_value`. The runtime detects it and
triggers control-flow behavior:

- **Stream request input**: closes the client side (equivalent to CloseSend).
  Ergonomic alternative to `close_request_when` -- both approaches are supported.
- **Generator value**: signals the generator is done.
- **Input**: closes the input channel.

All three stream handlers (bidi, client-stream, server-stream) already detect EOF
values in the input receive loop and close the request side accordingly.

### EOF() CEL function
Custom CEL function that returns the EOF sentinel value. Required so that CEL
expressions can produce the sentinel:

```yaml
request: "= condition ? EOF() : value"
```

Registered in `compileCEL()` as a zero-argument function returning the EOF
sentinel `*expr.Value`. Generator expressions also receive `this.count` (int64)
and `this.time` (timestamp) in their CEL activation.

### No other custom functions
All data access is through globals and `this` context variables (documented on
the `Flow` message in `flow.proto`). Standard CEL functions (size, has, type,
int, uint, double, string, bytes, matches, contains, startsWith, endsWith,
timestamp, duration, list/map operations, ternary `? :`, etc.) are available.

---

# Codebase Current State

> See `docs/plan-complete.md` for the full history of completed phases.

All core runtime features are implemented: node handlers (12 types), transform
pipelines, outbox pattern (transactional state + event writes), suspend/resume,
error strategies (TERMINATE/STOP/CONTINUE), retry with CEL-driven escalation,
interaction token protocol, per-connection RPC model, middleware (retry, poison,
recoverer, throttle, timeout, correlation ID), conformance test suites
(pubsubtest, cachetest, outboxtest), and cloud backends (Kafka, Valkey Streams,
Valkey Cache).

**Remaining:**
- Cursor-based event replay: buffer type migration + DB fallback
- Actor attribution: InteractionResponseEvent.actor stored but not surfaced
- CloudEvents envelope: deferred until external streaming interface is designed

## Kept Interfaces

- `rpc/rpc.go`: `rpc.Client` (RPC execution), `rpc.Connector` (pairs Client +
  Resolver per connection), `rpc.BidiStream`, `rpc.ClientStream`,
  `rpc.ServerStream` interfaces, `rpc.MethodKind` constants -- used by
  action/stream handlers for RPC method calls via Connection nodes
- `common/resolver.go`: `common.Resolver` interface (`FindMethodByName`) --
  minimal method-lookup contract used by `rpc.Connector.Resolver`
- `rpc/mock/mock.go`: in-memory `rpc.Client` + `common.Resolver` backed by
  registered functions; includes gRPC status error-producing methods
  (`google.golang.org/grpc/status`) for testing error paths
- `executor/nodes.go`: `NodeHandler` interface (marker sub-interfaces removed
  in Phase 10.1)

## Runtime Handler Pattern

- All handlers implement `NodeHandler.Run(ctx) error` (long-lived goroutine)
- Factory: `compileNode()` compiles CEL/retry/request into `compiled*` structs,
  then `newHandler()` wires them with pubsub/channels
- Transforms: pubsub-based pipeline (`Start()`, `runStep()`, `runSink()`)
- Activation uses `nodeToMap()` for CEL vars (typed `StateNode` → `map[string]any`)
- Command API (control plane): `Stop()`, `Terminate()`, `Suspend()`, `Resume()`,
  `StopNode()`, `TerminateNode()`, `SuspendNode()`, `ResumeNode()` on Executor.
  Commands are separate from the data plane (event streams). Per-node contexts
  enable individual node control. Run-state protected by mutex, cleared on
  Execute return.

## Unimplemented Proto Fields / Features

- **Actor attribution**: InteractionResponseEvent.actor field defined but not
  used in runtime (stored but not surfaced)

## CEL Type Checking

- Namespace variables (`inputs`, `vars`, `actions`, etc.) are `cel.DynType` --
  CEL does syntax checking at lint time but not type checking
- Type mismatches caught at eval time, not compile time
- Proto types ARE registered via `cel.Types(...)` so CEL knows the message
  structure, but the namespace maps themselves are untyped
- Future: declare typed map variables to get full compile-time type checking

---

# Why We Replaced Watermill

- Watermill's GoChannel dispatches via background goroutines with no ordering guarantee
- `BlockPublishUntilSubscriberAck` was a GoChannel-specific workaround that caused deadlocks in fan-out + multi-input graphs
- Watermill uses `[]byte` payloads -- pointless serialize/deserialize overhead when we pass typed protos in-process
- Go CDK (`gocloud.dev/pubsub`) evaluated and rejected: same `[]byte` problem, no outbox, no SQLite, no Valkey

---

# Two-Tier Architecture Detail

Both tiers persist runtime state. Cloud adds resumability.

## Local executor (`flowsdk/v1beta2/runtime/`)

- **PubSub**: Go channels (ephemeral, in-process)
- **Event log**: entstore outbox (ent + SQLite in CLI) records every committed `NodeEvent`
- **NOT resumable**: process dies -> inter-node gochannel messages lost, cannot
  re-drive the flow pipeline. Restart from scratch.
- **Capability**: history viewing -- `SnapshotAt(uid)` reconstructs state at any
  point from the outbox. Useful for debugging, visualization, post-mortem.
- **Outbox pattern**: executor opens ent tx -> UPDATE state -> ForwarderPublisher INSERT FlowEvent -> commit -> Forwarder relays outbox -> Go channels

## Cloud executor (`cloud-go/internal/flowsdk/`)

- **PubSub**: Kafka / Valkey Streams -- the broker IS the durable message log
- **Resumable**: durable broker retains inter-node messages; consumer offsets
  track where each subscriber left off; messages survive crashes
- **Transactional outbox**: state update + event append in one atomic ent tx (PostgreSQL) -> Forwarder relays to broker
- **Capability**: replay (re-process from any point), resume (continue from
  checkpoint), history viewing (same as local). This is the key differentiator.
- **Package**: cloud-specific backends (Kafka, Valkey) live in `cloud/` sub-packages

## Shared

- `pubsub/` interfaces: Message, Publisher, Subscriber, HandlerFunc, Middleware
- `pubsub/forwarder/`: ForwarderPublisher + Forwarder daemon (generic, backend-agnostic)
- Flat typed node protos + `NodeEvent`: the event payload -- same for both tiers
- State writes are executor logic (not pubsub) -- both tiers do it in the same tx as outbox publish

## Why Go channels aren't resumable

Making Go channels durable = writing every message to a log + tracking per-subscriber cursors + replaying on restart = reimplementing Kafka. Use a real broker instead.
