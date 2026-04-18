# modules/harness — WIP

> **WIP**: the Harness surface is in active flux. Only the
> `Instance` interface + `Event` / `EventType` set are frozen; the
> rest of this package is implementation detail and may change
> without deprecation.
>
> See [`brainkit-maps/brainkit/designs/09-harness-boundary.md`](../../../brainkit-maps/brainkit/designs/09-harness-boundary.md)
> for the boundary rules.

## What this gives you

- `NewModule(Config)` — a `brainkit.Module` that launches a Harness
  when the Kit boots.
- `(*Module).Instance() Instance` — the frozen consumer surface.
- `Instance` — the minimum set every release supports:
  `SendMessage`, `Abort`, `Steer`, `FollowUp`, `Subscribe`,
  `CurrentThread`, `CurrentMode`, `Close`.

## Frozen events

`Instance.Subscribe` delivers `Event` values whose `Type` is one of:

| EventType | Internal name |
|-----------|---------------|
| `EvAgentStart`    | `agent_start` |
| `EvAgentEnd`      | `agent_end` |
| `EvMessageDelta`  | `message_update` |
| `EvToolStart`     | `tool_start` |
| `EvToolEnd`       | `tool_end` |
| `EvError`         | `error` |

Events outside this set still flow through the same callback with
`Type` set to the raw internal string; consumers should treat
unknown values as opaque.

## Not frozen

Everything else: configuration structs, internal event types,
`HarnessEvent`, `DisplayState`, mode / model management methods,
subagent subsystem, observational memory, shell output events.
Expect these to change as multi-consumer use shakes the shape out.
