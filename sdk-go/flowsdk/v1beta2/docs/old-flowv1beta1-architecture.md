I added my CLI, SDK, and Core libraries to the VSCode workspace. This executor you have been working on was built as a sort of experimental v1beta2. It has LOTS of new features as compared to the existing flow executor but also lacks a lot of proper functionality.

There is a lot for you to look at but I want you to take note of the fact the existing (non-expiremental) executor supports a lot of important features that the new one currently does not, such as:
- An actual RPC interface for interacting with a flow
- Actual RPC method calling (via connections)
- proper proto registry handling (loaded from connections, etc.)
- connection handling
- the concept of a Flow and a FlowRun (Automation)

I'd like you to review and compare the two executors and identify the differences:
- naming
- functionality/features
- code structure
- etc.
- this should be in depth and comprehensive.

my goal is to eventually replace the old executor with the new one, but I want to make sure we don't lose any important functionality in the process.

I'm thinking we will add flowv1beta2 / flow2 packages where needed and the executor code will slowly be incrementally copied over to the right places.  Removing the old executor should be a smooth process.
Maybe we have dtkt flow2 command etc. until we finally drop the old one and rename everything.


Bellow find the existing (soon to be legacy) executor docs.

After reviewing everything generate a markdown doc next to plan that outlines the differences, gaps, etc.

---

# Flow Architecture

## Overview

A **Flow** is a declarative computation defined as a directed acyclic graph (DAG) of typed
nodes. Flows are authored as YAML/JSON specs conforming to the `flowv1beta1.Flow` protobuf
schema, stored and versioned in the cloud, and executed on-demand via **Automations**.

---

## Terminology

### Flow

A reusable blueprint that defines a computation's structure: its nodes, connections, inputs,
and outputs. Flows are stored centrally (cloud-go) and versioned with an etag for concurrency
control.

- Proto: `dtkt.flow.v1beta1.Flow`
- Proto definition: `dtkt-sdk/proto/dtkt/flow/v1beta1/spec.proto`

### Spec

The concrete schema for a flow definition (`v1beta1`). Contains the flow's name, description,
and lists of all node types: connections, inputs, vars, actions, outputs, and streams.

- Go type: `flowsdk.Spec` in `dtkt-sdk/sdk-go/flowsdk/spec.go`
- Wraps `flowv1beta1.Flow` with load/save/validate/parse-graph capabilities.

### Automation

A runnable **instance** of a Flow. It binds a flow reference (name + spec_etag) to runtime
configuration: connection mappings, input values, and a timeout. Automations track execution
state (PENDING -> RUNNING -> STOPPED/FAILED).

- Proto: `dtkt.core.v1.Automation`
- Proto definition: `dtkt-sdk/proto/dtkt/core/v1/messages.proto`
- States: `PENDING`, `RUNNING`, `STOPPED`, `FAILED`

### Runtime

The execution context that provides access to all runtime services during flow execution:
context cancellation, connectors (external integrations), CEL environment, node lookup, value
access, and inter-node communication channels.

- Interface: `shared.Runtime` in `dtkt-sdk/sdk-go/flowsdk/shared/runtime.go`
- Implementation: `runtime.Runtime` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/runtime.go`
- Proto state: `dtkt.flow.v1beta1.Runtime` in `dtkt-sdk/proto/dtkt/flow/v1beta1/eval.proto`

The `Runtime` interface exposes:

```go
type Runtime interface {
    Context() context.Context
    Connectors() ConnectorProvider
    Env() (Env, error)
    GetNode(string) (SpecNode, bool)
    GetValue(string) (any, error)
    GetSendCh(string) (chan<- ref.Val, error)
    GetRecvCh(string) (<-chan any, error)
}
```

### Executor

Orchestrates node execution over the compiled graph. Manages send/recv channels, triggers
evaluation cycles, tracks acknowledgments, and handles EOF retirement of streams.

- Type: `runtime.Executor` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/exec.go`

### Graph

The DAG representation of node dependencies. Built by walking CEL expression ASTs to discover
which nodes reference which other nodes, then applying topological sorting and transitive
reduction.

