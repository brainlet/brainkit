// Ported from: packages/core/src/evals/base.ts
package evals

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ============================================================================
// Scorer Step Definition
// ============================================================================

// ScorerStepDefinition describes a single step in the scorer pipeline.
type ScorerStepDefinition struct {
	Name           string
	Definition     any  // FunctionStep or nil when IsPromptObject is true
	IsPromptObject bool
}

// ============================================================================
// Scorer Type Shortcuts
// ============================================================================

// ScorerTypeShortcut identifies a predefined type contract.
type ScorerTypeShortcut string

const (
	// ScorerTypeAgent is the predefined type shortcut for agent scorers.
	ScorerTypeAgent ScorerTypeShortcut = "agent"
)

// ============================================================================
// Pipeline Scorer Config
// ============================================================================

// MastraModelConfig is a stub for the LLM model configuration.
// NO BENEFIT TO REPLACE: The real model.MastraModelConfig in
// llm/model/shared_types.go is also `= any` (a union of string |
// OpenAICompatibleConfig | LanguageModelV1). Importing llm/model would add
// a dependency for zero type-safety gain since both are `= any`.
// No circular dep — evals CAN import llm/model (llm/model does not import evals).
type MastraModelConfig = any

// ScorerJudgeConfig holds the model and instructions for a judge LLM.
type ScorerJudgeConfig struct {
	Model        MastraModelConfig `json:"model"`
	Instructions string            `json:"instructions"`
}

// ScorerConfig defines the configuration for a scorer pipeline.
type ScorerConfig struct {
	ID          string              `json:"id"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description"`
	Judge       *ScorerJudgeConfig  `json:"judge,omitempty"`
	// Type can be a ScorerTypeShortcut (e.g., "agent") or nil.
	Type any `json:"type,omitempty"`
}

// ============================================================================
// Scorer Run
// ============================================================================

// ScorerRun is the standardized input type for all scorer pipelines.
type ScorerRun struct {
	RunID          string                         `json:"runId,omitempty"`
	Input          any                            `json:"input,omitempty"`
	Output         any                            `json:"output"`
	GroundTruth    any                            `json:"groundTruth,omitempty"`
	RequestContext map[string]any                 `json:"requestContext,omitempty"`

	// ObservabilityContext fields (partial, runtime only).
	ObservabilityContext *obstypes.ObservabilityContext `json:"-"`
}

// ============================================================================
// Prompt Object
// ============================================================================

// PromptObject defines a prompt-based step that delegates to a judge LLM.
type PromptObject struct {
	Description  string
	OutputSchema any // In Go, this would be a schema descriptor or validator; stub for now.
	Judge        *ScorerJudgeConfig
	// CreatePrompt generates the prompt string from the step context.
	// Can return a string or error.
	CreatePrompt func(ctx *StepContext) (string, error)
}

// ============================================================================
// Step Context Types
// ============================================================================

// StepContext provides the run and accumulated results to step functions.
type StepContext struct {
	Run     *ScorerRun     `json:"run"`
	Results map[string]any `json:"results"`
}

// GenerateReasonContext extends StepContext with the score from generateScore.
type GenerateReasonContext struct {
	Run     *ScorerRun     `json:"run"`
	Results map[string]any `json:"results"`
	Score   any            `json:"score"`
}

// ============================================================================
// Function Step Types
// ============================================================================

// FunctionStep is a step function that takes a StepContext and returns a result.
type FunctionStep func(ctx *StepContext) (any, error)

// GenerateScoreFunctionStep is a step function that returns a numeric score.
type GenerateScoreFunctionStep func(ctx *StepContext) (float64, error)

// GenerateReasonFunctionStep is a step function that returns a reason string.
type GenerateReasonFunctionStep func(ctx *GenerateReasonContext) (any, error)

// ============================================================================
// Generate Score/Reason Prompt Objects
// ============================================================================

// GenerateScorePromptObject is a prompt object for generateScore (always returns a number).
type GenerateScorePromptObject struct {
	Description  string
	Judge        *ScorerJudgeConfig
	CreatePrompt func(ctx *StepContext) (string, error)
}

// GenerateReasonPromptObject is a prompt object for generateReason (always returns a string).
type GenerateReasonPromptObject struct {
	Description  string
	Judge        *ScorerJudgeConfig
	CreatePrompt func(ctx *GenerateReasonContext) (string, error)
}

// ============================================================================
// Scorer Run Result
// ============================================================================

