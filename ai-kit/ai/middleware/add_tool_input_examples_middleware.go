// Ported from: packages/ai/src/middleware/add-tool-input-examples-middleware.ts
package middleware

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

// defaultFormatExample formats an input example as JSON.
func defaultFormatExample(example lm.FunctionToolInputExample, _ int) string {
	b, err := json.Marshal(example.Input)
	if err != nil {
		return fmt.Sprintf("%v", example.Input)
	}
	return string(b)
}

// AddToolInputExamplesMiddlewareOptions holds configuration for AddToolInputExamplesMiddleware.
type AddToolInputExamplesMiddlewareOptions struct {
	// Prefix is a prefix to prepend before the examples.
	// Defaults to "Input Examples:".
	Prefix *string

	// Format is an optional custom formatter for each example.
	// Receives the example object and its index.
	// Default: JSON.stringify(example.input)
	Format func(example lm.FunctionToolInputExample, index int) string

	// Remove controls whether to remove the inputExamples property after adding
	// them to the description. Defaults to true.
	Remove *bool
}

// AddToolInputExamplesMiddleware creates middleware that appends input examples
// to tool descriptions.
func AddToolInputExamplesMiddleware(opts *AddToolInputExamplesMiddlewareOptions) mw.LanguageModelMiddleware {
	prefix := "Input Examples:"
	format := defaultFormatExample
	remove := true

	if opts != nil {
		if opts.Prefix != nil {
			prefix = *opts.Prefix
		}
		if opts.Format != nil {
			format = opts.Format
		}
		if opts.Remove != nil {
			remove = *opts.Remove
		}
	}

	return mw.LanguageModelMiddleware{
		TransformParams: func(tpOpts mw.TransformParamsOptions) (lm.CallOptions, error) {
			params := tpOpts.Params

			if len(params.Tools) == 0 {
				return params, nil
			}

			var transformedTools []lm.Tool
			for _, tool := range params.Tools {
				ft, ok := tool.(lm.FunctionTool)
				if !ok || len(ft.InputExamples) == 0 {
					transformedTools = append(transformedTools, tool)
					continue
				}

				var formattedParts []string
				for i, example := range ft.InputExamples {
					formattedParts = append(formattedParts, format(lm.FunctionToolInputExample{
						Input: jsonvalue.JSONObject(example.Input),
					}, i))
				}
				formattedExamples := strings.Join(formattedParts, "\n")

				examplesSection := prefix + "\n" + formattedExamples

				toolDescription := examplesSection
				if ft.Description != nil {
					toolDescription = *ft.Description + "\n\n" + examplesSection
				}

				newTool := lm.FunctionTool{
					Name:            ft.Name,
					Description:     &toolDescription,
					InputSchema:     ft.InputSchema,
					Strict:          ft.Strict,
					ProviderOptions: ft.ProviderOptions,
				}
				if !remove {
					newTool.InputExamples = ft.InputExamples
				}

				transformedTools = append(transformedTools, newTool)
			}

			params.Tools = transformedTools
			return params, nil
		},
	}
}
