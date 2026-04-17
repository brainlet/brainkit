package sdk

// ── Secrets Management ──

type SecretsSetMsg struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (SecretsSetMsg) BusTopic() string { return "secrets.set" }

type SecretsSetResp struct {
	Stored  bool `json:"stored"`
	Version int  `json:"version"`
}

type SecretsGetMsg struct {
	Name string `json:"name"`
}

func (SecretsGetMsg) BusTopic() string { return "secrets.get" }

type SecretsGetResp struct {
	Value string `json:"value"`
}

type SecretsDeleteMsg struct {
	Name string `json:"name"`
}

func (SecretsDeleteMsg) BusTopic() string { return "secrets.delete" }

type SecretsDeleteResp struct {
	Deleted bool `json:"deleted"`
}

type SecretsListMsg struct{}

func (SecretsListMsg) BusTopic() string { return "secrets.list" }

type SecretsListResp struct {
	Secrets []SecretMetaInfo `json:"secrets"`
}

type SecretMetaInfo struct {
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Version   int    `json:"version"`
}

type SecretsRotateMsg struct {
	Name     string `json:"name"`
	NewValue string `json:"newValue"`
	Restart  bool   `json:"restart"` // restart plugins that reference this secret
}

func (SecretsRotateMsg) BusTopic() string { return "secrets.rotate" }

type SecretsRotateResp struct {
	Rotated          bool     `json:"rotated"`
	Version          int      `json:"version"`
	RestartedPlugins []string `json:"restartedPlugins,omitempty"`
}

// ── Secret Events ──

type SecretsAccessedEvent struct {
	Name      string `json:"name"`
	Accessor  string `json:"accessor"`
	Timestamp string `json:"timestamp"`
}

func (SecretsAccessedEvent) BusTopic() string { return "secrets.accessed" }

type SecretsStoredEvent struct {
	Name      string `json:"name"`
	Version   int    `json:"version"`
	Timestamp string `json:"timestamp"`
}

func (SecretsStoredEvent) BusTopic() string { return "secrets.stored" }

type SecretsRotatedEvent struct {
	Name             string   `json:"name"`
	Version          int      `json:"version"`
	RestartedPlugins []string `json:"restartedPlugins,omitempty"`
	Timestamp        string   `json:"timestamp"`
}

func (SecretsRotatedEvent) BusTopic() string { return "secrets.rotated" }

type SecretsDeletedEvent struct {
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
}

func (SecretsDeletedEvent) BusTopic() string { return "secrets.deleted" }
