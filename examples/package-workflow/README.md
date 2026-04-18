# package-workflow

The on-disk package lifecycle end to end: **scaffold → edit →
add a sibling file → deploy → call → teardown → redeploy**.

This is exactly what the `brainkit` CLI does under the hood
when you run:

```sh
brainkit new package greeter
# edit greeter/index.ts
brainkit deploy greeter/
```

The example unpacks those two commands into a single Go process
so you see every step, and leaves the scaffolded directory on
disk so you can open it in your IDE.

## Run

```sh
go run ./examples/package-workflow
```

Flags:

| Flag | Default | Effect |
|------|---------|--------|
| `-out`  | `./package-workflow-demo` | Scaffold destination — survives the process so you can inspect it |
| `-keep` | `true`                   | Keep the scaffold on disk after exit |

## What the example proves

1. **Scaffold**. `brainkit.ScaffoldPackage(dir, name, entry, source)`
   writes the same layout the CLI does:
   ```
   package-workflow-demo/
     manifest.json
     index.ts
     tsconfig.json           (paths mapped to ./types/*)
     types/
       kit.d.ts
       ai.d.ts
       agent.d.ts
       brainkit.d.ts
       globals.d.ts
   ```
   The package is ready for any TypeScript-aware IDE the second
   it's on disk — no `npm install`, no setup.

2. **Deploy from directory**. `brainkit.PackageFromDir(dir)` returns
   a `Package` the Kit can install. `kit.Deploy(ctx, pkg)` reads
   the manifest + bundles the sources + evaluates the entry in
   its own SES compartment.

3. **Edit + redeploy**. Rewrite the entry file on disk, call
   `kit.Deploy` again with the same `Package`. brainkit
   hot-replaces the compartment — subscriptions, registered
   agents/tools/workflows from the previous version are torn
   down, new ones take over.

4. **Sibling files**. Add a `greetings.ts` next to `index.ts`,
   `import { greeting } from "./greetings"` from the entry. The
   bundler follows the import graph, so every file referenced
   from the entry lands inside the same compartment and shares
   its endowments.

5. **Teardown**. `kit.Teardown(ctx, "greeter")` removes the
   package. Any subsequent call to `ts.greeter.*` errors out
   with a timeout (no handler).

## Expected output

```
[1/5] scaffolding package at /tmp/package-workflow-demo
          index.ts       (296 bytes)
          manifest.json  (69 bytes)
          tsconfig.json  (412 bytes)
          types/agent.d.ts …
[2/5] deploying PackageFromDir(...)
        ts.greeter.greet reply: Hello, alice!
[3/5] editing index.ts on disk (adding a 'facts' handler)
        ts.greeter.greet reply (edited): Hello, bob! The weather is fine.
        ts.greeter.facts reply:
          • brainkit packages live on disk as a plain directory.
          • tsconfig.json + types/ gives the IDE first-class autocomplete.
          • kit.Deploy(PackageFromDir(path)) ships whatever's on disk.
[4/5] adding a sibling file (greetings.ts) + updating index.ts to import it
        ts.greeter.greet reply (with sibling): Howdy, carol. …
[5/5] tearing down the deployment
        post-teardown call errored as expected: call timeout on ts.greeter.greet …

Package kept on disk at: /tmp/package-workflow-demo
```

## Open it in your IDE

After the example exits, `cd` into the scaffolded dir and open
it — `tsconfig.json` already points `"kit"`, `"ai"`, `"agent"`
at the bundled `types/*.d.ts`, so `import { bus } from "kit"`
gets full autocomplete out of the box. No npm, no setup.

## Using ScaffoldPackage from your own code

```go
import "github.com/brainlet/brainkit"

err := brainkit.ScaffoldPackage(
    "./my-greeter",      // dir — created if missing
    "greeter",           // package name written into manifest.json
    "index.ts",          // entry filename
    mySourceString,      // contents of the entry file
)
```

Options:

```go
brainkit.ScaffoldPackage(dir, name, entry, source, brainkit.ScaffoldOptions{
    Version:     "1.2.0",
    Description: "A tiny greeter with facts",
    Extra: map[string]string{
        "greetings.ts":        helperFileSource,
        "fixtures/samples.json": string(samplesJSON),
    },
    Overwrite: true, // wipe an existing dir; defaults to a safety error
})
```

`Extra` is for sibling source files / fixtures / config. The
scaffold-owned paths (`manifest.json`, `tsconfig.json`, the
entry file, `types/*.d.ts`) are off-limits — ScaffoldPackage
errors if `Extra` tries to overwrite them so you can't
accidentally break the IDE-facing layout.

## When to use which package constructor

| Helper | When |
|---|---|
| `brainkit.PackageInline(name, entry, source)` | One-off deploys, tests, examples where the source is a literal string in the Go file. No disk layout. |
| `brainkit.PackageFromFile(path)` | You have a single `.ts` file on disk but don't need the full manifest + types shape. Manifest is synthesized from the filename. |
| `brainkit.PackageFromDir(dir)` | You have a full scaffold on disk (this example). Manifest + tsconfig + types/ already exist. The shape `brainkit new package` produces. |

## Under the hood

- The Go `ScaffoldPackage` helper is the exact code the CLI's
  `brainkit new package` runs. One source of truth.
- The embedded `.d.ts` content comes from `brainkit.KitDTS`,
  `AiDTS`, `AgentDTS`, `BrainkitDTS`, `GlobalsDTS` — the same
  declarations the runtime ships as its reference corpus (see
  `brainkit.Reference()` / `examples/agent-forge`).
- Because the types are shipped with every scaffold, an upgrade
  of brainkit in your Go go.mod automatically refreshes every
  newly scaffolded package's types — the d.ts files are static
  at build time, not resolved from node_modules.
