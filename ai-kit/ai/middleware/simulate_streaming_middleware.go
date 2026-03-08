// Ported from: packages/ai/src/middleware/simulate-streaming-middleware.ts
package middleware

import (
	"fmt"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

// SimulateStreamingMiddleware simulates streaming chunks with the response from
// a generate call.
func SimulateStreamingMiddleware() mw.LanguageModelMiddleware {
	return mw.LanguageModelMiddleware{
		WrapStream: func(opts mw.WrapStreamOptions) (lm.StreamResult, error) {
			result, err := opts.DoGenerate()
			if err != nil {
				return lm.StreamResult{}, err
			}

			ch := make(chan lm.StreamPart)

			go func() {
				defer close(ch)

				id := 0

				// Stream start
				ch <- lm.StreamPartStreamStart{
					Warnings: result.Warnings,
				}

				// Response metadata
				if result.Response != nil {
					ch <- lm.StreamPartResponseMetadata{
						ResponseMetadata: lm.ResponseMetadata{
							ID:        result.Response.ID,
							ModelID:   result.Response.ModelID,
							Timestamp: result.Response.Timestamp,
						},
					}
				}

				for _, part := range result.Content {
					switch p := part.(type) {
					case lm.Text:
						if len(p.Text) > 0 {
							ch <- lm.StreamPartTextStart{ID: fmt.Sprintf("%d", id)}
							ch <- lm.StreamPartTextDelta{
								ID:    fmt.Sprintf("%d", id),
								Delta: p.Text,
							}
							ch <- lm.StreamPartTextEnd{ID: fmt.Sprintf("%d", id)}
							id++
						}
					case lm.Reasoning:
						ch <- lm.StreamPartReasoningStart{
							ID:               fmt.Sprintf("%d", id),
							ProviderMetadata: p.ProviderMetadata,
						}
						ch <- lm.StreamPartReasoningDelta{
							ID:    fmt.Sprintf("%d", id),
							Delta: p.Text,
						}
						ch <- lm.StreamPartReasoningEnd{ID: fmt.Sprintf("%d", id)}
						id++
					default:
						// For any other content type that also implements StreamPart,
						// pass it through.
						if sp, ok := part.(lm.StreamPart); ok {
							ch <- sp
						}
					}
				}

				ch <- lm.StreamPartFinish{
					FinishReason:     result.FinishReason,
					Usage:            result.Usage,
					ProviderMetadata: result.ProviderMetadata,
				}
			}()

			return lm.StreamResult{
				Stream:   ch,
				Request:  toStreamResultRequest(result.Request),
				Response: toStreamResultResponse(result.Response),
			}, nil
		},
	}
}

// toStreamResultRequest converts GenerateResultRequest to StreamResultRequest.
func toStreamResultRequest(req *lm.GenerateResultRequest) *lm.StreamResultRequest {
	if req == nil {
		return nil
	}
	return &lm.StreamResultRequest{
		Body: req.Body,
	}
}

// toStreamResultResponse converts GenerateResultResponse to StreamResultResponse.
func toStreamResultResponse(resp *lm.GenerateResultResponse) *lm.StreamResultResponse {
	if resp == nil {
		return nil
	}
	return &lm.StreamResultResponse{
		Headers: resp.Headers,
	}
}
