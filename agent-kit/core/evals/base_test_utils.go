// Ported from: packages/core/src/evals/base.test-utils.ts
package evals

import "fmt"

// ============================================================================
// Function-Based Scorer Builders
// ============================================================================

// FunctionBasedScorerBuilders contains pre-built function-based scorers
// used for testing the scorer pipeline.
//
// Each field returns a *MastraScorer configured with different pipeline
// step combinations (preprocess, analyze, generateScore, generateReason).
var FunctionBasedScorerBuilders = struct {
	Basic                                func() *MastraScorer
	WithPreprocess                       func() *MastraScorer
	WithPreprocessAndAnalyze             func() *MastraScorer
	WithPreprocessAndAnalyzeAndReason    func() *MastraScorer
	WithPreprocessAndReason              func() *MastraScorer
	WithAnalyze                          func() *MastraScorer
	WithAnalyzeAndReason                 func() *MastraScorer
	WithReason                           func() *MastraScorer
}{
	Basic: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			// In TS: if (run.input?.[0]?.content.length > 0 && run.output.text.length > 0) return 1;
			input := ctx.Run.Input
			output := ctx.Run.Output
			if input != nil && output != nil {
				return 1, nil
			}
			return 0, nil
		}))
	},

	WithPreprocess: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			// In TS: return { reformattedInput: run.input?.[0]?.content.toUpperCase(), ... }
			return map[string]any{
				"reformattedInput":  "REFORMATTED_INPUT",
				"reformattedOutput": "REFORMATTED_OUTPUT",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			if preprocessResult != nil {
				ri, _ := preprocessResult["reformattedInput"].(string)
				ro, _ := preprocessResult["reformattedOutput"].(string)
				if len(ri) > 0 && len(ro) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithPreprocessAndAnalyze: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"reformattedInput":  "REFORMATTED_INPUT",
				"reformattedOutput": "REFORMATTED_OUTPUT",
			}, nil
		})).Analyze(FunctionStep(func(ctx *StepContext) (any, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			ri, _ := preprocessResult["reformattedInput"].(string)
			ro, _ := preprocessResult["reformattedOutput"].(string)
			return map[string]any{
				"inputFromAnalyze":  ri + "!",
				"outputFromAnalyze": ro + "!",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				ia, _ := analyzeResult["inputFromAnalyze"].(string)
				oa, _ := analyzeResult["outputFromAnalyze"].(string)
				if len(ia) > 0 && len(oa) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithPreprocessAndAnalyzeAndReason: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"reformattedInput":  "REFORMATTED_INPUT",
				"reformattedOutput": "REFORMATTED_OUTPUT",
			}, nil
		})).Analyze(FunctionStep(func(ctx *StepContext) (any, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			ri, _ := preprocessResult["reformattedInput"].(string)
			ro, _ := preprocessResult["reformattedOutput"].(string)
			return map[string]any{
				"inputFromAnalyze":  ri + "!",
				"outputFromAnalyze": ro + "!",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				ia, _ := analyzeResult["inputFromAnalyze"].(string)
				oa, _ := analyzeResult["outputFromAnalyze"].(string)
				if len(ia) > 0 && len(oa) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		})).GenerateReason(GenerateReasonFunctionStep(func(ctx *GenerateReasonContext) (any, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			ia, _ := analyzeResult["inputFromAnalyze"].(string)
			oa, _ := analyzeResult["outputFromAnalyze"].(string)
			return fmt.Sprintf("the reason the score is %v is because the input is %s and the output is %s",
				ctx.Score, ia, oa), nil
		}))
	},

	WithPreprocessAndReason: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"reformattedInput":  "REFORMATTED_INPUT",
				"reformattedOutput": "REFORMATTED_OUTPUT",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			if preprocessResult != nil {
				ri, _ := preprocessResult["reformattedInput"].(string)
				ro, _ := preprocessResult["reformattedOutput"].(string)
				if len(ri) > 0 && len(ro) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		})).GenerateReason(GenerateReasonFunctionStep(func(ctx *GenerateReasonContext) (any, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			ri, _ := preprocessResult["reformattedInput"].(string)
			ro, _ := preprocessResult["reformattedOutput"].(string)
			return fmt.Sprintf("the reason the score is %v is because the input is %s and the output is %s",
				ctx.Score, ri, ro), nil
		}))
	},

	WithAnalyze: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Analyze(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"inputFromAnalyze":  "input!",
				"outputFromAnalyze": "output!",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				ia, _ := analyzeResult["inputFromAnalyze"].(string)
				oa, _ := analyzeResult["outputFromAnalyze"].(string)
				if len(ia) > 0 && len(oa) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithAnalyzeAndReason: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Analyze(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"inputFromAnalyze":  "input!",
				"outputFromAnalyze": "output!",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				ia, _ := analyzeResult["inputFromAnalyze"].(string)
				oa, _ := analyzeResult["outputFromAnalyze"].(string)
				if len(ia) > 0 && len(oa) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		})).GenerateReason(GenerateReasonFunctionStep(func(ctx *GenerateReasonContext) (any, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			ia, _ := analyzeResult["inputFromAnalyze"].(string)
			oa, _ := analyzeResult["outputFromAnalyze"].(string)
			return fmt.Sprintf("the reason the score is %v is because the input is %s and the output is %s",
				ctx.Score, ia, oa), nil
		}))
	},

	WithReason: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			if ctx.Run.Input != nil {
				return 1, nil
			}
			return 0, nil
		})).GenerateReason(GenerateReasonFunctionStep(func(ctx *GenerateReasonContext) (any, error) {
			return fmt.Sprintf("the reason the score is %v is because the input is %v and the output is %v",
				ctx.Score, ctx.Run.Input, ctx.Run.Output), nil
		}))
	},
}

