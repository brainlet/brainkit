// Ported from: packages/ai/src/middleware/extract-reasoning-middleware.ts
package middleware

import (
	"fmt"
	"regexp"
	"strings"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// ExtractReasoningMiddlewareOptions holds the configuration for ExtractReasoningMiddleware.
type ExtractReasoningMiddlewareOptions struct {
	// TagName is the name of the XML tag to extract reasoning from.
	TagName string

	// Separator is the separator to use between reasoning and text sections.
	// Defaults to "\n".
	Separator string

	// StartWithReasoning indicates whether to start with reasoning tokens.
	// Defaults to false.
	StartWithReasoning bool
}

// ExtractReasoningMiddleware extracts an XML-tagged reasoning section from the
// generated text and exposes it as a reasoning content part on the result.
func ExtractReasoningMiddleware(options ExtractReasoningMiddlewareOptions) mw.LanguageModelMiddleware {
	separator := options.Separator
	if separator == "" {
		separator = "\n"
	}
	startWithReasoning := options.StartWithReasoning

	openingTag := "<" + options.TagName + ">"
	closingTag := "</" + options.TagName + ">"

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

				text := textPart.Text
				if startWithReasoning {
					text = openingTag + text
				}

				pattern := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(openingTag) + `(.*?)` + regexp.QuoteMeta(closingTag))
				matches := pattern.FindAllStringSubmatchIndex(text, -1)

				if len(matches) == 0 {
					transformedContent = append(transformedContent, part)
					continue
				}

				// Extract reasoning texts
				var reasoningParts []string
				for _, match := range matches {
					reasoningParts = append(reasoningParts, text[match[2]:match[3]])
				}
				reasoningText := strings.Join(reasoningParts, separator)

				// Remove reasoning tags from text
				textWithoutReasoning := text
				for i := len(matches) - 1; i >= 0; i-- {
					match := matches[i]
					beforeMatch := textWithoutReasoning[:match[0]]
					afterMatch := textWithoutReasoning[match[1]:]

					sep := ""
					if len(beforeMatch) > 0 && len(afterMatch) > 0 {
						sep = separator
					}
					textWithoutReasoning = beforeMatch + sep + afterMatch
				}

				transformedContent = append(transformedContent, lm.Reasoning{
					Text: reasoningText,
				})
				transformedContent = append(transformedContent, lm.Text{
					Text: textWithoutReasoning,
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

			type reasoningExtraction struct {
				isFirstReasoning bool
				isFirstText      bool
				afterSwitch      bool
				isReasoning      bool
				buffer           string
				idCounter        int
				textID           string
			}

			reasoningExtractions := make(map[string]*reasoningExtraction)
			var delayedTextStart lm.StreamPart

			outCh := make(chan lm.StreamPart)
			go func() {
				defer close(outCh)
				for chunk := range result.Stream {
					switch c := chunk.(type) {
					case lm.StreamPartTextStart:
						delayedTextStart = c
						continue

					case lm.StreamPartTextEnd:
						if delayedTextStart != nil {
							outCh <- delayedTextStart
							delayedTextStart = nil
						}
						outCh <- chunk
						continue

					case lm.StreamPartTextDelta:
						if _, exists := reasoningExtractions[c.ID]; !exists {
							reasoningExtractions[c.ID] = &reasoningExtraction{
								isFirstReasoning: true,
								isFirstText:      true,
								afterSwitch:      false,
								isReasoning:      startWithReasoning,
								buffer:           "",
								idCounter:        0,
								textID:           c.ID,
							}
						}

						active := reasoningExtractions[c.ID]
						active.buffer += c.Delta

						publish := func(text string) {
							if len(text) > 0 {
								prefix := ""
								if active.afterSwitch {
									if active.isReasoning && !active.isFirstReasoning {
										prefix = separator
									} else if !active.isReasoning && !active.isFirstText {
										prefix = separator
									}
								}

								if active.isReasoning && (active.afterSwitch || active.isFirstReasoning) {
									outCh <- lm.StreamPartReasoningStart{
										ID: fmt.Sprintf("reasoning-%d", active.idCounter),
									}
								}

								if active.isReasoning {
									outCh <- lm.StreamPartReasoningDelta{
										Delta: prefix + text,
										ID:    fmt.Sprintf("reasoning-%d", active.idCounter),
									}
								} else {
									if delayedTextStart != nil {
										outCh <- delayedTextStart
										delayedTextStart = nil
									}
									outCh <- lm.StreamPartTextDelta{
										Delta: prefix + text,
										ID:    active.textID,
									}
								}
								active.afterSwitch = false

								if active.isReasoning {
									active.isFirstReasoning = false
								} else {
									active.isFirstText = false
								}
							}
						}

						for {
							nextTag := closingTag
							if !active.isReasoning {
								nextTag = openingTag
							}

							startIndex, found := util.GetPotentialStartIndex(active.buffer, nextTag)

							if !found {
								publish(active.buffer)
								active.buffer = ""
								break
							}

							// publish text before the tag
							publish(active.buffer[:startIndex])

							foundFullMatch := startIndex+len(nextTag) <= len(active.buffer)

							if foundFullMatch {
								active.buffer = active.buffer[startIndex+len(nextTag):]

								if active.isReasoning {
									// Emit reasoning-start for empty reasoning blocks
									if active.isFirstReasoning {
										outCh <- lm.StreamPartReasoningStart{
											ID: fmt.Sprintf("reasoning-%d", active.idCounter),
										}
									}

									outCh <- lm.StreamPartReasoningEnd{
										ID: fmt.Sprintf("reasoning-%d", active.idCounter),
									}
									active.idCounter++
								}

								active.isReasoning = !active.isReasoning
								active.afterSwitch = true
							} else {
								active.buffer = active.buffer[startIndex:]
								break
							}
						}

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
