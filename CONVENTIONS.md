Technical conventions for brainkit — a Go module with a public root package, implementation-only subsystems under internal/, and embedded runtime assets for agents, AI, and WASM. These conventions cover package boundaries, subsystem façades, test layout, and structural refactor workflow.

## Stack

> languages, runtime substrate, and key dependencies

- the project is written in Go
- JavaScript execution uses QuickJS through jsbridge and the embedded runtime packages under internal/embed/
- WASM execution uses wazero and the embedded compiler assets under internal/embed/compiler/
- kit-to-kit networking and plugin RPC use gRPC with protocol types under proto/plugin/v1/
- message routing uses the bus package with in-process, gRPC, and NATS transports
- persistent local state uses SQLite-backed stores and the libsql bridge layer

RULES:
- generated protocol code stays under proto/
- embedded runtime assets stay under runtime/ or internal/embed/

## Package Boundaries

> public surface, reusable packages, and implementation-only subsystems

- the root package is the public API surface for kit construction, handlers, scaling, and compatibility wrappers over selected sibling packages
- reusable support packages live as sibling packages at the module root — agent-embed, harness, transport, bus, registry, jsbridge, libsql, mcp, proto, and sdk
- implementation-heavy subsystems live under internal/transport/, internal/network/, internal/plugin/, internal/wasm/, and internal/harness/
- same-package file splits follow subsystem roles — config, api, network, storage, stream, lifecycle, or handlers
- generated or mechanical binding code stays in the package owned by its generator or runtime substrate

RULES:
- external consumers do not import internal/ packages
- package extraction requires a real dependency or API boundary, not line count alone
- new public types and convenience APIs belong in the root package or a reusable sibling package

## Public API

> how the root package exposes subsystem capabilities

- the root package exposes thin façades and aliases for transport, WASM, plugin config, and selected harness surface types
- direct embedded-agent types are exposed through the public agent-embed sibling package instead of internal/embed/agent
- new public subsystem packages should own their implementation and tests in their own directory, with the root package keeping only the compatibility surface that is still justified
- configuration types stay grouped near the root package configuration surface
- root package files are named for the concern they expose — kit_network, kit_storage, transport_aliases, harness_aliases, and similar groupings
- root-level methods delegate to internal packages instead of duplicating subsystem logic

RULES:
- public façades stay consumer-facing and avoid carrying subsystem implementation details
- changes to public type locations preserve the external import surface unless the user explicitly asks for an API break

## Testing

> test placement, suite boundaries, and environment-sensitive coverage

- tests live next to the package or subsystem they exercise
- broader root-package tests are split by concern instead of accumulating in monolithic test files
- fixtures live under testdata/
- environment-sensitive suites use explicit build tags when they should not run by default — integration, e2e, experiment, and stress
- stress and recovery tests keep their historical names so regressions remain easy to compare across refactors

RULES:
- experiments/lifecycle tests keep the experiment build tag
- experiment modules stay source-only and flat unless they are promoted into reusable library code
- explicit stress suites keep the stress build tag, including stress-only cases split out of broader default test files
- new slow or environment-bound tests use an explicit tag or a skip path when external dependencies are missing
- structural refactors preserve existing test behavior unless the task is explicitly about changing behavior

## Workflow

> structural refactor process and repository maintenance expectations

- structural changes preserve behavior before changing semantics
- structural refactors are followed by gofmt, go build ./..., and focused regression slices around the touched subsystem
- broader verification runs exclude experiments unless the task is explicitly about experiments
- repository navigation documents are updated when package boundaries or document locations change
- experiment sandbox rules live in experiments/README.md and should be updated when experiment structure changes

WORKFLOW: structural-refactor
1. Identify a real package or file boundary
2. Move or split code without semantic changes
3. Run focused verification around the touched subsystem
4. Run a broader repository sweep when the environment allows it
5. Update MAP.md and CONVENTIONS.md when the structural boundary changes

RULES:
- do not reorganize code only to reduce line counts
- documentation updates follow structural changes in the same tranche