// ScorerRunResult is the output from executing a scorer pipeline.
type ScorerRunResult struct {
	RunID  string `json:"runId"`
	Input  any    `json:"input,omitempty"`
	Output any    `json:"output"`

	Score  any    `json:"score"`
	Reason any    `json:"reason,omitempty"`

	PreprocessPrompt     string `json:"preprocessPrompt,omitempty"`
	AnalyzePrompt        string `json:"analyzePrompt,omitempty"`
	GenerateScorePrompt  string `json:"generateScorePrompt,omitempty"`
	GenerateReasonPrompt string `json:"generateReasonPrompt,omitempty"`

	PreprocessStepResult any `json:"preprocessStepResult,omitempty"`
	AnalyzeStepResult    any `json:"analyzeStepResult,omitempty"`

	GroundTruth    any            `json:"groundTruth,omitempty"`
	RequestContext map[string]any `json:"requestContext,omitempty"`
}

// ============================================================================
// Step Info
// ============================================================================

// StepInfo describes a step in the scorer pipeline for introspection.
type StepInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "function" or "prompt"
	Description string `json:"description,omitempty"`
}

// ============================================================================
// MastraScorer
// ============================================================================

// MastraScorer is the pipeline scorer that chains preprocess, analyze,
// generateScore, and generateReason steps.
type MastraScorer struct {
	Config               ScorerConfig
	Source               string // "code" or "stored"

	steps                []ScorerStepDefinition
	originalPromptObjects map[string]any // name -> PromptObject | GenerateScorePromptObject | GenerateReasonPromptObject
	mastra               any            // *Mastra, typed as any to avoid circular dependency
	rawConfig            map[string]any
}

// NewMastraScorer creates a new MastraScorer with the given config.
func NewMastraScorer(config ScorerConfig) *MastraScorer {
	if config.ID == "" {
		panic(mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTR_SCORER_FAILED_TO_CREATE_MISSING_ID",
			Domain:   mastraerror.ErrorDomainScorer,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "Scorers must have an ID field. Please provide an ID in the scorer config.",
		}).Error())
	}

	return &MastraScorer{
		Config:                config,
		steps:                 make([]ScorerStepDefinition, 0),
		originalPromptObjects: make(map[string]any),
	}
}

// RegisterMastra registers the Mastra instance with the scorer.
// This enables access to custom gateways for model resolution.
func (s *MastraScorer) RegisterMastra(mastra any) {
	s.mastra = mastra
}

// ToRawConfig returns the raw storage configuration this scorer was created from,
// or nil if it was created from code.
func (s *MastraScorer) ToRawConfig() map[string]any {
	return s.rawConfig
}

// SetRawConfig sets the raw storage configuration for this scorer.
func (s *MastraScorer) SetRawConfig(rawConfig map[string]any) {
	s.rawConfig = rawConfig
}

// GetType returns the scorer type configuration.
func (s *MastraScorer) GetType() any {
	return s.Config.Type
}

// ID returns the scorer ID.
func (s *MastraScorer) ID() string {
	return s.Config.ID
}

// Name returns the scorer name (falls back to ID).
func (s *MastraScorer) Name() string {
	if s.Config.Name != "" {
		return s.Config.Name
	}
	return s.Config.ID
}

// Description returns the scorer description.
func (s *MastraScorer) Description() string {
	return s.Config.Description
}

// Judge returns the scorer's judge configuration.
func (s *MastraScorer) Judge() *ScorerJudgeConfig {
	return s.Config.Judge
}

// Preprocess adds a preprocess step to the scorer pipeline.
// stepDef can be a FunctionStep or a *PromptObject.
func (s *MastraScorer) Preprocess(stepDef any) *MastraScorer {
	isPromptObj := isPromptObject(stepDef)

	if isPromptObj {
		s.originalPromptObjects["preprocess"] = stepDef
	}

	newSteps := make([]ScorerStepDefinition, len(s.steps), len(s.steps)+1)
	copy(newSteps, s.steps)

	var def any
	if isPromptObj {
		def = nil
	} else {
		def = stepDef
	}

	newSteps = append(newSteps, ScorerStepDefinition{
		Name:           "preprocess",
		Definition:     def,
		IsPromptObject: isPromptObj,
	})

	newPromptObjects := make(map[string]any, len(s.originalPromptObjects))
	for k, v := range s.originalPromptObjects {
		newPromptObjects[k] = v
	}

	return &MastraScorer{
		Config:                s.Config,
		steps:                 newSteps,
		originalPromptObjects: newPromptObjects,
		mastra:                s.mastra,
	}
}

