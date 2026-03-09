package azure

import (
	"context"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// doGenerateOpts returns minimal CallOptions for DoGenerate with a simple prompt.
func doGenerateOpts() languagemodel.CallOptions {
	return languagemodel.CallOptions{
		Prompt: languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
		},
		Ctx: context.Background(),
	}
}

// doGenerateOptsWithHeaders returns CallOptions with custom request headers.
func doGenerateOptsWithHeaders(headers map[string]*string) languagemodel.CallOptions {
	opts := doGenerateOpts()
	opts.Headers = headers
	return opts
}

// doEmbedOpts returns minimal CallOptions for DoEmbed.
func doEmbedOpts(values []string) embeddingmodel.CallOptions {
	return embeddingmodel.CallOptions{
		Values: values,
		Ctx:    context.Background(),
	}
}

// doEmbedOptsWithHeaders returns CallOptions for DoEmbed with custom request headers.
func doEmbedOptsWithHeaders(values []string, headers map[string]*string) embeddingmodel.CallOptions {
	// Convert map[string]*string to map[string]string for embeddingmodel.CallOptions
	h := make(map[string]string)
	for k, v := range headers {
		if v != nil {
			h[k] = *v
		}
	}
	return embeddingmodel.CallOptions{
		Values:  values,
		Ctx:     context.Background(),
		Headers: h,
	}
}

// doImageGenerateOpts returns minimal CallOptions for image DoGenerate.
func doImageGenerateOpts(prompt string) imagemodel.CallOptions {
	return imagemodel.CallOptions{
		Prompt: &prompt,
		N:      1,
		Size:   strPtr("1024x1024"),
		Ctx:    context.Background(),
	}
}

// doImageGenerateOptsWithHeaders returns image CallOptions with custom request headers.
func doImageGenerateOptsWithHeaders(prompt string, headers map[string]*string) imagemodel.CallOptions {
	opts := doImageGenerateOpts(prompt)
	opts.Headers = headers
	return opts
}

// doImageGenerateOptsWithN returns image CallOptions with a specific number of images.
func doImageGenerateOptsWithN(prompt string, n int) imagemodel.CallOptions {
	return imagemodel.CallOptions{
		Prompt: &prompt,
		N:      n,
		Size:   strPtr("1024x1024"),
		Ctx:    context.Background(),
	}
}
