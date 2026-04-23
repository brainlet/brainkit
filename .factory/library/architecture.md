# brainkit Type-Alignment Mission — Architecture

This document describes how the system under test is put together, what a worker is changing, and what the invariants are. It is not a how-to; see worker skills for procedure.

## What brainkit is

brainkit is a Go library that embeds a QuickJS runtime running **TypeScript fixtures inside an SES Compartment**. Fixtures import mastra + ai-sdk code via bare specifiers `"agent"`, `"ai"`, and `"kit"`. The Go kernel transpiles TS → JS at deploy time (type-stripping only, `tsc --isolatedModules --noCheck` equivalent via `vendor_typescript`), strips ES imports, and evaluates inside the Compartment with brainkit-specific endowments.

## The three surfaces

1. **`internal/engine/runtime/agent.d.ts`** (3072 lines): flat ambient `declare module "agent" { ... }` block re-packaging every mastra sub-package (`@mastra/core/agent|tools|memory|workflows|rag|evals|voice|observability|mcp|...`, plus `@mastra/memory`, `@mastra/rag`, `@mastra/evals`, `@mastra/libsql`, `@mastra/pg`, `@mastra/upstash`, `@mastra/mongodb`, `@mastra/voice-*`, etc.).

2. **`internal/engine/runtime/ai.d.ts`** (1068 lines): flat ambient `declare module "ai" { ... }` block for the ai-sdk surface (`generateText`, `streamText`, `CallSettings`, `Message`, `ToolCall`, `FinishReason`, zod, provider-utils).

3. **`fixtures/ts/**/index.ts`**: ~289 TypeScript fixtures that import from `"agent"` / `"ai"` / `"kit"` and are deployed into the QuickJS Compartment by the Go runner in `test/fixtures/`.

Out of scope: `kit.d.ts`, `globals.d.ts`, `brainkit.d.ts`, `assemblyscript.d.ts`, the legacy v4 `internal/embed/ai/bundle/`, and all `vendor_*` directories.

## The validation pipeline (what a worker must make true)

Two independent gates must pass for every domain milestone to seal:

1. **Type gate** (`make type-check`): a single `tsc --noEmit -p fixtures/tsconfig.base.json` pass that type-checks the entire fixtures tree against the three `.d.ts` files. Empirically ~0.65s wall, ~190 MB RSS. Single pass gates all 289 fixtures at once; per-fixture tsconfig iteration produces identical output and is wasteful.

2. **Runtime gate** (`go test ./test/fixtures/ -run TestFixtures/<domain>`): exercises each fixture through the real brainkit kernel (deploy → transpile-strip → SES Compartment eval). Per-fixture `expect.json` sidecars carry the behavioral assertions; otherwise a fixture passes if deploy completes without throwing.

## Canonical truth hierarchy

When a worker needs to answer "what is the correct shape for symbol X":

1. **First preference: the mastra clone at `/Users/davidroman/Documents/code/clones/mastra`**, checked out to `@mastra/core@1.13.1` (or the closest matching tag batch from the 1.13.x series). Browse `packages/{core,memory,rag,evals,mcp,schema-compat,loggers,fastembed}/src/` for source.

2. **Second preference: the ai clone at `/Users/davidroman/Documents/code/clones/ai`**, checked out to `ai@6.0.x`. Browse `packages/{ai,provider-utils,openai,anthropic,google,...}/src/`.

3. **Third preference: pnpm-installed node_modules at `/Users/davidroman/Documents/code/brainlet/brainkit/internal/embed/agent/bundle/node_modules/<pkg>/dist/**/*.d.ts`**. This is the only source for packages that are NOT in the clones: `@mastra/libsql`, `@mastra/pg`, `@mastra/mongodb`, `@mastra/upstash`, `@mastra/voice-openai`, `@mastra/voice-deepgram`, `@mastra/voice-elevenlabs`, `@mastra/chroma`, `@mastra/pinecone`, `@mastra/qdrant`, `@mastra/observability`, and every `@ai-sdk/<provider>`.

## Invariants (never violate)

