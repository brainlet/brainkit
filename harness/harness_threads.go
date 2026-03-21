package harness

import "encoding/json"

// CreateThread creates a new conversation thread.
func (h *Harness) CreateThread(opts ...ThreadOption) (string, error) {
	o := &threadOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{}
	if o.title != "" {
		args["title"] = o.title
	}
	b, _ := json.Marshal(args)
	r, err := h.callJS("createThread", string(b))
	if err != nil {
		return "", err
	}
	var thread HarnessThread
	if err := json.Unmarshal([]byte(r), &thread); err != nil {
		var id string
		json.Unmarshal([]byte(r), &id)
		return id, nil
	}
	return thread.ID, nil
}

// SwitchThread switches to a different thread.
func (h *Harness) SwitchThread(threadID string) error {
	b, _ := json.Marshal(map[string]string{"threadId": threadID})
	return h.callJSVoid("switchThread", string(b))
}

// DeleteThread deletes a thread.
func (h *Harness) DeleteThread(threadID string) error {
	b, _ := json.Marshal(map[string]string{"threadId": threadID})
	return h.callJSVoid("deleteThread", string(b))
}

// ListThreads returns all threads, optionally filtered by resource.
func (h *Harness) ListThreads(opts ...ListThreadsOption) ([]HarnessThread, error) {
	o := &listThreadsOptions{}
	for _, opt := range opts {
		opt(o)
	}
	var argsJSON string
	if o.resourceID != "" {
		b, _ := json.Marshal(map[string]string{"resourceId": o.resourceID})
		argsJSON = string(b)
	}
	r, err := h.callJS("listThreads", argsJSON)
	if err != nil {
		return nil, err
	}
	var threads []HarnessThread
	json.Unmarshal([]byte(r), &threads)
	return threads, nil
}

// RenameThread renames the current thread.
func (h *Harness) RenameThread(title string) error {
	b, _ := json.Marshal(map[string]string{"title": title})
	return h.callJSVoid("renameThread", string(b))
}

// CloneThread clones a thread.
func (h *Harness) CloneThread(opts ...CloneOption) (string, error) {
	o := &cloneOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{}
	if o.sourceThreadID != "" {
		args["sourceThreadId"] = o.sourceThreadID
	}
	if o.title != "" {
		args["title"] = o.title
	}
	if o.resourceID != "" {
		args["resourceId"] = o.resourceID
	}
	b, _ := json.Marshal(args)
	r, err := h.callJS("cloneThread", string(b))
	if err != nil {
		return "", err
	}
	var thread HarnessThread
	if err := json.Unmarshal([]byte(r), &thread); err != nil {
		var id string
		json.Unmarshal([]byte(r), &id)
		return id, nil
	}
	return thread.ID, nil
}

// GetCurrentThreadID returns the current thread ID.
func (h *Harness) GetCurrentThreadID() string {
	r, _ := h.callJSSimple("getCurrentThreadId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// ListMessages returns messages for the current or specified thread.
func (h *Harness) ListMessages(opts ...ListMessagesOption) ([]HarnessMessage, error) {
	o := &listMessagesOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{}
	if o.threadID != "" {
		args["threadId"] = o.threadID
	}
	if o.limit > 0 {
		args["limit"] = o.limit
	}
	b, _ := json.Marshal(args)
	r, err := h.callJS("listMessages", string(b))
	if err != nil {
		return nil, err
	}
	var msgs []HarnessMessage
	json.Unmarshal([]byte(r), &msgs)
	return msgs, nil
}
