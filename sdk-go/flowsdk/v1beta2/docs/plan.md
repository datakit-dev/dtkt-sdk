# Implementation Plan

> See `docs/architecture.md` for architecture decisions and rationale.
> Completed phases: `docs/plan-complete.md`.

## How to Execute This Plan

Each item below is designed to be implemented sequentially. For each:

1. **Review** -- read the description and relevant architecture sections.
2. **Implement** -- write code following existing Go conventions.
3. **Test** -- run `task test`. Write meaningful tests (behavior, edge cases,
   error paths).
4. **Verify** -- `task build` and `task test` pass, no regressions.
5. **Mark complete** -- move to `docs/plan-complete.md`.

---

## Tech Debt

- **Old `automations` edge on Flow**: still present alongside the new
  `flow_runs` edge. Remove once Automation migration is complete.

---

## Future

- **Periodic snapshots:** Add `snapshot bool` and `events_since_snapshot int`
  to RunSnapshot ent schema. Snapshot every N events, update `SnapshotAt` to
  use nearest checkpoint. See architecture.md "Periodic Snapshots for
  Efficient History Access".

- **Cloud entstore (dtkt-cloud-go):** Implement entstore with ent +
  PostgreSQL. Wire `outbox/outboxtest` conformance suite. Not in this repo --
  tracked here for dependency awareness only.

- **CloudEvents envelope:** Deferred until the external streaming backend
  interface is designed. Applies only when a flow's inputs/outputs connect to
  an external streaming backend (Kafka, NATS, webhooks). CloudEvents wraps
  `FlowEvent` at the boundary. Needs: external streaming backend interface
  design, then a CloudEvents serializer (`id`, `source`, `type`,
  `specversion`, `time`, `data: RunSnapshot_FlowEvent`).

- **Input behavior design:** Define clear rules for input modes and their
  interactions with the runtime. Currently inputs are always streaming, which
  makes one-shot/constant values awkward -- users must set `cache: true` and
  a throttle (or rely on `WithDefaultInputThrottle`) to prevent the default
  from overwriting a user-provided value on subsequent cycles. Design work
  needed:
  - **Input modes:** streaming (continuous values), cached (retain last
    value across throttle cycles), constant (set once, never re-evaluated).
  - **Cache vs. constant semantics:** `cache: true` retains the last value
    but still participates in throttle cycles. A true constant should be set
    once and never re-requested.
  - **Default interaction rules:** When does a type-level default apply?
    Only on first resolution? On every throttle miss? Should `cache` always
    take precedence over `default`?
  - **Throttle coupling:** Currently cache/default fallback is dead code
    without a throttle. Consider whether cache should imply a throttle, or
    whether constant inputs should bypass throttle entirely.
  - **Flow YAML ergonomics:** Make it obvious to flow authors which mode an
    input uses and what happens when a value isn't provided.
