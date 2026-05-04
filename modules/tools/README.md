# modules/tools - stable

Tool registry command surface plus helper constructors for mounting typed Go
tools.

## Bus commands

- `tools.call` - call a registered tool by name.
- `tools.resolve` - return a tool schema/metadata record.
- `tools.list` - list registered tools, optionally by namespace.

## Go tools

`GoTool(name, TypedTool[T])` returns a hot-mountable module that registers one
typed Go tool. The mounted module ID is `go-tool:<name>`, and unmounting it
unregisters the tool.

## Capabilities

The command module requires `brainkit.core.tool_commands`. `GoTool` modules
provide a scoped tool resource for the registered tool.

## Runtime resources

The command module owns only `tools.*` command handlers. Each `GoTool` module
owns one `tool` resource and its tool registration lease.

## Hot unmount

Unmounting the command module unregisters `tools.*` handlers. Unmounting a
`GoTool` module unregisters that specific tool.