// ============================================================================
// Prompt-Based Scorer Builders
// ============================================================================

// PromptBasedScorerBuilders contains pre-built prompt-based scorers
// used for testing the scorer pipeline with judge LLM prompt objects.
//
// NOTE: In Go, the MockLanguageModelV1 and zod schemas from TS are replaced
// with placeholder judge configs and prompt objects. The actual LLM execution
// path is not yet implemented (see base.go executePromptStep TODO).
var PromptBasedScorerBuilders = struct {
	WithAnalyze                      func() *MastraScorer
	WithPreprocessAndAnalyze         func() *MastraScorer
	WithAnalyzeAndReason             func() *MastraScorer
	WithGenerateScoreAsPromptObject  func() *MastraScorer
	WithAllSteps                     func() *MastraScorer
}{
	WithAnalyze: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Analyze(&PromptObject{
			Description: "Analyze the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model", // TODO: Replace with real model once LLM package is ported
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Analyze prompt", nil
			},
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				_, hasInput := analyzeResult["inputLength"]
				_, hasOutput := analyzeResult["outputLength"]
				if hasInput && hasOutput {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithPreprocessAndAnalyze: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(&PromptObject{
			Description: "Preprocess the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Preprocess prompt", nil
			},
		}).Analyze(&PromptObject{
			Description: "Analyze the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Analyze prompt", nil
			},
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				_, hasInput := analyzeResult["inputLength"]
				_, hasOutput := analyzeResult["outputLength"]
				if hasInput && hasOutput {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithAnalyzeAndReason: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Analyze(&PromptObject{
			Description: "Analyze the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Analyze prompt", nil
			},
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				_, hasInput := analyzeResult["inputLength"]
				_, hasOutput := analyzeResult["outputLength"]
				if hasInput && hasOutput {
					return 1, nil
				}
			}
			return 0, nil
		})).GenerateReason(&GenerateReasonPromptObject{
			Description: "Generate a reason for the score",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *GenerateReasonContext) (string, error) {
				return "Test Generate Reason prompt", nil
			},
		})
	},

	WithGenerateScoreAsPromptObject: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).GenerateScore(&GenerateScorePromptObject{
			Description: "Generate a score",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Generate Score prompt", nil
			},
		}).GenerateReason(GenerateReasonFunctionStep(func(ctx *GenerateReasonContext) (any, error) {
			return fmt.Sprintf("the reason the score is %v is because the input is %v and the output is %v",
				ctx.Score, ctx.Run.Input, ctx.Run.Output), nil
		}))
	},

	WithAllSteps: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(&PromptObject{
			Description: "Preprocess the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Preprocess prompt", nil
			},
		}).Analyze(&PromptObject{
			Description: "Analyze the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Analyze prompt", nil
			},
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				_, hasInput := analyzeResult["inputLength"]
				_, hasOutput := analyzeResult["outputLength"]
				if hasInput && hasOutput {
					return 1, nil
				}
			}
			return 0, nil
		})).GenerateReason(&GenerateReasonPromptObject{
			Description: "Generate a reason for the score",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *GenerateReasonContext) (string, error) {
				return "Test Generate Reason prompt", nil
			},
		})
	},
}

// ============================================================================
// Mixed Scorer Builders
// ============================================================================

