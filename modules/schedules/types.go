package schedules

import (
	"time"

	"github.com/brainlet/brainkit/internal/types"
)

// Store is the narrow persistence surface the module needs. brainkit's
// KitStore (returned by brainkit.NewSQLiteStore) satisfies it structurally,
// so the common case is `Config{Store: kitStore}`.
type Store interface {
	SaveSchedule(s types.PersistedSchedule) error
	LoadSchedules() ([]types.PersistedSchedule, error)
	DeleteSchedule(id string) error
	// ClaimScheduleFire atomically claims a schedule fire across replicas.
	// Returns true if this replica claimed it, false if another already did.
	ClaimScheduleFire(scheduleID string, fireTime time.Time) (bool, error)
}

// Config configures the schedules module.
type Config struct {
	// Store is optional. When nil: schedules are in-memory only and do not
	// survive restart. When provided: schedules are persisted and restored on
	// module Init, and ClaimScheduleFire is used for multi-replica dedup.
	Store Store
}
