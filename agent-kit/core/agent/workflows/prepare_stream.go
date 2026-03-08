// Ported from: packages/core/src/agent/workflows/prepare-stream/index.ts
package workflows

// CreatePrepareStreamWorkflowOptions holds all options for creating the
// prepare-stream workflow that orchestrates tool preparation, memory setup,
// result mapping, and LLM streaming.
type CreatePrepareStreamWorkflowOptions struct {
	Capabilities           AgentCapabilities
	Options                InnerAgentExecutionOptions
	ThreadFromArgs         *StorageThreadType
	ResourceID             string
	RunID                  string
	RequestContext         RequestContext
	AgentSpan              Span
	MethodType             AgentMethodType
	Instructions           SystemMessage
	MemoryConfig           MemoryConfig
	Memory                 MastraMemory
	ReturnScorerData       bool
	SaveQueueManager       SaveQueueManager
	RequireToolApproval    bool
	ToolCallConcurrency    int
	ResumeContext          *ResumeContext
	AgentID                string
	AgentName              string
	ToolCallID             string
	Workspace              Workspace
}

// PrepareStreamWorkflow holds the assembled workflow steps.
// In TypeScript, this returns a Workflow instance built with createWorkflow().
// In Go, the steps are exposed directly for the caller to orchestrate.
type PrepareStreamWorkflow struct {
	PrepareTools  func() (*PrepareToolsStepOutput, error)
	PrepareMemory func() (*PrepareMemoryStepOutput, error)
	MapResults    func(*PrepareToolsStepOutput, *PrepareMemoryStepOutput) (*ModelLoopStreamArgs, error)
	Stream        func(any) (MastraModelOutput, error)
}

// CreatePrepareStreamWorkflow creates the prepare-stream workflow.
// The workflow is structured as:
//
//	parallel([prepareToolsStep, prepareMemoryStep])
//	  .map(mapResultsStep)
//	  .then(streamStep)
//
// The prepare-tools and prepare-memory steps run in parallel, their outputs
// are mapped into ModelLoopStreamArgs by the map-results step, and then
// the stream step initiates the LLM streaming loop.
func CreatePrepareStreamWorkflow(opts CreatePrepareStreamWorkflowOptions) *PrepareStreamWorkflow {
	prepareToolsStep := CreatePrepareToolsStep(PrepareToolsStepOptions{
		Capabilities:   opts.Capabilities,
		Options:        opts.Options,
		ThreadFromArgs: opts.ThreadFromArgs,
		ResourceID:     opts.ResourceID,
		RunID:          opts.RunID,
		RequestContext: opts.RequestContext,
		AgentSpan:      opts.AgentSpan,
		MethodType:     opts.MethodType,
		Memory:         opts.Memory,
	})

	prepareMemoryStep := CreatePrepareMemoryStep(PrepareMemoryStepOptions{
		Capabilities:   opts.Capabilities,
		Options:        opts.Options,
		ThreadFromArgs: opts.ThreadFromArgs,
		ResourceID:     opts.ResourceID,
		RunID:          opts.RunID,
		RequestContext: opts.RequestContext,
		MethodType:     opts.MethodType,
		Instructions:   opts.Instructions,
		MemoryConfig:   opts.MemoryConfig,
		Memory:         opts.Memory,
	})

	streamStep := CreateStreamStep(StreamStepOptions{
		Capabilities:           opts.Capabilities,
		RunID:                  opts.RunID,
		ReturnScorerData:       opts.ReturnScorerData,
		RequireToolApproval:    opts.RequireToolApproval,
		ToolCallConcurrency:    opts.ToolCallConcurrency,
		ResumeContext:          opts.ResumeContext,
		AgentID:                opts.AgentID,
		AgentName:              opts.AgentName,
		ToolCallID:             opts.ToolCallID,
		MethodType:             opts.MethodType,
		SaveQueueManager:       opts.SaveQueueManager,
		MemoryConfig:           opts.MemoryConfig,
		Memory:                 opts.Memory,
		ResourceID:             opts.ResourceID,
		AutoResumeSuspendedTools: opts.Options.AutoResumeSuspendedTools,
		Workspace:              opts.Workspace,
	})

	mapResultsStep := CreateMapResultsStep(MapResultsStepOptions{
		Capabilities:   opts.Capabilities,
		Options:        opts.Options,
		ResourceID:     opts.ResourceID,
		RunID:          opts.RunID,
		RequestContext: opts.RequestContext,
		Memory:         opts.Memory,
		MemoryConfig:   opts.MemoryConfig,
		AgentSpan:      opts.AgentSpan,
		AgentID:        opts.AgentID,
		MethodType:     opts.MethodType,
	})

	return &PrepareStreamWorkflow{
		PrepareTools:  prepareToolsStep,
		PrepareMemory: prepareMemoryStep,
		MapResults:    mapResultsStep,
		Stream:        streamStep,
	}
}

// Execute runs the prepare-stream workflow:
// 1. Runs PrepareTools and PrepareMemory in parallel
// 2. Maps their results via MapResults
// 3. Streams via the Stream step
func (w *PrepareStreamWorkflow) Execute() (MastraModelOutput, error) {
	type toolsResult struct {
		output *PrepareToolsStepOutput
		err    error
	}
	type memoryResult struct {
		output *PrepareMemoryStepOutput
		err    error
	}

	toolsCh := make(chan toolsResult, 1)
	memoryCh := make(chan memoryResult, 1)

	// Run prepare-tools and prepare-memory in parallel
	go func() {
		out, err := w.PrepareTools()
		toolsCh <- toolsResult{output: out, err: err}
	}()
	go func() {
		out, err := w.PrepareMemory()
		memoryCh <- memoryResult{output: out, err: err}
	}()

	toolsRes := <-toolsCh
	if toolsRes.err != nil {
		return nil, toolsRes.err
	}

	memoryRes := <-memoryCh
	if memoryRes.err != nil {
		return nil, memoryRes.err
	}

	// Map results
	mapped, err := w.MapResults(toolsRes.output, memoryRes.output)
	if err != nil {
		return nil, err
	}

	// Stream
	return w.Stream(mapped)
}
