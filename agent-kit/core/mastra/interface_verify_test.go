package mastra

// interface_verify_test.go — Compile-time verification that *Mastra satisfies
// all narrow Mastra interfaces defined across the codebase.
//
// Each package defines its own minimal Mastra interface to break circular
// import dependencies. These assertions ensure that when core.Mastra gains
// or changes methods, any interface mismatch is caught at compile time.
//
// When syncing with the TypeScript source:
//  1. Update core.Mastra methods as needed
//  2. Run `go test -c ./experiments/agent-kit/core/` — any interface
//     violations will show as compile errors
//  3. Update the consumer package's interface to match
//
// Status key:
//   VERIFIED  — compile-time assertion active
//   MISMATCH  — interface method signatures don't match *Mastra; details in comment
//   SKIP      — would create import cycle (package imports core)

import (
	"testing"

	// Packages whose Mastra/MastraRef interfaces are VERIFIED:
	agentpkg "github.com/brainlet/brainkit/agent-kit/core/agent"
	"github.com/brainlet/brainkit/agent-kit/core/action"
	"github.com/brainlet/brainkit/agent-kit/core/datasets/experiment/analytics"
	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
	looppkg "github.com/brainlet/brainkit/agent-kit/core/loop"
	"github.com/brainlet/brainkit/agent-kit/core/tools"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"

	// Packages whose Mastra interfaces have MISMATCHED signatures are imported
	// only for documentation — the assertions below are commented out.
	// If/when the interfaces are aligned, uncomment the assertion and move the
	// import above.
	//
	// "github.com/brainlet/brainkit/agent-kit/core/datasets"
	// "github.com/brainlet/brainkit/agent-kit/core/datasets/experiment"
	// "github.com/brainlet/brainkit/agent-kit/core/events"
	// "github.com/brainlet/brainkit/agent-kit/core/evals/scoretraces"
	// "github.com/brainlet/brainkit/agent-kit/core/loop/network"
	// "github.com/brainlet/brainkit/agent-kit/core/mcp"
	// "github.com/brainlet/brainkit/agent-kit/core/memory"
	// "github.com/brainlet/brainkit/agent-kit/core/observability"
	// "github.com/brainlet/brainkit/agent-kit/core/processors"
	// "github.com/brainlet/brainkit/agent-kit/core/server"
	// "github.com/brainlet/brainkit/agent-kit/core/workflows"
	// "github.com/brainlet/brainkit/agent-kit/core/workflows/evented"
	// "github.com/brainlet/brainkit/agent-kit/core/agent/workflows"
)

// ═══════════════════════════════════════════════════════════════════════════
// VERIFIED — *Mastra satisfies these interfaces
// ═══════════════════════════════════════════════════════════════════════════

// agent.Mastra (agent/types.go)
// Required: GetLogger() IMastraLogger  (where IMastraLogger = logger.IMastraLogger)
var _ agentpkg.Mastra = (*Mastra)(nil)

// action.MastraRef (action/action.go)
// Required: GetLogger() logger.IMastraLogger, GetStorage() *storage.MastraCompositeStore
var _ action.MastraRef = (*Mastra)(nil)

// tools.MastraRef (tools/types.go)
// Required: GetLogger() logger.IMastraLogger, GetStorage() *storage.MastraCompositeStore
var _ tools.MastraRef = (*Mastra)(nil)

// types.MastraRef (types/dynamic_argument.go)
// Required: GetLogger() logger.IMastraLogger
var _ aktypes.MastraRef = (*Mastra)(nil)

// loop.Mastra (loop/types.go)
// Required: GetLogger() IMastraLogger  (where IMastraLogger = logger.IMastraLogger)
var _ looppkg.Mastra = (*Mastra)(nil)

// llm/model.MastraRef (llm/model/model.go)
// Required: GetLogger() logger.IMastraLogger
var _ model.MastraRef = (*Mastra)(nil)

// datasets/experiment/analytics.Mastra (datasets/experiment/analytics/compare.go)
// Required: GetStorage() *storage.MastraCompositeStore
var _ analytics.Mastra = (*Mastra)(nil)

// ═══════════════════════════════════════════════════════════════════════════
// MISMATCH — *Mastra does NOT satisfy these interfaces (yet)
//
// Each comment explains the specific signature differences. These need to be
// resolved by either:
//   (a) updating the consumer interface to match core.Mastra, or
//   (b) adding/changing methods on core.Mastra, or
//   (c) replacing the local logger/storage stubs with real package aliases.
// ═══════════════════════════════════════════════════════════════════════════

