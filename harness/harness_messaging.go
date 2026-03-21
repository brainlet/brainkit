package harness

import "encoding/json"

// SendMessage sends a user message to the current agent.
// Blocks until the agent finishes. Events stream to subscribers during execution.
func (h *Harness) SendMessage(content string, opts ...SendOption) error {
	o := &sendOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{"content": content}
	if o.files != nil {
		args["files"] = o.files
	}
	if o.requestContext != nil {
		args["requestContext"] = o.requestContext
	}
	b, _ := json.Marshal(args)
	return h.callJSVoid("sendMessage", string(b))
}

// Abort cancels the current agent execution.
func (h *Harness) Abort() error {
	return h.callJSVoid("abort", "")
}

// Steer aborts the current execution and sends a new message.
func (h *Harness) Steer(content string, opts ...SendOption) error {
	o := &sendOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{"content": content}
	b, _ := json.Marshal(args)
	return h.callJSVoid("steer", string(b))
}

// FollowUp queues a message after the current execution finishes.
func (h *Harness) FollowUp(content string, opts ...SendOption) error {
	o := &sendOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{"content": content}
	b, _ := json.Marshal(args)
	return h.callJSVoid("followUp", string(b))
}

// GetCurrentRunID returns the active run ID, or empty string.
func (h *Harness) GetCurrentRunID() string {
	r, _ := h.callJSSimple("getCurrentRunId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}
