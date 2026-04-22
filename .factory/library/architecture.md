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

## Baseline error snapshot

(To be filled in by the M0 worker after `make type-check` runs for the first time.)