1. **Zero backward compatibility**: if a canonical signature changed, brainkit adopts it. No dual-shape declarations, no "accept both" aliases unless canonical itself does.

2. **Wrappers stay**: do not re-export, rename, or delete brainkit's module namespacing. `"agent"`, `"ai"`, `"kit"` remain the import specifiers fixtures use. Only the TYPES mapped underneath change.

3. **ai.d.ts targets ai-sdk v6 only**. The legacy `internal/embed/ai/bundle/` (v4) is out of scope; do not touch it and do not declare dual-version shapes in `ai.d.ts`.

4. **No edits to `kit.d.ts`, `globals.d.ts`, `brainkit.d.ts`, `vendor_quickjs/`, `vendor_typescript/`**, or `internal/embed/ai/bundle/`. These are off-limits for this mission.

5. **Fixture correctness trumps preservation**: if a fixture uses an API our types lied about, the fixture is updated to match the canonical shape (keeping runtime behavior). Do not preserve wrong fixtures to avoid type changes.

6. **No `as any` workarounds in fixtures**. If you hit one, the underlying type declaration is wrong — fix the type, not the fixture.

7. **No mocks, no fakes**: brainkit's testing convention. Fixture runtime uses real QuickJS + real mastra/ai-sdk JS + real containers for storage. Tests that need AI use real `OPENAI_API_KEY` from `.env`.

## Baseline type errors (to be recorded at M0)

Before M0 fixes anything, the M0 worker must capture the current `make type-check` output and append it to this file under a new section below. Downstream milestones use this baseline to prove progress; VAL-CROSS-005 requires all baseline errors to be eliminated by end of mission.

## Milestone boundaries

14 milestones, sequential. Each milestone reshapes a specific slice of `agent.d.ts` or `ai.d.ts`, updates/adds fixtures, and must pass both gates (type + runtime) before sealing.

- M0 — foundation: clones refresh, pnpm install, Makefile wiring, baseline capture
- M1 — tools
- M2 — agent-core
- M3 — memory
- M4 — workflow
- M5 — rag
- M6 — evals
- M7 — voice
- M8 — observability
- M9 — mcp
- M10 — processors
- M11 — vector
- M12 — ai-sdk
- M13 — coverage audit

Validation (`scrutiny-validator` + `user-testing-validator`) is auto-injected by the mission runtime at each milestone boundary; workers do not create validation features.

## Known pre-existing drift (confirmed in dry run)

- `fixtures/ts/voice/composite/basic/index.ts` and `fixtures/ts/agent/voice/composite/index.ts` use `speakProvider` / `listenProvider` keys on `CompositeVoice` — these are not canonical fields (canonical is `input` / `output` / `realtime`). Fix at M7.
- `fixtures/ts/memory/messages/save-and-recall/index.ts` uses extraneous `threadId` in `saveMessages`. Canonical `saveMessages({ messages })` does not accept `threadId`. Fix at M3.
- `fixtures/ts/memory/threads/management/index.ts` omits required `createdAt` / `updatedAt` on thread literals. Canonical `StorageThreadType` requires them. Fix at M3.
- `createTool` currently declared non-generic in agent.d.ts; canonical is 7-generic. Fix at M1.
- `Agent` generic constraint currently `TTools extends Record<string, Tool>`; canonical is `TTools extends ToolsInput` (heterogeneous). Fix at M2.
- `ConsoleLogger` declared `implements IMastraLogger`; canonical is `extends MastraLogger`. Fix at M8.

## Pinned versions (M0)

The two canonical clones are pinned at the following refs for the lifetime of this mission:

| Clone | Tag / ref | Commit SHA | `package.json` version |
| --- | --- | --- | --- |
| `/Users/davidroman/Documents/code/clones/mastra` | `@mastra/core@1.13.1` (monorepo batch tag) | `512a786a0cccc6ea3b36f2f33723570584235383` | `packages/core/package.json` → `1.13.1` |
| `/Users/davidroman/Documents/code/clones/ai` | `ai@6.0.168` (newest 6.0.x at M0 time) | `c38119a2e3df201a95a9979580f2c7a3c1b319ab` | `packages/ai/package.json` → `6.0.168` |

