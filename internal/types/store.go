package types

import (
	"encoding/json"
	"time"
)

// KitStore provides optional persistence for deployments, schedules, and plugins.
// When configured on a Kit, data survives Kit restarts.
type KitStore interface {
	// Deployments
	SaveDeployment(d PersistedDeployment) error
	LoadDeployments() ([]PersistedDeployment, error)
	LoadDeployment(source string) (PersistedDeployment, error)
	DeleteDeployment(source string) error

	// Schedules
	SaveSchedule(s PersistedSchedule) error
	LoadSchedules() ([]PersistedSchedule, error)
	DeleteSchedule(id string) error

	// Schedule deduplication (for multi-replica)
	// ClaimScheduleFire atomically claims a schedule fire.
	// Returns true if this replica claimed it, false if another already did.
	ClaimScheduleFire(scheduleID string, fireTime time.Time) (bool, error)

	// Installed plugins
	SaveInstalledPlugin(p InstalledPlugin) error
	LoadInstalledPlugins() ([]InstalledPlugin, error)
	DeleteInstalledPlugin(name string) error

	// Running plugins (for restart recovery)
	SaveRunningPlugin(p RunningPluginRecord) error
	LoadRunningPlugins() ([]RunningPluginRecord, error)
	DeleteRunningPlugin(name string) error

	// Lifecycle
	Close() error
}

// PersistedDeployment is the on-disk format for a .ts deployment.
type PersistedDeployment struct {
	Source      string    `json:"source"`
	Code        string    `json:"code"`
	Order       int       `json:"order"`
	DeployedAt  time.Time `json:"deployedAt"`
	PackageName string    `json:"packageName,omitempty"`
}

// PersistedSchedule is the on-disk format for a scheduled bus message.
type PersistedSchedule struct {
	ID         string          `json:"id"`
	Expression string          `json:"expression"`
	Duration   time.Duration   `json:"duration"`
	Topic      string          `json:"topic"`
	Payload    json.RawMessage `json:"payload"`
	Source     string          `json:"source"`
	CreatedAt  time.Time       `json:"createdAt"`
	NextFire   time.Time       `json:"nextFire"`
	OneTime    bool            `json:"oneTime"`
}