// Analyze adds an analyze step to the scorer pipeline.
// stepDef can be a FunctionStep or a *PromptObject.
func (s *MastraScorer) Analyze(stepDef any) *MastraScorer {
	isPromptObj := isPromptObject(stepDef)

	if isPromptObj {
		s.originalPromptObjects["analyze"] = stepDef
	}

	newSteps := make([]ScorerStepDefinition, len(s.steps), len(s.steps)+1)
	copy(newSteps, s.steps)

	var def any
	if isPromptObj {
		def = nil
	} else {
		def = stepDef
	}

	newSteps = append(newSteps, ScorerStepDefinition{
		Name:           "analyze",
		Definition:     def,
		IsPromptObject: isPromptObj,
	})

	newPromptObjects := make(map[string]any, len(s.originalPromptObjects))
	for k, v := range s.originalPromptObjects {
		newPromptObjects[k] = v
	}

	return &MastraScorer{
		Config:                s.Config,
		steps:                 newSteps,
		originalPromptObjects: newPromptObjects,
		mastra:                s.mastra,
	}
}

// GenerateScore adds a generateScore step to the scorer pipeline.
// stepDef can be a GenerateScoreFunctionStep or a *GenerateScorePromptObject.
func (s *MastraScorer) GenerateScore(stepDef any) *MastraScorer {
	isPromptObj := isPromptObject(stepDef)

	if isPromptObj {
		s.originalPromptObjects["generateScore"] = stepDef
	}

	newSteps := make([]ScorerStepDefinition, len(s.steps), len(s.steps)+1)
	copy(newSteps, s.steps)

	var def any
	if isPromptObj {
		def = nil
	} else {
		def = stepDef
	}

	newSteps = append(newSteps, ScorerStepDefinition{
		Name:           "generateScore",
		Definition:     def,
		IsPromptObject: isPromptObj,
	})

	newPromptObjects := make(map[string]any, len(s.originalPromptObjects))
	for k, v := range s.originalPromptObjects {
		newPromptObjects[k] = v
	}

	return &MastraScorer{
		Config:                s.Config,
		steps:                 newSteps,
		originalPromptObjects: newPromptObjects,
		mastra:                s.mastra,
	}
}

// GenerateReason adds a generateReason step to the scorer pipeline.
// stepDef can be a GenerateReasonFunctionStep or a *GenerateReasonPromptObject.
func (s *MastraScorer) GenerateReason(stepDef any) *MastraScorer {
	isPromptObj := isPromptObject(stepDef)

	if isPromptObj {
		s.originalPromptObjects["generateReason"] = stepDef
	}

	newSteps := make([]ScorerStepDefinition, len(s.steps), len(s.steps)+1)
	copy(newSteps, s.steps)

	var def any
	if isPromptObj {
		def = nil
	} else {
		def = stepDef
	}

	newSteps = append(newSteps, ScorerStepDefinition{
		Name:           "generateReason",
		Definition:     def,
		IsPromptObject: isPromptObj,
	})

	newPromptObjects := make(map[string]any, len(s.originalPromptObjects))
	for k, v := range s.originalPromptObjects {
		newPromptObjects[k] = v
	}

	return &MastraScorer{
		Config:                s.Config,
		steps:                 newSteps,
		originalPromptObjects: newPromptObjects,
		mastra:                s.mastra,
	}
}

// hasGenerateScore checks whether the pipeline includes a generateScore step.
func (s *MastraScorer) hasGenerateScore() bool {
	for _, step := range s.steps {
		if step.Name == "generateScore" {
			return true
		}
	}
	return false
}

