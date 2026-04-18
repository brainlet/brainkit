# CLI Test Map

**Purpose:** Verifies the five-verb brainkit CLI (`start`, `deploy`,
`call`, `inspect`, `new`) via Cobra programmatic execution + bus-
level command messages (kit.eval, kit.health, kit.send).
**Tests:** 14 functions across 2 files.
**Entry point:** `cli_test.go` → `Run(t, env)`
**Campaigns:** none (CLI is standalone, not called by transport /
fullstack campaigns).

## Files

### cobra.go — CLI command execution via Cobra root command

| Function | Purpose |
|----------|---------|
| testVersion | Runs `brainkit version` and asserts output contains "brainkit version" |
| testVersionJSON | Runs `brainkit --json version` and asserts output contains a JSON "version" key |
| testNewPackage | Runs `brainkit new package my-pkg --dir <tmp>` and asserts manifest.json, index.ts, tsconfig.json, and type declaration files are created |
| testNewPlugin | Runs `brainkit new plugin my-plug --owner testorg` and asserts main.go is created with the owner string |
| testNewServer | Runs `brainkit new server my-srv --dir <tmp>` and asserts main.go / brainkit.yaml / go.mod / README.md are created and reference the `github.com/brainlet/brainkit/server` package |
| testInspectHealth | Boots a test server and asserts `brainkit inspect health --endpoint <addr>` renders a STATUS column |
| testInspectHealthJSON | Same server; `--json inspect health` returns a parseable JSON payload |
| testCallVerb | Runs `brainkit call kit.health --payload '{}'` against the test server; asserts the reply is JSON |
| testDeployVerb | Deploys a tiny .ts file via `brainkit deploy <file> --endpoint <addr>`; asserts `inspect packages` then lists it |
| testDeployFullWorkflow | End-to-end: deploy a .ts echo handler, call it via `brainkit call ts.echo-cli.ping`, verify the payload round-trips |

### commands.go — Bus-level command messages (kit.eval, kit.health, kit.send)

| Function | Purpose |
|----------|---------|
| testKitEval | Publishes KitEvalMsg with JS expressions (arithmetic, object, async Promise) and verifies the evaluated results |
| testKitHealth | Publishes KitHealthMsg and verifies the response contains healthy=true and status="running" |
| testKitSendRequestReply | Deploys a .ts echo service, publishes KitSendMsg to its topic, and verifies the reply payload |
| testKitSendWithAwait | Deploys a .ts async compute service, publishes KitSendMsg, and verifies the async-computed sum |

## Cross-references

- **Campaigns:** none
- **Related domains:** bus (overlapping bus.on patterns), deploy
  (deploy lifecycle), gateway (bus_api routes drive every CLI verb)
- **Fixtures:** none