- Type: `runtime.Graph` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/graph.go`
- Proto: `dtkt.flow.v1beta1.Graph` in `dtkt-sdk/proto/dtkt/flow/v1beta1/eval.proto`
- Generic graph utility: `shared.Graph` in `dtkt-sdk/sdk-go/flowsdk/shared/graph.go`
- Graph library: `github.com/dominikbraun/graph`

### Env (CEL Environment)

Wraps Google's CEL engine with the flow's type system. Contains a custom type provider/adapter
for protobuf messages, the proto vars activation (the `Runtime` proto as context), and custom
flow functions.

- Interface: `shared.Env` in `dtkt-sdk/sdk-go/flowsdk/shared/runtime.go`
- Implementation: `runtime.Env` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/env.go`

### Connector / ConnectorProvider

Connectors adapt external integrations for use within a flow. A `ConnectorProvider` maps
connection node IDs to `Connector` instances, each of which provides a proto resolver (for
discovering available services/methods) and a gRPC dynamic client.

- Interfaces: `shared.ConnectorProvider`, `shared.Connector` in `dtkt-sdk/sdk-go/flowsdk/shared/connect.go`
- Implementation: `runtime.Connectors`, `runtime.Connector` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/connector.go`

### Resolver

A proto type resolver that provides access to service descriptors, method descriptors, file
descriptors, and a protovalidate validator. Used during compilation to resolve method calls and
during CEL environment setup for custom types.

- Interface: `shared.Resolver` in `dtkt-sdk/sdk-go/flowsdk/shared/connect.go`

---

## Node Types

All nodes implement `shared.ExecNode` with three optional execution modes:

```go
type ExecNode interface {
    Compile(Runtime) error
    Recv() (RecvFunc, bool)   // Receive external data
    Send() (SendFunc, bool)   // Emit values downstream
    Eval() (EvalFunc, bool)   // Pure evaluation
}
```

Node IDs are prefixed by type: `connections.X`, `inputs.X`, `vars.X`, `actions.X`,
`streams.X`, `outputs.X`.

### Connection

References an external integration by package identity or service list. Evaluated to
return the connection proto for use in CEL expressions.

- Proto: `dtkt.flow.v1beta1.Connection` in `dtkt-sdk/proto/dtkt/flow/v1beta1/spec.proto`
- Implementation: `spec.Connection` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/connection.go`
- Mode: **Eval only**

### Input

Entry point for external data. Supports typed values (bool, string, int, message, list, map,
etc.), optional defaults, caching, and validation.

- Proto: `dtkt.flow.v1beta1.Input` in `dtkt-sdk/proto/dtkt/flow/v1beta1/spec.proto`
- Implementation: `spec.Input` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/input.go`
- Type resolution: `spec.InputType` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/input_type.go`
- Validation: `spec.ValidateInput` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/input_validate.go`
- Mode: **Recv + Send** (receives external values, emits them as events)

### Var

Intermediate computed value. Either a CEL expression (`value`) or conditional logic
(`switch` with cases and a default). Supports caching.

- Proto: `dtkt.flow.v1beta1.Var` in `dtkt-sdk/proto/dtkt/flow/v1beta1/spec.proto`
- Implementation: `spec.Var` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/var.go`
- Switch logic: `spec.CompileSwitch` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/switch.go`
- Mode: **Eval only**

### Action

Side-effect operation. Either a unary RPC (`call`) to a connection's service or a user
interaction (`user`). Supports `run_if` guards, `on_error` recovery, and caching.

- Proto: `dtkt.flow.v1beta1.Action` in `dtkt-sdk/proto/dtkt/flow/v1beta1/spec.proto`
- Implementation: `spec.Action` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/action.go`
- RPC caller: `spec.NewCaller` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/caller.go`
- User action: `spec.CompileUserAction` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/user.go`
- Mode: **Eval only**

### Stream

Continuous data source. One of:

| Sub-type | Description | File |
|----------|-------------|------|
| **ServerStream** | One request, N responses | `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/stream_server.go` |
| **ClientStream** | N requests, one response | `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/stream_client.go` |
| **BidiStream** | N requests, N responses (decoupled) | `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/stream_bidi.go` |
| **Ticker** | Periodic timer emitting at `every` interval | `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/ticker.go` |

