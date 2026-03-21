package harness

import "encoding/json"

// SwitchMode switches to a different mode.
func (h *Harness) SwitchMode(modeID string) error {
	b, _ := json.Marshal(map[string]string{"modeId": modeID})
	return h.callJSVoid("switchMode", string(b))
}

// ListModes returns all configured modes.
func (h *Harness) ListModes() []Mode {
	r, _ := h.callJSSimple("listModes")
	var modes []Mode
	json.Unmarshal([]byte(r), &modes)
	return modes
}

// GetCurrentMode returns the active mode.
func (h *Harness) GetCurrentMode() Mode {
	r, _ := h.callJSSimple("getCurrentMode")
	var mode Mode
	json.Unmarshal([]byte(r), &mode)
	return mode
}

// GetCurrentModeID returns the active mode ID.
func (h *Harness) GetCurrentModeID() string {
	r, _ := h.callJSSimple("getCurrentModeId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// SwitchModel changes the active model.
func (h *Harness) SwitchModel(modelID string, opts ...ModelOption) error {
	o := &modelOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{"modelId": modelID}
	if o.scope != "" {
		args["scope"] = o.scope
	}
	if o.modeID != "" {
		args["modeId"] = o.modeID
	}
	b, _ := json.Marshal(args)
	return h.callJSVoid("switchModel", string(b))
}

// ListAvailableModels returns all available models with auth status.
func (h *Harness) ListAvailableModels() ([]AvailableModel, error) {
	r, err := h.callJSSimple("listAvailableModels")
	if err != nil {
		return nil, err
	}
	var models []AvailableModel
	json.Unmarshal([]byte(r), &models)
	return models, nil
}

// GetCurrentModelID returns the current model ID.
func (h *Harness) GetCurrentModelID() string {
	r, _ := h.callJSSimple("getCurrentModelId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// HasModelSelected returns true if a model is selected.
func (h *Harness) HasModelSelected() bool {
	r, _ := h.callJSSimple("hasModelSelected")
	var b bool
	json.Unmarshal([]byte(r), &b)
	return b
}