// MixedScorerBuilders contains pre-built scorers that mix function-based
// and prompt-based steps for testing hybrid pipelines.
var MixedScorerBuilders = struct {
	WithPreprocessFunctionAnalyzePrompt func() *MastraScorer
	WithPreprocessPromptAnalyzeFunction func() *MastraScorer
	WithReasonFunctionAnalyzePrompt     func() *MastraScorer
	WithReasonPromptAnalyzeFunction     func() *MastraScorer
}{
	WithPreprocessFunctionAnalyzePrompt: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"reformattedInput":  "INPUT from preprocess function!",
				"reformattedOutput": "OUTPUT from preprocess function!",
			}, nil
		})).Analyze(&PromptObject{
			Description: "Analyze the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Analyze prompt", nil
			},
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				_, hasInput := analyzeResult["inputLength"]
				_, hasOutput := analyzeResult["outputLength"]
				if hasInput && hasOutput {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithPreprocessPromptAnalyzeFunction: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(&PromptObject{
			Description: "Preprocess the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Preprocess prompt", nil
			},
		}).Analyze(FunctionStep(func(ctx *StepContext) (any, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			ri, _ := preprocessResult["reformattedInput"].(string)
			ro, _ := preprocessResult["reformattedOutput"].(string)
			return map[string]any{
				"inputFromAnalyze":  ri + "!",
				"outputFromAnalyze": ro + "!",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			if analyzeResult == nil || preprocessResult == nil {
				return 0, nil
			}
			ia, _ := analyzeResult["inputFromAnalyze"].(string)
			oa, _ := analyzeResult["outputFromAnalyze"].(string)
			ri, _ := preprocessResult["reformattedInput"].(string)
			ro, _ := preprocessResult["reformattedOutput"].(string)
			if len(ia) > 0 && len(oa) > 0 && len(ri) > 0 && len(ro) > 0 {
				return 1, nil
			}
			return 0, nil
		}))
	},

	WithReasonFunctionAnalyzePrompt: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Analyze(&PromptObject{
			Description: "Analyze the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return "Test Analyze prompt", nil
			},
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				_, hasInput := analyzeResult["inputLength"]
				_, hasOutput := analyzeResult["outputLength"]
				if hasInput && hasOutput {
					return 1, nil
				}
			}
			return 0, nil
		})).GenerateReason(GenerateReasonFunctionStep(func(ctx *GenerateReasonContext) (any, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			il, _ := analyzeResult["inputLength"]
			ol, _ := analyzeResult["outputLength"]
			return fmt.Sprintf("the reason is because the input is %v and the output is %v from generateReason function",
				il, ol), nil
		}))
	},

	WithReasonPromptAnalyzeFunction: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Analyze(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"inputFromAnalyze":  "input from analyze function!",
				"outputFromAnalyze": "output from analyze function!",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				ia, _ := analyzeResult["inputFromAnalyze"].(string)
				oa, _ := analyzeResult["outputFromAnalyze"].(string)
				if len(ia) > 0 && len(oa) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		})).GenerateReason(&GenerateReasonPromptObject{
			Description: "Generate a reason for the score",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Test instructions",
			},
			CreatePrompt: func(ctx *GenerateReasonContext) (string, error) {
				return "Test Generate Reason prompt", nil
			},
		})
	},
}

// ============================================================================
// Async Function-Based Scorer Builders
// ============================================================================

