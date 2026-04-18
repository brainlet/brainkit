# workspace-agent

A coding agent that reads and writes real files on disk and runs
shell commands — all inside a sandboxed workspace rooted at the
Kit's `FSRoot`. The agent sees four tools:

| Tool | What it does |
|---|---|
| `list_files` | `fs.readdir("ws")` — returns filenames |
| `read_file`  | `fs.readFile("ws/<path>", "utf8")` |
| `write_file` | `fs.writeFile("ws/<path>", content)` |
| `execute_command` | `child_process.exec("cd ws && <command>")` — cwd rebased under `FSRoot` by the exec polyfill |

Under the hood the tools route through brainkit's `fs` + `exec`
polyfills (`internal/jsbridge/fs.go` + `exec.go`), which prefix
every relative path with the configured `FSRoot`. Commands and
file ops physically cannot escape the sandbox.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/workspace-agent
```

Expected tail:

```
[5/5] workspace after the run:
          COUNT.txt  (2 bytes)
          README.md  (59 bytes)
          TODO.md  (82 bytes)
          sample.ts  (296 bytes)
--- TODO.md ---
- TODO: add a subtract function
- TODO: write unit tests
- TODO: document the API
---
--- COUNT.txt ---
13
---
```

The agent executed four tasks:

1. listed the workspace,
2. read `sample.ts`, extracted its TODO comments, wrote `TODO.md`,
3. ran `wc -l sample.ts` via `child_process.exec`,
4. wrote the numeric output into `COUNT.txt`.

Every file operation landed on the real filesystem under the
Kit's `FSRoot/ws/` directory.

## Why custom `createTool` instead of Mastra's Workspace

The plan's original goal was to wire `Workspace({filesystem: new
LocalFilesystem(...), sandbox: new LocalSandbox(...)})` on the
Agent and let Mastra auto-inject its built-in tools
(`mastra_workspace_read_file`, `..._write_file`, etc). That path
uncovered three integration gaps:

1. **`fs.realpath`** wasn't exposed as a promisified method on
   `globalThis.fs`. `LocalFilesystem.init()` needs it for its
   setup sanity check. Fixed in this session by mirroring every
   `fs.promises.<name>` as `fs.<name>` (matches Node's dual
   callback + promise shape).
2. **`fs/promises` bundle stubs** hardcoded `throwFn` for
   `realpath`, `access`, `copyFile`, `rename`, `appendFile`,
   `symlink`, `readlink` instead of probing `globalThis.fs.X ||
   throwFn`. Fixed in this session.
3. **`execute_command`'s cwd** wasn't rebased under `FSRoot`,
   so a tool call with `cwd: "ws"` tried to `chdir` to the raw
   string relative to the Go process cwd — escaped the sandbox
   and failed with `ENOENT`. Fixed this session: the exec
   polyfill now takes a root arg and rebases relative cwds just
   like the fs polyfill.

With all three fixes in place, Mastra's auto-injected workspace
tools ALMOST work — `list_files`, `read_file`, and
`execute_command` round-trip cleanly. One cosmetic issue
remained: Mastra's `LocalFilesystem.writeFile` internal
implementation calls something that surfaces as
`not a function` on QuickJS. The example sidesteps it by
defining the four tools directly via `createTool`, which routes
through brainkit's polyfills without Mastra's LocalFilesystem
adapter — clean, predictable, and demonstrates the pattern any
consumer would actually use when building a coding agent.

## Polyfill fixes landed alongside this example

| File | Change |
|---|---|
| `internal/jsbridge/fs.go` | Expose promisified `readFile`, `writeFile`, `mkdir`, `realpath`, etc. at the top level of `globalThis.fs`, matching Node's dual callback + promise surface |
| `internal/embed/agent/bundle/build.mjs` | Bundle `fs` + `fs/promises` stubs check `F.<name>`/`T.<name>` before falling back to `throwFn` for every async method Mastra imports |
| `internal/jsbridge/exec.go` | `Exec(root)` factory rebases relative `cwd` under Kit `FSRoot` on spawn + exec paths |
| `internal/embed/agent/sandbox.go` | Pass `cfg.CWD` through to the exec polyfill |

All three fixes ship as part of the same PR so the workspace
pattern works end-to-end on a fresh clone. Bundle + bytecode
rebuilt per CLAUDE.md's 3-step protocol.

## Security boundary — what CAN'T the agent do?

- **Read / write outside `FSRoot`**: every `fs.*` call is rebased
  under the Kit's `FSRoot`. Absolute paths are cleaned + rebased
  too. `fs.readFile("/etc/passwd")` lands at `<FSRoot>/etc/passwd`
  (doesn't exist → ENOENT).
- **Exec outside the workspace**: every `child_process.*` call
  with a relative `cwd` rebases under the same root. Absolute
  `cwd`s pass through (dev choice); you can harden by setting
  `FSRoot` in a container + refusing absolute cwds in your
  custom tool wrapper.
- **Network**: workspace tools don't open sockets. If you want
  HTTP, add a `fetch` tool explicitly + scope the allowlist.

## Extension ideas

- **Persistent workspace**: point `FSRoot` at a long-lived dir
  instead of a tempdir. Then `list_files` across runs reveals
  prior work.
- **Shared workspace across agents**: multiple agents with the
  same `FSRoot` see the same files. Combine with `modules/harness`
  or the `agent-forge` pattern to give one agent a second agent
  as a tool.
- **Git-aware**: add a `git` tool that forwards to
  `child_process.exec("cd ws && git <args>")`. Use with a
  pre-init'd repo so the agent can branch / commit / diff.
- **Read-only tool**: drop `write_file` + `execute_command` to
  build a code-review agent that can only inspect.
- **Streaming shell**: for long-running builds, swap
  `child_process.exec` for `child_process.spawn` and stream
  stdout into `msg.send` chunks (see `examples/agent-stream/`).

## Comparison with agent-forge

`agent-forge` scaffolds a brand-new package on disk from an
architect's spec, then Go deploys it via `PackageFromDir`.
`workspace-agent` is the inverse: the agent already has a
workspace directory, and it edits + runs things inside it. The
two compose — a forge could output into a workspace-agent's
workspace for a "write, review, run, tweak" loop.

## See also

- `examples/package-workflow/` — scaffold + edit + redeploy
  lifecycle.
- `examples/agent-forge/` — forge a brand-new agent package.
- `internal/jsbridge/fs.go` + `exec.go` — the polyfills this
  example rides on.
