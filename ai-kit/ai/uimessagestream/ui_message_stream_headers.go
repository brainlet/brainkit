// Ported from: packages/ai/src/ui-message-stream/ui-message-stream-headers.ts
package uimessagestream

// UIMessageStreamHeaders are the default HTTP headers for a UI message stream response.
var UIMessageStreamHeaders = map[string]string{
	"content-type":                    "text/event-stream",
	"cache-control":                   "no-cache",
	"connection":                      "keep-alive",
	"x-vercel-ai-ui-message-stream":  "v1",
	"x-accel-buffering":              "no", // disable nginx buffering
}