- Proto: `dtkt.flow.v1beta1.Stream` in `dtkt-sdk/proto/dtkt/flow/v1beta1/spec.proto`
- Factory: `spec.NewStream` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/stream.go`
- Mode: **Send** (emits values over time); some also have **Recv**

### Output

Terminal node that evaluates a CEL expression and logs the result.

- Proto: `dtkt.flow.v1beta1.Output` in `dtkt-sdk/proto/dtkt/flow/v1beta1/spec.proto`
- Implementation: `spec.Output` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/output.go`
- Mode: **Eval only**

---

## Lifecycle Phases

### 1. Spec Definition (Author Time)

A flow is authored as YAML/JSON conforming to the `flowv1beta1.Flow` proto schema. It
declares named node lists: connections, inputs, vars, actions, streams, outputs. CEL
expressions use the `= <expr>` prefix syntax.

**Files:**
- Proto schema: `dtkt-sdk/proto/dtkt/flow/v1beta1/spec.proto`
- Spec loader: `dtkt-sdk/sdk-go/flowsdk/spec.go` (`flowsdk.Spec`, `ReadSpec`, `WriteSpec`, `SpecLoader`)

### 2. Validation

The spec is validated via `protovalidate` against constraints defined in the proto schema
(`@required`, `@pattern`, etc.).

**Files:**
- `flowsdk.Spec.Validate()` in `dtkt-sdk/sdk-go/flowsdk/spec.go`

### 3. Parse Phase (CEL AST Generation)

All CEL expressions in the spec are parsed and type-checked, producing ASTs. No compilation
to programs yet -- this is syntax and type validation only. The expression prefix `= ` is
validated via regex `^\s?=\s?`.

For each node, `spec.ParseNode()` dispatches to the appropriate parse function
(`ParseVar`, `ParseAction`, `ParseMethodCall`, etc.), which calls `shared.ParseExpr()`
for each CEL expression field.

**Files:**
- Expression parsing: `shared.ParseExpr()` in `dtkt-sdk/sdk-go/flowsdk/shared/expr.go`
- Node dispatch: `spec.ParseNode()` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/node.go`
- Per-node parsers: `ParseVar`, `ParseAction`, `ParseSwitch`, `ParseMethodCall`, etc.

### 4. Graph Building

While parsing CEL expressions, an **AST visitor** (`GraphVisitor`) walks each expression to
discover node references. When it encounters a select expression like `inputs.foo` or
`vars.bar`, it creates a directed edge from that source node to the target node.

After all edges are added:
1. **Transitive reduction** minimizes redundant edges.
2. **Predecessor maps** (`forward`/`reverse`) are computed for dependency lookup.
3. **Topological grouping** groups independent nodes into parallel execution levels.

Cycle detection uses `graphlib.PreventCycles()` -- adding a cycle edge returns an error.

**Files:**
- Graph visitor: `runtime.GraphVisitor()` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/graph.go`
- Graph building: `runtime.NewGraph()`, `Graph.Build()`, `Graph.computeGroups()` in same file.
- Lint command (graph-only): `dtkt-cli/cmd/flow/lint.go` -- calls `spec.ParseGraph()`

### 5. CEL Environment Setup

A `runtime.Env` is created with:

- **Custom type provider/adapter** (`common.CELTypes`) -- handles protobuf message types.
- **Proto vars activation** (`cel.ContextProtoVars`) -- the `Runtime` proto is the context
  object, so CEL expressions can reference `connections.X`, `inputs.X`, `vars.X`, etc.
- **Container** set to `"dtkt"`.
- **Type abbreviations** for `Runtime_Done` and `Runtime_EOF`.
- **Custom functions** (see below).