// -----------------------------------------------------------------------
// mcp.Mastra (mcp/mcp.go)
// -----------------------------------------------------------------------
// MISMATCH: 4 method signature differences
//   - GenerateID() string
//       *Mastra has: GenerateID(ctx *IdGeneratorContext) string (extra param)
//   - AddTool(tool any, key string) error
//       *Mastra has: AddTool(tool ToolAction, key string) (typed param, no error return)
//   - AddAgent(agent any, key string) error
//       *Mastra has: AddAgent(agent Agent, key string, options *AddPrimitiveOptions) (typed param, extra param, no error return)
//   - AddWorkflow(workflow any, key string) error
//       *Mastra has: AddWorkflow(workflow AnyWorkflow, key string) (typed param, no error return)
//
// var _ mcp.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// agent/workflows.Mastra (agent/workflows/schema.go)
// -----------------------------------------------------------------------
// MISMATCH: 1 method signature difference
//   - GenerateID() string
//       *Mastra has: GenerateID(ctx *IdGeneratorContext) string (extra param)
//
// var _ agentworkflows.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// memory.Mastra (memory/memory.go)
// -----------------------------------------------------------------------
// MISMATCH: 1 method signature difference (type mismatch)
//   - GenerateID(ctx IdGeneratorContext) string
//       *Mastra has: GenerateID(ctx *IdGeneratorContext) string
//       Difference: memory uses a local IdGeneratorContext value type with
//       different field names (ResourceID vs ThreadID); core uses
//       *core.IdGeneratorContext (pointer to a different struct).
//
// var _ memorypkg.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// evals/scoretraces.Mastra (evals/scoretraces/score_traces.go)
// -----------------------------------------------------------------------
// MISMATCH: 1 method signature difference (return type mismatch)
//   - GetInternalWorkflow(id string) (InternalWorkflow, error)
//       *Mastra has: GetInternalWorkflow(id string) (AnyWorkflow, error)
//       Difference: scoretraces defines a local InternalWorkflow interface
//       (CreateRun with context.Context param) that differs from core.AnyWorkflow.
//   - GetLogger() logger.IMastraLogger — this one matches
//
// var _ scoretraces.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// datasets.Mastra (datasets/dataset.go)
// -----------------------------------------------------------------------
// MISMATCH: 6 method signature differences
//   - GetStorage() MastraCompositeStore
//       *Mastra has: GetStorage() *storage.MastraCompositeStore
//       Difference: datasets uses a local MastraCompositeStore interface
//       (GetStore(name string) any) vs the real *storage.MastraCompositeStore struct.
//   - GetScorerByID(id string) experiment.MastraScorer
//       *Mastra has: GetScorerByID(id string) MastraScorer (core.MastraScorer, different type)
//   - GetAgentByID(id string) (any, error)
//       *Mastra has: GetAgentByID(id string) (Agent, error) (typed return, not any)
//   - GetAgent(name string) (any, error)
//       *Mastra has: GetAgent(name string) (Agent, error) (typed return, not any)
//   - GetWorkflowByID(id string) (any, error)
//       *Mastra has: GetWorkflowByID(id string) (AnyWorkflow, error) (typed return, not any)
//   - GetWorkflow(name string) (any, error)
//       *Mastra has: GetWorkflow(id string) (AnyWorkflow, error) (typed return, not any)
//
// var _ datasetspkg.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// datasets/experiment.Mastra (datasets/experiment/types.go)
// -----------------------------------------------------------------------
// MISMATCH: Same 6 differences as datasets.Mastra above.
//   This is effectively a copy of the datasets.Mastra interface.
//
// var _ experiment.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// workflows.Mastra (workflows/types.go)
// -----------------------------------------------------------------------
// MISMATCH: 3 method signature differences
//   - GetStorage() Storage
//       *Mastra has: GetStorage() *storage.MastraCompositeStore
//       Difference: workflows uses a local Storage interface.
//   - GenerateID(opts GenerateIDOpts) string
//       *Mastra has: GenerateID(ctx *IdGeneratorContext) string (different param type)
//   - GetPubSub() events.PubSub
//       *Mastra has: PubSub() events.PubSub (method named PubSub, not GetPubSub)
//
// var _ workflowspkg.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// workflows/evented.Mastra (workflows/evented/step_executor.go)
// -----------------------------------------------------------------------
// MISMATCH: 3 method signature differences
//   - GetPubSub() PubSub
//       *Mastra has: PubSub() events.PubSub (different method name, local PubSub type)
//   - GetStorage() Storage
//       *Mastra has: GetStorage() *storage.MastraCompositeStore (local Storage type)
//   - GetLogger() logger.IMastraLogger — this one matches
//
// var _ evented.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// processors.Mastra (processors/types.go)
// -----------------------------------------------------------------------
// MISMATCH: 1 method return type mismatch
//   - GetLogger() MastraLogger
//       *Mastra has: GetLogger() logger.IMastraLogger
//       Difference: processors defines a local MastraLogger interface
//       (Debug/Info/Warn/Error only). Although logger.IMastraLogger has those
//       methods, Go requires identical return types for interface satisfaction.
//       Fix: replace local MastraLogger with `type MastraLogger = logger.IMastraLogger`
//
// var _ processorspkg.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// server.Mastra (server/types.go)
// -----------------------------------------------------------------------
// MISMATCH: 1 method return type mismatch
//   - GetLogger() MastraLogger
//       *Mastra has: GetLogger() logger.IMastraLogger
//       Same issue as processors.Mastra — local MastraLogger is not a type alias.
//       Fix: replace local MastraLogger with `type MastraLogger = logger.IMastraLogger`
//
// var _ serverpkg.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// observability.Mastra (observability/context.go)
// -----------------------------------------------------------------------
// MISMATCH: 1 method return type mismatch
//   - GetLogger() ObsLogger
//       *Mastra has: GetLogger() logger.IMastraLogger
//       Same pattern — local ObsLogger interface is not a type alias.
//       Fix: replace local ObsLogger with `type ObsLogger = logger.IMastraLogger`
//            or import logger and use the real type.
//
// var _ obspkg.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// loop/network.Mastra (loop/network/network.go)
// -----------------------------------------------------------------------
// MISMATCH: 1 method return type mismatch
//   - GetLogger() Logger
//       *Mastra has: GetLogger() logger.IMastraLogger
//       Same pattern — local Logger interface is not a type alias.
//       Fix: replace local Logger with `type Logger = logger.IMastraLogger`
//            or import logger and use the real type.
//
// var _ network.Mastra = (*Mastra)(nil)