// AsyncFunctionBasedScorerBuilders contains pre-built scorers that use
// asynchronous function steps.
//
// NOTE: In Go, all function steps are synchronous. The "async" variants
// from TS are functionally identical to their sync counterparts here.
// The struct is preserved for 1:1 parity with the TS test utils.
var AsyncFunctionBasedScorerBuilders = struct {
	Basic                                        func() *MastraScorer
	WithPreprocess                               func() *MastraScorer
	WithPreprocessFunctionAndAnalyzePromptObject func() *MastraScorer
	WithPreprocessPromptObjectAndAnalyzeFunction func() *MastraScorer
	WithAsyncCreatePromptInPreprocess            func() *MastraScorer
	WithAsyncCreatePromptInAnalyze               func() *MastraScorer
	WithAsyncCreatePromptInGenerateScore         func() *MastraScorer
	WithAsyncCreatePromptInGenerateReason        func() *MastraScorer
}{
	Basic: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			return 1, nil
		}))
	},

	WithPreprocess: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"reformattedInput":  "REFORMATTED_INPUT",
				"reformattedOutput": "REFORMATTED_OUTPUT",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			if preprocessResult != nil {
				ri, _ := preprocessResult["reformattedInput"].(string)
				ro, _ := preprocessResult["reformattedOutput"].(string)
				if len(ri) > 0 && len(ro) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithPreprocessFunctionAndAnalyzePromptObject: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"reformattedInput":  "REFORMATTED_INPUT",
				"reformattedOutput": "REFORMATTED_OUTPUT",
			}, nil
		})).Analyze(&PromptObject{
			Description: "Analyze the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Analyze the input and output",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return fmt.Sprintf("Analyze the input and output: %v and %v",
					ctx.Run.Input, ctx.Run.Output), nil
			},
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				ia, _ := analyzeResult["inputFromAnalyze"].(string)
				oa, _ := analyzeResult["outputFromAnalyze"].(string)
				if len(ia) > 0 && len(oa) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithPreprocessPromptObjectAndAnalyzeFunction: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(&PromptObject{
			Description: "Preprocess the input and output",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Analyze the input and output",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return fmt.Sprintf("Analyze the input and output: %v and %v",
					ctx.Run.Input, ctx.Run.Output), nil
			},
		}).Analyze(FunctionStep(func(ctx *StepContext) (any, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			ri, _ := preprocessResult["reformattedInput"].(string)
			ro, _ := preprocessResult["reformattedOutput"].(string)
			return map[string]any{
				"inputFromAnalyze":  ri,
				"outputFromAnalyze": ro,
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				ia, _ := analyzeResult["inputFromAnalyze"].(string)
				oa, _ := analyzeResult["outputFromAnalyze"].(string)
				if len(ia) > 0 && len(oa) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithAsyncCreatePromptInPreprocess: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(&PromptObject{
			Description: "Preprocess with async createPrompt",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Analyze the input and output",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				return fmt.Sprintf("Async prompt: %v and %v",
					ctx.Run.Input, ctx.Run.Output), nil
			},
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			if preprocessResult != nil {
				ri, _ := preprocessResult["reformattedInput"].(string)
				ro, _ := preprocessResult["reformattedOutput"].(string)
				if len(ri) > 0 && len(ro) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithAsyncCreatePromptInAnalyze: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"reformattedInput":  "REFORMATTED_INPUT",
				"reformattedOutput": "REFORMATTED_OUTPUT",
			}, nil
		})).Analyze(&PromptObject{
			Description: "Analyze with async createPrompt",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Analyze the input and output",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
				ri, _ := preprocessResult["reformattedInput"].(string)
				ro, _ := preprocessResult["reformattedOutput"].(string)
				return fmt.Sprintf("Async analyze prompt: %s and %s", ri, ro), nil
			},
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			analyzeResult, _ := ctx.Results["analyzeStepResult"].(map[string]any)
			if analyzeResult != nil {
				ia, _ := analyzeResult["inputFromAnalyze"].(string)
				oa, _ := analyzeResult["outputFromAnalyze"].(string)
				if len(ia) > 0 && len(oa) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		}))
	},

	WithAsyncCreatePromptInGenerateScore: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"reformattedInput":  "REFORMATTED_INPUT",
				"reformattedOutput": "REFORMATTED_OUTPUT",
			}, nil
		})).GenerateScore(&GenerateScorePromptObject{
			Description: "Generate score with async createPrompt",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Generate a score",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) {
				preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
				ri, _ := preprocessResult["reformattedInput"].(string)
				ro, _ := preprocessResult["reformattedOutput"].(string)
				return fmt.Sprintf("Async score prompt: %s and %s", ri, ro), nil
			},
		})
	},

	WithAsyncCreatePromptInGenerateReason: func() *MastraScorer {
		return CreateScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "test-scorer",
			Description: "A test scorer",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return map[string]any{
				"reformattedInput":  "REFORMATTED_INPUT",
				"reformattedOutput": "REFORMATTED_OUTPUT",
			}, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
			if preprocessResult != nil {
				ri, _ := preprocessResult["reformattedInput"].(string)
				ro, _ := preprocessResult["reformattedOutput"].(string)
				if len(ri) > 0 && len(ro) > 0 {
					return 1, nil
				}
			}
			return 0, nil
		})).GenerateReason(&GenerateReasonPromptObject{
			Description: "Generate reason with async createPrompt",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "Generate a reason",
			},
			CreatePrompt: func(ctx *GenerateReasonContext) (string, error) {
				preprocessResult, _ := ctx.Results["preprocessStepResult"].(map[string]any)
				ri, _ := preprocessResult["reformattedInput"].(string)
				ro, _ := preprocessResult["reformattedOutput"].(string)
				return fmt.Sprintf("Async reason prompt: Score %v for %s and %s",
					ctx.Score, ri, ro), nil
			},
		})
	},
}