**Files:**
- Env creation: `runtime.NewEnv()` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/env.go`
- CEL types: `common.NewCELTypes()`, `common.NewCELEnv()` in `dtkt-sdk/sdk-go/common/`

### 6. Compilation Phase

`NewExecutor(runtime, graph)` triggers compilation for every node. For each node:

1. **CEL expression compilation** -- each parsed AST is compiled to a `cel.Program`
   (optimized bytecode via `shared.CompileExpr()`).
2. **Method resolution** -- for action/stream nodes, `NewCaller()` resolves the connection,
   looks up the method descriptor via the connector's resolver, and creates the appropriate
   caller type (unary, server-stream, bidi-stream, etc.).
3. **Channel setup** -- `recvChs` and `sendChs` are created for nodes that have
   `Recv()`/`Send()` modes.
4. **Caching** -- nodes with `cache: true` are wrapped via `CacheableEval()` to memoize
   their first non-null result.

**Files:**
- Executor creation: `runtime.NewExecutor()` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/exec.go`
- Node compilation: `Node.Compile()` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/node.go`
- Expression compilation: `shared.CompileExpr()` in `dtkt-sdk/sdk-go/flowsdk/shared/expr.go`
- Caller factory: `spec.NewCaller()` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/caller.go`
- Caching wrapper: `spec.CacheableEval()` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/cache.go`

### 7. Execution Phase (Cycles)

The executor runs in a loop of **cycles**:

1. **Ready check** (`sendTriggersForReadyNodes`): determines which nodes can be triggered
   based on DAG predecessor completion, required input availability, and EOF state.

2. **Trigger emission**: emitter nodes (Input, Stream) receive a trigger on their `recvCh`,
   evaluate, and send their value to their `sendCh`.

3. **Value application**: when a send channel emits, the value is immediately applied to the
   runtime proto (`node.applyValue`), setting `curr_value` and `state = SUCCESS`. An ack is
   sent to `sendAckCh`.

4. **Cycle completion**: once all active (non-EOF) emitter nodes have acked, `readyCh` fires.
   The outer loop calls `Executor.Eval()`, which evaluates subscriber nodes (Var, Action,
   Output) in topological groups -- independent nodes within a group run in parallel.

5. **Reset**: `Executor.Reset()` clears node states (`prev_value` <- `curr_value`,
   `curr_value` <- nil, `state` <- UNSPECIFIED) and signals the ack loop to proceed with the
   next cycle.

6. **EOF handling**: when a stream emits `Runtime_EOF`, the node is retired (added to
   `eofIds`). The in-flight cycle is abandoned and restarted without that node.

**Node state transitions per cycle:**

```
UNSPECIFIED -> PENDING -> SUCCESS
                       -> ERROR
```

**Files:**
- Main loop: `Executor.Start()` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/exec.go`
- Trigger logic: `Executor.sendTriggersForReadyNodes()` in same file.
- Eval loop: `Executor.Eval()` in same file.
- Node startup: `Executor.startNodeExecution()` in same file.
- Runtime state: `Runtime.Reset()` in `dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime/runtime.go`

### 8. Termination

Execution stops when:
- All streams have sent EOF and all nodes have been evaluated.
- All required inputs have been received and processed.
- Context is cancelled (timeout, user interrupt, or `Runtime_Done`).

---

## CEL Expression System

### Expression Syntax

All CEL expressions in the spec use the prefix `= <expression>`:

```yaml
value: "= inputs.foo.getValue()"
run_if: "= vars.enabled && inputs.count > 0"
on_error: "= { status: 'failed' }"
```

### Parse vs Compile

- **Parse** (`shared.ParseExpr`): validates prefix, parses to AST, type-checks. Used during
  graph building (phase 3-4). No programs are created.
- **Compile** (`shared.CompileExpr`): parses + compiles to a `cel.Program` with interrupt
  check frequency. Used during compilation (phase 6).

### Custom CEL Functions

Defined in `dtkt-sdk/sdk-go/flowsdk/v1beta1/funcs/`:

| Function | Signature | Description | File |
|----------|-----------|-------------|------|
| `now()` | `-> timestamp` | Current timestamp | `funcs.go` |
| `getValue()` | `Node -> dyn` | Get node's current value | `value.go` |
| `getPrev()` | `Node -> dyn` | Get node's previous cycle value | `prev.go` |
| `getCount()` | `Node -> uint` | Get node's call count | `count.go` |
| `isEOF()` | `dyn -> bool` | Check if value is EOF sentinel | `eof.go` |

### CEL Activation Context

The `Runtime` proto is declared as the CEL context via `cel.DeclareContextProto`. This means
CEL expressions can directly reference top-level fields:

- `connections.X` -- connection node state
- `inputs.X` -- input node state
- `vars.X` -- var node state
- `actions.X` -- action node state
- `streams.X` -- stream node state
- `outputs.X` -- output node state

Each resolves to a `flowv1beta1.Node` proto, on which custom functions like `.getValue()`,
`.getPrev()`, `.getCount()` can be called.

---

## CLI Flow Commands

Located in `dtkt-cli/cmd/flow/`:

