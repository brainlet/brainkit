# CLI Test Map

**Purpose:** Verifies the brainkit CLI commands (version, init, new, deploy, send, eval, health, secrets) and bus-level command messages (kit.eval, kit.health, kit.send)
**Tests:** 11 functions across 2 files
**Entry point:** `cli_test.go` → `Run(t, env)`
**Campaigns:** none (CLI is standalone, not called by transport/fullstack campaigns)

## Files

### cobra.go — CLI command execution via Cobra root command

| Function | Purpose |
|----------|---------|
| testVersion | Runs `brainkit version` and asserts output contains "brainkit version" |
| testVersionJSON | Runs `brainkit --json version` and asserts output contains a JSON "version" key |
| testInit | Runs `brainkit init` in a temp dir and asserts brainkit.yaml is created |
| testNewModule | Runs `brainkit new module my-mod` and asserts manifest.json, hello.ts, tsconfig.json, and type declaration files are created |
| testNewPlugin | Runs `brainkit new plugin my-plug --owner testorg` and asserts main.go is created with the owner string |
| testFullWorkflow | Starts a brainkit instance, then exercises health, deploy, list, send, eval, secrets set/get/list/delete, teardown, and list again end-to-end |
| testSendWithAsyncHandler | Starts a brainkit instance, deploys a .ts with async bus.on handler using setTimeout, sends a message, and verifies the computed sum is returned |

### commands.go — Bus-level command messages (kit.eval, kit.health, kit.send)

| Function | Purpose |
|----------|---------|
| testKitEval | Publishes KitEvalMsg with JS expressions (arithmetic, object, async Promise) and verifies the evaluated results |
| testKitHealth | Publishes KitHealthMsg and verifies the response contains healthy=true and status="running" |
| testKitSendRequestReply | Deploys a .ts echo service, publishes KitSendMsg to its topic, and verifies the reply payload |
| testKitSendWithAwait | Deploys a .ts async compute service, publishes KitSendMsg, and verifies the async-computed sum |

## Cross-references

- **Campaigns:** none
- **Related domains:** bus (overlapping bus.on patterns), deploy (deploy lifecycle)
- **Fixtures:** none