The repo-root `package.json` pins `typescript@5.9.3` as a devDependency. `node_modules/.bin/tsc --version` reports `Version 5.9.3`.

The bundle at `internal/embed/agent/bundle` was populated with `pnpm install --prefer-offline`; peer-dep warnings for `zod@^3` vs resolved `zod@4.1.12` and for voice packages pinning `@mastra/core@">=0.18.1-0 <0.25.0-0"` vs resolved `@mastra/core@1.26.0` are expected (the bundle uses caret ranges against `^1.13.1` that naturally float forward — the clones remain the canonical-shape source of truth at the pinned tags above).

## Baseline error snapshot

Captured at mission commit `c0d046cc72a0a0fdcf77a733c4a7e481c5a5409b` (HEAD prior to M0 implementation), with `typescript@5.9.3`, `tsc --noEmit -p fixtures/tsconfig.base.json`.

```
baseline type-check errors: 5
```

### Grouped listing

All 5 baseline errors match the pre-existing drift enumerated in the "Known pre-existing drift" section above. They are scheduled for resolution in M3 (memory) and M7 (voice).

- **M7 — voice (`CompositeVoice.speakProvider` / `listenProvider` drift)** — 2 errors
  - `fixtures/ts/agent/voice/composite/index.ts(7,3): error TS2353: Object literal may only specify known properties, and 'speakProvider' does not exist in type '{ input?: MastraVoice; output?: MastraVoice; realtime?: MastraVoice; }'.`
  - `fixtures/ts/voice/composite/basic/index.ts(9,40): error TS2353: Object literal may only specify known properties, and 'speakProvider' does not exist in type '{ input?: MastraVoice; output?: MastraVoice; realtime?: MastraVoice; }'.`
- **M3 — memory (`saveMessages` / `deleteMessages` extraneous `threadId`)** — 3 errors
  - `fixtures/ts/memory/messages/save-and-recall/index.ts(7,3): error TS2353: Object literal may only specify known properties, and 'threadId' does not exist in type '{ messages: Message[]; }'.`
  - `fixtures/ts/memory/threads/management/index.ts(40,28): error TS2353: Object literal may only specify known properties, and 'threadId' does not exist in type '{ messages: Message[]; }'.`
  - `fixtures/ts/memory/threads/management/index.ts(55,30): error TS2353: Object literal may only specify known properties, and 'threadId' does not exist in type 'string[] | { id: string; }[]'.`

Raw log retained at `/tmp/m0-baseline.log` for the duration of M0 validation.

## M1 — tools alignment

The tools block in `internal/engine/runtime/agent.d.ts` now mirrors `@mastra/core/tools/{tool.ts,types.ts}` 1:1:

- `createTool` — 7 generics (`TId`, `TSchemaIn`, `TSchemaOut`, `TSuspendSchema`, `TResumeSchema`, `TRequestContext`, `TContext`) returning `Tool<...>`. Defaults on `TSchemaIn` / `TSchemaOut` are `any` (canonical uses `unknown`); this is a deliberate narrow deviation because brainkit's `ai.d.ts` `z` stub is structural and does not carry `z.ZodType<T>` inference, so `unknown` would break the long-standing `async ({a,b}) => …` destructuring pattern in every fixture without explicit generics. Explicit generics on `createTool<Id, In, Out>` still narrow exactly as canonical does.
- `Tool` — 7-generic class implementing `ToolAction` with every canonical field (`id`, `description`, `inputSchema`, `outputSchema`, `suspendSchema`, `resumeSchema`, `requestContextSchema`, `execute`, `mastra`, `requireApproval`, `providerOptions`, `toModelOutput`, `mcp`, `onInputStart`, `onInputDelta`, `onInputAvailable`, `onOutput`, `inputExamples`, `mcpMetadata`). Schema slots use `import("ai").ZodType` (narrower than canonical `StandardSchemaWithJSON<T>` — preserves assignability to `ai.d.ts` `ToolDefinition.inputSchema`).
- `ToolAction` — structural interface with the canonical 7 generics. `ToolAction.execute` context is marked `context?` (deviation from canonical's `context: TContext`) to preserve assignment into the looser `ToolDefinition.execute?: (args, options?) => ...` used by `generateText({ tools })` in ai-sdk-v6. Runtime always passes context, so the optional marker is purely a type-surface accommodation.
- `ToolExecutionContext` — 3-generic (`TSuspend`, `TResume`, `TRequestContext`) with the canonical `agent?`, `workflow?`, `mcp?` nested slices and `writer?: ToolStream`.
- New ambient types surfaced for fixture coverage: `ToolStream` (class), `AgentToolExecutionContext`, `WorkflowToolExecutionContext`, `MCPToolExecutionContext`, `ValidationError`.
- `ToolsInput` is `Record<string, ToolAction<any,any,any,any,any> | VercelTool | VercelToolV5 | ProviderDefinedTool>`.

All 9 `fixtures/ts/tools/*` fixtures pass both gates after M1:

```
make type-check 2>&1 | grep -c 'fixtures/ts/tools.*error TS' → 0
go test ./test/fixtures/ -run 'TestFixtures/tools' -count=1 -timeout 600s → ok  (9 PASS)
```

Baseline is still 5 (pre-existing drift in memory + voice — unchanged, to be fixed at M3 / M7).

### M1 follow-up — `tool` endowment collision in kit_runtime.js

**Symptom:** `fixtures/ts/agent/tools/with-registered-tool/index.ts` failed at
deploy time with `TypeError: invalid 'in' operand` inside the Mastra
`prepare-tools-step` at `listAssignedTools` → `listTools` →
`ensureToolProperties` → `isVercelTool`. The `'parameters' in tool` check
was being applied to a string, which is invalid on primitives.

**Root cause:** The Compartment endowments built in
`internal/engine/runtime/kit_runtime.js` defined `tool` **twice** in the
same object literal:

1. Around line ~198: the kit surface `tool(name: string)` — resolves a
   Go-registered tool, parses its JSON-schema, and wraps it in
   `embed.createTool({...})` so downstream Mastra code receives a real
   `Tool` instance (passes `isMastraTool` via the shared
   `MASTRA_TOOL_MARKER` symbol).
2. Around line ~549: `tool: embed.tool` — AI SDK v6 authoring helper
   (literally `(x) => x`).

Because imports are stripped at transpile time and free identifiers
resolve against the single endowments object, the second `tool` key
silently shadowed the first. Every `import { tool } from "kit"` call
site therefore resolved to the AI SDK identity helper instead of the
kit resolver; `tool("multiply")` returned the string `"multiply"`,
which Mastra then tried to treat as a tool object — crashing as soon as
`isVercelTool` reached `'parameters' in tool` on a primitive.

The issue was pre-existing (the duplicate endowment predates M1), but
only surfaced now because earlier milestones did not exercise the
`agent/tools/with-registered-tool` fixture and M1's `-run TestFixtures/tools`
filter does not match `agent/tools/*`.

**Fix:** Consolidate the two endowments into a single discriminating
function at the original `tool:` slot:

```js
tool: function(nameOrDefinition) {
  if (typeof nameOrDefinition === "string") {
    // kit surface: resolve Go-registered tool by name
    // (parse info.inputSchema, wrap in embed.createTool)
  }
  // AI SDK surface: identity-ish pass-through via embed.tool
  return typeof embed.tool === "function" ? embed.tool(nameOrDefinition) : nameOrDefinition;
}
```

The later `tool: embed.tool` entry was removed and replaced with a
comment explaining the naming collision so future refactors don't
re-introduce it. Both call paths stay green:

- `agent/tools/with-registered-tool` (kit path, string arg) — PASS
- `ai/tool-authoring/tool-and-stops` (AI SDK path, object arg) — PASS

**Scope note:** this fix does not touch the M1 Tool / ToolAction type
shapes nor any off-limits file. It is a runtime-only
endowment-collision fix; the M1 canonical alignment remains intact.
`make type-check` reports the same 5 baseline errors (M3 memory + M7
voice drift) before and after.