| Command | File | Description |
|---------|------|-------------|
| `flow` | `flow.go` | Parent command -- manage flow specs |
| `flow create` | `create.go` | Create a new flow from name or spec |
| `flow run` | `run.go` | Run a flow (creates an Automation, polls, streams I/O) |
| `flow lint` | `lint.go` | Validate a spec (parse + build graph) |
| `flow get` | `get.go` | Get a flow by name |
| `flow list` | `list.go` | List flows |
| `flow update` | `update.go` | Update a flow |
| `flow delete` | `delete.go` | Delete a flow |

### Run Command Flow

The `flow run` command (`run.go`) orchestrates the full lifecycle:

1. Resolve the core client and flow by name/path/URI.
2. Resolve connections (map node IDs to connection resources).
3. Resolve inputs (initial input values).
4. Create an `Automation` proto with flow ref, connections, inputs, timeout.
5. Call `CreateAutomation` RPC.
6. Poll until the automation is running (`ops.Poll`).
7. Stream I/O via `RunIO` (`io.go`) -- bidirectional event streaming for
   inputs, outputs, user actions.

### I/O Streaming

`RunIO` (`io.go`) sets up bidirectional event streaming between the CLI and the running
automation. It handles:
- Receiving output events and displaying them.
- Sending input values and user action responses.
- Terminal UI rendering via `runModel` (interactive TUI or noop for non-terminal).

**Files:**
- I/O orchestration: `dtkt-cli/cmd/flow/io.go`
- Run context: `dtkt-cli/cmd/flow/run_ctx.go`
- Run options: `dtkt-cli/cmd/flow/run_opts.go`
- Run model (TUI): `dtkt-cli/cmd/flow/run_model.go`, `io_model.go`

---

## Cloud Service

Located in `dtkt-cloud/cloud-go/service/flow/`:

| File | Description |
|------|-------------|
| `crud.go` | Flow CRUD operations (Create, Update, Delete with ent ORM) |
| `graph.go` | Graph retrieval from flow revision |
| `rule.go` | Authorization / business rules |
| `rest.go` | REST API handlers |
| `body.go` | Request body handling |

Flows are stored with revisions -- each update creates a new `FlowRevision` (via
`dtkt-cloud/cloud-go/service/flowrevision/`). The graph is computed from the spec at
revision time and stored on the revision entity.

---

## Proto State Structures

### `dtkt.flow.v1beta1.Runtime` (eval.proto)

The execution state container, used as the CEL activation context:

```protobuf
message Runtime {
  message Done { string id; string reason; bool is_error; }
  message EOF {}

  map<string, Node> connections = 1;
  map<string, Node> inputs = 2;
  map<string, Node> vars = 3;
  map<string, Node> actions = 5;
  map<string, Node> streams = 8;
  map<string, Node> outputs = 7;
}
```

### `dtkt.flow.v1beta1.Node` (eval.proto)

Execution state for any node:

```protobuf
message Node {
  enum State { STATE_UNSPECIFIED; STATE_PENDING; STATE_SUCCESS; STATE_ERROR; }

  string id = 1;
  State state = 2;
  uint64 call_count = 4;
  cel.expr.Value prev_value = 5;
  cel.expr.Value curr_value = 6;
  google.protobuf.Timestamp start_time = 20;
  google.protobuf.Timestamp finish_time = 21;

  oneof type {
    Connection connection; Input input; Var var;
    Action action; Output output; Stream stream;
  }
}
```

### `dtkt.flow.v1beta1.Graph` (eval.proto)

```protobuf
message Graph {
  repeated Node nodes = 1;
  repeated Edge edges = 2;
}

message Edge {
  string source = 1;  // e.g. "inputs.userId"
  string target = 2;  // e.g. "vars.computed"
}
```

---

## File Map

### SDK -- Flow SDK (`dtkt-sdk/sdk-go/flowsdk/`)

