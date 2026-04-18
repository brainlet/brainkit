# harness-lite — WIP

> **The harness module is WIP.** Only the `Instance` interface plus
> the `Event` / `EventType` set shown below are frozen across
> releases. Everything else — `HarnessConfig` fields, `DisplayState`,
> subagents, observational memory, internal event types, display
> machinery — may change without deprecation.
>
> See
> [`modules/harness/README.md`](../../modules/harness/README.md)
> and the boundary design doc at
> `brainkit-maps/brainkit/designs/09-harness-boundary.md` for the
> rules.

## What this example shows

The minimum a consumer needs to wire the harness module and
subscribe to its stable event surface:

```go
harn := harness.NewModule(harness.Config{
    Harness: harness.HarnessConfig{
        ID: "demo",
        Modes: []harness.ModeConfig{{
            ID: "build", Default: true, AgentName: "demo-agent",
        }},
        Permissions: harness.DefaultPermissions(),
    },
})

kit, _ := brainkit.New(brainkit.Config{
    Modules: []brainkit.Module{harn},
    // ...
})
defer kit.Close()

inst := harn.Instance()           // frozen Instance surface
unsub := inst.Subscribe(func(ev harness.Event) { /* ... */ })
defer unsub()

_ = inst.SendMessage("hello")
```

## Frozen event types

`Instance.Subscribe` delivers `Event` values whose `Type` is one of
the six constants below. Events outside this set still reach the
callback with `Type` set to the raw internal string — consumers
should treat those as opaque.

| Go constant         | Wire name         | When it fires                                         |
|---------------------|-------------------|-------------------------------------------------------|
| `EvAgentStart`      | `agent_start`     | Agent begins processing a user message                |
| `EvAgentEnd`        | `agent_end`       | Agent finishes (normal completion or cancellation)   |
| `EvMessageDelta`    | `message_update`  | Assistant message content updated (streaming chunks) |
| `EvToolStart`       | `tool_start`      | Tool invocation begins                                |
| `EvToolEnd`         | `tool_end`        | Tool invocation completes                             |
| `EvError`           | `error`           | An error reached the harness event stream             |

Everything else — `tool_input_start`, `task_updated`, `shell_output`,
`subagent_start`, approval flows — is NOT frozen. Today they pass
through the same `Subscribe` callback with the raw type, but the
names and shapes are free to change.

## Frozen Instance surface

```go
type Instance interface {
    SendMessage(content string, opts ...SendOption) error
    Abort() error
    Steer(content string, opts ...SendOption) error
    FollowUp(content string, opts ...SendOption) error
    Subscribe(fn func(Event)) func()
    CurrentThread() string
    CurrentMode() string
    Close() error
}
```

Anything on the underlying `*Harness` struct beyond this interface
is implementation detail.

## Running

```sh
go run ./examples/harness-lite
```

Current output (JS backend not wired in this build):

```
Harness JS backend is not wired in this build.
The frozen Go-side contract (Module / Instance / Event) still compiles:
  module name        : harness
  module status      : wip (WIP)
  Instance interface : SendMessage / Abort / Steer / FollowUp / Subscribe / CurrentThread / CurrentMode / Close
  frozen event types : agent_start, agent_end, message_update, tool_start, tool_end, error
  boot error         : brainkit: module "harness" init: harness: create JS harness: ...
```

When the JS-side `__kit.createHarness` lands and an agent named
`demo-agent` is registered via `kit.register("agent", "demo-agent",
new Agent({...}))`, the example will transition to live mode: the
Instance resolves, Subscribe starts receiving events, and
`SendMessage` drives an actual agent run.

## What NOT to do

- Import anything from `modules/harness` besides `Module`,
  `Config`, `HarnessConfig`, `ModeConfig`, `Instance`, `Event`,
  `EventType`, the `Ev*` constants, `DefaultPermissions`, `Policy*`
  and `Category*`. The rest is moving.
- Rely on the shape of `Event.Payload` for non-frozen types —
  today a single raw JSON dump of the internal `HarnessEvent`;
  tomorrow it may be narrowed per event type.
- Build production dashboards off `DisplayState` / `ActiveTools` /
  `PendingApproval` — those belong to the inner Harness and are
  not part of the frozen surface.

## Upgrade path

When the Harness surface stabilizes (multi-consumer shake-out
completes and `modules/harness` drops the WIP banner), this example
will grow into a richer flow — threads, modes, tool approval. Until
then, the frozen `Instance` is the only safe contract.