// -----------------------------------------------------------------------
// events.MastraRef (events/processor.go)
// -----------------------------------------------------------------------
// MISMATCH: 1 method return type mismatch
//   - GetLogger() IMastraLogger
//       *Mastra has: GetLogger() logger.IMastraLogger
//       events defines a local IMastraLogger interface (Debug/Info/Warn/Error)
//       instead of using `= logger.IMastraLogger` type alias.
//       Fix: import logger and use `type IMastraLogger = logger.IMastraLogger`
//
// var _ events.MastraRef is not checked — events.MastraRef uses local IMastraLogger.

// ═══════════════════════════════════════════════════════════════════════════
// AgentRef, MastraMemoryRef — cross-type verification
// ═══════════════════════════════════════════════════════════════════════════
//
// action.AgentRef and tools.AgentRef are designed to be satisfied by the
// concrete agent.Agent struct, not by core's Agent interface stub. Since
// core.Agent is itself an interface (not a struct), we cannot meaningfully
// verify it here. The real verification belongs in the agent package's tests.
//
// action.MastraMemoryRef requires ID() string — this is designed to be
// satisfied by memory.MastraMemoryBase, not a core type.
//
// These are documented here for completeness but NOT asserted:
//
// var _ action.AgentRef = (core.Agent)(nil)       // core.Agent is an interface, not struct
// var _ tools.AgentRef = (core.Agent)(nil)         // same
// var _ action.MastraMemoryRef = (core.MastraMemory)(nil) // core.MastraMemory is an interface

// ═══════════════════════════════════════════════════════════════════════════
// Summary of gaps to close (priority order)
// ═══════════════════════════════════════════════════════════════════════════
//
// LOW-HANGING FRUIT (return type aliases — no logic change needed):
//   1. processors/types.go: change `type MastraLogger interface{...}` to
//      `type MastraLogger = logger.IMastraLogger`
//   2. server/types.go: same change for local MastraLogger
//   3. observability/context.go: change `type ObsLogger interface{...}` to
//      `type ObsLogger = logger.IMastraLogger`
//   4. loop/network/network.go: change `type Logger interface{...}` to
//      `type Logger = logger.IMastraLogger`
//   5. events/processor.go: change `type IMastraLogger interface{...}` to
//      import logger and use `type IMastraLogger = logger.IMastraLogger`
//
// MEDIUM (method naming / parameter alignment):
//   6. workflows/types.go & workflows/evented: rename GetPubSub() → PubSub()
//      to match core.Mastra.PubSub()
//   7. memory/memory.go: change GenerateID param from value
//      IdGeneratorContext to *core.IdGeneratorContext (or use type alias)
//   8. agent/workflows/schema.go: add *IdGeneratorContext param to GenerateID
//
// LARGER (structural mismatches):
//   9. mcp/mcp.go: align all 4 method signatures (GenerateID, AddTool,
//      AddAgent, AddWorkflow) with core.Mastra
//  10. datasets + datasets/experiment: align GetStorage return type and
//      scorer/agent/workflow return types with core.Mastra
//  11. workflows/types.go: align GetStorage return type and GenerateID param
//  12. evals/scoretraces: align GetInternalWorkflow return type with AnyWorkflow

// TestInterfaceVerification exists only to prevent the "no tests" warning.
// The real verification happens at compile time via the var _ assertions above.
func TestInterfaceVerification(t *testing.T) {
	t.Log("All compile-time interface assertions passed.")
}