```
flowsdk/
  spec.go                    # Spec type, ReadSpec, WriteSpec, ParseGraph
  spec_test.go
  shared/
    connect.go               # ConnectorProvider, Connector, Resolver interfaces
    eval.go                  # EvalFunc, RecvFunc, SendFunc, Eval interface
    expr.go                  # ParseExpr, CompileExpr, expression validation
    graph.go                 # Generic Graph[T] with Build, Forward, Reverse, Starts, Ends
    node.go                  # SpecNode, ExecNode interfaces, node ID parsing
    runtime.go               # Runtime, Env interfaces
  v1beta1/
    funcs/
      funcs.go               # EnvOptions (registers all custom CEL functions)
      count.go               # getCount() -- node call count
      eof.go                 # isEOF() -- EOF sentinel check
      prev.go                # getPrev() -- previous cycle value
      value.go               # getValue() -- current value
    runtime/
      connector.go           # Connector, Connectors (ConnectorProvider impl)
      env.go                 # Env (CEL environment impl)
      exec.go                # Executor -- cycle-based orchestration
      exec_test.go
      graph.go               # Graph -- DAG, GraphVisitor, topological grouping
      graph_test.go
      node.go                # Node, NodeMap -- Parse, Compile, applyValue
      option.go              # Runtime/Env options
      runtime.go             # Runtime impl -- NewFromSpec, ProtoFromSpec, Reset
      user_action.go         # PendingUserAction
    spec/
      action.go              # Action node
      cache.go               # CacheableEval wrapper
      call.go                # CallNode, MethodCall parsing
      caller.go              # NewCaller factory -- dispatches to unary/stream callers
      connection.go          # Connection node
      input.go               # Input node (recv + send)
      input_type.go          # InputType -- typed value handling
      input_validate.go      # Input validation
      node.go                # ParseNode factory, GetID, GetIDPrefix
      output.go              # Output node
      stream.go              # Stream factory (call -> StreamCall, generate -> Ticker)
      stream_bidi.go         # BidiStream
      stream_client.go       # ClientStream
      stream_server.go       # ServerStream
      switch.go              # Switch (conditional var logic)
      ticker.go              # Ticker (periodic stream)
      user.go                # UserAction (interactive actions)
      var.go                 # Var node
```

### SDK -- Proto Definitions (`dtkt-sdk/proto/dtkt/flow/v1beta1/`)

```
eval.proto                   # Runtime, Graph, Edge, Node (execution state)
spec.proto                   # Flow, Connection, Input, Var, Action, Output, Stream, MethodCall, etc.
type.proto                   # Flow type definitions (Bool, String, Message, List, Map, etc.)
```

### Cloud (`dtkt-cloud/cloud-go/`)

```
service/flow/
  crud.go                    # Flow CRUD (Create, Update, Delete)
  graph.go                   # Graph retrieval from revision
  rule.go                    # Authorization rules
  rest.go                    # REST handlers
  body.go                    # Request body handling
internal/flowutil/
  graph.go                   # Graph/Node/Edge display types (for API responses)
```

---
---

# v1beta1 -> v1beta2 Migration: Comparison & Status

## Naming Changes

| Concept | v1beta1 | v1beta2 |
|---------|---------|---------|
| Resource instance | `Automation` | `FlowRun` |
| CLI command | `dtkt flow run` | `dtkt flow2 run` |
| State proto | `flowv1beta1.Runtime` | `flowv1beta2.RunSnapshot` |
| State per node | `flowv1beta1.Node` (generic, `oneof type`) | Flat typed: `RunSnapshot_InputNode`, `RunSnapshot_VarNode`, etc. |
| RPC service | `AutomationService` | `FlowRunService` |
| Node event | N/A (monolithic snapshot) | `RunSnapshot_NodeEvent` (per-node event) |
| Stream types | `streams` (ticker is a stream) | `generators` (range/ticker/cron) + `streams` (RPC only) |
| User actions | `actions.user` variant | `interactions` (first-class node type) |
| Error handling | `on_error` CEL expr | `retry` strategy (when/skip_when/suspend_when/terminate_when/backoff) |

## Execution Model

| Aspect | v1beta1 | v1beta2 |
|--------|---------|---------|
| Concurrency | Cycle-based: Ready → Eval → Reset loop | PubSub-based: one goroutine per node, concurrent |
| Inter-node comms | Go channels (sendCh/recvCh) | PubSub topics (gochannel local, Kafka/NATS cloud) |
| State tracking | Monolithic `Runtime` proto (all nodes) | Per-node `NodeEvent` messages + `RunSnapshot` reconstructed |
| Eval trigger | `Executor.Ready()` signal → `Eval()` batch | Each node handler loops on its subscription independently |
| EOF handling | `eofIds` set, abandon + restart cycle | Each handler returns on EOF; errgroup collects |
| Entry point | `exec.Start()` → `exec.Ready()` → `exec.Eval()` loop | `exec.Execute(ctx, graph)` -- blocks until done |
| Runtime state | `runtime.Proto()` returns full snapshot each cycle | Outbox collects `NodeEvent`s; `SnapshotAt(uid)` replays |

