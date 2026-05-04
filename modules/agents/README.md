# modules/agents - stable

Agent registry command surface. This module exposes the runtime's registered
agents over the bus without exposing the underlying registry object.

## Bus commands

- `agents.list` - list known agents, optionally filtered.
- `agents.discover` - find agents by capability, model, or status.
- `agents.get-status` - read one agent's status.
- `agents.set-status` - update one agent's status.

## Capabilities

- Requires: `brainkit.core.agent_registry`.
- Provides: none.

## Runtime resources

None. The agent registry is owned by the Kit; this module only exposes command
handlers over it.

## Hot unmount

Unmounting the module unregisters the `agents.*` handlers and drops its registry
reference.