// Run executes the scorer pipeline on the given input.
func (s *MastraScorer) Run(ctx context.Context, input *ScorerRun) (*ScorerRunResult, error) {
	if !s.hasGenerateScore() {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTR_SCORER_FAILED_TO_RUN_MISSING_GENERATE_SCORE",
			Domain:   mastraerror.ErrorDomainScorer,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "Cannot execute pipeline without generateScore() step",
			Details: map[string]any{
				"scorerId": s.Config.ID,
				"steps":    s.stepNames(),
			},
		})
	}

	runID := input.RunID
	if runID == "" {
		runID = uuid.New().String()
	}

	run := &ScorerRun{
		RunID:          runID,
		Input:          input.Input,
		Output:         input.Output,
		GroundTruth:    input.GroundTruth,
		RequestContext: input.RequestContext,
		ObservabilityContext: input.ObservabilityContext,
	}

	// Execute each step sequentially, accumulating results.
	accumulatedResults := make(map[string]any)
	generatedPrompts := make(map[string]string)

	for _, scorerStep := range s.steps {
		stepCtx := s.createScorerContext(scorerStep.Name, run, accumulatedResults)

		var stepResult any
		var err error

		if scorerStep.IsPromptObject {
			var prompt string
			stepResult, prompt, err = s.executePromptStep(ctx, scorerStep, stepCtx)
			if err != nil {
				return nil, fmt.Errorf("scorer step %q failed: %w", scorerStep.Name, err)
			}
			generatedPrompts[scorerStep.Name+"Prompt"] = prompt
		} else {
			stepResult, err = s.executeFunctionStep(scorerStep, stepCtx)
			if err != nil {
				return nil, fmt.Errorf("scorer step %q failed: %w", scorerStep.Name, err)
			}
		}

		accumulatedResults[scorerStep.Name+"StepResult"] = stepResult
	}

	return s.transformToScorerResult(accumulatedResults, generatedPrompts, run), nil
}

// GetSteps returns descriptive information about each step in the pipeline.
func (s *MastraScorer) GetSteps() []StepInfo {
	infos := make([]StepInfo, len(s.steps))
	for i, step := range s.steps {
		stepType := "function"
		if step.IsPromptObject {
			stepType = "prompt"
		}
		infos[i] = StepInfo{
			Name: step.Name,
			Type: stepType,
		}
	}
	return infos
}

// stepNames returns a comma-separated list of step names.
func (s *MastraScorer) stepNames() string {
	names := ""
	for i, step := range s.steps {
		if i > 0 {
			names += ", "
		}
		names += step.Name
	}
	return names
}

// createScorerContext builds the appropriate context for a step.
func (s *MastraScorer) createScorerContext(stepName string, run *ScorerRun, accumulatedResults map[string]any) any {
	if stepName == "generateReason" {
		score := accumulatedResults["generateScoreStepResult"]
		return &GenerateReasonContext{
			Run:     run,
			Results: accumulatedResults,
			Score:   score,
		}
	}
	return &StepContext{
		Run:     run,
		Results: accumulatedResults,
	}
}

// executeFunctionStep runs a function-based step.
func (s *MastraScorer) executeFunctionStep(scorerStep ScorerStepDefinition, ctx any) (any, error) {
	switch fn := scorerStep.Definition.(type) {
	case FunctionStep:
		return fn(ctx.(*StepContext))
	case GenerateScoreFunctionStep:
		return fn(ctx.(*StepContext))
	case GenerateReasonFunctionStep:
		return fn(ctx.(*GenerateReasonContext))
	case func(ctx *StepContext) (any, error):
		return fn(ctx.(*StepContext))
	case func(ctx *GenerateReasonContext) (any, error):
		return fn(ctx.(*GenerateReasonContext))
	default:
		return nil, fmt.Errorf("unsupported function step type for step %q: %T", scorerStep.Name, scorerStep.Definition)
	}
}

