package harness

import "encoding/json"

// GetState returns the current Harness state.
func (h *Harness) GetState() map[string]any {
	r, _ := h.callJSSimple("getState")
	var state map[string]any
	json.Unmarshal([]byte(r), &state)
	return state
}

// SetState updates Harness state. Validated by Zod in JS.
func (h *Harness) SetState(updates map[string]any) error {
	b, _ := json.Marshal(updates)
	return h.callJSVoid("setState", string(b))
}

// SwitchObserverModel changes the OM observer model.
func (h *Harness) SwitchObserverModel(modelID string) error {
	b, _ := json.Marshal(map[string]string{"modelId": modelID})
	return h.callJSVoid("switchObserverModel", string(b))
}

// SwitchReflectorModel changes the OM reflector model.
func (h *Harness) SwitchReflectorModel(modelID string) error {
	b, _ := json.Marshal(map[string]string{"modelId": modelID})
	return h.callJSVoid("switchReflectorModel", string(b))
}

// GetObserverModelID returns the current observer model ID.
func (h *Harness) GetObserverModelID() string {
	r, _ := h.callJSSimple("getObserverModelId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// GetReflectorModelID returns the current reflector model ID.
func (h *Harness) GetReflectorModelID() string {
	r, _ := h.callJSSimple("getReflectorModelId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// SetSubagentModelID sets the model for a subagent type.
func (h *Harness) SetSubagentModelID(modelID, agentType string) error {
	b, _ := json.Marshal(map[string]string{"modelId": modelID, "agentType": agentType})
	return h.callJSVoid("setSubagentModelId", string(b))
}

// GetSubagentModelID returns the model for a subagent type.
func (h *Harness) GetSubagentModelID(agentType string) string {
	b, _ := json.Marshal(map[string]string{"agentType": agentType})
	r, _ := h.callJS("getSubagentModelId", string(b))
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// HasWorkspace returns true if a workspace is configured.
func (h *Harness) HasWorkspace() bool {
	r, _ := h.callJSSimple("hasWorkspace")
	var b bool
	json.Unmarshal([]byte(r), &b)
	return b
}

// IsWorkspaceReady returns true if the workspace is initialized and ready.
func (h *Harness) IsWorkspaceReady() bool {
	r, _ := h.callJSSimple("isWorkspaceReady")
	var b bool
	json.Unmarshal([]byte(r), &b)
	return b
}

// DestroyWorkspace destroys the current workspace.
func (h *Harness) DestroyWorkspace() error {
	return h.callJSVoid("destroyWorkspace", "")
}

// GetSession returns the current session info.
func (h *Harness) GetSession() HarnessSession {
	r, _ := h.callJSSimple("getSession")
	var sess HarnessSession
	json.Unmarshal([]byte(r), &sess)
	return sess
}

// SetResourceID scopes threads to a specific resource.
func (h *Harness) SetResourceID(resourceID string) error {
	b, _ := json.Marshal(map[string]string{"resourceId": resourceID})
	return h.callJSVoid("setResourceId", string(b))
}

// GetResourceID returns the current resource ID.
func (h *Harness) GetResourceID() string {
	r, _ := h.callJSSimple("getResourceId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// GetKnownResourceIDs returns all known resource IDs.
func (h *Harness) GetKnownResourceIDs() ([]string, error) {
	r, err := h.callJSSimple("getKnownResourceIds")
	if err != nil {
		return nil, err
	}
	var ids []string
	json.Unmarshal([]byte(r), &ids)
	return ids, nil
}