## Feature Comparison

### v1beta2 has, v1beta1 lacks

- **PubSub architecture**: pluggable message broker (gochannel, Kafka, Valkey Streams, NATS)
- **Outbox pattern**: atomic state + event writes in same DB transaction
- **History viewing**: `SnapshotAt(uid)` reconstructs state at any event
- **Cloud resume**: rebuild from outbox event log (planned, cloud-only)
- **Generators**: `range` (finite), `ticker` (periodic), `cron` (schedule) -- separate from streams
- **Interactions**: first-class node type with token-based request/response
- **Transform pipeline**: `map`/`filter`/`flatten`/`reduce`/`scan` on any node
- **Retry strategy**: per-action `when`/`skip_when`/`suspend_when`/`terminate_when` + exponential backoff
- **Error strategy**: flow-level `TERMINATE`/`STOP`/`CONTINUE`
- **Suspend/Resume**: per-node and flow-level (`Suspend()`, `Resume()`, `SuspendNode(id)`, `ResumeNode(id, val)`)
- **Flat typed CEL**: `inputs.x.closed`, `generators.seq.eval_count` -- no `.getValue()` indirection
- **Per-node lifecycle control**: `StopNode(id)`, `TerminateNode(id)` independent of flow
- **Input throttle**: rate-based input pacing (interval/count)
- **Compiled request trees**: structured request building from CEL + literal leaves

### v1beta1 has, v1beta2 NOW has (closed gaps)

- **RPC method calling**: v1beta2 has `rpc.Client` / `rpc.Resolver` interfaces with
  unary, server-stream, client-stream, bidi-stream handler types
- **Connection handling**: v1beta2 nodes reference connections by ID; RPC resolution
  uses `rpc.Resolver.LookupMethod()` to dispatch to the right handler type
- **Proto registry**: v1beta2 uses `rpc.Resolver` which wraps proto descriptor resolution
- **Graph building**: `flowgraph.Build()` in `sdk-go/flowsdk/v1beta2/graph/`
- **Spec validation/lint**: `Spec.Validate()` and `Spec.Lint()` in `sdk-go/flowsdk/v1beta2/`
- **CLI CRUD**: `dtkt flow2 create/get/list/update/delete` -- all implemented
- **Schema serving**: v1beta2 `Flow` JSON schema served at `/schemas/types/...`

### v1beta1 has, v1beta2 implementation IN PROGRESS

- **FlowRun handler (server-side)**: `FlowRunServiceHandler` registered as unimplemented
  stub. The CLI `flow2 run` command calls `CreateFlowRun` which returns `unimplemented`.
  Need to implement: handler.go + operation.go + run.go (analogous to automation package).
  **Status**: §9.5 in plan -- actively being implemented.

- **Connection resolution (server-side wiring)**: The v1beta2 executor has `rpc.Client` and
  `rpc.Resolver` interfaces, but the CLI daemon needs to wire these to actual connections.
  In v1beta1, `automation/helper.go` has `GetRegistryPool()` and `GetConnectors()` which:
  1. Look up `Connection` resources from the DB by name
  2. Auto-start deployments (integrations) if needed
  3. Create proxy clients for each connector
  The v1beta2 handler needs equivalent wiring: resolve `FlowRun.connections` map entries
  to actual `rpc.Client`/`rpc.Resolver` instances via the daemon's proxy infrastructure.

- **Event streaming (server-side)**: `StreamFlowRunEvents` bidi-stream, `ReceiveFlowRunEvents`
  server-stream, `SendFlowRunEvent` unary. The CLI `io.go` is fully implemented and expects
  these RPCs to work. The server handler needs to bridge executor pubsub events to/from
  the Connect RPC streams.

- **Reconcile on boot**: v1beta1 boots RUNNING automations on daemon startup via
  `automate.Reconcile(ctx)`. FlowRun handler needs equivalent to re-attach to running flows.

## CLI Integration Status

