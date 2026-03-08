// Ported from: packages/ai/src/middleware/extract-json-middleware.ts
package middleware

import (
	"regexp"
	"strings"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

var (
	jsonFencePrefix = regexp.MustCompile(`^` + "```" + `(?:json)?\s*\n?`)
	jsonFenceSuffix = regexp.MustCompile(`\n?` + "```" + `\s*$`)
)

// defaultTransformJSON strips markdown code fences from text.
func defaultTransformJSON(text string) string {
	text = jsonFencePrefix.ReplaceAllString(text, "")
	text = jsonFenceSuffix.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

// ExtractJsonMiddlewareOptions holds the configuration for ExtractJsonMiddleware.
type ExtractJsonMiddlewareOptions struct {
	// Transform is a custom transform function to apply to text content.
	// If not provided, the default transform strips markdown code fences.
	Transform func(string) string
}

// ExtractJsonMiddleware creates middleware that extracts JSON from text content
// by stripping markdown code fences and other formatting.
func ExtractJsonMiddleware(options *ExtractJsonMiddlewareOptions) mw.LanguageModelMiddleware {
	transform := defaultTransformJSON
	hasCustomTransform := false
	if options != nil && options.Transform != nil {
		transform = options.Transform
		hasCustomTransform = true
	}

	return mw.LanguageModelMiddleware{
		WrapGenerate: func(opts mw.WrapGenerateOptions) (lm.GenerateResult, error) {
			result, err := opts.DoGenerate()
			if err != nil {
				return lm.GenerateResult{}, err
			}

			var transformedContent []lm.Content
			for _, part := range result.Content {
				textPart, ok := part.(lm.Text)
				if !ok {
					transformedContent = append(transformedContent, part)
					continue
				}
				transformedContent = append(transformedContent, lm.Text{
					Text:             transform(textPart.Text),
					ProviderMetadata: textPart.ProviderMetadata,
				})
			}

			result.Content = transformedContent
			return result, nil
		},
		WrapStream: func(opts mw.WrapStreamOptions) (lm.StreamResult, error) {
			result, err := opts.DoStream()
			if err != nil {
				return lm.StreamResult{}, err
			}

			type textBlock struct {
				startEvent     lm.StreamPart
				phase          string // "prefix", "streaming", "buffering"
				buffer         string
				prefixStripped bool
			}

			textBlocks := make(map[string]*textBlock)
			const suffixBufferSize = 12

			outCh := make(chan lm.StreamPart)
			go func() {
				defer close(outCh)
				for chunk := range result.Stream {
					switch c := chunk.(type) {
					case lm.StreamPartTextStart:
						phase := "prefix"
						if hasCustomTransform {
							phase = "buffering"
						}
						textBlocks[c.ID] = &textBlock{
							startEvent: c,
							phase:      phase,
							buffer:     "",
						}

					case lm.StreamPartTextDelta:
						block, exists := textBlocks[c.ID]
						if !exists {
							outCh <- chunk
							continue
						}

						block.buffer += c.Delta

						// Custom transform: buffer everything, transform at end
						if block.phase == "buffering" {
							continue
						}

						if block.phase == "prefix" {
							if len(block.buffer) > 0 && !strings.HasPrefix(block.buffer, "`") {
								block.phase = "streaming"
								outCh <- block.startEvent
							} else if strings.HasPrefix(block.buffer, "```") {
								if strings.Contains(block.buffer, "\n") {
									prefixMatch := regexp.MustCompile(`^` + "```" + `(?:json)?\s*\n`)
									if loc := prefixMatch.FindStringIndex(block.buffer); loc != nil {
										block.buffer = block.buffer[loc[1]:]
										block.prefixStripped = true
										block.phase = "streaming"
										outCh <- block.startEvent
									} else {
										block.phase = "streaming"
										outCh <- block.startEvent
									}
								}
								// else keep buffering until we see a newline
							} else if len(block.buffer) >= 3 && !strings.HasPrefix(block.buffer, "```") {
								block.phase = "streaming"
								outCh <- block.startEvent
							}
						}

						// Stream content
						if block.phase == "streaming" && len(block.buffer) > suffixBufferSize {
							toStream := block.buffer[:len(block.buffer)-suffixBufferSize]
							block.buffer = block.buffer[len(block.buffer)-suffixBufferSize:]
							outCh <- lm.StreamPartTextDelta{
								ID:    c.ID,
								Delta: toStream,
							}
						}

					case lm.StreamPartTextEnd:
						block, exists := textBlocks[c.ID]
						if exists {
							if block.phase == "prefix" || block.phase == "buffering" {
								outCh <- block.startEvent
							}

							remaining := block.buffer
							if block.phase == "buffering" {
								remaining = transform(remaining)
							} else if block.prefixStripped {
								remaining = jsonFenceSuffix.ReplaceAllString(remaining, "")
								remaining = strings.TrimRight(remaining, " \t\n\r")
							} else {
								remaining = transform(remaining)
							}

							if len(remaining) > 0 {
								outCh <- lm.StreamPartTextDelta{
									ID:    c.ID,
									Delta: remaining,
								}
							}
							outCh <- chunk
							delete(textBlocks, c.ID)
							continue
						}
						outCh <- chunk

					default:
						outCh <- chunk
					}
				}
			}()

			return lm.StreamResult{
				Stream:   outCh,
				Request:  result.Request,
				Response: result.Response,
			}, nil
		},
	}
}
