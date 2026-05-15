# modules/browser - wip

Browser/CDP lifecycle owner for concrete Mastra browser providers.

This module does not claim support for `@mastra/agent-browser`,
`@mastra/stagehand`, or `@mastra/browser-viewer` by itself. It creates the
Brainkit-owned place those providers can bind to: local browser process launch,
CDP endpoint discovery, session tracking, scoped teardown, and lifecycle debug
counters.

## Bus commands

- `browser.session.launch` - launch a local CDP-capable browser.
- `browser.session.close` - close a tracked browser session.
- `browser.session.list` - list tracked browser sessions.

## Capabilities

- Provides: `brainkit.core.browser_manager`.
- Uses when present: `brainkit.core.lifecycle_debug_registry`.

## Runtime resources

- `browser.manager` - process lifecycle owner.
- `browser.sessions` - tracked browser/CDP sessions.

## Current Boundary

The module owns process/session cleanup. It does not yet inject this manager
into deployed TypeScript packages or emulate provider behavior. Real provider
support still requires package fixtures that import and execute the concrete
provider packages, plus live model/browser tests using the root `.env` where
model behavior is involved.

## Hot unmount

Unmounting closes every tracked browser session, terminates owned browser
processes, removes module-created temporary profile directories, unregisters
`browser.session.*` commands, and removes the
`brainkit.core.browser_manager` capability.
