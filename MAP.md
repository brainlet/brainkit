brainkit is a Go module organized around a public root package, reusable support packages, and implementation-only subsystems under internal/. Runtime assets, generated protocol types, experiments, fixtures, and in-tree plugins live alongside the library packages at the root.

## Root

> top-level directory structure and key files

- go.mod defines the module root
- the root package holds the public brainkit API surface and orchestration layer
- internal/ contains implementation-only packages organized by subsystem
- agent-embed/, harness/, transport/, bus/, registry/, jsbridge/, libsql/, mcp/, proto/, and sdk/ hold reusable support packages
- runtime/ holds runtime assets loaded by the library
- docs/ holds API reference material and guides
- plugins/ holds in-tree plugin packages
- testdata/ holds fixtures used by package, integration, and plugin tests
- experiments/ holds standalone probes and exploratory programs
- scripts/ holds repository helper scripts

## Documents

> project navigation and repository documentation

- MAP.md at the root covers the project structure described in this file
- CONVENTIONS.md at the root covers project-wide implementation conventions
- experiments/README.md covers the boundary and maintenance rules for the experiment tree
- scripts/test_non_experiments.sh is the canonical default verification entry point outside experiment-tagged coverage
- docs/api/ holds package and subsystem reference material
- docs/guides/ holds task-oriented guides

## Modules

> key packages and what they contain

- the root package contains kit construction, handlers, scaling, and compatibility aliases for selected public subpackages
- agent-embed/ exposes the public embedded agent client, sandbox, and tool types
- harness/ exposes the public harness runtime, config, event, display, and test surface
- transport/ exposes the public networking and transport surface
- bus/ is the message bus package
- registry/ is the tool registration and resolution package
- jsbridge/ is the QuickJS bridge layer
- libsql/ is the embedded LibSQL bridge package
- mcp/ is the MCP integration layer
- sdk/messages/ holds SDK message schema types
- proto/plugin/v1/ holds generated plugin service protocol types
- internal/transport/ holds transport implementations and peer discovery
- internal/network/ holds the inbound host server for kit-to-kit connections
- internal/plugin/ holds plugin process management and manifest wiring
- internal/wasm/ holds WASM compile, run, shard, host, and persistence internals
- internal/harness/ holds the harness runtime bridge layer
- internal/embed/agent/ holds embedded agent runtime assets and wrappers
- internal/embed/ai/ holds embedded AI runtime assets and client helpers
- internal/embed/compiler/ holds embedded compiler assets, bindings, and support tools

## Assets

> runtime assets, fixtures, and packaged examples

- runtime/wasm/ holds WASM runtime source assets
- testdata/ts/ holds TypeScript fixtures
- testdata/as/ holds AssemblyScript fixtures
- testdata/plugin/ holds plugin test fixtures
- plugins/brainkit-plugin-cron/ holds an in-tree plugin package
- experiments/lifecycle/ holds lifecycle and multicontext probes
- experiments/quickjs-*/ hold focused runtime and bridge experiments

## Boundaries

> package boundaries and dependency directions

- the root package depends on internal/ packages and selected sibling support packages for implementation details and compatibility aliases
- agent-embed/ is a public facade over internal/embed/agent for consumers that need direct agent and tool types
- harness/ owns the harness runtime and model surface; the root package keeps `Kit.InitHarness` plus compatibility aliases
- transport/ owns the public networking and transport surface; the root package keeps compatibility aliases for existing callers
- internal/network/ and internal/plugin/ depend on proto/plugin/v1/ for plugin and peer protocol types
- internal/wasm/ depends on internal/embed/compiler/ for compiler support and on runtime assets exposed at the root
- internal/embed/ packages provide shared embedded runtime and compiler substrate for the root package and internal subsystems
- experiments/, plugins/, and testdata/ sit outside the library import surface

RULES:
- packages under internal/ are implementation-only and are not part of the external import surface
- generated protocol types stay under proto/
- experiments/ and testdata/ are not package boundaries for the public library API
- tracked source packages under plugins/ and testdata/ are source-only; local built binaries are not part of the intended repository layout
- nested experiment modules under experiments/ are also source-only; their local compiled executables are not part of the intended repository layout
- experiments stay flat unless they graduate into reusable library code with a real package boundary