| Component | v1beta1 (`cmd/flow/`) | v1beta2 (`cmd/flow2/`) | Status |
|-----------|----------------------|----------------------|--------|
| `create` | `create.go` | `create.go` | ✅ Working |
| `get` | `get.go` | `get.go` | ✅ Working |
| `list` | `list.go` | `list.go` | ✅ Working |
| `update` | `update.go` | `update.go` | ✅ Working |
| `delete` | `delete.go` | `delete.go` | ✅ Working |
| `lint` | `lint.go` | `lint.go` | ✅ Working |
| `run` | `run.go` → `io.go` | `run.go` → `io.go` | 🔄 CLI implemented, server stub |
| Handler | `automation/` | (TBD `flowrun/`) | 🔄 In progress |
| Graph compile | `flow/handler.go` | `flow/handler.go` | ✅ v1beta2 case added |
| Schema serve | `types/syncer.go` | `types/adapter.go` | ✅ `syncFlowSpecV1Beta2()` |

## Server Handler Architecture: v1beta1 Automation vs v1beta2 FlowRun

### v1beta1 Pattern (reference implementation)

```
internal/core/automation/
  handler.go   -- AutomationServiceHandler (CRUD + event streaming)
  helper.go    -- DB queries, ToProto, GetRegistryPool, GetConnectors, GetInputs
  operation.go -- Long-running operation wrapper (boot/create/update/delete)
  run.go       -- Execution engine: 3 goroutines (executor, snapshot, events)
```

Key design:
- `SyncMap[uuid.UUID, *Run]` tracks active runs in memory
- `Run` owns: `flowruntime.Executor`, `flowruntime.Runtime`, event channels
- Three goroutines: `runExecutor` (eval loop), `runSnapshot` (output collection),
  `runEvents` (input routing)
- LRU cache (100 entries) + per-subscriber notification for event delivery
- `Operation` wraps state transitions via `longrunning.Operation`

### v1beta2 Target Pattern

```
internal/core/flowrun/
  handler.go   -- FlowRunServiceHandler (CRUD + event streaming)
  helper.go    -- DB queries, ToProto, connection/input resolution
  operation.go -- Long-running operation wrapper
  run.go       -- Execution bridge: v1beta2 executor + pubsub event delivery
```

Key differences from v1beta1:
- v1beta2 executor uses `Execute(ctx, graph)` -- blocks until done (no Start/Eval/Ready loop)
- Events flow through pubsub topics instead of snapshot channels
- Outputs received by subscribing to `output.{nodeID}` topics
- Inputs sent by publishing to `input.{nodeID}` topics
- Interactions use dedicated channels (`WithInteractions(prompt, response)`)
- Flow control via `Stop()`, `Terminate()`, `Suspend()`, `Resume()` methods
- Outbox + forwarder for durable event log (even locally with SQLite)

### Connection Wiring Path

v1beta1 path (working):
```
Automation.connections (map[string]*FlowConnectionMetadata)
  → GetRegistryPool(ctx, auto) → coreclient.RegistryPool
  → GetConnectors(ctx, pool, auto) → flowruntime.Connectors
  → runtime.NewFromProto(ctx, cancel, runtimeProto, WithConnectors(conns))
```

v1beta2 target path:
```
FlowRun.connections (map[string]*FlowConnectionMetadata)
  → resolve connection resources from DB (same as v1beta1)
  → auto-start deployments if needed (same as v1beta1)
  → create rpc.Client + rpc.Resolver per connection
  → runtime.NewExecutor(pubsub, WithRPCClient(client), WithResolver(resolver))
```

The `rpc.Client` interface in v1beta2 maps to the proxy client that the daemon
already provides. The `rpc.Resolver` interface maps to the proto descriptor resolver
loaded from connection registries.

### CLI (`dtkt-cli/cmd/flow/`)

```
flow.go                      # Parent command
create.go                    # flow create
run.go                       # flow run (Automation lifecycle)
lint.go                      # flow lint (parse + graph)
get.go                       # flow get
list.go                      # flow list
update.go                    # flow update
delete.go                    # flow delete
io.go                        # RunIO -- bidirectional event streaming
io_model.go                  # I/O model interface
run_ctx.go                   # Run context with cancel-cause
run_model.go                 # TUI model for run
run_opts.go                  # Run options (timeout, throttle, etc.)
helper.go                    # Shared helpers (GetWithClient, ResolveConnections, etc.)
spec_util.go                 # Spec resolution utilities
```