// executePromptStep runs a prompt-based step using the judge LLM.
func (s *MastraScorer) executePromptStep(ctx context.Context, scorerStep ScorerStepDefinition, stepCtx any) (any, string, error) {
	originalStep, ok := s.originalPromptObjects[scorerStep.Name]
	if !ok {
		return nil, "", fmt.Errorf("step %q is not a prompt object", scorerStep.Name)
	}

	// Determine the prompt, model config, and instructions based on prompt object type.
	var prompt string
	var modelConfig MastraModelConfig
	var instructions string

	switch po := originalStep.(type) {
	case *PromptObject:
		p, err := po.CreatePrompt(stepCtx.(*StepContext))
		if err != nil {
			return nil, "", fmt.Errorf("createPrompt failed for step %q: %w", scorerStep.Name, err)
		}
		prompt = p
		if po.Judge != nil {
			modelConfig = po.Judge.Model
			instructions = po.Judge.Instructions
		}
	case *GenerateScorePromptObject:
		p, err := po.CreatePrompt(stepCtx.(*StepContext))
		if err != nil {
			return nil, "", fmt.Errorf("createPrompt failed for step %q: %w", scorerStep.Name, err)
		}
		prompt = p
		if po.Judge != nil {
			modelConfig = po.Judge.Model
			instructions = po.Judge.Instructions
		}
	case *GenerateReasonPromptObject:
		p, err := po.CreatePrompt(stepCtx.(*GenerateReasonContext))
		if err != nil {
			return nil, "", fmt.Errorf("createPrompt failed for step %q: %w", scorerStep.Name, err)
		}
		prompt = p
		if po.Judge != nil {
			modelConfig = po.Judge.Model
			instructions = po.Judge.Instructions
		}
	default:
		return nil, "", fmt.Errorf("unknown prompt object type for step %q: %T", scorerStep.Name, originalStep)
	}

	// Fall back to scorer-level judge config if step-level is not set.
	if modelConfig == nil && s.Config.Judge != nil {
		modelConfig = s.Config.Judge.Model
	}
	if instructions == "" && s.Config.Judge != nil {
		instructions = s.Config.Judge.Instructions
	}

	if modelConfig == nil || instructions == "" {
		return nil, "", mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTR_SCORER_FAILED_TO_RUN_MISSING_MODEL_OR_INSTRUCTIONS",
			Domain:   mastraerror.ErrorDomainScorer,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Step %q requires a model and instructions", scorerStep.Name),
			Details: map[string]any{
				"scorerId": s.Config.ID,
				"step":     scorerStep.Name,
			},
		})
	}

	// TODO: Resolve model config and create judge Agent to execute the prompt.
	// This requires the Agent, resolveModelConfig, and tryGenerateWithJsonFallback
	// implementations to be ported. For now, return an error indicating the
	// prompt-based execution path is not yet implemented.
	_ = ctx       // Will be used for context propagation once Agent is ported.
	_ = prompt
	_ = instructions
	_ = modelConfig

	return nil, prompt, fmt.Errorf(
		"prompt-based step execution not yet implemented (requires Agent and model resolution); step=%q, scorer=%q",
		scorerStep.Name, s.Config.ID,
	)
}

// transformToScorerResult converts accumulated results into a ScorerRunResult.
func (s *MastraScorer) transformToScorerResult(
	accumulatedResults map[string]any,
	generatedPrompts map[string]string,
	originalInput *ScorerRun,
) *ScorerRunResult {
	return &ScorerRunResult{
		RunID:  originalInput.RunID,
		Input:  originalInput.Input,
		Output: originalInput.Output,

		Score:  accumulatedResults["generateScoreStepResult"],
		Reason: accumulatedResults["generateReasonStepResult"],

		GenerateScorePrompt:  generatedPrompts["generateScorePrompt"],
		GenerateReasonPrompt: generatedPrompts["generateReasonPrompt"],
		PreprocessStepResult: accumulatedResults["preprocessStepResult"],
		PreprocessPrompt:     generatedPrompts["preprocessPrompt"],
		AnalyzeStepResult:    accumulatedResults["analyzeStepResult"],
		AnalyzePrompt:        generatedPrompts["analyzePrompt"],

		GroundTruth:    originalInput.GroundTruth,
		RequestContext: originalInput.RequestContext,
	}
}

// isPromptObject checks whether a step definition is a prompt object
// (has Description and CreatePrompt fields).
func isPromptObject(stepDef any) bool {
	switch stepDef.(type) {
	case *PromptObject:
		return true
	case *GenerateScorePromptObject:
		return true
	case *GenerateReasonPromptObject:
		return true
	default:
		return false
	}
}

// ============================================================================
// createScorer factory function
// ============================================================================

// CreateScorer creates a new MastraScorer with the given config.
// This is the primary entry point for defining scorer pipelines.
func CreateScorer(config ScorerConfig) *MastraScorer {
	name := config.Name
	if name == "" {
		name = config.ID
	}
	return NewMastraScorer(ScorerConfig{
		ID:          config.ID,
		Name:        name,
		Description: config.Description,
		Judge:       config.Judge,
		Type:        config.Type,
	})
}

// ============================================================================
// MastraScorerEntry
// ============================================================================

// MastraScorerEntry pairs a scorer with an optional sampling configuration.
type MastraScorerEntry struct {
	Scorer   *MastraScorer          `json:"scorer"`
	Sampling *ScoringSamplingConfig `json:"sampling,omitempty"`
}

// MastraScorers is a named map of scorer entries.
type MastraScorers = map[string]MastraScorerEntry
